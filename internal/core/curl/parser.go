package curl

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

var (
	// Flag patterns for curl command parsing
	flagPatterns = map[string]*regexp.Regexp{
		"method":     regexp.MustCompile(`(?i)-X|--request\s+(\S+)`),
		"header":     regexp.MustCompile(`(?i)-H|--header\s+['"]([^'"]+)['"]`),
		"data":       regexp.MustCompile(`(?i)-d|--data|--data-raw\s+['"]([^'"]+)['"]`),
		"url":        regexp.MustCompile(`(?:^|\s)(https?://\S+)`),
	}
)

// ParseCurl parses a curl command string into a structured ParsedCurl
func ParseCurl(cmd string) (*types.ParsedCurl, error) {
	// Normalize whitespace
	cmd = strings.Join(strings.Fields(cmd), " ")

	result := &types.ParsedCurl{
		Headers: make(map[string]string),
		Method:  "GET",
	}

	// Parse method (-X or --request)
	if matches := flagPatterns["method"].FindAllStringSubmatch(cmd, -1); len(matches) > 0 {
		// Get the last method definition
		lastMatch := matches[len(matches)-1]
		if len(lastMatch) >= 2 {
			result.Method = strings.ToUpper(lastMatch[1])
		} else if len(lastMatch) >= 1 {
			result.Method = strings.ToUpper(lastMatch[1])
		}
	}

	// Parse headers (-H or --header)
	headerPattern := regexp.MustCompile(`(?i)-H\s+['"]([^'"]+)['"]`)
	for _, match := range headerPattern.FindAllStringSubmatch(cmd, -1) {
		if len(match) >= 2 {
			header := match[1]
			if idx := strings.Index(header, ":"); idx != -1 {
				key := strings.TrimSpace(header[:idx])
				value := strings.TrimSpace(header[idx+1:])
				result.Headers[key] = value
			}
		}
	}

	// Also check for --header without quotes
	headerPattern2 := regexp.MustCompile(`(?i)--header\s+(\S+)`)
	for _, match := range headerPattern2.FindAllStringSubmatch(cmd, -1) {
		if len(match) >= 2 {
			header := match[1]
			if idx := strings.Index(header, ":"); idx != -1 {
				key := strings.TrimSpace(header[:idx])
				value := strings.TrimSpace(header[idx+1:])
				result.Headers[key] = value
			}
		}
	}

	// Parse body (-d, --data, --data-raw)
	dataPattern := regexp.MustCompile(`(?i)-d\s+['"]([^'"]+)['"]`)
	for _, match := range dataPattern.FindAllStringSubmatch(cmd, -1) {
		if len(match) >= 2 {
			result.Body = match[1]
			// If body is present and method is default GET, switch to POST
			if result.Method == "GET" {
				result.Method = "POST"
			}
			break // Use first body data
		}
	}

	// Also check for --data without quotes
	dataPattern2 := regexp.MustCompile(`(?i)--data\s+['"]([^'"]+)['"]`)
	for _, match := range dataPattern2.FindAllStringSubmatch(cmd, -1) {
		if len(match) >= 2 {
			result.Body = match[1]
			if result.Method == "GET" {
				result.Method = "POST"
			}
			break
		}
	}

	// Parse URL
	urlPattern := regexp.MustCompile(`(?:^|\s)(https?://\S+)`)
	if matches := urlPattern.FindAllStringSubmatch(cmd, -1); len(matches) > 0 {
		result.URL = matches[len(matches)-1][1] // Take last URL found
	}

	// Validate URL
	if result.URL == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	// Normalize URL
	result.URL = NormalizeURL(result.URL)

	return result, nil
}

// NormalizeURL removes trailing slashes and normalizes the URL
func NormalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Remove trailing slash from path
	if parsed.Path != "" && strings.HasSuffix(parsed.Path, "/") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}

	// Reconstruct URL
	result := parsed.Scheme + "://" + parsed.Host + parsed.Path
	if parsed.RawQuery != "" {
		result += "?" + parsed.RawQuery
	}

	return result
}
