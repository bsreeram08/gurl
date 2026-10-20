package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/assertions"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/formatter"
	"github.com/sreeram/gurl/internal/plugins"
	"github.com/sreeram/gurl/internal/plugins/builtins"
	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

var outputPluginRegistry *plugins.Registry

func init() {
	outputPluginRegistry = plugins.NewRegistry()
	builtins.RegisterBuiltins(outputPluginRegistry)
}

// SetPluginRegistry replaces the process-wide plugin registry after .so discovery.
func SetPluginRegistry(r *plugins.Registry) {
	if r != nil {
		outputPluginRegistry = r
	}
}

// RunCommand creates the run command. runnerOpts configure the runner that
// sends the requests, e.g. runner.WithCookieJar for the shared cookie jar.
func RunCommand(db storage.DB, envStorage env.Store, runnerOpts ...runner.Option) *cli.Command {
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r", "execute"},
		Usage:   "Execute a saved request",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Environment name to use for variable substitution",
			},
			&cli.StringSliceFlag{
				Name:    "var",
				Aliases: []string{"v"},
				Usage:   "Variable substitution (--var key=value)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format (auto|json|table)",
				Value:   "auto",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (use - for stdout)",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Force overwrite existing file",
			},
			&cli.StringFlag{
				Name:  "timeout",
				Usage: "Request timeout (e.g., 5s, 1m, 30s)",
			},
			&cli.BoolFlag{
				Name:    "chain",
				Aliases: []string{"ch"},
				Usage:   "Enable request chaining via setNextRequest",
			},
			&cli.BoolFlag{
				Name:  "persist",
				Usage: "Persist extracted/script variables back to the selected environment",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview request execution without sending it",
			},
			&cli.StringSliceFlag{
				Name:    "assert",
				Aliases: []string{"a"},
				Usage:   "Assertion to evaluate (e.g., status=200, body contains success)",
			},
			&cli.BoolFlag{
				Name:  "fail",
				Usage: "Exit non-zero when the response status is not 2xx (body is still printed)",
			},
			&cli.StringFlag{
				Name:    "data",
				Aliases: []string{"d"},
				Usage:   "Data file (CSV or JSON) for data-driven iteration",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "JSON object per execution (NDJSON for --chain/--data), nothing else on stdout",
			},
		},
		Action: jsonAction(func(ctx context.Context, c *cli.Command) error {
			if agentMode(c) && c.String("output") != "" {
				// The file would hold the unmasked body outside agent-mode redaction.
				return cli.Exit("--output is not allowed in agent mode", 2)
			}
			args := c.Args()
			var name string
			if args.Len() < 1 {
				if noInput(c) {
					return cli.Exit("request name is required: gurl run <name> (list names with: gurl list --json)", 2)
				}
				requests, err := db.ListRequests(nil)
				if err != nil {
					return fmt.Errorf("failed to list requests: %w", err)
				}
				chosen, err := promptSelectRequest(bufio.NewReader(os.Stdin), os.Stdout, requests, "Run request by number or exact name: ")
				if err != nil {
					return fmt.Errorf("request selection failed: %w", err)
				}
				if chosen == "" {
					return nil
				}
				name = chosen
			} else {
				name = args.Get(0)
			}
			if c.Bool("persist") && c.Bool("dry-run") {
				return cli.Exit("--persist and --dry-run cannot be used together", 2)
			}

			enableChain := c.Bool("chain")
			dataFile := c.String("data")
			requestedEnv := c.String("env")
			persistEnvName := ""
			if c.Bool("persist") {
				var err error
				persistEnvName, err = env.ResolveOptionalPersistEnvironmentName(envStorage, requestedEnv)
				if err != nil {
					return cli.Exit(err.Error(), 2)
				}
				if requestedEnv == "" && persistEnvName != "" {
					requestedEnv = persistEnvName
				}
			}

			var runEnv *env.Environment
			if requestedEnv != "" {
				e, err := envStorage.GetEnvByName(requestedEnv)
				if err != nil || e == nil {
					return fmt.Errorf("environment '%s' not found", requestedEnv)
				}
				runEnv = e
			} else if activeEnvName, err := envStorage.GetActiveEnv(); err == nil && activeEnvName != "" {
				if e, err := envStorage.GetEnvByName(activeEnvName); err == nil && e != nil {
					runEnv = e
				}
			}

			cliVars := make(map[string]string)
			for _, pair := range c.StringSlice("var") {
				if idx := strings.Index(pair, "="); idx != -1 {
					cliVars[pair[:idx]] = pair[idx+1:]
				}
			}

			if c.Bool("dry-run") {
				if dataFile != "" || enableChain {
					return cli.Exit("--dry-run cannot be combined with --data or --chain", 2)
				}
				if jsonOutput(c) {
					return executeDryRunJSON(ctx, db, envStorage, name, runEnv, cliVars, os.Stdout)
				}
				return executeDryRun(ctx, db, envStorage, name, runEnv, cliVars, os.Stdout)
			}

			if dataFile != "" {
				return executeDataDriven(ctx, db, envStorage, name, runEnv, cliVars, c, persistEnvName, runnerOpts)
			}

			if enableChain {
				return executeChain(ctx, db, envStorage, name, runEnv, cliVars, c, persistEnvName, runnerOpts)
			}

			return executeSingleRequest(ctx, db, envStorage, name, runEnv, cliVars, c, persistEnvName, runnerOpts)
		}),
	}
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func executeChain(ctx context.Context, db storage.DB, envStorage env.Store, name string, runEnv *env.Environment, cliVars map[string]string, c *cli.Command, persistEnvName string, runnerOpts []runner.Option) error {
	currentName := name
	visited := make(map[string]bool)
	singleRunner := runner.NewRunner(db, envStorage, runnerOpts...)
	var chainAssertionErr error
	dirtyForPersist := make(map[string]string)
	dirtyOriginsForPersist := make(map[string]resolve.Origin)
	dirtyCollectionTargets := make(map[string]string)
	currentCollectionName := ""
	chainStarted := false
	jsonOut, strict := jsonOutput(c), agentMode(c)
	reported := func(err error) error {
		if jsonOut {
			return markReported(err)
		}
		return err
	}
	// In JSON mode each step is one line; the persist summary goes on the
	// last one.
	steps := &ndjsonWriter{w: os.Stdout}
	finish := func(err error) error {
		if !c.Bool("persist") && !jsonOut {
			warnUnpersisted(os.Stderr, dirtyForPersist)
		}
		var persisted *runner.PersistSummary
		if c.Bool("persist") && chainStarted {
			summary, persistErr := runner.PersistDirtyVariablesWithCollectionTargets(envStorage, db, persistEnvName, currentCollectionName, dirtyForPersist, dirtyOriginsForPersist, dirtyCollectionTargets)
			if persistErr != nil {
				return errors.Join(persistErr, steps.finish(nil))
			}
			persisted = summary
			if !jsonOut {
				runner.PrintPersistSummaries(os.Stderr, persisted)
			}
		}
		if werr := steps.finish(persistedOutputOf(persisted)); werr != nil {
			return werr
		}
		return err
	}
	const maxChainIterations = 100
	// Each step's failure notice is printed after its response body.
	var failed *types.ExecutionHistory
	stdout := newLineTracker(os.Stdout)
	if !jsonOut {
		defer func() { printFailureNotice(os.Stderr, failed, stdout) }()
	}

	for i := 0; i < maxChainIterations; i++ {
		req, err := db.GetRequestByName(currentName)
		if err != nil {
			return finish(fmt.Errorf("request not found: %s", currentName))
		}

		if visited[currentName] {
			return finish(fmt.Errorf("next request loop detected: %q would revisit an already executed request", currentName))
		}
		visited[currentName] = true
		if c.Bool("persist") && req.Collection == "" && persistEnvName == "" {
			return finish(fmt.Errorf("--persist requires --env or an active environment"))
		}
		scopes := runner.ScopesFor(db, runEnv, req.Collection, cliVars)
		scopes.Runtime = make(map[string]resolve.Var, len(dirtyForPersist))
		for key, value := range dirtyForPersist {
			scopes.Runtime[key] = resolve.Var{Value: value, Origin: dirtyOriginsForPersist[key]}
		}
		currentCollectionName = req.Collection

		if timeoutStr := c.String("timeout"); timeoutStr != "" {
			if _, err := time.ParseDuration(timeoutStr); err != nil {
				return finish(fmt.Errorf("invalid timeout format '%s': %w", timeoutStr, err))
			}
			copy := *req
			copy.Timeout = timeoutStr
			req = &copy
		} else if req.Timeout != "" {
			if _, err := time.ParseDuration(req.Timeout); err != nil {
				return finish(fmt.Errorf("invalid timeout format '%s': %w", req.Timeout, err))
			}
		}

		execution := singleRunner.RunSavedRequestInScopes(ctx, req, scopes)
		chainStarted = true
		for key, value := range execution.Result.DirtyVars {
			dirtyForPersist[key] = value
			if origin, ok := execution.Result.DirtyVarOrigins[key]; ok {
				dirtyOriginsForPersist[key] = origin
				if origin != resolve.OriginEnvironment && req.Collection != "" {
					dirtyCollectionTargets[key] = req.Collection
				} else {
					delete(dirtyCollectionTargets, key)
				}
			}
		}
		if execution.Result.Error != "" {
			rec := execution.Record(envName(runEnv))
			failed = keepRecord(db, rec)
			if jsonOut {
				steps.add(executionOutput(execution, rec, strict))
			}
			return finish(reported(fmt.Errorf("%s", rec.Error))) // redacted copy of Result.Error
		}
		var assertionErr error
		if execution.Result.FailurePhase == runner.FailurePhaseAssertion {
			assertionErr = fmt.Errorf("assertion failed: %s", runnerAssertionFailureMessage(execution.Result))
			chainAssertionErr = mergeAssertionErrors(chainAssertionErr, assertionErr)
		}
		if execution.Result.Skipped || execution.Request == nil || execution.Response == nil {
			if jsonOut && execution.Result.Skipped {
				steps.add(skippedOutput(execution, envName(runEnv)))
			}
			if execution.Result.NextRequestOverride == "" {
				return finish(reported(chainAssertionErr))
			}
			currentName = execution.Result.NextRequestOverride
			continue
		}
		clientReq := *execution.Request
		resp := *execution.Response

		rec := execution.Record(envName(runEnv))
		if err := db.SaveHistory(rec); err != nil {
			return finish(fmt.Errorf("failed to save history: %w", err))
		}
		failed = rec

		if outputPath := c.String("output"); outputPath != "" {
			if !c.Bool("force") {
				cwd, err := os.Getwd()
				if err == nil {
					if err := ValidateSafePath(outputPath, cwd); err != nil {
						return finish(fmt.Errorf("output path escapes allowed directory: %w", err))
					}
				}
			}
			resp.URL = clientReq.URL
			force := c.Bool("force")
			if err := client.SaveToFile(&resp, outputPath, force); err != nil {
				return finish(fmt.Errorf("failed to save output: %w", err))
			}
		}

		if jsonOut {
			steps.add(executionOutput(execution, rec, strict))
		} else {
			format := c.String("format")
			if err := printResponse(stdout, clientReq.Method, clientReq.URL, resp, format); err != nil {
				return finish(err)
			}
			printFailureNotice(os.Stderr, failed, stdout)
		}
		failed = nil
		chainAssertionErr = mergeAssertionErrors(chainAssertionErr, httpFailure(c, resp.StatusCode))
		nextReq := execution.Result.NextRequestOverride
		if nextReq == "" {
			return finish(reported(chainAssertionErr))
		}

		currentName = nextReq
	}

	return finish(fmt.Errorf("max iterations (%d) reached", maxChainIterations))
}

func executeSingleRequest(ctx context.Context, db storage.DB, envStorage env.Store, name string, runEnv *env.Environment, cliVars map[string]string, c *cli.Command, persistEnvName string, runnerOpts []runner.Option) error {
	req, err := db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
	}
	if c.Bool("persist") && req.Collection == "" && persistEnvName == "" {
		return fmt.Errorf("--persist requires --env or an active environment")
	}

	if timeoutStr := c.String("timeout"); timeoutStr != "" {
		copy := *req
		copy.Timeout = timeoutStr
		req = &copy
	}

	scopes := runner.ScopesFor(db, runEnv, req.Collection, cliVars)
	execution := runner.NewRunner(db, envStorage, runnerOpts...).RunSavedRequestInScopes(ctx, req, scopes)
	jsonOut, strict := jsonOutput(c), agentMode(c)
	reported := func(err error) error {
		if jsonOut {
			return markReported(err)
		}
		return err
	}
	// The failure notice is printed last, after the response body. JSON
	// output carries the record id instead.
	var failed *types.ExecutionHistory
	stdout := newLineTracker(os.Stdout)
	if !jsonOut {
		defer func() { printFailureNotice(os.Stderr, failed, stdout) }()
	}
	var persisted *runner.PersistSummary
	if c.Bool("persist") {
		summary, persistErr := runner.PersistDirtyVariables(envStorage, db, persistEnvName, req.Collection, execution.Result.DirtyVars, execution.Result.DirtyVarOrigins)
		if persistErr != nil {
			return persistErr
		}
		persisted = summary
		if !jsonOut {
			runner.PrintPersistSummaries(os.Stderr, persisted)
			if execution.Result.Error != "" && len(execution.Result.DirtyVars) > 0 {
				fmt.Fprintf(os.Stderr, "Run aborted after persisting %d variables.\n", len(execution.Result.DirtyVars))
			}
		}
	} else if !jsonOut {
		warnUnpersisted(os.Stderr, execution.Result.DirtyVars)
	}
	emit := func(out runOutput) error {
		out.Persisted = persistedOutputOf(persisted)
		return writeJSONLine(os.Stdout, out)
	}
	if execution.Result.Error != "" {
		rec := execution.Record(envName(runEnv))
		failed = keepRecord(db, rec)
		if jsonOut {
			if err := emit(executionOutput(execution, rec, strict)); err != nil {
				return err
			}
		}
		return reported(fmt.Errorf("%s", rec.Error)) // redacted copy of Result.Error
	}
	var assertionErr error
	if execution.Result.FailurePhase == runner.FailurePhaseAssertion {
		assertionErr = fmt.Errorf("assertion failed: %s", runnerAssertionFailureMessage(execution.Result))
	}
	if execution.Result.Skipped || execution.Request == nil || execution.Response == nil {
		if jsonOut && execution.Result.Skipped {
			return emit(skippedOutput(execution, envName(runEnv)))
		}
		return nil
	}
	clientReq := *execution.Request
	resp := *execution.Response
	rec := execution.Record(envName(runEnv))

	// Evaluate assertions if provided
	if assertFlags := c.StringSlice("assert"); len(assertFlags) > 0 {
		parser := assertions.NewCLIParser()
		asserts, err := parser.ParseSlice(assertFlags)
		if err != nil {
			failed = keepRecord(db, rec)
			return fmt.Errorf("failed to parse assertions: %w", err)
		}
		evaluator := assertions.NewEvaluator()
		results := evaluator.Evaluate(&resp, asserts, assertionVarsFromResult(execution.Result))
		summary := assertions.Summarize(results)

		if !jsonOut {
			fmt.Fprintf(os.Stderr, "\n=== Assertions: %d passed, %d failed ===\n", summary.Passed, summary.Failed)
			for _, r := range results {
				fmt.Fprintf(os.Stderr, "%s\n", r.Message)
			}
		}

		rec.Assertions = append(rec.Assertions, runner.AssertionRecords(results)...)
		if summary.Failed > 0 {
			assertionErr = mergeAssertionErrors(assertionErr, fmt.Errorf("assertion failed: %s", cliAssertionFailureMessage(results)))
			if rec.FailurePhase == "" {
				rec.FailurePhase = runner.FailurePhaseAssertion
			}
		}
	}

	if err := db.SaveHistory(rec); err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}
	failed = rec

	if outputPath := c.String("output"); outputPath != "" {
		if !c.Bool("force") {
			cwd, err := os.Getwd()
			if err == nil {
				if err := ValidateSafePath(outputPath, cwd); err != nil {
					return fmt.Errorf("output path escapes allowed directory: %w", err)
				}
			}
		}
		resp.URL = clientReq.URL
		force := c.Bool("force")
		if err := client.SaveToFile(&resp, outputPath, force); err != nil {
			return fmt.Errorf("failed to save output: %w", err)
		}
		if !jsonOut {
			fmt.Fprintf(os.Stderr, "Response saved to %s\n", outputPath)
			return mergeAssertionErrors(assertionErr, httpFailure(c, resp.StatusCode))
		}
	}

	if jsonOut {
		if err := emit(executionOutput(execution, rec, strict)); err != nil {
			return err
		}
		return reported(mergeAssertionErrors(assertionErr, httpFailure(c, resp.StatusCode)))
	}
	format := c.String("format")
	if err := printResponse(stdout, clientReq.Method, clientReq.URL, resp, format); err != nil {
		return err
	}
	return mergeAssertionErrors(assertionErr, httpFailure(c, resp.StatusCode))
}

// envName is the name of e, or "" for no environment.
func envName(e *env.Environment) string {
	if e == nil {
		return ""
	}
	return e.Name
}

// keepRecord saves rec, the record of a failed execution, and returns it for
// the failure notice. It returns nil, after a warning on stderr, when there
// is no record or it cannot be saved.
func keepRecord(db storage.DB, rec *types.ExecutionHistory) *types.ExecutionHistory {
	if rec == nil {
		return nil
	}
	if err := db.SaveHistory(rec); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save execution record: %v\n", err)
		return nil
	}
	return rec
}

// printFailureNotice prints the failure notice of rec when it failed,
// starting a new line when the response body printed to stdout did not end
// one (stdout may be nil).
func printFailureNotice(w io.Writer, rec *types.ExecutionHistory, stdout *lineTracker) {
	if rec == nil {
		return
	}
	notice := runner.FailureNotice(rec)
	if notice == "" {
		return
	}
	if stdout != nil && stdout.midLine {
		fmt.Fprintln(w)
		stdout.midLine = false
	}
	fmt.Fprintln(w, notice)
}

// lineTracker is a writer that remembers whether the last write to a
// terminal ended mid-line, so a notice on the same terminal starts on its
// own line.
type lineTracker struct {
	w        io.Writer
	terminal bool
	midLine  bool
}

func newLineTracker(f *os.File) *lineTracker {
	return &lineTracker{w: f, terminal: term.IsTerminal(int(f.Fd()))}
}

func (t *lineTracker) Write(p []byte) (int, error) {
	n, err := t.w.Write(p)
	if n > 0 && t.terminal {
		t.midLine = p[n-1] != '\n'
	}
	return n, err
}

// httpFailure returns an error for non-2xx statuses when --fail is set.
func httpFailure(c *cli.Command, status int) error {
	if !c.Bool("fail") || (status >= 200 && status < 300) {
		return nil
	}
	return fmt.Errorf("HTTP %d", status)
}

// warnUnpersisted tells the user that extracted/script variables are dropped
// at exit unless --persist is passed, so multi-step flows don't reuse stale ids.
func warnUnpersisted(out io.Writer, dirty map[string]string) {
	if len(dirty) == 0 {
		return
	}
	fmt.Fprintf(out, "note: %s not saved; add --persist to use them in later runs\n", strings.Join(sortedStringKeys(dirty), ", "))
}

// executeDryRun prints the request as it would be sent, after pre-request
// scripts, and fails when it cannot be resolved. Secret values are masked in
// the output.
func executeDryRun(ctx context.Context, db storage.DB, envStorage env.Store, name string, runEnv *env.Environment, cliVars map[string]string, out io.Writer) error {
	req, err := db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
	}
	scopes := runner.ScopesFor(db, runEnv, req.Collection, cliVars)
	execution := runner.NewRunner(db, envStorage).PlanSavedRequest(ctx, req, scopes)
	result := execution.Result
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	if result.Skipped {
		fmt.Fprintf(out, "Skipped: %s\n", result.SkipReason)
		return nil
	}

	secrets := redact.ScopeSecrets(scopes, result.DirtyVars)
	mask := func(s string) string { return redact.Replace(s, secrets) }

	preview := execution.Request
	fmt.Fprintf(out, "%s %s\n", result.PlannedMethod, mask(result.PlannedURL))
	for _, h := range preview.Headers {
		value := h.Value
		if redact.SensitiveName(h.Key) && redact.Literal(value) {
			value = redact.Mask
		}
		fmt.Fprintf(out, "%s: %s\n", h.Key, mask(value))
	}
	if preview.Body != "" {
		fmt.Fprintf(out, "\n%s\n", mask(preview.Body))
	}
	for _, extract := range result.PlannedExtracts {
		fmt.Fprintf(out, "extract: %s <- %s\n", extract.Name, extract.Source)
	}
	if len(result.DryRunWarnings) > 0 {
		return fmt.Errorf("dry run found %s", strings.Join(result.DryRunWarnings, ", "))
	}
	if err := result.DryRunProblems.Err(); err != nil {
		return fmt.Errorf("dry run: %w", err)
	}
	return nil
}

// executeDryRunJSON is executeDryRun printing one JSON object with
// "dry_run": true and structured problems. Secret values are masked as in
// executeDryRun.
func executeDryRunJSON(ctx context.Context, db storage.DB, envStorage env.Store, name string, runEnv *env.Environment, cliVars map[string]string, out io.Writer) error {
	req, err := db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
	}
	scopes := runner.ScopesFor(db, runEnv, req.Collection, cliVars)
	execution := runner.NewRunner(db, envStorage).PlanSavedRequest(ctx, req, scopes)
	result := execution.Result
	secrets := execution.Secrets()
	mask := func(s string) string { return redact.Replace(s, secrets) }

	preview := runOutput{DryRun: true, Request: req.Name, Collection: req.Collection, Environment: envName(runEnv)}
	if result.Error != "" {
		preview.Error = mask(result.Error)
		preview.FailurePhase = result.FailurePhase
		if err := writeJSONLine(out, preview); err != nil {
			return err
		}
		return markReported(fmt.Errorf("%s", preview.Error))
	}
	if result.Skipped {
		preview.OK = true
		preview.Skipped = true
		preview.SkipReason = result.SkipReason
		return writeJSONLine(out, preview)
	}

	preview.Method = result.PlannedMethod
	preview.URL = mask(result.PlannedURL)
	for _, h := range execution.Request.Headers {
		value := h.Value
		if redact.SensitiveName(h.Key) && redact.Literal(value) {
			value = redact.Mask
		}
		preview.Headers = append(preview.Headers, types.Header{Key: h.Key, Value: mask(value)})
	}
	preview.Body = jsonBody(mask(execution.Request.Body))
	preview.Extracts = result.PlannedExtracts
	preview.Problems = problemOutputs(result.Problems, secrets)
	preview.OK = len(preview.Problems) == 0
	if err := writeJSONLine(out, preview); err != nil {
		return err
	}
	if len(result.DryRunWarnings) > 0 {
		return markReported(fmt.Errorf("dry run found %s", strings.Join(result.DryRunWarnings, ", ")))
	}
	if err := result.Problems.Err(); err != nil {
		return markReported(fmt.Errorf("dry run: %w", err))
	}
	return nil
}

func runnerAssertionFailureMessage(result *runner.RequestResult) string {
	messages := make([]string, 0, len(result.AssertionResults))
	for _, assertion := range result.AssertionResults {
		if assertion.Passed {
			continue
		}
		messages = append(messages, assertion.Message)
	}
	if len(messages) == 0 {
		return "one or more assertions failed"
	}
	return strings.Join(messages, "; ")
}

func cliAssertionFailureMessage(results []assertions.Result) string {
	messages := make([]string, 0, len(results))
	for _, assertion := range results {
		if assertion.Passed {
			continue
		}
		messages = append(messages, assertion.Message)
	}
	if len(messages) == 0 {
		return "one or more assertions failed"
	}
	return strings.Join(messages, "; ")
}

func mergeAssertionErrors(existing error, next error) error {
	if existing == nil {
		return next
	}
	if next == nil {
		return existing
	}
	return fmt.Errorf("%v; %v", existing, next)
}

func assertionVarsFromResult(result *runner.RequestResult) map[string]string {
	vars := make(map[string]string)
	for key, value := range result.ExtractedVars {
		vars[key] = value
	}
	for key, value := range result.DirtyVars {
		vars[key] = value
	}
	return vars
}

func executeDataDriven(ctx context.Context, db storage.DB, envStorage env.Store, name string, runEnv *env.Environment, cliVars map[string]string, c *cli.Command, persistEnvName string, runnerOpts []runner.Option) error {
	req, err := db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
	}
	if c.Bool("persist") && req.Collection == "" && persistEnvName == "" {
		return fmt.Errorf("--persist requires --env or an active environment")
	}
	if timeoutStr := c.String("timeout"); timeoutStr != "" {
		if _, err := time.ParseDuration(timeoutStr); err != nil {
			return fmt.Errorf("invalid timeout format '%s': %w", timeoutStr, err)
		}
	} else if req.Timeout != "" {
		if _, err := time.ParseDuration(req.Timeout); err != nil {
			return fmt.Errorf("invalid timeout format '%s': %w", req.Timeout, err)
		}
	}

	dataFile := c.String("data")
	if _, err := os.Stat(dataFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("data file not found: %s", dataFile)
		}
		return fmt.Errorf("failed to access data file: %w", err)
	}
	loader, err := runner.NewDataLoader(dataFile)
	if err != nil {
		return fmt.Errorf("failed to load data file: %w", err)
	}

	rowNum := 0
	dirtyForPersist := make(map[string]string)
	dirtyOriginsForPersist := make(map[string]resolve.Origin)
	singleRunner := runner.NewRunner(db, envStorage, runnerOpts...)
	// In JSON mode each row is one line; the persist summary goes on the
	// last one.
	jsonOut, strict := jsonOutput(c), agentMode(c)
	rows := &ndjsonWriter{w: os.Stdout}
	err = loader.Iterate(func(rowVars map[string]string) error {
		rowNum++

		effectiveReq := req
		if timeoutStr := c.String("timeout"); timeoutStr != "" {
			copy := *req
			copy.Timeout = timeoutStr
			effectiveReq = &copy
		}

		scopes := runner.ScopesFor(db, runEnv, effectiveReq.Collection, cliVars)
		scopes.Data = rowVars
		execution := singleRunner.RunSavedRequestInScopes(ctx, effectiveReq, scopes)
		for key, value := range execution.Result.DirtyVars {
			dirtyForPersist[key] = value
			if origin, ok := execution.Result.DirtyVarOrigins[key]; ok {
				dirtyOriginsForPersist[key] = origin
			}
		}
		if execution.Result.Error != "" {
			rec := execution.Record(envName(runEnv))
			kept := keepRecord(db, rec)
			if jsonOut {
				out := executionOutput(execution, rec, strict)
				out.Row = rowNum
				rows.add(out)
				return nil
			}
			fmt.Fprintf(os.Stderr, "Row %d: request failed: %v\n", rowNum, rec.Error)
			printFailureNotice(os.Stderr, kept, nil)
			return nil
		}
		if execution.Result.Skipped || execution.Response == nil {
			if jsonOut {
				out := skippedOutput(execution, envName(runEnv))
				out.Row = rowNum
				rows.add(out)
				return nil
			}
			fmt.Fprintf(os.Stdout, "Row %d [%s]: skipped\n", rowNum, name)
			return nil
		}
		resp := *execution.Response

		built := execution.Record(envName(runEnv))
		rec := built
		if err := db.SaveHistory(rec); err != nil {
			fmt.Fprintf(os.Stderr, "Row %d: failed to save history: %v\n", rowNum, err)
			rec = nil
		}

		if jsonOut {
			out := executionOutput(execution, built, strict)
			out.Row = rowNum
			rows.add(out)
			return nil
		}
		fmt.Fprintf(os.Stdout, "Row %d [%s]: %d (%dms)\n", rowNum, name, resp.StatusCode, resp.Duration.Milliseconds())
		printFailureNotice(os.Stderr, rec, nil)

		return nil
	})

	if err != nil {
		return errors.Join(fmt.Errorf("data iteration failed: %w", err), rows.finish(nil))
	}
	var persisted *runner.PersistSummary
	if c.Bool("persist") {
		summary, persistErr := runner.PersistDirtyVariables(envStorage, db, persistEnvName, req.Collection, dirtyForPersist, dirtyOriginsForPersist)
		if persistErr != nil {
			return errors.Join(persistErr, rows.finish(nil))
		}
		persisted = summary
		if !jsonOut {
			runner.PrintPersistSummaries(os.Stderr, persisted)
		}
	}

	return rows.finish(persistedOutputOf(persisted))
}

func convertHeaders(headers []types.Header) []client.Header {
	result := make([]client.Header, len(headers))
	for i, h := range headers {
		result[i] = client.Header{Key: h.Key, Value: h.Value}
	}
	return result
}

func printResponse(out io.Writer, method string, url string, resp client.Response, format string) error {
	if outputPlugin, found := outputPluginRegistry.GetOutputByFormat(format); found {
		ctx := &plugins.ResponseContext{
			Request: &client.Request{
				Method: method,
				URL:    url,
			},
			Response: &resp,
		}
		_, err := io.WriteString(out, outputPlugin.Render(ctx))
		return err
	}

	switch format {
	case "json":
		var data interface{}
		if json.Unmarshal(resp.Body, &data) == nil {
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(data)
		}
		_, err := out.Write(resp.Body)
		return err
	case "table":
		if tableOutput := formatter.FormatTableFromBytes(resp.Body); tableOutput != "" {
			_, err := io.WriteString(out, tableOutput)
			return err
		}
		_, err := out.Write(resp.Body)
		return err
	default:
		_, err := out.Write(resp.Body)
		return err
	}
}
