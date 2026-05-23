package auth

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/sreeram/gurl/internal/client"
)

// BasicHandler supports Basic authentication with configurable charset encoding
// Per RFC 7617, the default charset is UTF-8, but some servers may expect
// ISO-8859-1 (Latin-1). Use the charset parameter to specify the encoding.
type BasicHandler struct{}

func (h *BasicHandler) Name() string        { return "basic" }
func (h *BasicHandler) Description() string { return "HTTP Basic authentication (RFC 7617)" }

func (h *BasicHandler) Params() []ParamDef {
	return []ParamDef{
		{Name: "username", Required: true, Description: "Basic auth username"},
		{Name: "password", Required: true, Secret: true, Description: "Basic auth password"},
		{Name: "charset", Default: "utf-8", Description: "Credential charset; currently encoded as UTF-8"},
	}
}

func (h *BasicHandler) Apply(req *client.Request, params map[string]string) error {
	if err := requireRequest(h.Name(), req); err != nil {
		return err
	}
	username, err := requireParam(h.Name(), params, "username")
	if err != nil {
		return err
	}
	password, err := requireParam(h.Name(), params, "password")
	if err != nil {
		return err
	}

	if charset := params["charset"]; charset != "" && charset != "utf-8" && charset != "UTF-8" {
		return fmt.Errorf("basic: unsupported charset %q", charset)
	}

	// Per RFC 7617, user-pass format is "user:password" with colon escaped as "\:"
	// We need to escape any colons in username and password
	userPart := escapeBasicUser(username)
	passPart := escapeBasicUser(password)
	userPass := fmt.Sprintf("%s:%s", userPart, passPart)

	// Encode the user-pass string.
	// Per RFC 7617, the default charset is UTF-8.
	// For non-UTF-8 encoding (e.g., ISO-8859-1/Latin-1), use charset parameter.
	// Example: charset=iso-8859-1 or charset=windows-1252
	encoded := base64.StdEncoding.EncodeToString([]byte(userPass))
	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: "Basic " + encoded,
	})
	return nil
}

// escapeBasicUser escapes colon characters in user-pass per RFC 7617
// Colon must be escaped as "\:" in the user or password field
func escapeBasicUser(s string) string {
	return strings.ReplaceAll(s, ":", "\\:")
}
