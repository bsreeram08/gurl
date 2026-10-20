// Package redact holds the rules that keep secret values out of everything
// gurl prints or stores: saved requests shown or exported, dry-run previews
// and execution records.
//
// A value is secret when its variable is marked secret, when its name looks
// sensitive (Authorization, Cookie, *key*, *token*, *secret*, ...), or when it
// sits under a sensitive header, auth param or query parameter name. Values
// that still contain a {{placeholder}} only name a variable and stay visible.
package redact

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"

	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/pkg/types"
)

// Mask is the placeholder shown in place of secret values.
const Mask = "*****"

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

// SensitiveName reports whether a header, variable or parameter name usually
// holds a secret.
func SensitiveName(name string) bool {
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

// Literal reports whether a value under a sensitive name is a literal secret.
// Values containing template references ({{VAR}}) are left visible because
// they only name a variable.
func Literal(value string) bool {
	return value != "" && !strings.Contains(value, "{{")
}

// Replace replaces every occurrence of each secret in s with Mask, longest
// secret first so a secret containing another is masked whole.
// minSecretLen is the shortest value Replace masks inside free text.
const minSecretLen = 4

func Replace(s string, secrets []string) string {
	for _, secret := range bySize(secrets) {
		s = strings.ReplaceAll(s, secret, Mask)
	}
	return s
}

// encodedForms returns s plus the URL- and JSON-escaped spellings it can take
// in a URL, error message or body, so masking doesn't miss an encoded copy.
func encodedForms(s string) []string {
	forms := []string{s, url.QueryEscape(s), url.PathEscape(s)}
	if b, err := json.Marshal(s); err == nil {
		forms = append(forms, string(b[1:len(b)-1]))
	}
	return forms
}

func bySize(secrets []string) []string {
	out := make([]string, 0, len(secrets))
	for _, s := range secrets {
		// ponytail: values shorter than minSecretLen are not replaced; they
		// would garble unrelated text and reveal themselves by the gaps.
		if len(s) >= minSecretLen {
			out = append(out, encodedForms(s)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

// ScopeSecrets returns every value from scopes that must never be shown: the
// values of secret or sensitively named variables, every secret-marked
// environment and collection value (a secret overridden by a higher scope
// can still reach the request through an alias), and the values in updates
// (variables extracted or set by scripts) whose names are secret or
// sensitive.
func ScopeSecrets(s resolve.Scopes, updates map[string]string) []string {
	vars := resolve.Vars(s)
	var secrets []string
	for key, v := range vars {
		if (v.Secret || SensitiveName(key)) && v.Value != "" {
			secrets = append(secrets, v.Value)
		}
	}
	for key := range s.EnvSecrets {
		if value := s.Env[key]; value != "" {
			secrets = append(secrets, value)
		}
	}
	for key := range s.CollectionSecrets {
		if value := s.Collection[key]; value != "" {
			secrets = append(secrets, value)
		}
	}
	for key, value := range updates {
		if (vars[key].Secret || SensitiveName(key)) && value != "" {
			secrets = append(secrets, value)
		}
	}
	return secrets
}

// SavedRequest returns a copy of req with literal secret values in sensitive
// headers, auth params, variable examples and query parameters replaced by
// Mask. Templated values are kept. Masked literals are also scrubbed from the
// URL, body and stored curl command.
func SavedRequest(req *types.SavedRequest) *types.SavedRequest {
	if req == nil {
		return nil
	}
	masked := *req
	var literals []string

	if req.Headers != nil {
		masked.Headers = make([]types.Header, len(req.Headers))
		for i, h := range req.Headers {
			if SensitiveName(h.Key) && Literal(h.Value) {
				literals = append(literals, h.Value)
				h.Value = Mask
			}
			masked.Headers[i] = h
		}
	}

	if req.AuthConfig != nil {
		ac := *req.AuthConfig
		if req.AuthConfig.Params != nil {
			ac.Params = make(map[string]string, len(req.AuthConfig.Params))
			for k, v := range req.AuthConfig.Params {
				if sensitiveAuthParam(k) && Literal(v) {
					literals = append(literals, v)
					v = Mask
				}
				ac.Params[k] = v
			}
		}
		masked.AuthConfig = &ac
	}

	masked.Variables = maskVars(req.Variables, &literals)
	masked.PathParams = maskVars(req.PathParams, &literals)
	masked.URL = maskQuery(req.URL, &literals)

	masked.CurlCmd = Replace(masked.CurlCmd, literals)
	masked.URL = Replace(masked.URL, literals)
	masked.Body = Replace(masked.Body, literals)
	return &masked
}

// SavedRequestSecrets returns the literal secrets written into req itself:
// sensitive header and auth param values, sensitive variable examples and
// sensitive query parameter values.
func SavedRequestSecrets(req *types.SavedRequest) []string {
	if req == nil {
		return nil
	}
	var literals []string
	for _, h := range req.Headers {
		if SensitiveName(h.Key) && Literal(h.Value) {
			literals = append(literals, h.Value)
		}
	}
	if req.AuthConfig != nil {
		for k, v := range req.AuthConfig.Params {
			if sensitiveAuthParam(k) && Literal(v) {
				literals = append(literals, v)
			}
		}
	}
	maskVars(req.Variables, &literals)
	maskVars(req.PathParams, &literals)
	maskQuery(req.URL, &literals)
	return literals
}

// Execution masks secrets in rec, in place, before it is stored: values of
// secret or sensitively named variables, literal values of sensitive request
// headers and query parameters, every occurrence of those values and of
// secrets in the URL, request headers, request body, error, assertion
// messages and other variable values, and the values of sensitive response
// headers. The response body is left as received.
func Execution(rec *types.ExecutionHistory, secrets []string) {
	literals := append([]string(nil), secrets...)

	for i, v := range rec.Vars {
		if (v.Secret || SensitiveName(v.Name)) && v.Value != "" {
			literals = append(literals, v.Value)
			rec.Vars[i].Value = Mask
		}
	}
	for i, h := range rec.RequestHeaders {
		if SensitiveName(h.Key) && Literal(h.Value) {
			literals = append(literals, h.Value)
			rec.RequestHeaders[i].Value = Mask
		}
	}
	rec.URL = maskQuery(rec.URL, &literals)

	rec.URL = Replace(rec.URL, literals)
	rec.RequestBody = Replace(rec.RequestBody, literals)
	rec.Error = Replace(rec.Error, literals)
	for i, a := range rec.Assertions {
		rec.Assertions[i].Message = Replace(a.Message, literals)
	}
	for i, h := range rec.RequestHeaders {
		rec.RequestHeaders[i].Value = Replace(h.Value, literals)
	}
	for i, v := range rec.Vars {
		if v.Value != Mask {
			rec.Vars[i].Value = Replace(v.Value, literals)
		}
	}
	rec.ResponseHeaders = maskHeaderMap(rec.ResponseHeaders)
}

func maskHeaderMap(headers map[string][]string) map[string][]string {
	if headers == nil {
		return nil
	}
	out := make(map[string][]string, len(headers))
	for key, values := range headers {
		if SensitiveName(key) {
			masked := make([]string, len(values))
			for i := range masked {
				masked[i] = Mask
			}
			out[key] = masked
			continue
		}
		out[key] = append([]string(nil), values...)
	}
	return out
}

func sensitiveAuthParam(name string) bool {
	return SensitiveName(name) || sensitiveAuthParams[strings.ToLower(name)]
}

// maskVars masks example values of sensitively named request variables.
func maskVars(vars []types.Var, literals *[]string) []types.Var {
	if vars == nil {
		return nil
	}
	out := make([]types.Var, len(vars))
	for i, v := range vars {
		if SensitiveName(v.Name) && Literal(v.Example) {
			*literals = append(*literals, v.Example)
			v.Example = Mask
		}
		out[i] = v
	}
	return out
}

// maskQuery masks literal values of sensitively named query parameters (e.g.
// ?api_key=...). The raw string is edited so templates elsewhere stay intact.
func maskQuery(rawURL string, literals *[]string) string {
	base, query, ok := strings.Cut(rawURL, "?")
	if !ok {
		return rawURL
	}
	params := strings.Split(query, "&")
	for i, param := range params {
		name, value, hasValue := strings.Cut(param, "=")
		if hasValue && SensitiveName(name) && Literal(value) {
			*literals = append(*literals, value)
			params[i] = name + "=" + Mask
		}
	}
	return base + "?" + strings.Join(params, "&")
}
