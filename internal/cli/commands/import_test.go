package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestImportCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB) string
		args    []string
		wantErr bool
	}{
		{
			name: "lists supported formats with --list",
			setup: func(db *mockDB) string {
				return ""
			},
			args:    []string{"--list"},
			wantErr: false,
		},
		{
			name: "fails when path is missing",
			setup: func(db *mockDB) string {
				return ""
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "fails when file does not exist",
			setup: func(db *mockDB) string {
				return ""
			},
			args:    []string{"/nonexistent/path/file.json"},
			wantErr: true,
		},
		{
			name: "imports with --force flag creates temp file first",
			setup: func(db *mockDB) string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "test.json")
				os.WriteFile(tmpFile, []byte(`{}`), 0644)
				return tmpFile
			},
			args:    []string{"--force"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			var filePath string
			if tt.setup != nil {
				filePath = tt.setup(db)
			}

			args := tt.args
			if filePath != "" && len(args) > 0 && args[0] != "--list" {
				args = append(args, filePath)
			} else if filePath != "" {
				args = append(args, filePath)
			}

			fullArgs := append([]string{"import"}, args...)
			cmd := ImportCommand(db)
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

func TestImportListFormats(t *testing.T) {
	db := newMockDB()
	cmd := ImportCommand(db)

	err := cmd.Run(context.Background(), []string{"import", "--list"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
