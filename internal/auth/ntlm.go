package auth

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/go-ntlmssp"
	"github.com/sreeram/gurl/internal/client"
)

type NTLMHandler struct{}

func (h *NTLMHandler) Name() string {
	return "ntlm"
}

func (h *NTLMHandler) Params() []ParamDef {
	return []ParamDef{
		{Name: "username", Required: true, Description: "NTLM username, optionally with domain"},
		{Name: "password", Required: true, Secret: true, Description: "NTLM password"},
		{Name: "domain", Description: "Optional NTLM domain retained for compatibility"},
		{Name: "workstation", Description: "Optional client workstation name for negotiate message"},
		{Name: "challenge", Description: "Base64 NTLM Type 2 challenge for the authenticate step"},
	}
}

func (h *NTLMHandler) Apply(req *client.Request, params map[string]string) error {
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

	workstation := params["workstation"]

	// Check if we have a Type 2 challenge from a 401 response
	challengeB64 := params["challenge"]
	if challengeB64 != "" {
		// We have a Type 2 challenge - this is step 2 of the handshake
		// Process the challenge and create Type 3 response
		challenge, err := base64.StdEncoding.DecodeString(challengeB64)
		if err != nil {
			return fmt.Errorf("ntlm: invalid challenge: %w", err)
		}

		// ProcessChallenge crafts an AUTHENTICATE message in response to the CHALLENGE
		// It handles domain extraction from username automatically
		authenticateMsg, err := ntlmssp.ProcessChallenge(challenge, username, password, false)
		if err != nil {
			return fmt.Errorf("ntlm: process challenge: %w", err)
		}

		encoded := base64.StdEncoding.EncodeToString(authenticateMsg)
		req.Headers = append(req.Headers, client.Header{
			Key:   "Authorization",
			Value: "NTLM " + encoded,
		})
		return nil
	}

	// No challenge - this is step 1 (Type 1 Negotiate)
	// Note: domain should be the workstation name, not user domain
	// Per go-ntlmssp docs: "Don't pass the resulting domain to NewNegotiateMessage,
	// that function expects the client machine domain, not the user domain."
	negotiateMsg, err := ntlmssp.NewNegotiateMessage(workstation, "")
	if err != nil {
		return fmt.Errorf("ntlm: create negotiate message: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(negotiateMsg)
	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: "NTLM " + encoded,
	})
	return nil
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
