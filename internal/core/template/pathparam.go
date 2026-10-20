package template

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
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

// pathParamMatches returns the index pairs of :name and {name} path
// parameters in urlStr that are not part of a {{...}} placeholder (such as
// {{base}} or the ":num" suffix of {{amount:num}}).
func pathParamMatches(urlStr string) [][]int {
	placeholders := templatePattern.FindAllStringIndex(urlStr, -1)
	var out [][]int
	for _, m := range pathParamRegexp.FindAllStringIndex(urlStr, -1) {
		inside := false
		for _, p := range placeholders {
			if m[0] < p[1] && p[0] < m[1] {
				inside = true
				break
			}
		}
		if !inside {
			out = append(out, m)
		}
	}
	return out
}

func pathParamName(match string) string {
	name := match[1:]
	if name[len(name)-1] == '}' {
		name = name[:len(name)-1]
	}
	return name
}

// PathParamNames returns the :name and {name} path parameter names of urlStr
// in order of appearance, without duplicates, ignoring {{...}} placeholders.
func PathParamNames(urlStr string) []string {
	var names []string
	seen := make(map[string]bool)
	for _, m := range pathParamMatches(urlStr) {
		name := pathParamName(urlStr[m[0]:m[1]])
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// FillPathParams replaces each :name and {name} path parameter of urlStr
// that has a value in params with the URL path-escaped value. Parameters
// without a value and {{...}} placeholders are left as written.
func FillPathParams(urlStr string, params map[string]string) string {
	matches := pathParamMatches(urlStr)
	if len(matches) == 0 {
		return urlStr
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		val, ok := params[pathParamName(urlStr[m[0]:m[1]])]
		if !ok {
			continue
		}
		b.WriteString(urlStr[last:m[0]])
		b.WriteString(url.PathEscape(val))
		last = m[1]
	}
	b.WriteString(urlStr[last:])
	return b.String()
}
