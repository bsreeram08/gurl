package storage

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

const (
	defaultCollectionWatchPollInterval = 100 * time.Millisecond
	defaultCollectionWatchDebounce     = 100 * time.Millisecond
)

type CollectionWatchOptions struct {
	PollInterval time.Duration
	Debounce     time.Duration
}

type CollectionChange struct {
	Collection string
	Time       time.Time
}

type CollectionWatcher struct {
	events <-chan CollectionChange
	done   <-chan struct{}
	cancel context.CancelFunc
}

type CollectionWatcherStore interface {
	WatchCollection(ctx context.Context, name string, opts CollectionWatchOptions) (*CollectionWatcher, error)
	WatchCollections(ctx context.Context, opts CollectionWatchOptions) (*CollectionWatcher, error)
}

func (w *CollectionWatcher) Events() <-chan CollectionChange {
	if w == nil {
		return nil
	}
	return w.events
}

func (w *CollectionWatcher) Done() <-chan struct{} {
	if w == nil {
		return nil
	}
	return w.done
}

func (w *CollectionWatcher) Stop() {
	if w == nil || w.cancel == nil {
		return
	}
	w.cancel()
	<-w.done
}

func (w *CollectionWatcher) Changed() bool {
	if w == nil {
		return false
	}
	changed := false
	for {
		select {
		case _, ok := <-w.events:
			if !ok {
				return changed
			}
			changed = true
		default:
			return changed
		}
	}
}

func (s *FileStore) WatchCollection(ctx context.Context, name string, opts CollectionWatchOptions) (*CollectionWatcher, error) {
	path, err := s.CollectionPath(name)
	if err != nil {
		return nil, err
	}
	return newCollectionWatcher(ctx, name, []string{path}, opts)
}

func (s *FileStore) WatchCollections(ctx context.Context, opts CollectionWatchOptions) (*CollectionWatcher, error) {
	if !s.Enabled() {
		return nil, nil
	}
	return newCollectionWatcher(ctx, "", []string{s.project.CollectionsDir()}, opts)
}

func (db *ProjectDB) WatchCollection(ctx context.Context, name string, opts CollectionWatchOptions) (*CollectionWatcher, error) {
	if !db.hasFileStore() {
		return nil, nil
	}
	return db.files.WatchCollection(ctx, name, opts)
}

func (db *ProjectDB) WatchCollections(ctx context.Context, opts CollectionWatchOptions) (*CollectionWatcher, error) {
	if !db.hasFileStore() {
		return nil, nil
	}
	return db.files.WatchCollections(ctx, opts)
}

func newCollectionWatcher(ctx context.Context, collection string, roots []string, opts CollectionWatchOptions) (*CollectionWatcher, error) {
	opts = normalizeCollectionWatchOptions(opts)
	watchCtx, cancel := context.WithCancel(ctx)
	events := make(chan CollectionChange, 8)
	done := make(chan struct{})

	initial, err := collectionWatchSnapshot(roots)
	if err != nil {
		cancel()
		return nil, err
	}

	watcher := &CollectionWatcher{
		events: events,
		done:   done,
		cancel: cancel,
	}

	go func() {
		defer close(done)
		defer close(events)

		last := initial
		ticker := time.NewTicker(opts.PollInterval)
		defer ticker.Stop()

		var debounce *time.Timer
		var debounceC <-chan time.Time
		pending := false

		stopDebounce := func() {
			if debounce == nil {
				return
			}
			if !debounce.Stop() {
				select {
				case <-debounce.C:
				default:
				}
			}
			debounce = nil
			debounceC = nil
		}
		defer stopDebounce()

		queueDebounce := func() {
			pending = true
			if debounce == nil {
				debounce = time.NewTimer(opts.Debounce)
				debounceC = debounce.C
				return
			}
			if !debounce.Stop() {
				select {
				case <-debounce.C:
				default:
				}
			}
			debounce.Reset(opts.Debounce)
			debounceC = debounce.C
		}

		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				next, err := collectionWatchSnapshot(roots)
				if err != nil {
					continue
				}
				if !reflect.DeepEqual(last, next) {
					last = next
					queueDebounce()
				}
			case <-debounceC:
				debounce = nil
				debounceC = nil
				if !pending {
					continue
				}
				pending = false
				select {
				case events <- CollectionChange{Collection: collection, Time: time.Now()}:
				case <-watchCtx.Done():
					return
				}
			}
		}
	}()

	return watcher, nil
}

func normalizeCollectionWatchOptions(opts CollectionWatchOptions) CollectionWatchOptions {
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultCollectionWatchPollInterval
	}
	if opts.Debounce <= 0 {
		opts.Debounce = defaultCollectionWatchDebounce
	}
	return opts
}

type collectionWatchFile struct {
	Size    int64
	ModTime int64
}

func collectionWatchSnapshot(roots []string) (map[string]collectionWatchFile, error) {
	snapshot := make(map[string]collectionWatchFile)
	for _, root := range roots {
		if root == "" {
			continue
		}
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			snapshot[path] = collectionWatchFile{
				Size:    info.Size(),
				ModTime: info.ModTime().UnixNano(),
			}
			return nil
		}); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
	}
	return snapshot, nil
}

var _ CollectionWatcherStore = (*FileStore)(nil)
var _ CollectionWatcherStore = (*ProjectDB)(nil)
