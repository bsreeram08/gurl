package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/sreeram/gurl/internal/client"
)

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

type OAuth2Handler struct {
	mu    sync.Mutex
	cache map[string]cachedToken
}

func (h *OAuth2Handler) Name() string {
	return "oauth2"
}

func (h *OAuth2Handler) Apply(req *client.Request, params map[string]string) {
	if h.cache == nil {
		h.cache = make(map[string]cachedToken)
	}

	clientID := params["client_id"]
	clientSecret := params["client_secret"]
	tokenURL := params["token_url"]
	flow := params["flow"]

	if clientID == "" || tokenURL == "" {
		return
	}

	cacheKey := tokenURL + ":" + clientID

	switch flow {
	case "auth_code":
		h.applyAuthCodeFlow(req, params, cacheKey, clientID, clientSecret, tokenURL)
	case "client_credentials":
		h.applyClientCredentialsFlow(req, params, cacheKey, clientID, clientSecret, tokenURL)
	default:
		return
	}
}

func (h *OAuth2Handler) applyAuthCodeFlow(req *client.Request, params map[string]string, cacheKey, clientID, clientSecret, tokenURL string) {
	authCode := params["auth_code"]
	redirectURI := params["redirect_uri"]
	scope := params["scope"]

	if authCode == "" {
		return
	}

	if token, ok := h.getCachedToken(cacheKey); ok {
		h.setBearerHeader(req, token)
		return
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
		return
	}

	h.cacheToken(cacheKey, token)
	h.setBearerHeader(req, token.accessToken)
}

func (h *OAuth2Handler) applyClientCredentialsFlow(req *client.Request, params map[string]string, cacheKey, clientID, clientSecret, tokenURL string) {
	scope := params["scope"]

	if token, ok := h.getCachedToken(cacheKey); ok {
		h.setBearerHeader(req, token)
		return
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
		return
	}

	h.cacheToken(cacheKey, token)
	h.setBearerHeader(req, token.accessToken)
}

func (h *OAuth2Handler) fetchToken(tokenURL, body, contentType string) (cachedToken, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(tokenURL, contentType, bytes.NewBufferString(body))
	if err != nil {
		return cachedToken{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cachedToken{}, fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return cachedToken{}, err
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return cachedToken{
		accessToken: tokenResp.AccessToken,
		expiresAt:   expiresAt,
	}, nil
}

func (h *OAuth2Handler) getCachedToken(cacheKey string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	token, ok := h.cache[cacheKey]
	if !ok {
		return "", false
	}

	if time.Now().Add(30 * time.Second).After(token.expiresAt) {
		delete(h.cache, cacheKey)
		return "", false
	}

	return token.accessToken, true
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
