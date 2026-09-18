package runner

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/client"
)

// verboseBodyLimit caps how much of a body is printed, so one large response
// cannot flood a terminal or a CI log.
const verboseBodyLimit = 2048

// redactedValue replaces the value of a sensitive header.
const redactedValue = "<redacted>"

// sensitiveHeaders are never printed in full. gurl encrypts collection secrets
// at rest, so echoing credentials to stdout would work against that.
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"proxy-authorization": true,
	"cookie":              true,
	"set-cookie":          true,
	"x-api-key":           true,
	"x-auth-token":        true,
}

// FormatVerboseExchange renders one request/response pair using the curl -v
// convention: "> " for what was sent, "< " for what came back. Sensitive header
// values are redacted and long bodies are truncated.
//
// Either side may be nil: a request that never reached the network still shows
// what gurl was about to send.
func FormatVerboseExchange(req *client.Request, resp *client.Response) string {
	var b strings.Builder
	writeVerboseRequest(&b, req)
	if req != nil && resp != nil {
		b.WriteString("\n")
	}
	writeVerboseResponse(&b, resp)
	return b.String()
}

func writeVerboseRequest(b *strings.Builder, req *client.Request) {
	if req == nil {
		return
	}
	fmt.Fprintf(b, "  > %s %s\n", req.Method, req.URL)
	for _, h := range sortedRequestHeaders(req.Headers) {
		fmt.Fprintf(b, "  > %s: %s\n", h.Key, headerValue(h.Key, h.Value))
	}
	writeVerboseBody(b, ">", req.Body)
}

func writeVerboseResponse(b *strings.Builder, resp *client.Response) {
	if resp == nil {
		return
	}
	fmt.Fprintf(b, "  < %d %s (%s, %d B)\n",
		resp.StatusCode,
		http.StatusText(resp.StatusCode),
		resp.Duration.Round(time.Millisecond),
		resp.Size,
	)
	for _, name := range sortedHeaderNames(resp.Headers) {
		fmt.Fprintf(b, "  < %s: %s\n", name, headerValue(name, strings.Join(resp.Headers[name], ", ")))
	}
	writeVerboseBody(b, "<", string(resp.Body))
}

// writeVerboseBody prints a blank marker line then the body, matching curl's
// header/body separation. Nothing is written for an empty body.
func writeVerboseBody(b *strings.Builder, marker, body string) {
	if body == "" {
		return
	}
	truncated, omitted := truncateBody(body)
	fmt.Fprintf(b, "  %s\n", marker)
	for _, line := range strings.Split(truncated, "\n") {
		fmt.Fprintf(b, "  %s %s\n", marker, line)
	}
	if omitted > 0 {
		fmt.Fprintf(b, "  %s ... (%d bytes truncated)\n", marker, omitted)
	}
}

// truncateBody returns the printable portion of body and how many bytes were
// dropped.
func truncateBody(body string) (string, int) {
	if len(body) <= verboseBodyLimit {
		return body, 0
	}
	return body[:verboseBodyLimit], len(body) - verboseBodyLimit
}

// headerValue redacts the value of a sensitive header, matched case
// insensitively because header casing is not guaranteed.
func headerValue(name, value string) string {
	if sensitiveHeaders[strings.ToLower(name)] {
		return redactedValue
	}
	return value
}

// sortedRequestHeaders copies and sorts headers by key so output is
// deterministic and diffable.
func sortedRequestHeaders(headers []client.Header) []client.Header {
	sorted := make([]client.Header, len(headers))
	copy(sorted, headers)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
	return sorted
}

func sortedHeaderNames(headers http.Header) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
