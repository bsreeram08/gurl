package storage

import (
	"encoding/json"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
)

const schemaVersionKey = "schema_version"

const currentSchemaVersion = 3

var migrations = map[int]func(db *leveldb.DB) error{
	1: migrateToV1,
	2: migrateToV2,
	3: migrateToV3,
}

func migrateToV1(db *leveldb.DB) error {
	return nil
}

func migrateToV2(db *leveldb.DB) error {
	return nil
}

func migrateToV3(db *leveldb.DB) error {
	return nil
}

func (db *LMDB) GetSchemaVersion() (int, error) {
	data, err := db.DB.Get([]byte(schemaVersionKey), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}

	var version int
	if err := json.Unmarshal(data, &version); err != nil {
		return 0, fmt.Errorf("failed to unmarshal schema version: %w", err)
	}

	return version, nil
}

func (db *LMDB) setSchemaVersion(version int) error {
	data, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal schema version: %w", err)
	}

	if err := db.DB.Put([]byte(schemaVersionKey), data, nil); err != nil {
		return fmt.Errorf("failed to write schema version: %w", err)
	}

	return nil
}

func (db *LMDB) MigrateIfNeeded() error {
	version, err := db.GetSchemaVersion()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	if version >= currentSchemaVersion {
		return nil
	}

	for v := version + 1; v <= currentSchemaVersion; v++ {
		migrateFn, ok := migrations[v]
		if !ok {
			return fmt.Errorf("no migration function for version %d", v)
		}

		if err := migrateFn(db.DB); err != nil {
			return fmt.Errorf("migration to version %d failed: %w", v, err)
		}

		if err := db.setSchemaVersion(v); err != nil {
			return fmt.Errorf("failed to persist schema version %d: %w", v, err)
		}
	}

	return nil
}
