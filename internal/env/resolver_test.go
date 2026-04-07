package env

import (
	"path/filepath"
	"testing"

	"github.com/sreeram/gurl/internal/storage"
)

func TestResolveVariablesSimple(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_resolver.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		ID:        "env-resolver-1",
		Name:      "Test Resolver",
		Variables: map[string]string{"BASE_URL": "https://api.example.com", "API_KEY": "secret123"},
		ParentID:  "",
	}

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	text := "Hello {{BASE_URL}} with key {{API_KEY}}"
	expected := "Hello https://api.example.com with key secret123"

	resolver := NewResolver(envStore)
	result, err := resolver.ResolveVariables(text, "env-resolver-1")
	if err != nil {
		t.Fatalf("failed to resolve variables: %v", err)
	}

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestResolveVariablesNoEnv(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_resolver_noenv.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	text := "Hello {{BASE_URL}}"
	resolver := NewResolver(envStore)
	result, err := resolver.ResolveVariables(text, "non-existent-env")

	if err != nil {
		t.Fatalf("resolve should not error for non-existent env, got: %v", err)
	}

	if result != text {
		t.Errorf("expected unchanged text '%s', got '%s'", text, result)
	}
}

func TestResolveVariablesMissingVar(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_resolver_missing.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		ID:        "env-missing-var",
		Name:      "Missing Var Test",
		Variables: map[string]string{"EXISTING": "value1"},
		ParentID:  "",
	}

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	text := "Existing: {{EXISTING}}, Missing: {{MISSING}}"
	resolver := NewResolver(envStore)
	result, err := resolver.ResolveVariables(text, "env-missing-var")

	if err != nil {
		t.Fatalf("resolve should not error for missing variables: %v", err)
	}

	if result != "Existing: value1, Missing: {{MISSING}}" {
		t.Errorf("expected missing var to remain as placeholder, got '%s'", result)
	}
}

func TestResolveVariablesEmptyText(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_resolver_empty.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		ID:        "env-empty-text",
		Name:      "Empty Text Test",
		Variables: map[string]string{"KEY": "value"},
		ParentID:  "",
	}

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	resolver := NewResolver(envStore)
	result, err := resolver.ResolveVariables("", "env-empty-text")

	if err != nil {
		t.Fatalf("resolve should not error for empty text: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

func TestResolveVariablesMultipleSameVar(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_resolver_multi.db")

	db := storage.NewLMDBWithPath(dbPath)
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	envStore := NewEnvStorage(db)

	env := &Environment{
		ID:        "env-multi",
		Name:      "Multi Same Var",
		Variables: map[string]string{"HOST": "localhost:8080"},
		ParentID:  "",
	}

	if err := envStore.SaveEnv(env); err != nil {
		t.Fatalf("failed to save environment: %v", err)
	}

	text := "http://{{HOST}}/api && http://{{HOST}}/auth"
	expected := "http://localhost:8080/api && http://localhost:8080/auth"

	resolver := NewResolver(envStore)
	result, err := resolver.ResolveVariables(text, "env-multi")
	if err != nil {
		t.Fatalf("failed to resolve variables: %v", err)
	}

	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}
