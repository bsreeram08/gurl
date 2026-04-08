package formatter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func FormatTable(data interface{}) string {
	if data == nil {
		return ""
	}

	switch v := data.(type) {
	case []interface{}:
		return formatTableArray(v)
	case map[string]interface{}:
		return formatKeyValueTable(v)
	default:
		return ""
	}
}

func formatTableArray(arr []interface{}) string {
	if len(arr) == 0 {
		return ""
	}

	if _, ok := arr[0].(map[string]interface{}); !ok {
		return ""
	}

	keySet := make(map[string]bool)
	for _, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			for k := range obj {
				keySet[k] = true
			}
		}
	}

	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		return ""
	}

	colWidths := make(map[string]int)
	for _, k := range keys {
		colWidths[k] = len(k)
	}

	for _, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			for _, k := range keys {
				if val, exists := obj[k]; exists {
					valStr := formatValue(val)
					if len(valStr) > colWidths[k] {
						colWidths[k] = len(valStr)
					}
				}
			}
		}
	}

	separator := buildArraySeparator(keys, colWidths)

	var builder strings.Builder
	builder.WriteString(separator)

	builder.WriteString("│")
	for i, k := range keys {
		width := colWidths[k]
		if i > 0 {
			builder.WriteString("│")
		}
		builder.WriteString(fmt.Sprintf(" %-*s ", width, k))
	}
	builder.WriteString("│\n")
	builder.WriteString(separator)

	for _, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			builder.WriteString("│")
			for i, k := range keys {
				width := colWidths[k]
				if i > 0 {
					builder.WriteString("│")
				}
				valStr := ""
				if val, exists := obj[k]; exists {
					valStr = formatValue(val)
				}
				builder.WriteString(fmt.Sprintf(" %-*s ", width, valStr))
			}
			builder.WriteString("│\n")
		}
	}

	builder.WriteString(separator)
	return builder.String()
}

func formatKeyValueTable(obj map[string]interface{}) string {
	if len(obj) == 0 {
		return ""
	}

	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxKeyWidth := 0
	for _, k := range keys {
		if len(k) > maxKeyWidth {
			maxKeyWidth = len(k)
		}
	}

	var builder strings.Builder
	builder.WriteString("╭─")
	builder.WriteString(strings.Repeat("─", maxKeyWidth))
	builder.WriteString("─┬─")
	builder.WriteString("───────────────────────────────────╮\n")

	for _, k := range keys {
		val := obj[k]
		valStr := formatValue(val)
		builder.WriteString(fmt.Sprintf("│ %-*s │ %s │\n", maxKeyWidth, k, valStr))
	}

	builder.WriteString("╰─")
	builder.WriteString(strings.Repeat("─", maxKeyWidth))
	builder.WriteString("─┴─")
	builder.WriteString(strings.Repeat("─", 35))
	builder.WriteString("╯")

	return builder.String()
}

func buildArraySeparator(keys []string, colWidths map[string]int) string {
	var builder strings.Builder
	builder.WriteString("├")
	for i, k := range keys {
		width := colWidths[k]
		if i > 0 {
			builder.WriteString("┼")
		}
		builder.WriteString(strings.Repeat("─", width+2))
	}
	builder.WriteString("┤\n")
	return builder.String()
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%v", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	case []interface{}, map[string]interface{}:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func FormatTableFromBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}

	return FormatTable(v)
}
