package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/pkg/types"
)

const collectionFileName = "collection.json"

var errFileCollectionNotFound = errors.New("file collection not found")
var errFileRequestNotFound = errors.New("file request not found")

type FileStore struct {
	project *project.Project
}

type collectionRecord struct {
	collection *types.Collection
	dir        string
}

type requestRecord struct {
	request *types.SavedRequest
	path    string
}

func NewFileStore(proj *project.Project) *FileStore {
	return &FileStore{project: proj}
}

func (s *FileStore) Enabled() bool {
	return s != nil && s.project != nil && s.project.CollectionsDir() != ""
}

func (s *FileStore) SaveCollection(collection *types.Collection) error {
	return s.saveCollection(collection, false)
}

func (s *FileStore) SaveCollectionAllowLocked(collection *types.Collection) error {
	return s.saveCollection(collection, true)
}

func (s *FileStore) SaveCollectionPassphrase(collection *types.Collection, passphrase string) error {
	if err := s.SaveCollection(collection); err != nil {
		return err
	}
	dir, err := s.CollectionPath(collection.Name)
	if err != nil {
		return err
	}
	stored, err := encryptCollectionForPassphrase(collection, passphrase)
	if err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(dir, collectionFileName), stored); err != nil {
		return err
	}
	if collectionHasSecrets(stored) {
		if err := os.Remove(collectionKeyPath(dir)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove local collection key: %w", err)
		}
	}
	return nil
}

func (s *FileStore) saveCollection(collection *types.Collection, allowLocked bool) error {
	if !s.Enabled() {
		return fmt.Errorf("file storage is not configured")
	}
	if collection == nil {
		return fmt.Errorf("collection cannot be nil")
	}
	if collection.Name == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	findCollectionByName := s.findCollectionByName
	findCollectionByID := s.findCollectionByID
	if allowLocked {
		findCollectionByName = s.findRawCollectionByName
		findCollectionByID = s.findRawCollectionByID
	}

	now := time.Now().Unix()
	if collection.ID == "" {
		if existing, _, err := findCollectionByName(collection.Name); err == nil && existing != nil {
			return fmt.Errorf("collection %q already exists", collection.Name)
		} else if IsCollectionLocked(err) {
			return err
		}
		collection.ID = uuid.New().String()
	}

	var existingDir string
	if existing, dir, err := findCollectionByID(collection.ID); err == nil && existing != nil {
		existingDir = dir
		if collection.CreatedAt == 0 {
			collection.CreatedAt = existing.CreatedAt
		}
	} else if IsCollectionLocked(err) {
		return err
	}
	if existing, _, err := findCollectionByName(collection.Name); err == nil && existing.ID != collection.ID {
		return fmt.Errorf("collection %q already exists", collection.Name)
	} else if IsCollectionLocked(err) {
		return err
	}

	if collection.CreatedAt == 0 {
		collection.CreatedAt = now
	}
	if collection.UpdatedAt == 0 {
		collection.UpdatedAt = now
	}
	if collection.Variables == nil {
		collection.Variables = make(map[string]string)
	}
	if collection.SecretKeys == nil {
		collection.SecretKeys = make(map[string]bool)
	}

	targetDir := filepath.Join(s.project.CollectionsDir(), safePathComponent(collection.Name))
	if existingDir != "" && existingDir != targetDir {
		if _, err := os.Stat(targetDir); err == nil {
			return fmt.Errorf("collection path already exists: %s", targetDir)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect collection path: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
			return fmt.Errorf("failed to create collections directory: %w", err)
		}
		if err := os.Rename(existingDir, targetDir); err != nil {
			return fmt.Errorf("failed to rename collection directory: %w", err)
		}
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create collection directory: %w", err)
	}

	stored, err := s.collectionForStorage(collection, targetDir)
	if err != nil {
		return err
	}
	return writeJSONFile(filepath.Join(targetDir, collectionFileName), stored)
}

func (s *FileStore) GetCollection(id string) (*types.Collection, error) {
	collection, _, err := s.findCollectionByID(id)
	if err != nil {
		if IsCollectionLocked(err) {
			return nil, err
		}
		if errors.Is(err, errFileCollectionNotFound) {
			return nil, newCollectionNotFoundError(id)
		}
		return nil, err
	}
	return collection, nil
}

func (s *FileStore) GetCollectionByName(name string) (*types.Collection, error) {
	collection, _, err := s.findCollectionByName(name)
	if err != nil {
		if IsCollectionLocked(err) {
			return nil, err
		}
		if errors.Is(err, errFileCollectionNotFound) {
			return nil, newCollectionNotFoundError(name)
		}
		return nil, err
	}
	return collection, nil
}

func (s *FileStore) GetRawCollectionByName(name string) (*types.Collection, error) {
	collection, _, err := s.findRawCollectionByName(name)
	if err != nil {
		if errors.Is(err, errFileCollectionNotFound) {
			return nil, newCollectionNotFoundError(name)
		}
		return nil, err
	}
	return collection, nil
}

func (s *FileStore) ListCollections() ([]*types.Collection, error) {
	records, err := s.scanCollections()
	if err != nil {
		return nil, err
	}
	collections := make([]*types.Collection, 0, len(records))
	for _, record := range records {
		collections = append(collections, record.collection)
	}
	sort.SliceStable(collections, func(i, j int) bool {
		return collections[i].Name < collections[j].Name
	})
	return collections, nil
}

func (s *FileStore) DeleteCollection(id string) error {
	_, dir, err := s.findCollectionByID(id)
	if err != nil {
		if IsCollectionLocked(err) {
			return err
		}
		if errors.Is(err, errFileCollectionNotFound) {
			return newCollectionNotFoundError(id)
		}
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

func (s *FileStore) UpdateCollection(collection *types.Collection) error {
	if collection == nil {
		return fmt.Errorf("collection cannot be nil")
	}
	if collection.ID == "" {
		return fmt.Errorf("cannot update collection without ID")
	}
	existing, _, err := s.findCollectionByID(collection.ID)
	if err != nil {
		if IsCollectionLocked(err) {
			return err
		}
		if errors.Is(err, errFileCollectionNotFound) {
			return newCollectionNotFoundError(collection.ID)
		}
		return err
	}
	collection.CreatedAt = existing.CreatedAt
	if collection.UpdatedAt == 0 || collection.UpdatedAt == existing.UpdatedAt {
		collection.UpdatedAt = time.Now().Unix()
	}
	return s.SaveCollection(collection)
}

func (s *FileStore) SaveRequest(req *types.SavedRequest) error {
	if !s.Enabled() {
		return fmt.Errorf("file storage is not configured")
	}
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if req.Collection == "" {
		return fmt.Errorf("file-backed requests must belong to a collection")
	}

	_, collectionDir, err := s.findCollectionByName(req.Collection)
	if err != nil {
		if IsCollectionLocked(err) {
			return err
		}
		collection := types.NewCollection(req.Collection)
		if err := s.SaveCollection(collection); err != nil {
			return err
		}
		_, collectionDir, err = s.findCollectionByName(req.Collection)
		if err != nil {
			return err
		}
	}

	now := time.Now().Unix()
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	existing, oldPath, err := s.findRequestByID(req.ID)
	if err != nil && !errors.Is(err, errFileRequestNotFound) {
		return err
	}
	if err == nil && existing != nil && req.CreatedAt == 0 {
		req.CreatedAt = existing.CreatedAt
	}
	if req.CreatedAt == 0 {
		req.CreatedAt = now
	}
	if req.UpdatedAt == 0 {
		req.UpdatedAt = now
	}

	targetPath := filepath.Join(collectionDir, requestFileName(req))
	if err := writeJSONFile(targetPath, req); err != nil {
		return err
	}
	if oldPath != "" && oldPath != targetPath {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove old request file: %w", err)
		}
	}
	return nil
}

func (s *FileStore) GetRequest(id string) (*types.SavedRequest, error) {
	req, _, err := s.findRequestByID(id)
	if err != nil {
		if IsCollectionLocked(err) {
			return nil, err
		}
		return nil, fmt.Errorf("request not found: %s", id)
	}
	return req, nil
}

func (s *FileStore) GetRequestByName(name string) (*types.SavedRequest, error) {
	req, _, err := s.findRequestByName(name)
	if err != nil {
		if IsCollectionLocked(err) {
			return nil, err
		}
		return nil, fmt.Errorf("request not found: %s", name)
	}
	return req, nil
}

func (s *FileStore) ListRequests(opts *ListOptions) ([]*types.SavedRequest, error) {
	if !s.Enabled() {
		return []*types.SavedRequest{}, nil
	}
	if opts == nil {
		opts = &ListOptions{}
	}

	var records []collectionRecord
	if opts.Collection != "" {
		collection, dir, err := s.findCollectionByName(opts.Collection)
		if err != nil {
			if errors.Is(err, errFileCollectionNotFound) {
				return []*types.SavedRequest{}, nil
			}
			return nil, err
		}
		records = append(records, collectionRecord{collection: collection, dir: dir})
	} else {
		var err error
		records, err = s.scanCollections()
		if err != nil {
			return nil, err
		}
	}

	var requests []*types.SavedRequest
	for _, record := range records {
		if opts.Collection != "" && record.collection.Name != opts.Collection {
			continue
		}
		reqs, err := s.readRequestsInDir(record.dir)
		if err != nil {
			return nil, err
		}
		for _, req := range reqs {
			if req.Collection == "" {
				req.Collection = record.collection.Name
			}
			if requestMatchesOptions(req, opts) {
				requests = append(requests, req)
			}
		}
	}

	sortRequestsByOptions(requests, opts)
	if opts.Limit > 0 && len(requests) > opts.Limit {
		requests = requests[:opts.Limit]
	}
	return requests, nil
}

func (s *FileStore) DeleteRequest(id string) error {
	_, path, err := s.findRequestByID(id)
	if err != nil {
		if IsCollectionLocked(err) {
			return err
		}
		return fmt.Errorf("request not found: %s", id)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete request: %w", err)
	}
	return nil
}

func (s *FileStore) UpdateRequest(req *types.SavedRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if req.ID == "" {
		return fmt.Errorf("cannot update request without ID")
	}
	if _, _, err := s.findRequestByID(req.ID); err != nil {
		if IsCollectionLocked(err) {
			return err
		}
		return fmt.Errorf("request not found: %w", err)
	}
	return s.SaveRequest(req)
}

func (s *FileStore) ListFolder(path string) ([]*types.SavedRequest, error) {
	requests, err := s.ListRequests(nil)
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

func (s *FileStore) ListFolderRecursive(path string) ([]*types.SavedRequest, error) {
	requests, err := s.ListRequests(nil)
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

func (s *FileStore) DeleteFolder(path string) error {
	requests, err := s.ListFolder(path)
	if err != nil {
		return err
	}
	for _, req := range requests {
		req.Folder = ""
		req.UpdatedAt = time.Now().Unix()
		if err := s.SaveRequest(req); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStore) GetAllFolders() ([]string, error) {
	requests, err := s.ListRequests(nil)
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

func (s *FileStore) HasCollection(name string) bool {
	if !s.Enabled() {
		return false
	}
	_, _, err := s.findCollectionByName(name)
	return err == nil
}

func (s *FileStore) HasRequest(id string) bool {
	if !s.Enabled() || id == "" {
		return false
	}
	_, _, err := s.findRequestByID(id)
	return err == nil
}

func (s *FileStore) CollectionPath(name string) (string, error) {
	_, dir, err := s.findCollectionByName(name)
	if err == nil {
		return dir, nil
	}
	if !s.Enabled() {
		return "", fmt.Errorf("file storage is not configured")
	}
	return filepath.Join(s.project.CollectionsDir(), safePathComponent(name)), nil
}

func (s *FileStore) scanCollections() ([]collectionRecord, error) {
	return s.scanCollectionsForUse(true)
}

func (s *FileStore) scanCollectionsRaw() ([]collectionRecord, error) {
	return s.scanCollectionsForUse(false)
}

func (s *FileStore) scanCollectionsForUse(decrypt bool) ([]collectionRecord, error) {
	if !s.Enabled() {
		return nil, nil
	}
	entries, err := os.ReadDir(s.project.CollectionsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read collections directory: %w", err)
	}

	records := make([]collectionRecord, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(s.project.CollectionsDir(), entry.Name(), collectionFileName)
		dir := filepath.Dir(path)
		collection, err := readCollectionFile(path)
		if err != nil {
			continue
		}
		if decrypt {
			if err := s.decryptCollectionForUse(collection, dir); err != nil {
				return nil, err
			}
		}
		records = append(records, collectionRecord{
			collection: collection,
			dir:        dir,
		})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].collection.Name < records[j].collection.Name
	})
	return records, nil
}

func (s *FileStore) findRawCollectionByName(name string) (*types.Collection, string, error) {
	records, err := s.scanCollectionsRaw()
	if err != nil {
		return nil, "", err
	}
	for _, record := range records {
		if record.collection.Name == name {
			return record.collection, record.dir, nil
		}
	}
	return nil, "", errFileCollectionNotFound
}

func (s *FileStore) findRawCollectionByID(id string) (*types.Collection, string, error) {
	records, err := s.scanCollectionsRaw()
	if err != nil {
		return nil, "", err
	}
	for _, record := range records {
		if record.collection.ID == id {
			return record.collection, record.dir, nil
		}
	}
	return nil, "", errFileCollectionNotFound
}

func (s *FileStore) findCollectionByID(id string) (*types.Collection, string, error) {
	records, err := s.scanCollectionsRaw()
	if err != nil {
		return nil, "", err
	}
	for _, record := range records {
		if record.collection.ID == id {
			if err := s.decryptCollectionForUse(record.collection, record.dir); err != nil {
				return nil, "", err
			}
			return record.collection, record.dir, nil
		}
	}
	return nil, "", errFileCollectionNotFound
}

func (s *FileStore) findCollectionByName(name string) (*types.Collection, string, error) {
	records, err := s.scanCollectionsRaw()
	if err != nil {
		return nil, "", err
	}
	for _, record := range records {
		if record.collection.Name == name {
			if err := s.decryptCollectionForUse(record.collection, record.dir); err != nil {
				return nil, "", err
			}
			return record.collection, record.dir, nil
		}
	}
	return nil, "", errFileCollectionNotFound
}

func (s *FileStore) findRequestByID(id string) (*types.SavedRequest, string, error) {
	return s.findRequest(func(req *types.SavedRequest) bool {
		return req.ID == id
	})
}

func (s *FileStore) findRequestByName(name string) (*types.SavedRequest, string, error) {
	return s.findRequest(func(req *types.SavedRequest) bool {
		return req.Name == name
	})
}

func (s *FileStore) findRequest(match func(*types.SavedRequest) bool) (*types.SavedRequest, string, error) {
	records, err := s.scanCollectionsRaw()
	if err != nil {
		return nil, "", err
	}
	for _, record := range records {
		requests, err := s.readRequestRecordsInDir(record.dir)
		if err != nil {
			return nil, "", err
		}
		for _, reqRecord := range requests {
			if match(reqRecord.request) {
				if err := s.decryptCollectionForUse(record.collection, record.dir); err != nil {
					return nil, "", err
				}
				if reqRecord.request.Collection == "" {
					reqRecord.request.Collection = record.collection.Name
				}
				return reqRecord.request, reqRecord.path, nil
			}
		}
	}
	return nil, "", errFileRequestNotFound
}

func (s *FileStore) readRequestsInDir(dir string) ([]*types.SavedRequest, error) {
	records, err := s.readRequestRecordsInDir(dir)
	if err != nil {
		return nil, err
	}
	requests := make([]*types.SavedRequest, 0, len(records))
	for _, record := range records {
		requests = append(requests, record.request)
	}
	return requests, nil
}

func (s *FileStore) readRequestRecordsInDir(dir string) ([]requestRecord, error) {
	return readRequestRecordsInDir(dir)
}

func readRequestRecordsInDir(dir string) ([]requestRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection directory: %w", err)
	}
	records := make([]requestRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == collectionFileName || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		req, err := readRequestFile(path)
		if err != nil {
			continue
		}
		records = append(records, requestRecord{request: req, path: path})
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].request.Name < records[j].request.Name
	})
	return records, nil
}

func readCollectionFile(path string) (*types.Collection, error) {
	var collection types.Collection
	if err := readJSONFile(path, &collection); err != nil {
		return nil, err
	}
	if collection.Variables == nil {
		collection.Variables = make(map[string]string)
	}
	if collection.SecretKeys == nil {
		collection.SecretKeys = make(map[string]bool)
	}
	return &collection, nil
}

func readRequestFile(path string) (*types.SavedRequest, error) {
	var req types.SavedRequest
	if err := readJSONFile(path, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func readJSONFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(value)
	closeErr := tmp.Close()
	if encodeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode JSON: %w", encodeErr)
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", closeErr)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace JSON file: %w", err)
	}
	return nil
}

func requestFileName(req *types.SavedRequest) string {
	name := "request"
	if req != nil && req.Name != "" {
		name = req.Name
	}
	id := "new"
	if req != nil && req.ID != "" {
		id = req.ID
	}
	return safePathComponent(name) + "--" + safePathComponent(id) + ".json"
}

func safePathComponent(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "item"
	}
	if isSimplePathComponent(name) {
		return name
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(builder.String(), "-.")
	if slug == "" {
		slug = "item"
	}
	sum := sha256.Sum256([]byte(name))
	return slug + "--" + hex.EncodeToString(sum[:])[:8]
}

func isSimplePathComponent(name string) bool {
	if name == "." || name == ".." || strings.HasPrefix(name, ".") {
		return false
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func requestMatchesOptions(req *types.SavedRequest, opts *ListOptions) bool {
	if opts == nil {
		return true
	}
	if opts.Collection != "" && req.Collection != opts.Collection {
		return false
	}
	if opts.Tag != "" && !stringSliceContains(req.Tags, opts.Tag) {
		return false
	}
	if opts.Pattern != "" && !strings.Contains(req.Name, opts.Pattern) && !strings.Contains(req.URL, opts.Pattern) {
		return false
	}
	return true
}

func sortRequestsByOptions(requests []*types.SavedRequest, opts *ListOptions) {
	sortKey := ""
	if opts != nil {
		sortKey = opts.Sort
	}
	sort.SliceStable(requests, func(i, j int) bool {
		switch sortKey {
		case "collection":
			if requests[i].Collection != requests[j].Collection {
				return requests[i].Collection < requests[j].Collection
			}
			return requests[i].Name < requests[j].Name
		case "name":
			return requests[i].Name < requests[j].Name
		case "updated", "":
			if requests[i].UpdatedAt != requests[j].UpdatedAt {
				return requests[i].UpdatedAt > requests[j].UpdatedAt
			}
			return requests[i].Name < requests[j].Name
		default:
			return requests[i].Name < requests[j].Name
		}
	})
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var _ CollectionStore = (*FileStore)(nil)
