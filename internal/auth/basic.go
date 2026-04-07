package auth

import (
	"encoding/base64"
	"fmt"

	"github.com/sreeram/gurl/internal/client"
)

type BasicHandler struct{}

func (h *BasicHandler) Name() string {
	return "basic"
}

func (h *BasicHandler) Apply(req *client.Request, params map[string]string) {
	username, hasUsername := params["username"]
	password, hasPassword := params["password"]

	if !hasUsername || !hasPassword {
		return
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: "Basic " + encoded,
	})
}
