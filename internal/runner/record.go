package runner

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/assertions"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/pkg/types"
)

// Attempt is one attempt to run a saved request, as recorded in its
// execution record.
type Attempt struct {
	Saved   *types.SavedRequest
	EnvName string
	// Request is the request as sent, or as far as it was built when the
	// attempt failed earlier; nil means Saved as written.
	Request  *client.Request
	Response *client.Response
	// Vars is the variable view Request was resolved against; nil means
	// resolve.Vars(Scopes).
	Vars   map[string]resolve.Var
	Scopes resolve.Scopes
	// Updates are variables extracted or set by scripts during the attempt.
	Updates      map[string]string
	Error        string
	FailurePhase string
	Assertions   []assertions.Result
	Duration     time.Duration
}

// Record returns the execution record of a, with every secret masked (see
// redact.Execution). It is ready to store.
func (a Attempt) Record() *types.ExecutionHistory {
	rec := types.NewExecutionHistory(a.Saved.ID, "", 0, a.Duration.Milliseconds(), 0)
	rec.RequestName = a.Saved.Name
	rec.CollectionName = a.Saved.Collection
	rec.EnvName = a.EnvName

	req := a.Request
	if req == nil {
		req = &client.Request{Method: a.Saved.Method, URL: a.Saved.URL, Headers: convertHeaders(a.Saved.Headers), Body: a.Saved.Body}
	}
	rec.Method = req.Method
	if rec.Method == "" {
		rec.Method = "GET"
	}
	rec.URL = req.URL
	rec.RequestHeaders = convertClientHeadersToTypes(req.Headers)
	rec.RequestBody = req.Body

	if resp := a.Response; resp != nil {
		rec.Response = string(resp.Body)
		rec.StatusCode = resp.StatusCode
		rec.SizeBytes = resp.Size
		rec.DurationMs = resp.Duration.Milliseconds()
		rec.ResponseHeaders = resp.Headers
	}

	rec.Error = a.Error
	rec.FailurePhase = a.FailurePhase
	if rec.FailurePhase == "" && rec.Error == "" && a.Response != nil && (rec.StatusCode < 200 || rec.StatusCode >= 300) {
		rec.FailurePhase = FailurePhaseStatus
	}
	rec.Assertions = AssertionRecords(a.Assertions)

	vars := a.Vars
	if vars == nil {
		vars = resolve.Vars(a.Scopes)
	}
	for name, v := range vars {
		rec.Vars = append(rec.Vars, types.VarRecord{Name: name, Origin: string(v.Origin), Value: v.Value, Secret: v.Secret})
	}
	sort.Slice(rec.Vars, func(i, j int) bool { return rec.Vars[i].Name < rec.Vars[j].Name })

	secrets := append(redact.ScopeSecrets(a.Scopes, a.Updates), redact.SavedRequestSecrets(a.Saved)...)
	redact.Execution(rec, secrets)
	return rec
}

// AssertionRecords converts assertion results for an execution record.
func AssertionRecords(results []assertions.Result) []types.AssertionRecord {
	var records []types.AssertionRecord
	for _, r := range results {
		records = append(records, types.AssertionRecord{
			Name:    strings.TrimSpace(fmt.Sprintf("%s %s %s", r.Assertion.Field, r.Assertion.Op, r.Assertion.Value)),
			Passed:  r.Passed,
			Message: r.Message,
		})
	}
	return records
}

// Record returns the execution record of e, or nil when nothing was
// attempted: dry runs and skipped requests.
func (e *SingleRequestExecution) Record(envName string) *types.ExecutionHistory {
	if e == nil || e.saved == nil || e.Result == nil || e.Result.DryRun || e.Result.Skipped {
		return nil
	}
	return Attempt{
		Saved:        e.saved,
		EnvName:      envName,
		Request:      e.attempted,
		Response:     e.Response,
		Vars:         e.vars,
		Scopes:       e.scopes,
		Updates:      e.Result.DirtyVars,
		Error:        e.Result.Error,
		FailurePhase: e.Result.FailurePhase,
		Assertions:   e.Result.AssertionResults,
		Duration:     e.Result.Duration,
	}.Record()
}

// FailureReason is a short reason why rec failed, or "" when it did not.
func FailureReason(rec *types.ExecutionHistory) string {
	if !rec.Failed() {
		return ""
	}
	if rec.Error != "" {
		reason := strings.TrimPrefix(rec.Error, "request failed: ")
		reason = strings.TrimPrefix(reason, "variable substitution failed: ")
		if rec.FailurePhase == FailurePhaseHTTP {
			// Transport errors end with the cause: "...: connection refused".
			if i := strings.LastIndex(reason, ": "); i >= 0 {
				reason = reason[i+2:]
			}
		}
		return reason
	}
	if rec.FailurePhase == FailurePhaseAssertion {
		var failed []string
		for _, a := range rec.Assertions {
			if !a.Passed {
				failed = append(failed, a.Message)
			}
		}
		switch len(failed) {
		case 0:
			return "assertion failed"
		case 1:
			return "assertion failed: " + failed[0]
		}
		return fmt.Sprintf("assertion failed: %s (+%d more)", failed[0], len(failed)-1)
	}
	return fmt.Sprintf("HTTP %d", rec.StatusCode)
}

// FailureNotice is the line printed when an execution fails, naming its
// record, or "" when rec did not fail.
func FailureNotice(rec *types.ExecutionHistory) string {
	reason := FailureReason(rec)
	if reason == "" {
		return ""
	}
	id := rec.ShortID()
	return fmt.Sprintf("request failed: %s (id: %s) — inspect with: gurl diag %s", reason, id, id)
}

// saveRecord stores the record of execution, if any, and returns it; nil
// when nothing was recorded or saving failed (reported to stderr).
func (r *Runner) saveRecord(execution *SingleRequestExecution, envName string) *types.ExecutionHistory {
	rec := execution.Record(envName)
	if rec == nil {
		return nil
	}
	if err := r.db.SaveHistory(rec); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save execution record: %v\n", err)
		return nil
	}
	return rec
}
