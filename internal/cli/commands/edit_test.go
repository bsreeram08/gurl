package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestEditCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
		checkFn func(*testing.T, *mockDB)
	}{
		{
			name: "change method",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--method", "POST"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.Method != "POST" {
					t.Errorf("expected method POST, got %s", req.Method)
				}
			},
		},
		{
			name: "add header",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--header", "Authorization: Bearer token"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if len(req.Headers) != 1 {
					t.Errorf("expected 1 header, got %d", len(req.Headers))
				}
				if req.Headers[0].Key != "Authorization" || req.Headers[0].Value != "Bearer token" {
					t.Errorf("expected header Authorization: Bearer token, got %s: %s",
						req.Headers[0].Key, req.Headers[0].Value)
				}
			},
		},
		{
			name: "remove header",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
					Headers: []types.Header{
						{Key: "Authorization", Value: "Bearer token"},
						{Key: "Content-Type", Value: "application/json"},
					},
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--remove-header", "Authorization"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if len(req.Headers) != 1 {
					t.Errorf("expected 1 header remaining, got %d", len(req.Headers))
				}
				if req.Headers[0].Key != "Content-Type" {
					t.Errorf("expected remaining header Content-Type, got %s", req.Headers[0].Key)
				}
			},
		},
		{
			name: "change URL",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://old-api.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--url", "https://new-api.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.URL != "https://new-api.com" {
					t.Errorf("expected URL https://new-api.com, got %s", req.URL)
				}
			},
		},
		{
			name: "change body",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--body", `{"new":"data"}`},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.Body != `{"new":"data"}` {
					t.Errorf("expected body {\"new\":\"data\"}, got %s", req.Body)
				}
			},
		},
		{
			name: "set collection",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:         "id1",
					Name:       "api",
					URL:        "https://example.com",
					Method:     "GET",
					Collection: "v1",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--collection", "v2"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.Collection != "v2" {
					t.Errorf("expected collection v2, got %s", req.Collection)
				}
			},
		},
		{
			name: "add tag",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
					Tags:   []string{"existing"},
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--tag", "critical"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if len(req.Tags) != 2 {
					t.Errorf("expected 2 tags, got %d", len(req.Tags))
				}
				found := false
				for _, tag := range req.Tags {
					if tag == "critical" {
						found = true
					}
				}
				if !found {
					t.Error("expected tag 'critical' to be added")
				}
			},
		},
		{
			name: "add assertion",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--assert", "status=200"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if len(req.Assertions) != 1 {
					t.Errorf("expected 1 assertion, got %d", len(req.Assertions))
				}
				if req.Assertions[0].Field != "status" || req.Assertions[0].Op != "=" || req.Assertions[0].Value != "200" {
					t.Errorf("expected assertion status=200, got %s%s%s",
						req.Assertions[0].Field, req.Assertions[0].Op, req.Assertions[0].Value)
				}
			},
		},
		{
			name:    "fails for non-existent request",
			setup:   func(db *mockDB) {},
			args:    []string{"nonexistent", "--method", "POST"},
			wantErr: true,
		},
		{
			name: "fails for invalid HTTP method",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--method", "INVALID"},
			wantErr: true,
		},
		{
			name: "fails without request name argument",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "multiple flags in one command",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://old-api.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--method", "POST", "--url", "https://new-api.com", "--header", "X-Custom: value"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.Method != "POST" {
					t.Errorf("expected method POST, got %s", req.Method)
				}
				if req.URL != "https://new-api.com" {
					t.Errorf("expected URL https://new-api.com, got %s", req.URL)
				}
				if len(req.Headers) != 1 || req.Headers[0].Key != "X-Custom" {
					t.Errorf("expected header X-Custom: value")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EditCommand(db)

			fullArgs := append([]string{"edit"}, tt.args...)

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
