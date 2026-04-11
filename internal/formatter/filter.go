package formatter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/PaesslerAG/jsonpath"
	"github.com/antchfx/xmlquery"
)

// FilterJSON extracts values from JSON body using JSONPath syntax.
// Returns the result pretty-printed as JSON.
// Path must start with '$' (e.g., "$.name", "$.data.users[0].email").
func FilterJSON(body []byte, path string) (string, error) {
	if len(path) == 0 || path[0] != '$' {
		return "", fmt.Errorf("invalid JSONPath: must start with '$', got %q", path)
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %v", err)
	}

	result, err := jsonpath.Get(path, data)
	if err != nil {
		return "", fmt.Errorf("JSONPath error: %v", err)
	}

	// Handle nil result (no match)
	if result == nil {
		return "", nil
	}

	// Pretty-print the result
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format result: %v", err)
	}

	return string(output), nil
}

// FilterJSONValue extracts values from JSON body using JSONPath syntax.
// Returns the raw value directly without additional JSON marshaling.
// This avoids double-parsing when the caller needs to extract typed values.
func FilterJSONValue(body []byte, path string) (interface{}, error) {
	if len(path) == 0 || path[0] != '$' {
		return nil, fmt.Errorf("invalid JSONPath: must start with '$', got %q", path)
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}

	result, err := jsonpath.Get(path, data)
	if err != nil {
		return nil, fmt.Errorf("JSONPath error: %v", err)
	}

	return result, nil
}

// FilterXML extracts values from XML body using XPath syntax.
// Returns the result pretty-printed as XML.
// Path must start with '/' or '//' (e.g., "//title", "//book[@category='fiction']").
func FilterXML(body []byte, xpath string) (string, error) {
	if len(xpath) == 0 || (xpath[0] != '/' && xpath[0] != '*') {
		return "", fmt.Errorf("invalid XPath: must start with '/' or '//', got %q", xpath)
	}

	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("invalid XML: %v", err)
	}

	nodes, err := xmlquery.QueryAll(doc, xpath)
	if err != nil {
		return "", fmt.Errorf("XPath error: %v", err)
	}

	if len(nodes) == 0 {
		return "", nil
	}

	var results []string
	for _, node := range nodes {
		results = append(results, node.OutputXML(true))
	}

	// Return single element without array wrapper, or multiple elements
	if len(results) == 1 {
		return results[0], nil
	}

	// Multiple results - wrap in array-like format
	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format result: %v", err)
	}

	return string(output), nil
}

// Filter detects the path type and applies the appropriate filter.
// Use this for automatic path type detection.
func Filter(body []byte, path string) (string, error) {
	if len(path) == 0 {
		return "", errors.New("path cannot be empty")
	}

	switch {
	case path[0] == '$':
		return FilterJSON(body, path)
	case path[0] == '/' || path[0] == '*':
		return FilterXML(body, path)
	default:
		return "", fmt.Errorf("unknown path type: must start with '$' (JSONPath) or '/' (XPath), got %q", path)
	}
}
