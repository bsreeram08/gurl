package commands

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

func TestCollectionListCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "lists collections",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:         "req-1",
					Name:       "request1",
					URL:        "https://example.com",
					Collection: "api",
					UpdatedAt:  1700000000,
				}
				db.names["request1"] = "req-1"
				db.requests["req-2"] = &types.SavedRequest{
					ID:         "req-2",
					Name:       "request2",
					URL:        "https://example.com",
					Collection: "web",
					UpdatedAt:  1700000001,
				}
				db.names["request2"] = "req-2"
			},
			args:    []string{"list"},
			wantErr: false,
		},
		{
			name:    "shows empty message when no collections",
			setup:   func(db *mockDB) {},
			args:    []string{"list"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := CollectionCommand(db, &env.EnvStorage{})
			fullArgs := append([]string{"collection"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCollectionShowCommand(t *testing.T) {
	db := newMockDB()
	db.requests["req-1"] = &types.SavedRequest{
		ID:         "req-1",
		Name:       "get-user",
		URL:        "https://example.com/users/1",
		Method:     "GET",
		Collection: "api",
		UpdatedAt:  1700000000,
	}
	db.names["get-user"] = "req-1"

	cmd := CollectionCommand(db, &env.EnvStorage{})
	output := captureStdout(t, func() {
		err := cmd.Run(context.Background(), []string{"collection", "show", "api"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Collection: api") {
		t.Errorf("expected collection name in output, got %q", output)
	}
	if !strings.Contains(output, "get-user") || !strings.Contains(output, "https://example.com/users/1") {
		t.Errorf("expected request details in output, got %q", output)
	}
}

func TestCollectionRunCommandIncludesAssertBailFlag(t *testing.T) {
	db := newMockDB()
	cmd := CollectionCommand(db, &env.EnvStorage{})

	var runCmd *cli.Command
	for _, subcommand := range cmd.Commands {
		if subcommand.Name == "run" {
			runCmd = subcommand
			break
		}
	}
	if runCmd == nil {
		t.Fatal("expected collection run subcommand")
	}

	if !commandHasFlag(runCmd, "assert-bail") {
		t.Fatalf("expected collection run command to expose --assert-bail flag")
	}
}

func commandHasFlag(cmd *cli.Command, name string) bool {
	for _, flag := range cmd.Flags {
		named, ok := flag.(interface{ Names() []string })
		if !ok {
			continue
		}
		for _, flagName := range named.Names() {
			if flagName == name {
				return true
			}
		}
	}
	return false
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	return buf.String()
}

func TestCollectionAddCommand(t *testing.T) {
	db := newMockDB()
	cmd := CollectionCommand(db, &env.EnvStorage{})

	err := cmd.Run(context.Background(), []string{"collection", "add", "newcollection"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCollectionRemoveCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "removes collection",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:         "req-1",
					Name:       "request1",
					URL:        "https://example.com",
					Collection: "api",
				}
				db.names["request1"] = "req-1"
			},
			args:    []string{"remove", "api"},
			wantErr: false,
		},
		{
			name:    "fails when collection name is missing",
			setup:   func(db *mockDB) {},
			args:    []string{"remove"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := CollectionCommand(db, &env.EnvStorage{})
			fullArgs := append([]string{"collection"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCollectionRenameCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "renames collection",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:         "req-1",
					Name:       "request1",
					URL:        "https://example.com",
					Collection: "oldname",
				}
				db.names["request1"] = "req-1"
			},
			args:    []string{"rename", "oldname", "newname"},
			wantErr: false,
		},
		{
			name:    "fails when old name is missing",
			setup:   func(db *mockDB) {},
			args:    []string{"rename"},
			wantErr: true,
		},
		{
			name:    "fails when new name is missing",
			setup:   func(db *mockDB) {},
			args:    []string{"rename", "oldname"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := CollectionCommand(db, &env.EnvStorage{})
			fullArgs := append([]string{"collection"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
