package importers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sreeram/gurl/pkg/types"
)

// GurlImporter handles native gurl .gurl export files
type GurlImporter struct{}

// Name returns the importer name
func (g *GurlImporter) Name() string {
	return "gurl"
}

// Extensions returns supported file extensions
func (g *GurlImporter) Extensions() []string {
	return []string{".gurl"}
}

// gurlExportFile represents the structure of a gurl export file
type gurlExportFile struct {
	Version    string                `json:"version"`
	ExportedAt string                `json:"exported_at"`
	Requests   []*types.SavedRequest `json:"requests"`
}

// Parse reads and parses a gurl export file
func (g *GurlImporter) Parse(path string) ([]*types.SavedRequest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, expected a .gurl file")
	}

	if filepath.Ext(path) != ".gurl" {
		return nil, fmt.Errorf("not a .gurl file")
	}

	data, err := readFile(path)
	if err != nil {
		return nil, err
	}

	var export gurlExportFile
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("parse gurl export: %w", err)
	}

	if export.Version != "1.0" {
		return nil, fmt.Errorf("unsupported gurl export version: %q (expected \"1.0\")", export.Version)
	}

	if export.Requests == nil {
		return []*types.SavedRequest{}, nil
	}

	return export.Requests, nil
}
