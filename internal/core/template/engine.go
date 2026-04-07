package template

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// templatePattern matches {{variableName}}
var templatePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// Substitute replaces all {{varName}} placeholders with values from vars map
func Substitute(cmd string, vars map[string]string) (string, error) {
	result := cmd

	// Find all template variables in the command
	matches := templatePattern.FindAllStringSubmatch(cmd, -1)
	if len(matches) == 0 {
		return cmd, nil
	}

	// Collect all variable names used
	usedVars := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			usedVars[match[1]] = true
		}
	}

	// Check for missing variables
	var missingVars []string
	for varName := range usedVars {
		if _, ok := vars[varName]; !ok {
			missingVars = append(missingVars, varName)
		}
	}

	if len(missingVars) > 0 {
		return "", fmt.Errorf("missing required variables: %s", strings.Join(missingVars, ", "))
	}

	// Replace all placeholders
	for varName := range usedVars {
		if replacement, ok := vars[varName]; ok {
			result = strings.ReplaceAll(result, "{{"+varName+"}}", replacement)
		}
	}

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

	// Check URL
	urlVars := ExtractVarNames(request.URL)
	for _, name := range urlVars {
		vars = append(vars, types.Var{
			Name:    name,
			Pattern: "", // Pattern not known from template alone
			Example: "",
		})
	}

	// Check body
	bodyVars := ExtractVarNames(request.Body)
	for _, name := range bodyVars {
		// Check if already added
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
