package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bsreeram08/apigent/internal/env"
	"github.com/bsreeram08/apigent/internal/storage/memstore"
	"github.com/bsreeram08/apigent/pkg/types"
)

// mcpFixture is a server over a request library with one environment that
// defines BASE_URL and the secret API_KEY (sk-test-123).
func mcpFixture(t *testing.T) (*mcpServer, *[][]string) {
	t.Helper()
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{
		Name:    "get user",
		Method:  "GET",
		URL:     "{{BASE_URL}}/users/{{USER_ID}}",
		Headers: []types.Header{{Key: "Authorization", Value: "Bearer {{API_KEY}}"}},
	})
	saveTestRequest(t, db, &types.SavedRequest{Name: "create-order", Method: "POST", URL: "{{BASE_URL}}/orders"})

	envs := env.NewMemoryStore()
	dev := env.NewEnvironment("dev", "")
	dev.SetVariable("BASE_URL", "https://api.example.test")
	dev.SetVariable("API_KEY", "sk-test-123")
	dev.SecretKeys["API_KEY"] = true
	if err := envs.SaveEnv(dev); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	srv := &mcpServer{
		db: db, envStorage: envs, envName: "dev", version: "test",
		runRequest: func(ctx context.Context, args []string) ([]byte, error) {
			calls = append(calls, args)
			return []byte(`{"ok":true,"status":200,"body":{"id":"u_1"}}` + "\n"), nil
		},
	}
	return srv, &calls
}

// rpc sends the messages through serve and returns the responses by id.
func rpc(t *testing.T, srv *mcpServer, msgs ...string) map[string]map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := srv.serve(context.Background(), strings.NewReader(strings.Join(msgs, "\n")), &out); err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("server wrote non-JSON line %q: %v", line, err)
		}
		id, _ := json.Marshal(m["id"])
		byID[string(id)] = m
	}
	return byID
}

func TestMCPInitializeAndListTools(t *testing.T) {
	srv, _ := mcpFixture(t)
	got := rpc(t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	if len(got) != 2 {
		t.Fatalf("want 2 responses (no reply to the notification), got %d: %v", len(got), got)
	}
	init := got["1"]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-03-26" {
		t.Errorf("should echo a supported client version, got %v", init["protocolVersion"])
	}

	raw, _ := json.Marshal(got["2"]["result"])
	if strings.Contains(string(raw), "sk-test-123") {
		t.Fatalf("tools/list leaked a secret: %s", raw)
	}
	var list struct {
		Tools []mcpTool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 2 || list.Tools[0].Name != "create-order" || list.Tools[1].Name != "get_user" {
		t.Fatalf("unexpected tools: %+v", list.Tools)
	}
	getUser := list.Tools[1]
	props := getUser.InputSchema["properties"].(map[string]any)
	if len(props) != 1 || props["USER_ID"] == nil {
		t.Fatalf("only USER_ID is undefined by the environment, got properties %v", props)
	}
	if getUser.Annotations["readOnlyHint"] != true {
		t.Errorf("GET should be marked read-only")
	}
}

func TestMCPCallRunsInAgentModeWithAllowedVars(t *testing.T) {
	srv, calls := mcpFixture(t)
	got := rpc(t, srv, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"get_user","arguments":{"USER_ID":"42"}}}`)
	res := got["7"]["result"].(map[string]any)
	if res["isError"] != false {
		t.Fatalf("want success, got %v", res)
	}
	if len(*calls) != 1 {
		t.Fatalf("want one run, got %v", *calls)
	}
	want := []string{"get user", "--env", "dev", "--var", "USER_ID=42"}
	if strings.Join((*calls)[0], "|") != strings.Join(want, "|") {
		t.Fatalf("run args = %q, want %q", (*calls)[0], want)
	}
}

func TestMCPCallRefusesOverridingEnvironmentValues(t *testing.T) {
	srv, calls := mcpFixture(t)
	// Redirecting BASE_URL would send the API_KEY header to another host.
	got := rpc(t, srv, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_user","arguments":{"USER_ID":"1","BASE_URL":"https://evil.test"}}}`)
	res := got["3"]["result"].(map[string]any)
	if res["isError"] != true {
		t.Fatalf("overriding BASE_URL must be refused, got %v", res)
	}
	if len(*calls) != 0 {
		t.Fatalf("nothing should run, got %v", *calls)
	}
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, `"BASE_URL" is not allowed`) {
		t.Fatalf("error should name the argument, got %q", text)
	}
}

func TestMCPErrors(t *testing.T) {
	srv, _ := mcpFixture(t)
	got := rpc(t, srv,
		`not json`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nope"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
	)
	codes := map[string]float64{"null": rpcParseError, "1": rpcInvalidParams, "2": rpcMethodNotFound}
	for id, code := range codes {
		e, _ := got[id]["error"].(map[string]any)
		if e == nil || e["code"] != code {
			t.Errorf("id %s: want error %v, got %v", id, code, got[id])
		}
	}
	if got["3"]["error"] != nil {
		t.Errorf("ping should succeed, got %v", got["3"])
	}
}

func TestUniqueToolName(t *testing.T) {
	used := map[string]bool{}
	for _, tc := range []struct{ in, want string }{
		{"get user", "get_user"},
		{"get/user", "get_user_2"},
		{"!!!", "request"},
		{"orders-v2", "orders-v2"},
	} {
		if got := uniqueToolName(tc.in, used); got != tc.want {
			t.Errorf("uniqueToolName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
