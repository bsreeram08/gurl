package env

import (
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/internal/storage"
)

func TestScopingHierarchy(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_scoping.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	globalEnv := &Environment{
		ID:        "env-global",
		Name:      "Global",
		Variables: map[string]string{"GLOBAL_VAR": "global-value", "SHARED": "from-global"},
		ParentID:  "",
	}
	envStore.SaveEnv(globalEnv)

	collectionEnv := &Environment{
		ID:        "env-collection",
		Name:      "Collection",
		Variables: map[string]string{"COLLECTION_VAR": "collection-value", "SHARED": "from-collection"},
		ParentID:  "env-global",
	}
	envStore.SaveEnv(collectionEnv)

	folderEnv := &Environment{
		ID:        "env-folder",
		Name:      "Folder",
		Variables: map[string]string{"FOLDER_VAR": "folder-value", "SHARED": "from-folder"},
		ParentID:  "env-collection",
	}
	envStore.SaveEnv(folderEnv)

	requestEnv := &Environment{
		ID:        "env-request",
		Name:      "Request",
		Variables: map[string]string{"REQUEST_VAR": "request-value", "SHARED": "from-request"},
		ParentID:  "env-folder",
	}
	envStore.SaveEnv(requestEnv)

	scoper := NewScoper(envStore)

	vars := scoper.GetScopedVariables("env-request")

	if vars["REQUEST_VAR"] != "request-value" {
		t.Errorf("expected REQUEST_VAR 'request-value', got '%s'", vars["REQUEST_VAR"])
	}
	if vars["FOLDER_VAR"] != "folder-value" {
		t.Errorf("expected FOLDER_VAR 'folder-value', got '%s'", vars["FOLDER_VAR"])
	}
	if vars["COLLECTION_VAR"] != "collection-value" {
		t.Errorf("expected COLLECTION_VAR 'collection-value', got '%s'", vars["COLLECTION_VAR"])
	}
	if vars["GLOBAL_VAR"] != "global-value" {
		t.Errorf("expected GLOBAL_VAR 'global-value', got '%s'", vars["GLOBAL_VAR"])
	}
	if vars["SHARED"] != "from-request" {
		t.Errorf("expected SHARED to be 'from-request' (most specific), got '%s'", vars["SHARED"])
	}
}

func TestScopingNoParent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_scoping_noparent.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		ID:        "env-isolated",
		Name:      "Isolated",
		Variables: map[string]string{"ISOLATED_VAR": "isolated-value"},
		ParentID:  "",
	}
	envStore.SaveEnv(env)

	scoper := NewScoper(envStore)
	vars := scoper.GetScopedVariables("env-isolated")

	if vars["ISOLATED_VAR"] != "isolated-value" {
		t.Errorf("expected ISOLATED_VAR 'isolated-value', got '%s'", vars["ISOLATED_VAR"])
	}

	_, hasGlobal := vars["GLOBAL_VAR"]
	if hasGlobal {
		t.Error("expected no GLOBAL_VAR for isolated environment")
	}
}

func TestScopingEmptyEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_scoping_empty.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	scoper := NewScoper(envStore)
	vars := scoper.GetScopedVariables("non-existent-env")

	if len(vars) != 0 {
		t.Errorf("expected empty variables for non-existent env, got %d vars", len(vars))
	}
}

func TestScopingPartialChain(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_scoping_partial.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	parentEnv := &Environment{
		ID:        "env-parent-partial",
		Name:      "Parent",
		Variables: map[string]string{"PARENT_VAR": "parent-value"},
		ParentID:  "",
	}
	envStore.SaveEnv(parentEnv)

	childEnv := &Environment{
		ID:        "env-child-partial",
		Name:      "Child",
		Variables: map[string]string{"CHILD_VAR": "child-value"},
		ParentID:  "env-parent-partial",
	}
	envStore.SaveEnv(childEnv)

	scoper := NewScoper(envStore)
	vars := scoper.GetScopedVariables("env-child-partial")

	if vars["CHILD_VAR"] != "child-value" {
		t.Errorf("expected CHILD_VAR 'child-value', got '%s'", vars["CHILD_VAR"])
	}
	if vars["PARENT_VAR"] != "parent-value" {
		t.Errorf("expected PARENT_VAR 'parent-value', got '%s'", vars["PARENT_VAR"])
	}
}

func TestScopingOrder(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_scoping_order.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	globalEnv := &Environment{
		ID:        "env-order-global",
		Name:      "Global",
		Variables: map[string]string{"VAR": "global"},
		ParentID:  "",
	}
	envStore.SaveEnv(globalEnv)

	env := &Environment{
		ID:        "env-order-test",
		Name:      "Order Test",
		Variables: map[string]string{"VAR": "local"},
		ParentID:  "env-order-global",
	}
	envStore.SaveEnv(env)

	scoper := NewScoper(envStore)
	vars := scoper.GetScopedVariables("env-order-test")

	if vars["VAR"] != "local" {
		t.Errorf("expected VAR to be 'local' (most specific wins), got '%s'", vars["VAR"])
	}
}
