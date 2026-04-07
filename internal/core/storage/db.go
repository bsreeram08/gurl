package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sreeram/gurl/pkg/types"
)

// DB represents the database interface
type DB interface {
	SaveRequest(req *types.SavedRequest) error
	GetRequest(name string) (*types.SavedRequest, error)
	DeleteRequest(name string) error
	RenameRequest(oldName, newName string) error
	ListRequests(opts types.ListOptions) ([]*types.SavedRequest, error)
	SaveHistory(entry *types.HistoryEntry) error
	Close() error
}

// LMDB implements DB using LMDB
type LMDB struct {
	dbPath string
}

// NewLMDB creates a new LMDB instance
func NewLMDB() (*LMDB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".local", "share", "scurl")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	return &LMDB{
		dbPath: filepath.Join(dbDir, "scurl.db"),
	}, nil
}

// InMemoryDB implements DB using in-memory storage (for testing/phase1)
type InMemoryDB struct {
	requests map[string]*types.SavedRequest
}

// NewInMemoryDB creates a new in-memory database
func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		requests: make(map[string]*types.SavedRequest),
	}
}

// SaveRequest saves a request to the database
func (db *InMemoryDB) SaveRequest(req *types.SavedRequest) error {
	db.requests[req.Name] = req
	return nil
}

// GetRequest retrieves a request by name
func (db *InMemoryDB) GetRequest(name string) (*types.SavedRequest, error) {
	req, ok := db.requests[name]
	if !ok {
		return nil, fmt.Errorf("request not found: %s", name)
	}
	return req, nil
}

// DeleteRequest deletes a request by name
func (db *InMemoryDB) DeleteRequest(name string) error {
	if _, ok := db.requests[name]; !ok {
		return fmt.Errorf("request not found: %s", name)
	}
	delete(db.requests, name)
	return nil
}

// RenameRequest renames a request
func (db *InMemoryDB) RenameRequest(oldName, newName string) error {
	req, ok := db.requests[oldName]
	if !ok {
		return fmt.Errorf("request not found: %s", oldName)
	}
	req.Name = newName
	db.requests[newName] = req
	delete(db.requests, oldName)
	return nil
}

// ListRequests returns all requests matching the options
func (db *InMemoryDB) ListRequests(opts types.ListOptions) ([]*types.SavedRequest, error) {
	var results []*types.SavedRequest

	for _, req := range db.requests {
		// Apply filters
		if opts.Collection != "" && req.Collection != opts.Collection {
			continue
		}
		if opts.Tag != "" {
			found := false
			for _, t := range req.Tags {
				if t == opts.Tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		results = append(results, req)
	}

	return results, nil
}

// SaveHistory saves a history entry
func (db *InMemoryDB) SaveHistory(entry *types.HistoryEntry) error {
	// In-memory implementation - history not stored
	return nil
}

// Close closes the database
func (db *InMemoryDB) Close() error {
	return nil
}

// JSONMarshal is exported for use by commands
var JSONMarshal = json.Marshal
var JSONUnmarshal = json.Unmarshal
