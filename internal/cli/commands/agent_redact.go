package commands

import (
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// currentSecrets returns the secret values that may appear in rec's
// response body, taken from its saved request, collection and environment
// as they are now. Records store secrets masked, so the values used at run
// time are not known; a secret changed since then is not caught.
func currentSecrets(db storage.DB, envs env.Store, rec *types.ExecutionHistory) []string {
	var e *env.Environment
	if rec.EnvName != "" && envs != nil {
		if found, err := envs.GetEnvByName(rec.EnvName); err == nil {
			e = found
		}
	}
	secrets := redact.ScopeSecrets(runner.ScopesFor(db, e, rec.CollectionName, nil), nil)
	if req, err := db.GetRequest(rec.RequestID); err == nil {
		secrets = append(secrets, redact.SavedRequestSecrets(req)...)
	}
	return secrets
}

// maskedRecord returns a copy of rec with secrets also replaced in the
// response body, response header values and error, for agent-mode output.
// The stored record is not changed.
func maskedRecord(rec *types.ExecutionHistory, secrets []string) *types.ExecutionHistory {
	if rec == nil {
		return nil
	}
	masked := *rec
	masked.Response = redact.Replace(rec.Response, secrets)
	masked.Error = redact.Replace(rec.Error, secrets)
	if rec.ResponseHeaders != nil {
		masked.ResponseHeaders = make(map[string][]string, len(rec.ResponseHeaders))
		for key, values := range rec.ResponseHeaders {
			out := make([]string, len(values))
			for i, v := range values {
				out[i] = redact.Replace(v, secrets)
			}
			masked.ResponseHeaders[key] = out
		}
	}
	return &masked
}
