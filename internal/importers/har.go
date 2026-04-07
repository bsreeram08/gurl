package importers

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sreeram/gurl/pkg/types"
)

// HARImporter handles HAR (HTTP Archive) files
type HARImporter struct{}

// Name returns the importer name
func (h *HARImporter) Name() string {
	return "har"
}

// Extensions returns supported file extensions
func (h *HARImporter) Extensions() []string {
	return []string{".har", ".json"}
}

// HAR represents the structure of a HAR file
type HAR struct {
	Log HARLog `json:"log"`
}

// HARLog represents the log section of a HAR file
type HARLog struct {
	Version string    `json:"version"`
	Creator HARCreator `json:"creator"`
	Browser HARBrowser `json:"browser,omitempty"`
	Pages   []HARPage  `json:"pages,omitempty"`
	Entries []HAREntry `json:"entries"`
}

// HARCreator represents the tool that created the HAR
type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HARBrowser represents the browser used
type HARBrowser struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HARPage represents a page in the HAR
type HARPage struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Started   string    `json:"startedDateTime"`
	PageTimings HARPageTimings `json:"pageTimings,omitempty"`
}

// HARPageTimings represents page timing information
type HARPageTimings struct {
	OnContentLoad float64 `json:"onContentLoad,omitempty"`
	OnLoad       float64 `json:"onLoad,omitempty"`
}

// HAREntry represents a single HTTP transaction
type HAREntry struct {
	StartedDateTime string     `json:"startedDateTime"`
	Time            float64    `json:"time"`
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Cache           HARCache    `json:"cache,omitempty"`
	Timings         HARTimings  `json:"timings"`
	ServerIPAddress string     `json:"serverIPAddress,omitempty"`
	Connection      string     `json:"connection,omitempty"`
	Comment         string     `json:"comment,omitempty"`
}

// HARRequest represents the request section
type HARRequest struct {
	Method      string          `json:"method"`
	URL         string          `json:"url"`
	HTTPVersion string          `json:"httpVersion,omitempty"`
	Cookies     []HARCookie     `json:"cookies"`
	Headers     []HARHeader     `json:"headers"`
	QueryString []HARQueryParam `json:"queryString"`
	PostData    *HARPostData    `json:"postData,omitempty"`
	HeadersSize int64           `json:"headersSize,omitempty"`
	BodySize    int64           `json:"bodySize,omitempty"`
}

// HARResponse represents the response section
type HARResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion,omitempty"`
	Cookies     []HARCookie `json:"cookies"`
	Headers     []HARHeader `json:"headers"`
	Content     HARContent   `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int64       `json:"headersSize,omitempty"`
	BodySize    int64       `json:"bodySize,omitempty"`
}

// HARCookie represents a cookie
type HARCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Expires  string `json:"expires,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
}

// HARHeader represents an HTTP header
type HARHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARQueryParam represents a query parameter
type HARQueryParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HARPostData represents posted data
type HARPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

// HARContent represents content of the response
type HARContent struct {
	Size        int64  `json:"size"`
	MimeType    string `json:"mimeType"`
	Compression int64  `json:"compression,omitempty"`
	Text        string `json:"text,omitempty"`
}

// HARCache represents cached resource info
type HARCache struct {
}

// HARTimings represents timing info
type HARTimings struct {
	Blocked float64 `json:"blocked,omitempty"`
	DNS     float64 `json:"dns,omitempty"`
	Connect float64 `json:"connect,omitempty"`
	SSL     float64 `json:"ssl,omitempty"`
	Send    float64 `json:"send,omitempty"`
	Wait    float64 `json:"wait,omitempty"`
	Receive float64 `json:"receive,omitempty"`
}

// Parse reads and parses a HAR file
func (h *HARImporter) Parse(path string) ([]*types.SavedRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var har HAR
	if err := json.Unmarshal(data, &har); err != nil {
		return nil, fmt.Errorf("parse HAR file: %w", err)
	}

	return h.convertToRequests(&har), nil
}

// convertToRequests transforms HAR entries into SavedRequests
func (h *HARImporter) convertToRequests(har *HAR) []*types.SavedRequest {
	var requests []*types.SavedRequest

	// Build page lookup map
	pageMap := make(map[string]string)
	for _, page := range har.Log.Pages {
		pageMap[page.ID] = page.Title
	}

	// Process each entry
	for i, entry := range har.Log.Entries {
		req := h.entryToSavedRequest(&entry, i, pageMap)
		requests = append(requests, req)
	}

	return requests
}

// entryToSavedRequest converts a HAR entry to a SavedRequest
func (h *HARImporter) entryToSavedRequest(entry *HAREntry, index int, pageMap map[string]string) *types.SavedRequest {
	// Generate name
	name := h.generateName(entry, index)

	// Build headers (excluding host-related)
	var headers []types.Header
	for _, header := range entry.Request.Headers {
		// Skip common headers that shouldn't be manually set
		switch header.Name {
		case "Host", "host", "Content-Length", "content-length":
			continue
		default:
			headers = append(headers, types.Header{
				Key:   header.Name,
				Value: header.Value,
			})
		}
	}

	// Add cookies as headers if they're simple
	for _, cookie := range entry.Request.Cookies {
		headers = append(headers, types.Header{
			Key:   "Cookie",
			Value: cookie.Name + "=" + cookie.Value,
		})
	}

	// Extract body
	var body string
	if entry.Request.PostData != nil {
		body = entry.Request.PostData.Text
	}

	// Build URL with query string
	url := entry.Request.URL

	// Determine tags
	var tags []string
	if pageTitle, ok := pageMap[entry.Request.URL]; ok {
		tags = append(tags, pageTitle)
	}

	// Determine collection from page
	var collection string
	if pageID := h.findPageID(entry, pageMap); pageID != "" {
		collection = pageID
	}

	// Get method
	method := entry.Request.Method
	if method == "" {
		method = "GET"
	}

	return &types.SavedRequest{
		Name:       name,
		URL:        url,
		Method:     method,
		Headers:    headers,
		Body:       body,
		Collection: collection,
		Tags:       tags,
	}
}

// generateName creates a human-readable name for the request
func (h *HARImporter) generateName(entry *HAREntry, index int) string {
	// Try to get a meaningful name from the URL
	url := entry.Request.URL

	// Extract path from URL
	path := url
	if idx := len(url) - 1; idx >= 0 {
		if qidx := -1; true {
			// Find last slash
			for i := len(path) - 1; i >= 0; i-- {
				if path[i] == '?' {
					qidx = i
					break
				}
				if path[i] == '/' {
					path = path[i:]
					break
				}
			}
			if qidx != -1 && !contains(path, "/") {
				path = path[:qidx]
			}
		}
	}

	// Clean up path
	path = cleanPath(path)

	// If path is too long or empty, use index
	if len(path) > 50 || path == "" {
		path = fmt.Sprintf("request_%d", index)
	}

	return fmt.Sprintf("%s %s", entry.Request.Method, path)
}

// cleanPath cleans a URL path for display
func cleanPath(path string) string {
	// Remove protocol and host
	if idx := 8; len(path) > idx {
		if path[:8] == "https://" {
			rest := path[8:]
			if slashIdx := -1; true {
				for i, c := range rest {
					if c == '/' {
						slashIdx = i
						break
					}
				}
				if slashIdx > 0 {
					return rest[slashIdx:]
				}
			}
		}
		if path[:7] == "http://" {
			rest := path[7:]
			if slashIdx := -1; true {
				for i, c := range rest {
					if c == '/' {
						slashIdx = i
						break
					}
				}
				if slashIdx > 0 {
					return rest[slashIdx:]
				}
			}
		}
	}
	return path
}

// findPageID attempts to find the page ID for an entry
func (h *HARImporter) findPageID(entry *HAREntry, pageMap map[string]string) string {
	// HAR entries don't directly reference pages in v1.2
	// We'd need to match by timing or connection info
	// For now, return empty
	_ = entry
	return ""
}

// contains is a simple substring check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
