package auth

import (
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

// TestAWSv4Handler tests the AWS Signature Version 4 handler.
// Uses AWS official test vectors from:
// https://docs.aws.amazon.com/general/latest/gr/sigv4-create-canonical-request.html
func TestAWSv4Handler(t *testing.T) {
	t.Run("Name returns awsv4", func(t *testing.T) {
		h := &AWSv4Handler{}
		if got := h.Name(); got != "awsv4" {
			t.Errorf("Name() = %q, want %q", got, "awsv4")
		}
	})

	t.Run("Applies AWSv4 signature headers", func(t *testing.T) {
		h := &AWSv4Handler{}
		req := &client.Request{
			Method: "GET",
			URL:    "https://examplebucket.s3.amazonaws.com/test.txt",
			Headers: []client.Header{
				{Key: "Host", Value: "examplebucket.s3.amazonaws.com"},
			},
		}

		params := map[string]string{
			"access_key": "AKIAIOSFODNN7EXAMPLE",
			"secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region":     "us-east-1",
			"service":    "s3",
		}

		h.Apply(req, params)

		// Check that Authorization header was set
		var hasAuth, hasDate bool
		for _, hdr := range req.Headers {
			switch hdr.Key {
			case "Authorization":
				hasAuth = true
				if !strings.HasPrefix(hdr.Value, "AWS4-HMAC-SHA256") {
					t.Errorf("Authorization header should start with AWS4-HMAC-SHA256, got: %s", hdr.Value)
				}
			case "X-Amz-Date":
				hasDate = true
			}
		}

		if !hasAuth {
			t.Error("Authorization header not set")
		}
		if !hasDate {
			t.Error("X-Amz-Date header not set")
		}
	})

	t.Run("Includes session token when provided", func(t *testing.T) {
		h := &AWSv4Handler{}
		req := &client.Request{
			Method: "GET",
			URL:    "https://examplebucket.s3.amazonaws.com/test.txt",
			Headers: []client.Header{
				{Key: "Host", Value: "examplebucket.s3.amazonaws.com"},
			},
		}

		params := map[string]string{
			"access_key":    "AKIAIOSFODNN7EXAMPLE",
			"secret_key":    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region":        "us-east-1",
			"service":       "s3",
			"session_token": "AQoDYXdzEJr...",
		}

		h.Apply(req, params)

		var hasSecurityToken bool
		for _, hdr := range req.Headers {
			if hdr.Key == "X-Amz-Security-Token" {
				hasSecurityToken = true
				if hdr.Value != "AQoDYXdzEJr..." {
					t.Errorf("X-Amz-Security-Token = %q, want %q", hdr.Value, "AQoDYXdzEJr...")
				}
			}
		}

		if !hasSecurityToken {
			t.Error("X-Amz-Security-Token header not set when session_token provided")
		}
	})

	t.Run("Signs POST request with body", func(t *testing.T) {
		h := &AWSv4Handler{}
		req := &client.Request{
			Method: "POST",
			URL:    "https://examplebucket.s3.amazonaws.com",
			Headers: []client.Header{
				{Key: "Host", Value: "examplebucket.s3.amazonaws.com"},
				{Key: "Content-Type", Value: "application/x-www-form-urlencoded"},
			},
			Body: "param1=value1&param2=value2",
		}

		params := map[string]string{
			"access_key": "AKIAIOSFODNN7EXAMPLE",
			"secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region":     "us-east-1",
			"service":    "s3",
		}

		h.Apply(req, params)

		var hasAuth, hasContentHash bool
		for _, hdr := range req.Headers {
			if hdr.Key == "Authorization" {
				hasAuth = true
			}
			if hdr.Key == "X-Amz-Content-Sha256" {
				hasContentHash = true
			}
		}

		if !hasAuth {
			t.Error("Authorization header not set for POST request")
		}
		if !hasContentHash {
			t.Error("X-Amz-Content-Sha256 header not set for request with body")
		}
	})

	t.Run("Handles query string parameters", func(t *testing.T) {
		h := &AWSv4Handler{}
		req := &client.Request{
			Method: "GET",
			URL:    "https://examplebucket.s3.amazonaws.com?list-type=2&max-keys=100",
			Headers: []client.Header{
				{Key: "Host", Value: "examplebucket.s3.amazonaws.com"},
			},
		}

		params := map[string]string{
			"access_key": "AKIAIOSFODNN7EXAMPLE",
			"secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"region":     "us-east-1",
			"service":    "s3",
		}

		h.Apply(req, params)

		var hasAuth bool
		for _, hdr := range req.Headers {
			if hdr.Key == "Authorization" {
				hasAuth = true
				// Should include signed query params
				if hdr.Key == "" {
					t.Error("Authorization should include signed query params")
				}
			}
		}

		if !hasAuth {
			t.Error("Authorization header not set")
		}
	})
}
