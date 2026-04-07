package commands

import (
	"errors"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// mockDB implements storage.DB for testing
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

func (m *mockDB) Open() error         { return nil }
func (m *mockDB) Close() error       { return nil }
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
	// Return a copy to avoid test asserting on mutated data
	req := m.requests[id]
	copy := *req // shallow copy - sufficient for test purposes
	return &copy, nil
}
func (m *mockDB) ListRequests(opts *storage.ListOptions) ([]*types.SavedRequest, error) {
	var result []*types.SavedRequest
	for _, req := range m.requests {
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
	// Capture old name before modifying
	oldName := ""
	if oldReq, ok := m.requests[req.ID]; ok {
		oldName = oldReq.Name
	}
	// If name changed, remove old name from index
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
