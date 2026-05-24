package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
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
			name: "imports empty supported file",
			setup: func(db *mockDB) string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "test.json")
				os.WriteFile(tmpFile, []byte(`{}`), 0644)
				return tmpFile
			},
			args:    []string{"--force"},
			wantErr: false,
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

func TestImportCommandImportsNativeCollectionExport(t *testing.T) {
	sourceCollection := types.NewCollection("payments")
	sourceCollection.SetVariable("BASE_URL", "https://api.example.com")
	sourceCollection.SetSecretVariable("API_KEY", "secret-token")
	sourceRequest := &types.SavedRequest{
		ID:         "req-1",
		Name:       "list payments",
		URL:        "https://{{BASE_URL}}/payments",
		Method:     "GET",
		Collection: "payments",
	}
	exportData, err := storage.BuildCollectionExport(sourceCollection, []*types.SavedRequest{sourceRequest}, "team-pass")
	if err != nil {
		t.Fatalf("BuildCollectionExport failed: %v", err)
	}
	data, err := storage.MarshalCollectionExport(exportData)
	if err != nil {
		t.Fatalf("MarshalCollectionExport failed: %v", err)
	}
	exportPath := filepath.Join(t.TempDir(), "payments.gurl")
	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		t.Fatalf("failed to write export: %v", err)
	}

	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "target.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	cmd := ImportCommand(db)
	output := captureStdout(t, func() {
		err = cmd.Run(context.Background(), []string{"import", "--passphrase", "team-pass", exportPath})
	})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if !strings.Contains(output, "Imported collection 'payments' (1 requests, 0 skipped)") {
		t.Fatalf("expected collection import summary, got %q", output)
	}

	importedCollection, err := db.GetCollectionByName("payments")
	if err != nil {
		t.Fatalf("GetCollectionByName failed: %v", err)
	}
	if importedCollection.Variables["API_KEY"] != "secret-token" || !importedCollection.IsSecret("API_KEY") {
		t.Fatalf("expected decrypted collection secret, got %+v", importedCollection)
	}
	importedRequest, err := db.GetRequestByName("list payments")
	if err != nil {
		t.Fatalf("GetRequestByName failed: %v", err)
	}
	if importedRequest.Collection != "payments" || importedRequest.URL != "https://{{BASE_URL}}/payments" {
		t.Fatalf("imported request mismatch: %+v", importedRequest)
	}
}

func TestImportCommandKeepsNativeRequestExportPath(t *testing.T) {
	exportPath := filepath.Join(t.TempDir(), "requests.gurl")
	data := []byte(`{
  "version": "1.0",
  "exported_at": "2026-05-23T00:00:00Z",
  "requests": [
    {
      "id": "req-1",
      "name": "health",
      "url": "https://example.com/health",
      "method": "GET"
    }
  ]
}`)
	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		t.Fatalf("failed to write export: %v", err)
	}

	db := newMockDB()
	cmd := ImportCommand(db)
	if err := cmd.Run(context.Background(), []string{"import", exportPath}); err != nil {
		t.Fatalf("import failed: %v", err)
	}
	req, err := db.GetRequestByName("health")
	if err != nil {
		t.Fatalf("GetRequestByName failed: %v", err)
	}
	if req.URL != "https://example.com/health" {
		t.Fatalf("expected native request export to import via request importer, got %+v", req)
	}
}
