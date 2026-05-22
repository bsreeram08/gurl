package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/assertions"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/core/curl"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/formatter"
	"github.com/sreeram/gurl/internal/plugins"
	"github.com/sreeram/gurl/internal/plugins/builtins"
	"github.com/sreeram/gurl/internal/runner"
	"github.com/sreeram/gurl/internal/scripting"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

var outputPluginRegistry *plugins.Registry

func init() {
	outputPluginRegistry = plugins.NewRegistry()
	builtins.RegisterBuiltins(outputPluginRegistry)
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
			&cli.BoolFlag{
				Name:    "cache",
				Aliases: []string{"c"},
				Usage:   "Use cached response if fresh",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (use - for stdout)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force overwrite existing file",
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
			&cli.StringSliceFlag{
				Name:    "assert",
				Aliases: []string{"a"},
				Usage:   "Assertion to evaluate (e.g., status=200, body contains success)",
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

			enableChain := c.Bool("chain")
			dataFile := c.String("data")

			envVars := make(map[string]string)

			if envName := c.String("env"); envName != "" {
				if env, err := envStorage.GetEnvByName(envName); err == nil {
					for k, v := range env.Variables {
						envVars[k] = v
					}
				}
			} else if activeEnvName, err := envStorage.GetActiveEnv(); err == nil && activeEnvName != "" {
				if env, err := envStorage.GetEnvByName(activeEnvName); err == nil {
					for k, v := range env.Variables {
						envVars[k] = v
					}
				}
			}

			cliVars := make(map[string]string)
			for _, pair := range c.StringSlice("var") {
				if idx := strings.Index(pair, "="); idx != -1 {
					cliVars[pair[:idx]] = pair[idx+1:]
				}
			}
			baseVars := mergeRunVars(envVars, cliVars)

			if dataFile != "" {
				return executeDataDriven(ctx, db, envStorage, name, envVars, cliVars, c)
			}

			if enableChain {
				return executeChain(ctx, db, envStorage, name, baseVars, c)
			}

			return executeSingleRequest(ctx, db, envStorage, name, baseVars, c)
		},
	}
}

func mergeRunVars(maps ...map[string]string) map[string]string {
	merged := make(map[string]string)
	for _, vars := range maps {
		for key, value := range vars {
			merged[key] = value
		}
	}
	return merged
}

func executeChain(ctx context.Context, db storage.DB, envStorage *env.EnvStorage, name string, vars map[string]string, c *cli.Command) error {
	engine := scripting.NewEngine(envStorage)
	chainExec := scripting.NewChainExecutor(engine)

	currentName := name
	visited := make(map[string]int)

	for i := 0; i < chainExec.MaxIterations() || i == 0; i++ {
		req, err := db.GetRequestByName(currentName)
		if err != nil {
			return fmt.Errorf("request not found: %s", currentName)
		}

		visited[currentName]++
		if visited[currentName] >= 3 {
			return fmt.Errorf("circular chain detected: request '%s' visited 3 times", currentName)
		}

		clientReq, err := curl.BuildClientRequest(req, vars)
		if err != nil {
			return fmt.Errorf("variable substitution failed: %w", err)
		}

		var timeout time.Duration
		if timeoutStr := c.String("timeout"); timeoutStr != "" {
			var err error
			timeout, err = time.ParseDuration(timeoutStr)
			if err != nil {
				return fmt.Errorf("invalid timeout format '%s': %w", timeoutStr, err)
			}
		} else if req.Timeout != "" {
			var err error
			timeout, err = time.ParseDuration(req.Timeout)
			if err != nil {
				return fmt.Errorf("invalid timeout format '%s': %w", req.Timeout, err)
			}
		}

		if timeout > 0 {
			clientReq.Timeout = timeout
		}

		resp, err := client.Execute(clientReq)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
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
		}

		format := c.String("format")
		if err := printResponse(os.Stdout, clientReq.Method, clientReq.URL, resp, format); err != nil {
			return err
		}

		chainExec.MarkIteration(currentName)
		if chainExec.IsCircular() {
			return fmt.Errorf("circular chain detected for request '%s'", currentName)
		}

		nextReq := chainExec.GetNextRequest()
		if nextReq == "" {
			return nil
		}

		for k, v := range chainExec.Variables() {
			vars[k] = v
		}

		currentName = nextReq
	}

	return fmt.Errorf("max iterations (%d) reached", chainExec.MaxIterations())
}

func executeSingleRequest(ctx context.Context, db storage.DB, envStorage *env.EnvStorage, name string, vars map[string]string, c *cli.Command) error {
	req, err := db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
	}

	if timeoutStr := c.String("timeout"); timeoutStr != "" {
		copy := *req
		copy.Timeout = timeoutStr
		req = &copy
	}

	execution := runner.NewRunner(db, envStorage).RunSavedRequest(ctx, req, vars)
	if execution.Result.Error != "" {
		return fmt.Errorf("%s", execution.Result.Error)
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
			// Don't fail the request, just report
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
		fmt.Printf("Response saved to %s\n", outputPath)
		return nil
	}

	format := c.String("format")
	return printResponse(os.Stdout, clientReq.Method, clientReq.URL, resp, format)
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

func executeDataDriven(ctx context.Context, db storage.DB, envStorage *env.EnvStorage, name string, envVars map[string]string, cliVars map[string]string, c *cli.Command) error {
	req, err := db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
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
	singleRunner := runner.NewRunner(db, envStorage)
	err = loader.Iterate(func(rowVars map[string]string) error {
		rowNum++

		mergedVars := mergeRunVars(envVars, rowVars, cliVars)
		effectiveReq := req
		if timeoutStr := c.String("timeout"); timeoutStr != "" {
			copy := *req
			copy.Timeout = timeoutStr
			effectiveReq = &copy
		}

		execution := singleRunner.RunSavedRequest(ctx, effectiveReq, mergedVars)
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
