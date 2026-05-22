package auth

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sreeram/gurl/internal/client"
)

type APIKeyHandler struct{}

func (h *APIKeyHandler) Name() string {
	return "apikey"
}

func (h *APIKeyHandler) Params() []ParamDef {
	return []ParamDef{
		{Name: "header", Default: "X-API-Key", Description: "Header name for saved API key configs"},
		{Name: "value", Required: true, Secret: true, Description: "API key value for saved header configs"},
		{Name: "in", Default: "header", Description: "Legacy API key location: header or query"},
		{Name: "key", Secret: true, Description: "Legacy API key value"},
		{Name: "header_name", Default: "X-API-Key", Description: "Legacy header name when in=header"},
		{Name: "param_name", Default: "api_key", Description: "Legacy query parameter name when in=query"},
	}
}

func (h *APIKeyHandler) Apply(req *client.Request, params map[string]string) error {
	if err := requireRequest(h.Name(), req); err != nil {
		return err
	}

	if params["value"] != "" || params["header"] != "" {
		value, err := requireParam(h.Name(), params, "value")
		if err != nil {
			return err
		}
		headerName := params["header"]
		if headerName == "" {
			headerName = "X-API-Key"
		}
		req.Headers = append(req.Headers, client.Header{Key: headerName, Value: value})
		return nil
	}

	key, err := requireParam(h.Name(), params, "key")
	if err != nil {
		return err
	}

	in := params["in"]
	if in == "" {
		in = "header"
	}

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
		return fmt.Errorf("apikey: unsupported location %q", in)
	}
	return nil
}
