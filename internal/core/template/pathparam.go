package template

import (
	"fmt"
	"net/url"
	"regexp"
)

// pathParamRegexp matches :param and {param} in URLs
var pathParamRegexp = regexp.MustCompile(`(:[a-zA-Z_][a-zA-Z0-9_]*|\{[a-zA-Z_][a-zA-Z0-9_]*\})`)

// ResolvePathParams replaces :param and {param} placeholders with values from params map.
// Returns error for any unresolved parameters or empty values.
func ResolvePathParams(urlStr string, params map[string]string) (string, error) {
	result := urlStr

	// Find all path parameters in the URL
	matches := pathParamRegexp.FindAllStringSubmatchIndex(result, -1)
	if len(matches) == 0 {
		return urlStr, nil
	}

	// Collect all param names used
	var missingParams []string
	var emptyParams []string

	// Check each unique param
	for _, match := range matches {
		// Extract the full match (e.g., ":id" or "{id}")
		paramFull := result[match[0]:match[1]]
		// Extract param name (without : or {})
		paramName := paramFull[1:] // remove leading : or {
		if paramName[len(paramName)-1] == '}' {
			paramName = paramName[:len(paramName)-1]
		}

		if val, ok := params[paramName]; !ok {
			missingParams = append(missingParams, paramName)
		} else if val == "" {
			emptyParams = append(emptyParams, paramName)
		}
	}

	if len(missingParams) > 0 {
		return "", fmt.Errorf("unresolved path parameter: %s", missingParams[0])
	}

	if len(emptyParams) > 0 {
		return "", fmt.Errorf("empty value for path parameter: %s", emptyParams[0])
	}

	// Replace all path parameters with URL-encoded values
	// We need to do this carefully to avoid double-encoding
	result = pathParamRegexp.ReplaceAllStringFunc(result, func(match string) string {
		// Remove leading : or {
		paramName := match[1:]
		// Remove trailing } if present
		if len(paramName) > 0 && paramName[len(paramName)-1] == '}' {
			paramName = paramName[:len(paramName)-1]
		}
		if val, ok := params[paramName]; ok {
			return url.PathEscape(val)
		}
		return match // should not happen if validation passed
	})

	return result, nil
}

// ExtractPathParamNames extracts all path parameter names from a URL
func ExtractPathParamNames(urlStr string) []string {
	names := make([]string, 0)
	seen := make(map[string]bool)

	matches := pathParamRegexp.FindAllStringSubmatch(urlStr, -1)
	for _, match := range matches {
		if len(match) >= 1 {
			name := match[0][1:]
			if len(name) > 0 && name[len(name)-1] == '}' {
				name = name[:len(name)-1]
			}
			if !seen[name] {
				names = append(names, name)
				seen[name] = true
			}
		}
	}

	return names
}

// HasPathParams checks if a URL contains path parameters (:id or {id})
// but NOT {{var}} template variables (which start with double braces)
func HasPathParams(urlStr string) bool {
	matches := pathParamRegexp.FindAllStringSubmatchIndex(urlStr, -1)
	for _, match := range matches {
		if match[0] > 1 && urlStr[match[0]-1] == '{' && urlStr[match[0]-2] == '{' {
			continue
		}
		return true
	}
	return false
}

// extractPathParamNamesFiltered is like ExtractPathParamNames but skips {{var}} style templates
func extractPathParamNamesFiltered(urlStr string) []string {
	names := make([]string, 0)
	seen := make(map[string]bool)

	matches := pathParamRegexp.FindAllStringSubmatchIndex(urlStr, -1)
	for _, match := range matches {
		if match[0] > 1 && urlStr[match[0]-1] == '{' && urlStr[match[0]-2] == '{' {
			continue
		}
		fullMatch := urlStr[match[0]:match[1]]
		name := fullMatch[1:]
		if len(name) > 0 && name[len(name)-1] == '}' {
			name = name[:len(name)-1]
		}
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}

	return names
}
