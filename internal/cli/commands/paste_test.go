package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestPasteCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "copies curl command for GET request",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
			},
			args:    []string{"test-request"},
			wantErr: false,
		},
		{
			name:    "fails when request name not provided",
			setup:   func(db *mockDB) {},
			args:    []string{},
			wantErr: true,
		},
		{
			name:  "fails for non-existent request",
			setup: func(db *mockDB) {},
			args:  []string{"nonexistent-request"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := PasteCommand(db)
			fullArgs := append([]string{"paste"}, tt.args...)

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

func TestIsCommandAvailable(t *testing.T) {
	if isCommandAvailable("nonexistent-command-xyz") {
		t.Error("expected nonexistent command to return false")
	}

	if !isCommandAvailable("true") {
		t.Error("expected 'true' command to be available")
	}
}

func TestPasteCommandWithBody(t *testing.T) {
	db := newMockDB()
	db.requests["req-1"] = &types.SavedRequest{
		ID:     "req-1",
		Name:   "post-request",
		URL:    "https://example.com/api",
		Method: "POST",
		Body:   `{"key":"value"}`,
	}
	db.names["post-request"] = "req-1"

	cmd := PasteCommand(db)
	err := cmd.Run(context.Background(), []string{"paste", "post-request"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
