package assertions

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/formatter"
)

// Assertion represents a declarative assertion to evaluate against a response.
type Assertion struct {
	Field    string // Field to check: "status", "headers.{name}", "body", "body.{jsonpath}", "time", "size"
	Op       string // Operator: "=", "!=", "<", ">", "<=", ">=", "contains", "not_contains", "matches", "exists"
	Value    string // Expected value (string for comparison)
	jsonPath string // Parsed JSONPath if field starts with "body."
}

// Result represents the result of evaluating an assertion.
type Result struct {
	Assertion Assertion // The assertion that was evaluated
	Passed    bool      // Whether the assertion passed
	Actual    string    // Actual value found
	Expected  string    // Expected value
	Message   string    // Human-readable message
}

// Evaluator evaluates assertions against HTTP responses.
type Evaluator struct{}

// NewEvaluator creates a new assertion evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate runs all assertions against a response and returns all results.
func (e *Evaluator) Evaluate(resp *client.Response, assertions []Assertion) []Result {
	results := make([]Result, 0, len(assertions))
	for _, assertion := range assertions {
		result := e.evaluateSingle(resp, assertion)
		results = append(results, result)
	}
	return results
}

// evaluateSingle evaluates a single assertion.
func (e *Evaluator) evaluateSingle(resp *client.Response, assertion Assertion) Result {
	actual, err := e.extractField(resp, assertion.Field)
	if err != nil {
		return Result{
			Assertion: assertion,
			Passed:    false,
			Actual:    "",
			Expected:  assertion.Value,
			Message:   fmt.Sprintf("failed to extract field %q: %v", assertion.Field, err),
		}
	}

	passed := e.compare(actual, assertion.Op, assertion.Value)
	msg := e.buildMessage(assertion, actual, passed)
	return Result{
		Assertion: assertion,
		Passed:    passed,
		Actual:    actual,
		Expected:  assertion.Value,
		Message:   msg,
	}
}

// extractField extracts the value of a field from the response.
func (e *Evaluator) extractField(resp *client.Response, field string) (string, error) {
	// Handle headers.{name} syntax
	if strings.HasPrefix(field, "headers.") {
		headerName := strings.TrimPrefix(field, "headers.")
		// Remove quotes if present (from TOML parsing)
		headerName = strings.Trim(headerName, `"'`)
		return e.extractHeader(resp, headerName)
	}

	// Handle body.{jsonpath} syntax
	if strings.HasPrefix(field, "body.") {
		jsonPath := strings.TrimPrefix(field, "body.")
		// Remove quotes if present (from TOML parsing)
		jsonPath = strings.Trim(jsonPath, `"'`)
		if !strings.HasPrefix(jsonPath, "$") {
			jsonPath = "$." + jsonPath
		}
		return e.extractJSONPath(resp.Body, jsonPath)
	}

	// Handle simple body field
	if field == "body" {
		return string(resp.Body), nil
	}

	// Handle status field (also accept "status_code" as alias)
	if field == "status" || field == "status_code" {
		return strconv.Itoa(resp.StatusCode), nil
	}

	// Handle time field (in milliseconds)
	if field == "time" {
		return strconv.FormatInt(resp.Duration.Milliseconds(), 10), nil
	}

	// Handle size field
	if field == "size" {
		return strconv.FormatInt(resp.Size, 10), nil
	}

	return "", fmt.Errorf("unknown field: %q", field)
}

// extractHeader extracts a header value from the response.
func (e *Evaluator) extractHeader(resp *client.Response, name string) (string, error) {
	// Case-insensitive header lookup
	for k, values := range resp.Headers {
		if strings.EqualFold(k, name) && len(values) > 0 {
			return values[0], nil
		}
	}
	// For exists/non-empty checks, return empty string if not found
	return "", nil
}

// extractJSONPath extracts a value from JSON body using JSONPath.
func (e *Evaluator) extractJSONPath(body []byte, path string) (string, error) {
	result, err := formatter.FilterJSON(body, path)
	if err != nil {
		return "", err
	}
	// FilterJSON returns pretty-printed JSON, extract just the value
	// For simple values, demarshal and return the raw value
	if result != "" {
		// Try to extract as simple value
		var val interface{}
		if err := json.Unmarshal([]byte(result), &val); err == nil {
			switch v := val.(type) {
			case string:
				return v, nil
			case float64:
				// Check if it's an integer
				if v == float64(int64(v)) {
					return strconv.FormatInt(int64(v), 10), nil
				}
				return strconv.FormatFloat(v, 'f', -1, 64), nil
			case bool:
				return strconv.FormatBool(v), nil
			case nil:
				return "", nil
			}
		}
	}
	return result, nil
}

// compare compares actual value with expected using the operator.
func (e *Evaluator) compare(actual, op, expected string) bool {
	switch op {
	case "=", "equals":
		return actual == expected
	case "!=", "not_equals":
		return actual != expected
	case "<":
		return e.compareNumeric(actual, expected, func(a, b float64) bool { return a < b })
	case ">":
		return e.compareNumeric(actual, expected, func(a, b float64) bool { return a > b })
	case "<=":
		return e.compareNumeric(actual, expected, func(a, b float64) bool { return a <= b })
	case ">=":
		return e.compareNumeric(actual, expected, func(a, b float64) bool { return a >= b })
	case "contains":
		return strings.Contains(actual, expected)
	case "not_contains":
		return !strings.Contains(actual, expected)
	case "matches":
		matched, err := regexp.MatchString(expected, actual)
		if err != nil {
			return false
		}
		return matched
	case "exists":
		// For exists: actual value must be non-empty
		return actual != ""
	default:
		return false
	}
}

// compareNumeric compares two numeric values using the provided comparison function.
func (e *Evaluator) compareNumeric(actual, expected string, cmp func(float64, float64) bool) bool {
	// Try to parse as integers first
	actualInt, errA := strconv.ParseInt(actual, 10, 64)
	expectedInt, errE := strconv.ParseInt(expected, 10, 64)
	if errA == nil && errE == nil {
		return cmp(float64(actualInt), float64(expectedInt))
	}

	// Try as floats
	actualFloat, errA := strconv.ParseFloat(actual, 64)
	expectedFloat, errE := strconv.ParseFloat(expected, 64)
	if errA == nil && errE == nil {
		return cmp(actualFloat, expectedFloat)
	}

	// Fall back to string comparison for duration strings like "500ms"
	if strings.HasSuffix(actual, "ms") && strings.HasSuffix(expected, "ms") {
		actualMs, errA := strconv.ParseInt(strings.TrimSuffix(actual, "ms"), 10, 64)
		expectedMs, errE := strconv.ParseInt(strings.TrimSuffix(expected, "ms"), 10, 64)
		if errA == nil && errE == nil {
			return cmp(float64(actualMs), float64(expectedMs))
		}
	}

	return false
}

// buildMessage creates a human-readable message for the result.
func (e *Evaluator) buildMessage(assertion Assertion, actual string, passed bool) string {
	if passed {
		return fmt.Sprintf("PASS: %s %s %s (actual: %s)", assertion.Field, assertion.Op, assertion.Value, actual)
	}

	if actual == "" && assertion.Op == "exists" {
		return fmt.Sprintf("FAIL: %s does not exist", assertion.Field)
	}

	return fmt.Sprintf("FAIL: %s %s %s (actual: %s)", assertion.Field, assertion.Op, assertion.Value, actual)
}

// Summary returns a summary of assertion results.
type Summary struct {
	Total   int
	Passed  int
	Failed  int
	Results []Result
}

// Summarize returns a summary of the assertion results.
func Summarize(results []Result) Summary {
	s := Summary{
		Total:   len(results),
		Passed:  0,
		Failed:  0,
		Results: results,
	}
	for _, r := range results {
		if r.Passed {
			s.Passed++
		} else {
			s.Failed++
		}
	}
	return s
}
