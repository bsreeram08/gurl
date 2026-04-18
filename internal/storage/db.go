package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// DB represents the database interface
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
	ListFolder(path string) ([]*types.SavedRequest, error)
	ListFolderRecursive(path string) ([]*types.SavedRequest, error)
	DeleteFolder(path string) error
	GetAllFolders() ([]string, error)
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

// NewLMDB creates a new LMDB instance
func NewLMDB() (*LMDB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".local", "share", "gurl")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "gurl.db")
	if envPath := os.Getenv("GURL_DB_PATH"); envPath != "" {
		dbPath = envPath
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

	if req.ID == "" {
		// Check if a request with this name already exists
		nameKey := fmt.Sprintf("idx:name:%s", req.Name)
		if existingID, err := db.DB.Get([]byte(nameKey), nil); err == nil && len(existingID) > 0 {
			req.ID = string(existingID)
		} else {
			req.ID = uuid.New().String()
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use atomic batch write for all index updates
	batch := new(leveldb.Batch)

	// Save the request
	key := fmt.Sprintf("request:%s", req.ID)
	batch.Put([]byte(key), data)

	// Update name index
	nameKey := fmt.Sprintf("idx:name:%s", req.Name)
	batch.Put([]byte(nameKey), []byte(req.ID))

	// Add to collection index if set
	if req.Collection != "" {
		colKey := fmt.Sprintf("idx:collection:%s", req.Collection)
		db.addToIndexBatch(batch, colKey, req.ID)
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
		if opts.Pattern != "" {
			match := false
			for _, pattern := range []string{req.Name, req.URL} {
				if strings.Contains(pattern, opts.Pattern) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
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
	if req.ID == "" {
		// Check if a request with this name already exists
		nameKey := fmt.Sprintf("idx:name:%s", req.Name)
		if existingID, err := db.DB.Get([]byte(nameKey), nil); err == nil && len(existingID) > 0 {
			req.ID = string(existingID)
		} else {
			req.ID = uuid.New().String()
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use atomic batch write for all index updates
	batch := new(leveldb.Batch)

	// Save the request
	key := fmt.Sprintf("request:%s", req.ID)
	batch.Put([]byte(key), data)

	// Update name index
	nameKey := fmt.Sprintf("idx:name:%s", req.Name)
	batch.Put([]byte(nameKey), []byte(req.ID))

	// Add to collection index if set
	if req.Collection != "" {
		colKey := fmt.Sprintf("idx:collection:%s", req.Collection)
		db.addToIndexBatch(batch, colKey, req.ID)
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

// SaveHistory saves an execution history entry with per-request limit of 100 entries
func (db *LMDB) SaveHistory(history *types.ExecutionHistory) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if history.ID == "" {
		history.ID = uuid.New().String()
	}

	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Use timestamp as part of key for ordering
	key := fmt.Sprintf("history:%s:%d", history.RequestID, history.Timestamp)

	// Enforce history limit: collect existing keys and delete oldest if over 100
	const maxHistory = 100
	var historyKeys []int64
	prefix := fmt.Sprintf("history:%s:", history.RequestID)
	iter := db.DB.NewIterator(nil, nil)
	for iter.Next() {
		k := string(iter.Key())
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			var ts int64
			fmt.Sscanf(k[len(prefix):], "%d", &ts)
			historyKeys = append(historyKeys, ts)
		}
	}
	iter.Release()

	// If we would exceed limit, delete oldest entries
	if len(historyKeys) >= maxHistory {
		// Sort timestamps ascending (oldest first)
		for i := 0; i < len(historyKeys)-1; i++ {
			for j := i + 1; j < len(historyKeys); j++ {
				if historyKeys[i] > historyKeys[j] {
					historyKeys[i], historyKeys[j] = historyKeys[j], historyKeys[i]
				}
			}
		}
		// Delete oldest entries to get down to maxHistory - 1 (leaving room for new entry)
		toDelete := len(historyKeys) - (maxHistory - 1)
		batch := new(leveldb.Batch)
		for i := 0; i < toDelete; i++ {
			oldKey := fmt.Sprintf("history:%s:%d", history.RequestID, historyKeys[i])
			batch.Delete([]byte(oldKey))
		}
		if err := db.DB.Write(batch, &opt.WriteOptions{Sync: true}); err != nil {
			return fmt.Errorf("failed to trim old history: %w", err)
		}
	}

	if err := db.DB.Put([]byte(key), data, &opt.WriteOptions{Sync: true}); err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

// GetHistory retrieves execution history for a request
func (db *LMDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if limit <= 0 {
		limit = 100
	}

	var results []*types.ExecutionHistory
	prefix := fmt.Sprintf("history:%s:", requestID)

	iter := db.DB.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := string(iter.Key())
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			data := iter.Value()
			var history types.ExecutionHistory
			if err := json.Unmarshal(data, &history); err != nil {
				continue
			}
			results = append(results, &history)

			if len(results) >= limit {
				break
			}
		}
	}

	// Reverse to get most recent first
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return results, nil
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

var JSONMarshal = json.Marshal
var JSONUnmarshal = json.Unmarshal
