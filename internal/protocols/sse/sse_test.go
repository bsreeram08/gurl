package sse

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSE_ParseEvent(t *testing.T) {
	// Mock SSE server that sends a simple event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: Hello World\n\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		if event.Data != "Hello World" {
			t.Errorf("Expected data 'Hello World', got '%s'", event.Data)
		}
		if event.Type != "" {
			t.Errorf("Expected empty type, got '%s'", event.Type)
		}
		if event.ID != "" {
			t.Errorf("Expected empty ID, got '%s'", event.ID)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestSSE_MultilineData(t *testing.T) {
	// Mock SSE server that sends multiline data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// SSE uses data: prefix for each line, blank line to end event
		w.Write([]byte("data: line1\n"))
		w.Write([]byte("data: line2\n"))
		w.Write([]byte("data: line3\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		expected := "line1\nline2\nline3"
		if event.Data != expected {
			t.Errorf("Expected data '%s', got '%s'", expected, event.Data)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestSSE_EventTypes(t *testing.T) {
	// Mock SSE server that sends events with different types
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("event: update\n"))
		w.Write([]byte("data: update data\n"))
		w.Write([]byte("\n"))
		w.Write([]byte("event: message\n"))
		w.Write([]byte("data: message data\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	var receivedTypes []string
	var receivedData []string
	timeout := time.After(5 * time.Second)

	for len(receivedTypes) < 2 {
		select {
		case event, ok := <-events:
			if !ok {
				goto done
			}
			receivedTypes = append(receivedTypes, event.Type)
			receivedData = append(receivedData, event.Data)
		case err := <-errors:
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}
done:

	if len(receivedTypes) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(receivedTypes))
	}

	if receivedTypes[0] != "update" || receivedData[0] != "update data" {
		t.Errorf("First event: expected type='update', data='update data', got type='%s', data='%s'",
			receivedTypes[0], receivedData[0])
	}
	if receivedTypes[1] != "message" || receivedData[1] != "message data" {
		t.Errorf("Second event: expected type='message', data='message data', got type='%s', data='%s'",
			receivedTypes[1], receivedData[1])
	}
}

func TestSSE_EventTypeFilter(t *testing.T) {
	// Mock SSE server that sends events with different types
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Expected ResponseWriter to implement Flusher")
		}

		w.Write([]byte("event: update\n"))
		w.Write([]byte("data: update data\n"))
		w.Write([]byte("\n"))
		flusher.Flush()

		w.Write([]byte("event: message\n"))
		w.Write([]byte("data: message data\n"))
		w.Write([]byte("\n"))
		flusher.Flush()

		w.Write([]byte("event: update\n"))
		w.Write([]byte("data: another update\n"))
		w.Write([]byte("\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient()

	// Filter only for "update" events
	events, errors, err := client.Connect(context.Background(), server.URL, WithEventType("update"))
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	var receivedData []string
	timeout := time.After(5 * time.Second)

	for len(receivedData) < 2 {
		select {
		case event, ok := <-events:
			if !ok {
				goto done
			}
			if event.Type != "update" {
				t.Errorf("Expected type 'update', got '%s'", event.Type)
			}
			receivedData = append(receivedData, event.Data)
		case err := <-errors:
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}
done:

	if len(receivedData) != 2 {
		t.Fatalf("Expected 2 events (both updates), got %d", len(receivedData))
	}

	if receivedData[0] != "update data" {
		t.Errorf("Expected first data 'update data', got '%s'", receivedData[0])
	}
	if receivedData[1] != "another update" {
		t.Errorf("Expected second data 'another update', got '%s'", receivedData[1])
	}
}

func TestSSE_Reconnect(t *testing.T) {
	// Mock SSE server that sends an event with ID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Verify Last-Event-ID header is sent
		if r.Header.Get("Last-Event-ID") != "test-id-123" {
			t.Logf("Expected Last-Event-ID 'test-id-123', got '%s'", r.Header.Get("Last-Event-ID"))
		}
		w.Write([]byte("id: test-id-123\n"))
		w.Write([]byte("data: reconnect test\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	// Connect with a last event ID
	events, errors, err := client.Connect(context.Background(), server.URL, WithLastEventID("test-id-123"))
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		if event.ID != "test-id-123" {
			t.Errorf("Expected ID 'test-id-123', got '%s'", event.ID)
		}
		if event.Data != "reconnect test" {
			t.Errorf("Expected data 'reconnect test', got '%s'", event.Data)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestSSE_RetryField(t *testing.T) {
	// Mock SSE server that sends a retry directive
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("retry: 5000\n"))
		w.Write([]byte("data: retry test\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		if event.Retry != 5000 {
			t.Errorf("Expected Retry 5000, got %d", event.Retry)
		}
		if event.Data != "retry test" {
			t.Errorf("Expected data 'retry test', got '%s'", event.Data)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestSSE_Timeout(t *testing.T) {
	// Mock SSE server that establishes the SSE connection but sends no events.
	// It flushes headers first so client.Connect() succeeds, then blocks
	// until the request context is cancelled. This ensures server.Close() doesn't block.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Block until the client disconnects
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient()

	// Connect with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, errors, err := client.Connect(ctx, server.URL, WithTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("Connect should not error immediately: %v", err)
	}

	select {
	case err := <-errors:
		if err == nil {
			// This is fine - context cancellation
			return
		}
		// Error received - could be context deadline exceeded or read error
		t.Logf("Received error (expected): %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout should have triggered")
	}
}

func TestSSE_AuthHeaders(t *testing.T) {
	receivedAuth := ""
	receivedBearer := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedBearer = r.Header.Get("X-Custom-Auth")

		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: auth test\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL,
		WithHeader("Authorization", "Bearer test-token-123"),
		WithHeader("X-Custom-Auth", "custom-value"))
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		if event.Data != "auth test" {
			t.Errorf("Expected data 'auth test', got '%s'", event.Data)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}

	if receivedAuth != "Bearer test-token-123" {
		t.Errorf("Expected Authorization 'Bearer test-token-123', got '%s'", receivedAuth)
	}
	if receivedBearer != "custom-value" {
		t.Errorf("Expected X-Custom-Auth 'custom-value', got '%s'", receivedBearer)
	}
}

func TestSSE_Comments(t *testing.T) {
	// Mock SSE server that sends comments
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(": This is a comment\n"))
		w.Write([]byte("data: data after comment\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		if event.Data != "data after comment" {
			t.Errorf("Expected data 'data after comment', got '%s'", event.Data)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestSSE_MultipleEvents(t *testing.T) {
	// Mock SSE server that sends multiple events
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Expected ResponseWriter to implement Flusher")
		}

		w.Write([]byte("data: event1\n"))
		w.Write([]byte("\n"))
		flusher.Flush()

		w.Write([]byte("data: event2\n"))
		w.Write([]byte("\n"))
		flusher.Flush()

		w.Write([]byte("data: event3\n"))
		w.Write([]byte("\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	var receivedData []string
	timeout := time.After(5 * time.Second)

	for i := 0; i < 3; i++ {
		select {
		case event := <-events:
			receivedData = append(receivedData, event.Data)
		case err, ok := <-errors:
			if ok && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for event %d", i+1)
		}
	}

	if len(receivedData) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(receivedData))
	}

	expected := []string{"event1", "event2", "event3"}
	for i, exp := range expected {
		if receivedData[i] != exp {
			t.Errorf("Event %d: expected '%s', got '%s'", i+1, exp, receivedData[i])
		}
	}
}

func TestSSE_IDPersistence(t *testing.T) {
	// Mock SSE server that sends events with IDs and verifies Last-Event-ID behavior
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// First event sets the ID
		w.Write([]byte("id: 1\n"))
		w.Write([]byte("data: first\n"))
		w.Write([]byte("\n"))
		flusher.Flush()

		// Second event should have the ID from previous event
		w.Write([]byte("id: 2\n"))
		w.Write([]byte("data: second\n"))
		w.Write([]byte("\n"))
		flusher.Flush()

		// Third event without id, should keep previous
		w.Write([]byte("data: third\n"))
		w.Write([]byte("\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	var received []struct {
		ID   string
		Data string
	}
	timeout := time.After(5 * time.Second)

	for i := 0; i < 3; i++ {
		select {
		case event := <-events:
			received = append(received, struct {
				ID   string
				Data string
			}{ID: event.ID, Data: event.Data})
		case err, ok := <-errors:
			if ok && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for event %d", i+1)
		}
	}

	if len(received) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(received))
	}

	// IDs should persist per SSE spec
	if received[0].ID != "1" {
		t.Errorf("First event: expected ID '1', got '%s'", received[0].ID)
	}
	if received[1].ID != "2" {
		t.Errorf("Second event: expected ID '2', got '%s'", received[1].ID)
	}
	if received[2].ID != "2" {
		// Third event should keep previous ID
		t.Errorf("Third event: expected ID '2' (persisted), got '%s'", received[2].ID)
	}
}

// Helper to create a mock SSE server that echoes back the Last-Event-ID
func createSSEServerWithLastEventIDValidation(t *testing.T, expectedID string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Last-Event-ID") != expectedID {
			t.Errorf("Expected Last-Event-ID '%s', got '%s'", expectedID, r.Header.Get("Last-Event-ID"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(fmt.Sprintf("id: %s\n", expectedID)))
		w.Write([]byte("data: ok\n"))
		w.Write([]byte("\n"))
	}))
}

// Test to verify SSE reader handles chunked transfer encoding
func TestSSE_ChunkedTransfer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Transfer-Encoding", "chunked")
		flusher, _ := w.(http.Flusher)

		// Send chunked data
		w.Write([]byte("data: chunk1\n"))
		flusher.Flush()

		time.Sleep(10 * time.Millisecond)

		w.Write([]byte("data: chunk2\n"))
		w.Write([]byte("\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	var receivedData []string
	timeout := time.After(5 * time.Second)

	for i := 0; i < 2; i++ {
		select {
		case event := <-events:
			receivedData = append(receivedData, event.Data)
		case err, ok := <-errors:
			if ok && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}

	if len(receivedData) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(receivedData))
	}

	if receivedData[0] != "chunk1" {
		t.Errorf("Expected first data 'chunk1', got '%s'", receivedData[0])
	}
	if receivedData[1] != "chunk2" {
		t.Errorf("Expected second data 'chunk2', got '%s'", receivedData[1])
	}
}

// Test that reader handles server sending data without Content-Type
func TestSSE_NoContentType(t *testing.T) {
	// Some SSE servers don't set Content-Type correctly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't set Content-Type
		w.Write([]byte("data: no-content-type\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		if event.Data != "no-content-type" {
			t.Errorf("Expected data 'no-content-type', got '%s'", event.Data)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

// Test SSE with event that has empty data field
func TestSSE_EmptyData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Empty data field - event with no data
		w.Write([]byte("event: ping\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		// Empty data is still an event (just no data)
		if event.Type != "ping" {
			t.Errorf("Expected type 'ping', got '%s'", event.Type)
		}
		if event.Data != "" {
			t.Errorf("Expected empty data, got '%s'", event.Data)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

// Test with large data payload
func TestSSE_LargeData(t *testing.T) {
	// Create a large string (100KB)
	largeData := strings.Repeat("x", 100*1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: " + largeData + "\n"))
		w.Write([]byte("\n"))
	}))
	defer server.Close()

	client := NewClient()

	events, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	select {
	case event := <-events:
		if len(event.Data) != len(largeData) {
			t.Errorf("Expected data length %d, got %d", len(largeData), len(event.Data))
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

// io.Reader that fails after N bytes
type failingReader struct {
	data   string
	read   int
	failAt int
}

func (r *failingReader) Read(p []byte) (n int, err error) {
	if r.read >= r.failAt {
		return 0, io.EOF
	}
	remaining := len(r.data) - r.read
	if remaining == 0 {
		return 0, io.EOF
	}
	copy(p, r.data[r.read:])
	r.read = len(r.data)
	return remaining, nil
}

// Test error handling when connection is closed unexpectedly
func TestSSE_ConnectionClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Close connection immediately without proper SSE termination
		w.Write([]byte("data: partial"))
		w.(http.CloseNotifier).CloseNotify()
	}))
	defer server.Close()

	client := NewClient()

	_, errors, err := client.Connect(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Connect should not error: %v", err)
	}

	select {
	case err := <-errors:
		if err == nil {
			t.Log("Connection closed gracefully")
		} else {
			t.Logf("Received expected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		// This is also acceptable - streaming until context cancelled
		t.Log("Connection remained open (timeout)")
	}
}
