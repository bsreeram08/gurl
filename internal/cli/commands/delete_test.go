package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestDeleteCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
		checkFn func(*testing.T, *mockDB)
	}{
		{
			name: "deletes existing request",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:   "id1",
					Name: "to_delete",
					URL:  "https://delete.me",
				}
				db.names["to_delete"] = "id1"
			},
			args:    []string{"to_delete"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("to_delete")
				if req != nil {
					t.Error("expected request to be deleted")
				}
			},
		},
		{
			name:    "fails for non-existent request",
			setup:   func(db *mockDB) {},
			args:    []string{"nonexistent"},
			wantErr: true,
		},
		{
			name: "fails when no argument provided",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:   "id1",
					Name: "some_req",
					URL:  "https://example.com",
				}
				db.names["some_req"] = "id1"
			},
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

			cmd := DeleteCommand(db)

			fullArgs := append([]string{"delete"}, tt.args...)

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
