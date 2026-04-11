package auth

import (
	"net/url"
	"strings"

	"github.com/sreeram/gurl/internal/client"
)

type APIKeyHandler struct{}

func (h *APIKeyHandler) Name() string {
	return "apikey"
}

func (h *APIKeyHandler) Apply(req *client.Request, params map[string]string) {
	key, hasKey := params["key"]
	if !hasKey {
		return
	}

	in := params["in"]

	switch in {
	case "header":
		headerName := params["header_name"]
		if headerName == "" {
			headerName = "X-API-Key"
		}
		req.Headers = append(req.Headers, client.Header{
			Key:   headerName,
			Value: key,
		})
	case "query":
		paramName := params["param_name"]
		if paramName == "" {
			paramName = "api_key"
		}
		// URL-escape the key value for query params
		escapedKey := url.QueryEscape(key)
		if strings.Contains(req.URL, "?") {
			req.URL = req.URL + "&" + paramName + "=" + escapedKey
		} else {
			req.URL = req.URL + "?" + paramName + "=" + escapedKey
		}
	default:
		if in != "" {
			return
		}
	}
}
