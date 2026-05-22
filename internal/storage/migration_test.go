package storage

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
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
	data, err := db.DB.Get([]byte("schema_version"), nil)
	if err != nil {
		t.Fatalf("schema_version key not found after Open: %v", err)
	}

	var version int
	if err := json.Unmarshal(data, &version); err != nil {
		t.Fatalf("failed to unmarshal schema version: %v", err)
	}

	if version != 3 {
		t.Errorf("expected schema version 3, got %d", version)
	}
}

// TestSchemaVersionLegacyDB tests that a DB without version key migrates to version 3 on Open()
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
	data, err := db.DB.Get([]byte("schema_version"), nil)
	if err != nil {
		t.Fatalf("schema_version key not found after migration: %v", err)
	}

	var version int
	if err := json.Unmarshal(data, &version); err != nil {
		t.Fatalf("failed to unmarshal schema version: %v", err)
	}

	if version != 3 {
		t.Errorf("expected schema version 3 after migration, got %d", version)
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

	// Create DB with version 0 (below current version 3)
	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}

	// Manually set version to 0
	versionData, _ := json.Marshal(0)
	db.DB.Put([]byte("schema_version"), versionData, nil)
	db.Close()

	// Re-open and check MigrateIfNeeded runs
	db2 := &LMDB{dbPath: dbPath}
	if err := db2.Open(); err != nil {
		t.Fatalf("failed to re-open DB: %v", err)
	}
	defer db2.Close()

	// Verify version is now 3
	data, err := db2.DB.Get([]byte("schema_version"), nil)
	if err != nil {
		t.Fatalf("schema_version key not found: %v", err)
	}

	var version int
	if err := json.Unmarshal(data, &version); err != nil {
		t.Fatalf("failed to unmarshal schema version: %v", err)
	}

	if version != 3 {
		t.Errorf("expected schema version 3 after migration, got %d", version)
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

	// New DB should have version 3 after Open
	version, err := db.GetSchemaVersion()
	if err != nil {
		t.Fatalf("GetSchemaVersion failed: %v", err)
	}
	if version != 3 {
		t.Errorf("expected version 3, got %d", version)
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
	if version != 3 {
		t.Errorf("expected version 3 after Open(), got %d", version)
	}
}

func TestSavedRequestFlowMetadataJSONCompatibility(t *testing.T) {
	oldJSON := []byte(`{"id":"legacy-1","name":"legacy","url":"https://example.com","method":"GET","headers":[],"output_format":"auto","created_at":1,"updated_at":1}`)

	var legacy types.SavedRequest
	if err := json.Unmarshal(oldJSON, &legacy); err != nil {
		t.Fatalf("old saved request JSON should still unmarshal: %v", err)
	}
	if legacy.PreScript != "" || legacy.PostScript != "" || legacy.RunIf != "" || len(legacy.Extracts) != 0 {
		t.Fatalf("expected absent flow metadata to use zero values, got pre=%q post=%q run_if=%q extracts=%v", legacy.PreScript, legacy.PostScript, legacy.RunIf, legacy.Extracts)
	}

	req := types.SavedRequest{
		ID:           "flow-1",
		Name:         "flow",
		URL:          "https://example.com",
		Method:       "POST",
		Headers:      []types.Header{},
		OutputFormat: "auto",
		PreScript:    "gurl.setVar('tenant', 'acme')",
		PostScript:   "gurl.setVar('seen', 'yes')",
		RunIf:        "tenant != ''",
		Extracts: []types.Extract{
			{Name: "token", Source: "jsonpath:$.token"},
			{Name: "requestId", Source: "header:X-Request-Id"},
		},
		CreatedAt: 1,
		UpdatedAt: 1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal saved request with flow metadata: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal saved request JSON map: %v", err)
	}
	for _, key := range []string{"pre_script", "post_script", "run_if", "extracts"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected JSON key %q in %s", key, string(data))
		}
	}
	if _, ok := decoded["extractions"]; ok {
		t.Fatalf("must not create duplicate extractions field in %s", string(data))
	}

	var roundTrip types.SavedRequest
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("round-trip saved request JSON: %v", err)
	}
	if roundTrip.PreScript != req.PreScript || roundTrip.PostScript != req.PostScript || roundTrip.RunIf != req.RunIf {
		t.Fatalf("flow metadata did not round-trip: got pre=%q post=%q run_if=%q", roundTrip.PreScript, roundTrip.PostScript, roundTrip.RunIf)
	}
	if len(roundTrip.Extracts) != 2 || roundTrip.Extracts[0].Name != "token" || roundTrip.Extracts[0].Source != "jsonpath:$.token" {
		t.Fatalf("extract metadata did not round-trip: %#v", roundTrip.Extracts)
	}
}

func TestSavedRequestFlowMetadataStorageRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_flow_metadata.db")

	db := &LMDB{dbPath: dbPath}
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	req := &types.SavedRequest{
		ID:           "flow-1",
		Name:         "flow",
		URL:          "https://example.com",
		Method:       "GET",
		Headers:      []types.Header{},
		OutputFormat: "auto",
		PreScript:    "gurl.setVar('tenant', 'acme')",
		PostScript:   "gurl.setVar('seen', 'yes')",
		RunIf:        "tenant == 'acme'",
		Extracts: []types.Extract{
			{Name: "token", Source: "jsonpath:$.token"},
		},
		CreatedAt: 1,
		UpdatedAt: 1,
	}

	if err := db.SaveRequest(req); err != nil {
		t.Fatalf("save request: %v", err)
	}

	loaded, err := db.GetRequestByName("flow")
	if err != nil {
		t.Fatalf("load request by name: %v", err)
	}
	if loaded.PreScript != req.PreScript || loaded.PostScript != req.PostScript || loaded.RunIf != req.RunIf {
		t.Fatalf("loaded metadata mismatch: got pre=%q post=%q run_if=%q", loaded.PreScript, loaded.PostScript, loaded.RunIf)
	}
	if len(loaded.Extracts) != 1 || loaded.Extracts[0].Name != "token" || loaded.Extracts[0].Source != "jsonpath:$.token" {
		t.Fatalf("loaded extracts mismatch: %#v", loaded.Extracts)
	}

	loaded.RunIf = "token != ''"
	loaded.Extracts = append(loaded.Extracts, types.Extract{Name: "requestId", Source: "header:X-Request-Id"})
	if err := db.UpdateRequest(loaded); err != nil {
		t.Fatalf("update request: %v", err)
	}

	updated, err := db.GetRequestByName("flow")
	if err != nil {
		t.Fatalf("reload request by name: %v", err)
	}
	if updated.RunIf != "token != ''" {
		t.Fatalf("updated run_if mismatch: %q", updated.RunIf)
	}
	if len(updated.Extracts) != 2 || updated.Extracts[1].Name != "requestId" || updated.Extracts[1].Source != "header:X-Request-Id" {
		t.Fatalf("updated extracts mismatch: %#v", updated.Extracts)
	}
}
