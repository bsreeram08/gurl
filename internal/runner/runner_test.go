package runner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

type mockDB struct {
	requests map[string]*types.SavedRequest
	names    map[string]string
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
