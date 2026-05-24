package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

func TestExportTimestamp(t *testing.T) {
	db := newMockDB()
	db.requests["req-1"] = &types.SavedRequest{
		ID:     "req-1",
		Name:   "test-request",
		URL:    "https://example.com",
		Method: "GET",
	}

	cmd := ExportCommand(db)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "export-test.json")

	cwd, _ := os.Getwd()
	t.Logf("CWD: %s", cwd)
	t.Logf("tmpFile: %s", tmpFile)
	absFile, _ := filepath.Abs(tmpFile)
	t.Logf("absFile: %s", absFile)
	resolved, _ := filepath.EvalSymlinks(absFile)
	t.Logf("resolved: %s", resolved)

	fullArgs := []string{"export", "--all", "--output", tmpFile}
	err := cmd.Run(context.Background(), fullArgs)
	if err != nil {
		t.Fatalf("ExportCommand.Run() error = %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	var exportData struct {
		Version    string                `json:"version"`
		ExportedAt string                `json:"exported_at"`
		Requests   []*types.SavedRequest `json:"requests"`
	}
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("failed to parse exported JSON: %v", err)
	}

	if exportData.ExportedAt == "2024-01-01T00:00:00Z" {
		t.Error("exported_at is still hardcoded to 2024-01-01T00:00:00Z")
	}

	exportedTime, err := time.Parse(time.RFC3339, exportData.ExportedAt)
	if err != nil {
		t.Fatalf("exported_at is not valid RFC3339: %v", err)
	}

	now := time.Now().UTC()
	diff := now.Sub(exportedTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Second {
		t.Errorf("exported_at is not within 5 seconds of current time: got %v, now %v, diff %v", exportedTime, now, diff)
	}
}

func TestExportCollectionIncludesEncryptedCollectionMetadata(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	collection := types.NewCollection("payments")
	collection.SetVariable("BASE_URL", "https://api.example.com")
	collection.SetSecretVariable("API_KEY", "secret-token")
	if err := db.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "list payments",
		URL:        "{{BASE_URL}}/payments",
		Method:     "GET",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	cmd := ExportCommand(db)
	outputPath := filepath.Join(t.TempDir(), "payments.gurl")
	if err := cmd.Run(context.Background(), []string{
		"export",
		"--collection",
		"payments",
		"--passphrase",
		"team-pass",
		"--output",
		outputPath,
	}); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read export: %v", err)
	}
	if strings.Contains(string(data), "secret-token") {
		t.Fatal("collection export should not contain plaintext secret")
	}
	exportedCollection, requests, err := storage.ParseCollectionExport(data, "team-pass")
	if err != nil {
		t.Fatalf("ParseCollectionExport failed: %v", err)
	}
	if exportedCollection.Name != "payments" {
		t.Fatalf("expected collection metadata, got %+v", exportedCollection)
	}
	if exportedCollection.Variables["BASE_URL"] != "https://api.example.com" {
		t.Fatalf("expected collection variable, got %+v", exportedCollection.Variables)
	}
	if exportedCollection.Variables["API_KEY"] != "secret-token" {
		t.Fatalf("expected decrypted secret, got %q", exportedCollection.Variables["API_KEY"])
	}
	if len(requests) != 1 || requests[0].Collection != "payments" {
		t.Fatalf("expected exported request collection metadata, got %+v", requests)
	}
}

func TestExportCollectionRequiresPassphraseForSecrets(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	collection := types.NewCollection("payments")
	collection.SetSecretVariable("API_KEY", "secret-token")
	if err := db.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	cmd := ExportCommand(db)
	err := cmd.Run(context.Background(), []string{"export", "--collection", "payments"})
	if err == nil || !strings.Contains(err.Error(), "passphrase is required") {
		t.Fatalf("expected passphrase error, got %v", err)
	}
}
