// Package memstore is an in-memory storage.DB adapter for tests. It keeps
// saved requests, execution history, folders and collections (including
// collection lock state) in maps and never touches disk.
//
// The zero value is ready to use. Values are copied on the way in and out,
// so callers cannot mutate stored records through returned pointers.
package memstore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// DB implements storage.DB in memory.
type DB struct {
	mu          sync.Mutex
	requests    map[string]*types.SavedRequest // by ID
	requestIDs  map[string]string              // name -> ID
	history     []*types.ExecutionHistory
	collections map[string]*types.Collection // by ID
	locked      map[string]string            // collection ID -> unlock passphrase
}

// New returns an empty DB.
func New() *DB {
	return &DB{}
}

func (db *DB) init() {
	if db.requests == nil {
		db.requests = make(map[string]*types.SavedRequest)
		db.requestIDs = make(map[string]string)
		db.collections = make(map[string]*types.Collection)
		db.locked = make(map[string]string)
	}
}

func (db *DB) Open() error  { return nil }
func (db *DB) Close() error { return nil }

// SaveRequest stores req, reusing the ID of an existing request with the same
// name when req has none, and creates its collection if missing (as LMDB
// does).
func (db *DB) SaveRequest(req *types.SavedRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	db.init()

	if req.ID == "" {
		if id, ok := db.requestIDs[req.Name]; ok {
			req.ID = id
		} else {
			req.ID = uuid.New().String()
		}
	}
	db.putRequestLocked(req)
	if req.Collection != "" && db.collectionByNameLocked(req.Collection) == nil {
		c := types.NewCollection(req.Collection)
		db.collections[c.ID] = c
	}
	return nil
}

func (db *DB) putRequestLocked(req *types.SavedRequest) {
	if old, ok := db.requests[req.ID]; ok && old.Name != req.Name {
		delete(db.requestIDs, old.Name)
	}
	db.requests[req.ID] = copyRequest(req)
	db.requestIDs[req.Name] = req.ID
}

func (db *DB) GetRequest(id string) (*types.SavedRequest, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	req, ok := db.requests[id]
	if !ok {
		return nil, fmt.Errorf("request not found: %s", id)
	}
	return copyRequest(req), nil
}

func (db *DB) GetRequestByName(name string) (*types.SavedRequest, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	id, ok := db.requestIDs[name]
	if !ok {
		return nil, fmt.Errorf("request not found: %s", name)
	}
	return copyRequest(db.requests[id]), nil
}

// ListRequests filters by collection, tag and pattern (substring of name, URL
// or a header value), sorted by name, then applies the limit.
func (db *DB) ListRequests(opts *storage.ListOptions) ([]*types.SavedRequest, error) {
	if opts == nil {
		opts = &storage.ListOptions{}
	}
	return db.filterRequests(func(req *types.SavedRequest) bool {
		if opts.Collection != "" && req.Collection != opts.Collection {
			return false
		}
		if opts.Tag != "" && !containsString(req.Tags, opts.Tag) {
			return false
		}
		return opts.Pattern == "" || matchesPattern(req, opts.Pattern)
	}, opts.Limit), nil
}

func (db *DB) DeleteRequest(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	req, ok := db.requests[id]
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}
	delete(db.requestIDs, req.Name)
	delete(db.requests, id)
	return nil
}

func (db *DB) UpdateRequest(req *types.SavedRequest) error {
	if req == nil || req.ID == "" {
		return errors.New("cannot update request without ID")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	db.init()
	if _, ok := db.requests[req.ID]; !ok {
		return fmt.Errorf("request not found: %s", req.ID)
	}
	db.putRequestLocked(req)
	return nil
}

func (db *DB) SaveHistory(history *types.ExecutionHistory) error {
	if history == nil {
		return errors.New("history cannot be nil")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if history.ID == "" {
		history.ID = uuid.New().String()
	}
	h := *history
	db.history = append(db.history, &h)
	return nil
}

// GetHistory returns the request's history, most recent first. A
// non-positive limit means 100, as in LMDB.
func (db *DB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	if limit <= 0 {
		limit = 100
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	var results []*types.ExecutionHistory
	for i := len(db.history) - 1; i >= 0 && len(results) < limit; i-- {
		if db.history[i].RequestID == requestID {
			h := *db.history[i]
			results = append(results, &h)
		}
	}
	return results, nil
}

func (db *DB) ListFolder(path string) ([]*types.SavedRequest, error) {
	return db.filterRequests(func(req *types.SavedRequest) bool {
		return req.Folder == path
	}, 0), nil
}

func (db *DB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) {
	prefix := strings.TrimSuffix(path, "/") + "/"
	return db.filterRequests(func(req *types.SavedRequest) bool {
		return req.Folder == path || strings.HasPrefix(req.Folder, prefix)
	}, 0), nil
}

// DeleteFolder moves the folder's requests (not its subfolders') to the root,
// as LMDB does.
func (db *DB) DeleteFolder(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, req := range db.requests {
		if req.Folder == path {
			req.Folder = ""
		}
	}
	return nil
}

func (db *DB) GetAllFolders() ([]string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	seen := make(map[string]bool)
	folders := []string{}
	for _, req := range db.requests {
		if req.Folder != "" && !seen[req.Folder] {
			seen[req.Folder] = true
			folders = append(folders, req.Folder)
		}
	}
	sort.Strings(folders)
	return folders, nil
}

func (db *DB) filterRequests(keep func(*types.SavedRequest) bool, limit int) []*types.SavedRequest {
	db.mu.Lock()
	defer db.mu.Unlock()
	var results []*types.SavedRequest
	for _, req := range db.requests {
		if keep(req) {
			results = append(results, copyRequest(req))
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// SaveCollection creates or replaces a collection. Names are unique, and a
// locked collection cannot be replaced (use SaveCollectionAllowLocked).
func (db *DB) SaveCollection(collection *types.Collection) error {
	return db.saveCollection(collection, false)
}

// SaveCollectionAllowLocked saves collection even over a locked one; the
// replacement is unlocked.
func (db *DB) SaveCollectionAllowLocked(collection *types.Collection) error {
	return db.saveCollection(collection, true)
}

func (db *DB) saveCollection(collection *types.Collection, allowLocked bool) error {
	if collection == nil {
		return errors.New("collection cannot be nil")
	}
	if collection.Name == "" {
		return errors.New("collection name cannot be empty")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	db.init()

	if existing := db.collectionByNameLocked(collection.Name); existing != nil && existing.ID != collection.ID {
		return fmt.Errorf("collection %q already exists", collection.Name)
	}
	if collection.ID == "" {
		collection.ID = uuid.New().String()
	}
	if err := db.lockedErrLocked(collection.ID); err != nil && !allowLocked {
		return err
	}
	now := time.Now().Unix()
	if collection.CreatedAt == 0 {
		collection.CreatedAt = now
	}
	if collection.UpdatedAt == 0 {
		collection.UpdatedAt = now
	}
	db.collections[collection.ID] = copyCollection(collection)
	delete(db.locked, collection.ID)
	return nil
}

func (db *DB) GetCollection(id string) (*types.Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	collection, ok := db.collections[id]
	if !ok {
		return nil, &storage.CollectionNotFoundError{Ref: id}
	}
	if err := db.lockedErrLocked(id); err != nil {
		return nil, err
	}
	return copyCollection(collection), nil
}

func (db *DB) GetCollectionByName(name string) (*types.Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	collection := db.collectionByNameLocked(name)
	if collection == nil {
		return nil, &storage.CollectionNotFoundError{Ref: name}
	}
	if err := db.lockedErrLocked(collection.ID); err != nil {
		return nil, err
	}
	return copyCollection(collection), nil
}

// GetRawCollectionByName returns the collection even when it is locked.
func (db *DB) GetRawCollectionByName(name string) (*types.Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	collection := db.collectionByNameLocked(name)
	if collection == nil {
		return nil, &storage.CollectionNotFoundError{Ref: name}
	}
	return copyCollection(collection), nil
}

// ListCollections returns every collection, locked ones included (as the
// file store does), sorted by name.
func (db *DB) ListCollections() ([]*types.Collection, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	results := make([]*types.Collection, 0, len(db.collections))
	for _, collection := range db.collections {
		results = append(results, copyCollection(collection))
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results, nil
}

func (db *DB) DeleteCollection(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, ok := db.collections[id]; !ok {
		return &storage.CollectionNotFoundError{Ref: id}
	}
	if err := db.lockedErrLocked(id); err != nil {
		return err
	}
	delete(db.collections, id)
	return nil
}

func (db *DB) UpdateCollection(collection *types.Collection) error {
	if collection == nil {
		return errors.New("collection cannot be nil")
	}
	if collection.ID == "" {
		return errors.New("cannot update collection without ID")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	existing, ok := db.collections[collection.ID]
	if !ok {
		return &storage.CollectionNotFoundError{Ref: collection.ID}
	}
	if err := db.lockedErrLocked(collection.ID); err != nil {
		return err
	}
	if other := db.collectionByNameLocked(collection.Name); other != nil && other.ID != collection.ID {
		return fmt.Errorf("collection %q already exists", collection.Name)
	}
	collection.CreatedAt = existing.CreatedAt
	collection.UpdatedAt = time.Now().Unix()
	db.collections[collection.ID] = copyCollection(collection)
	return nil
}

// Lock marks the named collection as locked until UnlockCollection is called
// with passphrase. It panics if the collection does not exist: it is a test
// setup helper and a missing collection is a bug in the test.
func (db *DB) Lock(name, passphrase string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	collection := db.collectionByNameLocked(name)
	if collection == nil {
		panic(fmt.Sprintf("memstore: Lock: collection %q not found", name))
	}
	db.locked[collection.ID] = passphrase
}

// UnlockCollection unlocks a collection locked with Lock.
func (db *DB) UnlockCollection(name string, passphrase string) error {
	if passphrase == "" {
		return errors.New("passphrase is required")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	collection := db.collectionByNameLocked(name)
	if collection == nil {
		return &storage.CollectionNotFoundError{Ref: name}
	}
	want, ok := db.locked[collection.ID]
	if !ok {
		return fmt.Errorf("collection %q is not passphrase-locked", name)
	}
	if passphrase != want {
		return fmt.Errorf("failed to unlock collection %q: wrong passphrase", name)
	}
	delete(db.locked, collection.ID)
	return nil
}

// MigrateCollectionToFiles is not supported: memstore has no file storage.
func (db *DB) MigrateCollectionToFiles(name string) (int, string, error) {
	return 0, "", storage.ErrNotSupported
}

// WatchCollection returns a nil watcher: memstore has no collection files.
func (db *DB) WatchCollection(ctx context.Context, name string, opts storage.CollectionWatchOptions) (*storage.CollectionWatcher, error) {
	return nil, nil
}

// WatchCollections returns a nil watcher: memstore has no collection files.
func (db *DB) WatchCollections(ctx context.Context, opts storage.CollectionWatchOptions) (*storage.CollectionWatcher, error) {
	return nil, nil
}

func (db *DB) collectionByNameLocked(name string) *types.Collection {
	for _, collection := range db.collections {
		if collection.Name == name {
			return collection
		}
	}
	return nil
}

func (db *DB) lockedErrLocked(id string) error {
	if _, ok := db.locked[id]; !ok {
		return nil
	}
	return &storage.CollectionLockedError{
		Name: db.collections[id].Name,
		Hint: "locked in memstore",
	}
}

func copyRequest(req *types.SavedRequest) *types.SavedRequest {
	c := *req
	return &c
}

func copyCollection(source *types.Collection) *types.Collection {
	c := *source
	c.Variables = make(map[string]string, len(source.Variables))
	for k, v := range source.Variables {
		c.Variables[k] = v
	}
	c.SecretKeys = make(map[string]bool, len(source.SecretKeys))
	for k, v := range source.SecretKeys {
		c.SecretKeys[k] = v
	}
	return &c
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// matchesPattern mirrors the LMDB filter: pattern is a substring of the
// request's name, URL, or any header value.
func matchesPattern(req *types.SavedRequest, pattern string) bool {
	if strings.Contains(req.Name, pattern) || strings.Contains(req.URL, pattern) {
		return true
	}
	for _, h := range req.Headers {
		if strings.Contains(h.Value, pattern) {
			return true
		}
	}
	return false
}

var _ storage.DB = (*DB)(nil)
