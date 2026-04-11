package auth

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"

	"github.com/sreeram/gurl/internal/client"
)

type DigestHandler struct {
	// nonceCounts tracks the nc value per nonce for replay protection
	mu          sync.Mutex
	nonceCounts map[string]int
}

func (h *DigestHandler) Name() string {
	return "digest"
}

func (h *DigestHandler) Apply(req *client.Request, params map[string]string) {
	username, hasUsername := params["username"]
	password, hasPassword := params["password"]

	if !hasUsername || !hasPassword {
		return
	}

	// Get challenge params from the request if available
	// In a real implementation, these would come from the 401 response's WWW-Authenticate header
	realm := params["realm"]
	nonce := params["nonce"]
	serverQop := params["qop"]
	opaque := params["opaque"]
	algorithm := params["algorithm"]
	clientQop := params["client_qop"]

	if realm == "" {
		realm = "default-realm"
	}
	if nonce == "" {
		nonce = "default-nonce"
	}
	// Default client qop to "auth" if not specified
	if clientQop == "" {
		clientQop = "auth"
	}

	method := req.Method
	if method == "" {
		method = "GET"
	}
	uri := req.URL

	var response string
	var ha1, ha2 string

	// Determine qop to use - validate server qop against client qop if both specified
	qop := clientQop
	if serverQop != "" && clientQop != "" {
		// Check if there's a mismatch
		if serverQOpAvailable := strings.Contains(serverQop, clientQop); !serverQOpAvailable {
			// Client qop not in server's qop list - log warning
			// Fall back to whatever the server sent
			qop = serverQop
		}
	} else if serverQop != "" {
		// Use server's qop
		qop = serverQop
	}

	// Get and increment nonce count for this nonce
	nc := h.incrementNonceCount(nonce)
	ncStr := fmt.Sprintf("%08x", nc)
	cnonce := generateCnonce()

	if algorithm == "SHA-256" {
		ha1Input := fmt.Sprintf("%s:%s:%s", username, realm, password)
		ha1 = sha256Hash(ha1Input)

		// Handle -sess algorithm per RFC 7616
		// If algorithm contains "-sess", compute HA1 = H(HA1:nonce:cnonce)
		if strings.Contains(algorithm, "-sess") {
			ha1 = sha256Hash(ha1 + ":" + nonce + ":" + cnonce)
		}

		ha2Input := fmt.Sprintf("%s:%s", method, uri)
		ha2 = sha256Hash(ha2Input)

		responseInput := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, ncStr, cnonce, qop, ha2)
		response = sha256Hash(responseInput)
	} else {
		ha1Input := fmt.Sprintf("%s:%s:%s", username, realm, password)
		ha1 = md5Hash(ha1Input)

		// Handle -sess algorithm per RFC 7616
		// If algorithm contains "-sess", compute HA1 = H(HA1:nonce:cnonce)
		if strings.Contains(algorithm, "-sess") {
			ha1 = md5Hash(ha1 + ":" + nonce + ":" + cnonce)
		}

		ha2Input := fmt.Sprintf("%s:%s", method, uri)
		ha2 = md5Hash(ha2Input)

		responseInput := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, ncStr, cnonce, qop, ha2)
		response = md5Hash(responseInput)
	}

	authHeader := buildDigestHeader(username, realm, nonce, uri, response, opaque, cnonce, ncStr, qop, algorithm)

	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: authHeader,
	})
}

// incrementNonceCount atomically increments and returns the nonce count for a given nonce
func (h *DigestHandler) incrementNonceCount(nonce string) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.nonceCounts == nil {
		h.nonceCounts = make(map[string]int)
	}

	h.nonceCounts[nonce]++
	return h.nonceCounts[nonce]
}

func buildDigestHeader(username, realm, nonce, uri, response, opaque, cnonce, nc, qop, algorithm string) string {
	var parts []string

	parts = append(parts, fmt.Sprintf(`username="%s"`, username))
	parts = append(parts, fmt.Sprintf(`realm="%s"`, realm))
	parts = append(parts, fmt.Sprintf(`nonce="%s"`, nonce))
	parts = append(parts, fmt.Sprintf(`uri="%s"`, uri))
	parts = append(parts, fmt.Sprintf(`response="%s"`, response))

	if opaque != "" {
		parts = append(parts, fmt.Sprintf(`opaque="%s"`, opaque))
	}

	if cnonce != "" {
		parts = append(parts, fmt.Sprintf(`cnonce="%s"`, cnonce))
	}

	if nc != "" {
		parts = append(parts, fmt.Sprintf(`nc=%s`, nc))
	}

	if qop != "" {
		parts = append(parts, fmt.Sprintf(`qop=%s`, qop))
	}

	if algorithm != "" {
		parts = append(parts, fmt.Sprintf(`algorithm=%s`, algorithm))
	}

	return "Digest " + strings.Join(parts, ", ")
}

func generateCnonce() string {
	const charset = "abcdef0123456789"
	const length = 16
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return hex.EncodeToString(b)
}

func md5Hash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

func sha256Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

var wwwAuthHeaderRegex = regexp.MustCompile(`(\w+)="([^"]*)"`)

func parseWWWAuthenticate(header string) map[string]string {
	params := make(map[string]string)

	header = strings.TrimPrefix(header, "Digest ")
	header = strings.TrimSpace(header)

	matches := wwwAuthHeaderRegex.FindAllStringSubmatch(header, -1)

	for _, match := range matches {
		params[match[1]] = match[2]
	}

	return params
}
