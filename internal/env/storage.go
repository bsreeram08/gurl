package env

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

type EnvStorage struct {
	db     *storage.LMDB
	dbPath string
	files  *FileEnvStore
}

func NewEnvStorage(db *storage.LMDB) *EnvStorage {
	dbPath := ""
	if db != nil {
		dbPath = db.Path()
	}
	return &EnvStorage{db: db, dbPath: dbPath}
}

func NewEnvStorageWithPath(dbPath string) *EnvStorage {
	return &EnvStorage{dbPath: dbPath}
}

func NewEnvStorageWithPathAndProject(dbPath string, proj *project.Project) *EnvStorage {
	return &EnvStorage{dbPath: dbPath, files: NewFileEnvStore(proj)}
}

func (s *EnvStorage) hasFileStore() bool {
	return s != nil && s.files != nil && s.files.Enabled()
}

func (s *EnvStorage) dbHasEnv(env *Environment) bool {
	if env == nil || (env.ID == "" && env.Name == "") || s.dbPath == "" && (s.db == nil || s.db.DB == nil) {
		return false
	}
	db, closeDB, err := s.openDB()
	if err != nil {
		return false
	}
	defer closeDB()

	if env.ID != "" {
		dbKey := fmt.Sprintf("env:%s", env.ID)
		if _, err := db.DB.Get([]byte(dbKey), nil); err == nil {
			return true
		}
	}
	if env.Name != "" {
		nameKey := fmt.Sprintf("idx:env:name:%s", env.Name)
		if _, err := db.DB.Get([]byte(nameKey), nil); err == nil {
			return true
		}
	}
	return false
}

func (s *EnvStorage) openDB() (*storage.LMDB, func() error, error) {
	if s.db != nil && s.db.DB != nil {
		return s.db, func() error { return nil }, nil
	}
	if s.dbPath == "" {
		return nil, nil, fmt.Errorf("environment storage database is not configured")
	}

	db := storage.NewLMDBWithPath(s.dbPath)
	if err := db.Open(); err != nil {
		return nil, nil, err
	}

	return db, db.Close, nil
}

func (s *EnvStorage) SaveEnv(env *Environment) error {
	if env == nil {
		return fmt.Errorf("environment cannot be nil")
	}
	if s.hasFileStore() && (s.files.HasEnvID(env.ID) || s.files.HasEnvName(env.Name) || !s.dbHasEnv(env)) {
		return s.files.SaveEnv(env)
	}

	db, closeDB, err := s.openDB()
	if err != nil {
		return err
	}
	defer closeDB()

	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	key, err := GetOrCreateMachineKey()
	if err != nil {
		return fmt.Errorf("failed to get encryption key: %w", err)
	}

	for k, isSecret := range env.SecretKeys {
		if isSecret {
			encrypted, encErr := EncryptSecret(key, env.Variables[k])
			if encErr != nil {
				return fmt.Errorf("failed to encrypt secret %s: %w", k, encErr)
			}
			env.Variables[k] = encrypted
		}
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal environment: %w", err)
	}

	dbKey := fmt.Sprintf("env:%s", env.ID)
	if err := db.DB.Put([]byte(dbKey), data, nil); err != nil {
		return fmt.Errorf("failed to save environment: %w", err)
	}

	nameKey := fmt.Sprintf("idx:env:name:%s", env.Name)
	if err := db.DB.Put([]byte(nameKey), []byte(env.ID), nil); err != nil {
		return fmt.Errorf("failed to update env name index: %w", err)
	}

	return nil
}

func (s *EnvStorage) GetEnv(id string) (*Environment, error) {
	if s.hasFileStore() {
		if env, err := s.files.GetEnv(id); err == nil {
			return env, nil
		}
	}

	db, closeDB, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer closeDB()

	return s.getEnvWithDB(db, id)
}

func (s *EnvStorage) getEnvWithDB(db *storage.LMDB, id string) (*Environment, error) {
	dbKey := fmt.Sprintf("env:%s", id)
	data, err := db.DB.Get([]byte(dbKey), nil)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
	}

	key, err := GetOrCreateMachineKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	for k, isSecret := range env.SecretKeys {
		if isSecret && IsEncryptedValue(env.Variables[k]) {
			decrypted, decErr := DecryptSecret(key, env.Variables[k])
			if decErr != nil {
				return nil, fmt.Errorf("failed to decrypt secret %s: %w", k, decErr)
			}
			env.Variables[k] = decrypted
		}
	}

	return &env, nil
}

func (s *EnvStorage) DeleteEnv(id string) error {
	if s.hasFileStore() && s.files.HasEnvID(id) {
		return s.files.DeleteEnv(id)
	}

	db, closeDB, err := s.openDB()
	if err != nil {
		return err
	}
	defer closeDB()

	env, err := s.getEnvWithDB(db, id)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("env:%s", id)
	if err := db.DB.Delete([]byte(key), nil); err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	nameKey := fmt.Sprintf("idx:env:name:%s", env.Name)
	db.DB.Delete([]byte(nameKey), nil)

	return nil
}

func (s *EnvStorage) ListEnvs() ([]*Environment, error) {
	if s.hasFileStore() {
		byName := make(map[string]*Environment)
		fileEnvs, err := s.files.ListEnvs()
		if err != nil {
			return nil, err
		}
		for _, env := range fileEnvs {
			byName[env.Name] = env
		}
		if s.hasDBConfig() {
			dbEnvs, err := s.listDBEnvs()
			if err != nil {
				return nil, err
			}
			for _, env := range dbEnvs {
				if _, exists := byName[env.Name]; !exists {
					byName[env.Name] = env
				}
			}
		}
		envs := make([]*Environment, 0, len(byName))
		for _, env := range byName {
			envs = append(envs, env)
		}
		sortEnvs(envs)
		return envs, nil
	}

	return s.listDBEnvs()
}

func (s *EnvStorage) listDBEnvs() ([]*Environment, error) {
	db, closeDB, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer closeDB()

	var envs []*Environment

	iter := db.DB.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := string(iter.Key())
		if len(key) >= 4 && key[:4] == "env:" {
			data := iter.Value()
			var env Environment
			if err := json.Unmarshal(data, &env); err != nil {
				continue
			}
			envs = append(envs, &env)
		}
	}

	return envs, nil
}

func (s *EnvStorage) GetEnvByName(name string) (*Environment, error) {
	if s.hasFileStore() {
		if env, err := s.files.GetEnvByName(name); err == nil {
			return env, nil
		}
	}

	db, closeDB, err := s.openDB()
	if err != nil {
		return nil, err
	}
	defer closeDB()

	nameKey := fmt.Sprintf("idx:env:name:%s", name)
	idData, err := db.DB.Get([]byte(nameKey), nil)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", name)
	}

	return s.getEnvWithDB(db, string(idData))
}

func (s *EnvStorage) GetActiveEnv() (string, error) {
	if s.hasFileStore() {
		active, err := s.files.GetActiveEnv()
		if err != nil {
			return "", err
		}
		if active != "" {
			return active, nil
		}
	}
	if !s.hasDBConfig() {
		return "", nil
	}

	db, closeDB, err := s.openDB()
	if err != nil {
		return "", err
	}
	defer closeDB()

	data, err := db.DB.Get([]byte("cfg:activeEnv"), nil)
	if err == leveldb.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get active env: %w", err)
	}
	return string(data), nil
}

func (s *EnvStorage) SetActiveEnv(name string) error {
	if s.hasFileStore() {
		return s.files.SetActiveEnv(name)
	}

	db, closeDB, err := s.openDB()
	if err != nil {
		return err
	}
	defer closeDB()

	if name == "" {
		if err := db.DB.Delete([]byte("cfg:activeEnv"), nil); err != nil && err != leveldb.ErrNotFound {
			return fmt.Errorf("failed to clear active env: %w", err)
		}
		return nil
	}
	if err := db.DB.Put([]byte("cfg:activeEnv"), []byte(name), nil); err != nil {
		return fmt.Errorf("failed to set active env: %w", err)
	}
	return nil
}

func sortEnvs(envs []*Environment) {
	sort.SliceStable(envs, func(i, j int) bool {
		return envs[i].Name < envs[j].Name
	})
}

func (s *EnvStorage) hasDBConfig() bool {
	return s != nil && (s.dbPath != "" || (s.db != nil && s.db.DB != nil))
}
