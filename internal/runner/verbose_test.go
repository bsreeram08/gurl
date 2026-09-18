package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

func sampleExchange() (*client.Request, *client.Response) {
	req := &client.Request{
		Method: "POST",
		URL:    "https://api.example.com/posts",
		Headers: []client.Header{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "Authorization", Value: "Bearer super-secret-token"},
		},
		Body: `{"title":"hello"}`,
	}
	resp := &client.Response{
		StatusCode: 201,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
			"Set-Cookie":   []string{"session=abc123"},
		},
		Body:     []byte(`{"id":101}`),
		Duration: 544 * time.Millisecond,
		Size:     10,
	}
	return req, resp
}

func TestFormatVerboseExchangeShowsRequestAndResponse(t *testing.T) {
	out := FormatVerboseExchange(sampleExchange())

	for _, want := range []string{
		"> POST https://api.example.com/posts",
		"> Content-Type: application/json",
		`> {"title":"hello"}`,
		"< 201 Created",
		"< Content-Type: application/json",
		`< {"id":101}`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q\ngot:\n%s", want, out)
		}
	}
}

func TestFormatVerboseExchangeRedactsSensitiveHeaders(t *testing.T) {
	out := FormatVerboseExchange(sampleExchange())

	for _, secret := range []string{"super-secret-token", "session=abc123"} {
		if strings.Contains(out, secret) {
			t.Errorf("expected %q to be redacted\ngot:\n%s", secret, out)
		}
	}
	if !strings.Contains(out, "Authorization: "+redactedValue) {
		t.Errorf("expected redacted Authorization header\ngot:\n%s", out)
	}
	if !strings.Contains(out, "Set-Cookie: "+redactedValue) {
		t.Errorf("expected redacted Set-Cookie header\ngot:\n%s", out)
	}
}

func TestFormatVerboseExchangeRedactsRegardlessOfHeaderCase(t *testing.T) {
	req := &client.Request{
		Method:  "GET",
		URL:     "https://api.example.com",
		Headers: []client.Header{{Key: "AUTHORIZATION", Value: "Bearer leaked"}},
	}

	out := FormatVerboseExchange(req, nil)

	if strings.Contains(out, "leaked") {
		t.Errorf("expected uppercase header name to still be redacted\ngot:\n%s", out)
	}
}

func TestFormatVerboseExchangeTruncatesLargeBodies(t *testing.T) {
	large := strings.Repeat("x", verboseBodyLimit+500)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{},
		Body:       []byte(large),
		Size:       int64(len(large)),
	}

	out := FormatVerboseExchange(nil, resp)

	if strings.Contains(out, large) {
		t.Error("expected large body to be truncated, got it in full")
	}
	if !strings.Contains(out, "... (500 bytes truncated)") {
		t.Errorf("expected truncation notice\ngot:\n%s", out)
	}
}

func TestFormatVerboseExchangeOmitsMissingSides(t *testing.T) {
	req, resp := sampleExchange()

	// Match the line prefixes rather than bare angle brackets: the
	// "<redacted>" marker contains both.
	requestOnly := FormatVerboseExchange(req, nil)
	if strings.Contains(requestOnly, "  < ") {
		t.Errorf("expected no response lines when response is nil\ngot:\n%s", requestOnly)
	}

	responseOnly := FormatVerboseExchange(nil, resp)
	if strings.Contains(responseOnly, "  > ") {
		t.Errorf("expected no request lines when request is nil\ngot:\n%s", responseOnly)
	}

	if FormatVerboseExchange(nil, nil) != "" {
		t.Error("expected empty output when both sides are nil")
	}
}

func TestFormatVerboseExchangeSortsHeadersForDeterministicOutput(t *testing.T) {
	req := &client.Request{
		Method: "GET",
		URL:    "https://api.example.com",
		Headers: []client.Header{
			{Key: "Z-Last", Value: "z"},
			{Key: "A-First", Value: "a"},
		},
	}

	out := FormatVerboseExchange(req, nil)

	if strings.Index(out, "A-First") > strings.Index(out, "Z-Last") {
		t.Errorf("expected headers sorted by key\ngot:\n%s", out)
	}
}

func TestFormatVerboseExchangeSkipsEmptyBody(t *testing.T) {
	req := &client.Request{Method: "GET", URL: "https://api.example.com"}

	out := FormatVerboseExchange(req, nil)

	if strings.Contains(out, "  >\n") {
		t.Errorf("expected no body separator for an empty body\ngot:\n%s", out)
	}
}

func TestCollectionRunVerbosePrintsExchange(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":101}`))
	}))
	defer server.Close()

	collection := types.NewCollection("payments")
	if err := db.SaveCollection(collection); err != nil {
		t.Fatalf("failed to save collection: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "create-payment",
		Name:       "create payment",
		Method:     "POST",
		URL:        server.URL + "/payments",
		Headers:    []types.Header{{Key: "Authorization", Value: "Bearer super-secret-token"}},
		Body:       `{"amount":10}`,
		Collection: "payments",
	}); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	var out strings.Builder
	results, err := NewRunner(db, newTestEnvStorage(t)).Run(context.Background(), RunConfig{
		CollectionName: "payments",
		Verbose:        true,
		VerboseOut:     &out,
	})
	if err != nil {
		t.Fatalf("collection run failed: %v", err)
	}
	if len(results) != 1 || results[0].Passed != 1 {
		t.Fatalf("expected request to pass, got %+v", results)
	}

	output := out.String()
	for _, want := range []string{
		"create payment",
		"> POST " + server.URL + "/payments",
		`> {"amount":10}`,
		"< 201 Created",
		`< {"id":101}`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected verbose output to contain %q\ngot:\n%s", want, output)
		}
	}
	if strings.Contains(output, "super-secret-token") {
		t.Errorf("expected Authorization header to be redacted\ngot:\n%s", output)
	}
}

func TestCollectionRunWithoutVerbosePrintsNothing(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	if err := db.SaveCollection(types.NewCollection("payments")); err != nil {
		t.Fatalf("failed to save collection: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "ping",
		Name:       "ping",
		Method:     "GET",
		URL:        server.URL,
		Collection: "payments",
	}); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	var out strings.Builder
	if _, err := NewRunner(db, newTestEnvStorage(t)).Run(context.Background(), RunConfig{
		CollectionName: "payments",
		VerboseOut:     &out,
	}); err != nil {
		t.Fatalf("collection run failed: %v", err)
	}

	if out.String() != "" {
		t.Errorf("expected no verbose output when Verbose is false, got:\n%s", out.String())
	}
}
