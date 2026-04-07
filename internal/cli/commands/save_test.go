package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestSaveCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		checkFn func(*testing.T, *mockDB)
	}{
		{
			name:    "saves basic request with name and URL",
			args:    []string{"test", "https://example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("test")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.URL != "https://example.com" {
					t.Errorf("expected URL 'https://example.com', got '%s'", req.URL)
				}
				if req.Method != "GET" {
					t.Errorf("expected method 'GET', got '%s'", req.Method)
				}
			},
		},
		{
			name:    "saves with custom format flag",
			args:    []string{"json_req", "https://api.example.com", "-f", "json"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("json_req")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req.OutputFormat != "json" {
					t.Errorf("expected format 'json', got '%s'", req.OutputFormat)
				}
			},
		},
		{
			name:    "fails when name argument is missing",
			args:    []string{"https://example.com"},
			wantErr: true,
		},
		{
			name:    "fails when URL argument is missing",
			args:    []string{"testname"},
			wantErr: true,
		},
		{
			name:    "saves multiple requests",
			args:    []string{"multi1", "https://first.example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				// Save another request
				db.names["multi2"] = "id2"
				db.requests["id2"] = &types.SavedRequest{
					ID:   "id2",
					Name: "multi2",
					URL:  "https://second.example.com",
				}

				req1, _ := db.GetRequestByName("multi1")
				req2, _ := db.GetRequestByName("multi2")
				if req1 == nil || req2 == nil {
					t.Fatal("expected both requests to exist")
				}
			},
		},
		{
			name:    "saves with description",
			args:    []string{"with_desc", "https://desc.example.com", "-d", "My description"},
			wantErr: false,
		},
		{
			name:    "saves with multiple tags",
			args:    []string{"multi_tag", "https://tag.example.com", "--tag", "api", "--tag", "auth"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("multi_tag")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if len(req.Tags) != 2 {
					t.Errorf("expected 2 tags, got %d", len(req.Tags))
				}
				if req.Tags[0] != "api" || req.Tags[1] != "auth" {
					t.Errorf("expected tags [api auth], got %v", req.Tags)
				}
			},
		},
		{
			name:    "saves with single tag",
			args:    []string{"single_tag", "https://single.example.com", "--tag", "important"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("single_tag")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if len(req.Tags) != 1 {
					t.Errorf("expected 1 tag, got %d", len(req.Tags))
				}
				if req.Tags[0] != "important" {
					t.Errorf("expected tag [important], got %v", req.Tags)
				}
			},
		},
		{
			name:    "saves with --curl flag and full curl command",
			args:    []string{"curl_test", "--curl", `curl -X POST -H "Content-Type: application/json" -d '{"key":"value"}' https://example.com`},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("curl_test")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.Method != "POST" {
					t.Errorf("expected method 'POST', got '%s'", req.Method)
				}
				if req.URL != "https://example.com" {
					t.Errorf("expected URL 'https://example.com', got '%s'", req.URL)
				}
				if len(req.Headers) != 1 {
					t.Errorf("expected 1 header, got %d", len(req.Headers))
				}
				if req.Body != `{"key":"value"}` {
					t.Errorf("expected body '{\"key\":\"value\"}', got '%s'", req.Body)
				}
			},
		},
		{
			name:    "saves with -X -H -d individual flags",
			args:    []string{"flags_test", "-X", "PUT", "-H", "Authorization: Bearer token123", "-d", "name=test", "https://api.example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("flags_test")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.Method != "PUT" {
					t.Errorf("expected method 'PUT', got '%s'", req.Method)
				}
				if req.URL != "https://api.example.com" {
					t.Errorf("expected URL 'https://api.example.com', got '%s'", req.URL)
				}
				if len(req.Headers) != 1 {
					t.Errorf("expected 1 header, got %d", len(req.Headers))
				}
				if req.Headers[0].Key != "Authorization" || req.Headers[0].Value != "Bearer token123" {
					t.Errorf("expected header 'Authorization: Bearer token123', got '%s: %s'", req.Headers[0].Key, req.Headers[0].Value)
				}
				if req.Body != "name=test" {
					t.Errorf("expected body 'name=test', got '%s'", req.Body)
				}
			},
		},
		{
			name:    "saves with multiple -H flags",
			args:    []string{"multi_header", "-X", "POST", "-H", "Content-Type: application/json", "-H", "Accept: text/plain", "https://multi-header.example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("multi_header")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if len(req.Headers) != 2 {
					t.Errorf("expected 2 headers, got %d", len(req.Headers))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			cmd := SaveCommand(db)

			// Build args slice: first element is command name (ignored by action)
			fullArgs := append([]string{"save"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, db)
			}
		})
	}
}
