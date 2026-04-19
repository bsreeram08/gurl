package env

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

type EnvStorage struct {
	db     *storage.LMDB
	dbPath string
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
