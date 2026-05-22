package runner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/assertions"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

type mockDB struct {
	requests map[string]*types.SavedRequest
	names    map[string]string
	history  []*types.ExecutionHistory
}

func newMockDB() *mockDB {
	return &mockDB{
		requests: make(map[string]*types.SavedRequest),
		names:    make(map[string]string),
	}
}

func (m *mockDB) Open() error  { return nil }
func (m *mockDB) Close() error { return nil }
func (m *mockDB) SaveRequest(req *types.SavedRequest) error {
	if req.ID == "" {
		req.ID = "test-id-1"
	}
	m.requests[req.ID] = req
	m.names[req.Name] = req.ID
	return nil
}
func (m *mockDB) GetRequest(id string) (*types.SavedRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return nil, nil
	}
	return req, nil
}
func (m *mockDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	id, ok := m.names[name]
	if !ok {
		return nil, errors.New("request not found: " + name)
	}
	req := m.requests[id]
	copy := *req
	return &copy, nil
}
func (m *mockDB) ListRequests(opts *storage.ListOptions) ([]*types.SavedRequest, error) {
	var result []*types.SavedRequest
	for _, req := range m.requests {
		if opts != nil && opts.Collection != "" {
			if req.Collection != opts.Collection {
				continue
			}
		}
		result = append(result, req)
	}
	return result, nil
}
func (m *mockDB) DeleteRequest(id string) error {
	req, ok := m.requests[id]
	if !ok {
		return nil
	}
	delete(m.names, req.Name)
	delete(m.requests, id)
	return nil
}
func (m *mockDB) UpdateRequest(req *types.SavedRequest) error {
	if req.ID == "" {
		return nil
	}
	oldName := ""
	if oldReq, ok := m.requests[req.ID]; ok {
		oldName = oldReq.Name
	}
	if oldName != "" && oldName != req.Name {
		delete(m.names, oldName)
	}
	m.requests[req.ID] = req
	m.names[req.Name] = req.ID
	return nil
}
func (m *mockDB) SaveHistory(history *types.ExecutionHistory) error {
	m.history = append(m.history, history)
	return nil
}
func (m *mockDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	return nil, nil
}
func (m *mockDB) ListFolder(path string) ([]*types.SavedRequest, error)          { return nil, nil }
func (m *mockDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) { return nil, nil }
func (m *mockDB) DeleteFolder(path string) error                                 { return nil }
func (m *mockDB) GetAllFolders() ([]string, error)                               { return nil, nil }

type mockEnvStorage struct {
	envs map[string]*env.Environment
}

func newMockEnvStorage() *mockEnvStorage {
	return &mockEnvStorage{
		envs: make(map[string]*env.Environment),
	}
}

func (m *mockEnvStorage) GetEnvByName(name string) (*env.Environment, error) {
	e, ok := m.envs[name]
	if !ok {
		return nil, errors.New("environment not found: " + name)
	}
	return e, nil
}

func (m *mockEnvStorage) GetActiveEnv() (string, error) {
	return "", nil
}

func (m *mockEnvStorage) SaveEnv(e *env.Environment) error {
	m.envs[e.Name] = e
	return nil
}

func (m *mockEnvStorage) ListEnvs() ([]*env.Environment, error) {
	var result []*env.Environment
	for _, e := range m.envs {
		result = append(result, e)
	}
	return result, nil
}

func (m *mockEnvStorage) DeleteEnv(name string) error {
	delete(m.envs, name)
	return nil
}

func TestRunner_RunCollection(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "req1",
		URL:        ts.URL + "/get",
		Method:     "GET",
		Collection: "test-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "test-col",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Total != 1 {
		t.Errorf("expected Total=1, got %d", result.Total)
	}
	if result.Passed != 1 {
		t.Errorf("expected Passed=1, got %d", result.Passed)
	}
	if len(result.RequestResults) != 1 {
		t.Errorf("expected 1 request result, got %d", len(result.RequestResults))
	}
}

func TestRequestResultLifecycleMetadata(t *testing.T) {
	tests := []struct {
		name                string
		result              RequestResult
		extractedVars       map[string]string
		dirtyVars           map[string]string
		skipReason          string
		failurePhase        string
		nextRequestOverride string
		passed              bool
		skipped             bool
	}{
		{
			name: "success carries extracted and dirty vars",
			result: RequestResult{
				RequestName:   "login",
				Passed:        true,
				ExtractedVars: map[string]string{"token": "abc123"},
				DirtyVars:     map[string]string{"token": "abc123"},
			},
			extractedVars: map[string]string{"token": "abc123"},
			dirtyVars:     map[string]string{"token": "abc123"},
			passed:        true,
		},
		{
			name: "skipped request records neutral skip reason",
			result: RequestResult{
				RequestName: "optional-step",
				Skipped:     true,
				SkipReason:  SkipReasonRunIf,
			},
			skipReason: SkipReasonRunIf,
			skipped:    true,
		},
		{
			name: "assertion failure records assertion phase",
			result: RequestResult{
				RequestName:  "check-order",
				FailurePhase: FailurePhaseAssertion,
				AssertionResults: []assertions.Result{
					{
						Assertion: assertions.Assertion{Field: "status_code", Op: "equals", Value: "201"},
						Passed:    false,
						Actual:    "200",
						Expected:  "201",
					},
				},
			},
			failurePhase: FailurePhaseAssertion,
		},
		{
			name: "script failure records pre request script phase",
			result: RequestResult{
				RequestName:  "prepare-order",
				Error:        "script failed: boom",
				FailurePhase: FailurePhasePreRequestScript,
			},
			failurePhase: FailurePhasePreRequestScript,
		},
		{
			name: "next request override records target request name",
			result: RequestResult{
				RequestName:         "route-order",
				Passed:              true,
				NextRequestOverride: "confirm-order",
			},
			nextRequestOverride: "confirm-order",
			passed:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Passed != tt.passed {
				t.Fatalf("expected Passed=%v, got %v", tt.passed, tt.result.Passed)
			}
			if tt.result.Skipped != tt.skipped {
				t.Fatalf("expected Skipped=%v, got %v", tt.skipped, tt.result.Skipped)
			}
			assertStringMap(t, "ExtractedVars", tt.result.ExtractedVars, tt.extractedVars)
			assertStringMap(t, "DirtyVars", tt.result.DirtyVars, tt.dirtyVars)
			if tt.result.SkipReason != tt.skipReason {
				t.Fatalf("expected SkipReason=%q, got %q", tt.skipReason, tt.result.SkipReason)
			}
			if tt.result.FailurePhase != tt.failurePhase {
				t.Fatalf("expected FailurePhase=%q, got %q", tt.failurePhase, tt.result.FailurePhase)
			}
			if tt.result.NextRequestOverride != tt.nextRequestOverride {
				t.Fatalf("expected NextRequestOverride=%q, got %q", tt.nextRequestOverride, tt.result.NextRequestOverride)
			}
		})
	}
}

func TestRequestResultLifecycleMetadataNames(t *testing.T) {
	tests := map[string]string{
		"skip run_if":              SkipReasonRunIf,
		"skip script":              SkipReasonScript,
		"skip bail":                SkipReasonBail,
		"run_if phase":             FailurePhaseRunIf,
		"request build phase":      FailurePhaseRequestBuild,
		"http phase":               FailurePhaseHTTP,
		"pre request script phase": FailurePhasePreRequestScript,
		"post response script":     FailurePhasePostResponseScript,
		"assertion phase":          FailurePhaseAssertion,
	}

	for name, got := range tests {
		t.Run(name, func(t *testing.T) {
			want := map[string]string{
				"skip run_if":              "run_if",
				"skip script":              "script",
				"skip bail":                "bail",
				"run_if phase":             "run_if",
				"request build phase":      "request_build",
				"http phase":               "http",
				"pre request script phase": "pre_request_script",
				"post response script":     "post_response_script",
				"assertion phase":          "assertion",
			}[name]
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})
	}
}

func TestRunner_PreScriptVariablesAvailableForTemplates(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	var gotPath string
	var gotTenantHeader string
	var gotBody string
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		gotPath = r.URL.Path
		gotTenantHeader = r.Header.Get("X-Tenant")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "pre-script-template",
		Name:       "pre-script-template",
		URL:        ts.URL + "/tenants/{{tenant}}",
		Method:     "POST",
		Collection: "script-flow",
		Headers: []types.Header{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "X-Tenant", Value: "{{tenant}}"},
		},
		Body:      `{"tenant":"{{tenant}}"}`,
		PreScript: `gurl.setVar("tenant", "acme")`,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "script-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestCount != 1 {
		t.Fatalf("expected exactly one HTTP request, got %d", requestCount)
	}
	if gotPath != "/tenants/acme" {
		t.Fatalf("expected pre-script variable in URL path, got %q", gotPath)
	}
	if gotTenantHeader != "acme" {
		t.Fatalf("expected pre-script variable in header, got %q", gotTenantHeader)
	}
	if gotBody != `{"tenant":"acme"}` {
		t.Fatalf("expected pre-script variable in body, got %q", gotBody)
	}
	if results[0].Passed != 1 || results[0].Failed != 0 {
		t.Fatalf("expected request to pass, got result: %+v", results[0])
	}
}

func TestRunner_PostScriptVariablesAvailableForNextRequestAndExtractAssertion(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	var gotAuthorization string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login":
			json.NewEncoder(w).Encode(map[string]string{"token": "Bearer abc123"})
		case "/profile":
			gotAuthorization = r.Header.Get("Authorization")
			json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "login",
		Name:       "login",
		URL:        ts.URL + "/login",
		Method:     "GET",
		Collection: "post-script-flow",
		PostScript: `
			var data = gurl.response.json();
			gurl.setVar("authToken", data.token);
			gurl.setVar("scriptToken", "scripted");
		`,
		Assertions: []types.Assertion{
			{Field: "extract:scriptToken", Op: "=", Value: "scripted"},
		},
		SortOrder: 1,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "profile",
		Name:       "profile",
		URL:        ts.URL + "/profile",
		Method:     "GET",
		Collection: "post-script-flow",
		Headers: []types.Header{
			{Key: "Authorization", Value: "{{authToken}}"},
		},
		SortOrder: 2,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "post-script-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuthorization != "Bearer abc123" {
		t.Fatalf("expected post-script variable in following request header, got %q", gotAuthorization)
	}
	firstResult := results[0].RequestResults[0]
	assertStringMap(t, "DirtyVars", firstResult.DirtyVars, map[string]string{"authToken": "Bearer abc123", "scriptToken": "scripted"})
	if len(firstResult.AssertionResults) != 1 || !firstResult.AssertionResults[0].Passed {
		t.Fatalf("expected extract assertion to see post-script variable, got %+v", firstResult.AssertionResults)
	}
	if results[0].Passed != 2 || results[0].Failed != 0 {
		t.Fatalf("expected both requests to pass, got result: %+v", results[0])
	}
}

func TestRunner_SkipRequestIsNeutralAndAvoidsHTTP(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusTeapot)
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "skip-me",
		Name:       "skip-me",
		URL:        ts.URL + "/skip",
		Method:     "GET",
		Collection: "skip-flow",
		PreScript:  `gurl.skipRequest("not needed for this data row")`,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "skip-flow", Bail: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestCount != 0 {
		t.Fatalf("expected skipped request to avoid HTTP, got %d calls", requestCount)
	}
	if len(db.history) != 0 {
		t.Fatalf("expected skipped request to avoid history writes, got %d", len(db.history))
	}
	result := results[0]
	if result.Passed != 0 || result.Failed != 0 || result.Skipped != 1 {
		t.Fatalf("expected neutral skip summary, got %+v", result)
	}
	requestResult := result.RequestResults[0]
	if !requestResult.Skipped || requestResult.Passed || requestResult.Error != "" {
		t.Fatalf("expected neutral skipped request result, got %+v", requestResult)
	}
	if requestResult.SkipReason != "not needed for this data row" {
		t.Fatalf("expected script skip reason to be preserved, got %q", requestResult.SkipReason)
	}
}

func TestRunner_ScriptErrorFailurePhases(t *testing.T) {
	t.Run("pre script error fails before HTTP", func(t *testing.T) {
		db := newMockDB()
		envStorage := newMockEnvStorage()

		requestCount := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		db.SaveRequest(&types.SavedRequest{
			ID:         "pre-error",
			Name:       "pre-error",
			URL:        ts.URL + "/pre",
			Method:     "GET",
			Collection: "pre-error-flow",
			PreScript:  `throw new Error("boom-pre")`,
		})

		runner := NewRunner(db, envStorage)
		results, err := runner.Run(context.Background(), RunConfig{CollectionName: "pre-error-flow"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requestCount != 0 {
			t.Fatalf("expected pre-script error to avoid HTTP, got %d calls", requestCount)
		}
		requestResult := results[0].RequestResults[0]
		if requestResult.FailurePhase != FailurePhasePreRequestScript {
			t.Fatalf("expected failure phase %q, got %q", FailurePhasePreRequestScript, requestResult.FailurePhase)
		}
		if !strings.Contains(requestResult.Error, "boom-pre") {
			t.Fatalf("expected error to include script error, got %q", requestResult.Error)
		}
	})

	t.Run("post script error fails after HTTP", func(t *testing.T) {
		db := newMockDB()
		envStorage := newMockEnvStorage()

		requestCount := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
		}))
		defer ts.Close()

		db.SaveRequest(&types.SavedRequest{
			ID:         "post-error",
			Name:       "post-error",
			URL:        ts.URL + "/post",
			Method:     "GET",
			Collection: "post-error-flow",
			PostScript: `throw new Error("boom-post")`,
		})

		runner := NewRunner(db, envStorage)
		results, err := runner.Run(context.Background(), RunConfig{CollectionName: "post-error-flow"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requestCount != 1 {
			t.Fatalf("expected post-script error after HTTP, got %d calls", requestCount)
		}
		requestResult := results[0].RequestResults[0]
		if requestResult.FailurePhase != FailurePhasePostResponseScript {
			t.Fatalf("expected failure phase %q, got %q", FailurePhasePostResponseScript, requestResult.FailurePhase)
		}
		if !strings.Contains(requestResult.Error, "boom-post") {
			t.Fatalf("expected error to include script error, got %q", requestResult.Error)
		}
	})
}

func assertStringMap(t *testing.T, name string, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %s length %d, got %d", name, len(want), len(got))
	}
	for key, wantValue := range want {
		if gotValue := got[key]; gotValue != wantValue {
			t.Fatalf("expected %s[%q]=%q, got %q", name, key, wantValue, gotValue)
		}
	}
}

func TestRunner_RunWithOrder(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		order = append(order, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": r.URL.Path})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-a",
		Name:       "request-a",
		URL:        ts.URL + "/first",
		Method:     "GET",
		Collection: "ordered-col",
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-b",
		Name:       "request-b",
		URL:        ts.URL + "/second",
		Method:     "GET",
		Collection: "ordered-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "ordered-col",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(order))
	}

	if order[0] != "/first" {
		t.Errorf("expected first request to /first, got %s", order[0])
	}
	if order[1] != "/second" {
		t.Errorf("expected second request to /second, got %s", order[1])
	}

	if results[0].Passed != 2 {
		t.Errorf("expected 2 passed, got %d", results[0].Passed)
	}
}

func TestRunner_ContinueOnError(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "first",
		URL:        ts.URL + "/first",
		Method:     "GET",
		Collection: "error-col",
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "error",
		URL:        ts.URL + "/error",
		Method:     "GET",
		Collection: "error-col",
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-3",
		Name:       "third",
		URL:        ts.URL + "/third",
		Method:     "GET",
		Collection: "error-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "error-col",
		Iterations:     1,
		Bail:           false,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Errorf("expected all 3 requests to run, got %d", len(order))
	}

	result := results[0]
	if result.Total != 3 {
		t.Errorf("expected Total=3, got %d", result.Total)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
	if result.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", result.Passed)
	}
}

func TestRunner_StopOnError(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "first",
		URL:        ts.URL + "/first",
		Method:     "GET",
		Collection: "bail-col",
		SortOrder:  1,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "error",
		URL:        ts.URL + "/error",
		Method:     "GET",
		Collection: "bail-col",
		SortOrder:  2,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-3",
		Name:       "third",
		URL:        ts.URL + "/third",
		Method:     "GET",
		Collection: "bail-col",
		SortOrder:  3,
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "bail-col",
		Iterations:     1,
		Bail:           true,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 2 {
		t.Errorf("expected only 2 requests to run (bail on error), got %d: %v", len(order), order)
	}

	result := results[0]
	if result.Total != 3 {
		t.Errorf("expected Total=3, got %d", result.Total)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped (after bail), got %d", result.Skipped)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
}

func TestRunner_BailMarksRemainingRequestsWithBailSkipReason(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{ID: "req-1", Name: "first", URL: ts.URL + "/first", Method: "GET", Collection: "bail-reason-col", SortOrder: 1})
	db.SaveRequest(&types.SavedRequest{ID: "req-2", Name: "fail", URL: ts.URL + "/fail", Method: "GET", Collection: "bail-reason-col", SortOrder: 2})
	db.SaveRequest(&types.SavedRequest{ID: "req-3", Name: "third", URL: ts.URL + "/third", Method: "GET", Collection: "bail-reason-col", SortOrder: 3})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "bail-reason-col", Bail: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected bail to stop before third request, got order %v", order)
	}
	if got := results[0].RequestResults[2].SkipReason; got != SkipReasonBail {
		t.Fatalf("expected remaining request skip reason %q, got %q", SkipReasonBail, got)
	}
}

func TestRunner_AssertBailStopsOnlyAfterAssertionFailure(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		if r.URL.Path == "/http-fail" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{ID: "req-1", Name: "http-fail", URL: ts.URL + "/http-fail", Method: "GET", Collection: "assert-bail-col", SortOrder: 1})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "assert-fail",
		URL:        ts.URL + "/assert-fail",
		Method:     "GET",
		Collection: "assert-bail-col",
		SortOrder:  2,
		Assertions: []types.Assertion{{Field: "status", Op: "equals", Value: "201"}},
	})
	db.SaveRequest(&types.SavedRequest{ID: "req-3", Name: "third", URL: ts.URL + "/third", Method: "GET", Collection: "assert-bail-col", SortOrder: 3})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "assert-bail-col", AssertBail: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := []string{"/http-fail", "/assert-fail"}; strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("expected assert-bail to ignore HTTP failure then stop after assertion failure, got order %v", order)
	}
	result := results[0]
	if result.Failed != 2 || result.Skipped != 1 {
		t.Fatalf("expected two failures and one assert-bail skip, got failed=%d skipped=%d", result.Failed, result.Skipped)
	}
	assertResult := result.RequestResults[1]
	if assertResult.FailurePhase != FailurePhaseAssertion {
		t.Fatalf("expected assertion failure phase, got %q", assertResult.FailurePhase)
	}
	if got := result.RequestResults[2].SkipReason; got != SkipReasonBail {
		t.Fatalf("expected remaining request skip reason %q, got %q", SkipReasonBail, got)
	}
}

func TestRunner_AssertBailDoesNotStopOnSkippedRequest(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{ID: "req-1", Name: "optional", URL: ts.URL + "/optional", Method: "GET", Collection: "assert-bail-skip-col", SortOrder: 1, RunIf: "run_optional == true"})
	db.SaveRequest(&types.SavedRequest{ID: "req-2", Name: "next", URL: ts.URL + "/next", Method: "GET", Collection: "assert-bail-skip-col", SortOrder: 2})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "assert-bail-skip-col", AssertBail: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Join(order, ",") != "/next" {
		t.Fatalf("expected skipped request to be neutral and next request to run, got order %v", order)
	}
	result := results[0]
	if result.Skipped != 1 || result.Passed != 1 || result.Failed != 0 {
		t.Fatalf("expected one skipped and one passed request, got passed=%d failed=%d skipped=%d", result.Passed, result.Failed, result.Skipped)
	}
}

func TestRunner_Summary(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/ok" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		} else if r.URL.Path == "/notfound" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "success-req",
		URL:        ts.URL + "/ok",
		Method:     "GET",
		Collection: "summary-col",
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "fail-req",
		URL:        ts.URL + "/notfound",
		Method:     "GET",
		Collection: "summary-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "summary-col",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := results[0]
	if result.Total != 2 {
		t.Errorf("expected Total=2, got %d", result.Total)
	}
	if result.Passed != 1 {
		t.Errorf("expected Passed=1, got %d", result.Passed)
	}
	if result.Failed != 1 {
		t.Errorf("expected Failed=1, got %d", result.Failed)
	}
	if result.Duration <= 0 {
		t.Errorf("expected positive duration, got %v", result.Duration)
	}
	if result.CollectionName != "summary-col" {
		t.Errorf("expected CollectionName=summary-col, got %s", result.CollectionName)
	}
}

func TestRunner_Variables(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"headers": r.Header,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	e := env.NewEnvironment("staging", "")
	e.SetVariable("baseUrl", ts.URL)
	e.SetVariable("apiKey", "secret-key")
	envStorage.SaveEnv(e)

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "var-test",
		URL:        "{{baseUrl}}/test",
		Method:     "GET",
		Collection: "var-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "var-col",
		Environment:    "staging",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Passed != 1 {
		t.Errorf("expected request to pass with variable substitution")
	}

	if results[0].RequestResults[0].Error != "" {
		t.Errorf("unexpected error: %s", results[0].RequestResults[0].Error)
	}
}

func TestRunner_Iterations(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"count": count})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "iter-test",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "iter-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "iter-col",
		Iterations:     3,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 iterations, got %d", len(results))
	}

	if count != 3 {
		t.Errorf("expected 3 total requests, got %d", count)
	}

	for i, r := range results {
		if r.Iteration != i+1 {
			t.Errorf("expected Iteration=%d, got %d", i+1, r.Iteration)
		}
		if r.Total != 1 {
			t.Errorf("iteration %d: expected Total=1, got %d", i+1, r.Total)
		}
	}
}

func TestRunner_Delay(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	start := time.Now()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "delay-1",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "delay-col",
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "delay-2",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "delay-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "delay-col",
		Iterations:     1,
		Delay:          50 * time.Millisecond,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected delay between requests, elapsed=%v", elapsed)
	}

	result := results[0]
	if result.Total != 2 {
		t.Errorf("expected 2 requests, got %d", result.Total)
	}
}

func TestRunner_EmptyCollection(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "nonexistent",
		Iterations:     1,
	}

	_, err := runner.Run(context.Background(), config)
	if err == nil {
		t.Error("expected error for empty collection")
	}
}

func TestRunner_SubstitutesHeadersAndBody(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	var gotHeader string
	var gotBody map[string]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("x-surfboard-merchant-id")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-template",
		Name:       "templated",
		URL:        ts.URL,
		Method:     "POST",
		Collection: "api",
		Headers: []types.Header{
			{Key: "x-surfboard-merchant-id", Value: "{{MERCHANT_ID}}"},
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"merchantId":"{{MERCHANT_ID}}"}`,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{
		CollectionName: "api",
		Vars:           map[string]string{"MERCHANT_ID": "merchant-123"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Failed != 0 {
		t.Fatalf("expected collection run to pass, got result: %+v", results[0])
	}
	if gotHeader != "merchant-123" {
		t.Errorf("expected substituted header, got %q", gotHeader)
	}
	if gotBody["merchantId"] != "merchant-123" {
		t.Errorf("expected substituted body, got %#v", gotBody)
	}
}

func TestRunner_ConvertHeaders(t *testing.T) {
	headers := []types.Header{
		{Key: "Content-Type", Value: "application/json"},
		{Key: "Authorization", Value: "Bearer token"},
	}

	result := convertHeaders(headers)
	if len(result) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(result))
	}
	if result[0].Key != "Content-Type" || result[0].Value != "application/json" {
		t.Errorf("unexpected header: %+v", result[0])
	}
	if result[1].Key != "Authorization" || result[1].Value != "Bearer token" {
		t.Errorf("unexpected header: %+v", result[1])
	}
}

func TestRunner_ConvertHeaders_Nil(t *testing.T) {
	result := convertHeaders(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestRunner_ConvertAssertions(t *testing.T) {
	typeAssertions := []types.Assertion{
		{Field: "status", Op: "equals", Value: "200"},
		{Field: "body.name", Op: "contains", Value: "Alice"},
	}

	result := convertAssertions(typeAssertions)
	if len(result) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(result))
	}
	if result[0].Field != "status" || result[0].Op != "equals" || result[0].Value != "200" {
		t.Errorf("unexpected assertion: %+v", result[0])
	}
}

func TestRunner_ConvertAssertions_Nil(t *testing.T) {
	result := convertAssertions(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestRunner_RunWithDataFile(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "test_data.json")
	if err := os.WriteFile(jsonPath, []byte(`[{"name":"Alice"},{"name":"Bob"}]`), 0644); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "data-test",
		URL:        ts.URL + "/{{name}}",
		Method:     "GET",
		Collection: "data-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "data-col",
		DataFile:       jsonPath,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results (2 data rows), got %d", len(results))
	}
}

func TestRunner_RunWithDataFile_CSV(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test_data.csv")
	if err := os.WriteFile(csvPath, []byte("name,value\nAlice,100\nBob,200"), 0644); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "csv-test",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "csv-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "csv-col",
		DataFile:       csvPath,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestRunner_RunRequest_VariableSubstitutionError(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "var-error",
		URL:        ts.URL + "/{{nonexistent}}",
		Method:     "GET",
		Collection: "var-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "var-col",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Failed != 1 {
		t.Errorf("expected 1 failed request, got %d", results[0].Failed)
	}
}

func TestRunner_RunRequest_Timeout(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "timeout-test",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "timeout-col",
		Timeout:    "50ms",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "timeout-col",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Failed != 1 {
		t.Errorf("expected 1 failed request (timeout), got %d", results[0].Failed)
	}
}

func TestRunner_RunWithAssertions(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"name": "Alice", "age": "30"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "assertion-test",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "assert-col",
		Assertions: []types.Assertion{
			{Field: "status_code", Op: "equals", Value: "200"},
		},
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "assert-col",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Passed != 1 {
		t.Errorf("expected passed=1, got %d", results[0].Passed)
	}
}

func TestRunner_ExtractedVarsChainIntoNextRequest(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	var paymentBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/orders":
			_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
		case "/payments":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read payment body: %v", err)
			}
			paymentBody = string(body)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-create-order",
		Name:       "create-order",
		URL:        ts.URL + "/orders",
		Method:     "POST",
		Collection: "checkout-flow",
		SortOrder:  1,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-initiate-payment",
		Name:       "initiate-payment",
		URL:        ts.URL + "/payments",
		Method:     "POST",
		Collection: "checkout-flow",
		SortOrder:  2,
		Headers: []types.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"orderId":"{{orderId}}"}`,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "checkout-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Failed != 0 {
		t.Fatalf("expected collection to pass, got result: %+v", results[0])
	}
	if paymentBody != `{"orderId":"ord_123"}` {
		t.Fatalf("expected second request body to receive extracted orderId, got %q", paymentBody)
	}
	assertStringMap(t, "ExtractedVars", results[0].RequestResults[0].ExtractedVars, map[string]string{"orderId": "ord_123"})
}

func TestRunner_CollectionExtractChainUsesPaymentIDStatus(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	var paymentBody string
	var statusPath string
	var statusBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/orders":
			_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
		case "/payments":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read payment body: %v", err)
			}
			paymentBody = string(body)
			_, _ = w.Write([]byte(`{"data":{"paymentId":"pay_456"}}`))
		case "/status/pay_456":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read status body: %v", err)
			}
			statusPath = r.URL.Path
			statusBody = string(body)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-create-order-chain",
		Name:       "create order",
		URL:        ts.URL + "/orders",
		Method:     "POST",
		Collection: "checkout-chain",
		SortOrder:  1,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-initiate-payment-chain",
		Name:       "initiate payment",
		URL:        ts.URL + "/payments",
		Method:     "POST",
		Collection: "checkout-chain",
		SortOrder:  2,
		Headers: []types.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"orderId":"{{orderId}}"}`,
		Extracts: []types.Extract{
			{Name: "paymentId", Source: "jsonpath:$.data.paymentId"},
		},
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-payment-status-chain",
		Name:       "payment status",
		URL:        ts.URL + "/status/{{paymentId}}",
		Method:     "POST",
		Collection: "checkout-chain",
		SortOrder:  3,
		Headers: []types.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"paymentId":"{{paymentId}}"}`,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "checkout-chain"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Failed != 0 || results[0].Passed != 3 {
		t.Fatalf("expected three-request collection to pass, got result: %+v", results[0])
	}
	if paymentBody != `{"orderId":"ord_123"}` {
		t.Fatalf("expected payment body to use extracted orderId, got %q", paymentBody)
	}
	if statusPath != "/status/pay_456" {
		t.Fatalf("expected status URL to use extracted paymentId, got %q", statusPath)
	}
	if statusBody != `{"paymentId":"pay_456"}` {
		t.Fatalf("expected status body to use extracted paymentId, got %q", statusBody)
	}
	assertStringMap(t, "payment ExtractedVars", results[0].RequestResults[1].ExtractedVars, map[string]string{"paymentId": "pay_456"})
}

func TestRunner_DataDrivenCollectionResetsRunningVarsPerRow(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	envObj := env.NewEnvironment("beta", "")
	envObj.SetVariable("token", "env-token")
	envStorage.SaveEnv(envObj)

	tmpDir := t.TempDir()
	dataPath := filepath.Join(tmpDir, "rows.csv")
	if err := os.WriteFile(dataPath, []byte("item,token\none,row-token-one\ntwo,row-token-two\n"), 0644); err != nil {
		t.Fatalf("failed to write data file: %v", err)
	}

	usedTokenPaths := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/prepare/"):
			_, _ = w.Write([]byte(`{"ok":true}`))
		case strings.HasPrefix(r.URL.Path, "/use/"):
			usedTokenPaths = append(usedTokenPaths, r.URL.Path)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-prepare-row",
		Name:       "prepare row",
		URL:        ts.URL + "/prepare/{{item}}",
		Method:     "GET",
		Collection: "data-driven-chain",
		SortOrder:  1,
		PostScript: `if (gurl.getVar("item") === "one") { gurl.setVar("token", "tok_one"); }`,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-use-token",
		Name:       "use token",
		URL:        ts.URL + "/use/{{token}}",
		Method:     "GET",
		Collection: "data-driven-chain",
		SortOrder:  2,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{
		CollectionName: "data-driven-chain",
		Environment:    "beta",
		DataFile:       dataPath,
		Vars:           map[string]string{"token": "cli-token"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected two data iterations, got %d", len(results))
	}
	if got, want := usedTokenPaths, []string{"/use/tok_one", "/use/cli-token"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected per-row running vars to reset with CLI defaults re-applied, got paths %v", got)
	}
}

func TestRunner_NextRequestOverrideExactName(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-start-next",
		Name:       "start",
		URL:        ts.URL + "/start",
		Method:     "GET",
		Collection: "next-flow",
		SortOrder:  1,
		PostScript: `gurl.setNextRequest("final")`,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-middle-next",
		Name:       "middle",
		URL:        ts.URL + "/middle",
		Method:     "GET",
		Collection: "next-flow",
		SortOrder:  2,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-final-next",
		Name:       "final",
		URL:        ts.URL + "/final",
		Method:     "GET",
		Collection: "next-flow",
		SortOrder:  3,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "next-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := order, []string{"/start", "/final"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected next-request override to skip natural middle request, got order %v", got)
	}
	if results[0].Failed != 0 || results[0].Passed != 2 {
		t.Fatalf("expected override flow to pass executed requests, got result: %+v", results[0])
	}
}

func TestRunner_NextRequestOverrideUnknownTargetFails(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-missing-next",
		Name:       "start",
		URL:        ts.URL + "/start",
		Method:     "GET",
		Collection: "unknown-next-flow",
		SortOrder:  1,
		PostScript: `gurl.setNextRequest("missing")`,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "unknown-next-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requestResult := results[0].RequestResults[0]
	if results[0].Failed != 1 || requestResult.Passed || !strings.Contains(requestResult.Error, `next request "missing" not found`) {
		t.Fatalf("expected unknown next-request target to fail current request clearly, got result: %+v", requestResult)
	}
}

func TestRunner_NextRequestOverrideLoopDetected(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-loop-a",
		Name:       "loop-a",
		URL:        ts.URL + "/loop-a",
		Method:     "GET",
		Collection: "loop-next-flow",
		SortOrder:  1,
		PostScript: `gurl.setNextRequest("loop-b")`,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-loop-b",
		Name:       "loop-b",
		URL:        ts.URL + "/loop-b",
		Method:     "GET",
		Collection: "loop-next-flow",
		SortOrder:  2,
		PostScript: `gurl.setNextRequest("loop-a")`,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "loop-next-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := order, []string{"/loop-a", "/loop-b"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("expected loop to fail before revisiting loop-a, got order %v", got)
	}
	last := results[0].RequestResults[len(results[0].RequestResults)-1]
	if results[0].Failed != 1 || !strings.Contains(last.Error, `next request loop detected`) {
		t.Fatalf("expected next-request loop to fail fast, got result: %+v", last)
	}
}

func TestRunner_ExtractedVarsOverrideExistingVars(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/first":
			_, _ = w.Write([]byte(`{"orderId":"from-response"}`))
		default:
			gotPath = r.URL.Path
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-first",
		Name:       "first",
		URL:        ts.URL + "/first",
		Method:     "GET",
		Collection: "override-flow",
		SortOrder:  1,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.orderId"},
		},
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-second",
		Name:       "second",
		URL:        ts.URL + "/orders/{{orderId}}",
		Method:     "GET",
		Collection: "override-flow",
		SortOrder:  2,
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{
		CollectionName: "override-flow",
		Vars:           map[string]string{"orderId": "from-config"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Failed != 0 {
		t.Fatalf("expected collection to pass, got result: %+v", results[0])
	}
	if gotPath != "/orders/from-response" {
		t.Fatalf("expected extracted orderId to override config var, got path %q", gotPath)
	}
}

func TestRunner_AssertionReceivesExtractedVars(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-assert-extract",
		Name:       "assert-extract",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "assert-extract-flow",
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
		Assertions: []types.Assertion{
			{Field: "extract:orderId", Op: "equals", Value: "ord_123"},
		},
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "assert-extract-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requestResult := results[0].RequestResults[0]
	if !requestResult.Passed {
		t.Fatalf("expected assertion on extracted var to pass, got request result: %+v", requestResult)
	}
	if len(requestResult.AssertionResults) != 1 || requestResult.AssertionResults[0].Actual != "ord_123" {
		t.Fatalf("expected assertion actual to be extracted orderId, got %#v", requestResult.AssertionResults)
	}
}

func TestRunner_MissingExtractionDoesNotFailRequest(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-missing-extract",
		Name:       "missing-extract",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "missing-extract-flow",
		Extracts: []types.Extract{
			{Name: "missing", Source: "jsonpath:$.missing"},
		},
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "missing-extract-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requestResult := results[0].RequestResults[0]
	if !requestResult.Passed || requestResult.Error != "" {
		t.Fatalf("expected missing extraction to remain neutral, got request result: %+v", requestResult)
	}
	assertStringMap(t, "ExtractedVars", requestResult.ExtractedVars, map[string]string{"missing": ""})
}

func TestRunner_MultipleIterations(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"count": count})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "multi-iter",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "multi-iter-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "multi-iter-col",
		Iterations:     5,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 iterations, got %d", len(results))
	}
	if count != 5 {
		t.Errorf("expected 5 total requests, got %d", count)
	}

	for i, r := range results {
		if r.Iteration != i+1 {
			t.Errorf("iteration %d: expected Iteration=%d, got %d", i+1, i+1, r.Iteration)
		}
	}
}

func TestRunner_ContextCancellation(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "slow-req",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "cancel-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "cancel-col",
		Iterations:     1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results, err := runner.Run(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Total != 1 {
		t.Errorf("expected 1 total request, got %d", results[0].Total)
	}
}

func TestRunner_ContextCancellationWithDelay(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "req-1",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "delay-cancel-col",
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "req-2",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "delay-cancel-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "delay-cancel-col",
		Iterations:     1,
		Delay:          100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results, err := runner.Run(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Total != 2 {
		t.Errorf("expected 2 total requests (collection size), got %d", results[0].Total)
	}
}

func TestRunner_DataFileNotFound(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "data-test",
		URL:        "http://localhost/test",
		Method:     "GET",
		Collection: "data-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "data-col",
		DataFile:       "/nonexistent/path/data.csv",
	}

	_, err := runner.Run(context.Background(), config)
	if err == nil {
		t.Error("expected error for nonexistent data file, got nil")
	}
}

func TestRunner_VarsOverrideEnv(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": r.URL.Path})
	}))
	defer ts.Close()

	e := env.NewEnvironment("test-env", "")
	e.SetVariable("baseUrl", "http://env-fallback")
	envStorage.SaveEnv(e)

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "var-override",
		URL:        "{{baseUrl}}/test",
		Method:     "GET",
		Collection: "override-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "override-col",
		Environment:    "test-env",
		Iterations:     1,
		Vars:           map[string]string{"baseUrl": ts.URL},
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Passed != 1 {
		t.Errorf("expected request to pass with var override")
	}
}

func TestRunner_EnvNotFound(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "env-test",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "env-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "env-col",
		Environment:    "nonexistent-env",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Passed != 1 {
		t.Errorf("expected request to pass even with nonexistent env")
	}
}

func TestRunner_IterationsZero(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "zero-iter",
		URL:        ts.URL,
		Method:     "GET",
		Collection: "zero-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "zero-col",
		Iterations:     0,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 iteration (default), got %d", len(results))
	}
}

func TestRunner_BailOnPassedSkipsRemaining(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		if r.URL.Path == "/skip" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "first",
		URL:        ts.URL + "/first",
		Method:     "GET",
		Collection: "bail-pass-col",
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "skip",
		URL:        ts.URL + "/skip",
		Method:     "GET",
		Collection: "bail-pass-col",
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-3",
		Name:       "third",
		URL:        ts.URL + "/third",
		Method:     "GET",
		Collection: "bail-pass-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "bail-pass-col",
		Iterations:     1,
		Bail:           true,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 2 {
		t.Errorf("expected only 2 requests to run (bail after skip), got %d: %v", len(order), order)
	}

	result := results[0]
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped (after bail), got %d", result.Skipped)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
}

func TestRunner_RequestExecutionError(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "bad-url",
		URL:        "http://localhost:99999/no-connection",
		Method:     "GET",
		Collection: "error-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "error-col",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Failed != 1 {
		t.Errorf("expected 1 failed request, got %d", results[0].Failed)
	}
	if results[0].RequestResults[0].Error == "" {
		t.Errorf("expected error message for failed request")
	}
}

func TestRunner_ConfigVarsMerged(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "vars-test",
		URL:        ts.URL + "/{{path}}",
		Method:     "GET",
		Collection: "vars-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "vars-col",
		Iterations:     1,
		Vars:           map[string]string{"path": "/test"},
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Passed != 1 {
		t.Errorf("expected request to pass")
	}
}

func TestRunner_DataFileWithDelay(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "delay_data.json")
	if err := os.WriteFile(jsonPath, []byte(`[{"name":"Alice"},{"name":"Bob"}]`), 0644); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "data-delay-test",
		URL:        ts.URL + "/{{name}}",
		Method:     "GET",
		Collection: "data-delay-col",
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "data-delay-col",
		DataFile:       jsonPath,
		Delay:          20 * time.Millisecond,
	}

	start := time.Now()
	results, err := runner.Run(context.Background(), config)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if elapsed < 20*time.Millisecond {
		t.Errorf("expected delay between data iterations, elapsed=%v", elapsed)
	}
}
