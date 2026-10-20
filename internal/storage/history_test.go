package storage_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/internal/storage/memstore"
	"github.com/sreeram/gurl/pkg/types"
)

func openLMDB(t *testing.T) *storage.LMDB {
	t.Helper()
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestLMDBHistorySameSecondSavesAreKept(t *testing.T) {
	db := openLMDB(t)
	// Old-style records carry only a seconds timestamp.
	first := &types.ExecutionHistory{ID: "aaaa0001", RequestID: "req", StatusCode: 200, Timestamp: 1700000000}
	second := &types.ExecutionHistory{ID: "aaaa0002", RequestID: "req", StatusCode: 500, Timestamp: 1700000000}
	third := types.NewExecutionHistory("req", "", 201, 1, 0)
	fourth := types.NewExecutionHistory("req", "", 202, 1, 0)
	fourth.Timestamp, fourth.TimestampNs = third.Timestamp, third.TimestampNs
	for _, h := range []*types.ExecutionHistory{first, second, third, fourth} {
		if err := db.SaveHistory(h); err != nil {
			t.Fatalf("SaveHistory: %v", err)
		}
	}

	got, err := db.GetHistory("req", 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d records, want 4 (same-second saves overwrote each other)", len(got))
	}
	if got[3].ID != "aaaa0001" || got[2].ID != "aaaa0002" {
		t.Fatalf("records not ordered newest first: %v", ids(got))
	}
}

func TestLMDBGetHistoryLimitReturnsNewest(t *testing.T) {
	db := openLMDB(t)
	for i := 1; i <= 5; i++ {
		h := &types.ExecutionHistory{ID: fmt.Sprintf("id-%d", i), RequestID: "req", StatusCode: 200, Timestamp: int64(1700000000 + i)}
		if err := db.SaveHistory(h); err != nil {
			t.Fatal(err)
		}
	}
	got, err := db.GetHistory("req", 2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(ids(got), ",") != "id-5,id-4" {
		t.Fatalf("got %v, want the two newest", ids(got))
	}
}

func TestLMDBReadsOldFormatHistoryKeys(t *testing.T) {
	db := openLMDB(t)
	old := types.ExecutionHistory{ID: "0ld00000-0000-0000-0000-000000000000", RequestID: "req", StatusCode: 200, Timestamp: 1700000000, Response: "old"}
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Put([]byte("history:req:1700000000"), data, nil); err != nil {
		t.Fatal(err)
	}
	fresh := types.NewExecutionHistory("req", "new", 500, 1, 3)
	if err := db.SaveHistory(fresh); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetHistory("req", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != fresh.ID || got[1].ID != old.ID {
		t.Fatalf("got %v, want new then old", ids(got))
	}
	byID, err := db.GetHistoryByID("0ld0")
	if err != nil || byID.Response != "old" {
		t.Fatalf("GetHistoryByID(old) = %+v, %v", byID, err)
	}
}

func TestLMDBHistoryTrimsOldestIncludingOldKeys(t *testing.T) {
	db := openLMDB(t)
	old, _ := json.Marshal(types.ExecutionHistory{ID: "old", RequestID: "req", Timestamp: 1})
	if err := db.DB.Put([]byte("history:req:1"), old, nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		h := &types.ExecutionHistory{ID: fmt.Sprintf("id-%03d", i), RequestID: "req", StatusCode: 200, Timestamp: int64(1700000000 + i)}
		if err := db.SaveHistory(h); err != nil {
			t.Fatal(err)
		}
	}
	got, err := db.GetHistory("req", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100 || got[0].ID != "id-099" || got[99].ID != "id-000" {
		t.Fatalf("got %d records (newest %s, oldest %s), want the 100 newest", len(got), got[0].ID, got[len(got)-1].ID)
	}
}

func TestGetHistoryByID(t *testing.T) {
	adapters := map[string]func(t *testing.T) storage.DB{
		"lmdb":     func(t *testing.T) storage.DB { return openLMDB(t) },
		"memstore": func(t *testing.T) storage.DB { return memstore.New() },
	}
	records := []*types.ExecutionHistory{
		{ID: "3f9a2c1b-1111-4000-8000-000000000001", RequestID: "a", StatusCode: 500, Timestamp: 1700000001},
		{ID: "3f9a2c1b-2222-4000-8000-000000000002", RequestID: "b", StatusCode: 200, Timestamp: 1700000002},
		{ID: "7c0ffee0-3333-4000-8000-000000000003", RequestID: "a", StatusCode: 200, Timestamp: 1700000003},
	}
	tests := []struct {
		name    string
		query   string
		wantID  string
		wantErr error
		errHas  string
	}{
		{name: "exact", query: "3f9a2c1b-2222-4000-8000-000000000002", wantID: records[1].ID},
		{name: "unique prefix", query: "7c0f", wantID: records[2].ID},
		{name: "prefix is case and space insensitive", query: " 7C0FFEE0 ", wantID: records[2].ID},
		{name: "longer prefix disambiguates", query: "3f9a2c1b-1", wantID: records[0].ID},
		{name: "ambiguous", query: "3f9a2c1b", wantErr: storage.ErrAmbiguousHistoryID, errHas: records[0].ID},
		{name: "ambiguous short", query: "3f9a", wantErr: storage.ErrAmbiguousHistoryID, errHas: "3f9a2c1b, 3f9a2c1b"},
		{name: "not found", query: "deadbeef", wantErr: storage.ErrHistoryNotFound},
		{name: "empty", query: "  ", wantErr: storage.ErrHistoryIDRequired},
	}
	for name, open := range adapters {
		t.Run(name, func(t *testing.T) {
			db := open(t)
			for _, h := range records {
				copy := *h
				if err := db.SaveHistory(&copy); err != nil {
					t.Fatal(err)
				}
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := db.GetHistoryByID(tt.query)
					if tt.wantErr != nil {
						if !errors.Is(err, tt.wantErr) {
							t.Fatalf("err = %v, want %v", err, tt.wantErr)
						}
						if tt.errHas != "" && !strings.Contains(err.Error(), tt.errHas) {
							t.Fatalf("error %q should mention %q", err, tt.errHas)
						}
						return
					}
					if err != nil {
						t.Fatalf("GetHistoryByID(%q): %v", tt.query, err)
					}
					if got.ID != tt.wantID {
						t.Fatalf("got %s, want %s", got.ID, tt.wantID)
					}
				})
			}
		})
	}
}

func ids(records []*types.ExecutionHistory) []string {
	out := make([]string, len(records))
	for i, h := range records {
		out[i] = h.ID
	}
	return out
}
