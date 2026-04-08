package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

type historyMockDB struct {
	*mockDB
	history []*types.ExecutionHistory
}

func newHistoryMockDB() *historyMockDB {
	return &historyMockDB{
		mockDB:  newMockDB(),
		history: []*types.ExecutionHistory{},
	}
}

func (m *historyMockDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	if limit > 0 && len(m.history) > limit {
		return m.history[:limit], nil
	}
	return m.history, nil
}

func TestHistoryCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*historyMockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "shows history for request with history",
			setup: func(db *historyMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				db.history = []*types.ExecutionHistory{
					{ID: "hist-1", RequestID: "req-1", StatusCode: 200, DurationMs: 100, SizeBytes: 1024, Timestamp: 1700000000, Response: "OK"},
				}
			},
			args:    []string{"test-request"},
			wantErr: false,
		},
		{
			name: "shows message when no history for request",
			setup: func(db *historyMockDB) {
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
			wantErr: false,
		},
		{
			name:    "fails when request name not provided",
			setup:   func(db *historyMockDB) {},
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "fails for non-existent request",
			setup:   func(db *historyMockDB) {},
			args:    []string{"nonexistent-request"},
			wantErr: true,
		},
		{
			name: "applies --limit flag",
			setup: func(db *historyMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				db.history = []*types.ExecutionHistory{
					{ID: "hist-0", RequestID: "req-1", StatusCode: 200, DurationMs: 100, Timestamp: 1700000000},
					{ID: "hist-1", RequestID: "req-1", StatusCode: 200, DurationMs: 110, Timestamp: 1700000001},
					{ID: "hist-2", RequestID: "req-1", StatusCode: 200, DurationMs: 120, Timestamp: 1700000002},
					{ID: "hist-3", RequestID: "req-1", StatusCode: 200, DurationMs: 130, Timestamp: 1700000003},
					{ID: "hist-4", RequestID: "req-1", StatusCode: 200, DurationMs: 140, Timestamp: 1700000004},
				}
			},
			args:    []string{"test-request", "--limit", "3"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newHistoryMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := HistoryCommand(db)
			fullArgs := append([]string{"history"}, tt.args...)

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

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0B"},
		{100, "100B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1024 * 1024, "1.0MB"},
		{1024 * 1024 * 2, "2.0MB"},
		{1024 * 1024 / 2, "512.0KB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
