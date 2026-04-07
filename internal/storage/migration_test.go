package storage

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

// TestSchemaVersionNewDB tests that a fresh DB gets version 1 written on first Open()
func TestSchemaVersionNewDB(t *testing.T) {
	// Create temp dir for test DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_new.db")

	// Create LMDB instance pointing to temp path
	db := &LMDB{dbPath: dbPath}

	// Open the DB
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Read the schema version key directly
	data, err := db.db.Get([]byte("schema_version"), nil)
	if err != nil {
		t.Fatalf("schema_version key not found after Open: %v", err)
	}

	var version int
	if err := json.Unmarshal(data, &version); err != nil {
		t.Fatalf("failed to unmarshal schema version: %v", err)
	}

	if version != 1 {
		t.Errorf("expected schema version 1, got %d", version)
	}
}

// TestSchemaVersionLegacyDB tests that a DB without version key migrates to version 1 on Open()
func TestSchemaVersionLegacyDB(t *testing.T) {
	// Create temp dir for test DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_legacy.db")

	// Create a "legacy" DB (version 0 / no version key)
	legacyDB, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatalf("failed to create legacy DB: %v", err)
	}

	// Write some legacy data
	legacyDB.Put([]byte("request:legacy-1"), []byte(`{"name":"legacy"}`), nil)
	legacyDB.Close()

	// Now open with our LMDB which should trigger migration
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// Verify schema version is now 1
	data, err := db.db.Get([]byte("schema_version"), nil)
	if err != nil {
		t.Fatalf("schema_version key not found after migration: %v", err)
	}

	var version int
	if err := json.Unmarshal(data, &version); err != nil {
		t.Fatalf("failed to unmarshal schema version: %v", err)
	}

	if version != 1 {
		t.Errorf("expected schema version 1 after migration, got %d", version)
	}

	// Verify legacy data still exists
	_, err = db.GetRequest("legacy-1")
	if err != nil {
		t.Errorf("legacy data should still exist: %v", err)
	}
}

// TestMigrateIfNeeded tests that migration runs when version < current
func TestMigrateIfNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_migrate.db")

	// Create DB with version 0 (below current version 1)
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}

	// Manually set version to 0
	versionData, _ := json.Marshal(0)
	db.db.Put([]byte("schema_version"), versionData, nil)
	db.Close()

	// Re-open and check MigrateIfNeeded runs
	db2 := &LMDB{dbPath: dbPath}
	if err := db2.Open(); err != nil {
		t.Fatalf("failed to re-open DB: %v", err)
	}
	defer db2.Close()

	// Verify version is now 1
	data, err := db2.db.Get([]byte("schema_version"), nil)
	if err != nil {
		t.Fatalf("schema_version key not found: %v", err)
	}

	var version int
	if err := json.Unmarshal(data, &version); err != nil {
		t.Fatalf("failed to unmarshal schema version: %v", err)
	}

	if version != 1 {
		t.Errorf("expected schema version 1 after migration, got %d", version)
	}
}

// TestGetSchemaVersion tests the GetSchemaVersion method
func TestGetSchemaVersion(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_getversion.db")

	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	// New DB should have version 1 after Open
	version, err := db.GetSchemaVersion()
	if err != nil {
		t.Fatalf("GetSchemaVersion failed: %v", err)
	}
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}
}

func TestSchemaVersionAfterOpen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_version_after_open.db")

	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	version, err := db.GetSchemaVersion()
	if err != nil {
		t.Fatalf("GetSchemaVersion failed: %v", err)
	}
	if version != 1 {
		t.Errorf("expected version 1 after Open(), got %d", version)
	}
}
