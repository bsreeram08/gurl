package runner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// seqMockDB implements storage.DB for testing
type seqMockDB struct {
	requests map[string]*types.SavedRequest
	names    map[string]string
}

func newSeqMockDB() *seqMockDB {
	return &seqMockDB{
		requests: make(map[string]*types.SavedRequest),
		names:    make(map[string]string),
	}
}

func (m *seqMockDB) Open() error  { return nil }
func (m *seqMockDB) Close() error { return nil }
func (m *seqMockDB) SaveRequest(req *types.SavedRequest) error {
	if req.ID == "" {
		req.ID = "test-id-1"
	}
	m.requests[req.ID] = req
	m.names[req.Name] = req.ID
	return nil
}
func (m *seqMockDB) GetRequest(id string) (*types.SavedRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return nil, nil
	}
	return req, nil
}
func (m *seqMockDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	id, ok := m.names[name]
	if !ok {
		return nil, errors.New("request not found: " + name)
	}
	req := m.requests[id]
	copy := *req
	return &copy, nil
}
func (m *seqMockDB) ListRequests(opts *storage.ListOptions) ([]*types.SavedRequest, error) {
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
func (m *seqMockDB) DeleteRequest(id string) error {
	req, ok := m.requests[id]
	if !ok {
		return nil
	}
	delete(m.names, req.Name)
	delete(m.requests, id)
	return nil
}
func (m *seqMockDB) UpdateRequest(req *types.SavedRequest) error {
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
func (m *seqMockDB) SaveHistory(history *types.ExecutionHistory) error {
	return nil
}
func (m *seqMockDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	return nil, nil
}
func (m *seqMockDB) ListFolder(path string) ([]*types.SavedRequest, error)          { return nil, nil }
func (m *seqMockDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) { return nil, nil }
func (m *seqMockDB) DeleteFolder(path string) error                                 { return nil }
func (m *seqMockDB) GetAllFolders() ([]string, error)                               { return nil, nil }

// seqMockEnvStorage implements env.EnvStorage for testing
type seqMockEnvStorage struct{}

func newSeqMockEnvStorage() *seqMockEnvStorage {
	return &seqMockEnvStorage{}
}
func (m *seqMockEnvStorage) GetEnvByName(name string) (*env.Environment, error) {
	return nil, errors.New("not found")
}
func (m *seqMockEnvStorage) GetActiveEnv() (string, error)         { return "", nil }
func (m *seqMockEnvStorage) SaveEnv(e *env.Environment) error      { return nil }
func (m *seqMockEnvStorage) ListEnvs() ([]*env.Environment, error) { return nil, nil }
func (m *seqMockEnvStorage) DeleteEnv(name string) error           { return nil }

// sortRequests is a helper to sort requests by SortOrder then by Name
func sortRequests(reqs []*types.SavedRequest) []*types.SavedRequest {
	sorted := make([]*types.SavedRequest, len(reqs))
	copy(sorted, reqs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].SortOrder != sorted[j].SortOrder {
			return sorted[i].SortOrder < sorted[j].SortOrder
		}
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func TestSequence_Explicit(t *testing.T) {
	db := newSeqMockDB()

	// Set sort orders explicitly
	req1 := &types.SavedRequest{
		ID:         "req-1",
		Name:       "login",
		URL:        "http://localhost/1",
		Method:     "GET",
		Collection: "test-col",
		SortOrder:  2,
	}
	req2 := &types.SavedRequest{
		ID:         "req-2",
		Name:       "get-data",
		URL:        "http://localhost/2",
		Method:     "GET",
		Collection: "test-col",
		SortOrder:  1,
	}

	db.SaveRequest(req1)
	db.SaveRequest(req2)

	// Verify sort orders were set
	got1, _ := db.GetRequest("req-1")
	got2, _ := db.GetRequest("req-2")

	if got1.SortOrder != 2 {
		t.Errorf("expected req-1 SortOrder=2, got %d", got1.SortOrder)
	}
	if got2.SortOrder != 1 {
		t.Errorf("expected req-2 SortOrder=1, got %d", got2.SortOrder)
	}

	// Verify sorted order
	sorted := sortRequests([]*types.SavedRequest{got1, got2})
	if len(sorted) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(sorted))
	}

	// First should be get-data (SortOrder=1), second should be login (SortOrder=2)
	if sorted[0].Name != "get-data" {
		t.Errorf("expected first request to be 'get-data', got '%s'", sorted[0].Name)
	}
	if sorted[1].Name != "login" {
		t.Errorf("expected second request to be 'login', got '%s'", sorted[1].Name)
	}
}

func TestSequence_InCollection(t *testing.T) {
	db := newSeqMockDB()
	envStorage := newSeqMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-a",
		Name:       "request-a",
		URL:        ts.URL + "/first",
		Method:     "GET",
		Collection: "ordered-col",
		SortOrder:  1,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-b",
		Name:       "request-b",
		URL:        ts.URL + "/second",
		Method:     "GET",
		Collection: "ordered-col",
		SortOrder:  2,
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

	// Should run in sequence order: request-a (1) before request-b (2)
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

func TestSequence_Unordered(t *testing.T) {
	db := newSeqMockDB()
	envStorage := newSeqMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	// All SortOrder=0 (unordered) - should run alphabetically by name
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-z",
		Name:       "zebra",
		URL:        ts.URL + "/zebra",
		Method:     "GET",
		Collection: "alpha-col",
		SortOrder:  0,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-a",
		Name:       "alpha",
		URL:        ts.URL + "/alpha",
		Method:     "GET",
		Collection: "alpha-col",
		SortOrder:  0,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-m",
		Name:       "mango",
		URL:        ts.URL + "/mango",
		Method:     "GET",
		Collection: "alpha-col",
		SortOrder:  0,
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "alpha-col",
		Iterations:     1,
	}

	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(order))
	}

	// Should run alphabetically: alpha, mango, zebra
	if order[0] != "/alpha" {
		t.Errorf("expected first request to /alpha, got %s", order[0])
	}
	if order[1] != "/mango" {
		t.Errorf("expected second request to /mango, got %s", order[1])
	}
	if order[2] != "/zebra" {
		t.Errorf("expected third request to /zebra, got %s", order[2])
	}

	if results[0].Passed != 3 {
		t.Errorf("expected 3 passed, got %d", results[0].Passed)
	}
}

func TestSequence_Reorder(t *testing.T) {
	db := newSeqMockDB()
	envStorage := newSeqMockEnvStorage()

	order := []string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "first",
		URL:        ts.URL + "/first",
		Method:     "GET",
		Collection: "reorder-col",
		SortOrder:  1,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "second",
		URL:        ts.URL + "/second",
		Method:     "GET",
		Collection: "reorder-col",
		SortOrder:  2,
	})

	runner := NewRunner(db, envStorage)
	config := RunConfig{
		CollectionName: "reorder-col",
		Iterations:     1,
	}

	// First run should be 1 then 2
	results, err := runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(order))
	}

	if order[0] != "/first" {
		t.Errorf("expected first to /first, got %s", order[0])
	}
	if order[1] != "/second" {
		t.Errorf("expected second to /second, got %s", order[1])
	}

	// Now reorder: second gets 1, first gets 2
	req1, _ := db.GetRequest("req-1")
	req1.SortOrder = 2
	db.UpdateRequest(req1)

	req2, _ := db.GetRequest("req-2")
	req2.SortOrder = 1
	db.UpdateRequest(req2)

	order = order[:0] // reset
	_, err = runner.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("unexpected error after reorder: %v", err)
	}

	// Should now run in new order: second then first
	if order[0] != "/second" {
		t.Errorf("expected first to /second after reorder, got %s", order[0])
	}
	if order[1] != "/first" {
		t.Errorf("expected second to /first after reorder, got %s", order[1])
	}

	_ = results
}

func TestSequence_Gaps(t *testing.T) {
	db := newSeqMockDB()

	// Set sort orders with gaps: 1, 5, 10
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "first",
		URL:        "http://localhost/1",
		Method:     "GET",
		Collection: "gaps-col",
		SortOrder:  1,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "second",
		URL:        "http://localhost/2",
		Method:     "GET",
		Collection: "gaps-col",
		SortOrder:  5,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-3",
		Name:       "third",
		URL:        "http://localhost/3",
		Method:     "GET",
		Collection: "gaps-col",
		SortOrder:  10,
	})

	seq, err := GetSequence(db, "gaps-col")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seq) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(seq))
	}

	// Should be sorted by SortOrder: 1, 5, 10
	if seq[0].Name != "first" {
		t.Errorf("expected first to be 'first' (order 1), got '%s'", seq[0].Name)
	}
	if seq[1].Name != "second" {
		t.Errorf("expected second to be 'second' (order 5), got '%s'", seq[1].Name)
	}
	if seq[2].Name != "third" {
		t.Errorf("expected third to be 'third' (order 10), got '%s'", seq[2].Name)
	}
}

func TestSequence_Display(t *testing.T) {
	db := newSeqMockDB()

	db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "login",
		URL:        "http://localhost/login",
		Method:     "POST",
		Collection: "my-col",
		SortOrder:  1,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-2",
		Name:       "get-data",
		URL:        "http://localhost/data",
		Method:     "GET",
		Collection: "my-col",
		SortOrder:  2,
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "req-3",
		Name:       "logout",
		URL:        "http://localhost/logout",
		Method:     "POST",
		Collection: "my-col",
		SortOrder:  3,
	})

	seq, err := GetSequence(db, "my-col")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seq) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(seq))
	}

	// Verify display order
	expected := []string{"login", "get-data", "logout"}
	for i, name := range expected {
		if seq[i].Name != name {
			t.Errorf("position %d: expected '%s', got '%s'", i, name, seq[i].Name)
		}
	}
}
