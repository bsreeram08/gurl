package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/sreeram/gurl/internal/client"
)

type cachedToken struct {
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

type OAuth2Handler struct {
	mu     sync.Mutex
	cache  map[string]cachedToken
	client *http.Client
}

func (h *OAuth2Handler) Name() string        { return "oauth2" }
func (h *OAuth2Handler) Description() string { return "OAuth 2.0 token-based authentication (RFC 6749)" }

func (h *OAuth2Handler) Params() []ParamDef {
	return []ParamDef{
		{Name: "client_id", Required: true, Description: "OAuth 2.0 client identifier"},
		{Name: "client_secret", Secret: true, Description: "OAuth 2.0 client secret"},
		{Name: "token_url", Required: true, Description: "OAuth 2.0 token endpoint URL"},
		{Name: "flow", Required: true, Description: "OAuth 2.0 flow: auth_code or client_credentials"},
		{Name: "auth_code", Description: "Authorization code for auth_code flow"},
		{Name: "redirect_uri", Description: "Redirect URI for auth_code flow"},
		{Name: "registered_redirect_uri", Description: "Registered redirect URI to validate against"},
		{Name: "scope", Description: "Space-delimited OAuth scopes"},
	}
}

func (h *OAuth2Handler) Apply(req *client.Request, params map[string]string) error {
	if err := requireRequest(h.Name(), req); err != nil {
		return err
	}
	h.mu.Lock()
	if h.cache == nil {
		h.cache = make(map[string]cachedToken)
	}
	if h.client == nil {
		h.client = &http.Client{Timeout: 30 * time.Second}
	}
	h.mu.Unlock()

	clientID, err := requireParam(h.Name(), params, "client_id")
	if err != nil {
		return err
	}
	clientSecret := params["client_secret"]
	tokenURL, err := requireParam(h.Name(), params, "token_url")
	if err != nil {
		return err
	}
	flow, err := requireParam(h.Name(), params, "flow")
	if err != nil {
		return err
	}

	cacheKey := tokenURL + ":" + clientID

	switch flow {
	case "auth_code":
		return h.applyAuthCodeFlow(req, params, cacheKey, clientID, clientSecret, tokenURL)
	case "client_credentials":
		return h.applyClientCredentialsFlow(req, params, cacheKey, clientID, clientSecret, tokenURL)
	default:
		return fmt.Errorf("oauth2: unsupported flow %q", flow)
	}
}

func (h *OAuth2Handler) applyAuthCodeFlow(req *client.Request, params map[string]string, cacheKey, clientID, clientSecret, tokenURL string) error {
	authCode := params["auth_code"]
	redirectURI := params["redirect_uri"]
	registeredRedirectURI := params["registered_redirect_uri"]
	scope := params["scope"]

	if authCode == "" {
		return fmt.Errorf("oauth2: missing required param %q for auth_code flow", "auth_code")
	}

	// Validate redirect_uri if registered one is provided
	if registeredRedirectURI != "" {
		if !validateRedirectURI(registeredRedirectURI, redirectURI) {
			return fmt.Errorf("oauth2: redirect_uri does not match registered_redirect_uri")
		}
	}

	if token, ok := h.getCachedToken(cacheKey); ok {
		h.setBearerHeader(req, token.accessToken)
		return nil
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", authCode)
	data.Set("client_id", clientID)
	data.Set("redirect_uri", redirectURI)

	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}
	if scope != "" {
		data.Set("scope", scope)
	}

	token, err := h.fetchToken(tokenURL, data.Encode(), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("oauth2: token request failed: %w", err)
	}

	h.cacheToken(cacheKey, token)
	h.setBearerHeader(req, token.accessToken)
	return nil
}

func (h *OAuth2Handler) applyClientCredentialsFlow(req *client.Request, params map[string]string, cacheKey, clientID, clientSecret, tokenURL string) error {
	scope := params["scope"]

	if token, ok := h.getCachedToken(cacheKey); ok {
		h.setBearerHeader(req, token.accessToken)
		return nil
	}

	var data string
	contentType := "application/x-www-form-urlencoded"

	if clientSecret != "" {
		data = url.Values{
			"grant_type":    []string{"client_credentials"},
			"client_id":     []string{clientID},
			"client_secret": []string{clientSecret},
		}.Encode()
	} else {
		data = url.Values{
			"grant_type": []string{"client_credentials"},
			"client_id":  []string{clientID},
		}.Encode()
	}

	if scope != "" {
		data += "&scope=" + url.QueryEscape(scope)
	}

	token, err := h.fetchToken(tokenURL, data, contentType)
	if err != nil {
		return fmt.Errorf("oauth2: token request failed: %w", err)
	}

	h.cacheToken(cacheKey, token)
	h.setBearerHeader(req, token.accessToken)
	return nil
}

// validateRedirectURI checks that the sent redirect_uri matches the registered one
// Per OAuth 2.0 RFC 6749, the client MUST validate that the redirect_uri matches
func validateRedirectURI(registered, sent string) bool {
	if registered == sent {
		return true
	}
	// Handle exact match failure cases - be strict about matching
	return false
}

// generateState creates a cryptographically random state parameter for CSRF protection
// Per OAuth 2.0 RFC 6749, the state parameter is recommended to prevent CSRF attacks
func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// OAuth2State stores state for CSRF validation
type OAuth2State struct {
	State     string
	CreatedAt time.Time
}

func (h *OAuth2Handler) fetchToken(tokenURL, body, contentType string) (cachedToken, error) {
	// Get client reference without holding lock during network I/O
	var client *http.Client
	h.mu.Lock()
	if h.client != nil {
		client = h.client
	} else {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	h.mu.Unlock()

	resp, err := client.Post(tokenURL, contentType, bytes.NewBufferString(body))
	if err != nil {
		return cachedToken{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cachedToken{}, fmt.Errorf("token request failed")
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return cachedToken{}, err
	}
	if tokenResp.AccessToken == "" {
		return cachedToken{}, fmt.Errorf("token response missing access_token")
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return cachedToken{
		accessToken:  tokenResp.AccessToken,
		refreshToken: tokenResp.RefreshToken,
		expiresAt:    expiresAt,
	}, nil
}

func (h *OAuth2Handler) getCachedToken(cacheKey string) (cachedToken, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	token, ok := h.cache[cacheKey]
	if !ok {
		return cachedToken{}, false
	}

	if time.Now().Add(30 * time.Second).After(token.expiresAt) {
		delete(h.cache, cacheKey)
		return cachedToken{}, false
	}

	return token, true
}

func (h *OAuth2Handler) cacheToken(cacheKey string, token cachedToken) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cache[cacheKey] = token
}

func (h *OAuth2Handler) setBearerHeader(req *client.Request, accessToken string) {
	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: "Bearer " + accessToken,
	})
}
