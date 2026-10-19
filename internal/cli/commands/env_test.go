package commands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/urfave/cli/v3"
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
			name: "sets variable from positional KEY=VALUE",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
			},
			args:    []string{"set", "dev", "API_KEY=secret123"},
			wantErr: false,
		},
		{
			name: "sets multiple variables from positional KEY=VALUE args",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
			},
			args:    []string{"set", "dev", "API_URL=https://api", "DEBUG=true"},
			wantErr: false,
		},
		{
			name: "sets variable from positional KEY VALUE",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
			},
			args:    []string{"set", "dev", "API_KEY", "secret123"},
			wantErr: false,
		},
		{
			name: "rejects positional key without value",
			setup: func(db *mockEnvDB) {
				helperCreateTestEnv(db, "dev", nil)
			},
			args:    []string{"set", "dev", "API_KEY"},
			wantErr: true,
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
			if !tt.wantErr && len(tt.args) >= 3 && strings.HasPrefix(tt.args[2], "API_KEY") {
				env, err := db.GetEnvByName("dev")
				if err != nil {
					t.Fatalf("failed to fetch env: %v", err)
				}
				if got := env.Variables["API_KEY"]; got != "secret123" {
					t.Errorf("expected API_KEY=secret123, got %q", got)
				}
			}
			if !tt.wantErr && tt.name == "sets multiple variables from positional KEY=VALUE args" {
				env, err := db.GetEnvByName("dev")
				if err != nil {
					t.Fatalf("failed to fetch env: %v", err)
				}
				if got := env.Variables["API_URL"]; got != "https://api" {
					t.Errorf("expected API_URL=https://api, got %q", got)
				}
				if got := env.Variables["DEBUG"]; got != "true" {
					t.Errorf("expected DEBUG=true, got %q", got)
				}
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

func TestEnvShowJSON(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantSecret bool
	}{
		{name: "masks secrets by default", args: []string{"env", "show", "--json", "prod"}},
		{name: "table masks secrets", args: []string{"env", "show", "prod"}},
		{name: "reveal shows secrets", args: []string{"env", "show", "--json", "--reveal", "prod"}, wantSecret: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			e := env.NewEnvironment("prod", "")
			e.SetVariable("BASE_URL", "https://api.example.com")
			e.SetSecretVariable("API_KEY", "sk_live_123")
			db.SaveEnv(e)

			var runErr error
			out := captureStdout(t, func() {
				runErr = EnvCommand(db).Run(context.Background(), tt.args)
			})
			if runErr != nil {
				t.Fatalf("unexpected error: %v", runErr)
			}
			if got := strings.Contains(out, "sk_live_123"); got != tt.wantSecret {
				t.Errorf("secret present = %v, want %v; output:\n%s", got, tt.wantSecret, out)
			}
			if !strings.Contains(out, "https://api.example.com") {
				t.Errorf("non-secret value missing; output:\n%s", out)
			}
			if !strings.Contains(tt.args[2], "--json") {
				return
			}
			var got struct {
				Name      string            `json:"name"`
				ID        string            `json:"id"`
				Variables map[string]string `json:"variables"`
				Secrets   []string          `json:"secrets"`
			}
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, out)
			}
			if got.Name != "prod" || got.ID == "" || len(got.Secrets) != 1 || got.Secrets[0] != "API_KEY" {
				t.Errorf("unexpected JSON: %+v", got)
			}
			if !tt.wantSecret && got.Variables["API_KEY"] != "*****" {
				t.Errorf("API_KEY = %q, want masked", got.Variables["API_KEY"])
			}
		})
	}
}

func TestEnvListJSON(t *testing.T) {
	db := newMockEnvDB()
	helperCreateTestEnv(db, "dev", map[string]string{"A": "1", "B": "2"})
	db.SetActiveEnv("dev")

	var runErr error
	out := captureStdout(t, func() {
		runErr = EnvCommand(db).Run(context.Background(), []string{"env", "list", "--json"})
	})
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	var got []struct {
		Name      string `json:"name"`
		Active    bool   `json:"active"`
		Variables int    `json:"variables"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].Name != "dev" || !got[0].Active || got[0].Variables != 2 {
		t.Errorf("unexpected JSON: %+v", got)
	}
}

func TestEnvSetFromStdin(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		stdin   string
		key     string
		want    string
		wantErr bool
	}{
		{name: "KEY=- positional", args: []string{"set", "dev", "API_KEY=-"}, stdin: "sk_live_123\n", key: "API_KEY", want: "sk_live_123"},
		{name: "--var KEY=-", args: []string{"set", "dev", "--var", "API_KEY=-", "--var", "X=1"}, stdin: "abc", key: "API_KEY", want: "abc"},
		{name: "trims only one newline", args: []string{"set", "dev", "K=-"}, stdin: "v\n\n", key: "K", want: "v\n"},
		{name: "create with --var KEY=-", args: []string{"create", "new", "--var", "K=-"}, stdin: "s\n", key: "K", want: "s"},
		{name: "two stdin keys rejected", args: []string{"set", "dev", "A=-", "B=-"}, stdin: "x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			helperCreateTestEnv(db, "dev", nil)

			cmd := EnvCommand(db)
			cmd.Reader = strings.NewReader(tt.stdin)
			err := cmd.Run(context.Background(), append([]string{"env"}, tt.args...))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			e, _ := db.GetEnvByName(tt.args[1])
			if got, _ := e.GetVariable(tt.key); got != tt.want {
				t.Errorf("%s = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestParseEnvSetArgsRejectsLoneDashValue(t *testing.T) {
	cmd := &cli.Command{Name: "set", Action: func(ctx context.Context, c *cli.Command) error {
		_, err := parseEnvSetArgs(c.Args())
		if err == nil || !strings.Contains(err.Error(), "API_KEY=-") {
			t.Fatalf("expected KEY=- hint, got %v", err)
		}
		return nil
	}}
	if err := cmd.Run(context.Background(), []string{"set", "dev", "API_KEY", "-"}); err != nil {
		t.Fatal(err)
	}
}
