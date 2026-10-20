package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/pkg/types"
)

// runOutput is one object of `run --json` output: one per execution (chain
// step or data row), or one for a dry run. Empty fields are omitted.
type runOutput struct {
	DryRun      bool   `json:"dry_run,omitempty"`
	ID          string `json:"id,omitempty"`
	ShortID     string `json:"short_id,omitempty"`
	Request     string `json:"request,omitempty"`
	Collection  string `json:"collection,omitempty"`
	Environment string `json:"environment,omitempty"`
	// Row is the 1-based data row for --data runs.
	Row        int    `json:"row,omitempty"`
	OK         bool   `json:"ok"`
	Skipped    bool   `json:"skipped,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
	Method     string `json:"method,omitempty"`
	URL        string `json:"url,omitempty"`
	// Headers are the request headers; dry runs only.
	Headers         []types.Header    `json:"headers,omitempty"`
	Status          int               `json:"status,omitempty"`
	StatusText      string            `json:"status_text,omitempty"`
	DurationMs      int64             `json:"duration_ms"`
	SizeBytes       int64             `json:"size_bytes"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	// Body is the response body (the request body for dry runs): the parsed
	// JSON value when it is valid JSON, otherwise a string.
	Body         any                     `json:"body,omitempty"`
	Error        string                  `json:"error,omitempty"`
	FailurePhase string                  `json:"failure_phase,omitempty"`
	Problems     []problemOutput         `json:"problems,omitempty"`
	Assertions   []types.AssertionRecord `json:"assertions,omitempty"`
	Extracted    map[string]string       `json:"extracted,omitempty"`
	// Extracts are the planned extractions; dry runs only.
	Extracts  []types.Extract  `json:"extracts,omitempty"`
	Persisted *persistedOutput `json:"persisted,omitempty"`
}

// problemOutput is one resolution problem. Message never holds a value.
type problemOutput struct {
	Kind     string `json:"kind"`
	Variable string `json:"variable,omitempty"`
	Location string `json:"location,omitempty"`
	Message  string `json:"message"`
}

// persistedOutput names the variables --persist wrote back. Values are
// never included.
type persistedOutput struct {
	Environment string                `json:"environment,omitempty"`
	Variables   []string              `json:"variables,omitempty"`
	Collections []persistedCollection `json:"collections,omitempty"`
}

type persistedCollection struct {
	Name      string   `json:"name"`
	Variables []string `json:"variables"`
}

// runOutputOf builds the output object of one execution from its execution
// record, so it is masked exactly like the record: URL, request headers,
// error and assertion messages, sensitive response headers. extracted are
// the values extracted from the response; those of secret or sensitively
// named variables are masked, and secrets inside the rest are replaced.
//
// The body is as received (as in the record) unless strict, which also
// replaces secrets in the body, response header values and error: agent
// mode.
func runOutputOf(rec *types.ExecutionHistory, problems resolve.Problems, extracted map[string]string, secrets []string, strict bool) runOutput {
	out := runOutput{
		ID:           rec.ID,
		ShortID:      rec.ShortID(),
		Request:      rec.RequestName,
		Collection:   rec.CollectionName,
		Environment:  rec.EnvName,
		OK:           !rec.Failed(),
		Method:       rec.Method,
		URL:          rec.URL,
		Status:       rec.StatusCode,
		StatusText:   http.StatusText(rec.StatusCode),
		DurationMs:   rec.DurationMs,
		SizeBytes:    rec.SizeBytes,
		Error:        rec.Error,
		FailurePhase: rec.FailurePhase,
		Problems:     problemOutputs(problems, secrets),
		Assertions:   rec.Assertions,
	}
	body := rec.Response
	if strict {
		body = redact.Replace(body, secrets)
		out.Error = redact.Replace(out.Error, secrets)
	}
	out.Body = jsonBody(body)
	if len(rec.ResponseHeaders) > 0 {
		out.ResponseHeaders = make(map[string]string, len(rec.ResponseHeaders))
		for key, values := range rec.ResponseHeaders {
			value := strings.Join(values, ", ")
			if strict {
				value = redact.Replace(value, secrets)
			}
			out.ResponseHeaders[key] = value
		}
	}
	if len(extracted) > 0 {
		secretVars := make(map[string]bool, len(rec.Vars))
		for _, v := range rec.Vars {
			secretVars[v.Name] = v.Secret
		}
		out.Extracted = make(map[string]string, len(extracted))
		for name, value := range extracted {
			if (secretVars[name] || redact.SensitiveName(name)) && value != "" {
				value = redact.Mask
			}
			out.Extracted[name] = redact.Replace(value, secrets)
		}
	}
	return out
}

// jsonBody is body as a JSON value when it is valid JSON, otherwise as a
// string; nil when empty.
func jsonBody(body string) any {
	if body == "" {
		return nil
	}
	if json.Valid([]byte(body)) {
		return json.RawMessage(body)
	}
	return body
}

func problemOutputs(problems resolve.Problems, secrets []string) []problemOutput {
	if len(problems) == 0 {
		return nil
	}
	out := make([]problemOutput, len(problems))
	for i, p := range problems {
		out[i] = problemOutput{
			Kind:     string(p.Kind),
			Variable: p.Variable,
			Location: p.Location,
			Message:  redact.Replace(p.Message, secrets),
		}
	}
	return out
}

// persistedOutputOf names what summary persisted; nil when nothing was.
func persistedOutputOf(summary *runner.PersistSummary) *persistedOutput {
	if summary == nil {
		return nil
	}
	out := &persistedOutput{}
	if len(summary.EnvironmentVars) > 0 {
		out.Environment = summary.EnvironmentName
		out.Variables = sortedStringKeys(summary.EnvironmentVars)
	}
	for _, c := range summary.Collections {
		if len(c.Vars) > 0 {
			out.Collections = append(out.Collections, persistedCollection{Name: c.Name, Variables: sortedStringKeys(c.Vars)})
		}
	}
	sort.Slice(out.Collections, func(i, j int) bool { return out.Collections[i].Name < out.Collections[j].Name })
	if out.Environment == "" && len(out.Collections) == 0 {
		return nil
	}
	return out
}

// executionOutput builds the output object of execution from its record.
func executionOutput(execution *runner.SingleRequestExecution, rec *types.ExecutionHistory, strict bool) runOutput {
	return runOutputOf(rec, execution.Result.Problems, execution.Result.ExtractedVars, execution.Secrets(), strict)
}

// skippedOutput is the output object of an execution a script or run_if
// skipped; nothing was sent or recorded.
func skippedOutput(execution *runner.SingleRequestExecution, envName string) runOutput {
	out := runOutput{Request: execution.Result.RequestName, Environment: envName, OK: true, Skipped: true, SkipReason: execution.Result.SkipReason}
	if out.SkipReason == "" {
		out.SkipReason = runner.SkipReasonScript
	}
	return out
}

// ndjsonWriter writes run output objects one per line. The latest object is
// held back so the persist summary, known only at the end of a chain or data
// run, can be attached to the last object.
type ndjsonWriter struct {
	w       io.Writer
	pending *runOutput
	err     error
}

func (n *ndjsonWriter) add(out runOutput) {
	n.flushPending()
	n.pending = &out
}

// finish writes the held-back object with persisted attached and returns the
// first write error.
func (n *ndjsonWriter) finish(persisted *persistedOutput) error {
	if n.pending != nil {
		n.pending.Persisted = persisted
	}
	n.flushPending()
	return n.err
}

func (n *ndjsonWriter) flushPending() {
	if n.pending == nil {
		return
	}
	if err := writeJSONLine(n.w, n.pending); err != nil && n.err == nil {
		n.err = err
	}
	n.pending = nil
}
