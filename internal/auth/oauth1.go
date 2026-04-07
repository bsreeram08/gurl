package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/client"
)

type OAuth1Handler struct{}

func (h *OAuth1Handler) Name() string {
	return "oauth1"
}

func (h *OAuth1Handler) Apply(req *client.Request, params map[string]string) {
	consumerKey := params["consumer_key"]
	consumerSecret := params["consumer_secret"]
	token := params["token"]
	tokenSecret := params["token_secret"]

	if consumerKey == "" || consumerSecret == "" {
		return
	}

	if token == "" {
		return
	}

	nonce := generateNonce()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	reqURL, queryString := splitURL(req.URL)

	signatureBase := signatureBaseString(req.Method, reqURL, queryString)

	signature := hmacSHA1(consumerSecret, tokenSecret, signatureBase)

	authHeader := buildAuthHeader(consumerKey, nonce, timestamp, signature, token, req.Body)

	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: authHeader,
	})
}

func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func signatureBaseString(method, baseURL, queryString string) string {
	encodedMethod := percentEncode(strings.ToUpper(method))
	encodedBaseURL := percentEncode(baseURL)
	encodedQuery := percentEncode(queryString)

	return encodedMethod + "&" + encodedBaseURL + "&" + encodedQuery
}

func hmacSHA1(consumerSecret, tokenSecret, baseString string) string {
	key := percentEncode(consumerSecret) + "&" + percentEncode(tokenSecret)

	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(baseString))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func buildAuthHeader(consumerKey, nonce, timestamp, signature, token, body string) string {
	params := []string{
		`oauth_consumer_key="` + consumerKey + `"`,
		`oauth_nonce="` + nonce + `"`,
		`oauth_signature="` + signature + `"`,
		`oauth_signature_method="HMAC-SHA1"`,
		`oauth_timestamp="` + timestamp + `"`,
		`oauth_token="` + token + `"`,
		`oauth_version="1.0"`,
	}

	if body != "" {
		hash := sha1.Sum([]byte(body))
		bodyHash := base64.StdEncoding.EncodeToString(hash[:])
		params = append(params, `oauth_body_hash="`+bodyHash+`"`)
	}

	return "OAuth " + strings.Join(params, ", ")
}

func splitURL(requestURL string) (baseURL, queryString string) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return requestURL, ""
	}

	baseURL = u.Scheme + "://" + u.Host + u.Path
	queryString = u.RawQuery

	return baseURL, queryString
}

func percentEncode(s string) string {
	var encoded strings.Builder
	for _, c := range s {
		if isUnreserved(c) {
			encoded.WriteRune(c)
		} else {
			encoded.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return encoded.String()
}

func isUnreserved(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

func percentEncodeParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	return strings.Join(pairs, "&")
}
