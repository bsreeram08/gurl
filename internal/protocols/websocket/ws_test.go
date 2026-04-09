package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// testServer is a simple WebSocket echo server for testing
type testServer struct {
	mu       sync.Mutex
	messages []string
	server   *httptest.Server
	upgrader websocket.Upgrader
}

func newTestServer() *testServer {
	ts := &testServer{
		upgrader: websocket.Upgrader{},
	}
	ts.server = httptest.NewServer(http.HandlerFunc(ts.handle))
	return ts
}

func (ts *testServer) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := ts.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		ts.mu.Lock()
		ts.messages = append(ts.messages, string(data))
		ts.mu.Unlock()

		// Echo the message back
		if err := conn.WriteMessage(msgType, data); err != nil {
			return
		}
	}
}

func (ts *testServer) URL() string {
	return "ws" + ts.server.URL[4:] // http:// -> ws://
}

func (ts *testServer) Close() {
	ts.server.Close()
}

func (ts *testServer) Messages() []string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.messages
}

func TestWS_Connect(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx := context.Background()

	err := client.Connect(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Fatal("expected client to be connected")
	}
}

func TestWS_ConnectTLS(t *testing.T) {
	// Test that we can create a client with TLS config
	client := NewClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// We can't easily test actual TLS without a wss:// server
	// but we verify the client is created correctly
}

func TestWS_SendText(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx := context.Background()

	err := client.Connect(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Send a text message
	testMsg := "Hello, WebSocket!"
	err = client.SendText(testMsg)
	if err != nil {
		t.Fatalf("failed to send text: %v", err)
	}

	// Receive the echo
	respData, msgType, err := client.Receive()
	if err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	if msgType != MessageTypeText {
		t.Fatalf("expected text message, got %v", msgType)
	}

	if string(respData) != testMsg {
		t.Fatalf("expected %q, got %q", testMsg, string(respData))
	}
}

func TestWS_SendJSON(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx := context.Background()

	err := client.Connect(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Send JSON
	data := map[string]interface{}{
		"message": "Hello",
		"number":  42,
	}
	err = client.SendJSON(data)
	if err != nil {
		t.Fatalf("failed to send JSON: %v", err)
	}

	// Receive the echo
	respData, msgType, err := client.Receive()
	if err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	if msgType != MessageTypeText {
		t.Fatalf("expected text message, got %v", msgType)
	}

	// Parse the response as JSON
	var resp map[string]interface{}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp["message"] != "Hello" || resp["number"] != float64(42) {
		t.Fatalf("unexpected JSON response: %v", resp)
	}
}

func TestWS_ReceiveMultiple(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := client.Connect(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Send multiple messages
	for i := 0; i < 3; i++ {
		msg := strings.Repeat("x", i+1)
		if err := client.SendText(msg); err != nil {
			t.Fatalf("failed to send text: %v", err)
		}
	}

	// Receive multiple via channel
	msgChan, err := client.ReceiveMultiple(ctx)
	if err != nil {
		t.Fatalf("failed to start receive multiple: %v", err)
	}

	count := 0
	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed — verify we got all 3 messages
				if count < 3 {
					t.Fatalf("expected 3 messages, got %d", count)
				}
				return
			}
			if msg.Type != MessageTypeText {
				t.Fatalf("expected text message, got %v", msg.Type)
			}
			count++
			if count == 3 {
				// All 3 messages received; close the connection to stop ReceiveMultiple.
				// CloseNormalClosure causes IsUnexpectedCloseError to return false,
				// so the goroutine exits and closes msgChan.
				client.Close()
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out waiting for messages, got %d of 3", count)
		}
	}
}

func TestWS_Close(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx := context.Background()

	err := client.Connect(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Close should succeed
	err = client.Close()
	if err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	// Client should no longer be connected
	if client.IsConnected() {
		t.Fatal("expected client to be disconnected after close")
	}
}

func TestWS_Reconnect(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx := context.Background()

	// First connection
	err := client.Connect(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Send a message to verify connection works
	if err := client.SendText("test"); err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	// Close and reconnect
	client.Close()
	if client.IsConnected() {
		t.Fatal("expected client to be disconnected")
	}

	// Reconnect
	err = client.Reconnect(ctx)
	if err != nil {
		t.Fatalf("failed to reconnect: %v", err)
	}

	if !client.IsConnected() {
		t.Fatal("expected client to be connected after reconnect")
	}
	defer client.Close()
}

func TestWS_Headers(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx := context.Background()

	// Set custom headers
	headers := http.Header{
		"Authorization":   []string{"Bearer test-token"},
		"X-Custom-Header": []string{"custom-value"},
	}

	err := client.Connect(ctx, server.URL(), headers)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Note: Our test server doesn't validate headers
	// but the connection should succeed with headers set
	if !client.IsConnected() {
		t.Fatal("expected client to be connected")
	}
}

func TestWS_Ping(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx := context.Background()

	err := client.Connect(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Send a ping - this should not error
	err = client.Ping()
	if err != nil {
		t.Fatalf("failed to ping: %v", err)
	}
}

func TestWS_BinaryMessage(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	client := NewClient()
	ctx := context.Background()

	err := client.Connect(ctx, server.URL(), nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	// Send binary data
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	err = client.SendBinary(binaryData)
	if err != nil {
		t.Fatalf("failed to send binary: %v", err)
	}

	// Receive the echo
	respData, msgType, err := client.Receive()
	if err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	if msgType != MessageTypeBinary {
		t.Fatalf("expected binary message, got %v", msgType)
	}

	if string(respData) != string(binaryData) {
		t.Fatalf("expected %v, got %v", binaryData, respData)
	}
}

func TestWS_ClientNotConnected(t *testing.T) {
	client := NewClient()

	// Operations on unconnected client should fail
	_, _, err := client.Receive()
	if err == nil {
		t.Fatal("expected error on unconnected client")
	}

	err = client.SendText("test")
	if err == nil {
		t.Fatal("expected error on unconnected client")
	}

	err = client.Close()
	// Closing an unconnected client is a no-op, not an error
	_ = err
}

func TestWS_MessageTypeConstants(t *testing.T) {
	// Verify message type constants are correct
	if MessageTypeText != websocket.TextMessage {
		t.Errorf("MessageTypeText = %v, want %v", MessageTypeText, websocket.TextMessage)
	}
	if MessageTypeBinary != websocket.BinaryMessage {
		t.Errorf("MessageTypeBinary = %v, want %v", MessageTypeBinary, websocket.BinaryMessage)
	}
	if MessageTypeClose != websocket.CloseMessage {
		t.Errorf("MessageTypeClose = %v, want %v", MessageTypeClose, websocket.CloseMessage)
	}
	if MessageTypePing != websocket.PingMessage {
		t.Errorf("MessageTypePing = %v, want %v", MessageTypePing, websocket.PingMessage)
	}
	if MessageTypePong != websocket.PongMessage {
		t.Errorf("MessageTypePong = %v, want %v", MessageTypePong, websocket.PongMessage)
	}
}

func TestWS_ReconnectConfig(t *testing.T) {
	client := NewClient()

	// Verify default reconnect is disabled
	client.mu.RLock()
	defaultEnabled := client.reconnect.Enabled
	client.mu.RUnlock()

	if defaultEnabled != false {
		t.Error("expected reconnect to be disabled by default")
	}

	// Enable reconnect with custom settings
	client.SetReconnect(true, 5, 2*time.Second)

	client.mu.RLock()
	reconnect := client.reconnect
	client.mu.RUnlock()

	if !reconnect.Enabled {
		t.Error("expected reconnect to be enabled")
	}
	if reconnect.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", reconnect.MaxRetries)
	}
	if reconnect.Backoff != 2*time.Second {
		t.Errorf("Backoff = %v, want 2s", reconnect.Backoff)
	}
}

func TestWS_SetHeaders(t *testing.T) {
	client := NewClient()

	headers := http.Header{
		"Authorization": []string{"Bearer token"},
	}
	client.SetHeaders(headers)

	client.mu.RLock()
	setHeaders := client.headers
	client.mu.RUnlock()

	if setHeaders.Get("Authorization") != "Bearer token" {
		t.Error("expected authorization header to be set")
	}
}

func TestWS_InteractiveConfig(t *testing.T) {
	cfg := InteractiveConfig{
		PrettyPrint: true,
		Color:       true,
		ShowTime:    true,
	}

	runner := NewInteractiveRunner(nil, cfg)

	if runner.config.PrettyPrint != true {
		t.Error("expected PrettyPrint to be true")
	}
	if runner.config.Color != true {
		t.Error("expected Color to be true")
	}
	if runner.config.ShowTime != true {
		t.Error("expected ShowTime to be true")
	}
}
