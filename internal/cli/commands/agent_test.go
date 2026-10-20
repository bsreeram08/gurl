package commands

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/internal/storage/memstore"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

const agentTestSecret = "sk-live-agent-secret-1234"

// echoServer answers with the request it got as JSON (path, auth header and
// body), so secrets sent come back in the response. /missing is a 404.
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "session=cookie-value-5678")
		if strings.HasPrefix(r.URL.Path, "/missing") {
			w.WriteHeader(http.StatusNotFound)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"path": r.URL.RequestURI(), "auth": r.Header.Get("Authorization"), "body": string(body), "id": "u-1"})
	}))
	t.Cleanup(server.Close)
	return server
}

// quietServer answers 200 {"ok":true} without echoing anything back.
func quietServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"id":"u-1"}`))
	}))
	t.Cleanup(server.Close)
	return server
}

// secretEnvStore holds environment "dev" with API_KEY marked secret.
func secretEnvStore(t *testing.T) env.Store {
	t.Helper()
	store := env.NewMemoryStore()
	dev := env.NewEnvironment("dev", "")
	dev.SetSecretVariable("API_KEY", agentTestSecret)
	if err := store.SaveEnv(dev); err != nil {
		t.Fatalf("SaveEnv: %v", err)
	}
	return store
}

// secretRequest sends API_KEY in a header, the query and the body.
func secretRequest(name, baseURL string) *types.SavedRequest {
	return &types.SavedRequest{
		Name:    name,
		Method:  "POST",
		URL:     baseURL + "/users?api_key={{API_KEY}}",
		Headers: []types.Header{{Key: "Authorization", Value: "Bearer {{API_KEY}}"}},
		Body:    `{"key":"{{API_KEY}}"}`,
	}
}

// decodeLines decodes every stdout line as one JSON object.
func decodeLines(t *testing.T, stdout string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("stdout line is not a JSON object: %v\n%q", err, line)
		}
		out = append(out, obj)
	}
	return out
}

func decodeOne(t *testing.T, stdout string) map[string]any {
	t.Helper()
	lines := decodeLines(t, stdout)
	if len(lines) != 1 {
		t.Fatalf("want 1 JSON object, got %d:\n%s", len(lines), stdout)
	}
	return lines[0]
}

func clearAgentEnv(t *testing.T) {
	t.Helper()
	t.Setenv(AgentEnvVar, "")
	t.Setenv(NoInputEnvVar, "")
}

func TestRunJSONSuccess(t *testing.T) {
	clearAgentEnv(t)
	server := quietServer(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "ok", Method: "GET", URL: server.URL + "/ok", Extracts: []types.Extract{{Name: "uid", Source: "jsonpath:$.id"}}})

	stdout, stderr, err := runGurl(t, db, env.NewMemoryStore(), "ok", "--json")
	if err != nil {
		t.Fatalf("run --json: %v\nstderr: %s", err, stderr)
	}
	got := decodeOne(t, stdout)
	for key, want := range map[string]any{"request": "ok", "ok": true, "method": "GET", "status": 200.0, "status_text": "OK"} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v", key, got[key], want)
		}
	}
	if id, _ := got["id"].(string); id == "" || got["short_id"] != types.ShortID(id) {
		t.Errorf("bad ids: %v / %v", got["id"], got["short_id"])
	}
	body, ok := got["body"].(map[string]any)
	if !ok || body["ok"] != true {
		t.Errorf("body not parsed as JSON: %#v", got["body"])
	}
	if extracted, _ := got["extracted"].(map[string]any); extracted["uid"] != "u-1" {
		t.Errorf("extracted = %#v", got["extracted"])
	}
	if strings.Contains(stderr, "request failed") {
		t.Errorf("no failure notice expected:\n%s", stderr)
	}
}

func TestRunJSONNon2xx(t *testing.T) {
	clearAgentEnv(t)
	server := echoServer(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "gone", Method: "GET", URL: server.URL + "/missing"})

	stdout, stderr, err := runGurl(t, db, env.NewMemoryStore(), "gone", "--json")
	if err != nil {
		t.Fatalf("404 without --fail must not fail: %v", err)
	}
	got := decodeOne(t, stdout)
	if got["ok"] != false || got["status"] != 404.0 || got["failure_phase"] != runner.FailurePhaseStatus {
		t.Fatalf("unexpected object: %v", got)
	}
	if strings.Contains(stderr, "request failed") {
		t.Fatalf("JSON mode prints no failure notice, got:\n%s", stderr)
	}

	stdout, _, err = runGurl(t, db, env.NewMemoryStore(), "gone", "--json", "--fail")
	if err == nil {
		t.Fatal("404 with --fail must fail")
	}
	if !isReported(err) {
		t.Fatalf("execution error should be marked reported: %v", err)
	}
	if got := decodeOne(t, stdout); got["ok"] != false {
		t.Fatalf("unexpected object: %v", got)
	}
}

func TestRunJSONTransportErrorIsDiagnosable(t *testing.T) {
	clearAgentEnv(t)
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "down", Method: "GET", URL: url + "/x"})

	stdout, _, err := runGurl(t, db, env.NewMemoryStore(), "down", "--json")
	if err == nil {
		t.Fatal("transport error must fail")
	}
	got := decodeOne(t, stdout)
	if got["ok"] != false || got["failure_phase"] != runner.FailurePhaseHTTP || got["error"] == "" {
		t.Fatalf("unexpected object: %v", got)
	}
	if _, ok := got["status"]; ok {
		t.Fatalf("no status without a response: %v", got)
	}
	id, _ := got["short_id"].(string)
	if out := diagGurl(t, db, id); !strings.Contains(out, "down") {
		t.Fatalf("diag %s did not load the record:\n%s", id, out)
	}
}

func TestRunJSONResolutionProblemsAreStructured(t *testing.T) {
	clearAgentEnv(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "broken", Method: "GET", URL: "http://127.0.0.1:1/{{missing}}?n={{count:num}}", Headers: []types.Header{{Key: "X-Id", Value: "{{missing}}"}}})

	stdout, _, err := runGurl(t, db, env.NewMemoryStore(), "broken", "--json", "--var", "count=abc")
	if err == nil {
		t.Fatal("unresolvable request must fail")
	}
	got := decodeOne(t, stdout)
	if got["failure_phase"] != runner.FailurePhaseRequestBuild {
		t.Fatalf("failure_phase = %v", got["failure_phase"])
	}
	problems, _ := got["problems"].([]any)
	want := map[string]bool{
		string(resolve.ProblemMissing) + " missing url":         false,
		string(resolve.ProblemMissing) + " missing header X-Id": false,
		string(resolve.ProblemBadType) + " count url":           false,
	}
	for _, p := range problems {
		m := p.(map[string]any)
		key := m["kind"].(string) + " " + m["variable"].(string) + " " + m["location"].(string)
		if _, ok := want[key]; ok {
			want[key] = true
		}
		if m["message"] == "" {
			t.Errorf("problem without message: %v", m)
		}
	}
	for key, seen := range want {
		if !seen {
			t.Errorf("problem %q not reported in %v", key, problems)
		}
	}
}

func TestRunJSONNeverPrintsSecrets(t *testing.T) {
	clearAgentEnv(t)
	server := quietServer(t)
	db := memstore.New()
	saveTestRequest(t, db, secretRequest("secret", server.URL))
	envs := secretEnvStore(t)

	for name, args := range map[string][]string{
		"run":     {"secret", "--env", "dev", "--json"},
		"dry run": {"secret", "--env", "dev", "--json", "--dry-run"},
	} {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, err := runGurl(t, db, envs, args...)
			if err != nil {
				t.Fatalf("%v\nstderr: %s", err, stderr)
			}
			got := decodeOne(t, stdout)
			if strings.Contains(stdout+stderr, agentTestSecret) {
				t.Fatalf("secret leaked:\nstdout: %s\nstderr: %s", stdout, stderr)
			}
			if url, _ := got["url"].(string); !strings.Contains(url, "api_key=*****") {
				t.Fatalf("url not masked: %v", got["url"])
			}
		})
	}
}

func TestAgentModeMasksSecretsEchoedInBody(t *testing.T) {
	clearAgentEnv(t)
	server := echoServer(t)
	db := memstore.New()
	saveTestRequest(t, db, secretRequest("echo", server.URL))
	envs := secretEnvStore(t)

	normal, _, err := runGurl(t, db, envs, "echo", "--env", "dev", "--json")
	if err != nil {
		t.Fatalf("run --json: %v", err)
	}
	if !strings.Contains(normal, agentTestSecret) {
		t.Fatalf("without agent mode the body is as received; expected the echoed secret:\n%s", normal)
	}

	t.Setenv(AgentEnvVar, "1")
	agent, stderr, err := runGurl(t, db, envs, "echo", "--env", "dev")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}
	if strings.Contains(agent+stderr, agentTestSecret) {
		t.Fatalf("agent mode leaked the secret:\n%s\n%s", agent, stderr)
	}
	body, _ := decodeOne(t, agent)["body"].(map[string]any)
	if body["auth"] != "Bearer *****" || body["id"] != "u-1" {
		t.Fatalf("body not masked as expected: %v", body)
	}
}

func TestRunJSONChainIsNDJSON(t *testing.T) {
	clearAgentEnv(t)
	server := quietServer(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "first", Method: "GET", URL: server.URL + "/1", PostScript: "gurl.setNextRequest('second')"})
	saveTestRequest(t, db, &types.SavedRequest{Name: "second", Method: "GET", URL: server.URL + "/2", PostScript: "gurl.setNextRequest('third')"})
	saveTestRequest(t, db, &types.SavedRequest{Name: "third", Method: "GET", URL: server.URL + "/3"})

	stdout, stderr, err := runGurl(t, db, env.NewMemoryStore(), "first", "--chain", "--json")
	if err != nil {
		t.Fatalf("chain: %v\nstderr: %s", err, stderr)
	}
	lines := decodeLines(t, stdout)
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d:\n%s", len(lines), stdout)
	}
	for i, name := range []string{"first", "second", "third"} {
		if lines[i]["request"] != name || lines[i]["ok"] != true {
			t.Errorf("line %d = %v", i, lines[i])
		}
	}
}

func TestRunJSONChainStopsWithErrorLine(t *testing.T) {
	clearAgentEnv(t)
	server := quietServer(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "first", Method: "GET", URL: server.URL + "/1", PostScript: "gurl.setNextRequest('nowhere')"})

	stdout, _, err := runGurl(t, db, env.NewMemoryStore(), "first", "--chain", "--json")
	if err == nil {
		t.Fatal("a missing next request must fail")
	}
	lines := decodeLines(t, stdout)
	if len(lines) != 2 || lines[0]["request"] != "first" || lines[1]["ok"] != false || !strings.Contains(lines[1]["error"].(string), "nowhere") {
		t.Fatalf("want the step then an error object, got:\n%s", stdout)
	}
}

func TestRunJSONDataRowsAreNDJSON(t *testing.T) {
	clearAgentEnv(t)
	server := quietServer(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "rows", Method: "GET", URL: server.URL + "/r/{{n}}"})
	data := filepath.Join(t.TempDir(), "rows.csv")
	if err := os.WriteFile(data, []byte("n\n1\n2\n3\n4\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runGurl(t, db, env.NewMemoryStore(), "rows", "--data", data, "--json")
	if err != nil {
		t.Fatalf("data run: %v\nstderr: %s", err, stderr)
	}
	lines := decodeLines(t, stdout)
	if len(lines) != 4 {
		t.Fatalf("want 4 lines, got %d:\n%s", len(lines), stdout)
	}
	for i, line := range lines {
		if line["row"] != float64(i+1) || !strings.HasSuffix(line["url"].(string), "/r/"+string(rune('1'+i))) {
			t.Errorf("line %d = %v", i, line)
		}
	}
}

func TestRunDryRunJSONProblems(t *testing.T) {
	clearAgentEnv(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "plan", Method: "POST", URL: "http://example.test/{{who}}", Body: `{"a":"b"}`})

	stdout, _, err := runGurl(t, db, env.NewMemoryStore(), "plan", "--dry-run", "--json")
	if err == nil {
		t.Fatal("dry run with problems must fail")
	}
	got := decodeOne(t, stdout)
	if got["dry_run"] != true || got["ok"] != false || got["method"] != "POST" {
		t.Fatalf("unexpected object: %v", got)
	}
	problems, _ := got["problems"].([]any)
	if len(problems) != 1 || problems[0].(map[string]any)["variable"] != "who" || problems[0].(map[string]any)["kind"] != string(resolve.ProblemMissing) {
		t.Fatalf("problems = %v", problems)
	}
	if body, _ := got["body"].(map[string]any); body["a"] != "b" {
		t.Fatalf("body = %v", got["body"])
	}

	stdout, _, err = runGurl(t, db, env.NewMemoryStore(), "plan", "--dry-run", "--json", "--var", "who=me")
	if err != nil {
		t.Fatalf("resolvable dry run: %v", err)
	}
	if got := decodeOne(t, stdout); got["ok"] != true || got["url"] != "http://example.test/me" {
		t.Fatalf("unexpected object: %v", got)
	}
}

func TestRunJSONPreExecutionErrors(t *testing.T) {
	clearAgentEnv(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "exists", Method: "GET", URL: "http://example.test"})

	for name, args := range map[string][]string{
		"unknown request": {"nope", "--json"},
		"unknown env":     {"exists", "--env", "prod", "--json"},
	} {
		t.Run(name, func(t *testing.T) {
			stdout, _, err := runGurl(t, db, env.NewMemoryStore(), args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			got := decodeOne(t, stdout)
			if got["ok"] != false || got["error"] != err.Error() || len(got) != 2 {
				t.Fatalf("want {ok:false,error:%q}, got %v", err.Error(), got)
			}
		})
	}
}

// historyDB has two records for request "r", one with a secret in the error.
func historyDB(t *testing.T) storage.DB {
	t.Helper()
	db := memstore.New()
	req := &types.SavedRequest{Name: "r", Method: "GET", URL: "http://example.test"}
	saveTestRequest(t, db, req)
	saved, err := db.GetRequestByName("r")
	if err != nil {
		t.Fatal(err)
	}
	older := types.NewExecutionHistory(saved.ID, `{"a":1,"b":"x"}`, 200, 12, 15)
	older.RequestName = "r"
	older.TimestampNs -= 1e9
	older.Timestamp--
	newer := types.NewExecutionHistory(saved.ID, `{"a":2,"c":true}`, 500, 30, 16)
	newer.RequestName = "r"
	newer.FailurePhase = runner.FailurePhaseStatus
	for _, rec := range []*types.ExecutionHistory{older, newer} {
		if err := db.SaveHistory(rec); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func runCmd(t *testing.T, cmd *cli.Command, args ...string) (string, error) {
	t.Helper()
	var err error
	stdout, _ := captureStreams(t, func() {
		err = cmd.Run(context.Background(), append([]string{cmd.Name}, args...))
	})
	return stdout, err
}

func TestHistoryAndTimelineJSON(t *testing.T) {
	clearAgentEnv(t)
	db := historyDB(t)

	for name, cmd := range map[string]*cli.Command{"history": HistoryCommand(db), "timeline": TimelineCommand(db)} {
		t.Run(name, func(t *testing.T) {
			args := []string{"--json"}
			if name == "history" {
				args = []string{"r", "--json"}
			}
			stdout, err := runCmd(t, cmd, args...)
			if err != nil {
				t.Fatal(err)
			}
			var got []recordSummary
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatalf("not JSON: %v\n%s", err, stdout)
			}
			if len(got) != 2 || got[0].Status != 500 || got[0].OK || !got[1].OK || got[0].Request != "r" || got[0].Time == "" || got[0].ShortID != types.ShortID(got[0].ID) {
				t.Fatalf("unexpected summaries: %+v", got)
			}
		})
	}
}

func TestDiffJSON(t *testing.T) {
	clearAgentEnv(t)
	db := historyDB(t)

	stdout, err := runCmd(t, DiffCommand(db, env.NewMemoryStore()), "r", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got diffReport
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if got.Format != "json" || got.Identical || got.Before.Status != 200 || got.After.Status != 500 {
		t.Fatalf("unexpected diff: %+v", got)
	}
	ops := map[string]string{}
	for _, c := range got.Changes {
		ops[c.Path] = c.Op
	}
	if ops["/a"] != "replace" || ops["/b"] != "remove" || ops["/c"] != "add" {
		t.Fatalf("unexpected changes: %+v", got.Changes)
	}
}

func TestCollectionListAndShowJSONNeverPrintSecrets(t *testing.T) {
	clearAgentEnv(t)
	db := memstore.New()
	col := types.NewCollection("payments")
	col.SetVariable("BASE_URL", "https://api.example.test")
	col.SetVariable("PASSWORD", "hunter2-plain")
	col.SetSecretVariable("API_KEY", agentTestSecret)
	if err := db.SaveCollection(col); err != nil {
		t.Fatal(err)
	}
	saveTestRequest(t, db, &types.SavedRequest{Name: "list-payments", Collection: "payments", URL: "{{BASE_URL}}/payments?token=literal-token-999"})

	stdout, err := runCmd(t, collectionListCommand(db), "--json")
	if err != nil {
		t.Fatal(err)
	}
	var list []struct {
		Name      string `json:"name"`
		Requests  int    `json:"requests"`
		Variables int    `json:"variables"`
	}
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if len(list) != 1 || list[0].Name != "payments" || list[0].Requests != 1 || list[0].Variables != 3 {
		t.Fatalf("unexpected list: %+v", list)
	}

	stdout, err = runCmd(t, collectionShowCommand(db), "payments", "--json")
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{agentTestSecret, "hunter2-plain", "literal-token-999"} {
		if strings.Contains(stdout, leak) {
			t.Fatalf("collection show --json leaked %q:\n%s", leak, stdout)
		}
	}
	var show collectionShowOutput
	if err := json.Unmarshal([]byte(stdout), &show); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if show.Variables["BASE_URL"] != "https://api.example.test" || show.Variables["API_KEY"] != "*****" || len(show.Secrets) != 1 || show.Secrets[0] != "API_KEY" || len(show.Requests) != 1 || show.Requests[0].Name != "list-payments" {
		t.Fatalf("unexpected show output: %+v", show)
	}
}

func TestNoInputRunWithoutNameFailsFast(t *testing.T) {
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "a", URL: "http://example.test"})

	for name, setup := range map[string]func(){
		"flag": func() {},
		"env":  func() { t.Setenv(NoInputEnvVar, "1") },
		"json": func() {},
	} {
		t.Run(name, func(t *testing.T) {
			clearAgentEnv(t)
			setup()
			root := &cli.Command{Name: "gurl", Flags: GlobalFlags(), Commands: []*cli.Command{RunCommand(db, env.NewMemoryStore())}}
			root.ExitErrHandler = func(context.Context, *cli.Command, error) {} // keep cli.Exit from exiting the test
			args := []string{"gurl", "run"}
			switch name {
			case "flag":
				args = []string{"gurl", "--no-input", "run"}
			case "json":
				args = append(args, "--json")
			}
			var err error
			stdout, _ := captureStreams(t, func() {
				err = root.Run(context.Background(), args)
			})
			if err == nil || !strings.Contains(err.Error(), "request name is required") {
				t.Fatalf("want a request-name error, got %v (stdout %q)", err, stdout)
			}
			if strings.Contains(stdout, "Run request by number") {
				t.Fatalf("prompted despite non-interactive mode:\n%s", stdout)
			}
		})
	}
}

func TestNoInputSaveToMissingCollectionFails(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv(NoInputEnvVar, "1")
	old := saveCollectionIsInteractive
	saveCollectionIsInteractive = func() bool { return true } // a terminal would prompt
	defer func() { saveCollectionIsInteractive = old }()

	db := memstore.New()
	err := SaveCommand(db).Run(context.Background(), []string{"save", "x", "http://example.test", "--collection", "nope"})
	if err == nil || !strings.Contains(err.Error(), `collection "nope" does not exist; create it first with 'gurl collection create nope'`) {
		t.Fatalf("want an actionable missing-collection error, got %v", err)
	}
	if _, err := db.GetCollectionByName("nope"); err == nil {
		t.Fatal("collection must not be created without confirmation")
	}
}

func TestNoInputPassphraseIsActionable(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("GURL_IMPORT_PASSPHRASE", "")
	t.Setenv(NoInputEnvVar, "1")
	old := collectionPassphraseIsInteractive
	collectionPassphraseIsInteractive = func() bool { return true }
	defer func() { collectionPassphraseIsInteractive = old }()

	cmd := &cli.Command{
		Name:  "test",
		Flags: []cli.Flag{&cli.StringFlag{Name: "passphrase"}},
		Action: func(ctx context.Context, c *cli.Command) error {
			_, err := collectionPassphrase(c, "Passphrase: ")
			return err
		},
	}
	if err := cmd.Run(context.Background(), []string{"test"}); err == nil || !strings.Contains(err.Error(), "--passphrase or set GURL_IMPORT_PASSPHRASE") {
		t.Fatalf("want an actionable passphrase error, got %v", err)
	}
}

// agentRoot is a gurl root with the global flags, as in cmd/gurl.
func agentRoot(db storage.DB, envs env.Store) *cli.Command {
	root := &cli.Command{Name: "gurl", Flags: GlobalFlags(), Commands: []*cli.Command{RunCommand(db, envs), ListCommand(db), EnvCommand(envs), ShowCommand(db)}}
	root.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	return root
}

func TestAgentFlagAndEnvPrintJSONForRunAndList(t *testing.T) {
	server := quietServer(t)
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "ok", Method: "GET", URL: server.URL})

	cases := map[string]struct {
		env  string
		args []string
	}{
		"run --agent":     {args: []string{"gurl", "--agent", "run", "ok"}},
		"run GURL_AGENT":  {env: "1", args: []string{"gurl", "run", "ok"}},
		"list --agent":    {args: []string{"gurl", "list", "--agent"}},
		"list GURL_AGENT": {env: "true", args: []string{"gurl", "list"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			clearAgentEnv(t)
			t.Setenv(AgentEnvVar, tc.env)
			var err error
			stdout, _ := captureStreams(t, func() {
				err = agentRoot(db, env.NewMemoryStore()).Run(context.Background(), tc.args)
			})
			if err != nil {
				t.Fatalf("%v", err)
			}
			if !json.Valid([]byte(stdout)) {
				t.Fatalf("stdout is not JSON:\n%s", stdout)
			}
			if strings.HasPrefix(name, "run") && decodeOne(t, stdout)["request"] != "ok" {
				t.Fatalf("unexpected run output: %s", stdout)
			}
		})
	}
}

func TestAgentModeRejectsReveal(t *testing.T) {
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "r", URL: "http://example.test", Headers: []types.Header{{Key: "Authorization", Value: "Bearer literal-secret-42"}}})
	envs := secretEnvStore(t)

	for _, args := range [][]string{
		{"gurl", "--agent", "list", "--reveal"},
		{"gurl", "--agent", "show", "r", "--reveal"},
		{"gurl", "--agent", "env", "show", "dev", "--reveal"},
	} {
		t.Run(strings.Join(args[2:], " "), func(t *testing.T) {
			clearAgentEnv(t)
			var err error
			stdout, stderr := captureStreams(t, func() {
				err = agentRoot(db, envs).Run(context.Background(), args)
			})
			if err == nil || !strings.Contains(err.Error(), "--reveal is not allowed in agent mode") {
				t.Fatalf("want --reveal rejected, got %v", err)
			}
			if strings.Contains(stdout+stderr, "literal-secret-42") || strings.Contains(stdout+stderr, agentTestSecret) {
				t.Fatalf("secret printed:\n%s%s", stdout, stderr)
			}
			if got := decodeOne(t, stdout); got["ok"] != false {
				t.Fatalf("want a JSON error object, got %v", got)
			}
		})
	}
}

func TestDocsSectionsNameEveryProblemKindAndFailurePhase(t *testing.T) {
	var runJSON string
	for _, s := range docSections {
		if s.Key == "run_json" {
			runJSON = s.Body
		}
	}
	for _, name := range []string{
		string(resolve.ProblemMissing), string(resolve.ProblemLeftoverPlaceholder), string(resolve.ProblemBadType), string(resolve.ProblemUnknownDynamic), string(resolve.ProblemAuth),
		runner.FailurePhaseRunIf, runner.FailurePhaseRequestBuild, runner.FailurePhaseHTTP, runner.FailurePhasePreRequestScript, runner.FailurePhasePostResponseScript,
		runner.FailurePhaseAssertion, runner.FailurePhaseNextRequest, runner.FailurePhaseExtraction, runner.FailurePhaseStatus,
	} {
		if !strings.Contains(runJSON, name) {
			t.Errorf("docs run_json section does not name %q", name)
		}
	}
}

func TestExecAgentModeMasksChildOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	store := env.NewMemoryStore()
	e := env.NewEnvironment("dev", "")
	e.SetSecretVariable("API_KEY", "sk_live_exec_secret")
	if err := store.SaveEnv(e); err != nil {
		t.Fatal(err)
	}
	cmd := &cli.Command{Name: "gurl", Flags: []cli.Flag{&cli.BoolFlag{Name: "agent"}}, Commands: []*cli.Command{ExecCommand(store)}}
	output := captureStdoutAndStderr(t, func() {
		if err := cmd.Run(context.Background(), []string{"gurl", "--agent", "exec", "--env", "dev", "--", "sh", "-c", `echo "key=$API_KEY"; echo "err=$API_KEY" >&2`}); err != nil {
			t.Fatalf("exec failed: %v", err)
		}
	})
	if strings.Contains(output, "sk_live_exec_secret") || !strings.Contains(output, "key=*****") || !strings.Contains(output, "err=*****") {
		t.Fatalf("expected masked child output, got:\n%s", output)
	}
}
