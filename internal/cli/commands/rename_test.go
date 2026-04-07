package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestRenameCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
		checkFn func(*testing.T, *mockDB)
	}{
		{
			name: "renames existing request",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:   "id1",
					Name: "old_name",
					URL:  "https://example.com",
				}
				db.names["old_name"] = "id1"
			},
			args:    []string{"old_name", "new_name"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("new_name")
				if req == nil {
					t.Error("expected new name to exist")
				}
				oldReq, _ := db.GetRequestByName("old_name")
				if oldReq != nil {
					t.Error("expected old name to not exist")
				}
			},
		},
		{
			name:    "fails for non-existent request",
			setup:   func(db *mockDB) {},
			args:    []string{"nonexistent", "new_name"},
			wantErr: true,
		},
		{
			name: "fails when only one argument provided",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:   "id1",
					Name: "some_req",
					URL:  "https://example.com",
				}
				db.names["some_req"] = "id1"
			},
			args:    []string{"some_req"},
			wantErr: true,
		},
		{
			name:    "fails when no arguments provided",
			setup:   func(db *mockDB) {},
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := RenameCommand(db)

			fullArgs := append([]string{"rename"}, tt.args...)

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
