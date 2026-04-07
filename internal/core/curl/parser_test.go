package curl

import (
	"testing"
)

// TestParseCurl tests the curl command parser
func TestParseCurl(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantURL     string
		wantMethod  string
		wantHeaders map[string]string
		wantBody    string
		wantErr     bool
	}{
		{
			name:        "basic GET request",
			input:       "curl https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "POST with -X flag",
			input:       "curl -X POST https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "POST with --request flag",
			input:       "curl --request POST https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "PUT with --request flag",
			input:       "curl --request PUT https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "PUT",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "DELETE with -X flag",
			input:       "curl -X DELETE https://example.com/resource/1",
			wantURL:     "https://example.com/resource/1",
			wantMethod:  "DELETE",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:       "single header with -H",
			input:      "curl -H 'Content-Type: application/json' https://example.com",
			wantURL:    "https://example.com",
			wantMethod: "GET",
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			wantBody: "",
			wantErr:  false,
		},
		{
			name:       "multiple headers with -H",
			input:      "curl -H 'Content-Type: application/json' -H 'Authorization: Bearer token123' https://example.com",
			wantURL:    "https://example.com",
			wantMethod: "GET",
			wantHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
			},
			wantBody: "",
			wantErr:  false,
		},
		{
			name:       "header without quotes",
			input:      "curl -H Content-Type:application/json https://example.com",
			wantURL:    "https://example.com",
			wantMethod: "GET",
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			wantBody: "",
			wantErr:  false,
		},
		{
			name:        "POST with JSON body using -d",
			input:       "curl -d '{\"name\":\"test\"}' https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "{\"name\":\"test\"}",
			wantErr:     false,
		},
		{
			name:        "POST with --data flag",
			input:       "curl --data 'name=test' https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "name=test",
			wantErr:     false,
		},
		{
			name:        "POST with --data-raw flag",
			input:       "curl --data-raw 'raw body data' https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "raw body data",
			wantErr:     false,
		},
		{
			name:        "URL with query parameters",
			input:       "curl 'https://example.com/search?q=golang&page=1'",
			wantURL:     "https://example.com/search?q=golang&page=1",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "HTTPS URL",
			input:       "curl https://secure.example.com",
			wantURL:     "https://secure.example.com",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "HTTP URL",
			input:       "curl http://insecure.example.com",
			wantURL:     "http://insecure.example.com",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "URL with port number",
			input:       "curl https://example.com:8080/api",
			wantURL:     "https://example.com:8080/api",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "URL with port 443",
			input:       "curl https://example.com:443/api",
			wantURL:     "https://example.com:443/api",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "URL with fragment",
			input:       "curl https://example.com/#section",
			wantURL:     "https://example.com/#section",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:       "POST with header and body",
			input:      "curl -X POST -H 'Content-Type: application/json' -d '{\"key\":\"value\"}' https://example.com",
			wantURL:    "https://example.com",
			wantMethod: "POST",
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			wantBody: "{\"key\":\"value\"}",
			wantErr:  false,
		},
		{
			name:        "no URL returns error",
			input:       "curl",
			wantURL:     "",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     true,
		},
		{
			name:        "PATCH request",
			input:       "curl -X PATCH -d '{\"update\":\"field\"}' https://example.com/resource",
			wantURL:     "https://example.com/resource",
			wantMethod:  "PATCH",
			wantHeaders: map[string]string{},
			wantBody:    "{\"update\":\"field\"}",
			wantErr:     false,
		},
		{
			name:        "double quoted header with escaping",
			input:       "curl -H \"Authorization: Bearer token\" https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "GET",
			wantHeaders: map[string]string{"Authorization": "Bearer token"},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "single quoted body with special chars",
			input:       "curl -d 'name=test&value=123' https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "name=test&value=123",
			wantErr:     false,
		},
		{
			name:        "data-urlencode flag",
			input:       "curl --data-urlencode 'name=test value' https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "name=test value",
			wantErr:     false,
		},
		{
			name:        "multipart form with -F",
			input:       "curl -F 'file=@test.txt' https://example.com/upload",
			wantURL:     "https://example.com/upload",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "multipart form key value",
			input:       "curl -F 'name=john' -F 'age=30' https://example.com/submit",
			wantURL:     "https://example.com/submit",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "user authentication -u",
			input:       "curl -u 'user:pass' https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "user authentication --user",
			input:       "curl --user 'user:pass' https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "cookie flag -b",
			input:       "curl -b 'session=abc123' https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "cookie-jar flag -c",
			input:       "curl -c /tmp/cookies.txt https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "insecure -k flag",
			input:       "curl -k https://example.com/secure",
			wantURL:     "https://example.com/secure",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "insecure --insecure flag",
			input:       "curl --insecure https://example.com/secure",
			wantURL:     "https://example.com/secure",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "follow location -L flag",
			input:       "curl -L https://example.com/redirect",
			wantURL:     "https://example.com/redirect",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "compressed flag",
			input:       "curl --compressed https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "max-redirs flag",
			input:       "curl --max-redirs 5 https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "connect-timeout flag",
			input:       "curl --connect-timeout 10 https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "combined flags and body",
			input:       "curl -X POST -H 'Content-Type: application/json' --compressed -k -L -d '{\"data\":\"test\"}' https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "POST",
			wantHeaders: map[string]string{"Content-Type": "application/json"},
			wantBody:    "{\"data\":\"test\"}",
			wantErr:     false,
		},
		{
			name:        "URL with fragment and query params",
			input:       "curl 'https://example.com/path?q=1#section'",
			wantURL:     "https://example.com/path?q=1#section",
			wantMethod:  "GET",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "OPTIONS request",
			input:       "curl -X OPTIONS https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "OPTIONS",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "HEAD request",
			input:       "curl -X HEAD https://example.com/api",
			wantURL:     "https://example.com/api",
			wantMethod:  "HEAD",
			wantHeaders: map[string]string{},
			wantBody:    "",
			wantErr:     false,
		},
		{
			name:        "dollar quote body",
			input:       "curl -d $'name=test\\nvalue=123' https://example.com",
			wantURL:     "https://example.com",
			wantMethod:  "POST",
			wantHeaders: map[string]string{},
			wantBody:    "name=test\nvalue=123",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurl(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCurl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.URL != tt.wantURL {
				t.Errorf("ParseCurl() URL = %v, want %v", got.URL, tt.wantURL)
			}
			if got.Method != tt.wantMethod {
				t.Errorf("ParseCurl() Method = %v, want %v", got.Method, tt.wantMethod)
			}
			if len(got.Headers) != len(tt.wantHeaders) {
				t.Errorf("ParseCurl() Headers len = %v, want %v", len(got.Headers), len(tt.wantHeaders))
			}
			for k, v := range tt.wantHeaders {
				if got.Headers[k] != v {
					t.Errorf("ParseCurl() Headers[%s] = %v, want %v", k, got.Headers[k], v)
				}
			}
			if got.Body != tt.wantBody {
				t.Errorf("ParseCurl() Body = %v, want %v", got.Body, tt.wantBody)
			}
		})
	}
}

// TestNormalizeURL tests the URL normalization function
func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple URL",
			input: "https://example.com",
			want:  "https://example.com",
		},
		{
			name:  "URL with trailing slash",
			input: "https://example.com/",
			want:  "https://example.com",
		},
		{
			name:  "URL with path and trailing slash",
			input: "https://example.com/api/",
			want:  "https://example.com/api",
		},
		{
			name:  "URL with query params",
			input: "https://example.com/search?q=test",
			want:  "https://example.com/search?q=test",
		},
		{
			name:  "URL with port",
			input: "https://example.com:8080/api",
			want:  "https://example.com:8080/api",
		},
		{
			name:  "URL with port and trailing slash",
			input: "https://example.com:8080/",
			want:  "https://example.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
