package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/cookies"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

const runCookieSecret = "sess-run-4b8e2a"

// emptyCookieJar returns a persistent cookie jar in a temp directory.
func emptyCookieJar(t *testing.T) *cookies.CookieJar {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GURL_COOKIE_DB_PATH", filepath.Join(dir, "cookies.db"))
	cdb, err := cookies.NewLMDBCookieDB()
	if err != nil {
		t.Fatal(err)
	}
	jar, err := cookies.NewCookieJar(cdb)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cdb.Close() })
	return jar
}

// TestRunCommandStoresAndSendsCookiesThroughSharedJar runs two saved
// requests one after the other: the first response sets a cookie, the second
// request must carry it, and `cookies list` must show it masked.
func TestRunCommandStoresAndSendsCookiesThroughSharedJar(t *testing.T) {
	var gotCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: runCookieSecret, Path: "/"})
		case "/me":
			if c, err := r.Cookie("session"); err == nil {
				gotCookie = c.Value
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	db := newMockDB()
	for _, req := range []*types.SavedRequest{
		{ID: "req-login", Name: "login", URL: server.URL + "/login", Method: "GET"},
		{ID: "req-me", Name: "me", URL: server.URL + "/me", Method: "GET"},
	} {
		if err := db.SaveRequest(req); err != nil {
			t.Fatal(err)
		}
	}

	jar := emptyCookieJar(t)
	for _, name := range []string{"login", "me"} {
		cmd := RunCommand(db, &env.EnvStorage{}, runner.WithCookieJar(jar))
		captureStdoutAndStderr(t, func() {
			if err := cmd.Run(context.Background(), []string{"run", name}); err != nil {
				t.Fatalf("run %s: %v", name, err)
			}
		})
	}

	if gotCookie != runCookieSecret {
		t.Fatalf("second run sent session cookie %q, want %q", gotCookie, runCookieSecret)
	}

	if len(db.history) != 2 {
		t.Fatalf("expected 2 execution records, got %d", len(db.history))
	}
	for _, rec := range db.history {
		raw, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), runCookieSecret) {
			t.Fatalf("execution record stores the cookie value unmasked:\n%s", raw)
		}
	}

	root := &cli.Command{Name: "gurl", Flags: GlobalFlags(), Commands: []*cli.Command{CookiesCommand(testCookieJar{jar})}}
	root.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	clearAgentEnv(t)
	var listErr error
	stdout, stderr := captureStreams(t, func() {
		listErr = root.Run(context.Background(), []string{"gurl", "cookies", "list", "--json"})
	})
	if listErr != nil {
		t.Fatalf("cookies list: %v\nstderr: %s", listErr, stderr)
	}
	if strings.Contains(stdout+stderr, runCookieSecret) {
		t.Fatalf("cookies list shows the cookie value:\n%s%s", stdout, stderr)
	}
	var listed []cookieJSON
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatalf("stdout is not a JSON cookie list: %v\n%s", err, stdout)
	}
	if len(listed) != 1 || listed[0].Name != "session" || listed[0].Value != redact.Mask {
		t.Fatalf("got %+v, want one masked session cookie", listed)
	}
}
