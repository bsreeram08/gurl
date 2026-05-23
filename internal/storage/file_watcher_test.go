package storage

import (
	"context"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/pkg/types"
)

func TestCollectionWatcherDetectsUpdates(t *testing.T) {
	store := newWatcherTestStore(t)
	if err := store.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "list payments",
		URL:        "https://old.example.com",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher, err := store.WatchCollection(ctx, "payments", CollectionWatchOptions{
		PollInterval: 5 * time.Millisecond,
		Debounce:     15 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("WatchCollection failed: %v", err)
	}
	defer watcher.Stop()

	if err := store.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "list payments",
		URL:        "https://new.example.com",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("SaveRequest update failed: %v", err)
	}

	change := waitForCollectionChange(t, watcher)
	if change.Collection != "payments" {
		t.Fatalf("expected payments change, got %q", change.Collection)
	}
}

func TestCollectionWatcherDebouncesUpdates(t *testing.T) {
	store := newWatcherTestStore(t)
	req := &types.SavedRequest{
		ID:         "req-1",
		Name:       "list payments",
		URL:        "https://old.example.com",
		Collection: "payments",
	}
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher, err := store.WatchCollection(ctx, "payments", CollectionWatchOptions{
		PollInterval: 5 * time.Millisecond,
		Debounce:     60 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("WatchCollection failed: %v", err)
	}
	defer watcher.Stop()

	for _, url := range []string{
		"https://new.example.com/a",
		"https://new.example.com/a/b",
		"https://new.example.com/a/b/c",
	} {
		req.URL = url
		if err := store.SaveRequest(req); err != nil {
			t.Fatalf("SaveRequest update failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	waitForCollectionChange(t, watcher)
	select {
	case change := <-watcher.Events():
		t.Fatalf("expected debounced updates to emit one change, got extra %+v", change)
	case <-time.After(90 * time.Millisecond):
	}
}

func TestCollectionWatcherStopsOnContextCancel(t *testing.T) {
	store := newWatcherTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	watcher, err := store.WatchCollections(ctx, CollectionWatchOptions{
		PollInterval: 5 * time.Millisecond,
		Debounce:     15 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("WatchCollections failed: %v", err)
	}

	cancel()
	select {
	case <-watcher.Done():
	case <-time.After(time.Second):
		t.Fatal("watcher did not stop after context cancellation")
	}

	select {
	case _, ok := <-watcher.Events():
		if ok {
			t.Fatal("expected events channel to close after watcher stops")
		}
	case <-time.After(time.Second):
		t.Fatal("events channel did not close after watcher stops")
	}
}

func newWatcherTestStore(t *testing.T) *FileStore {
	t.Helper()
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileStore(proj)
	if err := store.SaveCollection(types.NewCollection("payments")); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	return store
}

func waitForCollectionChange(t *testing.T, watcher *CollectionWatcher) CollectionChange {
	t.Helper()
	select {
	case change := <-watcher.Events():
		return change
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for collection change")
	}
	return CollectionChange{}
}
