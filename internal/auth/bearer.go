package auth

import (
	"github.com/sreeram/gurl/internal/client"
)

type BearerHandler struct{}

func (h *BearerHandler) Name() string {
	return "bearer"
}

func (h *BearerHandler) Apply(req *client.Request, params map[string]string) {
	token, ok := params["token"]
	if !ok || token == "" {
		return
	}

	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: "Bearer " + token,
	})
}
