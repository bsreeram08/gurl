package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

// diagBodyLimit is how much of the response body diag prints without --full.
const diagBodyLimit = 4096

// DiagCommand shows one execution record and how it differs from the last
// successful execution of the same request.
//
// In agent mode the output is JSON and secrets of the request, its
// collection and its environment are also masked in response bodies.
func DiagCommand(db storage.DB, envs env.Store) *cli.Command {
	return &cli.Command{
		Name:      "diag",
		Usage:     "Inspect one execution: request, variables, response, and what changed since the last success",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "Execution id or unique prefix (as printed by run and history)",
			},
			&cli.BoolFlag{
				Name:  "full",
				Usage: "Print the whole response body instead of the first 4KB",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Print the execution record and comparison as JSON",
			},
		},
		Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
			id := c.String("id")
			if id == "" {
				id = c.Args().First()
			}
			if strings.TrimSpace(id) == "" {
				return cli.Exit("execution id is required: gurl diag <id>", 2)
			}
			rec, err := db.GetHistoryByID(id)
			if err != nil {
				return err
			}
			base, err := lastSuccessBefore(db, rec)
			if err != nil {
				return err
			}
			if agentMode(c) {
				secrets := currentSecrets(db, envs, rec)
				if base != nil { // the baseline may have run in another environment
					secrets = append(secrets, currentSecrets(db, envs, base)...)
				}
				rec, base = maskedRecord(rec, secrets), maskedRecord(base, secrets)
			}
			cmp := compareExecutions(base, rec)
			if jsonOutput(c) {
				return writeJSON(os.Stdout, diagReport{Record: rec, ComparedWith: cmp})
			}
			return writeDiag(os.Stdout, rec, cmp, c.Bool("full"))
		}),
	}
}

type diagReport struct {
	Record       *types.ExecutionHistory `json:"record"`
	ComparedWith *diagComparison         `json:"compared_with"`
}

// diagComparison is what changed between the last successful execution of a
// request (the baseline) and the execution being inspected.
//
// Secret variable values are stored masked, so their values cannot be
// compared; only changes of origin, or a secret appearing or disappearing,
// are reported.
type diagComparison struct {
	BaselineID         string    `json:"baseline_id"`
	BaselineTime       time.Time `json:"baseline_time"`
	BaselineStatus     int       `json:"baseline_status"`
	Status             int       `json:"status"`
	BaselineDurationMs int64     `json:"baseline_duration_ms"`
	DurationDeltaMs    int64     `json:"duration_delta_ms"`
	// RequestCompared is false when the baseline predates stored request
	// details; Request and Vars are then empty.
	RequestCompared        bool            `json:"request_compared"`
	Request                []diagChange    `json:"request,omitempty"`
	Vars                   []diagVarChange `json:"vars,omitempty"`
	ResponseHeadersAdded   []string        `json:"response_headers_added,omitempty"`
	ResponseHeadersRemoved []string        `json:"response_headers_removed,omitempty"`
	ResponseFields         []diagChange    `json:"response_fields,omitempty"`
}

// diagChange is one changed field; an empty Before or After with the
// matching Added/Removed flag means the field is new or gone.
type diagChange struct {
	Field   string `json:"field"`
	Before  string `json:"before,omitempty"`
	After   string `json:"after,omitempty"`
	Added   bool   `json:"added,omitempty"`
	Removed bool   `json:"removed,omitempty"`
}

type diagVarChange struct {
	Name         string `json:"name"`
	BeforeOrigin string `json:"before_origin,omitempty"`
	AfterOrigin  string `json:"after_origin,omitempty"`
	Before       string `json:"before,omitempty"`
	After        string `json:"after,omitempty"`
	Secret       bool   `json:"secret,omitempty"`
}

// lastSuccessBefore returns the most recent successful execution of
// rec's request older than rec, or nil.
func lastSuccessBefore(db storage.DB, rec *types.ExecutionHistory) (*types.ExecutionHistory, error) {
	history, err := db.GetHistory(rec.RequestID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	pos := -1
	for i, h := range history {
		if h.ID == rec.ID {
			pos = i
			break
		}
	}
	for i, h := range history {
		older := i > pos
		if pos < 0 {
			older = h.Time().Before(rec.Time())
		}
		if older && h.ID != rec.ID && !h.Failed() {
			return h, nil
		}
	}
	return nil, nil
}

func compareExecutions(base, rec *types.ExecutionHistory) *diagComparison {
	if base == nil {
		return nil
	}
	cmp := &diagComparison{
		BaselineID:         base.ID,
		BaselineTime:       base.Time(),
		BaselineStatus:     base.StatusCode,
		Status:             rec.StatusCode,
		BaselineDurationMs: base.DurationMs,
		DurationDeltaMs:    rec.DurationMs - base.DurationMs,
		RequestCompared:    base.URL != "" && rec.URL != "",
	}
	if cmp.RequestCompared {
		cmp.Request = compareRequests(base, rec)
		cmp.Vars = compareVars(base.Vars, rec.Vars)
	}
	if base.ResponseHeaders != nil && rec.ResponseHeaders != nil {
		cmp.ResponseHeadersAdded, cmp.ResponseHeadersRemoved = compareKeys(headerKeys(base.ResponseHeaders), headerKeys(rec.ResponseHeaders))
	}
	cmp.ResponseFields = compareJSONFields(base.Response, rec.Response)
	return cmp
}

func compareRequests(base, rec *types.ExecutionHistory) []diagChange {
	var changes []diagChange
	add := func(field, before, after string) {
		if before != after {
			changes = append(changes, diagChange{Field: field, Before: before, After: after, Added: before == "", Removed: after == ""})
		}
	}
	add("method", base.Method, rec.Method)
	add("url", base.URL, rec.URL)

	before, after := requestHeaderMap(base.RequestHeaders), requestHeaderMap(rec.RequestHeaders)
	for _, key := range unionKeys(before, after) {
		add("header "+key, before[key], after[key])
	}
	add("body", base.RequestBody, rec.RequestBody)
	return changes
}

func requestHeaderMap(headers []types.Header) map[string]string {
	m := make(map[string]string, len(headers))
	for _, h := range headers {
		key := http.CanonicalHeaderKey(h.Key)
		if prev, ok := m[key]; ok {
			m[key] = prev + ", " + h.Value
			continue
		}
		m[key] = h.Value
	}
	return m
}

func compareVars(before, after []types.VarRecord) []diagVarChange {
	b := make(map[string]types.VarRecord, len(before))
	for _, v := range before {
		b[v.Name] = v
	}
	a := make(map[string]types.VarRecord, len(after))
	for _, v := range after {
		a[v.Name] = v
	}
	names := make(map[string]bool, len(a)+len(b))
	for name := range a {
		names[name] = true
	}
	for name := range b {
		names[name] = true
	}
	var changes []diagVarChange
	for _, name := range sortedBoolKeys(names) {
		bv, inBefore := b[name]
		av, inAfter := a[name]
		secret := bv.Secret || av.Secret
		change := diagVarChange{Name: name, BeforeOrigin: bv.Origin, AfterOrigin: av.Origin, Secret: secret}
		if !secret {
			change.Before, change.After = bv.Value, av.Value
		}
		switch {
		case inBefore != inAfter, bv.Origin != av.Origin:
		case !secret && bv.Value != av.Value:
		default:
			continue
		}
		changes = append(changes, change)
	}
	return changes
}

func headerKeys(headers map[string][]string) map[string]bool {
	keys := make(map[string]bool, len(headers))
	for key := range headers {
		keys[http.CanonicalHeaderKey(key)] = true
	}
	return keys
}

func compareKeys(before, after map[string]bool) (added, removed []string) {
	for key := range after {
		if !before[key] {
			added = append(added, key)
		}
	}
	for key := range before {
		if !after[key] {
			removed = append(removed, key)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// compareJSONFields compares the top-level fields of two JSON object bodies;
// it returns nil unless both bodies are JSON objects.
func compareJSONFields(before, after string) []diagChange {
	var b, a map[string]json.RawMessage
	if json.Unmarshal([]byte(before), &b) != nil || json.Unmarshal([]byte(after), &a) != nil || b == nil || a == nil {
		return nil
	}
	compact := func(raw json.RawMessage) string {
		if raw == nil {
			return ""
		}
		var buf bytes.Buffer
		if json.Compact(&buf, raw) != nil {
			return string(raw)
		}
		return buf.String()
	}
	var changes []diagChange
	for _, key := range unionRawKeys(b, a) {
		bv, av := compact(b[key]), compact(a[key])
		if bv != av {
			changes = append(changes, diagChange{Field: key, Before: bv, After: av, Added: b[key] == nil, Removed: a[key] == nil})
		}
	}
	return changes
}

func unionKeys(a, b map[string]string) []string {
	keys := make(map[string]bool, len(a)+len(b))
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	return sortedBoolKeys(keys)
}

func unionRawKeys(a, b map[string]json.RawMessage) []string {
	keys := make(map[string]bool, len(a)+len(b))
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	return sortedBoolKeys(keys)
}

func sortedBoolKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeDiag(out io.Writer, rec *types.ExecutionHistory, cmp *diagComparison, full bool) error {
	w := &diagWriter{out: out}

	w.section("Summary")
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	row := func(label, value string) {
		if value != "" {
			fmt.Fprintf(tw, "  %s:\t%s\n", label, value)
		}
	}
	row("Request", rec.RequestName)
	row("ID", fmt.Sprintf("%s (%s)", rec.ShortID(), rec.ID))
	row("Time", rec.Time().Format("2006-01-02 15:04:05"))
	row("Environment", rec.EnvName)
	row("Collection", rec.CollectionName)
	if rec.Failed() {
		row("Result", "FAILED: "+runner.FailureReason(rec))
	} else {
		row("Result", "ok")
	}
	row("Status", statusText(rec.StatusCode))
	row("Failure phase", rec.FailurePhase)
	row("Error", rec.Error)
	row("Duration", fmt.Sprintf("%dms", rec.DurationMs))
	if rec.StatusCode != 0 {
		row("Size", formatBytes(rec.SizeBytes))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if rec.URL != "" {
		w.section("Request")
		w.line("  %s %s", rec.Method, rec.URL)
		for _, h := range rec.RequestHeaders {
			w.line("  %s: %s", h.Key, h.Value)
		}
		if rec.RequestBody != "" {
			w.line("")
			w.block(prettyBody(rec.RequestBody))
		}
	}

	if len(rec.Vars) > 0 {
		w.section("Variables")
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		for _, v := range rec.Vars {
			value := v.Value
			if v.Secret {
				value += "  (secret)"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", v.Name, v.Origin, value)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}

	w.section("Response")
	if rec.StatusCode == 0 && rec.Response == "" {
		w.line("  no response")
	} else {
		w.line("  Status: %s", statusText(rec.StatusCode))
		for _, key := range sortedHeaderKeys(rec.ResponseHeaders) {
			for _, value := range rec.ResponseHeaders[key] {
				w.line("  %s: %s", key, value)
			}
		}
		if rec.Response != "" {
			w.line("")
			body := prettyBody(rec.Response)
			if !full && len(body) > diagBodyLimit {
				body = body[:diagBodyLimit] + fmt.Sprintf("\n... (truncated, %d bytes total; use --full)", len(body))
			}
			w.block(body)
		}
	}

	if len(rec.Assertions) > 0 {
		w.section("Assertions")
		for _, a := range rec.Assertions {
			mark := "PASS"
			if !a.Passed {
				mark = "FAIL"
			}
			w.line("  [%s] %s", mark, a.Name)
			if !a.Passed && a.Message != "" {
				w.line("         %s", a.Message)
			}
		}
	}

	w.section("Compared with last success")
	writeComparison(w, cmp)
	return w.err
}

func writeComparison(w *diagWriter, cmp *diagComparison) {
	if cmp == nil {
		w.line("  No earlier successful execution of this request.")
		return
	}
	w.line("  Baseline: %s at %s", types.ShortID(cmp.BaselineID), cmp.BaselineTime.Format("2006-01-02 15:04:05"))
	if cmp.BaselineStatus != cmp.Status {
		w.line("  status: %s -> %s", statusText(cmp.BaselineStatus), statusText(cmp.Status))
	}
	w.line("  duration: %+dms (%dms -> %dms)", cmp.DurationDeltaMs, cmp.BaselineDurationMs, cmp.BaselineDurationMs+cmp.DurationDeltaMs)
	if !cmp.RequestCompared {
		w.line("  (the baseline predates stored request details; request and variables not compared)")
	}
	for _, ch := range cmp.Request {
		w.line("  %s: %s", ch.Field, describeChange(ch, quoteShort))
	}
	for _, v := range cmp.Vars {
		w.line("  var %s: %s", v.Name, describeVarChange(v))
	}
	for _, key := range cmp.ResponseHeadersAdded {
		w.line("  response header added: %s", key)
	}
	for _, key := range cmp.ResponseHeadersRemoved {
		w.line("  response header removed: %s", key)
	}
	for _, ch := range cmp.ResponseFields {
		w.line("  response field %q: %s", ch.Field, describeChange(ch, shortJSON))
	}
	if cmp.BaselineStatus == cmp.Status && len(cmp.Request) == 0 && len(cmp.Vars) == 0 &&
		len(cmp.ResponseHeadersAdded) == 0 && len(cmp.ResponseHeadersRemoved) == 0 && len(cmp.ResponseFields) == 0 {
		w.line("  No other differences.")
	}
}

// describeChange describes ch, formatting values with show: quoteShort for
// plain text, shortJSON for JSON values (already quoted).
func describeChange(ch diagChange, show func(string) string) string {
	switch {
	case ch.Added:
		return "added " + show(ch.After)
	case ch.Removed:
		return "removed (was " + show(ch.Before) + ")"
	}
	return show(ch.Before) + " -> " + show(ch.After)
}

func describeVarChange(v diagVarChange) string {
	var desc string
	switch {
	case v.BeforeOrigin == "":
		desc = "added (" + v.AfterOrigin + ")"
		if !v.Secret {
			desc += " " + quoteShort(v.After)
		}
	case v.AfterOrigin == "":
		desc = "removed (was " + v.BeforeOrigin + ")"
	default:
		var parts []string
		if v.BeforeOrigin != v.AfterOrigin {
			parts = append(parts, "origin "+v.BeforeOrigin+" -> "+v.AfterOrigin)
		}
		if !v.Secret && v.Before != v.After {
			parts = append(parts, "value "+quoteShort(v.Before)+" -> "+quoteShort(v.After))
		}
		desc = strings.Join(parts, ", ")
	}
	if v.Secret {
		desc += " (secret; values are masked and not compared)"
	}
	return desc
}

// diagValueLimit is how much of a changed value diag prints.
const diagValueLimit = 80

func quoteShort(s string) string {
	return fmt.Sprintf("%q", truncate(s, diagValueLimit))
}

func shortJSON(s string) string {
	return truncate(s, diagValueLimit)
}

func statusText(code int) string {
	if code == 0 {
		return "none"
	}
	if text := http.StatusText(code); text != "" {
		return fmt.Sprintf("%d %s", code, text)
	}
	return fmt.Sprintf("%d", code)
}

func prettyBody(body string) string {
	var buf bytes.Buffer
	if json.Indent(&buf, []byte(body), "", "  ") == nil {
		return buf.String()
	}
	return body
}

func sortedHeaderKeys(headers map[string][]string) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// diagWriter writes diag output and keeps the first write error.
type diagWriter struct {
	out     io.Writer
	err     error
	started bool
}

func (w *diagWriter) line(format string, args ...any) {
	if w.err == nil {
		_, w.err = fmt.Fprintf(w.out, format+"\n", args...)
	}
}

func (w *diagWriter) section(title string) {
	if w.started {
		w.line("")
	}
	w.started = true
	w.line("%s", title)
}

func (w *diagWriter) block(text string) {
	for _, l := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		w.line("  %s", l)
	}
}
