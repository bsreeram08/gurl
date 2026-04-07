package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads defaults when no config file exists", func(t *testing.T) {
		// Create a temp directory with no config file
		tmpDir := t.TempDir()
		oldCwd, _ := os.Getwd()
		defer os.Chdir(oldCwd)
		os.Chdir(tmpDir)

		// Clear environment variables
		oldConfigPath := os.Getenv("SCURL_CONFIG_PATH")
		os.Unsetenv("SCURL_CONFIG_PATH")

		loader := NewLoader()
		config, err := loader.Load()

		os.Setenv("SCURL_CONFIG_PATH", oldConfigPath)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.General.HistoryDepth != 100 {
			t.Errorf("expected default HistoryDepth of 100, got %d", config.General.HistoryDepth)
		}
		if config.General.AutoTemplate != true {
			t.Errorf("expected default AutoTemplate of true, got %v", config.General.AutoTemplate)
		}
		if config.Output.SyntaxHighlight != true {
			t.Errorf("expected default SyntaxHighlight of true, got %v", config.Output.SyntaxHighlight)
		}
	})

	t.Run("loads from SCURL_CONFIG_PATH", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		// Write a config file with custom values
		content := `[general]
history_depth = 500
auto_template = false
completion_mode = "fuzzy"

[output]
default_format = "json"
syntax_highlight = false
json_pretty = false

[cache]
ttl_seconds = 600

[detect]
extract_variables = false
suggest_merge = false
prompt_templates = false

[ui]
tui_on_decisions = false
tui_threshold_lines = 50

[plugins]
enabled = ["plugin1", "plugin2"]
`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		oldConfigPath := os.Getenv("SCURL_CONFIG_PATH")
		os.Setenv("SCURL_CONFIG_PATH", configPath)
		defer os.Setenv("SCURL_CONFIG_PATH", oldConfigPath)

		loader := NewLoader()
		config, err := loader.Load()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.General.HistoryDepth != 500 {
			t.Errorf("expected HistoryDepth of 500, got %d", config.General.HistoryDepth)
		}
		if config.General.AutoTemplate != false {
			t.Errorf("expected AutoTemplate of false, got %v", config.General.AutoTemplate)
		}
		if config.Output.DefaultFormat != "json" {
			t.Errorf("expected DefaultFormat of 'json', got %s", config.Output.DefaultFormat)
		}
		if config.Cache.TTLSeconds != 600 {
			t.Errorf("expected TTLSeconds of 600, got %d", config.Cache.TTLSeconds)
		}
		if len(config.Plugins.Enabled) != 2 {
			t.Errorf("expected 2 plugins, got %d", len(config.Plugins.Enabled))
		}
	})

	t.Run("loads from .scurlrc in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldCwd, _ := os.Getwd()
		defer os.Chdir(oldCwd)
		os.Chdir(tmpDir)

		// Clear env var to ensure .scurlrc is used
		oldConfigPath := os.Getenv("SCURL_CONFIG_PATH")
		os.Unsetenv("SCURL_CONFIG_PATH")

		// Create .scurlrc
		content := `[general]
history_depth = 200
`
		if err := os.WriteFile(".scurlrc", []byte(content), 0644); err != nil {
			t.Fatalf("failed to write .scurlrc: %v", err)
		}

		loader := NewLoader()
		config, err := loader.Load()

		os.Setenv("SCURL_CONFIG_PATH", oldConfigPath)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.General.HistoryDepth != 200 {
			t.Errorf("expected HistoryDepth of 200, got %d", config.General.HistoryDepth)
		}
	})

	t.Run("loads from home directory .scurlrc", func(t *testing.T) {
		// Create a temp home directory
		tmpDir := t.TempDir()
		homeDir := filepath.Join(tmpDir, "home")
		os.MkdirAll(homeDir, 0755)

		// Write config to home/.scurlrc
		content := `[general]
history_depth = 300
`
		if err := os.WriteFile(filepath.Join(homeDir, ".scurlrc"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write .scurlrc: %v", err)
		}

		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", homeDir)
		defer os.Setenv("HOME", oldHome)

		// Clear env var to prevent SCURL_CONFIG_PATH from taking precedence
		oldConfigPath := os.Getenv("SCURL_CONFIG_PATH")
		os.Unsetenv("SCURL_CONFIG_PATH")

		// Create a temp cwd with no .scurlrc
		tmpCwd := t.TempDir()
		oldCwd, _ := os.Getwd()
		defer os.Chdir(oldCwd)
		os.Chdir(tmpCwd)

		loader := NewLoader()
		config, err := loader.Load()

		os.Setenv("SCURL_CONFIG_PATH", oldConfigPath)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.General.HistoryDepth != 300 {
			t.Errorf("expected HistoryDepth of 300, got %d", config.General.HistoryDepth)
		}
	})

	t.Run("handles invalid TOML gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := os.Getenv("SCURL_CONFIG_PATH")
		configPath := filepath.Join(tmpDir, "config.toml")

		// Write invalid TOML
		content := `[general]
history_depth = not_a_number
invalid_toml =
`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		os.Setenv("SCURL_CONFIG_PATH", configPath)
		defer os.Setenv("SCURL_CONFIG_PATH", oldConfigPath)

		loader := NewLoader()
		_, err := loader.Load()

		if err == nil {
			t.Error("expected error for invalid TOML, got nil")
		}
	})

	t.Run("merges user config with defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigPath := os.Getenv("SCURL_CONFIG_PATH")
		configPath := filepath.Join(tmpDir, "config.toml")

		// Write partial config - only override specific fields
		content := `[general]
history_depth = 999
`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		os.Setenv("SCURL_CONFIG_PATH", configPath)
		defer os.Setenv("SCURL_CONFIG_PATH", oldConfigPath)

		loader := NewLoader()
		config, err := loader.Load()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// User-specified value should be used
		if config.General.HistoryDepth != 999 {
			t.Errorf("expected HistoryDepth of 999, got %d", config.General.HistoryDepth)
		}
		// Default should be preserved for unspecified fields
		if config.General.AutoTemplate != true {
			t.Errorf("expected default AutoTemplate of true, got %v", config.General.AutoTemplate)
		}
		if config.Output.SyntaxHighlight != true {
			t.Errorf("expected default SyntaxHighlight of true, got %v", config.Output.SyntaxHighlight)
		}
	})
}
