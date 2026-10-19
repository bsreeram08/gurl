package template

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// templatePattern matches {{variableName}}
var templatePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// substitutePattern matches either a quoted typed placeholder ("{{name:num}}",
// whose surrounding quotes are consumed) or any plain {{...}} placeholder.
var substitutePattern = regexp.MustCompile(`"\{\{([^{}"]+:(?:num|bool|raw))\}\}"|\{\{([^}]+)\}\}`)

// jsonNumberPattern matches a JSON number literal.
var jsonNumberPattern = regexp.MustCompile(`^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$`)

// Placeholder types supported via the {{name:type}} suffix.
const (
	typeNum  = "num"
	typeBool = "bool"
	typeRaw  = "raw"
)

// parsePlaceholder splits the inner text of a placeholder into its base
// variable name and optional type suffix (num, bool, raw). An unknown suffix
// is treated as part of the name.
func parsePlaceholder(inner string) (name, typ string) {
	if i := strings.LastIndex(inner, ":"); i > 0 {
		switch inner[i+1:] {
		case typeNum, typeBool, typeRaw:
			return inner[:i], inner[i+1:]
		}
	}
	return inner, ""
}

// BaseName returns the variable name of a placeholder's inner text with any
// type suffix removed, e.g. "amount:num" -> "amount".
func BaseName(inner string) string {
	name, _ := parsePlaceholder(inner)
	return name
}

// resolver performs single-pass placeholder substitution against vars.
// Substituted values are never re-expanded: a value that itself contains a
// placeholder is an error in strict mode, so untrusted values (e.g. extracted
// from a response) cannot pull other variables such as secrets into a request.
type resolver struct {
	vars    map[string]string
	strict  bool // strict reports missing vars and value errors; otherwise best-effort
	missing []string
	errs    []string
	seen    map[string]bool
}

func (r *resolver) addOnce(list *[]string, item string) {
	if r.seen == nil {
		r.seen = make(map[string]bool)
	}
	if !r.seen[item] {
		r.seen[item] = true
		*list = append(*list, item)
	}
}

func (r *resolver) substitute(s string) string {
	return substitutePattern.ReplaceAllStringFunc(s, func(match string) string {
		quoted := match[0] == '"'
		inner := match[2 : len(match)-2]
		if quoted {
			inner = match[3 : len(match)-3]
		}
		name, typ := parsePlaceholder(inner)

		val, ok := r.vars[name]
		if !ok {
			if r.strict {
				r.addOnce(&r.missing, name)
			}
			return match
		}
		if !r.strict {
			return applyTypeOrKeep(match, val, typ)
		}

		if nested := templatePattern.FindAllString(val, -1); len(nested) > 0 {
			for _, p := range nested {
				r.addOnce(&r.errs, fmt.Sprintf("variable %q contains unresolved placeholder %s", name, p))
			}
			return match
		}

		out, err := applyType(name, val, typ)
		if err != nil {
			r.addOnce(&r.errs, err.Error())
			return match
		}
		return out
	})
}

// applyType validates val against typ and returns the text to insert.
// typeRaw and untyped values are inserted verbatim.
func applyType(name, val, typ string) (string, error) {
	switch typ {
	case typeNum:
		if v := strings.TrimSpace(val); jsonNumberPattern.MatchString(v) {
			return v, nil
		}
		return "", fmt.Errorf("variable %q must be a number", name)
	case typeBool:
		if v := strings.TrimSpace(val); v == "true" || v == "false" {
			return v, nil
		}
		return "", fmt.Errorf("variable %q must be true or false", name)
	}
	return val, nil
}

// applyTypeOrKeep is the best-effort form of applyType: on validation
// failure the original placeholder text is kept.
func applyTypeOrKeep(match, val, typ string) string {
	out, err := applyType("", val, typ)
	if err != nil {
		return match
	}
	return out
}

// Substitute replaces all {{varName}} placeholders with values from vars map
// in a single pass. Typed placeholders {{name:num}}, {{name:bool}} and
// {{name:raw}} validate the value and, when wrapped in double quotes
// ("{{name:num}}"), replace the quotes too so JSON bodies stay valid.
// Substituted values are not expanded again; a value that contains a
// placeholder is reported as an error. Returns an error listing every missing
// variable and every invalid value, in order of appearance.
func Substitute(cmd string, vars map[string]string) (string, error) {
	r := &resolver{vars: vars, strict: true}
	result := r.substitute(cmd)

	var problems []string
	if len(r.missing) > 0 {
		problems = append(problems, "missing required variables: "+strings.Join(r.missing, ", "))
	}
	problems = append(problems, r.errs...)
	if len(problems) > 0 {
		return "", errors.New(strings.Join(problems, "; "))
	}
	return result, nil
}

// ResolveKnown is a best-effort, single-pass Substitute: known variables
// (including typed placeholders) are replaced, unknown placeholders and values
// that fail type validation are left intact, and no error is returned.
func ResolveKnown(cmd string, vars map[string]string) string {
	r := &resolver{vars: vars}
	return r.substitute(cmd)
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
			varName := BaseName(match[1])
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
			varName := BaseName(match[1])
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
