package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

// enhancedMockDB adds history support for testing
type timelineMockDB struct {
	*mockDB
	history map[string][]*types.ExecutionHistory
}

func newTimelineMockDB() *timelineMockDB {
	return &timelineMockDB{
		mockDB:  newMockDB(),
		history: make(map[string][]*types.ExecutionHistory),
	}
}

func (m *timelineMockDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	entries, ok := m.history[requestID]
	if !ok {
		return nil, nil
	}
	if limit > 0 && len(entries) > limit {
		return entries[:limit], nil
	}
	return entries, nil
}

func TestTimelineCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*timelineMockDB)
		args    []string
		wantErr bool
	}{
		{
			name:  "shows empty message when no history",
			setup: func(db *timelineMockDB) {},
			args:  []string{},
		},
		{
			name: "shows timeline with history entries",
			setup: func(db *timelineMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				db.history["req-1"] = []*types.ExecutionHistory{
					{
						ID:         "hist-1",
						RequestID:  "req-1",
						StatusCode: 200,
						DurationMs: 100,
						SizeBytes:  1024,
						Timestamp:  1700000000,
						Response:   "OK",
					},
				}
			},
			args: []string{},
		},
		{
			name: "filters by --since flag (ignored, just passes through)",
			setup: func(db *timelineMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				db.history["req-1"] = []*types.ExecutionHistory{
					{
						ID:         "hist-1",
						RequestID:  "req-1",
						StatusCode: 200,
						DurationMs: 100,
						SizeBytes:  1024,
						Timestamp:  1700000000,
					},
				}
			},
			args: []string{"--since", "24h"},
		},
		{
			name: "filters by --filter flag",
			setup: func(db *timelineMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "api-request",
					URL:    "https://api.example.com",
					Method: "GET",
				}
				db.names["api-request"] = "req-1"
				db.requests["req-2"] = &types.SavedRequest{
					ID:     "req-2",
					Name:   "web-request",
					URL:    "https://web.example.com",
					Method: "GET",
				}
				db.names["web-request"] = "req-2"
				db.history["req-1"] = []*types.ExecutionHistory{
					{ID: "hist-1", RequestID: "req-1", StatusCode: 200, DurationMs: 100, Timestamp: 1700000000},
				}
				db.history["req-2"] = []*types.ExecutionHistory{
					{ID: "hist-2", RequestID: "req-2", StatusCode: 200, DurationMs: 100, Timestamp: 1700000001},
				}
			},
			args: []string{"--filter", "api"},
		},
		{
			name: "applies --limit flag",
			setup: func(db *timelineMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "test-request",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["test-request"] = "req-1"
				// Add multiple history entries
				for i := 0; i < 5; i++ {
					db.history["req-1"] = append(db.history["req-1"], &types.ExecutionHistory{
						ID:         "hist-" + string(rune('0'+i)),
						RequestID:  "req-1",
						StatusCode: 200,
						DurationMs: int64(100 + i),
						Timestamp:  int64(1700000000 + i),
					})
				}
			},
			args: []string{"--limit", "3"},
		},
		{
			name: "shows multiple requests sorted by timestamp",
			setup: func(db *timelineMockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:     "req-1",
					Name:   "first-request",
					URL:    "https://first.example.com",
					Method: "GET",
				}
				db.names["first-request"] = "req-1"
				db.history["req-1"] = []*types.ExecutionHistory{
					{ID: "hist-1", RequestID: "req-1", StatusCode: 200, Timestamp: 1700000001},
				}
				db.requests["req-2"] = &types.SavedRequest{
					ID:     "req-2",
					Name:   "second-request",
					URL:    "https://second.example.com",
					Method: "GET",
				}
				db.names["second-request"] = "req-2"
				db.history["req-2"] = []*types.ExecutionHistory{
					{ID: "hist-2", RequestID: "req-2", StatusCode: 404, Timestamp: 1700000002},
				}
			},
			args: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTimelineMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := TimelineCommand(db)
			fullArgs := append([]string{"timeline"}, tt.args...)

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

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"hello", "", true},
		{"", "test", false},
		{"abc", "abcd", false},
	}

	for _, tt := range tests {
		got := contains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestSortByTimestamp(t *testing.T) {
	entries := []timelineEntry{
		{requestName: "first", history: &types.ExecutionHistory{Timestamp: 1000}},
		{requestName: "second", history: &types.ExecutionHistory{Timestamp: 3000}},
		{requestName: "third", history: &types.ExecutionHistory{Timestamp: 2000}},
	}

	sortByTimestamp(entries)

	// Should be sorted in descending order (most recent first)
	if entries[0].history.Timestamp != 3000 {
		t.Errorf("expected first entry to have timestamp 3000, got %d", entries[0].history.Timestamp)
	}
	if entries[1].history.Timestamp != 2000 {
		t.Errorf("expected second entry to have timestamp 2000, got %d", entries[1].history.Timestamp)
	}
	if entries[2].history.Timestamp != 1000 {
		t.Errorf("expected third entry to have timestamp 1000, got %d", entries[2].history.Timestamp)
	}
}

func TestTimelineEmptyWithNoRequests(t *testing.T) {
	db := newTimelineMockDB()
	// Empty database - no requests
	cmd := TimelineCommand(db)
	err := cmd.Run(context.Background(), []string{"timeline"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
