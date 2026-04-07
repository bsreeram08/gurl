package config

import (
	"github.com/sreeram/gurl/pkg/types"
)

// DefaultConfig returns the default configuration
func DefaultConfig() *types.Config {
	return &types.Config{
		General: struct {
			HistoryDepth   int    `toml:"history_depth"`
			AutoTemplate   bool   `toml:"auto_template"`
			CompletionMode string `toml:"completion_mode"`
			Timeout        string `toml:"timeout"`
		}{
			HistoryDepth:   100,
			AutoTemplate:   true,
			CompletionMode: "both",
			Timeout:        "30s",
		},
		Output: struct {
			DefaultFormat   string `toml:"default_format"`
			SyntaxHighlight bool   `toml:"syntax_highlight"`
			JSONPretty      bool   `toml:"json_pretty"`
		}{
			DefaultFormat:   "auto",
			SyntaxHighlight: true,
			JSONPretty:      true,
		},
		Cache: struct {
			TTLSeconds int `toml:"ttl_seconds"`
		}{
			TTLSeconds: 300,
		},
		Detect: struct {
			ExtractVariables bool `toml:"extract_variables"`
			SuggestMerge     bool `toml:"suggest_merge"`
			PromptTemplates  bool `toml:"prompt_templates"`
		}{
			ExtractVariables: true,
			SuggestMerge:     true,
			PromptTemplates:  true,
		},
		UI: struct {
			TUIOnDecisions    bool `toml:"tui_on_decisions"`
			TUIThresholdLines int  `toml:"tui_threshold_lines"`
		}{
			TUIOnDecisions:    true,
			TUIThresholdLines: 100,
		},
		Plugins: struct {
			Enabled []string `toml:"enabled"`
		}{
			Enabled: []string{},
		},
	}
}
