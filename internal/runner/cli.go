package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/reporters"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

func CollectionRunCommand(db storage.DB, envStorage *env.EnvStorage) *cli.Command {
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Run a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Environment name to use",
			},
			&cli.BoolFlag{
				Name:    "bail",
				Aliases: []string{"b"},
				Usage:   "Stop on first failure",
			},
			&cli.BoolFlag{
				Name:  "assert-bail",
				Usage: "Stop on first assertion failure only",
			},
			&cli.BoolFlag{
				Name:  "persist",
				Usage: "Persist extracted/script variables back to the selected environment",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview collection execution without sending requests",
			},
			&cli.IntFlag{
				Name:    "iterations",
				Aliases: []string{"n"},
				Usage:   "Number of times to run the collection",
				Value:   1,
			},
			&cli.DurationFlag{
				Name:    "delay",
				Aliases: []string{"d"},
				Usage:   "Delay between requests (e.g., 100ms, 1s)",
				Value:   0,
			},
			&cli.StringSliceFlag{
				Name:    "var",
				Aliases: []string{"v"},
				Usage:   "Variable substitution (--var key=value)",
			},
			&cli.StringFlag{
				Name:    "data",
				Aliases: []string{"D"},
				Usage:   "Data file (CSV or JSON) for data-driven iteration",
			},
			&cli.StringSliceFlag{
				Name:    "reporter",
				Aliases: []string{"R"},
				Usage:   "Reporter(s) to use (junit, json, html, console)",
			},
			&cli.StringFlag{
				Name:    "reporter-output",
				Aliases: []string{"O"},
				Usage:   "Output directory for reporter files",
			},
			&cli.BoolFlag{
				Name:  "ci",
				Usage: "CI mode: treat skipped requests as failures (exit 1)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			args := c.Args()
			if args.Len() < 1 {
				return fmt.Errorf("collection name argument is required")
			}
			name := args.Get(0)
			if c.Bool("persist") && c.Bool("dry-run") {
				return cli.Exit("--persist and --dry-run cannot be used together", 2)
			}

			persistEnvName := ""
			if c.Bool("persist") {
				var err error
				persistEnvName, err = env.ResolveOptionalPersistEnvironmentName(envStorage, c.String("env"))
				if err != nil {
					return cli.Exit(err.Error(), 2)
				}
			}

			vars := make(map[string]string)
			for _, pair := range c.StringSlice("var") {
				if idx := strings.Index(pair, "="); idx != -1 {
					vars[pair[:idx]] = pair[idx+1:]
				}
			}

			runner := NewRunner(db, envStorage)
			envName := c.String("env")
			if envName == "" && persistEnvName != "" {
				envName = persistEnvName
			}
			config := RunConfig{
				CollectionName: name,
				Environment:    envName,
				Iterations:     c.Int("iterations"),
				Bail:           c.Bool("bail"),
				AssertBail:     c.Bool("assert-bail"),
				DryRun:         c.Bool("dry-run"),
				Delay:          c.Duration("delay"),
				Vars:           vars,
				DataFile:       c.String("data"),
			}

			ciMode := c.Bool("ci")

			results, runErr := runner.Run(ctx, config)
			if runErr != nil {
				fmt.Fprintf(os.Stderr, "collection run failed: %v\n", runErr)
				os.Exit(int(DetermineExitCode(nil, runErr, ciMode)))
				return nil
			}

			reporterNames := c.StringSlice("reporter")
			reporterOutput := c.String("reporter-output")

			if c.Bool("dry-run") {
				printDryRunDiagnostics(os.Stdout, results, envName)
				return nil
			}

			for _, name := range reporterNames {
				reporter := reporters.GetReporter(name)
				if reporter == nil {
					fmt.Fprintf(os.Stderr, "unknown reporter: %s (available: junit, json, html, console)\n", name)
					continue
				}

				reporterResults := convertToReporterResults(results)
				content, err := reporter.Report(reporterResults)
				if err != nil {
					fmt.Fprintf(os.Stderr, "reporter %s failed: %v\n", name, err)
					continue
				}

				if reporterOutput != "" {
					filename := getReporterFilename(name, reporterOutput)
					if err := os.WriteFile(filename, content, 0644); err != nil {
						fmt.Fprintf(os.Stderr, "failed to write %s report to %s: %v\n", name, filename, err)
					} else {
						fmt.Printf("Report written to %s\n", filename)
					}
				} else {
					fmt.Fprintf(os.Stdout, "\n--- %s reporter ---\n", name)
					os.Stdout.Write(content)
					os.Stdout.Write([]byte("\n"))
				}
			}

			printSummary(os.Stdout, results)
			if c.Bool("persist") {
				dirtyVars, dirtyOrigins := CollectDirtyVarsWithOrigins(results)
				collectionStore, _ := db.(storage.CollectionStore)
				persisted, err := PersistDirtyVariables(envStorage, collectionStore, persistEnvName, name, dirtyVars, dirtyOrigins)
				if err != nil {
					return err
				}
				PrintPersistSummaries(os.Stdout, persisted)
				if collectionRunAborted(results) {
					fmt.Fprintf(os.Stdout, "Run aborted after persisting %d variables.\n", len(dirtyVars))
				}
			}

			if code := DetermineExitCode(results, nil, ciMode); code != ExitSuccess {
				os.Exit(int(code))
			}
			return nil
		},
	}
}

func printDryRunDiagnostics(out io.Writer, results []RunResult, envName string) {
	if len(results) == 0 {
		return
	}
	requestCount := 0
	for _, result := range results {
		requestCount += len(result.RequestResults)
	}
	collectionName := results[0].CollectionName
	fmt.Fprintf(out, "Dry run: collection %q\n", collectionName)
	fmt.Fprintf(out, "Requests: %d\n", requestCount)
	if envName == "" {
		fmt.Fprintln(out, "Environment: <none>")
	} else {
		fmt.Fprintf(out, "Environment: %s\n", envName)
	}

	step := 1
	for _, result := range results {
		if len(results) > 1 {
			fmt.Fprintf(out, "\nIteration %d\n", result.Iteration)
		}
		for _, reqResult := range result.RequestResults {
			fmt.Fprintf(out, "\n%d. %s\n", step, reqResult.RequestName)
			fmt.Fprintf(out, "  %s %s\n", reqResult.PlannedMethod, reqResult.PlannedURL)
			if len(reqResult.PlannedVarSources) > 0 {
				keys := make([]string, 0, len(reqResult.PlannedVarSources))
				for key := range reqResult.PlannedVarSources {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					fmt.Fprintf(out, "  %s %s\n", key, reqResult.PlannedVarSources[key])
				}
			}
			for _, warning := range reqResult.DryRunWarnings {
				fmt.Fprintf(out, "  warning: %s\n", warning)
			}
			for _, extract := range reqResult.PlannedExtracts {
				fmt.Fprintf(out, "  %s ← %s\n", extract.Name, extract.Source)
			}
			step++
		}
	}
}

func getReporterFilename(reporterName, outputDir string) string {
	ext := ".txt"
	switch reporterName {
	case "junit":
		ext = ".xml"
	case "json":
		ext = ".json"
	case "html":
		ext = ".html"
	}
	filename := "report" + ext
	return filepath.Join(outputDir, filename)
}

func convertToReporterResults(results []RunResult) []reporters.RunResult {
	reporterResults := make([]reporters.RunResult, len(results))
	for i, r := range results {
		reqResults := make([]*reporters.RequestResult, len(r.RequestResults))
		for j, req := range r.RequestResults {
			assertionResults := make([]reporters.AssertionResult, len(req.AssertionResults))
			for k, a := range req.AssertionResults {
				assertionResults[k] = reporters.AssertionResult{
					Field:    a.Assertion.Field,
					Source:   a.Assertion.Field,
					Op:       a.Assertion.Op,
					Operator: a.Assertion.Op,
					Value:    a.Assertion.Value,
					Expected: a.Expected,
					Actual:   a.Actual,
					Passed:   a.Passed,
					Message:  a.Message,
				}
			}
			reqResults[j] = &reporters.RequestResult{
				RequestName:      req.RequestName,
				StatusCode:       req.StatusCode,
				Passed:           req.Passed,
				Skipped:          req.Skipped,
				Error:            req.Error,
				Duration:         req.Duration,
				AssertionResults: assertionResults,
			}
		}
		reporterResults[i] = reporters.RunResult{
			CollectionName: r.CollectionName,
			Total:          r.Total,
			Passed:         r.Passed,
			Failed:         r.Failed,
			Skipped:        r.Skipped,
			Duration:       r.Duration,
			RequestResults: reqResults,
			Iteration:      r.Iteration,
		}
	}
	return reporterResults
}

func PrintPersistSummary(out io.Writer, envName string, persisted map[string]string, targetEnv *env.Environment) {
	count := len(persisted)
	label := "variables"
	if count == 1 {
		label = "variable"
	}
	fmt.Fprintf(out, "\nPersisted %d %s to environment %q\n", count, label, envName)

	keys := make([]string, 0, len(persisted))
	for key := range persisted {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(out, "  %s = %s\n", key, env.MaskedValue(targetEnv, key, persisted[key]))
	}
}

func collectionRunAborted(results []RunResult) bool {
	for _, result := range results {
		for _, requestResult := range result.RequestResults {
			if requestResult.SkipReason == SkipReasonBail {
				return true
			}
		}
	}
	return false
}

func printSummary(out *os.File, results []RunResult) {
	for _, result := range results {
		fmt.Fprintf(out, "\n=== Collection: %s (Iteration %d) ===\n", result.CollectionName, result.Iteration)
		fmt.Fprintf(out, "Duration: %v\n", result.Duration)
		fmt.Fprintf(out, "Total: %d | Passed: %d | Failed: %d | Skipped: %d\n",
			result.Total, result.Passed, result.Failed, result.Skipped)

		for _, reqResult := range result.RequestResults {
			if reqResult.Skipped {
				fmt.Fprintf(out, "  [SKIP] %s\n", reqResult.RequestName)
			} else if reqResult.Error != "" {
				fmt.Fprintf(out, "  [FAIL] %s - %s\n", reqResult.RequestName, reqResult.Error)
			} else if reqResult.Passed {
				fmt.Fprintf(out, "  [PASS] %s (%d %s)\n",
					reqResult.RequestName, reqResult.Duration.Milliseconds(), "ms")
			} else {
				fmt.Fprintf(out, "  [FAIL] %s (status: %d)\n",
					reqResult.RequestName, reqResult.StatusCode)
			}
		}
	}

	if len(results) > 0 {
		totalPassed := 0
		totalFailed := 0
		totalSkipped := 0
		totalRequests := 0
		for _, r := range results {
			totalPassed += r.Passed
			totalFailed += r.Failed
			totalSkipped += r.Skipped
			totalRequests += r.Total
		}
		fmt.Fprintf(out, "\n=== SUMMARY ===\n")
		fmt.Fprintf(out, "Total: %d | Passed: %d | Failed: %d | Skipped: %d\n",
			totalRequests, totalPassed, totalFailed, totalSkipped)
	}
}
