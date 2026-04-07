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
	_, hasUsername := params["username"]
	_, hasPassword := params["password"]

	if !hasUsername || !hasPassword {
		return
	}

	domain := params["domain"]

	negotiateMsg, err := ntlmssp.NewNegotiateMessage(domain, "")
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
