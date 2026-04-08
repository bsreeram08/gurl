package assertions

import (
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// TOMLParser parses assertions from TOML format.
type TOMLParser struct{}

// NewTOMLParser creates a new TOML parser.
func NewTOMLParser() *TOMLParser {
	return &TOMLParser{}
}

// TOMLAssertion represents a single assertion in TOML format.
type TOMLAssertion struct {
	Field string `toml:"field"`
	Op    string `toml:"op"`
	Value string `toml:"value"`
}

// TOMLAssertConfig is the top-level TOML structure for assertions.
type TOMLAssertConfig struct {
	Assertions []TOMLAssertion `toml:"assertions"`
}

// ParseTOML parses assertions from TOML content.
func (p *TOMLParser) ParseTOML(content string) ([]Assertion, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	var cfg TOMLAssertConfig
	if err := toml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("TOML parse error: %w", err)
	}

	results := make([]Assertion, len(cfg.Assertions))
	for i, a := range cfg.Assertions {
		results[i] = Assertion{
			Field: a.Field,
			Op:    a.Op,
			Value: a.Value,
		}
	}
	return results, nil
}

// CLI parser

// CLIParser parses assertions from CLI --assert flag format.
type CLIParser struct{}

// NewCLIParser creates a new CLI parser.
func NewCLIParser() *CLIParser {
	return &CLIParser{}
}

// Parse parses a CLI assertion string like "status=200" or "body contains success".
func (p *CLIParser) Parse(input string) (Assertion, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Assertion{}, fmt.Errorf("empty assertion")
	}

	// Try operators with spaces (contains, not_contains, matches, exists)
	// Format: "field op value" or "field op" (for exists)
	for _, op := range []string{"not_contains", "contains", "matches", "exists"} {
		searchStr := " " + op
		idx := strings.Index(input, searchStr)
		if idx > 0 {
			field := strings.TrimSpace(input[:idx])
			value := strings.TrimSpace(input[idx+len(searchStr):])
			return Assertion{
				Field: field,
				Op:    op,
				Value: value,
			}, nil
		}
	}

	// Try single-character operators: =, !=, <, >, <=, >=
	// These are directly attached to the field
	for _, op := range []string{"<=", ">=", "!=", "=", "<", ">"} {
		idx := strings.Index(input, op)
		if idx > 0 {
			field := strings.TrimSpace(input[:idx])
			value := strings.TrimSpace(input[idx+len(op):])
			if field != "" && value != "" {
				return Assertion{
					Field: field,
					Op:    op,
					Value: value,
				}, nil
			}
		}
	}

	return Assertion{}, fmt.Errorf("invalid assertion format: %q", input)
}

// ParseSlice parses multiple CLI assertion strings.
func (p *CLIParser) ParseSlice(inputs []string) ([]Assertion, error) {
	results := make([]Assertion, 0, len(inputs))
	for _, input := range inputs {
		assertion, err := p.Parse(input)
		if err != nil {
			return nil, err
		}
		results = append(results, assertion)
	}
	return results, nil
}

// Parser is the unified parser interface.
type Parser interface {
	Parse(content string) ([]Assertion, error)
}

// TOMLAssertionParser implements Parser for TOML format.
type TOMLAssertionParser struct{}

// Parse parses TOML assertion content.
func (TOMLAssertionParser) Parse(content string) ([]Assertion, error) {
	return NewTOMLParser().ParseTOML(content)
}

// CLIAssertionParser implements Parser for CLI format.
type CLIAssertionParser struct{}

// Parse parses CLI assertion content.
func (CLIAssertionParser) Parse(content string) ([]Assertion, error) {
	return NewCLIParser().ParseSlice(strings.Split(content, ","))
}
