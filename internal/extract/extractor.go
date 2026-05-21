package extract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// Extractor handles response variable extraction
type Extractor struct{}

// NewExtractor creates a new extractor
func NewExtractor() *Extractor {
	return &Extractor{}
}

// Extract processes all extract rules against a response and returns extracted variables
func (e *Extractor) Extract(respBody []byte, headers map[string][]string, rules []types.Extract) (map[string]string, error) {
	result := make(map[string]string)

	for _, rule := range rules {
		value, err := e.extractOne(respBody, headers, rule.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to extract %s: %w", rule.Name, err)
		}
		result[rule.Name] = value
	}

	return result, nil
}

func (e *Extractor) extractOne(body []byte, headers map[string][]string, source string) (string, error) {
	source = strings.TrimSpace(source)

	switch {
	case strings.HasPrefix(source, "jsonpath:"):
		path := strings.TrimPrefix(source, "jsonpath:")
		return e.extractJSONPath(body, path)

	case strings.HasPrefix(source, "header:"):
		headerName := strings.TrimPrefix(source, "header:")
		return e.extractHeader(headers, headerName), nil

	case strings.HasPrefix(source, "regex:"):
		pattern := strings.TrimPrefix(source, "regex:")
		return e.extractRegex(body, pattern)

	case strings.HasPrefix(source, "jq:"):
		// For now, treat jq same as jsonpath (can be enhanced later)
		path := strings.TrimPrefix(source, "jq:")
		return e.extractJSONPath(body, path)

	default:
		return "", fmt.Errorf("unknown extract source: %s", source)
	}
}

func (e *Extractor) extractJSONPath(body []byte, path string) (string, error) {
	if len(body) == 0 {
		return "", nil
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Simple JSONPath implementation using gjson-like logic or manual traversal
	// For a production version we'd use github.com/tidwall/gjson
	// This is a minimal implementation for review
	path = strings.TrimPrefix(path, "$.")
	parts := strings.Split(path, ".")

	current := data
	for _, part := range parts {
		if part == "" {
			continue
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("cannot traverse path at %s", part)
		}
		current = m[part]
		if current == nil {
			return "", nil
		}
	}

	switch v := current.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return fmt.Sprintf("%v", v), nil
	default:
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}

func (e *Extractor) extractHeader(headers map[string][]string, name string) string {
	for k, vals := range headers {
		if strings.EqualFold(k, name) && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func (e *Extractor) extractRegex(body []byte, pattern string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	matches := re.FindSubmatch(body)
	if len(matches) > 1 {
		return string(matches[1]), nil
	}
	if len(matches) == 1 {
		return string(matches[0]), nil
	}
	return "", nil
}
