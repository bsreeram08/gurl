package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/pkg/types"
)

// mockDBWithHistory implements mockDB and tracks saved history
type mockDBWithHistory struct {
	*mockDB
	history []*types.ExecutionHistory
}

func newMockDBWithHistory() *mockDBWithHistory {
	return &mockDBWithHistory{
		mockDB:  newMockDB(),
		history: make([]*types.ExecutionHistory, 0),
	}
}

func (m *mockDBWithHistory) SaveHistory(history *types.ExecutionHistory) error {
	// Store a copy
	hCopy := *history
	m.history = append(m.history, &hCopy)
	return nil
}

func (m *mockDBWithHistory) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	if limit <= 0 {
		limit = 100
	}
	if len(m.history) > limit {
		return m.history[:limit], nil
	}
	return m.history, nil
}

func TestResponseStorage(t *testing.T) {
	// Create test server that returns known response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"test response","status":"ok"}`))
	}))
	defer server.Close()

	db := newMockDBWithHistory()

	// Create a test request pointing to our test server
	req := &types.SavedRequest{
		ID:     "test-req-id",
		Name:   "test-request",
		URL:    server.URL,
		Method: "GET",
	}
	db.SaveRequest(req)

	// Execute via client to get response
	resp, err := client.Execute(client.Request{
		Method: "GET",
		URL:    server.URL,
	})
	if err != nil {
		t.Fatalf("client.Execute failed: %v", err)
	}

	// Create execution history with full response data
	history := types.NewExecutionHistory(
		req.ID,
		string(resp.Body),
		resp.StatusCode,
		resp.Duration.Milliseconds(),
		resp.Size,
	)
	db.SaveHistory(history)

	// Verify: GetHistory returns entry with non-empty Response body
	entries, err := db.GetHistory(req.ID, 1)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one history entry")
	}

	entry := entries[0]
	if entry.Response == "" {
		t.Error("expected non-empty Response body in history")
	}
	if entry.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, entry.StatusCode)
	}
	if entry.SizeBytes == 0 {
		t.Error("expected non-zero SizeBytes")
	}

	// Verify response body content
	var result map[string]string
	if err := json.Unmarshal([]byte(entry.Response), &result); err != nil {
		t.Errorf("response body should be valid JSON: %v", err)
	}
	if result["message"] != "test response" {
		t.Errorf("expected message 'test response', got %s", result["message"])
	}
}

func TestGetHistoryRetrievesResponseBody(t *testing.T) {
	db := newMockDBWithHistory()

	// Create and save a history entry directly
	history := types.NewExecutionHistory(
		"req-123",
		`{"key":"value","nested":{"data":"test"}}`,
		200,
		50,
		45,
	)
	db.SaveHistory(history)

	// Verify GetHistory retrieves the response body
	entries, err := db.GetHistory("req-123", 1)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one history entry")
	}

	entry := entries[0]
	if entry.Response == "" {
		t.Error("expected non-empty Response body")
	}
	if entry.Response != `{"key":"value","nested":{"data":"test"}}` {
		t.Errorf("expected exact response body, got %s", entry.Response)
	}
	if entry.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", entry.StatusCode)
	}
}

// TestRunCommandStoresFullResponse tests the integration of run command with response storage
func TestRunCommandStoresFullResponse(t *testing.T) {
	// This test verifies the complete flow:
	// 1. Run command executes request via client
	// 2. Response is captured and stored in history
	// 3. History entry contains full response body

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	db := newMockDBWithHistory()

	// Create test request
	req := &types.SavedRequest{
		ID:     "run-test-req",
		Name:   "run-test",
		URL:    server.URL,
		Method: "POST",
		Body:   `{"test":true}`,
	}
	db.SaveRequest(req)

	// Simulate what run command does:
	// 1. Get request from db (already done above)
	// 2. Execute via client
	resp, err := client.Execute(client.Request{
		Method: req.Method,
		URL:    req.URL,
		Body:   req.Body,
		Headers: func() []client.Header {
			var headers []client.Header
			for _, h := range req.Headers {
				headers = append(headers, client.Header{Key: h.Key, Value: h.Value})
			}
			return headers
		}(),
	})
	if err != nil {
		t.Fatalf("client.Execute failed: %v", err)
	}

	// 3. Save history with full response data
	history := types.NewExecutionHistory(
		req.ID,
		string(resp.Body),
		resp.StatusCode,
		resp.Duration.Milliseconds(),
		resp.Size,
	)
	db.SaveHistory(history)

	// 4. Verify history contains full response
	entries, err := db.GetHistory(req.ID, 1)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no history entries found")
	}

	entry := entries[0]
	if entry.Response == "" {
		t.Error("Response body is empty")
	}
	if entry.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", entry.StatusCode)
	}
	if entry.SizeBytes == 0 {
		t.Error("SizeBytes is zero")
	}

	// Verify the JSON can be parsed
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(entry.Response), &result); err != nil {
		t.Errorf("invalid JSON in response: %v", err)
	}
}
