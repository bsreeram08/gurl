package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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
