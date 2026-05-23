package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/internal/storage"
)

func TestEnvStorageUsesFileStoreForProjectEnvironments(t *testing.T) {
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	dbPath := filepath.Join(t.TempDir(), "gurl.db")
	store := NewEnvStorageWithPathAndProject(dbPath, proj)

	env := NewEnvironment("local", "")
	env.SetVariable("BASE_URL", "https://file.example.com")
	if err := store.SaveEnv(env); err != nil {
		t.Fatalf("SaveEnv failed: %v", err)
	}

	loaded, err := store.GetEnvByName("local")
	if err != nil {
		t.Fatalf("GetEnvByName failed: %v", err)
	}
	if loaded.Variables["BASE_URL"] != "https://file.example.com" {
		t.Fatalf("expected file env variable, got %+v", loaded.Variables)
	}

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	dbStore := NewEnvStorage(db)
	if _, err := dbStore.GetEnvByName("local"); err == nil {
		t.Fatal("expected project environment to skip DB storage")
	}
}

func TestFileEnvStoreEncryptsSecretsAtRest(t *testing.T) {
	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewFileEnvStore(proj)

	env := NewEnvironment("production", "")
	env.SetVariable("BASE_URL", "https://api.example.com")
	env.SetSecretVariable("API_KEY", "secret-token")
	if err := store.SaveEnv(env); err != nil {
		t.Fatalf("SaveEnv failed: %v", err)
	}
	if env.Variables["API_KEY"] != "secret-token" {
		t.Fatalf("SaveEnv should not mutate caller secret, got %q", env.Variables["API_KEY"])
	}

	rawData, err := os.ReadFile(filepath.Join(proj.EnvironmentsDir(), safeEnvFileName("production")))
	if err != nil {
		t.Fatalf("failed to read environment file: %v", err)
	}
	if strings.Contains(string(rawData), "secret-token") {
		t.Fatal("file-backed environment should not contain plaintext secret")
	}
	if !strings.Contains(string(rawData), "gurlenc:v1:") {
		t.Fatalf("expected encrypted marker in environment file, got %s", rawData)
	}

	loaded, err := store.GetEnvByName("production")
	if err != nil {
		t.Fatalf("GetEnvByName failed: %v", err)
	}
	if loaded.Variables["API_KEY"] != "secret-token" {
		t.Fatalf("expected decrypted secret, got %q", loaded.Variables["API_KEY"])
	}
	if loaded.Variables["BASE_URL"] != "https://api.example.com" {
		t.Fatalf("expected non-secret variable to round trip, got %+v", loaded.Variables)
	}
}

func TestEnvStorageKeepsDBAndFileEnvironmentsTogether(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	dbStore := NewEnvStorage(db)
	dbEnv := NewEnvironment("db-env", "")
	if err := dbStore.SaveEnv(dbEnv); err != nil {
		t.Fatalf("DB SaveEnv failed: %v", err)
	}
	dbPath := db.Path()
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close db: %v", err)
	}

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewEnvStorageWithPathAndProject(dbPath, proj)
	fileEnv := NewEnvironment("file-env", "")
	if err := store.SaveEnv(fileEnv); err != nil {
		t.Fatalf("file SaveEnv failed: %v", err)
	}

	envs, err := store.ListEnvs()
	if err != nil {
		t.Fatalf("ListEnvs failed: %v", err)
	}
	if len(envs) != 2 {
		t.Fatalf("expected DB and file envs to coexist, got %+v", envs)
	}
}

func TestProjectEnvStorageClearActiveEnvClearsDBFallback(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	dbStore := NewEnvStorage(db)
	if err := dbStore.SetActiveEnv("db-env"); err != nil {
		t.Fatalf("DB SetActiveEnv failed: %v", err)
	}
	dbPath := db.Path()
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close db: %v", err)
	}

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	store := NewEnvStorageWithPathAndProject(dbPath, proj)
	if active, err := store.GetActiveEnv(); err != nil || active != "db-env" {
		t.Fatalf("expected DB active fallback before clear, active=%q err=%v", active, err)
	}

	if err := store.SetActiveEnv(""); err != nil {
		t.Fatalf("project SetActiveEnv clear failed: %v", err)
	}
	if active, err := store.GetActiveEnv(); err != nil || active != "" {
		t.Fatalf("expected active env to stay clear, active=%q err=%v", active, err)
	}
}
