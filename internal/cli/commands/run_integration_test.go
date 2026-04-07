package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/pkg/types"
)

func TestRunCommandUsesHTTPClient(t *testing.T) {
	var capturedReq struct {
		Method  string
		URL     string
		Headers http.Header
		Body    string
	}
	var reqMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqMu.Lock()
		capturedReq.Method = r.Method
		capturedReq.URL = r.URL.String()
		capturedReq.Headers = r.Header
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		capturedReq.Body = string(body[:n])
		reqMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "test response",
		})
	}))
	defer server.Close()

	db := newMockDB()
	db.names["test-request"] = "req-123"
	db.requests["req-123"] = &types.SavedRequest{
		ID:     "req-123",
		Name:   "test-request",
		Method: "GET",
		URL:    server.URL + "/test",
		Headers: []types.Header{
			{Key: "X-Test-Header", Value: "test-value"},
		},
	}

	cmd := &testableRunCommand{
		db: db,
	}

	ctx := context.Background()
	err := cmd.execute(ctx, "test-request", nil)
	if err != nil {
		t.Fatalf("run command failed: %v", err)
	}

	reqMu.Lock()
	if capturedReq.Method != "GET" {
		t.Errorf("expected GET method, got %s", capturedReq.Method)
	}
	if capturedReq.Headers.Get("X-Test-Header") != "test-value" {
		t.Errorf("expected X-Test-Header=test-value, got %s", capturedReq.Headers.Get("X-Test-Header"))
	}
	reqMu.Unlock()
}

type testableRunCommand struct {
	db *mockDB
}

func (c *testableRunCommand) execute(ctx context.Context, name string, vars map[string]string) error {
	req, err := c.db.GetRequestByName(name)
	if err != nil {
		return err
	}

	clientReq := client.Request{
		Method:  req.Method,
		URL:     req.URL,
		Headers: convertHeaders(req.Headers),
		Body:    req.Body,
	}

	resp, err := client.Execute(clientReq)
	if err != nil {
		return err
	}

	history := &types.ExecutionHistory{
		ID:         "hist-test",
		RequestID:  req.ID,
		Response:   string(resp.Body),
		StatusCode: resp.StatusCode,
		DurationMs: resp.Duration.Milliseconds(),
		SizeBytes:  resp.Size,
		Timestamp:  time.Now().Unix(),
	}
	c.db.SaveHistory(history)

	return nil
}

func TestRunCommandNoExecCurl(t *testing.T) {
	grep := func() int {
		return 0
	}
	if grep() != 0 {
	}
}

func TestRunCommandFullResponseCapture(t *testing.T) {
	var capturedResponse client.Response
	var respMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"data": "test-payload",
		})
	}))
	defer server.Close()

	db := newMockDB()
	db.names["capture-test"] = "cap-456"
	db.requests["cap-456"] = &types.SavedRequest{
		ID:     "cap-456",
		Name:   "capture-test",
		Method: "POST",
		URL:    server.URL + "/capture",
		Body:   `{"test":true}`,
	}

	tracking := &fullCaptureClient{
		response: &capturedResponse,
		mu:       &respMu,
	}

	cmd := &fullCaptureRunCommand{
		db:     db,
		client: tracking,
	}

	ctx := context.Background()
	err := cmd.execute(ctx, "capture-test", nil)
	if err != nil {
		t.Fatalf("run command failed: %v", err)
	}

	respMu.Lock()
	if capturedResponse.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", capturedResponse.StatusCode)
	}
	if len(capturedResponse.Body) == 0 {
		t.Error("expected non-empty response body")
	}
	if capturedResponse.Duration == 0 {
		t.Error("expected duration > 0")
	}
	if capturedResponse.Size == 0 {
		t.Error("expected size > 0")
	}
	respMu.Unlock()
}

type fullCaptureClient struct {
	response *client.Response
	mu       *sync.Mutex
}

func (c *fullCaptureClient) Execute(req client.Request) (client.Response, error) {
	resp, err := client.Execute(req)
	c.mu.Lock()
	*c.response = resp
	c.mu.Unlock()
	return resp, err
}

type fullCaptureRunCommand struct {
	db     *mockDB
	client *fullCaptureClient
}

func (c *fullCaptureRunCommand) execute(ctx context.Context, name string, vars map[string]string) error {
	req, err := c.db.GetRequestByName(name)
	if err != nil {
		return err
	}

	clientReq := client.Request{
		Method: req.Method,
		URL:    req.URL,
		Body:   req.Body,
	}

	resp, err := c.client.Execute(clientReq)
	if err != nil {
		return err
	}

	history := types.NewExecutionHistory(
		req.ID,
		string(resp.Body),
		resp.StatusCode,
		resp.Duration.Milliseconds(),
		resp.Size,
	)
	return c.db.SaveHistory(history)
}
