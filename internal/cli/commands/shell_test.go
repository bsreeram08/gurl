package commands

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestPromptSelectRequestByIndex(t *testing.T) {
	requests := []*types.SavedRequest{
		{Name: "bravo", Method: "POST", URL: "https://example.com/b"},
		{Name: "alpha", Method: "GET", URL: "https://example.com/a"},
	}

	out := &bytes.Buffer{}
	name, err := promptSelectRequest(bufio.NewReader(strings.NewReader("1\n")), out, requests, "Select: ")
	if err != nil {
		t.Fatalf("promptSelectRequest returned error: %v", err)
	}
	if name != "alpha" {
		t.Fatalf("expected sorted selection to return alpha, got %q", name)
	}
	if !strings.Contains(out.String(), "Select: ") {
		t.Fatalf("expected prompt output, got %q", out.String())
	}
}

func TestShellHistorySkipsWhenDisabled(t *testing.T) {
	history := &shellHistory{max: 3}

	history.Add("list")
	if history.Len() != 0 {
		t.Fatalf("expected disabled history to ignore entries, got %d", history.Len())
	}

	history.enabled = true
	history.Add("list")
	history.Add("list")
	history.Add("open test")
	history.Add("send")
	history.Add("quit")

	if history.Len() != 3 {
		t.Fatalf("expected history to cap at 3 entries, got %d", history.Len())
	}
	if history.At(0) != "quit" || history.At(1) != "send" || history.At(2) != "open test" {
		t.Fatalf("unexpected history order: %#v", history.entries)
	}
}

func TestShellSessionNewSaveAndOpen(t *testing.T) {
	db := newMockDB()
	out := &bytes.Buffer{}
	session := newShellSession(db, nil, strings.NewReader(""), out, "auto")

	steps := []string{
		"new health",
		"set method POST",
		"set url https://example.com/api",
		"header add Authorization Bearer token",
		"query add page 1",
		"auth bearer secret-token",
		"set timeout 5s",
		"save",
		"open health",
	}

	for _, step := range steps {
		if _, err := session.executeLine(step); err != nil {
			t.Fatalf("step %q failed: %v", step, err)
		}
	}

	req, err := db.GetRequestByName("health")
	if err != nil {
		t.Fatalf("GetRequestByName returned error: %v", err)
	}
	if req.Method != "POST" {
		t.Fatalf("expected POST method, got %q", req.Method)
	}
	if req.URL != "https://example.com/api?page=1" {
		t.Fatalf("unexpected URL %q", req.URL)
	}
	if len(req.Headers) != 1 || req.Headers[0].Key != "Authorization" || req.Headers[0].Value != "Bearer token" {
		t.Fatalf("unexpected headers: %#v", req.Headers)
	}
	if req.AuthConfig == nil || req.AuthConfig.Type != "bearer" || req.AuthConfig.Params["token"] != "secret-token" {
		t.Fatalf("unexpected auth config: %#v", req.AuthConfig)
	}
	if req.Timeout != "5s" {
		t.Fatalf("expected timeout 5s, got %q", req.Timeout)
	}
	if session.current == nil || session.current.Name != "health" {
		t.Fatalf("expected current request to be reopened, got %#v", session.current)
	}
}

func TestShellSessionDirtyQuitRequiresForce(t *testing.T) {
	db := newMockDB()
	session := newShellSession(db, nil, strings.NewReader(""), &bytes.Buffer{}, "auto")

	if _, err := session.executeLine("new draft"); err != nil {
		t.Fatalf("new draft failed: %v", err)
	}

	shouldExit, err := session.executeLine("quit")
	if err == nil {
		t.Fatal("expected unsaved changes error")
	}
	if shouldExit {
		t.Fatal("quit should not exit when the draft is dirty")
	}
	if !strings.Contains(err.Error(), "unsaved changes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShellSessionPromptForMissingVars(t *testing.T) {
	out := &bytes.Buffer{}
	session := newShellSession(newMockDB(), nil, strings.NewReader("10\nUSD\n"), out, "auto")

	vars := map[string]string{}
	if err := session.promptForMissingVars(vars, []string{"amount", "currency"}); err != nil {
		t.Fatalf("promptForMissingVars returned error: %v", err)
	}

	if vars["amount"] != "10" || vars["currency"] != "USD" {
		t.Fatalf("unexpected prompted vars: %#v", vars)
	}
	if !strings.Contains(out.String(), "amount: ") || !strings.Contains(out.String(), "currency: ") {
		t.Fatalf("expected prompts in output, got %q", out.String())
	}
}

func TestShellSessionUnknownCommandSuggestsHelp(t *testing.T) {
	session := newShellSession(newMockDB(), nil, strings.NewReader(""), &bytes.Buffer{}, "auto")

	_, err := session.executeLine("snd")
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "Did you mean: send") {
		t.Fatalf("expected send suggestion, got %v", err)
	}
}

func TestShellSessionHelpTopic(t *testing.T) {
	out := &bytes.Buffer{}
	session := newShellSession(newMockDB(), nil, strings.NewReader(""), out, "auto")

	if _, err := session.executeLine("help send"); err != nil {
		t.Fatalf("help send failed: %v", err)
	}
	if !strings.Contains(out.String(), "usage: send [key=value ...]") {
		t.Fatalf("expected detailed help output, got %q", out.String())
	}
}

func TestBuildClientRequestFromSavedRequestResolvesTemplatesAndAuth(t *testing.T) {
	req := &types.SavedRequest{
		Name:   "templated",
		Method: "POST",
		URL:    "https://api.example.com/users/:id",
		Headers: []types.Header{
			{Key: "X-Tenant", Value: "{{tenant}}"},
		},
		Body: `{"name":"{{name}}"}`,
		PathParams: []types.Var{
			{Name: "id"},
		},
		AuthConfig: &types.AuthConfig{
			Type: "bearer",
			Params: map[string]string{
				"token": "{{token}}",
			},
		},
		Timeout: "3s",
	}

	clientReq, err := buildClientRequestFromSavedRequest(req, map[string]string{
		"id":     "42",
		"tenant": "acme",
		"name":   "sreeram",
		"token":  "abc123",
	})
	if err != nil {
		t.Fatalf("buildClientRequestFromSavedRequest returned error: %v", err)
	}

	if clientReq.URL != "https://api.example.com/users/42" {
		t.Fatalf("unexpected URL %q", clientReq.URL)
	}
	if clientReq.Body != `{"name":"sreeram"}` {
		t.Fatalf("unexpected body %q", clientReq.Body)
	}
	if clientReq.Timeout.String() != "3s" {
		t.Fatalf("unexpected timeout %s", clientReq.Timeout)
	}
	if len(clientReq.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %#v", clientReq.Headers)
	}
}
