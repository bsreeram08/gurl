package env

import (
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/internal/storage"
)

func TestEnvStorageSaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_env_storage.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		ID:        "env-test-1",
		Name:      "Test Environment",
		Variables: map[string]string{"BASE_URL": "https://test.com"},
		ParentID:  "",
	}

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	retrieved, err := envStore.GetEnv("env-test-1")
	if err != nil {
		t.Fatalf("failed to get environment: %v", err)
	}

	if retrieved.ID != env.ID {
		t.Errorf("expected ID '%s', got '%s'", env.ID, retrieved.ID)
	}
	if retrieved.Name != env.Name {
		t.Errorf("expected Name '%s', got '%s'", env.Name, retrieved.Name)
	}
	if retrieved.Variables["BASE_URL"] != "https://test.com" {
		t.Errorf("expected BASE_URL 'https://test.com', got '%s'", retrieved.Variables["BASE_URL"])
	}
}

func TestEnvStorageDelete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_env_delete.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		ID:        "env-to-delete",
		Name:      "Delete Me",
		Variables: map[string]string{"KEY": "value"},
		ParentID:  "",
	}

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	if err := envStore.DeleteEnv("env-to-delete"); err != nil {
		t.Fatalf("failed to delete environment: %v", err)
	}

	_, err := envStore.GetEnv("env-to-delete")
	if err == nil {
		t.Error("expected error when getting deleted environment, got nil")
	}
}

func TestEnvStorageList(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_env_list.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env1 := &Environment{ID: "env-1", Name: "Environment 1", Variables: map[string]string{"A": "1"}, ParentID: ""}
	env2 := &Environment{ID: "env-2", Name: "Environment 2", Variables: map[string]string{"B": "2"}, ParentID: ""}
	env3 := &Environment{ID: "env-3", Name: "Environment 3", Variables: map[string]string{"C": "3"}, ParentID: ""}

	for _, env := range []*Environment{env1, env2, env3} {
		if err := envStore.SaveEnv(env); err != nil {
			t.Fatalf("failed to save environment %s: %v", env.ID, err)
		}
	}

	envs, err := envStore.ListEnvs()
	if err != nil {
		t.Fatalf("failed to list environments: %v", err)
	}

	if len(envs) != 3 {
		t.Errorf("expected 3 environments, got %d", len(envs))
	}
}

func TestEnvStorageUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_env_update.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		ID:        "env-update-test",
		Name:      "Original Name",
		Variables: map[string]string{"ORIGINAL": "value"},
		ParentID:  "",
	}

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	env.Name = "Updated Name"
	env.Variables["NEW_KEY"] = "new_value"

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to update environment: %v", err)
	}

	retrieved, err := envStore.GetEnv("env-update-test")
	if err != nil {
		t.Fatalf("failed to get updated environment: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("expected Name 'Updated Name', got '%s'", retrieved.Name)
	}
	if retrieved.Variables["ORIGINAL"] != "value" {
		t.Errorf("expected ORIGINAL variable to still exist")
	}
	if retrieved.Variables["NEW_KEY"] != "new_value" {
		t.Errorf("expected NEW_KEY variable to exist with value 'new_value'")
	}
}

func TestEnvStorageGetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_env_nonexistent.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	_, err := envStore.GetEnv("non-existent-env")
	if err == nil {
		t.Error("expected error when getting non-existent environment, got nil")
	}
}

func TestEnvStorageSaveWithoutID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_env_no_id.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		Name:      "No ID Environment",
		Variables: map[string]string{"KEY": "value"},
		ParentID:  "",
	}

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to save environment without ID: %v", err)
	}

	if env.ID == "" {
		t.Error("expected ID to be auto-generated")
	}

	retrieved, err := envStore.GetEnv(env.ID)
	if err != nil {
		t.Fatalf("failed to get environment by auto-generated ID: %v", err)
	}

	if retrieved.Name != "No ID Environment" {
		t.Errorf("expected Name 'No ID Environment', got '%s'", retrieved.Name)
	}
}
