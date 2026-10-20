package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// ErrNotSupported is returned by adapters for operations their backing store
// cannot perform, e.g. unlocking a passphrase-protected collection without
// project file storage.
var ErrNotSupported = errors.New("operation not supported by this storage backend")

// DB is the storage seam: saved requests, execution history, folders and
// collections. Every adapter implements all of it; an operation an adapter
// cannot perform returns ErrNotSupported rather than being left out.
type DB interface {
	Open() error
	Close() error
	SaveRequest(req *types.SavedRequest) error
	GetRequest(id string) (*types.SavedRequest, error)
	GetRequestByName(name string) (*types.SavedRequest, error)
	ListRequests(opts *ListOptions) ([]*types.SavedRequest, error)
	DeleteRequest(id string) error
	UpdateRequest(req *types.SavedRequest) error
	SaveHistory(history *types.ExecutionHistory) error
	GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error)
	// GetHistoryByID returns the execution record whose ID is idOrPrefix or
	// starts with it. It fails with ErrHistoryNotFound or, when several
	// records match, ErrAmbiguousHistoryID.
	GetHistoryByID(idOrPrefix string) (*types.ExecutionHistory, error)
	ListFolder(path string) ([]*types.SavedRequest, error)
	ListFolderRecursive(path string) ([]*types.SavedRequest, error)
	DeleteFolder(path string) error
	GetAllFolders() ([]string, error)

	CollectionStore

	// GetRawCollectionByName returns a collection as stored, without
	// decrypting its secrets, so a locked collection can still be identified
	// (ID, CreatedAt) and overwritten.
	GetRawCollectionByName(name string) (*types.Collection, error)
	// SaveCollectionAllowLocked saves collection even when the stored
	// collection it replaces is locked.
	SaveCollectionAllowLocked(collection *types.Collection) error
	// UnlockCollection binds a passphrase-protected collection locally.
	UnlockCollection(name string, passphrase string) error
	// MigrateCollectionToFiles copies a DB-backed collection and its requests
	// into project file storage, returning the number of requests copied and
	// the collection directory.
	MigrateCollectionToFiles(name string) (int, string, error)

	// CollectionWatcherStore methods return a nil watcher (and no error)
	// when the adapter has no collection files to watch.
	CollectionWatcherStore
}

// ListOptions defines filtering options for listing requests
type ListOptions struct {
	Collection string
	Tag        string
	Pattern    string
	Limit      int
	Sort       string
}

// LMDB implements DB using goleveldb (LMDB-like storage)
type LMDB struct {
	DB     *leveldb.DB
	dbPath string
	mu     sync.Mutex
}

func resolveDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".local", "share", "gurl")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create db directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "gurl.db")
	if envPath := os.Getenv("GURL_DB_PATH"); envPath != "" {
		dbPath = envPath
	}

	return dbPath, nil
}

// NewLMDB creates a new LMDB instance
func NewLMDB() (*LMDB, error) {
	dbPath, err := resolveDBPath()
	if err != nil {
		return nil, err
	}

	return &LMDB{
		dbPath: dbPath,
	}, nil
}

// NewLMDBWithPath creates a new LMDB instance with a custom database path
func NewLMDBWithPath(dbPath string) *LMDB {
	return &LMDB{
		dbPath: dbPath,
	}
}

func (db *LMDB) Path() string {
	if db == nil {
		return ""
	}
	return db.dbPath
}

// Open opens the database
func (db *LMDB) Open() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	var err error
	db.DB, err = leveldb.OpenFile(db.dbPath, &opt.Options{
		WriteBuffer: 4 * 1024 * 1024, // 4MB write buffer
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize or migrate schema version
	const currentSchemaVersion = 3
	data, err := db.DB.Get([]byte("schema_version"), nil)
	if err != nil || data == nil {
		// Fresh DB or legacy DB without version key — write current version
		versionData, _ := json.Marshal(currentSchemaVersion)
		if err := db.DB.Put([]byte("schema_version"), versionData, &opt.WriteOptions{Sync: true}); err != nil {
			return fmt.Errorf("failed to write schema version: %w", err)
		}
	} else {
		var version int
		if err := json.Unmarshal(data, &version); err == nil && version < currentSchemaVersion {
			// Migrate: update to current version
			versionData, _ := json.Marshal(currentSchemaVersion)
			db.DB.Put([]byte("schema_version"), versionData, &opt.WriteOptions{Sync: true})
		}
	}

	return nil
}

// Close closes the database
func (db *LMDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// GetSchemaVersion returns the current schema version of the database
func (db *LMDB) GetSchemaVersion() (int, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, err := db.DB.Get([]byte("schema_version"), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}

	var version int
	if err := json.Unmarshal(data, &version); err != nil {
		return 0, fmt.Errorf("failed to unmarshal schema version: %w", err)
	}

	return version, nil
}

// SaveRequest saves a request to the database atomically.
// If a request with this name already exists, it is updated in-place (upsert).
func (db *LMDB) SaveRequest(req *types.SavedRequest) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	var existing *types.SavedRequest
	if req.ID == "" {
		// Check if a request with this name already exists
		nameKey := fmt.Sprintf("idx:name:%s", req.Name)
		if existingID, err := db.DB.Get([]byte(nameKey), nil); err == nil && len(existingID) > 0 {
			req.ID = string(existingID)
		} else {
			req.ID = uuid.New().String()
		}
	}
	if req.ID != "" {
		if found, err := db.getRequestLocked(req.ID); err == nil {
			existing = found
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use atomic batch write for all index updates
	batch := new(leveldb.Batch)
	if existing != nil {
		if err := db.cleanupRequestIndicesBatch(batch, existing, req); err != nil {
			return err
		}
	}

	// Save the request
	key := fmt.Sprintf("request:%s", req.ID)
	batch.Put([]byte(key), data)

	// Update name index
	nameKey := fmt.Sprintf("idx:name:%s", req.Name)
	batch.Put([]byte(nameKey), []byte(req.ID))

	// Add to collection index if set
	if req.Collection != "" {
		if err := db.ensureCollectionBatch(batch, req.Collection); err != nil {
			return err
		}
		colKey := fmt.Sprintf("idx:collection:%s", req.Collection)
		if err := db.addToIndexBatch(batch, colKey, req.ID); err != nil {
			return err
		}
	}

	// Add to tag indices
	for _, tag := range req.Tags {
		tagKey := fmt.Sprintf("idx:tag:%s", tag)
		db.addToIndexBatch(batch, tagKey, req.ID)
	}

	if req.Folder != "" {
		folderKey := fmt.Sprintf("idx:folder:%s", req.Folder)
		db.addToIndexBatch(batch, folderKey, req.ID)
	}

	// Atomic write with sync for durability
	if err := db.DB.Write(batch, &opt.WriteOptions{Sync: true}); err != nil {
		return fmt.Errorf("failed to save request atomically: %w", err)
	}

	return nil
}

// addToIndexBatch adds a request ID to an index using a batch
func (db *LMDB) addToIndexBatch(batch *leveldb.Batch, indexKey string, requestID string) error {
	indexData, err := db.DB.Get([]byte(indexKey), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return fmt.Errorf("failed to read index: %w", err)
	}

	var ids []string
	if err == nil {
		if err := json.Unmarshal(indexData, &ids); err != nil {
			return fmt.Errorf("failed to unmarshal index: %w", err)
		}
	}

	// Check if already in index
	for _, id := range ids {
		if id == requestID {
			return nil
		}
	}

	ids = append(ids, requestID)
	newData, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	batch.Put([]byte(indexKey), newData)
	return nil
}

// addToIndex adds a request ID to an index
func (db *LMDB) addToIndex(indexKey string, requestID string) error {
	indexData, err := db.DB.Get([]byte(indexKey), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return fmt.Errorf("failed to read index: %w", err)
	}

	var ids []string
	if err == nil {
		if err := json.Unmarshal(indexData, &ids); err != nil {
			return fmt.Errorf("failed to unmarshal index: %w", err)
		}
	}

	// Check if already in index
	for _, id := range ids {
		if id == requestID {
			return nil
		}
	}

	ids = append(ids, requestID)
	newData, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return db.DB.Put([]byte(indexKey), newData, nil)
}

// GetRequest retrieves a request by ID
func (db *LMDB) GetRequest(id string) (*types.SavedRequest, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	key := fmt.Sprintf("request:%s", id)
	data, err := db.DB.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("request not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	var req types.SavedRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &req, nil
}

// GetRequestByName retrieves a request by name
func (db *LMDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Look up in name index
	nameKey := fmt.Sprintf("idx:name:%s", name)
	idData, err := db.DB.Get([]byte(nameKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("request not found: %s", name)
		}
		return nil, fmt.Errorf("failed to look up name index: %w", err)
	}

	id := string(idData)
	return db.getRequestLocked(id)
}

// ListRequests returns all requests matching the options
func (db *LMDB) ListRequests(opts *ListOptions) ([]*types.SavedRequest, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if opts == nil {
		opts = &ListOptions{}
	}

	var requestIDs []string

	// Determine which index to use
	switch {
	case opts.Collection != "":
		colKey := fmt.Sprintf("idx:collection:%s", opts.Collection)
		requestIDs = db.getFromIndex(colKey)
	case opts.Tag != "":
		tagKey := fmt.Sprintf("idx:tag:%s", opts.Tag)
		requestIDs = db.getFromIndex(tagKey)
	default:
		iter := db.DB.NewIterator(nil, nil)
		defer iter.Release()

		seekKey := []byte("request:")
		for iter.Seek(seekKey); iter.Valid(); iter.Next() {
			key := string(iter.Key())
			if len(key) < 8 || key[:8] != "request:" {
				break
			}
			requestIDs = append(requestIDs, key[8:])
		}
	}

	// Fetch and filter requests, deduplicating by name (keep only the canonical entry)
	seen := make(map[string]bool)
	var results []*types.SavedRequest
	for _, id := range requestIDs {
		req, err := db.getRequestLocked(id)
		if err != nil {
			continue // Skip requests that can't be retrieved
		}

		// Deduplicate: skip orphans (entries whose name index doesn't point to this ID)
		if seen[req.Name] {
			continue
		}
		nameKey := fmt.Sprintf("idx:name:%s", req.Name)
		canonicalID, err := db.DB.Get([]byte(nameKey), nil)
		if err != nil || string(canonicalID) != id {
			continue // orphaned entry — no name index or index points elsewhere
		}
		seen[req.Name] = true

		// Apply pattern filter
		if opts.Pattern != "" && !matchesPattern(req, opts.Pattern) {
			continue
		}

		results = append(results, req)
	}

	// Apply limit
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// getRequestLocked retrieves a request by ID (must be called with lock held)
func (db *LMDB) getRequestLocked(id string) (*types.SavedRequest, error) {
	key := fmt.Sprintf("request:%s", id)
	data, err := db.DB.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("request not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	var req types.SavedRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &req, nil
}

// getFromIndex retrieves all request IDs from an index
func (db *LMDB) getFromIndex(indexKey string) []string {
	data, err := db.DB.Get([]byte(indexKey), nil)
	if err != nil {
		return nil
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil
	}

	return ids
}

// DeleteRequest deletes a request by ID
func (db *LMDB) DeleteRequest(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// First get the request to clean up indices
	req, err := db.getRequestLocked(id)
	if err != nil {
		return err
	}

	// Use batch for atomic delete of request + all index entries
	batch := new(leveldb.Batch)

	// Delete the request
	key := fmt.Sprintf("request:%s", id)
	batch.Delete([]byte(key))

	// Delete from name index
	nameKey := fmt.Sprintf("idx:name:%s", req.Name)
	batch.Delete([]byte(nameKey))

	// Delete from collection index
	if req.Collection != "" {
		colKey := fmt.Sprintf("idx:collection:%s", req.Collection)
		db.removeFromIndexBatch(batch, colKey, id)
	}

	// Delete from tag indices
	for _, tag := range req.Tags {
		tagKey := fmt.Sprintf("idx:tag:%s", tag)
		db.removeFromIndexBatch(batch, tagKey, id)
	}

	// Delete from folder index
	if req.Folder != "" {
		folderKey := fmt.Sprintf("idx:folder:%s", req.Folder)
		db.removeFromIndexBatch(batch, folderKey, id)
	}

	// Atomic write with sync
	if err := db.DB.Write(batch, &opt.WriteOptions{Sync: true}); err != nil {
		return fmt.Errorf("failed to delete request: %w", err)
	}

	return nil
}

// removeFromIndex removes a request ID from an index
func (db *LMDB) removeFromIndex(indexKey string, requestID string) error {
	indexData, err := db.DB.Get([]byte(indexKey), nil)
	if err != nil {
		return nil // Index doesn't exist, nothing to remove
	}

	var ids []string
	if err := json.Unmarshal(indexData, &ids); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	// Filter out the request ID
	var newIDs []string
	for _, id := range ids {
		if id != requestID {
			newIDs = append(newIDs, id)
		}
	}

	newData, err := json.Marshal(newIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return db.DB.Put([]byte(indexKey), newData, nil)
}

// removeFromIndexBatch removes a request ID from an index using a batch
func (db *LMDB) removeFromIndexBatch(batch *leveldb.Batch, indexKey string, requestID string) error {
	indexData, err := db.DB.Get([]byte(indexKey), nil)
	if err != nil {
		return nil // Index doesn't exist, nothing to remove
	}

	var ids []string
	if err := json.Unmarshal(indexData, &ids); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	// Filter out the request ID
	var newIDs []string
	for _, id := range ids {
		if id != requestID {
			newIDs = append(newIDs, id)
		}
	}

	newData, err := json.Marshal(newIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	batch.Put([]byte(indexKey), newData)
	return nil
}

func (db *LMDB) cleanupRequestIndicesBatch(batch *leveldb.Batch, existing *types.SavedRequest, next *types.SavedRequest) error {
	if existing == nil || next == nil {
		return nil
	}
	if existing.Name != "" && existing.Name != next.Name {
		batch.Delete([]byte(fmt.Sprintf("idx:name:%s", existing.Name)))
	}
	if existing.Collection != "" && existing.Collection != next.Collection {
		if err := db.removeFromIndexBatch(batch, fmt.Sprintf("idx:collection:%s", existing.Collection), existing.ID); err != nil {
			return err
		}
	}
	nextTags := stringSet(next.Tags)
	for _, tag := range existing.Tags {
		if !nextTags[tag] {
			if err := db.removeFromIndexBatch(batch, fmt.Sprintf("idx:tag:%s", tag), existing.ID); err != nil {
				return err
			}
		}
	}
	if existing.Folder != "" && existing.Folder != next.Folder {
		if err := db.removeFromIndexBatch(batch, fmt.Sprintf("idx:folder:%s", existing.Folder), existing.ID); err != nil {
			return err
		}
	}
	return nil
}

// UpdateRequest updates an existing request
func (db *LMDB) UpdateRequest(req *types.SavedRequest) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if req.ID == "" {
		return fmt.Errorf("cannot update request without ID")
	}

	// Get existing request to clean up old indices
	existing, err := db.getRequestLocked(req.ID)
	if err != nil {
		return fmt.Errorf("request not found: %w", err)
	}

	// Remove old collection index
	if existing.Collection != "" && existing.Collection != req.Collection {
		colKey := fmt.Sprintf("idx:collection:%s", existing.Collection)
		db.removeFromIndex(colKey, req.ID)
	}

	for _, tag := range existing.Tags {
		tagKey := fmt.Sprintf("idx:tag:%s", tag)
		db.removeFromIndex(tagKey, req.ID)
	}

	if existing.Folder != "" && existing.Folder != req.Folder {
		folderKey := fmt.Sprintf("idx:folder:%s", existing.Folder)
		db.removeFromIndex(folderKey, req.ID)
	}

	return db.saveRequestLocked(req)
}

// saveRequestLocked saves a request to the database atomically (must be called with lock held).
// If a request with this name already exists, it is updated in-place (upsert).
func (db *LMDB) saveRequestLocked(req *types.SavedRequest) error {
	var existing *types.SavedRequest
	if req.ID == "" {
		// Check if a request with this name already exists
		nameKey := fmt.Sprintf("idx:name:%s", req.Name)
		if existingID, err := db.DB.Get([]byte(nameKey), nil); err == nil && len(existingID) > 0 {
			req.ID = string(existingID)
		} else {
			req.ID = uuid.New().String()
		}
	}
	if req.ID != "" {
		if found, err := db.getRequestLocked(req.ID); err == nil {
			existing = found
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use atomic batch write for all index updates
	batch := new(leveldb.Batch)
	if existing != nil {
		if err := db.cleanupRequestIndicesBatch(batch, existing, req); err != nil {
			return err
		}
	}

	// Save the request
	key := fmt.Sprintf("request:%s", req.ID)
	batch.Put([]byte(key), data)

	// Update name index
	nameKey := fmt.Sprintf("idx:name:%s", req.Name)
	batch.Put([]byte(nameKey), []byte(req.ID))

	// Add to collection index if set
	if req.Collection != "" {
		if err := db.ensureCollectionBatch(batch, req.Collection); err != nil {
			return err
		}
		colKey := fmt.Sprintf("idx:collection:%s", req.Collection)
		if err := db.addToIndexBatch(batch, colKey, req.ID); err != nil {
			return err
		}
	}

	// Add to tag indices
	for _, tag := range req.Tags {
		tagKey := fmt.Sprintf("idx:tag:%s", tag)
		db.addToIndexBatch(batch, tagKey, req.ID)
	}

	if req.Folder != "" {
		folderKey := fmt.Sprintf("idx:folder:%s", req.Folder)
		db.addToIndexBatch(batch, folderKey, req.ID)
	}

	// Atomic write with sync for durability
	if err := db.DB.Write(batch, &opt.WriteOptions{Sync: true}); err != nil {
		return fmt.Errorf("failed to save request atomically: %w", err)
	}

	return nil
}

// maxHistory is the number of execution records kept per request.
const maxHistory = 100

// historyKey is "history:<requestID>:n<unix nanos, 20 digits>:<id>". The
// "n" sorts after the digits of the old "history:<requestID>:<unix seconds>"
// keys, so old and new keys of a request iterate oldest first together, and
// the id keeps two records saved in the same instant apart.
func historyKey(h *types.ExecutionHistory) string {
	ns := h.TimestampNs
	if ns == 0 {
		ns = h.Timestamp * int64(time.Second)
	}
	return fmt.Sprintf("history:%s:n%020d:%s", h.RequestID, ns, h.ID)
}

// SaveHistory saves an execution record, keeping the newest maxHistory
// records per request.
func (db *LMDB) SaveHistory(history *types.ExecutionHistory) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if history.ID == "" {
		history.ID = uuid.New().String()
	}
	if history.Timestamp == 0 && history.TimestampNs == 0 {
		now := time.Now()
		history.Timestamp, history.TimestampNs = now.Unix(), now.UnixNano()
	}

	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Keys iterate oldest first; drop the oldest to leave room for this one.
	var existing [][]byte
	iter := db.DB.NewIterator(util.BytesPrefix([]byte("history:"+history.RequestID+":")), nil)
	for iter.Next() {
		existing = append(existing, append([]byte(nil), iter.Key()...))
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return fmt.Errorf("failed to read history: %w", err)
	}

	batch := new(leveldb.Batch)
	for i := 0; i < len(existing)-(maxHistory-1); i++ {
		batch.Delete(existing[i])
	}
	batch.Put([]byte(historyKey(history)), data)
	if err := db.DB.Write(batch, &opt.WriteOptions{Sync: true}); err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}
	return nil
}

// GetHistory returns the newest limit execution records of a request, most
// recent first. A non-positive limit means maxHistory.
func (db *LMDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if limit <= 0 {
		limit = maxHistory
	}

	var results []*types.ExecutionHistory
	iter := db.DB.NewIterator(util.BytesPrefix([]byte("history:"+requestID+":")), nil)
	defer iter.Release()
	for ok := iter.Last(); ok && len(results) < limit; ok = iter.Prev() {
		var history types.ExecutionHistory
		if err := json.Unmarshal(iter.Value(), &history); err != nil {
			continue
		}
		results = append(results, &history)
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("failed to read history: %w", err)
	}
	return results, nil
}

// GetHistoryByID returns the execution record whose ID is idOrPrefix or
// starts with it.
func (db *LMDB) GetHistoryByID(idOrPrefix string) (*types.ExecutionHistory, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	prefix := normalizeHistoryID(idOrPrefix)
	if prefix == "" {
		return nil, ErrHistoryIDRequired
	}
	// Records are marshalled with "id" first; checking the raw JSON skips
	// decoding every record. The decoded ID is still compared below.
	needle := []byte(`"id":"` + prefix)
	var candidates []*types.ExecutionHistory
	iter := db.DB.NewIterator(util.BytesPrefix([]byte("history:")), nil)
	defer iter.Release()
	for iter.Next() {
		if !bytes.Contains(iter.Value(), needle) {
			continue
		}
		var history types.ExecutionHistory
		if err := json.Unmarshal(iter.Value(), &history); err != nil {
			continue
		}
		candidates = append(candidates, &history)
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("failed to read history: %w", err)
	}
	return MatchHistoryID(candidates, prefix)
}

func (db *LMDB) ListFolder(path string) ([]*types.SavedRequest, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	folderKey := fmt.Sprintf("idx:folder:%s", path)
	requestIDs := db.getFromIndex(folderKey)
	if requestIDs == nil {
		return []*types.SavedRequest{}, nil
	}

	var results []*types.SavedRequest
	for _, id := range requestIDs {
		req, err := db.getRequestLocked(id)
		if err != nil {
			continue
		}
		results = append(results, req)
	}

	return results, nil
}

func (db *LMDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	iter := db.DB.NewIterator(nil, nil)
	defer iter.Release()

	var requestIDs []string
	folderPrefix := fmt.Sprintf("idx:folder:%s/", path)
	exactFolder := fmt.Sprintf("idx:folder:%s", path)

	for iter.Next() {
		key := string(iter.Key())
		if key == exactFolder {
			data := iter.Value()
			var ids []string
			if err := json.Unmarshal(data, &ids); err == nil {
				requestIDs = append(requestIDs, ids...)
			}
		} else if strings.HasPrefix(key, folderPrefix) {
			data := iter.Value()
			var ids []string
			if err := json.Unmarshal(data, &ids); err == nil {
				requestIDs = append(requestIDs, ids...)
			}
		}
	}

	if len(requestIDs) == 0 {
		return []*types.SavedRequest{}, nil
	}

	uniqueIDs := uniqueStrings(requestIDs)
	var results []*types.SavedRequest
	for _, id := range uniqueIDs {
		req, err := db.getRequestLocked(id)
		if err != nil {
			continue
		}
		results = append(results, req)
	}

	return results, nil
}

func (db *LMDB) DeleteFolder(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	requests, err := db.listFolderLocked(path)
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)
	for _, req := range requests {
		req.Folder = ""
		data, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		key := fmt.Sprintf("request:%s", req.ID)
		batch.Put([]byte(key), data)

		// Remove from old folder index
		folderKey := fmt.Sprintf("idx:folder:%s", path)
		db.removeFromIndexBatch(batch, folderKey, req.ID)
	}

	if err := db.DB.Write(batch, &opt.WriteOptions{Sync: true}); err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	return nil
}

// listFolderLocked lists requests in a folder (must be called with lock held)
func (db *LMDB) listFolderLocked(path string) ([]*types.SavedRequest, error) {
	folderKey := fmt.Sprintf("idx:folder:%s", path)
	requestIDs := db.getFromIndex(folderKey)
	if requestIDs == nil {
		return []*types.SavedRequest{}, nil
	}

	var results []*types.SavedRequest
	for _, id := range requestIDs {
		req, err := db.getRequestLocked(id)
		if err != nil {
			continue
		}
		results = append(results, req)
	}

	return results, nil
}

func (db *LMDB) GetAllFolders() ([]string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	iter := db.DB.NewIterator(nil, nil)
	defer iter.Release()

	folderSet := make(map[string]bool)
	prefix := "idx:folder:"

	for iter.Next() {
		key := string(iter.Key())
		if strings.HasPrefix(key, prefix) {
			folderPath := strings.TrimPrefix(key, prefix)
			folderSet[folderPath] = true
		}
	}

	if len(folderSet) == 0 {
		return []string{}, nil
	}

	folders := make([]string, 0, len(folderSet))
	for folder := range folderSet {
		folders = append(folders, folder)
	}

	return folders, nil
}

func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

var JSONMarshal = json.Marshal
var JSONUnmarshal = json.Unmarshal

// matchesPattern reports whether pattern is a substring of the request's name,
// URL, or any header value (e.g. a merchant ID).
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
