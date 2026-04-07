package env

import (
	"encoding/json"
	"testing"
)

func TestEnvironmentStruct(t *testing.T) {
	env := &Environment{
		ID:        "env-123",
		Name:      "Test Environment",
		Variables: map[string]string{"BASE_URL": "https://api.example.com"},
		ParentID:  "",
	}

	if env.ID != "env-123" {
		t.Errorf("expected ID 'env-123', got '%s'", env.ID)
	}
	if env.Name != "Test Environment" {
		t.Errorf("expected Name 'Test Environment', got '%s'", env.Name)
	}
	if env.Variables["BASE_URL"] != "https://api.example.com" {
		t.Errorf("expected BASE_URL 'https://api.example.com', got '%s'", env.Variables["BASE_URL"])
	}
	if env.ParentID != "" {
		t.Errorf("expected empty ParentID, got '%s'", env.ParentID)
	}
}

func TestEnvironmentJSON(t *testing.T) {
	env := &Environment{
		ID:        "env-456",
		Name:      "Production",
		Variables: map[string]string{"API_KEY": "secret123", "BASE_URL": "https://api.production.com"},
		ParentID:  "env-global",
	}

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal Environment: %v", err)
	}

	var unmarshaled Environment
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Environment: %v", err)
	}

	if unmarshaled.ID != env.ID {
		t.Errorf("expected ID '%s', got '%s'", env.ID, unmarshaled.ID)
	}
	if unmarshaled.Name != env.Name {
		t.Errorf("expected Name '%s', got '%s'", env.Name, unmarshaled.Name)
	}
	if unmarshaled.Variables["API_KEY"] != "secret123" {
		t.Errorf("expected API_KEY 'secret123', got '%s'", unmarshaled.Variables["API_KEY"])
	}
	if unmarshaled.ParentID != "env-global" {
		t.Errorf("expected ParentID 'env-global', got '%s'", unmarshaled.ParentID)
	}
}

func TestEnvironmentWithEmptyVariables(t *testing.T) {
	env := &Environment{
		ID:        "env-empty",
		Name:      "Empty Env",
		Variables: nil,
		ParentID:  "",
	}

	if env.Variables == nil {
		t.Log("Variables can be nil (will be handled as empty in storage)")
	}

	if env.ID != "env-empty" {
		t.Errorf("expected ID 'env-empty', got '%s'", env.ID)
	}
}

func TestEnvironmentParentInheritance(t *testing.T) {
	parentEnv := &Environment{
		ID:        "env-global",
		Name:      "Global",
		Variables: map[string]string{"GLOBAL_VAR": "global-value"},
		ParentID:  "",
	}

	childEnv := &Environment{
		ID:        "env-dev",
		Name:      "Development",
		Variables: map[string]string{"DEV_VAR": "dev-value"},
		ParentID:  "env-global",
	}

	if parentEnv.ParentID != "" {
		t.Errorf("parent should have empty ParentID")
	}
	if childEnv.ParentID != "env-global" {
		t.Errorf("child should have ParentID 'env-global', got '%s'", childEnv.ParentID)
	}
}
