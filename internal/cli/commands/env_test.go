package commands

import (
	"context"
	"testing"

	"github.com/sreeram/gurl/internal/env"
)

type mockEnvDB struct {
	envs      map[string]*env.Environment
	byName    map[string]string
	activeEnv string
}

func newMockEnvDB() *mockEnvDB {
	return &mockEnvDB{
		envs:   make(map[string]*env.Environment),
		byName: make(map[string]string),
	}
}

func (m *mockEnvDB) SaveEnv(e *env.Environment) error {
	if e.ID == "" {
		e.ID = "test-id-" + e.Name
	}
	m.envs[e.ID] = e
	m.byName[e.Name] = e.ID
	return nil
}

func (m *mockEnvDB) GetEnv(id string) (*env.Environment, error) {
	e, ok := m.envs[id]
	if !ok {
		return nil, nil
	}
	return e, nil
}

func (m *mockEnvDB) DeleteEnv(id string) error {
	e, ok := m.envs[id]
	if !ok {
		return nil
	}
	delete(m.byName, e.Name)
	delete(m.envs, id)
	if m.activeEnv == e.Name {
		m.activeEnv = ""
	}
	return nil
}

func (m *mockEnvDB) ListEnvs() ([]*env.Environment, error) {
	var result []*env.Environment
	for _, e := range m.envs {
		result = append(result, e)
	}
	return result, nil
}

func (m *mockEnvDB) GetEnvByName(name string) (*env.Environment, error) {
	id, ok := m.byName[name]
	if !ok {
		return nil, nil
	}
	return m.envs[id], nil
}

func (m *mockEnvDB) GetActiveEnv() (string, error) {
	return m.activeEnv, nil
}

func (m *mockEnvDB) SetActiveEnv(name string) error {
	m.activeEnv = name
	return nil
}

func helperCreateTestEnv(db *mockEnvDB, name string, vars map[string]string) {
	e := env.NewEnvironment(name, "")
	for k, v := range vars {
		e.SetVariable(k, v)
	}
	db.SaveEnv(e)
}

func TestEnvCreate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockEnvDB)
		args    []string
		wantErr bool
	}{
		{
			name:    "creates environment with name only",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"create", "dev"},
			wantErr: false,
		},
		{
			name:    "creates environment with variables",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"create", "prod", "--var", "BASE_URL=https://api.example.com"},
			wantErr: false,
		},
		{
			name:    "creates environment with multiple variables",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"create", "staging", "--var", "BASE_URL=https://staging.example.com", "--var", "API_KEY=secret123"},
			wantErr: false,
		},
		{
			name:    "fails when name is missing",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"create"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EnvCommand(db)

			err := cmd.Run(context.Background(), append([]string{"env"}, tt.args...))

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnvList(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockEnvDB)
		args    []string
		wantErr bool
	}{
		{
			name: "lists all environments",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", map[string]string{"FOO": "bar"})
				helperCreateTestEnv(db, "prod", map[string]string{"FOO": "baz"})
			},
			args:    []string{"list"},
			wantErr: false,
		},
		{
			name:    "empty list shows no environments",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"list"},
			wantErr: false,
		},
		{
			name: "marks active environment",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
				helperCreateTestEnv(db, "prod", nil)
				db.SetActiveEnv("dev")
			},
			args:    []string{"list"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EnvCommand(db)

			err := cmd.Run(context.Background(), append([]string{"env"}, tt.args...))

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnvSwitch(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockEnvDB)
		args    []string
		wantErr bool
	}{
		{
			name: "switches to existing environment",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
				helperCreateTestEnv(db, "prod", nil)
			},
			args:    []string{"switch", "dev"},
			wantErr: false,
		},
		{
			name: "fails when environment does not exist",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
			},
			args:    []string{"switch", "nonexistent"},
			wantErr: true,
		},
		{
			name:    "fails when name is missing",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"switch"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EnvCommand(db)

			err := cmd.Run(context.Background(), append([]string{"env"}, tt.args...))

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnvDelete(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockEnvDB)
		args    []string
		wantErr bool
	}{
		{
			name: "deletes existing environment",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
			},
			args:    []string{"delete", "dev"},
			wantErr: false,
		},
		{
			name:    "fails when environment does not exist",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"delete", "nonexistent"},
			wantErr: true,
		},
		{
			name:    "fails when name is missing",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"delete"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EnvCommand(db)

			err := cmd.Run(context.Background(), append([]string{"env"}, tt.args...))

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnvShow(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockEnvDB)
		args    []string
		wantErr bool
	}{
		{
			name: "shows environment details",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", map[string]string{
					"BASE_URL": "https://dev.example.com",
					"API_KEY":  "secret123",
				})
			},
			args:    []string{"show", "dev"},
			wantErr: false,
		},
		{
			name:    "fails when environment does not exist",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"show", "nonexistent"},
			wantErr: true,
		},
		{
			name:    "fails when name is missing",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"show"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EnvCommand(db)

			err := cmd.Run(context.Background(), append([]string{"env"}, tt.args...))

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnvSet(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockEnvDB)
		args    []string
		wantErr bool
	}{
		{
			name: "sets variable on existing environment",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
			},
			args:    []string{"set", "dev", "--var", "NEW_VAR=newvalue"},
			wantErr: false,
		},
		{
			name: "sets multiple variables",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
			},
			args:    []string{"set", "dev", "--var", "VAR1=value1", "--var", "VAR2=value2"},
			wantErr: false,
		},
		{
			name:    "fails when environment does not exist",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"set", "nonexistent", "--var", "FOO=bar"},
			wantErr: true,
		},
		{
			name:    "fails when name is missing",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"set"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EnvCommand(db)

			err := cmd.Run(context.Background(), append([]string{"env"}, tt.args...))

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnvUnset(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockEnvDB)
		args    []string
		wantErr bool
	}{
		{
			name: "unsets variable from existing environment",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", map[string]string{"TO_REMOVE": "value"})
			},
			args:    []string{"unset", "dev", "--var", "TO_REMOVE"},
			wantErr: false,
		},
		{
			name:    "fails when environment does not exist",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"unset", "nonexistent", "--var", "FOO"},
			wantErr: true,
		},
		{
			name:    "fails when name is missing",
			setup:   func(db *mockEnvDB) {},
			args:    []string{"unset"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EnvCommand(db)

			err := cmd.Run(context.Background(), append([]string{"env"}, tt.args...))

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
