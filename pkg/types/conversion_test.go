package types

import (
	"testing"
)

func TestParsedCurlToSavedRequest(t *testing.T) {
	tests := []struct {
		name   string
		parsed ParsedCurl
	}{
		{
			name: "full request with headers and body",
			parsed: ParsedCurl{
				URL:     "https://api.example.com/users",
				Method:  "POST",
				Headers: map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token123"},
				Body:    `{"name":"test"}`,
			},
		},
		{
			name: "request with nil headers",
			parsed: ParsedCurl{
				URL:     "https://example.com",
				Method:  "GET",
				Headers: nil,
				Body:    "",
			},
		},
		{
			name: "request with empty headers map",
			parsed: ParsedCurl{
				URL:     "https://example.com",
				Method:  "GET",
				Headers: map[string]string{},
				Body:    "",
			},
		},
		{
			name: "request with empty method and body",
			parsed: ParsedCurl{
				URL:     "https://example.com",
				Method:  "",
				Headers: nil,
				Body:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsedCurlToSavedRequest(tt.parsed)

			if result.URL != tt.parsed.URL {
				t.Errorf("URL mismatch: got %v, want %v", result.URL, tt.parsed.URL)
			}
			if result.Method != tt.parsed.Method {
				t.Errorf("Method mismatch: got %v, want %v", result.Method, tt.parsed.Method)
			}
			if result.Body != tt.parsed.Body {
				t.Errorf("Body mismatch: got %v, want %v", result.Body, tt.parsed.Body)
			}

			// Check headers conversion
			if tt.parsed.Headers == nil {
				if result.Headers != nil {
					t.Errorf("Headers should be nil when input is nil")
				}
			} else {
				if len(result.Headers) != len(tt.parsed.Headers) {
					t.Errorf("Headers count mismatch: got %v, want %v", len(result.Headers), len(tt.parsed.Headers))
				}
				for k, v := range tt.parsed.Headers {
					found := false
					for _, h := range result.Headers {
						if h.Key == k && h.Value == v {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Header %s:%s not found in result", k, v)
					}
				}
			}
		})
	}
}

func TestSavedRequestToParsedCurl(t *testing.T) {
	tests := []struct {
		name    string
		request SavedRequest
	}{
		{
			name: "full request with headers and body",
			request: SavedRequest{
				URL:    "https://api.example.com/users",
				Method: "POST",
				Headers: []Header{
					{Key: "Content-Type", Value: "application/json"},
					{Key: "Authorization", Value: "Bearer token123"},
				},
				Body: `{"name":"test"}`,
			},
		},
		{
			name: "request with nil headers",
			request: SavedRequest{
				URL:     "https://example.com",
				Method:  "GET",
				Headers: nil,
				Body:    "",
			},
		},
		{
			name: "request with empty headers slice",
			request: SavedRequest{
				URL:     "https://example.com",
				Method:  "GET",
				Headers: []Header{},
				Body:    "",
			},
		},
		{
			name: "request with empty method and body",
			request: SavedRequest{
				URL:     "https://example.com",
				Method:  "",
				Headers: nil,
				Body:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SavedRequestToParsedCurl(tt.request)

			if result.URL != tt.request.URL {
				t.Errorf("URL mismatch: got %v, want %v", result.URL, tt.request.URL)
			}
			if result.Method != tt.request.Method {
				t.Errorf("Method mismatch: got %v, want %v", result.Method, tt.request.Method)
			}
			if result.Body != tt.request.Body {
				t.Errorf("Body mismatch: got %v, want %v", result.Body, tt.request.Body)
			}

			// Check headers conversion
			if tt.request.Headers == nil {
				if result.Headers != nil {
					t.Errorf("Headers should be nil when input is nil")
				}
			} else if len(tt.request.Headers) == 0 {
				if result.Headers == nil {
					// nil map is acceptable for empty
				}
			} else {
				if len(result.Headers) != len(tt.request.Headers) {
					t.Errorf("Headers count mismatch: got %v, want %v", len(result.Headers), len(tt.request.Headers))
				}
				for _, h := range tt.request.Headers {
					if v, ok := result.Headers[h.Key]; !ok || v != h.Value {
						t.Errorf("Header %s:%s not found or mismatched in result", h.Key, h.Value)
					}
				}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	original := ParsedCurl{
		URL:     "https://api.example.com/users",
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json", "X-Request-ID": "abc123"},
		Body:    `{"name":"test","age":25}`,
	}

	// ParsedCurl -> SavedRequest -> ParsedCurl
	saved := ParsedCurlToSavedRequest(original)
	roundTripped := SavedRequestToParsedCurl(saved)

	if roundTripped.URL != original.URL {
		t.Errorf("URL mismatch after round-trip: got %v, want %v", roundTripped.URL, original.URL)
	}
	if roundTripped.Method != original.Method {
		t.Errorf("Method mismatch after round-trip: got %v, want %v", roundTripped.Method, original.Method)
	}
	if roundTripped.Body != original.Body {
		t.Errorf("Body mismatch after round-trip: got %v, want %v", roundTripped.Body, original.Body)
	}
	if len(roundTripped.Headers) != len(original.Headers) {
		t.Errorf("Headers count mismatch after round-trip: got %v, want %v", len(roundTripped.Headers), len(original.Headers))
	}
	for k, v := range original.Headers {
		if roundTripped.Headers[k] != v {
			t.Errorf("Header %s mismatch after round-trip: got %v, want %v", k, roundTripped.Headers[k], v)
		}
	}
}
