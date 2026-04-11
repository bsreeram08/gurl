package auth

import (
	"encoding/base64"
	"strings"

	"github.com/Azure/go-ntlmssp"
	"github.com/sreeram/gurl/internal/client"
)

type NTLMHandler struct{}

func (h *NTLMHandler) Name() string {
	return "ntlm"
}

func (h *NTLMHandler) Apply(req *client.Request, params map[string]string) {
	username, hasUsername := params["username"]
	password, hasPassword := params["password"]

	if !hasUsername || !hasPassword {
		return
	}

	workstation := params["workstation"]

	// Check if we have a Type 2 challenge from a 401 response
	challengeB64 := params["challenge"]
	if challengeB64 != "" {
		// We have a Type 2 challenge - this is step 2 of the handshake
		// Process the challenge and create Type 3 response
		challenge, err := base64.StdEncoding.DecodeString(challengeB64)
		if err != nil {
			return
		}

		// ProcessChallenge crafts an AUTHENTICATE message in response to the CHALLENGE
		// It handles domain extraction from username automatically
		authenticateMsg, err := ntlmssp.ProcessChallenge(challenge, username, password, false)
		if err != nil {
			return
		}

		encoded := base64.StdEncoding.EncodeToString(authenticateMsg)
		req.Headers = append(req.Headers, client.Header{
			Key:   "Authorization",
			Value: "NTLM " + encoded,
		})
		return
	}

	// No challenge - this is step 1 (Type 1 Negotiate)
	// Note: domain should be the workstation name, not user domain
	// Per go-ntlmssp docs: "Don't pass the resulting domain to NewNegotiateMessage,
	// that function expects the client machine domain, not the user domain."
	negotiateMsg, err := ntlmssp.NewNegotiateMessage(workstation, "")
	if err != nil {
		return
	}

	encoded := base64.StdEncoding.EncodeToString(negotiateMsg)
	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: "NTLM " + encoded,
	})
}

func getDomainAndUser(username string) (domain, user string) {
	if idx := strings.Index(username, "\\"); idx != -1 {
		return username[:idx], username[idx+1:]
	}
	if idx := strings.Index(username, "@"); idx != -1 {
		return username[idx+1:], username[:idx]
	}
	return "", username
}
