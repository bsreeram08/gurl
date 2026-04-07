package env

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

type EnvStorage struct {
	db *storage.LMDB
}

func NewEnvStorage(db *storage.LMDB) *EnvStorage {
	return &EnvStorage{db: db}
}

func (s *EnvStorage) SaveEnv(env *Environment) error {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal environment: %w", err)
	}

	key := fmt.Sprintf("env:%s", env.ID)
	if err := s.db.DB.Put([]byte(key), data, nil); err != nil {
		return fmt.Errorf("failed to save environment: %w", err)
	}

	nameKey := fmt.Sprintf("idx:env:name:%s", env.Name)
	if err := s.db.DB.Put([]byte(nameKey), []byte(env.ID), nil); err != nil {
		return fmt.Errorf("failed to update env name index: %w", err)
	}

	return nil
}

func (s *EnvStorage) GetEnv(id string) (*Environment, error) {
	key := fmt.Sprintf("env:%s", id)
	data, err := s.db.DB.Get([]byte(key), nil)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
	}

	return &env, nil
}

func (s *EnvStorage) DeleteEnv(id string) error {
	env, err := s.GetEnv(id)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("env:%s", id)
	if err := s.db.DB.Delete([]byte(key), nil); err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	nameKey := fmt.Sprintf("idx:env:name:%s", env.Name)
	s.db.DB.Delete([]byte(nameKey), nil)

	return nil
}

func (s *EnvStorage) ListEnvs() ([]*Environment, error) {
	var envs []*Environment

	iter := s.db.DB.NewIterator(nil, nil)
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
	nameKey := fmt.Sprintf("idx:env:name:%s", name)
	idData, err := s.db.DB.Get([]byte(nameKey), nil)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", name)
	}

	return s.GetEnv(string(idData))
}

func (s *EnvStorage) GetActiveEnv() (string, error) {
	data, err := s.db.DB.Get([]byte("cfg:activeEnv"), nil)
	if err == leveldb.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get active env: %w", err)
	}
	return string(data), nil
}

func (s *EnvStorage) SetActiveEnv(name string) error {
	if name == "" {
		if err := s.db.DB.Delete([]byte("cfg:activeEnv"), nil); err != nil && err != leveldb.ErrNotFound {
			return fmt.Errorf("failed to clear active env: %w", err)
		}
		return nil
	}
	if err := s.db.DB.Put([]byte("cfg:activeEnv"), []byte(name), nil); err != nil {
		return fmt.Errorf("failed to set active env: %w", err)
	}
	return nil
}
