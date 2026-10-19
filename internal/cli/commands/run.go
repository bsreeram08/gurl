package commands

import (
	"bufio"
	"context"
	"encoding/json"
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
	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
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

func RunCommand(db storage.DB, envStorage *env.EnvStorage) *cli.Command {
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
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			var name string
			if args.Len() < 1 {
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
				return executeDryRun(ctx, db, envStorage, name, runEnv, cliVars, os.Stdout)
			}

			if dataFile != "" {
				return executeDataDriven(ctx, db, envStorage, name, runEnv, cliVars, c, persistEnvName)
			}

			if enableChain {
				return executeChain(ctx, db, envStorage, name, runEnv, cliVars, c, persistEnvName)
			}

			return executeSingleRequest(ctx, db, envStorage, name, runEnv, cliVars, c, persistEnvName)
		},
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

func executeChain(ctx context.Context, db storage.DB, envStorage *env.EnvStorage, name string, runEnv *env.Environment, cliVars map[string]string, c *cli.Command, persistEnvName string) error {
	currentName := name
	visited := make(map[string]bool)
	singleRunner := runner.NewRunner(db, envStorage)
	var chainAssertionErr error
	dirtyForPersist := make(map[string]string)
	dirtyOriginsForPersist := make(map[string]resolve.Origin)
	dirtyCollectionTargets := make(map[string]string)
	currentCollectionName := ""
	chainStarted := false
	finish := func(err error) error {
		if !c.Bool("persist") {
			warnUnpersisted(os.Stderr, dirtyForPersist)
		}
		if c.Bool("persist") && chainStarted {
			collectionStore, _ := db.(storage.CollectionStore)
			persisted, persistErr := runner.PersistDirtyVariablesWithCollectionTargets(envStorage, collectionStore, persistEnvName, currentCollectionName, dirtyForPersist, dirtyOriginsForPersist, dirtyCollectionTargets)
			if persistErr != nil {
				return persistErr
			}
			runner.PrintPersistSummaries(os.Stderr, persisted)
		}
		return err
	}
	const maxChainIterations = 100

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
			return finish(fmt.Errorf("%s", execution.Result.Error))
		}
		var assertionErr error
		if execution.Result.FailurePhase == runner.FailurePhaseAssertion {
			assertionErr = fmt.Errorf("assertion failed: %s", runnerAssertionFailureMessage(execution.Result))
			chainAssertionErr = mergeAssertionErrors(chainAssertionErr, assertionErr)
		}
		if execution.Result.Skipped || execution.Request == nil || execution.Response == nil {
			if execution.Result.NextRequestOverride == "" {
				return finish(chainAssertionErr)
			}
			currentName = execution.Result.NextRequestOverride
			continue
		}
		clientReq := *execution.Request
		resp := *execution.Response

		history := types.NewExecutionHistory(
			req.ID,
			string(resp.Body),
			resp.StatusCode,
			resp.Duration.Milliseconds(),
			resp.Size,
		)
		if err := db.SaveHistory(history); err != nil {
			return finish(fmt.Errorf("failed to save history: %w", err))
		}

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

		format := c.String("format")
		if err := printResponse(os.Stdout, clientReq.Method, clientReq.URL, resp, format); err != nil {
			return finish(err)
		}
		chainAssertionErr = mergeAssertionErrors(chainAssertionErr, httpFailure(c, resp.StatusCode))
		nextReq := execution.Result.NextRequestOverride
		if nextReq == "" {
			return finish(chainAssertionErr)
		}

		currentName = nextReq
	}

	return finish(fmt.Errorf("max iterations (%d) reached", maxChainIterations))
}

func executeSingleRequest(ctx context.Context, db storage.DB, envStorage *env.EnvStorage, name string, runEnv *env.Environment, cliVars map[string]string, c *cli.Command, persistEnvName string) error {
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
	execution := runner.NewRunner(db, envStorage).RunSavedRequestInScopes(ctx, req, scopes)
	if c.Bool("persist") {
		collectionStore, _ := db.(storage.CollectionStore)
		persisted, persistErr := runner.PersistDirtyVariables(envStorage, collectionStore, persistEnvName, req.Collection, execution.Result.DirtyVars, execution.Result.DirtyVarOrigins)
		if persistErr != nil {
			return persistErr
		}
		runner.PrintPersistSummaries(os.Stderr, persisted)
		if execution.Result.Error != "" && len(execution.Result.DirtyVars) > 0 {
			fmt.Fprintf(os.Stderr, "Run aborted after persisting %d variables.\n", len(execution.Result.DirtyVars))
		}
	} else {
		warnUnpersisted(os.Stderr, execution.Result.DirtyVars)
	}
	if execution.Result.Error != "" {
		return fmt.Errorf("%s", execution.Result.Error)
	}
	var assertionErr error
	if execution.Result.FailurePhase == runner.FailurePhaseAssertion {
		assertionErr = fmt.Errorf("assertion failed: %s", runnerAssertionFailureMessage(execution.Result))
	}
	if execution.Result.Skipped || execution.Request == nil || execution.Response == nil {
		return nil
	}
	clientReq := *execution.Request
	resp := *execution.Response

	// Evaluate assertions if provided
	if assertFlags := c.StringSlice("assert"); len(assertFlags) > 0 {
		parser := assertions.NewCLIParser()
		asserts, err := parser.ParseSlice(assertFlags)
		if err != nil {
			return fmt.Errorf("failed to parse assertions: %w", err)
		}
		evaluator := assertions.NewEvaluator()
		results := evaluator.Evaluate(&resp, asserts, assertionVarsFromResult(execution.Result))
		summary := assertions.Summarize(results)

		fmt.Fprintf(os.Stderr, "\n=== Assertions: %d passed, %d failed ===\n", summary.Passed, summary.Failed)
		for _, r := range results {
			fmt.Fprintf(os.Stderr, "%s\n", r.Message)
		}

		if summary.Failed > 0 {
			assertionErr = mergeAssertionErrors(assertionErr, fmt.Errorf("assertion failed: %s", cliAssertionFailureMessage(results)))
		}
	}

	history := types.NewExecutionHistory(
		req.ID,
		string(resp.Body),
		resp.StatusCode,
		resp.Duration.Milliseconds(),
		resp.Size,
	)
	if err := db.SaveHistory(history); err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

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
		fmt.Fprintf(os.Stderr, "Response saved to %s\n", outputPath)
		return mergeAssertionErrors(assertionErr, httpFailure(c, resp.StatusCode))
	}

	format := c.String("format")
	if err := printResponse(os.Stdout, clientReq.Method, clientReq.URL, resp, format); err != nil {
		return err
	}
	return mergeAssertionErrors(assertionErr, httpFailure(c, resp.StatusCode))
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
func executeDryRun(ctx context.Context, db storage.DB, envStorage *env.EnvStorage, name string, runEnv *env.Environment, cliVars map[string]string, out io.Writer) error {
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

	vars := resolve.Vars(scopes)
	var secrets []string
	for key, v := range vars {
		if (v.Secret || isSensitiveName(key)) && v.Value != "" {
			secrets = append(secrets, v.Value)
		}
	}
	for key, value := range result.DirtyVars {
		if (vars[key].Secret || isSensitiveName(key)) && value != "" {
			secrets = append(secrets, value)
		}
	}
	mask := func(s string) string {
		for _, secret := range secrets {
			s = strings.ReplaceAll(s, secret, secretMask)
		}
		return s
	}

	preview := execution.Request
	fmt.Fprintf(out, "%s %s\n", result.PlannedMethod, mask(result.PlannedURL))
	for _, h := range preview.Headers {
		value := h.Value
		if isSensitiveName(h.Key) && shouldMaskValue(value) {
			value = secretMask
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

func executeDataDriven(ctx context.Context, db storage.DB, envStorage *env.EnvStorage, name string, runEnv *env.Environment, cliVars map[string]string, c *cli.Command, persistEnvName string) error {
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
	singleRunner := runner.NewRunner(db, envStorage)
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
			fmt.Fprintf(os.Stderr, "Row %d: request failed: %v\n", rowNum, execution.Result.Error)
			return nil
		}
		if execution.Result.Skipped || execution.Response == nil {
			fmt.Fprintf(os.Stdout, "Row %d [%s]: skipped\n", rowNum, name)
			return nil
		}
		resp := *execution.Response

		history := types.NewExecutionHistory(
			req.ID,
			string(resp.Body),
			resp.StatusCode,
			resp.Duration.Milliseconds(),
			resp.Size,
		)
		if err := db.SaveHistory(history); err != nil {
			fmt.Fprintf(os.Stderr, "Row %d: failed to save history: %v\n", rowNum, err)
		}

		fmt.Fprintf(os.Stdout, "Row %d [%s]: %d (%dms)\n", rowNum, name, resp.StatusCode, resp.Duration.Milliseconds())

		return nil
	})

	if err != nil {
		return fmt.Errorf("data iteration failed: %w", err)
	}
	if c.Bool("persist") {
		collectionStore, _ := db.(storage.CollectionStore)
		persisted, persistErr := runner.PersistDirtyVariables(envStorage, collectionStore, persistEnvName, req.Collection, dirtyForPersist, dirtyOriginsForPersist)
		if persistErr != nil {
			return persistErr
		}
		runner.PrintPersistSummaries(os.Stderr, persisted)
	}

	return nil
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
