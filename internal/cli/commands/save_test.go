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
