package storage

import "github.com/sreeram/gurl/pkg/types"

// LazyDB opens and closes the underlying LevelDB handle for each operation.
// This avoids holding a process-wide lock while interactive commands sit idle.
type LazyDB struct {
	dbPath string
}

func NewLazyDB() (*LazyDB, error) {
	dbPath, err := resolveDBPath()
	if err != nil {
		return nil, err
	}

	return &LazyDB{dbPath: dbPath}, nil
}

func NewLazyDBWithPath(dbPath string) *LazyDB {
	return &LazyDB{dbPath: dbPath}
}

func (db *LazyDB) Path() string {
	if db == nil {
		return ""
	}
	return db.dbPath
}

func (db *LazyDB) Open() error  { return nil }
func (db *LazyDB) Close() error { return nil }

func (db *LazyDB) withDB(fn func(*LMDB) error) error {
	lmdb := NewLMDBWithPath(db.dbPath)
	if err := lmdb.Open(); err != nil {
		return err
	}
	defer lmdb.Close()

	return fn(lmdb)
}

func (db *LazyDB) SaveRequest(req *types.SavedRequest) error {
	return db.withDB(func(lmdb *LMDB) error {
		return lmdb.SaveRequest(req)
	})
}

func (db *LazyDB) GetRequest(id string) (*types.SavedRequest, error) {
	var result *types.SavedRequest
	err := db.withDB(func(lmdb *LMDB) error {
		var innerErr error
		result, innerErr = lmdb.GetRequest(id)
		return innerErr
	})
	return result, err
}

func (db *LazyDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	var result *types.SavedRequest
	err := db.withDB(func(lmdb *LMDB) error {
		var innerErr error
		result, innerErr = lmdb.GetRequestByName(name)
		return innerErr
	})
	return result, err
}

func (db *LazyDB) ListRequests(opts *ListOptions) ([]*types.SavedRequest, error) {
	var result []*types.SavedRequest
	err := db.withDB(func(lmdb *LMDB) error {
		var innerErr error
		result, innerErr = lmdb.ListRequests(opts)
		return innerErr
	})
	return result, err
}

func (db *LazyDB) DeleteRequest(id string) error {
	return db.withDB(func(lmdb *LMDB) error {
		return lmdb.DeleteRequest(id)
	})
}

func (db *LazyDB) UpdateRequest(req *types.SavedRequest) error {
	return db.withDB(func(lmdb *LMDB) error {
		return lmdb.UpdateRequest(req)
	})
}

func (db *LazyDB) SaveHistory(history *types.ExecutionHistory) error {
	return db.withDB(func(lmdb *LMDB) error {
		return lmdb.SaveHistory(history)
	})
}

func (db *LazyDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	var result []*types.ExecutionHistory
	err := db.withDB(func(lmdb *LMDB) error {
		var innerErr error
		result, innerErr = lmdb.GetHistory(requestID, limit)
		return innerErr
	})
	return result, err
}

func (db *LazyDB) ListFolder(path string) ([]*types.SavedRequest, error) {
	var result []*types.SavedRequest
	err := db.withDB(func(lmdb *LMDB) error {
		var innerErr error
		result, innerErr = lmdb.ListFolder(path)
		return innerErr
	})
	return result, err
}

func (db *LazyDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) {
	var result []*types.SavedRequest
	err := db.withDB(func(lmdb *LMDB) error {
		var innerErr error
		result, innerErr = lmdb.ListFolderRecursive(path)
		return innerErr
	})
	return result, err
}

func (db *LazyDB) DeleteFolder(path string) error {
	return db.withDB(func(lmdb *LMDB) error {
		return lmdb.DeleteFolder(path)
	})
}

func (db *LazyDB) GetAllFolders() ([]string, error) {
	var result []string
	err := db.withDB(func(lmdb *LMDB) error {
		var innerErr error
		result, innerErr = lmdb.GetAllFolders()
		return innerErr
	})
	return result, err
}

var _ DB = (*LazyDB)(nil)
