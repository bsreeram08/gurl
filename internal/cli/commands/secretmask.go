package commands

import (
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// secretMask is the placeholder printed in place of secret values.
const secretMask = "*****"

var sensitiveHeaderNames = map[string]bool{
	"authorization":       true,
	"proxy-authorization": true,
	"cookie":              true,
	"set-cookie":          true,
}

var sensitiveNameFragments = []string{"key", "secret", "token", "password", "signature"}

// sensitiveAuthParams are auth param names that hold secrets but don't match
// sensitiveNameFragments (e.g. the API key auth "value" param).
var sensitiveAuthParams = map[string]bool{
	"value":     true,
	"auth_code": true,
}

func isSensitiveName(name string) bool {
	lower := strings.ToLower(name)
	if sensitiveHeaderNames[lower] {
		return true
	}
	for _, frag := range sensitiveNameFragments {
		if strings.Contains(lower, frag) {
			return true
		}
	}
	return false
}

// shouldMaskValue reports whether a value under a sensitive name is a literal
// secret. Values containing template references ({{VAR}}) are left visible
// because they only name a variable.
func shouldMaskValue(value string) bool {
	return value != "" && !strings.Contains(value, "{{")
}

// maskRequestSecrets returns a copy of req with literal secret values in
// sensitive headers and auth params replaced by "*****". Templated values are
// kept. Masked literals are also scrubbed from the stored curl command.
func maskRequestSecrets(req *types.SavedRequest) *types.SavedRequest {
	if req == nil {
		return nil
	}
	masked := *req
	var literals []string

	if req.Headers != nil {
		masked.Headers = make([]types.Header, len(req.Headers))
		for i, h := range req.Headers {
			if isSensitiveName(h.Key) && shouldMaskValue(h.Value) {
				literals = append(literals, h.Value)
				h.Value = secretMask
			}
			masked.Headers[i] = h
		}
	}

	if req.AuthConfig != nil {
		ac := *req.AuthConfig
		if req.AuthConfig.Params != nil {
			ac.Params = make(map[string]string, len(req.AuthConfig.Params))
			for k, v := range req.AuthConfig.Params {
				if (isSensitiveName(k) || sensitiveAuthParams[strings.ToLower(k)]) && shouldMaskValue(v) {
					literals = append(literals, v)
					v = secretMask
				}
				ac.Params[k] = v
			}
		}
		masked.AuthConfig = &ac
	}

	masked.Variables = maskVars(req.Variables, &literals)
	masked.PathParams = maskVars(req.PathParams, &literals)
	masked.URL = maskQuerySecrets(req.URL, &literals)

	for _, lit := range literals {
		masked.CurlCmd = strings.ReplaceAll(masked.CurlCmd, lit, secretMask)
		masked.URL = strings.ReplaceAll(masked.URL, lit, secretMask)
		masked.Body = strings.ReplaceAll(masked.Body, lit, secretMask)
	}
	return &masked
}

// maskVars masks example values of sensitively named request variables.
func maskVars(vars []types.Var, literals *[]string) []types.Var {
	if vars == nil {
		return nil
	}
	out := make([]types.Var, len(vars))
	for i, v := range vars {
		if isSensitiveName(v.Name) && shouldMaskValue(v.Example) {
			*literals = append(*literals, v.Example)
			v.Example = secretMask
		}
		out[i] = v
	}
	return out
}

// maskQuerySecrets masks literal values of sensitively named query parameters
// (e.g. ?api_key=...). The raw string is edited so templates elsewhere stay intact.
func maskQuerySecrets(rawURL string, literals *[]string) string {
	base, query, ok := strings.Cut(rawURL, "?")
	if !ok {
		return rawURL
	}
	params := strings.Split(query, "&")
	for i, param := range params {
		name, value, hasValue := strings.Cut(param, "=")
		if hasValue && isSensitiveName(name) && shouldMaskValue(value) {
			*literals = append(*literals, value)
			params[i] = name + "=" + secretMask
		}
	}
	return base + "?" + strings.Join(params, "&")
}
