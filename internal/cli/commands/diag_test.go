package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/internal/storage/memstore"
	"github.com/sreeram/gurl/pkg/types"
)

var failureNotice = regexp.MustCompile(`request failed: (.+) \(id: ([0-9a-f]{8})\) — inspect with: gurl diag ([0-9a-f]{8})`)

// captureStreams runs fn and returns what it wrote to stdout and stderr.
func captureStreams(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	read := func(r *os.File, into *string, done chan<- struct{}) {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		*into = buf.String()
		close(done)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outDone, errDone := make(chan struct{}), make(chan struct{})
	go read(outR, &stdout, outDone)
	go read(errR, &stderr, errDone)

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	fn()
	os.Stdout, os.Stderr = oldOut, oldErr
	_ = outW.Close()
	_ = errW.Close()
	<-outDone
	<-errDone
	return stdout, stderr
}

// noticeID returns the short id in the failure notice in stderr.
func noticeID(t *testing.T, stderr string) string {
	t.Helper()
	m := failureNotice.FindStringSubmatch(stderr)
	if m == nil {
		t.Fatalf("no failure notice in stderr:\n%s", stderr)
	}
	if m[2] != m[3] {
		t.Fatalf("notice ids differ: %q", m[0])
	}
	return m[2]
}

func runGurl(t *testing.T, db storage.DB, envs env.Store, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	stdout, stderr = captureStreams(t, func() {
		err = RunCommand(db, envs).Run(context.Background(), append([]string{"run"}, args...))
	})
	return stdout, stderr, err
}

func diagGurl(t *testing.T, db storage.DB, args ...string) string {
	t.Helper()
	var err error
	stdout, _ := captureStreams(t, func() {
		err = DiagCommand(db).Run(context.Background(), append([]string{"diag"}, args...))
	})
	if err != nil {
		t.Fatalf("diag %v: %v", args, err)
	}
	return stdout
}

func saveTestRequest(t *testing.T, db storage.DB, req *types.SavedRequest) {
	t.Helper()
	if err := db.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
}

func TestRunFailureNoticeNamesRecordDiagCanLoad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "always 200", Method: "GET", URL: server.URL + "/health"})

	stdout, stderr, err := runGurl(t, db, env.NewMemoryStore(), "always 200")
	if err != nil {
		t.Fatalf("a 500 without --fail should not error, got %v", err)
	}
	if stdout != `{"error":"boom"}` {
		t.Fatalf("stdout changed: %q", stdout)
	}
	id := noticeID(t, stderr)
	if !strings.Contains(stderr, "request failed: HTTP 500 (id: "+id+")") {
		t.Fatalf("unexpected notice:\n%s", stderr)
	}

	for _, args := range [][]string{{id}, {"--id", id}} {
		out := diagGurl(t, db, args...)
		for _, want := range []string{"Summary", "always 200", "FAILED: HTTP 500", "GET " + server.URL + "/health", `"error": "boom"`, "No earlier successful execution"} {
			if !strings.Contains(out, want) {
				t.Fatalf("diag %v missing %q:\n%s", args, want, out)
			}
		}
	}
}

func TestRunSuccessPrintsNoNoticeAndRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`ok`))
	}))
	defer server.Close()

	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "fine", Method: "GET", URL: server.URL})

	stdout, stderr, err := runGurl(t, db, env.NewMemoryStore(), "fine")
	if err != nil || stdout != "ok" || stderr != "" {
		t.Fatalf("err=%v stdout=%q stderr=%q", err, stdout, stderr)
	}
	req, _ := db.GetRequestByName("fine")
	history, _ := db.GetHistory(req.ID, 0)
	if len(history) != 1 || history[0].Failed() || history[0].RequestName != "fine" || history[0].URL != server.URL {
		t.Fatalf("unexpected record: %+v", history)
	}
}

func TestRunRecordsTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedURL := server.URL
	server.Close()

	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "down", Method: "GET", URL: closedURL + "/x"})

	_, stderr, err := runGurl(t, db, env.NewMemoryStore(), "down")
	if err == nil {
		t.Fatal("expected a transport error")
	}
	rec, getErr := db.GetHistoryByID(noticeID(t, stderr))
	if getErr != nil {
		t.Fatalf("GetHistoryByID: %v", getErr)
	}
	if rec.FailurePhase != runner.FailurePhaseHTTP || rec.Error == "" || rec.StatusCode != 0 || rec.URL != closedURL+"/x" {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestRunRecordsResolutionProblem(t *testing.T) {
	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "unresolved", Method: "POST", URL: "{{BASE_URL}}/orders"})

	_, stderr, err := runGurl(t, db, env.NewMemoryStore(), "unresolved")
	if err == nil || !strings.Contains(err.Error(), "BASE_URL") {
		t.Fatalf("expected missing BASE_URL error, got %v", err)
	}
	rec, getErr := db.GetHistoryByID(noticeID(t, stderr))
	if getErr != nil {
		t.Fatalf("GetHistoryByID: %v", getErr)
	}
	if rec.FailurePhase != runner.FailurePhaseRequestBuild || !strings.Contains(rec.Error, "BASE_URL") || rec.URL != "{{BASE_URL}}/orders" || rec.Method != "POST" {
		t.Fatalf("unexpected record: %+v", rec)
	}
	if !strings.Contains(stderr, "missing required variables: BASE_URL") {
		t.Fatalf("notice should give the reason:\n%s", stderr)
	}
}

func TestRunNeverStoresSecrets(t *testing.T) {
	const secret = "sk_live_never_stored"
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Api-Key")
		w.Header().Set("Set-Cookie", "session=abc")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"denied"}`))
	}))
	defer server.Close()

	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := db.Open(); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	saveTestRequest(t, db, &types.SavedRequest{
		Name:   "charge",
		Method: "POST",
		URL:    "{{BASE_URL}}/charges?k={{API_KEY}}&page=1",
		Headers: []types.Header{
			{Key: "X-Api-Key", Value: "{{API_KEY}}"},
			{Key: "X-Client", Value: "c-{{API_KEY}}"},
		},
		Body: `{"key":"{{API_KEY}}","amount":5}`,
	})
	envs := env.NewMemoryStore()
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("BASE_URL", server.URL)
	beta.SetSecretVariable("API_KEY", secret)
	if err := envs.SaveEnv(beta); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runGurl(t, db, envs, "charge", "--env", "beta")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotKey != secret {
		t.Fatalf("the request itself must carry the secret, got %q", gotKey)
	}
	id := noticeID(t, stderr)
	if strings.Contains(stderr, secret) {
		t.Fatalf("stderr leaked the secret:\n%s", stderr)
	}

	iter := db.DB.NewIterator(nil, nil)
	for iter.Next() {
		if bytes.Contains(iter.Value(), []byte(secret)) {
			t.Fatalf("stored key %s contains the secret:\n%s", iter.Key(), iter.Value())
		}
	}
	iter.Release()

	text, js := diagGurl(t, db, id), diagGurl(t, db, id, "--json")
	for _, out := range []string{text, js} {
		if strings.Contains(out, secret) || strings.Contains(out, "session=abc") {
			t.Fatalf("diag leaked a secret:\n%s", out)
		}
	}
	for _, want := range []string{"?k=*****&page=1", "X-Api-Key: *****", "X-Client: c-*****", `"key": "*****"`, "API_KEY", "Set-Cookie: *****"} {
		if !strings.Contains(text, want) {
			t.Fatalf("diag missing %q:\n%s", want, text)
		}
	}
}

func TestDiagComparesWithLastSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/us") {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"down","region":"us"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok","id":1}`))
	}))
	defer server.Close()

	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{
		Name:    "orders",
		Method:  "GET",
		URL:     "{{BASE_URL}}/{{REGION}}/orders",
		Headers: []types.Header{{Key: "Authorization", Value: "Bearer {{TOKEN}}"}},
	})
	envs := env.NewMemoryStore()
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("BASE_URL", server.URL)
	beta.SetVariable("REGION", "eu")
	beta.SetSecretVariable("TOKEN", "tok-secret")
	if err := envs.SaveEnv(beta); err != nil {
		t.Fatal(err)
	}

	if _, stderr, err := runGurl(t, db, envs, "orders", "--env", "beta"); err != nil || stderr != "" {
		t.Fatalf("first run: err=%v stderr=%q", err, stderr)
	}
	_, stderr, err := runGurl(t, db, envs, "orders", "--env", "beta", "--var", "REGION=us", "--var", "TOKEN=cli-secret")
	if err != nil {
		t.Fatal(err)
	}
	id := noticeID(t, stderr)

	out := diagGurl(t, db, id)
	for _, want := range []string{
		"Compared with last success",
		"status: 200 OK -> 503 Service Unavailable",
		"url: \"" + server.URL + "/eu/orders\" -> \"" + server.URL + "/us/orders\"",
		`var REGION: origin environment -> cli, value "eu" -> "us"`,
		"var TOKEN: origin environment -> cli (secret; values are masked and not compared)",
		"response header added: Retry-After",
		`response field "status": "ok" -> "down"`,
		`response field "region": added "us"`,
		`response field "id": removed`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("diag missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "tok-secret") || strings.Contains(out, "cli-secret") {
		t.Fatalf("diag leaked a secret:\n%s", out)
	}
	if strings.Contains(out, "var BASE_URL") {
		t.Fatalf("unchanged variable reported:\n%s", out)
	}

	var report struct {
		Record       types.ExecutionHistory `json:"record"`
		ComparedWith struct {
			BaselineID     string `json:"baseline_id"`
			BaselineStatus int    `json:"baseline_status"`
			Status         int    `json:"status"`
			Vars           []struct {
				Name         string `json:"name"`
				BeforeOrigin string `json:"before_origin"`
				AfterOrigin  string `json:"after_origin"`
			} `json:"vars"`
		} `json:"compared_with"`
	}
	if err := json.Unmarshal([]byte(diagGurl(t, db, "--json", id)), &report); err != nil {
		t.Fatalf("--json output is not valid JSON: %v", err)
	}
	if report.Record.ShortID() != id || report.ComparedWith.BaselineStatus != 200 || report.ComparedWith.Status != 503 || report.ComparedWith.BaselineID == "" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.ComparedWith.Vars) != 2 || report.ComparedWith.Vars[0].Name != "REGION" || report.ComparedWith.Vars[0].AfterOrigin != "cli" {
		t.Fatalf("unexpected var changes: %+v", report.ComparedWith.Vars)
	}
}

func TestRunDataDrivenRecordsEveryRow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("n") == "2" {
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	defer server.Close()

	db := memstore.New()
	saveTestRequest(t, db, &types.SavedRequest{Name: "rows", Method: "GET", URL: server.URL + "/?n={{n}}"})
	data := filepath.Join(t.TempDir(), "rows.csv")
	if err := os.WriteFile(data, []byte("n\n1\n2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runGurl(t, db, env.NewMemoryStore(), "rows", "--data", data)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(failureNotice.FindAllString(stderr, -1)); got != 1 {
		t.Fatalf("want one failure notice, got %d:\n%s", got, stderr)
	}
	req, _ := db.GetRequestByName("rows")
	history, _ := db.GetHistory(req.ID, 0)
	if len(history) != 2 || history[0].StatusCode != 502 || history[1].StatusCode != 200 {
		t.Fatalf("want a record per row, got %+v", history)
	}
}
