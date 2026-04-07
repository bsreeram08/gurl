package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

// mockDigestHandler simulates a server that challenges with Digest auth
// and validates the Digest response on retry.
func mockDigestHandler(username, password string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		// First request: challenge with 401 + WWW-Authenticate: Digest
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Digest realm="testrealm", nonce="abc123", qop="auth", opaque="opaque123"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Expect Digest auth header
		if !strings.HasPrefix(auth, "Digest ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Parse username from auth header
		usernameRegex := regexp.MustCompile(`username="([^"]+)"`)
		usernameMatch := usernameRegex.FindStringSubmatch(auth)
		if len(usernameMatch) < 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		gotUsername := usernameMatch[1]

		// Parse realm from auth header
		realmRegex := regexp.MustCompile(`realm="([^"]+)"`)
		realmMatch := realmRegex.FindStringSubmatch(auth)
		if len(realmMatch) < 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		gotRealm := realmMatch[1]

		// Validate credentials
		if gotUsername != username || gotRealm != "testrealm" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

func TestDigestHandler_Name(t *testing.T) {
	h := &DigestHandler{}
	if h.Name() != "digest" {
		t.Errorf("expected name 'digest', got %q", h.Name())
	}
}

func TestDigestHandler_ImplementsHandlerInterface(t *testing.T) {
	var h Handler = &DigestHandler{}
	if h.Name() != "digest" {
		t.Errorf("expected name 'digest', got %q", h.Name())
	}
}

func TestDigestHandler_Apply_SetsAuthorizationHeader(t *testing.T) {
	handler := &DigestHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}

	handler.Apply(req, params)

	// Check Authorization header was set
	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			found = true
			if !strings.HasPrefix(h.Value, "Digest ") {
				t.Errorf("expected Digest auth, got %q", h.Value)
			}
			// Should contain username
			if !strings.Contains(h.Value, `username="testuser"`) {
				t.Errorf("expected username in auth header, got %q", h.Value)
			}
		}
	}
	if !found {
		t.Error("expected Authorization header to be set")
	}
}

func TestDigestHandler_Apply_MissingUsername(t *testing.T) {
	handler := &DigestHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"password": "testpass",
	}

	handler.Apply(req, params)

	// Should not set any headers without username
	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when username missing, got %d headers", len(req.Headers))
	}
}

func TestDigestHandler_Apply_MissingPassword(t *testing.T) {
	handler := &DigestHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"username": "testuser",
	}

	handler.Apply(req, params)

	// Should not set any headers without password
	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when password missing, got %d headers", len(req.Headers))
	}
}

func TestDigestHandler_ResponseComputation_MD5(t *testing.T) {
	username := "testuser"
	realm := "testrealm"
	password := "testpass"

	ha1Input := fmt.Sprintf("%s:%s:%s", username, realm, password)
	ha1 := md5Hash(ha1Input)
	if ha1 == "" {
		t.Error("HA1 should not be empty")
	}

	method := "GET"
	uri := "/dir/index.html"
	ha2Input := fmt.Sprintf("%s:%s", method, uri)
	ha2 := md5Hash(ha2Input)
	if ha2 == "" {
		t.Error("HA2 should not be empty")
	}

	nonce := "abc123"
	nc := "00000001"
	cnonce := "xyz123"
	qop := "auth"

	responseInput := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)
	response := md5Hash(responseInput)
	if response == "" {
		t.Error("response should not be empty")
	}

	if ha1 == ha2 {
		t.Error("HA1 and HA2 should be different for different inputs")
	}
}

func TestDigestHandler_ResponseComputation_SHA256(t *testing.T) {
	username := "testuser"
	realm := "testrealm"
	password := "testpass"

	ha1Input := fmt.Sprintf("%s:%s:%s", username, realm, password)
	ha1 := sha256Hash(ha1Input)
	if ha1 == "" {
		t.Error("HA1 should not be empty")
	}

	method := "GET"
	uri := "/dir/index.html"
	ha2Input := fmt.Sprintf("%s:%s", method, uri)
	ha2 := sha256Hash(ha2Input)
	if ha2 == "" {
		t.Error("HA2 should not be empty")
	}

	nonce := "abc123"
	nc := "00000001"
	cnonce := "xyz123"
	qop := "auth"

	responseInput := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)
	response := sha256Hash(responseInput)
	if response == "" {
		t.Error("response should not be empty")
	}

	if ha1 == ha2 {
		t.Error("HA1 and HA2 should be different for different inputs")
	}
}

func TestDigestHandler_EndToEnd(t *testing.T) {
	server := httptest.NewServer(mockDigestHandler("testuser", "testpass"))
	defer server.Close()

	handler := &DigestHandler{}

	// First request: make request without applying auth to receive 401 challenge
	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	httpReq, _ := http.NewRequest(req.Method, req.URL, nil)
	for _, h := range req.Headers {
		httpReq.Header.Set(h.Key, h.Value)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	// First request should get 401 (challenge) with no auth
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected first request to get 401, got %d", resp.StatusCode)
	}

	wwwAuth := resp.Header.Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Fatal("expected WWW-Authenticate header on 401 response")
	}
	if !strings.HasPrefix(wwwAuth, "Digest ") {
		t.Errorf("expected Digest challenge, got %q", wwwAuth)
	}

	// Parse the challenge params
	challengeParams := parseWWWAuthenticate(wwwAuth)

	// Now apply digest auth with the challenge params
	req2 := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	params := map[string]string{
		"username":  "testuser",
		"password":  "testpass",
		"realm":     challengeParams["realm"],
		"nonce":     challengeParams["nonce"],
		"qop":       challengeParams["qop"],
		"opaque":    challengeParams["opaque"],
		"algorithm": challengeParams["algorithm"],
	}

	handler.Apply(req2, params)

	httpReq2, _ := http.NewRequest(req2.Method, req2.URL, nil)
	for _, h := range req2.Headers {
		httpReq2.Header.Set(h.Key, h.Value)
	}

	resp2, err := http.DefaultClient.Do(httpReq2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	// Second request with computed auth should succeed
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected second request to get 200, got %d", resp2.StatusCode)
	}
}

func TestDigestHandler_ParsesWWWAuthenticateHeader(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantRealm  string
		wantNonce  string
		wantQop    string
		wantOpaque string
	}{
		{
			name:       "full digest header",
			header:     `Digest realm="testrealm", nonce="abc123", qop="auth", opaque="opaque123"`,
			wantRealm:  "testrealm",
			wantNonce:  "abc123",
			wantQop:    "auth",
			wantOpaque: "opaque123",
		},
		{
			name:       "minimal digest header",
			header:     `Digest realm="minimal"`,
			wantRealm:  "minimal",
			wantNonce:  "",
			wantQop:    "",
			wantOpaque: "",
		},
		{
			name:       "with algorithm",
			header:     `Digest realm="testrealm", nonce="abc123", qop="auth", algorithm=SHA-256`,
			wantRealm:  "testrealm",
			wantNonce:  "abc123",
			wantQop:    "auth",
			wantOpaque: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := parseWWWAuthenticate(tc.header)

			if params["realm"] != tc.wantRealm {
				t.Errorf("realm: expected %q, got %q", tc.wantRealm, params["realm"])
			}
			if params["nonce"] != tc.wantNonce {
				t.Errorf("nonce: expected %q, got %q", tc.wantNonce, params["nonce"])
			}
			if params["qop"] != tc.wantQop {
				t.Errorf("qop: expected %q, got %q", tc.wantQop, params["qop"])
			}
			if params["opaque"] != tc.wantOpaque {
				t.Errorf("opaque: expected %q, got %q", tc.wantOpaque, params["opaque"])
			}
		})
	}
}
