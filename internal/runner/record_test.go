package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/pkg/types"
)

func TestCollectionRunRecordsEveryRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/broken" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	db := newMockDB()
	db.SaveRequest(&types.SavedRequest{ID: "req-ok", Name: "ok", URL: ts.URL + "/ok", Method: "GET", Collection: "col", SortOrder: 1})
	db.SaveRequest(&types.SavedRequest{ID: "req-broken", Name: "broken", URL: ts.URL + "/broken", Method: "GET", Collection: "col", SortOrder: 2})
	db.SaveRequest(&types.SavedRequest{ID: "req-assert", Name: "assert", URL: ts.URL + "/ok", Method: "GET", Collection: "col", SortOrder: 3,
		Assertions: []types.Assertion{{Field: "status", Op: "equals", Value: "201"}}})
	envs := env.NewMemoryStore()
	staging := env.NewEnvironment("staging", "")
	staging.SetSecretVariable("TOKEN", "tok_staging_1")
	if err := envs.SaveEnv(staging); err != nil {
		t.Fatal(err)
	}

	if _, err := NewRunner(db, envs).Run(context.Background(), RunConfig{CollectionName: "col", Environment: "staging"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(db.history) != 3 {
		t.Fatalf("want 3 records, got %d", len(db.history))
	}
	byName := make(map[string]*types.ExecutionHistory)
	for _, rec := range db.history {
		if rec.CollectionName != "col" || rec.EnvName != "staging" {
			t.Errorf("record %s missing collection/env: %+v", rec.RequestName, rec)
		}
		byName[rec.RequestName] = rec
	}
	if byName["ok"].Failed() || FailureNotice(byName["ok"]) != "" {
		t.Errorf("successful request recorded as failed: %+v", byName["ok"])
	}
	if got := FailureNotice(byName["broken"]); !strings.HasPrefix(got, "request failed: HTTP 500 (id: "+byName["broken"].ShortID()+")") {
		t.Errorf("unexpected notice %q", got)
	}
	if rec := byName["assert"]; rec.FailurePhase != FailurePhaseAssertion || len(rec.Assertions) != 1 || rec.Assertions[0].Passed {
		t.Errorf("assertion failure not recorded: %+v", rec)
	}
	if got := FailureNotice(byName["assert"]); !strings.Contains(got, "assertion failed") {
		t.Errorf("unexpected notice %q", got)
	}
}
