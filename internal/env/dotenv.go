package env

import (
	"fmt"
	"os"
	"strings"
)

// ParseDotenv parses .env content into key-value pairs.
// Handles: KEY=value, KEY="quoted", KEY='single quoted', # comments, empty lines, export KEY=value
func ParseDotenv(content string) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		lineType := classifyLine(line)

		switch lineType {
		case lineTypeEmpty, lineTypeComment:
			// Skip empty lines and comments
			continue

		case lineTypeKeyValue:
			key, value, err := parseKeyValue(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum+1, err)
			}
			result[key] = value

		default:
			// Should not reach here with proper classification
			return nil, fmt.Errorf("line %d: unrecognized line type", lineNum+1)
		}
	}

	return result, nil
}

// ParseDotenvFile reads and parses a .env file
func ParseDotenvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read .env file: %w", err)
	}

	return ParseDotenv(string(data))
}

// Line type classification
type lineType int

const (
	lineTypeEmpty lineType = iota
	lineTypeComment
	lineTypeKeyValue
)

// classifyLine determines the type of a line using switch (not if-else)
func classifyLine(line string) lineType {
	trimmed := strings.TrimSpace(line)

	// Empty line
	if trimmed == "" {
		return lineTypeEmpty
	}

	// Comment line
	if strings.HasPrefix(trimmed, "#") {
		return lineTypeComment
	}

	// Key-value line (contains =)
	if strings.Contains(trimmed, "=") {
		return lineTypeKeyValue
	}

	return lineTypeEmpty
}

// parseKeyValue parses a KEY=value or export KEY=value line
func parseKeyValue(line string) (key, value string, err error) {
	trimmed := strings.TrimSpace(line)

	// Remove export prefix if present
	if strings.HasPrefix(trimmed, "export") {
		trimmed = strings.TrimPrefix(trimmed, "export")
		trimmed = strings.TrimSpace(trimmed)
	}

	// Find the first = to split key and value
	eqIndex := strings.Index(trimmed, "=")
	if eqIndex == -1 {
		return "", "", fmt.Errorf("invalid key-value pair: missing '='")
	}

	key = trimmed[:eqIndex]
	value = trimmed[eqIndex+1:]

	// Validate key
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", fmt.Errorf("empty key")
	}

	// Handle quoting for value
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			value = value[1 : len(value)-1]
		}
	}

	return key, value, nil
}
