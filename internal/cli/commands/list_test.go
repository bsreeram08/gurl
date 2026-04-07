package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestListCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "lists all requests",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:   "id1",
					Name: "request1",
					URL:  "https://example.com",
				}
				db.names["request1"] = "id1"
				db.requests["id2"] = &types.SavedRequest{
					ID:   "id2",
					Name: "request2",
					URL:  "https://test.com",
				}
				db.names["request2"] = "id2"
			},
			args:    []string{},
			wantErr: false,
		},
		{
			name: "filters by collection",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:         "id1",
					Name:       "api_req",
					URL:        "https://api.example.com",
					Collection: "api",
				}
				db.names["api_req"] = "id1"
				db.requests["id2"] = &types.SavedRequest{
					ID:         "id2",
					Name:       "web_req",
					URL:        "https://web.example.com",
					Collection: "web",
				}
				db.names["web_req"] = "id2"
			},
			args:    []string{"--collection", "api"},
			wantErr: false,
		},
		{
			name: "filters by tag",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:   "id1",
					Name: "tagged_req",
					URL:  "https://tagged.example.com",
					Tags: []string{"important", "api"},
				}
				db.names["tagged_req"] = "id1"
			},
			args:    []string{"--tag", "important"},
			wantErr: false,
		},
		{
			name: "filters by pattern",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:   "id1",
					Name: "search_api",
					URL:  "https://api.example.com",
				}
				db.names["search_api"] = "id1"
				db.requests["id2"] = &types.SavedRequest{
					ID:   "id2",
					Name: "other_req",
					URL:  "https://other.com",
				}
				db.names["other_req"] = "id2"
			},
			args:    []string{"--pattern", "api"},
			wantErr: false,
		},
		{
			name: "outputs JSON",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:   "id1",
					Name: "json_req",
					URL:  "https://json.example.com",
				}
				db.names["json_req"] = "id1"
			},
			args:    []string{"--json"},
			wantErr: false,
		},
		{
			name: "applies limit",
			setup: func(db *mockDB) {
				for i := 0; i < 10; i++ {
					id := "id" + string(rune('0'+i))
					db.requests[id] = &types.SavedRequest{
						ID:   id,
						Name: "req" + string(rune('0'+i)),
						URL:  "https://example.com/" + string(rune('0'+i)),
					}
					db.names["req"+string(rune('0'+i))] = id
				}
			},
			args:    []string{"--limit", "5"},
			wantErr: false,
		},
		{
			name: "sorts by name",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:        "id1",
					Name:      "zebra",
					URL:       "https://zebra.com",
					UpdatedAt: 1000,
				}
				db.names["zebra"] = "id1"
				db.requests["id2"] = &types.SavedRequest{
					ID:        "id2",
					Name:      "alpha",
					URL:       "https://alpha.com",
					UpdatedAt: 2000,
				}
				db.names["alpha"] = "id2"
			},
			args:    []string{"--sort", "name"},
			wantErr: false,
		},
		{
			name:    "empty list shows no saved requests message",
			setup:   func(db *mockDB) {},
			args:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := ListCommand(db)

			fullArgs := append([]string{"list"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
