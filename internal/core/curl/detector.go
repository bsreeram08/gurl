package curl

import (
	"regexp"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

var (
	// Variable patterns to detect (no lookahead in Go regex)
	patterns = []struct {
		name    string
		pattern *regexp.Regexp
		replace func(match string) string
	}{
		// UUID pattern: 8-4-4-4-12 hex digits
		{
			name:    "uuid",
			pattern: uuidPattern,
			replace: func(match string) string { return "{{uuid}}" },
		},
		// Long numeric IDs
		{
			name:    "id",
			pattern: longIdPattern,
			replace: func(match string) string { return "/{{id}}" },
		},
		// Shorter numeric IDs - pattern without lookahead, match trailing char
		{
			name:    "numericId",
			pattern: shortIdPattern,
			replace: func(match string) string { return "/{{numericId}}" + string(match[len(match)-1]) },
		},
	}
)

// Package-level regex patterns for variable detection
var (
	uuidPattern    = regexp.MustCompile(`([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
	longIdPattern  = regexp.MustCompile(`/(\d{5,})`)
	shortIdPattern = regexp.MustCompile(`/\d{3,4}([/$\?]|/[a-z]|$|\?)`)
)

// ExtractVariables extracts variables from URL and body, replacing patterns with template placeholders
func ExtractVariables(url string, body string) []types.Var {
	vars := make([]types.Var, 0)
	seenVars := make(map[string]bool)

	// Process URL
	url = extractVarsFromString(url, &vars, seenVars)

	// Process body
	body = extractVarsFromString(body, &vars, seenVars)

	return vars
}

// extractVarsFromString extracts variables and replaces them with template placeholders
func extractVarsFromString(input string, vars *[]types.Var, seenVars map[string]bool) string {
	result := input

	// Extract UUIDs
	for _, match := range uuidPattern.FindAllStringSubmatch(result, -1) {
		if len(match) >= 2 {
			uuid := match[1]
			if !seenVars["uuid"] {
				*vars = append(*vars, types.Var{
					Name:    "uuid",
					Pattern: `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
					Example: uuid,
				})
				seenVars["uuid"] = true
			}
			result = strings.Replace(result, uuid, "{{uuid}}", 1)
		}
	}

	// Extract long numeric IDs (5+ digits)
	for _, match := range longIdPattern.FindAllStringSubmatch(result, -1) {
		if len(match) >= 2 {
			id := match[1]
			if !seenVars["id"] {
				*vars = append(*vars, types.Var{
					Name:    "id",
					Pattern: `\d{5,}`,
					Example: id,
				})
				seenVars["id"] = true
			}
			result = strings.Replace(result, "/"+id, "/{{id}}", 1)
		}
	}

	// Extract shorter numeric IDs (3-4 digits in path segments) - no lookahead in Go
	for _, match := range shortIdPattern.FindAllStringSubmatch(result, -1) {
		if len(match) >= 2 {
			id := match[1]
			// Skip if already replaced as "id"
			if strings.Contains(result, "{{id}}") {
				continue
			}
			if !seenVars["numericId"] {
				*vars = append(*vars, types.Var{
					Name:    "numericId",
					Pattern: `\d{3,4}`,
					Example: id,
				})
				seenVars["numericId"] = true
			}
			result = strings.Replace(result, "/"+id, "/{{numericId}}", 1)
		}
	}

	return result
}

// ReplaceVariablesInURL replaces detected variable patterns with placeholders
func ReplaceVariablesInURL(url string) (string, []types.Var) {
	vars := make([]types.Var, 0)
	seenVars := make(map[string]bool)

	result := url

	// UUID replacement
	for _, match := range uuidPattern.FindAllStringSubmatch(result, -1) {
		if len(match) >= 2 {
			uuid := match[1]
			if !seenVars["uuid"] {
				vars = append(vars, types.Var{
					Name:    "uuid",
					Pattern: `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
					Example: uuid,
				})
				seenVars["uuid"] = true
			}
			result = strings.Replace(result, uuid, "{{uuid}}", 1)
		}
	}

	// Long ID replacement
	for _, match := range longIdPattern.FindAllStringSubmatch(result, -1) {
		if len(match) >= 2 {
			id := match[1]
			if !seenVars["id"] {
				vars = append(vars, types.Var{
					Name:    "id",
					Pattern: `\d{5,}`,
					Example: id,
				})
				seenVars["id"] = true
			}
			result = strings.Replace(result, "/"+id, "/{{id}}", 1)
		}
	}

	// Short ID replacement - no lookahead in Go, match trailing char
	for _, match := range shortIdPattern.FindAllStringSubmatch(result, -1) {
		if len(match) >= 1 {
			fullMatch := match[0]
			id := fullMatch[1 : len(fullMatch)-1]
			if !seenVars["numericId"] {
				vars = append(vars, types.Var{
					Name:    "numericId",
					Pattern: `\d{3,4}`,
					Example: id,
				})
				seenVars["numericId"] = true
			}
			trailing := string(fullMatch[len(fullMatch)-1:])
			result = strings.Replace(result, fullMatch, "/{{numericId}}"+trailing, 1)
		}
	}

	return result, vars
}
