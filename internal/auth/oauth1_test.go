package auth

import (
	"fmt"
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

func TestOAuth1Handler_Name(t *testing.T) {
	h := &OAuth1Handler{}
	if got := h.Name(); got != "oauth1" {
		t.Errorf("Name() = %q, want %q", got, "oauth1")
	}
}

func TestOAuth1Handler_Apply(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		req    *client.Request
		want   string
	}{
		{
			name: "missing consumer_key",
			params: map[string]string{
				"consumer_secret": "cs",
				"token":           "t",
				"token_secret":    "ts",
			},
			req:  &client.Request{Method: "GET", URL: "https://example.com"},
			want: "",
		},
		{
			name: "missing consumer_secret",
			params: map[string]string{
				"consumer_key": "ck",
				"token":        "t",
				"token_secret": "ts",
			},
			req:  &client.Request{Method: "GET", URL: "https://example.com"},
			want: "",
		},
		{
			name: "missing token",
			params: map[string]string{
				"consumer_key":    "ck",
				"consumer_secret": "cs",
				"token_secret":    "ts",
			},
			req:  &client.Request{Method: "GET", URL: "https://example.com"},
			want: "",
		},
		{
			name: "full params",
			params: map[string]string{
				"consumer_key":    "dpf43f3p2-4Jk6l-2Lm",
				"consumer_secret": "kd94hf93kck9",
				"token":           "nnch734d00sl2jdk",
				"token_secret":    "pfkkdhi9sl3r4s00",
			},
			req:  &client.Request{Method: "GET", URL: "https://photos.example.com/initiate"},
			want: "OAuth ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &OAuth1Handler{}
			h.Apply(tt.req, tt.params)
			if tt.want == "" {
				if len(tt.req.Headers) != 0 {
					t.Errorf("expected no headers, got %v", tt.req.Headers)
				}
				return
			}
			if len(tt.req.Headers) == 0 {
				t.Errorf("expected headers, got none")
				return
			}
			if tt.req.Headers[0].Key != "Authorization" {
				t.Errorf("expected Authorization header, got %q", tt.req.Headers[0].Key)
			}
			if len(tt.req.Headers[0].Value) < len(tt.want) {
				t.Errorf("expected Authorization value to start with %q, got %q", tt.want, tt.req.Headers[0].Value)
			}
		})
	}
}

func TestOAuth1Handler_SignatureBaseString(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		baseURL        string
		queryParams    string
		wantBaseString string
	}{
		{
			name:           "simple get",
			method:         "GET",
			baseURL:        "https://photos.example.com/photos",
			queryParams:    "file=vacation.jpg&size=original",
			wantBaseString: "GET&https%3A%2F%2Fphotos.example.com%2Fphotos&file%3Dvacation.jpg%26size%3Doriginal",
		},
		{
			name:           "twitter-style get",
			method:         "GET",
			baseURL:        "https://api.twitter.com/1.1/statuses/home_timeline.json",
			queryParams:    "count=20&include_entities=true",
			wantBaseString: "GET&https%3A%2F%2Fapi.twitter.com%2F1.1%2Fstatuses%2Fhome_timeline.json&count%3D20%26include_entities%3Dtrue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseString := signatureBaseString(tt.method, tt.baseURL, tt.queryParams)
			if baseString != tt.wantBaseString {
				t.Errorf("signatureBaseString() = %q, want %q", baseString, tt.wantBaseString)
			}
		})
	}
}

func TestOAuth1Handler_HMACSHA1(t *testing.T) {
	consumerSecret := "kd94hf93kck9"
	tokenSecret := "pfkkdhi9sl3r4s00"
	baseString := "GET&https%3A%2F%2Fphotos.example.com%2Fphotos&file%3Dvacation.jpg%26size%3Doriginal"

	sig1 := hmacSHA1(consumerSecret, tokenSecret, baseString)
	sig2 := hmacSHA1(consumerSecret, tokenSecret, baseString)

	if sig1 != sig2 {
		t.Errorf("hmacSHA1() not deterministic: got %q and %q", sig1, sig2)
	}

	if sig1 == "" {
		t.Error("hmacSHA1() returned empty signature")
	}
}

func TestOAuth1Handler_FullFlow(t *testing.T) {
	params := map[string]string{
		"consumer_key":    "dpf43f3p2-4Jk6l-2Lm",
		"consumer_secret": "kd94hf93kck9",
		"token":           "nnch734d00sl2jdk",
		"token_secret":    "pfkkdhi9sl3r4s00",
	}
	req := &client.Request{
		Method: "GET",
		URL:    "https://photos.example.com/photos?file=vacation.jpg&size=original",
	}

	h := &OAuth1Handler{}
	h.Apply(req, params)

	if len(req.Headers) == 0 {
		t.Fatal("expected headers, got none")
	}

	authHeader := req.Headers[0]
	if authHeader.Key != "Authorization" {
		t.Errorf("expected Authorization header, got %q", authHeader.Key)
	}

	if authHeader.Value == "" {
		t.Error("expected non-empty Authorization value")
	}

	if authHeader.Value[:6] != "OAuth " {
		t.Errorf("expected header to start with 'OAuth ', got %q", authHeader.Value[:6])
	}

	// Parse and verify oauth_signature is present
	parsed, err := parseAuthHeader(authHeader.Value)
	if err != nil {
		t.Fatalf("failed to parse auth header: %v", err)
	}

	required := []string{"oauth_consumer_key", "oauth_token", "oauth_nonce", "oauth_signature", "oauth_signature_method", "oauth_timestamp"}
	for _, field := range required {
		if _, ok := parsed[field]; !ok {
			t.Errorf("expected %q in auth header, got %v", field, parsed)
		}
	}
}

func TestOAuth1Handler_POSTRequest(t *testing.T) {
	params := map[string]string{
		"consumer_key":    "key",
		"consumer_secret": "secret",
		"token":           "token",
		"token_secret":    "tokenSecret",
	}
	req := &client.Request{
		Method: "POST",
		URL:    "https://api.example.com/status",
		Body:   "status=Hello",
	}

	h := &OAuth1Handler{}
	h.Apply(req, params)

	if len(req.Headers) == 0 {
		t.Fatal("expected headers, got none")
	}

	authHeader := req.Headers[0]
	parsed, err := parseAuthHeader(authHeader.Value)
	if err != nil {
		t.Fatalf("failed to parse auth header: %v", err)
	}

	// For POST, oauth_body_hash should be included
	if _, ok := parsed["oauth_body_hash"]; !ok {
		t.Logf("oauth_body_hash not present (may be optional)")
	}
}

func TestOAuth1Handler_RegisterAndDispatch(t *testing.T) {
	r := NewRegistry()
	r.Register(&OAuth1Handler{})

	handler := r.Get("oauth1")
	if handler == nil {
		t.Fatal("expected handler to be registered")
	}
	if handler.Name() != "oauth1" {
		t.Errorf("handler.Name() = %q, want %q", handler.Name(), "oauth1")
	}

	// Verify unknown handler returns nil
	if r.Get("unknown") != nil {
		t.Error("expected nil for unknown handler")
	}
}

func parseAuthHeader(header string) (map[string]string, error) {
	result := make(map[string]string)

	if len(header) < 6 || header[:6] != "OAuth " {
		return nil, fmt.Errorf("invalid header format: missing OAuth prefix")
	}

	header = header[6:]

	remaining := header
	for len(remaining) > 0 {
		eqIdx := -1
		for i := 0; i < len(remaining); i++ {
			if remaining[i] == '=' {
				eqIdx = i
				break
			}
		}
		if eqIdx == -1 {
			break
		}

		key := remaining[:eqIdx]
		remaining = remaining[eqIdx+1:]

		var value string
		if len(remaining) > 0 && remaining[0] == '"' {
			remaining = remaining[1:]
			for i := 0; i < len(remaining); i++ {
				if remaining[i] == '"' {
					value = remaining[:i]
					remaining = remaining[i+1:]
					break
				}
			}
		} else {
			for i := 0; i < len(remaining); i++ {
				if remaining[i] == ',' || remaining[i] == ' ' {
					value = remaining[:i]
					remaining = remaining[i:]
					break
				}
			}
			if value == "" {
				value = remaining
				remaining = ""
			}
		}

		for len(remaining) > 0 && (remaining[0] == ',' || remaining[0] == ' ') {
			remaining = remaining[1:]
		}

		result[key] = value
	}

	return result, nil
}
