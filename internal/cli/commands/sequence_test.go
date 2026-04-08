package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestSequenceSetCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "sets order for existing request",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:   "req-1",
					Name: "test-request",
					URL:  "https://example.com",
				}
				db.names["test-request"] = "req-1"
			},
			args:    []string{"set", "test-request", "5"},
			wantErr: false,
		},
		{
			name:    "fails when request name is missing",
			setup:   func(db *mockDB) {},
			args:    []string{"set"},
			wantErr: true,
		},
		{
			name:    "fails when order is not a number",
			setup:   func(db *mockDB) {},
			args:    []string{"set", "test-request", "abc"},
			wantErr: true,
		},
		{
			name:    "fails when order is negative",
			setup:   func(db *mockDB) {},
			args:    []string{"set", "test-request", "-1"},
			wantErr: true,
		},
		{
			name:  "fails for non-existent request",
			setup: func(db *mockDB) {},
			args:  []string{"set", "nonexistent", "5"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := SequenceCommand(db)
			fullArgs := append([]string{"sequence"}, tt.args...)

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

func TestSequenceListCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "lists requests in collection",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:         "req-1",
					Name:       "first-request",
					URL:        "https://example.com",
					Collection: "mycoll",
					SortOrder:  1,
				}
				db.names["first-request"] = "req-1"
				db.requests["req-2"] = &types.SavedRequest{
					ID:         "req-2",
					Name:       "second-request",
					URL:        "https://example.com",
					Collection: "mycoll",
					SortOrder:  2,
				}
				db.names["second-request"] = "req-2"
			},
			args:    []string{"list", "mycoll"},
			wantErr: false,
		},
		{
			name:    "fails when collection name is missing",
			setup:   func(db *mockDB) {},
			args:    []string{"list"},
			wantErr: true,
		},
		{
			name:  "shows empty message for non-existent collection",
			setup: func(db *mockDB) {},
			args:  []string{"list", "nonexistent"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := SequenceCommand(db)
			fullArgs := append([]string{"sequence"}, tt.args...)

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
