package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/wI2L/jsondiff"
)

const (
	green = "\033[32m"
	red   = "\033[31m"
	reset = "\033[0m"
)

func DiffJSON(a, b []byte) (string, error) {
	var objA, objB interface{}
	if err := json.Unmarshal(a, &objA); err != nil {
		return "", fmt.Errorf("invalid JSON in first argument: %w", err)
	}
	if err := json.Unmarshal(b, &objB); err != nil {
		return "", fmt.Errorf("invalid JSON in second argument: %w", err)
	}

	patch, err := jsondiff.CompareJSON(a, b)
	if err != nil {
		return "", fmt.Errorf("failed to compute JSON diff: %w", err)
	}

	if len(patch) == 0 {
		return "No differences found (JSONs are semantically identical)", nil
	}

	return formatPatch(patch), nil
}

func formatPatch(patch jsondiff.Patch) string {
	var buf bytes.Buffer
	buf.WriteString("JSON Diff (RFC 6902):\n")

	for _, op := range patch {
		switch op.Type {
		case "add":
			buf.WriteString(fmt.Sprintf("  %s+ %s%s %s%v%s\n",
				green, reset, op.Path, green, op.Value, reset))
		case "remove":
			buf.WriteString(fmt.Sprintf("  %s- %s%s%s\n",
				red, reset, op.Path, reset))
		case "replace":
			buf.WriteString(fmt.Sprintf("  %s~ %s%s  (%sold → %s%v%s)\n",
				red, reset, op.Path, red, green, op.Value, reset))
		default:
			buf.WriteString(fmt.Sprintf("  %s: %s %v\n", op.Type, op.Path, op.Value))
		}
	}

	return buf.String()
}

type patchOp struct {
	Op       string `json:"op"`
	Path     string `json:"path"`
	Value    string `json:"value,omitempty"`
	OldValue string `json:"oldValue,omitempty"`
}

func DiffText(a, b []byte) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(a), string(b), true)

	hasDiff := false
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			hasDiff = true
			break
		}
	}
	if !hasDiff {
		return "No differences found (texts are identical)"
	}

	var buf bytes.Buffer
	buf.WriteString("Text Diff:\n")

	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			buf.WriteString(fmt.Sprintf("%s%s%s", green, d.Text, reset))
		case diffmatchpatch.DiffDelete:
			buf.WriteString(fmt.Sprintf("%s%s%s", red, d.Text, reset))
		case diffmatchpatch.DiffEqual:
			buf.WriteString(d.Text)
		}
	}

	return buf.String()
}

func DiffResponses(histA, histB types.ExecutionHistory) (string, error) {
	result, err := DiffJSON([]byte(histA.Response), []byte(histB.Response))
	if err == nil {
		return formatResponseHeader(histA, histB, result), nil
	}

	result = DiffText([]byte(histA.Response), []byte(histB.Response))
	return formatResponseHeader(histA, histB, result), nil
}

func formatResponseHeader(histA, histB types.ExecutionHistory, diffOutput string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("┌─ Response Diff ─────────────────────────────────────┐\n"))
	buf.WriteString(fmt.Sprintf("│ Response A: status=%d, id=%s                    │\n", histA.StatusCode, histA.ID))
	buf.WriteString(fmt.Sprintf("│ Response B: status=%d, id=%s                    │\n", histB.StatusCode, histB.ID))
	buf.WriteString(fmt.Sprintf("├─────────────────────────────────────────────────────┤\n"))
	buf.WriteString(fmt.Sprintf("%s\n", diffOutput))
	buf.WriteString(fmt.Sprintf("└─────────────────────────────────────────────────────┘"))
	return buf.String()
}

func NormalizeJSON(data []byte) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	normalized := sortKeys(obj)
	return json.Marshal(normalized)
}

func sortKeys(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		result := make(map[string]interface{})
		for _, k := range keys {
			result[k] = sortKeys(val[k])
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = sortKeys(v)
		}
		return result
	default:
		return val
	}
}

func GetDiffStats(a, b []byte) (added, removed, changed int) {
	patch, err := jsondiff.CompareJSON(a, b)
	if err != nil {
		return -1, -1, -1
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return -1, -1, -1
	}

	var ops []patchOp
	if err := json.Unmarshal(patchBytes, &ops); err != nil {
		return -1, -1, -1
	}

	for _, op := range ops {
		switch op.Op {
		case "add":
			added++
		case "remove":
			removed++
		case "replace":
			changed++
		}
	}
	return
}

func ColorizePatch(patch jsondiff.Patch) string {
	return formatPatch(patch)
}

func IsValidJSON(data []byte) bool {
	var v interface{}
	return json.Unmarshal(data, &v) == nil
}

func CombineDiffs(diffs ...string) string {
	var buf bytes.Buffer
	for _, d := range diffs {
		if d != "" {
			buf.WriteString(d)
			buf.WriteString("\n")
		}
	}
	return strings.TrimSpace(buf.String())
}

// DiffResult is a structured diff of two response bodies, from a (before) to
// b (after).
type DiffResult struct {
	// Format is "json" when both bodies are valid JSON, otherwise "text".
	Format    string `json:"format"`
	Identical bool   `json:"identical"`
	// Changes are RFC 6902 operations turning a into b (json format).
	Changes []JSONChange `json:"changes,omitempty"`
	// Text is the character diff, equal runs included (text format).
	Text []TextChange `json:"text,omitempty"`
}

// JSONChange is one RFC 6902 operation, with the replaced or removed value.
type JSONChange struct {
	Op       string `json:"op"`
	Path     string `json:"path"`
	From     string `json:"from,omitempty"`
	OldValue any    `json:"old_value,omitempty"`
	Value    any    `json:"value,omitempty"`
}

// TextChange is one run of a text diff: "equal", "insert" or "delete".
type TextChange struct {
	Op   string `json:"op"`
	Text string `json:"text"`
}

// Diff compares two response bodies structurally.
func Diff(a, b []byte) DiffResult {
	if json.Valid(a) && json.Valid(b) {
		if patch, err := jsondiff.CompareJSON(a, b); err == nil {
			result := DiffResult{Format: "json", Identical: len(patch) == 0}
			for _, op := range patch {
				result.Changes = append(result.Changes, JSONChange{Op: op.Type, Path: op.Path, From: op.From, OldValue: op.OldValue, Value: op.Value})
			}
			return result
		}
	}
	result := DiffResult{Format: "text", Identical: true}
	dmp := diffmatchpatch.New()
	for _, d := range dmp.DiffMain(string(a), string(b), true) {
		op := "equal"
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			op = "insert"
			result.Identical = false
		case diffmatchpatch.DiffDelete:
			op = "delete"
			result.Identical = false
		}
		result.Text = append(result.Text, TextChange{Op: op, Text: d.Text})
	}
	return result
}
