package runner

import (
	"fmt"
	"strings"
)

type runIfCondition struct {
	Variable string
	Operator string
	Value    string
}

func parseRunIf(expr string) (runIfCondition, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return runIfCondition{}, fmt.Errorf("empty run_if expression")
	}

	operator := ""
	operatorIndex := -1
	for _, candidate := range []string{"==", "!="} {
		if idx := strings.Index(expr, candidate); idx >= 0 {
			operator = candidate
			operatorIndex = idx
			break
		}
	}
	if operator == "" {
		return runIfCondition{}, unsupportedRunIfError(expr)
	}

	left := strings.TrimSpace(expr[:operatorIndex])
	right := strings.TrimSpace(expr[operatorIndex+len(operator):])
	if left == "" || right == "" {
		return runIfCondition{}, unsupportedRunIfError(expr)
	}
	if !isRunIfVariable(left) {
		return runIfCondition{}, unsupportedRunIfError(expr)
	}

	value, err := parseRunIfValue(right)
	if err != nil {
		return runIfCondition{}, unsupportedRunIfError(expr)
	}

	return runIfCondition{Variable: left, Operator: operator, Value: value}, nil
}

func evaluateRunIf(expr string, vars map[string]string) (bool, error) {
	condition, err := parseRunIf(expr)
	if err != nil {
		return false, err
	}
	return evaluateRunIfCondition(condition, vars), nil
}

func evaluateRunIfCondition(condition runIfCondition, vars map[string]string) bool {
	actual := ""
	if vars != nil {
		actual = vars[condition.Variable]
	}

	switch condition.Operator {
	case "==":
		return actual == condition.Value
	case "!=":
		return actual != condition.Value
	default:
		return false
	}
}

func parseRunIfValue(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("empty run_if value")
	}
	if len(raw) >= 2 {
		if (raw[0] == '\'' && raw[len(raw)-1] == '\'') || (raw[0] == '"' && raw[len(raw)-1] == '"') {
			return raw[1 : len(raw)-1], nil
		}
		if raw[0] == '\'' || raw[0] == '"' || raw[len(raw)-1] == '\'' || raw[len(raw)-1] == '"' {
			return "", fmt.Errorf("mismatched run_if quotes")
		}
	}
	if strings.ContainsAny(raw, " \t\n\r") {
		return "", fmt.Errorf("unquoted run_if value contains whitespace")
	}
	return raw, nil
}

func isRunIfVariable(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if !(r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
				return false
			}
			continue
		}
		if !(r == '_' || r == '-' || r == '.' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return false
		}
	}
	return true
}

func unsupportedRunIfError(expr string) error {
	return fmt.Errorf("unsupported run_if expression %q: supported syntax is VAR == VALUE or VAR != VALUE (VALUE may be wrapped in single or double quotes, for example VAR == 'beta' or VAR != \"\")", expr)
}
