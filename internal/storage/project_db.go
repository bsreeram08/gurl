package storage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/pkg/types"
)

type ProjectDB struct {
	base  DB
	files *FileStore
}

func NewProjectDB(base DB, files *FileStore) *ProjectDB {
	return &ProjectDB{base: base, files: files}
}

func (db *ProjectDB) Open() error {
	return db.base.Open()
}

func (db *ProjectDB) Close() error {
	return db.base.Close()
}

func (db *ProjectDB) SaveRequest(req *types.SavedRequest) error {
	if db.hasFileStore() && req != nil && req.Collection != "" && db.files.HasCollection(req.Collection) {
		return db.files.SaveRequest(req)
	}
	return db.base.SaveRequest(req)
}

func (db *ProjectDB) GetRequest(id string) (*types.SavedRequest, error) {
	if db.hasFileStore() {
		if req, err := db.files.GetRequest(id); err == nil {
			return req, nil
		}
	}
	return db.base.GetRequest(id)
}

func (db *ProjectDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	if db.hasFileStore() {
		if req, err := db.files.GetRequestByName(name); err == nil {
			return req, nil
		}
	}
	return db.base.GetRequestByName(name)
}

func (db *ProjectDB) ListRequests(opts *ListOptions) ([]*types.SavedRequest, error) {
	if !db.hasFileStore() {
		return db.base.ListRequests(opts)
	}

	listOpts := cloneListOptions(opts)
	listOpts.Limit = 0

	fileRequests, err := db.files.ListRequests(&listOpts)
	if err != nil {
		return nil, err
	}

	baseRequests, err := db.base.ListRequests(&listOpts)
	if err != nil {
		return nil, err
	}

	fileCollectionNames, err := db.fileCollectionNameSet()
	if err != nil {
		return nil, err
	}
	filteredBase := make([]*types.SavedRequest, 0, len(baseRequests))
	for _, req := range baseRequests {
		if req.Collection != "" && fileCollectionNames[req.Collection] {
			continue
		}
		filteredBase = append(filteredBase, req)
	}

	requests := mergeRequestLists(fileRequests, filteredBase)
	sortRequestsByOptions(requests, opts)
	if opts != nil && opts.Limit > 0 && len(requests) > opts.Limit {
		requests = requests[:opts.Limit]
	}
	return requests, nil
}

func (db *ProjectDB) DeleteRequest(id string) error {
	if db.hasFileStore() && db.files.HasRequest(id) {
		if err := db.files.DeleteRequest(id); err != nil {
			return err
		}
		if _, err := db.base.GetRequest(id); err == nil {
			return db.base.DeleteRequest(id)
		}
		return nil
	}
	return db.base.DeleteRequest(id)
}

func (db *ProjectDB) UpdateRequest(req *types.SavedRequest) error {
	if db.hasFileStore() && req != nil {
		if db.files.HasRequest(req.ID) {
			if req.Collection == "" {
				if err := db.files.DeleteRequest(req.ID); err != nil {
					return err
				}
				return db.base.SaveRequest(req)
			}
			return db.files.SaveRequest(req)
		}
		if req.Collection != "" && db.files.HasCollection(req.Collection) {
			return db.files.SaveRequest(req)
		}
	}
	return db.base.UpdateRequest(req)
}

func (db *ProjectDB) SaveHistory(history *types.ExecutionHistory) error {
	return db.base.SaveHistory(history)
}

func (db *ProjectDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	return db.base.GetHistory(requestID, limit)
}

func (db *ProjectDB) ListFolder(path string) ([]*types.SavedRequest, error) {
	requests, err := db.ListRequests(nil)
	if err != nil {
		return nil, err
	}
	var results []*types.SavedRequest
	for _, req := range requests {
		if req.Folder == path {
			results = append(results, req)
		}
	}
	return results, nil
}

func (db *ProjectDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) {
	requests, err := db.ListRequests(nil)
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSuffix(path, "/") + "/"
	var results []*types.SavedRequest
	for _, req := range requests {
		if req.Folder == path || strings.HasPrefix(req.Folder, prefix) {
			results = append(results, req)
		}
	}
	return results, nil
}

func (db *ProjectDB) DeleteFolder(path string) error {
	if db.hasFileStore() {
		requests, err := db.files.ListFolder(path)
		if err != nil {
			return err
		}
		for _, req := range requests {
			req.Folder = ""
			req.UpdatedAt = time.Now().Unix()
			if err := db.files.SaveRequest(req); err != nil {
				return err
			}
		}
	}
	return db.base.DeleteFolder(path)
}

func (db *ProjectDB) GetAllFolders() ([]string, error) {
	requests, err := db.ListRequests(nil)
	if err != nil {
		return nil, err
	}
	folders := make(map[string]bool)
	for _, req := range requests {
		if req.Folder != "" {
			folders[req.Folder] = true
		}
	}
	result := make([]string, 0, len(folders))
	for folder := range folders {
		result = append(result, folder)
	}
	sort.Strings(result)
	return result, nil
}

func (db *ProjectDB) SaveCollection(collection *types.Collection) error {
	if db.hasFileStore() && db.collectionIsFileBacked(collection) {
		return db.files.SaveCollection(collection)
	}
	if store, ok := db.base.(CollectionStore); ok {
		if collection != nil && collection.ID != "" {
			if _, err := store.GetCollection(collection.ID); err == nil {
				return store.SaveCollection(collection)
			}
		}
		if collection != nil && collection.Name != "" {
			if _, err := store.GetCollectionByName(collection.Name); err == nil {
				return store.SaveCollection(collection)
			}
		}
	}
	if db.hasFileStore() {
		return db.files.SaveCollection(collection)
	}
	store, ok := db.base.(CollectionStore)
	if !ok {
		return fmt.Errorf("collection variables are not supported by this storage backend")
	}
	return store.SaveCollection(collection)
}

func (db *ProjectDB) GetCollection(id string) (*types.Collection, error) {
	if db.hasFileStore() {
		if collection, err := db.files.GetCollection(id); err == nil {
			return collection, nil
		}
	}
	store, ok := db.base.(CollectionStore)
	if !ok {
		return nil, fmt.Errorf("collection variables are not supported by this storage backend")
	}
	return store.GetCollection(id)
}

func (db *ProjectDB) GetCollectionByName(name string) (*types.Collection, error) {
	if db.hasFileStore() {
		if collection, err := db.files.GetCollectionByName(name); err == nil {
			return collection, nil
		}
	}
	store, ok := db.base.(CollectionStore)
	if !ok {
		return nil, fmt.Errorf("collection variables are not supported by this storage backend")
	}
	return store.GetCollectionByName(name)
}

func (db *ProjectDB) ListCollections() ([]*types.Collection, error) {
	byName := make(map[string]*types.Collection)
	if db.hasFileStore() {
		collections, err := db.files.ListCollections()
		if err != nil {
			return nil, err
		}
		for _, collection := range collections {
			byName[collection.Name] = collection
		}
	}
	if store, ok := db.base.(CollectionStore); ok {
		collections, err := store.ListCollections()
		if err != nil {
			return nil, err
		}
		for _, collection := range collections {
			if _, exists := byName[collection.Name]; !exists {
				byName[collection.Name] = collection
			}
		}
	}
	results := make([]*types.Collection, 0, len(byName))
	for _, collection := range byName {
		results = append(results, collection)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	return results, nil
}

func (db *ProjectDB) DeleteCollection(id string) error {
	if db.hasFileStore() {
		if _, err := db.files.GetCollection(id); err == nil {
			return db.files.DeleteCollection(id)
		}
	}
	store, ok := db.base.(CollectionStore)
	if !ok {
		return fmt.Errorf("collection variables are not supported by this storage backend")
	}
	return store.DeleteCollection(id)
}

func (db *ProjectDB) UpdateCollection(collection *types.Collection) error {
	if db.hasFileStore() && db.collectionIsFileBacked(collection) {
		return db.files.UpdateCollection(collection)
	}
	store, ok := db.base.(CollectionStore)
	if !ok {
		return fmt.Errorf("collection variables are not supported by this storage backend")
	}
	return store.UpdateCollection(collection)
}

func (db *ProjectDB) MigrateCollectionToFiles(name string) (int, string, error) {
	if !db.hasFileStore() {
		return 0, "", fmt.Errorf("gurl project not found; run 'gurl init' or set GURL_PROJECT_DIR")
	}

	var collection *types.Collection
	if store, ok := db.base.(CollectionStore); ok {
		if found, err := store.GetCollectionByName(name); err == nil {
			collection = found
		}
	}

	requests, err := db.base.ListRequests(&ListOptions{Collection: name})
	if err != nil {
		return 0, "", err
	}
	if collection == nil && len(requests) == 0 {
		return 0, "", fmt.Errorf("collection %q not found or empty", name)
	}
	if collection == nil {
		collection = types.NewCollection(name)
	}
	if err := db.files.SaveCollection(collection); err != nil {
		return 0, "", err
	}
	for _, req := range requests {
		copy := *req
		copy.Collection = name
		if err := db.files.SaveRequest(&copy); err != nil {
			return 0, "", err
		}
	}
	path, err := db.files.CollectionPath(name)
	if err != nil {
		return 0, "", err
	}
	return len(requests), path, nil
}

func (db *ProjectDB) FileStore() *FileStore {
	return db.files
}

func (db *ProjectDB) hasFileStore() bool {
	return db != nil && db.files != nil && db.files.Enabled()
}

func (db *ProjectDB) collectionIsFileBacked(collection *types.Collection) bool {
	if collection == nil || !db.hasFileStore() {
		return false
	}
	if collection.ID != "" && db.files.HasCollectionID(collection.ID) {
		return true
	}
	return collection.Name != "" && db.files.HasCollection(collection.Name)
}

func (s *FileStore) HasCollectionID(id string) bool {
	if !s.Enabled() || id == "" {
		return false
	}
	_, _, err := s.findCollectionByID(id)
	return err == nil
}

func (db *ProjectDB) fileCollectionNameSet() (map[string]bool, error) {
	names := make(map[string]bool)
	if !db.hasFileStore() {
		return names, nil
	}
	collections, err := db.files.ListCollections()
	if err != nil {
		return nil, err
	}
	for _, collection := range collections {
		names[collection.Name] = true
	}
	return names, nil
}

func cloneListOptions(opts *ListOptions) ListOptions {
	if opts == nil {
		return ListOptions{}
	}
	return *opts
}

func mergeRequestLists(primary []*types.SavedRequest, secondary []*types.SavedRequest) []*types.SavedRequest {
	seen := make(map[string]bool)
	merged := make([]*types.SavedRequest, 0, len(primary)+len(secondary))
	for _, req := range primary {
		key := requestIdentity(req)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, req)
	}
	for _, req := range secondary {
		key := requestIdentity(req)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, req)
	}
	return merged
}

func requestIdentity(req *types.SavedRequest) string {
	if req == nil {
		return ""
	}
	if req.ID != "" {
		return "id:" + req.ID
	}
	return "name:" + req.Collection + "\x00" + req.Name
}

var _ DB = (*ProjectDB)(nil)
var _ CollectionStore = (*ProjectDB)(nil)
