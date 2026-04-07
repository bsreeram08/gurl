package curl

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/sreeram/gurl/pkg/types"
)

func expandANSICQuotes(s string) string {
	if strings.HasPrefix(s, "$'") && strings.HasSuffix(s, "'") {
		s = s[2 : len(s)-1]
	} else if strings.HasPrefix(s, "$") {
		s = s[1:]
	} else {
		return s
	}

	var result strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
				i++
			case 't':
				result.WriteByte('\t')
				i++
			case 'r':
				result.WriteByte('\r')
				i++
			case '\\':
				result.WriteByte('\\')
				i++
			case '\'':
				result.WriteByte('\'')
				i++
			case '"':
				result.WriteByte('"')
				i++
			default:
				result.WriteByte(s[i])
			}
		} else {
			result.WriteByte(s[i])
		}
	}
	return result.String()
}

func ParseCurl(cmd string) (*types.ParsedCurl, error) {
	tokens, err := shellquote.Split(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse curl command: %w", err)
	}

	result := &types.ParsedCurl{
		Headers: make(map[string]string),
		Method:  "GET",
	}

	i := 0
	if len(tokens) > 0 && tokens[0] == "curl" {
		i = 1
	}

	for i < len(tokens) {
		token := tokens[i]

		switch {
		case token == "-X" || token == "--request":
			if i+1 < len(tokens) {
				i++
				result.Method = strings.ToUpper(tokens[i])
			}

		case token == "-H" || token == "--header":
			if i+1 < len(tokens) {
				i++
				if idx := strings.Index(tokens[i], ":"); idx != -1 {
					key := strings.TrimSpace(tokens[i][:idx])
					value := strings.TrimSpace(tokens[i][idx+1:])
					result.Headers[key] = value
				}
			}

		case token == "-d" || token == "--data" || token == "--data-raw" || token == "--data-urlencode":
			if i+1 < len(tokens) {
				i++
				val := tokens[i]
				if strings.HasPrefix(val, "$") {
					val = expandANSICQuotes(val)
				}
				result.Body = val
				if result.Method == "GET" {
					result.Method = "POST"
				}
			}

		case token == "-F":
			if result.Method == "GET" {
				result.Method = "POST"
			}
			if i+1 < len(tokens) {
				i++
			}

		case token == "-u" || token == "--user":
			if i+1 < len(tokens) {
				i++
			}

		case token == "-b" || token == "--cookie":
			if i+1 < len(tokens) {
				i++
			}
		case token == "-c" || token == "--cookie-jar":
			if i+1 < len(tokens) {
				i++
			}

		case token == "--max-redirs":
			if i+1 < len(tokens) {
				i++
			}
		case token == "--connect-timeout":
			if i+1 < len(tokens) {
				i++
			}

		case token == "--compressed":
		case token == "-L" || token == "--location":
		case token == "-k" || token == "--insecure":

		case strings.HasPrefix(token, "http://") || strings.HasPrefix(token, "https://"):
			result.URL = token

		case !strings.HasPrefix(token, "-") && !strings.Contains(token, "=") && token != "":
			if strings.Contains(token, ".") || strings.HasPrefix(token, "localhost") {
				result.URL = token
			}
		}

		i++
	}

	if result.URL == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	result.URL = NormalizeURL(result.URL)

	return result, nil
}

// NormalizeURL removes trailing slashes and normalizes the URL
func NormalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	shouldTrimSlash := false
	if parsed.Path != "" && strings.HasSuffix(parsed.Path, "/") {
		if len(parsed.Path) > 1 {
			shouldTrimSlash = true
		} else if parsed.Fragment == "" {
			shouldTrimSlash = true
		}
	}

	if shouldTrimSlash {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}

	result := parsed.Scheme + "://" + parsed.Host + parsed.Path
	if parsed.RawQuery != "" {
		result += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		result += "#" + parsed.Fragment
	}

	return result
}
