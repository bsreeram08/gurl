package commands

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage"
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

func TestRunCommand_SingleSavedRequestLifecycle(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders" {
			t.Fatalf("expected /orders path, got %q", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord_789"}}`))
	}))
	defer server.Close()

	db := newMockDB()
	db.requests["single-lifecycle"] = &types.SavedRequest{
		ID:        "single-lifecycle",
		Name:      "create order",
		Method:    "POST",
		URL:       "{{baseUrl}}/orders",
		Body:      `{"customerId":"{{customerId}}"}`,
		PreScript: `gurl.setVar("customerId", "cust_123")`,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
		Assertions: []types.Assertion{
			{Field: "extract:orderId", Op: "equals", Value: "ord_789"},
		},
	}
	db.names["create order"] = "single-lifecycle"

	envDB := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "env.db"))
	if err := envDB.Open(); err != nil {
		t.Fatalf("failed to open env db: %v", err)
	}
	defer envDB.Close()
	envStorage := env.NewEnvStorage(envDB)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := RunCommand(db, envStorage)
	if err := cmd.Run(context.Background(), []string{"run", "create order", "--env", "beta"}); err != nil {
		t.Fatalf("run command failed: %v", err)
	}

	if gotBody != `{"customerId":"cust_123"}` {
		t.Fatalf("expected pre-script variable to feed request body, got %q", gotBody)
	}
	if len(db.history) != 1 {
		t.Fatalf("expected single run to preserve history save, got %d entries", len(db.history))
	}
}

func TestRunCommand_RequestChainingPRDFlowPersistsExtractedIDs(t *testing.T) {
	type observedRequest struct {
		Path   string
		Header string
		Body   string
	}

	var mu sync.Mutex
	observed := make([]observedRequest, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		mu.Lock()
		observed = append(observed, observedRequest{
			Path:   r.URL.Path,
			Header: r.Header.Get("X-Flow-ID"),
			Body:   string(body),
		})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/orders":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
		case "/payments":
			_, _ = w.Write([]byte(`{"data":{"paymentId":"pay_456"}}`))
		case "/payments/pay_456/status":
			_, _ = w.Write([]byte(`{"status":"paid"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	db := newMockDB()
	db.requests["req-create-order"] = &types.SavedRequest{
		ID:         "req-create-order",
		Name:       "create order",
		Method:     "POST",
		URL:        "{{baseUrl}}/orders",
		Collection: "checkout-prd",
		SortOrder:  1,
		Headers: []types.Header{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "X-Flow-ID", Value: "{{orderId}}"},
		},
		Body: `{"customerId":"{{customerId}}"}`,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	}
	db.names["create order"] = "req-create-order"
	db.requests["req-initiate-payment"] = &types.SavedRequest{
		ID:         "req-initiate-payment",
		Name:       "initiate payment",
		Method:     "POST",
		URL:        "{{baseUrl}}/payments",
		Collection: "checkout-prd",
		SortOrder:  2,
		Headers: []types.Header{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "X-Flow-ID", Value: "{{orderId}}"},
		},
		Body: `{"orderId":"{{orderId}}"}`,
		Extracts: []types.Extract{
			{Name: "paymentId", Source: "jsonpath:$.data.paymentId"},
		},
	}
	db.names["initiate payment"] = "req-initiate-payment"
	db.requests["req-payment-status"] = &types.SavedRequest{
		ID:         "req-payment-status",
		Name:       "check payment status",
		Method:     "GET",
		URL:        "{{baseUrl}}/payments/{{paymentId}}/status",
		Collection: "checkout-prd",
		SortOrder:  3,
		Headers: []types.Header{
			{Key: "X-Flow-ID", Value: "{{paymentId}}"},
		},
	}
	db.names["check payment status"] = "req-payment-status"

	envStorage := newRunTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	beta.SetVariable("manualOnly", "keep")
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := CollectionCommand(db, envStorage)
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"collection", "run", "checkout-prd", "--env", "beta", "--var", "customerId=cust_999", "--var", "orderId=ord_cli_seed", "--persist"}); err != nil {
			t.Fatalf("collection run failed: %v", err)
		}
	})

	mu.Lock()
	got := append([]observedRequest(nil), observed...)
	mu.Unlock()
	if len(got) != 3 {
		t.Fatalf("expected three HTTP requests, got %d: %#v", len(got), got)
	}
	expected := []observedRequest{
		{Path: "/orders", Header: "ord_cli_seed", Body: `{"customerId":"cust_999"}`},
		{Path: "/payments", Header: "ord_123", Body: `{"orderId":"ord_123"}`},
		{Path: "/payments/pay_456/status", Header: "pay_456", Body: ""},
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("request %d = %#v, want %#v", i+1, got[i], expected[i])
		}
	}

	reloaded, err := envStorage.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("failed to reload env: %v", err)
	}
	if reloaded.Variables["orderId"] != "ord_123" || reloaded.Variables["paymentId"] != "pay_456" {
		t.Fatalf("expected extracted orderId/paymentId to persist, got %+v", reloaded.Variables)
	}
	for _, unexpected := range []string{"customerId"} {
		if _, ok := reloaded.Variables[unexpected]; ok {
			t.Fatalf("expected %s not to persist, got %+v", unexpected, reloaded.Variables)
		}
	}
	if reloaded.Variables["manualOnly"] != "keep" || reloaded.Variables["baseUrl"] != server.URL {
		t.Fatalf("expected existing env vars to remain unchanged, got %+v", reloaded.Variables)
	}
	if !strings.Contains(output, "Persisted 2 variables to environment \"beta\"") || !strings.Contains(output, "orderId = ord_123") || !strings.Contains(output, "paymentId = pay_456") {
		t.Fatalf("expected persist summary for extracted ids, got output:\n%s", output)
	}
}

func TestRunCommand_FlowControlUnresolvedTemplateFailsBeforeHTTP(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_, _ = w.Write([]byte(`{"unexpected":true}`))
	}))
	defer server.Close()

	db := newMockDB()
	db.requests["req-unresolved-status"] = &types.SavedRequest{
		ID:     "req-unresolved-status",
		Name:   "check payment status unresolved",
		Method: "GET",
		URL:    server.URL + "/payments/{{paymentId}}/status",
	}
	db.names["check payment status unresolved"] = "req-unresolved-status"

	cmd := RunCommand(db, newRunTestEnvStorage(t))
	err := cmd.Run(context.Background(), []string{"run", "check payment status unresolved"})
	if err == nil {
		t.Fatal("expected unresolved template error")
	}
	if !strings.Contains(err.Error(), "paymentId") || !strings.Contains(err.Error(), "variable substitution failed") {
		t.Fatalf("expected clear missing paymentId error, got %v", err)
	}
	if requestCount != 0 {
		t.Fatalf("expected unresolved template to fail before HTTP, got %d requests", requestCount)
	}
}

func TestRunCommand_DryRunWarnsForUnresolvedTemplatesAtCLIIntegration(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_, _ = w.Write([]byte(`{"unexpected":true}`))
	}))
	defer server.Close()

	db := newMockDB()
	db.requests["req-dry-create"] = &types.SavedRequest{
		ID:         "req-dry-create",
		Name:       "create order dry",
		Method:     "POST",
		URL:        "{{baseUrl}}/orders/{{tenant}}",
		Collection: "checkout-dry",
		SortOrder:  1,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	}
	db.names["create order dry"] = "req-dry-create"
	db.requests["req-dry-pay"] = &types.SavedRequest{
		ID:         "req-dry-pay",
		Name:       "initiate payment dry",
		Method:     "POST",
		URL:        "{{baseUrl}}/payments/{{orderId}}/{{missingVar}}",
		Collection: "checkout-dry",
		SortOrder:  2,
	}
	db.names["initiate payment dry"] = "req-dry-pay"

	envStorage := newRunTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	beta.SetVariable("tenant", "acme")
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := CollectionCommand(db, envStorage)
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"collection", "run", "checkout-dry", "--env", "beta", "--dry-run"}); err != nil {
			t.Fatalf("collection dry-run failed: %v", err)
		}
	})

	if requestCount != 0 {
		t.Fatalf("expected dry-run to make zero HTTP requests, got %d", requestCount)
	}
	for _, want := range []string{
		`Dry run: collection "checkout-dry"`,
		`POST ` + server.URL + `/orders/acme`,
		`POST ` + server.URL + `/payments/{{orderId}}/{{missingVar}}`,
		`orderId from step 1 extraction`,
		`warning: unresolved {{missingVar}}`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected dry-run output to contain %q, got:\n%s", want, output)
		}
	}
	if strings.Contains(output, "warning: unresolved {{orderId}}") {
		t.Fatalf("expected planned extraction orderId not to warn as unresolved, got:\n%s", output)
	}
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
