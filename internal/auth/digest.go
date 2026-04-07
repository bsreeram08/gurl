package auth

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/sreeram/gurl/internal/client"
)

type DigestHandler struct{}

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
	qop := params["qop"]
	opaque := params["opaque"]
	algorithm := params["algorithm"]

	if realm == "" {
		realm = "default-realm"
	}
	if nonce == "" {
		nonce = "default-nonce"
	}
	if qop == "" {
		qop = "auth"
	}

	method := req.Method
	if method == "" {
		method = "GET"
	}
	uri := req.URL

	var response string
	var ha1, ha2 string
	nc := "00000001"
	cnonce := generateCnonce()

	if algorithm == "SHA-256" {
		ha1Input := fmt.Sprintf("%s:%s:%s", username, realm, password)
		ha1 = sha256Hash(ha1Input)
		ha2Input := fmt.Sprintf("%s:%s", method, uri)
		ha2 = sha256Hash(ha2Input)

		responseInput := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)
		response = sha256Hash(responseInput)
	} else {
		ha1Input := fmt.Sprintf("%s:%s:%s", username, realm, password)
		ha1 = md5Hash(ha1Input)
		ha2Input := fmt.Sprintf("%s:%s", method, uri)
		ha2 = md5Hash(ha2Input)

		responseInput := fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)
		response = md5Hash(responseInput)
	}

	authHeader := buildDigestHeader(username, realm, nonce, uri, response, opaque, cnonce, nc, qop, algorithm)

	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: authHeader,
	})
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

func parseWWWAuthenticate(header string) map[string]string {
	params := make(map[string]string)

	header = strings.TrimPrefix(header, "Digest ")
	header = strings.TrimSpace(header)

	re := regexp.MustCompile(`(\w+)="([^"]*)"`)
	matches := re.FindAllStringSubmatch(header, -1)

	for _, match := range matches {
		params[match[1]] = match[2]
	}

	return params
}
