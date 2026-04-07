package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/sreeram/gurl/pkg/types"
)

// Loader handles loading configuration from TOML files
type Loader struct {
	configPaths []string
}

// NewLoader creates a new config loader
func NewLoader() *Loader {
	return &Loader{
		configPaths: []string{
			".scurlrc",
			filepath.Join(os.Getenv("HOME"), ".scurlrc"),
			filepath.Join(os.Getenv("HOME"), ".config", "scurl", "config.toml"),
		},
	}
}

// Load loads the configuration from the first available config file
func (l *Loader) Load() (*types.Config, error) {
	config := DefaultConfig()

	// Check SCURL_CONFIG_PATH environment variable first
	if envPath := os.Getenv("SCURL_CONFIG_PATH"); envPath != "" {
		if err := l.loadFile(envPath, config); err != nil {
			return nil, fmt.Errorf("loading config from SCURL_CONFIG_PATH: %w", err)
		}
		return config, nil
	}

	// Load from the first found config file
	for _, path := range l.configPaths {
		if _, err := os.Stat(path); err == nil {
			if err := l.loadFile(path, config); err != nil {
				return nil, fmt.Errorf("loading config from %s: %w", path, err)
			}
			return config, nil
		}
	}

	return config, nil
}

// loadFile loads a TOML config file into the given config struct
func (l *Loader) loadFile(path string, config *types.Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	if err := toml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("parsing TOML: %w", err)
	}

	return nil
}

// Save saves the configuration to a TOML file
func (l *Loader) Save(path string, config *types.Config) error {
	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config to TOML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
