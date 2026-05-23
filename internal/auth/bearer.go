package auth

import "github.com/sreeram/gurl/internal/client"

type BearerHandler struct{}

func (h *BearerHandler) Name() string        { return "bearer" }
func (h *BearerHandler) Description() string { return "Bearer token authentication (RFC 6750)" }

func (h *BearerHandler) Params() []ParamDef {
	return []ParamDef{
		{Name: "token", Required: true, Secret: true, Description: "Bearer token value"},
	}
}

func (h *BearerHandler) Apply(req *client.Request, params map[string]string) error {
	if err := requireRequest(h.Name(), req); err != nil {
		return err
	}
	token, err := requireParam(h.Name(), params, "token")
	if err != nil {
		return err
	}

	req.Headers = append(req.Headers, client.Header{
		Key:   "Authorization",
		Value: "Bearer " + token,
	})
	return nil
}
