package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

type diffMockDB struct {
	*mockDB
	history []*types.ExecutionHistory
}

func newDiffMockDB() *diffMockDB {
	return &diffMockDB{
		mockDB:  newMockDB(),
		history: []*types.ExecutionHistory{},
	}
}

func (m *diffMockDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	return m.history, nil
}

func TestDiffCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*diffMockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "compares two responses",
			setup: func(db *diffMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				db.history = []*types.ExecutionHistory{
					{ID: "hist-1", RequestID: "req-1", StatusCode: 200, DurationMs: 100, SizeBytes: 1024, Timestamp: 1700000001, Response: "Response A"},
					{ID: "hist-2", RequestID: "req-1", StatusCode: 200, DurationMs: 150, SizeBytes: 2048, Timestamp: 1700000000, Response: "Response B"},
				}
			},
			args:    []string{"test-request"},
			wantErr: false,
		},
		{
			name:    "fails when request name not provided",
			setup:   func(db *diffMockDB) {},
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "fails for non-existent request",
			setup:   func(db *diffMockDB) {},
			args:    []string{"nonexistent-request"},
			wantErr: true,
		},
		{
			name: "fails when only one history entry",
			setup: func(db *diffMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				db.history = []*types.ExecutionHistory{
					{ID: "hist-1", RequestID: "req-1", StatusCode: 200, Timestamp: 1700000000, Response: "Only one"},
				}
			},
			args:    []string{"test-request"},
			wantErr: true,
		},
		{
			name: "fails when no history",
			setup: func(db *diffMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				db.history = []*types.ExecutionHistory{}
			},
			args:    []string{"test-request"},
			wantErr: true,
		},
		{
			name: "applies --limit flag for more entries",
			setup: func(db *diffMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				db.history = []*types.ExecutionHistory{
					{ID: "hist-0", RequestID: "req-1", StatusCode: 200, DurationMs: 100, Timestamp: 1700000000, Response: "Response A"},
					{ID: "hist-1", RequestID: "req-1", StatusCode: 200, DurationMs: 110, Timestamp: 1700000001, Response: "Response B"},
					{ID: "hist-2", RequestID: "req-1", StatusCode: 200, DurationMs: 120, Timestamp: 1700000002, Response: "Response C"},
				}
			},
			args:    []string{"test-request", "--limit", "3"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newDiffMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := DiffCommand(db)
			fullArgs := append([]string{"diff"}, tt.args...)

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
