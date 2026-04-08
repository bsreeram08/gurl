package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/assertions"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/internal/env"
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
			if args.Len() < 1 {
				return fmt.Errorf("request name argument is required")
			}
			name := args.Get(0)

			enableChain := c.Bool("chain")
			dataFile := c.String("data")

			baseVars := make(map[string]string)

			if envName := c.String("env"); envName != "" {
				if env, err := envStorage.GetEnvByName(envName); err == nil {
					for k, v := range env.Variables {
						baseVars[k] = v
					}
				}
			} else if activeEnvName, err := envStorage.GetActiveEnv(); err == nil && activeEnvName != "" {
				if env, err := envStorage.GetEnvByName(activeEnvName); err == nil {
					for k, v := range env.Variables {
						baseVars[k] = v
					}
				}
			}

			for _, pair := range c.StringSlice("var") {
				if idx := strings.Index(pair, "="); idx != -1 {
					baseVars[pair[:idx]] = pair[idx+1:]
				}
			}

			if dataFile != "" {
				return executeDataDriven(ctx, db, name, baseVars, c)
			}

			if enableChain {
				return executeChain(ctx, db, envStorage, name, baseVars, c)
			}

			return executeSingleRequest(db, name, baseVars, c)
		},
	}
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

		substitutedURL, err := template.Substitute(req.URL, vars)
		if err != nil {
			return fmt.Errorf("variable substitution failed: %w", err)
		}

		substitutedBody, _ := template.Substitute(req.Body, vars)

		var timeout time.Duration
		if timeoutStr := c.String("timeout"); timeoutStr != "" {
			timeout, _ = time.ParseDuration(timeoutStr)
		} else if req.Timeout != "" {
			timeout, _ = time.ParseDuration(req.Timeout)
		}

		clientReq := client.Request{
			Method:  req.Method,
			URL:     substitutedURL,
			Headers: convertHeaders(req.Headers),
			Body:    substitutedBody,
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
			resp.URL = substitutedURL
			force := c.Bool("force")
			if err := client.SaveToFile(&resp, outputPath, force); err != nil {
				return fmt.Errorf("failed to save output: %w", err)
			}
		}

		format := c.String("format")
		if err := printResponse(os.Stdout, req.Method, substitutedURL, resp, format); err != nil {
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

func executeSingleRequest(db storage.DB, name string, vars map[string]string, c *cli.Command) error {
	req, err := db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
	}

	substitutedURL, err := template.Substitute(req.URL, vars)
	if err != nil {
		return fmt.Errorf("variable substitution failed: %w", err)
	}

	substitutedBody, _ := template.Substitute(req.Body, vars)

	var timeout time.Duration
	if timeoutStr := c.String("timeout"); timeoutStr != "" {
		timeout, _ = time.ParseDuration(timeoutStr)
	} else if req.Timeout != "" {
		timeout, _ = time.ParseDuration(req.Timeout)
	}

	clientReq := client.Request{
		Method:  req.Method,
		URL:     substitutedURL,
		Headers: convertHeaders(req.Headers),
		Body:    substitutedBody,
	}
	if timeout > 0 {
		clientReq.Timeout = timeout
	}

	resp, err := client.Execute(clientReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	// Evaluate assertions if provided
	if assertFlags := c.StringSlice("assert"); len(assertFlags) > 0 {
		parser := assertions.NewCLIParser()
		asserts, err := parser.ParseSlice(assertFlags)
		if err != nil {
			return fmt.Errorf("failed to parse assertions: %w", err)
		}
		evaluator := assertions.NewEvaluator()
		results := evaluator.Evaluate(&resp, asserts)
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
		resp.URL = substitutedURL
		force := c.Bool("force")
		if err := client.SaveToFile(&resp, outputPath, force); err != nil {
			return fmt.Errorf("failed to save output: %w", err)
		}
		fmt.Printf("Response saved to %s\n", outputPath)
		return nil
	}

	format := c.String("format")
	return printResponse(os.Stdout, clientReq.Method, substitutedURL, resp, format)
}

func executeDataDriven(ctx context.Context, db storage.DB, name string, baseVars map[string]string, c *cli.Command) error {
	req, err := db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
	}

	dataFile := c.String("data")
	loader, err := runner.NewDataLoader(dataFile)
	if err != nil {
		return fmt.Errorf("failed to load data file: %w", err)
	}

	rowNum := 0
	err = loader.Iterate(func(rowVars map[string]string) error {
		rowNum++

		mergedVars := make(map[string]string)
		for k, v := range baseVars {
			mergedVars[k] = v
		}
		for k, v := range rowVars {
			mergedVars[k] = v
		}

		substitutedURL, err := template.Substitute(req.URL, mergedVars)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Row %d: variable substitution failed for URL: %v\n", rowNum, err)
			return nil
		}

		substitutedBody, _ := template.Substitute(req.Body, mergedVars)

		var timeout time.Duration
		if timeoutStr := c.String("timeout"); timeoutStr != "" {
			timeout, _ = time.ParseDuration(timeoutStr)
		} else if req.Timeout != "" {
			timeout, _ = time.ParseDuration(req.Timeout)
		}

		clientReq := client.Request{
			Method:  req.Method,
			URL:     substitutedURL,
			Headers: convertHeaders(req.Headers),
			Body:    substitutedBody,
		}
		if timeout > 0 {
			clientReq.Timeout = timeout
		}

		resp, err := client.Execute(clientReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Row %d: request failed: %v\n", rowNum, err)
			return nil
		}

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

func printResponse(out *os.File, method string, url string, resp client.Response, format string) error {
	if outputPlugin, found := outputPluginRegistry.GetOutputByFormat(format); found {
		ctx := &plugins.ResponseContext{
			Request: &client.Request{
				Method: method,
				URL:    url,
			},
			Response: &resp,
		}
		_, err := out.WriteString(outputPlugin.Render(ctx))
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
		var data interface{}
		if json.Unmarshal(resp.Body, &data) == nil {
			enc := json.NewEncoder(out)
			enc.SetIndent("  ", "")
			return enc.Encode(data)
		}
		_, err := out.Write(resp.Body)
		return err
	default:
		_, err := out.Write(resp.Body)
		return err
	}
}
