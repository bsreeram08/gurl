package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

// mockNTLMServer simulates an NTLM authentication server.
// It handles the 3-step handshake: negotiate -> challenge -> authenticate
func mockNTLMServer() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		if auth == "" {
			// Step 1: Client requests resource without auth - server sends 401 with NTLM challenge
			w.Header().Set("WWW-Authenticate", "NTLM")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Expect NTLM Base64 token
		if !strings.HasPrefix(auth, "NTLM ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		tokenB64 := strings.TrimPrefix(auth, "NTLM ")
		token, err := base64.StdEncoding.DecodeString(tokenB64)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Token should start with "NTLMSSP" (magic bytes)
		if !strings.HasPrefix(string(token), "NTLMSSP") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check message type (at offset 12, 4 bytes, little endian)
		// Type 1: 0x01 0x00 0x00 0x00
		// Type 3: 0x03 0x00 0x00 0x00
		if len(token) < 16 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		msgType := token[12]
		if msgType == 0x01 {
			// Step 2: Received Type 1 negotiate - respond with Type 2 challenge
			// For testing, we send a minimal Type 2 message
			// In real NTLM, Type 2 contains server challenge
			type2Challenge := createType2Challenge()
			challengeB64 := base64.StdEncoding.EncodeToString(type2Challenge)
			w.Header().Set("WWW-Authenticate", "NTLM "+challengeB64)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if msgType == 0x03 {
			// Step 3: Received Type 3 authenticate - server validates and grants access
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}

// createType2Challenge creates a minimal Type 2 challenge message for testing.
// A real Type 2 message is more complex (as per MS-NLMP spec), but for unit testing
// we just need a valid message that the client library can parse.
func createType2Challenge() []byte {
	// NTLM Type 2 message structure (simplified for testing):
	// Signature (8 bytes): "NTLMSSP\0"
	// MessageType (4 bytes): 0x00000002 (Type 2)
	// TargetName (variable)
	// NegotiateFlags (4 bytes)
	// ServerChallenge (8 bytes)
	// Reserved (8 bytes)
	// TargetInfo (variable)
	// Version (8 bytes, optional)

	sig := []byte("NTLMSSP\x00")
	msgType := []byte{0x02, 0x00, 0x00, 0x00} // Type 2

	// Negotiate flags: NTLM_NEGOTIATE_KEY_EXCH | NTLM_NEGOTIATE_VERSION
	// We want NTLMSSP_KEY_EXCHANGE (0x40000000) and NTLM_NEGOTIATE_EXTENDED_SESSION_SECURITY (0x00080000)
	negotiateFlags := []byte{0x00, 0x00, 0x08, 0x40}

	// Server challenge (8 bytes of random data for testing)
	serverChallenge := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	// Reserved (8 bytes)
	reserved := make([]byte, 8)

	// TargetNameLen (2 bytes) + TargetNameMaxLen (2 bytes) + TargetNameOffset (4 bytes)
	targetNameLen := []byte{0x00, 0x00}

	// TargetInfoLen (2 bytes) + TargetInfoMaxLen (2 bytes) + TargetInfoOffset (4 bytes)
	targetInfoLen := []byte{0x00, 0x00}

	// Version (8 bytes, optional - can be zeros for testing)
	version := make([]byte, 8)

	// Assemble: sig + msgType + targetNameLen + targetNameMaxLen + targetNameOffset +
	//          negotiateFlags + serverChallenge + reserved + targetInfoLen + targetInfoMaxLen +
	//          targetInfoOffset + version
	type2 := make([]byte, 0)
	type2 = append(type2, sig...)
	type2 = append(type2, msgType...)
	type2 = append(type2, targetNameLen...)                  // TargetNameLen
	type2 = append(type2, targetNameLen...)                  // TargetNameMaxLen
	type2 = append(type2, []byte{0x38, 0x00, 0x00, 0x00}...) // TargetNameOffset (56)
	type2 = append(type2, negotiateFlags...)
	type2 = append(type2, serverChallenge...)
	type2 = append(type2, reserved...)
	type2 = append(type2, targetInfoLen...)                  // TargetInfoLen
	type2 = append(type2, targetInfoLen...)                  // TargetInfoMaxLen
	type2 = append(type2, []byte{0x38, 0x00, 0x00, 0x00}...) // TargetInfoOffset (56)
	type2 = append(type2, version...)

	return type2
}

func TestNTLMHandler_ImplementsHandler(t *testing.T) {
	var h Handler = &NTLMHandler{}
	if h.Name() != "ntlm" {
		t.Errorf("expected name 'ntlm', got %q", h.Name())
	}
}

func TestNTLMHandler_Apply_MissingUsername(t *testing.T) {
	handler := &NTLMHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	// Missing username and password - should not set any header
	handler.Apply(req, map[string]string{"password": "testpass"})

	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when username missing, got %d headers", len(req.Headers))
	}
}

func TestNTLMHandler_Apply_MissingPassword(t *testing.T) {
	handler := &NTLMHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	// Missing password - should not set any header
	handler.Apply(req, map[string]string{"username": "testuser"})

	if len(req.Headers) != 0 {
		t.Errorf("expected no headers when password missing, got %d headers", len(req.Headers))
	}
}

func TestNTLMHandler_Apply_SetsNegotiateHeader(t *testing.T) {
	handler := &NTLMHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"username": "testuser",
		"password": "testpass",
	}

	handler.Apply(req, params)

	// Should have set an Authorization header with NTLM token
	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			found = true
			if !strings.HasPrefix(h.Value, "NTLM ") {
				t.Errorf("expected NTLM Authorization header, got %q", h.Value)
			}
			// The token should be Base64 encoded and start with NTLMSSP
			tokenB64 := strings.TrimPrefix(h.Value, "NTLM ")
			token, err := base64.StdEncoding.DecodeString(tokenB64)
			if err != nil {
				t.Errorf("failed to decode NTLM token: %v", err)
			}
			if !strings.HasPrefix(string(token), "NTLMSSP") {
				t.Errorf("expected NTLMSSP magic bytes, got %q", string(token[:8]))
			}
			// Check message type is Type 1 (0x01)
			if len(token) < 16 {
				t.Errorf("token too short: %d bytes", len(token))
			} else if token[12] != 0x01 {
				t.Errorf("expected Type 1 message (0x01), got 0x%02x", token[12])
			}
		}
	}
	if !found {
		t.Error("expected Authorization header to be set")
	}
}

func TestNTLMHandler_Apply_WithDomain(t *testing.T) {
	handler := &NTLMHandler{}
	req := &client.Request{
		Method: "GET",
		URL:    "http://example.com",
	}

	params := map[string]string{
		"username": "testuser",
		"password": "testpass",
		"domain":   "TESTDOMAIN",
	}

	handler.Apply(req, params)

	found := false
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			found = true
			if !strings.HasPrefix(h.Value, "NTLM ") {
				t.Errorf("expected NTLM Authorization header, got %q", h.Value)
			}
		}
	}
	if !found {
		t.Error("expected Authorization header to be set with domain")
	}
}

func TestRegistry_Apply_NTLMAuth(t *testing.T) {
	server := httptest.NewServer(mockNTLMServer())
	defer server.Close()

	r := NewRegistry()
	r.Register(&NTLMHandler{})

	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	r.Apply("ntlm", req, map[string]string{
		"username": "testuser",
		"password": "testpass",
	})

	// Build the HTTP request to check headers
	httpReq, _ := http.NewRequest(req.Method, req.URL, nil)
	for _, h := range req.Headers {
		httpReq.Header.Set(h.Key, h.Value)
	}

	// First request - should get 401 with NTLM challenge
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	// The first request should trigger a 401 with NTLM challenge
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected first request to get 401, got %d", resp.StatusCode)
	}

	// Check that we got the WWW-Authenticate header with NTLM
	challenge := resp.Header.Get("WWW-Authenticate")
	if challenge == "" {
		t.Error("expected WWW-Authenticate header with NTLM challenge")
	}
}

func TestRegistry_Apply_NTLMAuth_MissingCredentials(t *testing.T) {
	server := httptest.NewServer(mockNTLMServer())
	defer server.Close()

	r := NewRegistry()
	r.Register(&NTLMHandler{})

	req := &client.Request{
		Method: "GET",
		URL:    server.URL,
	}

	// Don't set any credentials
	r.Apply("ntlm", req, map[string]string{})

	// Build the HTTP request to check headers
	httpReq, _ := http.NewRequest(req.Method, req.URL, nil)
	for _, h := range req.Headers {
		httpReq.Header.Set(h.Key, h.Value)
	}

	// With no credentials, no Authorization header is set, so we should get 401
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 when no credentials, got %d", resp.StatusCode)
	}
}
