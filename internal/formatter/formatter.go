package formatter

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strings"
)

// FormatOptions controls formatting behavior
type FormatOptions struct {
	Indent   string // Indent string (e.g., "  " or "\t")
	Color    bool   // Enable ANSI color codes
	MaxWidth int    // Max line width (0 = no limit)
}

// Format auto-detects content type and dispatches to appropriate formatter
// Uses switch on content-type for dispatch (NOT if-else chains)
func Format(body []byte, contentType string, opts FormatOptions) string {
	if len(body) == 0 {
		return ""
	}

	switch {
	case strings.Contains(contentType, "json"):
		return FormatJSON(body, opts)
	case strings.Contains(contentType, "xml"):
		return FormatXML(body, opts)
	case strings.Contains(contentType, "html"):
		return FormatHTML(body, opts)
	default:
		// Raw passthrough for unknown types
		return string(body)
	}
}

// FormatJSON formats JSON with optional syntax highlighting
func FormatJSON(body []byte, opts FormatOptions) string {
	if len(body) == 0 {
		return ""
	}

	// Use json.MarshalIndent for formatting
	formatted, err := json.MarshalIndent(json.RawMessage(body), "", opts.Indent)
	if err != nil {
		// Invalid JSON - return raw input
		return string(body)
	}

	if opts.Color {
		return colorizeJSON(string(formatted))
	}
	return string(formatted)
}

// colorizeJSON applies ANSI colors to JSON string
func colorizeJSON(s string) string {
	var buf bytes.Buffer

	type state int
	const (
		stateKey state = iota
		stateAfterKey
		stateAfterColon
	)

	currentState := stateKey
	inString := false
	stringStart := -1

	i := 0
	for i < len(s) {
		c := s[i]

		if c == '"' {
			// Count preceding backslashes
			backslashes := 0
			for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
				backslashes++
			}
			// If even number of backslashes (including 0), the quote is unescaped
			if backslashes%2 == 0 {
				if !inString {
					inString = true
					stringStart = i
				} else {
					inString = false
					strContent := s[stringStart : i+1]
					switch currentState {
					case stateKey:
						buf.WriteString(Cyan + strContent + Reset)
					case stateAfterColon:
						buf.WriteString(Green + strContent + Reset)
					default:
						buf.WriteString(strContent)
					}
				}
			}
			buf.WriteByte(c)
			i++
			continue
		}

		if inString {
			buf.WriteByte(c)
			i++
			continue
		}

		switch currentState {
		case stateKey:
			if c == ':' {
				currentState = stateAfterColon
			}
		case stateAfterColon:
			if i+4 <= len(s) && s[i:i+4] == "null" {
				buf.WriteString(Red + "null" + Reset)
				i += 4
				currentState = stateKey
				continue
			}
			if i+4 <= len(s) && s[i:i+4] == "true" {
				buf.WriteString(Magenta + "true" + Reset)
				i += 4
				currentState = stateKey
				continue
			}
			if i+5 <= len(s) && s[i:i+5] == "false" {
				buf.WriteString(Magenta + "false" + Reset)
				i += 5
				currentState = stateKey
				continue
			}
			if isNumberStart(c) {
				start := i
				for i < len(s) && isNumberChar(s[i]) {
					i++
				}
				buf.WriteString(Yellow + s[start:i] + Reset)
				currentState = stateKey
				continue
			}
			if c == ',' || c == '}' || c == ']' {
				currentState = stateKey
			}
		case stateAfterKey:
			if c == ':' {
				currentState = stateAfterColon
			}
		}

		if c == ':' && currentState == stateAfterColon {
			// already handled above
		}
		buf.WriteByte(c)
		i++
	}

	return buf.String()
}

func isNumberStart(c byte) bool {
	return (c >= '0' && c <= '9') || c == '-'
}

func isNumberChar(c byte) bool {
	return (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-'
}

// FormatXML formats XML with optional syntax highlighting
func FormatXML(body []byte, opts FormatOptions) string {
	if len(body) == 0 {
		return ""
	}

	// Use xml.Encoder for proper indentation
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", opts.Indent)

	decoder := xml.NewDecoder(bytes.NewReader(body))

	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return string(body)
		}

		switch t := token.(type) {
		case xml.StartElement:
			encoder.EncodeToken(t)
		case xml.EndElement:
			encoder.EncodeToken(t)
		case xml.CharData:
			encoder.EncodeToken(t)
		case xml.Comment:
			encoder.EncodeToken(t)
		case xml.Directive:
			encoder.EncodeToken(t)
		case xml.ProcInst:
			encoder.EncodeToken(t)
		default:
			encoder.EncodeToken(t)
		}
	}
	encoder.Flush()

	output := buf.String()

	if opts.Color {
		return colorizeXML(output)
	}
	return output
}

// colorizeXML applies ANSI colors to XML string
func colorizeXML(s string) string {
	var buf bytes.Buffer

	re := regexp.MustCompile(`(<[^>]+>)`)
	parts := re.Split(s, -1)
	matches := re.FindAllStringSubmatchIndex(s, -1)

	pi := 0
	for i, match := range matches {
		if pi < len(parts) {
			buf.WriteString(parts[i])
		}

		tag := s[match[0]:match[1]]
		if strings.HasPrefix(tag, "<?") || strings.HasPrefix(tag, "<!") {
			buf.WriteString(tag)
		} else {
			buf.WriteString(colorizeXMLTag(tag))
		}

		pi = match[1]
	}

	if pi < len(s) {
		buf.WriteString(s[pi:])
	}

	return buf.String()
}

// colorizeXMLTag colors parts of an XML tag
func colorizeXMLTag(tag string) string {
	if len(tag) < 2 {
		return tag
	}

	var buf bytes.Buffer
	buf.WriteByte('<')

	rest := tag[1 : len(tag)-1]
	if strings.HasPrefix(rest, "/") {
		buf.WriteString("/")
		rest = rest[1:]
	} else if strings.HasPrefix(rest, "![CDATA[") {
		buf.WriteString("![CDATA[")
		rest = rest[8:]
	} else if strings.HasPrefix(rest, "!--") {
		buf.WriteString("!--")
		rest = rest[3:]
	}

	spaceIdx := strings.Index(rest, " ")
	if spaceIdx == -1 {
		spaceIdx = len(rest)
	}

	buf.WriteString(Cyan + rest[:spaceIdx] + Reset)

	if spaceIdx < len(rest) {
		attrs := rest[spaceIdx:]
		attrRe := regexp.MustCompile(`([a-zA-Z:-]+)="([^"]*)"`)
		attrs = attrRe.ReplaceAllString(attrs, Yellow+"$1"+Reset+"="+Green+"\"$2\""+Reset)
		buf.WriteString(attrs)
	}

	buf.WriteByte('>')
	return buf.String()
}

// FormatHTML formats HTML with lightweight tag-aware indentation
func FormatHTML(body []byte, opts FormatOptions) string {
	if len(body) == 0 {
		return ""
	}

	input := string(body)
	var buf bytes.Buffer
	indentLevel := 0

	tagRegex := regexp.MustCompile(`(<[^>]+>)`)

	parts := tagRegex.Split(input, -1)
	matches := tagRegex.FindAllStringSubmatchIndex(input, -1)

	pi := 0
	for i, match := range matches {
		if pi < len(parts) && i < len(parts) {
			text := parts[i]
			buf.WriteString(text)
		}

		tag := input[match[0]:match[1]]
		isClosing := strings.HasPrefix(tag, "</")
		isSelfClosing := strings.HasSuffix(tag, "/>")
		isComment := strings.HasPrefix(tag, "<!--")
		isDoctype := strings.HasPrefix(tag, "<!DOCTYPE")

		if !isComment && !isDoctype && !isSelfClosing && !strings.HasPrefix(tag, "<?") {
			if isClosing {
				indentLevel--
				if indentLevel < 0 {
					indentLevel = 0
				}
			}
		}

		if !isComment {
			buf.WriteString(opts.Indent)
			for j := 0; j < indentLevel; j++ {
				buf.WriteString(opts.Indent)
			}
		} else {
			buf.WriteString(opts.Indent)
		}
		buf.WriteString(tag)
		buf.WriteByte('\n')

		if !isClosing && !isSelfClosing && !isComment && !isDoctype && !strings.HasPrefix(tag, "<?") {
			indentLevel++
		}

		pi = match[1]
	}

	if pi < len(input) {
		remaining := input[pi:]
		if strings.TrimSpace(remaining) != "" {
			buf.WriteString(remaining)
		}
	}

	output := buf.String()
	output = strings.TrimRight(output, "\n")

	if opts.Color {
		return colorizeHTML(output)
	}
	return output
}

// colorizeHTML applies ANSI colors to HTML string
func colorizeHTML(s string) string {
	var buf bytes.Buffer

	re := regexp.MustCompile(`(<[^>]+>)`)
	parts := re.Split(s, -1)
	matches := re.FindAllStringSubmatchIndex(s, -1)

	pi := 0
	for i, match := range matches {
		if pi < len(parts) {
			// Color text content green
			if parts[i] != "" {
				buf.WriteString(Green + parts[i] + Reset)
			}
		}

		tag := s[match[0]:match[1]]
		if strings.HasPrefix(tag, "<!--") || strings.HasPrefix(tag, "<!DOCTYPE") {
			buf.WriteString(tag)
		} else {
			buf.WriteString(colorizeHTMLTag(tag))
		}

		pi = match[1]
	}

	if pi < len(s) {
		remaining := s[pi:]
		if remaining != "" {
			buf.WriteString(Green + remaining + Reset)
		}
	}

	return buf.String()
}

// colorizeHTMLTag colors parts of an HTML tag
func colorizeHTMLTag(tag string) string {
	if len(tag) < 2 {
		return tag
	}

	var buf bytes.Buffer
	buf.WriteByte('<')

	rest := tag[1 : len(tag)-1]
	if strings.HasPrefix(rest, "/") {
		buf.WriteString("/")
		rest = rest[1:]
	}

	// Find first space or end
	spaceIdx := strings.Index(rest, " ")
	if spaceIdx == -1 {
		spaceIdx = len(rest)
	}

	// Color the tag name
	buf.WriteString(Cyan + rest[:spaceIdx] + Reset)

	// Color attributes if any
	if spaceIdx < len(rest) {
		attrs := rest[spaceIdx:]
		// Color attribute names yellow
		attrRe := regexp.MustCompile(`([a-zA-Z-]+)=`)
		attrs = attrRe.ReplaceAllString(attrs, Yellow+"$1"+Reset+"=")
		buf.WriteString(attrs)
	}

	buf.WriteByte('>')
	return buf.String()
}

// JSONColorizeToken represents a JSON token for colorization
type JSONColorizeToken int

const (
	tokenInvalid JSONColorizeToken = iota
	tokenString
	tokenNumber
	tokenBool
	tokenNull
	tokenPunct
)

// ErrInvalidJSON is returned when JSON parsing fails
var ErrInvalidJSON = errors.New("invalid JSON")

// PrettyPrintJSON is a helper for simple pretty printing without color
func PrettyPrintJSON(body []byte, indent string) (string, error) {
	var buf bytes.Buffer
	decoder := json.NewDecoder(bytes.NewReader(body))
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", indent)

	if err := decoder.Decode(new(interface{})); err != nil {
		return "", ErrInvalidJSON
	}

	encoder.Encode(json.RawMessage(body))
	return buf.String(), nil
}
