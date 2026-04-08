package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	var err error
	db.DB, err = leveldb.OpenFile(db.dbPath, &opt.Options{
		WriteBuffer: 4 * 1024 * 1024, // 4MB write buffer
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.MigrateIfNeeded(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Close closes the database
func (db *LMDB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// SaveRequest saves a request to the database atomically
func (db *LMDB) SaveRequest(req *types.SavedRequest) error {
	if req.ID == "" {
		req.ID = uuid.New().String()
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

	// Atomic write
	if err := db.DB.Write(batch, nil); err != nil {
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
	return db.GetRequest(id)
}

// ListRequests returns all requests matching the options
func (db *LMDB) ListRequests(opts *ListOptions) ([]*types.SavedRequest, error) {
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
		// List all requests - scan all keys with "request:" prefix
		iter := db.DB.NewIterator(nil, nil)
		defer iter.Release()

		for iter.Next() {
			key := string(iter.Key())
			if len(key) > 8 && key[:8] == "request:" {
				requestIDs = append(requestIDs, key[8:])
			}
		}
	}

	// Fetch and filter requests
	var results []*types.SavedRequest
	for _, id := range requestIDs {
		req, err := db.GetRequest(id)
		if err != nil {
			continue // Skip requests that can't be retrieved
		}

		// Apply pattern filter
		if opts.Pattern != "" {
			match := false
			for _, pattern := range []string{req.Name, req.URL} {
				if contains(pattern, opts.Pattern) {
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

// contains checks if a string contains a substring (simple implementation)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

// findSubstring implements a simple substring search
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// DeleteRequest deletes a request by ID
func (db *LMDB) DeleteRequest(id string) error {
	// First get the request to clean up indices
	req, err := db.GetRequest(id)
	if err != nil {
		return err
	}

	// Delete the request
	key := fmt.Sprintf("request:%s", id)
	if err := db.DB.Delete([]byte(key), nil); err != nil {
		return fmt.Errorf("failed to delete request: %w", err)
	}

	// Delete from name index
	nameKey := fmt.Sprintf("idx:name:%s", req.Name)
	db.DB.Delete([]byte(nameKey), nil)

	// Delete from collection index
	if req.Collection != "" {
		colKey := fmt.Sprintf("idx:collection:%s", req.Collection)
		db.removeFromIndex(colKey, id)
	}

	// Delete from tag indices
	for _, tag := range req.Tags {
		tagKey := fmt.Sprintf("idx:tag:%s", tag)
		db.removeFromIndex(tagKey, id)
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

// UpdateRequest updates an existing request
func (db *LMDB) UpdateRequest(req *types.SavedRequest) error {
	if req.ID == "" {
		return fmt.Errorf("cannot update request without ID")
	}

	// Get existing request to clean up old indices
	existing, err := db.GetRequest(req.ID)
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

	return db.SaveRequest(req)
}

// SaveHistory saves an execution history entry
func (db *LMDB) SaveHistory(history *types.ExecutionHistory) error {
	if history.ID == "" {
		history.ID = uuid.New().String()
	}

	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Use timestamp as part of key for ordering
	key := fmt.Sprintf("history:%s:%d", history.RequestID, history.Timestamp)
	if err := db.DB.Put([]byte(key), data, nil); err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

// GetHistory retrieves execution history for a request
func (db *LMDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
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
	folderKey := fmt.Sprintf("idx:folder:%s", path)
	requestIDs := db.getFromIndex(folderKey)
	if requestIDs == nil {
		return []*types.SavedRequest{}, nil
	}

	var results []*types.SavedRequest
	for _, id := range requestIDs {
		req, err := db.GetRequest(id)
		if err != nil {
			continue
		}
		results = append(results, req)
	}

	return results, nil
}

func (db *LMDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) {
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
		req, err := db.GetRequest(id)
		if err != nil {
			continue
		}
		results = append(results, req)
	}

	return results, nil
}

func (db *LMDB) DeleteFolder(path string) error {
	requests, err := db.ListFolder(path)
	if err != nil {
		return err
	}

	for _, req := range requests {
		req.Folder = ""
		if err := db.UpdateRequest(req); err != nil {
			return err
		}
	}

	return nil
}

func (db *LMDB) GetAllFolders() ([]string, error) {
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
