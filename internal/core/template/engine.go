package template

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// templatePattern matches {{variableName}}
var templatePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// Substitute replaces all {{varName}} placeholders with values from vars map.
// Returns an error if any variable is missing. Uses single-pass deterministic
// substitution via regexp.ReplaceAllStringFunc.
func Substitute(cmd string, vars map[string]string) (string, error) {
	// Check for missing variables first (deterministic order)
	matches := templatePattern.FindAllStringSubmatchIndex(cmd, -1)
	var missingVars []string
	for _, match := range matches {
		varName := cmd[match[2]:match[3]]
		if _, ok := vars[varName]; !ok {
			missingVars = append(missingVars, varName)
		}
	}
	if len(missingVars) > 0 {
		return "", fmt.Errorf("missing required variables: %s", strings.Join(missingVars, ", "))
	}

	// Single-pass deterministic replacement (no re-substitution)
	result := templatePattern.ReplaceAllStringFunc(cmd, func(match string) string {
		varName := match[2 : len(match)-2] // strip {{ and }}
		return vars[varName]
	})
	return result, nil
}

// Validate ensures all template variables have corresponding values
func Validate(cmd string, vars map[string]string) error {
	matches := templatePattern.FindAllStringSubmatch(cmd, -1)
	if len(matches) == 0 {
		return nil
	}

	var missingVars []string
	for _, match := range matches {
		if len(match) >= 2 {
			varName := match[1]
			if _, ok := vars[varName]; !ok {
				// Check if already in missing list
				found := false
				for _, m := range missingVars {
					if m == varName {
						found = true
						break
					}
				}
				if !found {
					missingVars = append(missingVars, varName)
				}
			}
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required variables: %s", strings.Join(missingVars, ", "))
	}

	return nil
}

// ExtractVarNames extracts all variable names from a template string
func ExtractVarNames(cmd string) []string {
	varNames := make([]string, 0)
	seen := make(map[string]bool)

	matches := templatePattern.FindAllStringSubmatch(cmd, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			varName := match[1]
			if !seen[varName] {
				varNames = append(varNames, varName)
				seen[varName] = true
			}
		}
	}

	return varNames
}

// HasVariables checks if a string contains template variables
func HasVariables(cmd string) bool {
	return templatePattern.MatchString(cmd)
}

// GetVariablesFromRequest extracts variables from a SavedRequest
func GetVariablesFromRequest(request *types.SavedRequest) []types.Var {
	vars := make([]types.Var, 0)

	// Check URL for {{varName}} style
	urlVars := ExtractVarNames(request.URL)
	for _, name := range urlVars {
		vars = append(vars, types.Var{
			Name:    name,
			Pattern: "",
			Example: "",
		})
	}

	// Check body for {{varName}} style
	bodyVars := ExtractVarNames(request.Body)
	for _, name := range bodyVars {
		found := false
		for _, v := range vars {
			if v.Name == name {
				found = true
				break
			}
		}
		if !found {
			vars = append(vars, types.Var{
				Name:    name,
				Pattern: "",
				Example: "",
			})
		}
	}

	// Check URL for :param and {param} style path parameters
	// but NOT {{var}} style template variables
	pathParamNames := extractPathParamNamesFiltered(request.URL)
	for _, name := range pathParamNames {
		found := false
		for _, v := range vars {
			if v.Name == name {
				found = true
				break
			}
		}
		if !found {
			vars = append(vars, types.Var{
				Name:    name,
				Pattern: "",
				Example: "",
			})
		}
	}

	return vars
}

// ResolvePathParamsInRequest resolves :param and {param} placeholders in a request URL
// using the PathParams field from the request. Returns error for unresolved params.
func ResolvePathParamsInRequest(request *types.SavedRequest) error {
	if !HasPathParams(request.URL) {
		return nil
	}

	// Build params map from PathParams field
	params := make(map[string]string)
	for _, p := range request.PathParams {
		params[p.Name] = p.Example
	}

	resolved, err := ResolvePathParams(request.URL, params)
	if err != nil {
		return err
	}

	request.URL = resolved
	return nil
}
