package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/client"
)

type AWSv4Handler struct{}

func (h *AWSv4Handler) Name() string {
	return "awsv4"
}

func (h *AWSv4Handler) Apply(req *client.Request, params map[string]string) {
	accessKey := params["access_key"]
	secretKey := params["secret_key"]
	region := params["region"]
	service := params["service"]
	sessionToken := params["session_token"]

	if accessKey == "" || secretKey == "" || region == "" || service == "" {
		return
	}

	now := time.Now().UTC()
	dateStr := now.Format("20060102")
	amzDateStr := now.Format("20060102T150405Z")

	payloadHash := hashSHA256([]byte(req.Body))

	canonicalHeaders := buildCanonicalHeaders(req.Headers)
	signedHeaders := buildSignedHeaders(req.Headers)

	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return
	}
	canonicalURI := canonicalURI(parsedURL.Path)
	canonicalQueryString := canonicalQueryString(parsedURL.RawQuery)

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	canonicalRequestHash := hashSHA256([]byte(canonicalRequest))

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStr, region, service)

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDateStr,
		credentialScope,
		canonicalRequestHash,
	}, "\n")

	signingKey := getSigningKey(secretKey, dateStr, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authorizationHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey,
		credentialScope,
		signedHeaders,
		signature,
	)

	req.Headers = append(req.Headers, client.Header{
		Key:   "X-Amz-Date",
		Value: amzDateStr,
	})
	req.Headers = append(req.Headers, client.Header{
		Key:   "X-Amz-Content-Sha256",
		Value: payloadHash,
	})
	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: authorizationHeader,
	})

	if sessionToken != "" {
		req.Headers = append(req.Headers, client.Header{
			Key:   "X-Amz-Security-Token",
			Value: sessionToken,
		})
	}
}

func hashSHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSigningKey(secretKey, date, region, service string) []byte {
	kSecret := []byte("AWS4" + secretKey)
	kDate := hmacSHA256(kSecret, []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func canonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	uri := url.PathEscape(path)
	uri = strings.ReplaceAll(uri, "%2F", "/")
	if !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}
	return uri
}

func canonicalQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	params, _ := url.ParseQuery(rawQuery)
	encodedParams := make([]string, 0, len(params))
	for key, values := range params {
		sort.Strings(values)
		for _, value := range values {
			encodedParams = append(encodedParams, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	sort.Strings(encodedParams)
	return strings.Join(encodedParams, "&")
}

func buildCanonicalHeaders(headers []client.Header) string {
	headerMap := make(map[string]string)
	for _, h := range headers {
		headerMap[strings.ToLower(h.Key)] = strings.TrimSpace(h.Value)
	}

	keys := make([]string, 0, len(headerMap))
	for key := range headerMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var result []string
	for _, key := range keys {
		result = append(result, key+":"+headerMap[key])
	}
	return strings.Join(result, "\n") + "\n"
}

func buildSignedHeaders(headers []client.Header) string {
	keys := make([]string, 0, len(headers))
	seen := make(map[string]bool)
	for _, h := range headers {
		key := strings.ToLower(h.Key)
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}
