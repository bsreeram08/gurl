package runner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sreeram/gurl/internal/assertions"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

type RunConfig struct {
	CollectionName string
	Environment    string
	Iterations     int
	Bail           bool
	Delay          time.Duration
	Vars           map[string]string
	DataFile       string
}

type RunResult struct {
	CollectionName string
	Total          int
	Passed         int
	Failed         int
	Skipped        int
	Duration       time.Duration
	RequestResults []*RequestResult
	Iteration      int
}

type RequestResult struct {
	RequestName      string
	StatusCode       int
	Passed           bool
	Skipped          bool
	Error            string
	Duration         time.Duration
	AssertionResults []assertions.Result
}

type EnvProvider interface {
	GetEnvByName(name string) (*env.Environment, error)
}

type Runner struct {
	db         storage.DB
	envStorage EnvProvider
	client     *client.Client
	eval       *assertions.Evaluator
}

func NewRunner(db storage.DB, envStorage EnvProvider) *Runner {
	return &Runner{
		db:         db,
		envStorage: envStorage,
		client:     client.NewClient(),
		eval:       assertions.NewEvaluator(),
	}
}

func (r *Runner) Run(ctx context.Context, config RunConfig) ([]RunResult, error) {
	if config.Iterations <= 0 {
		config.Iterations = 1
	}

	requests, err := r.db.ListRequests(&storage.ListOptions{
		Collection: config.CollectionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list collection: %w", err)
	}

	if len(requests) == 0 {
		return nil, &EmptyCollectionError{Collection: config.CollectionName}
	}

	// Sort requests by SortOrder before execution
	requests = sortBySequence(requests)

	baseVars := make(map[string]string)
	if config.Environment != "" && r.envStorage != nil {
		env, err := r.envStorage.GetEnvByName(config.Environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: environment %q not found: %v\n", config.Environment, err)
		} else {
			for k, v := range env.Variables {
				baseVars[k] = v
			}
		}
	}

	for k, v := range config.Vars {
		baseVars[k] = v
	}

	results := make([]RunResult, 0)

	if config.DataFile != "" {
		dataResults, err := r.runWithData(ctx, requests, baseVars, config)
		if err != nil {
			return nil, err
		}
		results = append(results, dataResults...)
	} else {
		for iter := 0; iter < config.Iterations; iter++ {
			result := r.runIteration(ctx, requests, baseVars, config, iter+1)
			results = append(results, result)

			if config.Delay > 0 && iter < config.Iterations-1 {
				select {
				case <-ctx.Done():
					return results, ctx.Err()
				case <-time.After(config.Delay):
				}
			}
		}
	}

	return results, nil
}

func (r *Runner) runWithData(ctx context.Context, requests []*types.SavedRequest, baseVars map[string]string, config RunConfig) ([]RunResult, error) {
	loader, err := NewDataLoader(config.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load data file: %w", err)
	}

	iteration := 1
	var results []RunResult

	err = loader.Iterate(func(rowVars map[string]string) error {
		mergedVars := make(map[string]string)
		for k, v := range baseVars {
			mergedVars[k] = v
		}
		for k, v := range rowVars {
			mergedVars[k] = v
		}

		result := r.runIteration(ctx, requests, mergedVars, config, iteration)
		results = append(results, result)
		iteration++

		if config.Delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(config.Delay):
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("data iteration failed: %w", err)
	}

	return results, nil
}

func (r *Runner) runIteration(ctx context.Context, requests []*types.SavedRequest, vars map[string]string, config RunConfig, iteration int) RunResult {
	start := time.Now()
	result := RunResult{
		CollectionName: config.CollectionName,
		Iteration:      iteration,
		RequestResults: make([]*RequestResult, 0, len(requests)),
		Total:          len(requests), // Total = full collection size (includes skipped on bail)
	}

	for i, req := range requests {
		reqResult := r.runRequest(ctx, req, vars, config)
		result.RequestResults = append(result.RequestResults, reqResult)

		if reqResult.Passed {
			result.Passed++
		} else if reqResult.Skipped {
			result.Skipped++
		} else {
			result.Failed++
		}

		if config.Delay > 0 && i < len(requests)-1 {
			select {
			case <-ctx.Done():
				return result
			case <-time.After(config.Delay):
			}
		}

		if config.Bail && !reqResult.Passed && !reqResult.Skipped {
			remaining := len(requests) - i - 1
			result.Skipped += remaining
			// Add remaining requests as skipped (bail triggered).
			for j := i + 1; j < len(requests); j++ {
				result.RequestResults = append(result.RequestResults, &RequestResult{
					RequestName: requests[j].Name,
					Skipped:     true,
				})
			}
			break
		}
	}

	result.Duration = time.Since(start)
	return result
}

func (r *Runner) runRequest(ctx context.Context, req *types.SavedRequest, vars map[string]string, config RunConfig) *RequestResult {
	start := time.Now()
	result := &RequestResult{
		RequestName: req.Name,
	}

	substitutedURL, err := template.Substitute(req.URL, vars)
	if err != nil {
		result.Error = fmt.Sprintf("variable substitution failed for URL: %v", err)
		return result
	}

	substitutedBody, err := template.Substitute(req.Body, vars)
	if err != nil {
		result.Error = fmt.Sprintf("variable substitution failed for body: %v", err)
		return result
	}

	clientReq := client.Request{
		Method:  req.Method,
		URL:     substitutedURL,
		Headers: convertHeaders(req.Headers),
		Body:    substitutedBody,
	}

	if req.Timeout != "" {
		if d, err := time.ParseDuration(req.Timeout); err == nil && d > 0 {
			clientReq.Timeout = d
		}
	}

	resp, err := r.client.ExecuteWithContext(ctx, clientReq)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(start)

	if result.StatusCode < 200 || result.StatusCode >= 300 {
		result.Passed = false
		return result
	}

	if len(req.Assertions) > 0 {
		assertResults := r.eval.Evaluate(&resp, convertAssertions(req.Assertions))
		result.AssertionResults = assertResults

		allPassed := true
		for _, ar := range assertResults {
			if !ar.Passed {
				allPassed = false
				break
			}
		}
		result.Passed = allPassed && result.Error == ""
	} else {
		result.Passed = result.Error == ""
	}

	return result
}

func convertHeaders(headers []types.Header) []client.Header {
	if headers == nil {
		return nil
	}
	result := make([]client.Header, len(headers))
	for i, h := range headers {
		result[i] = client.Header{Key: h.Key, Value: h.Value}
	}
	return result
}

func convertAssertions(typeAssertions []types.Assertion) []assertions.Assertion {
	if typeAssertions == nil {
		return nil
	}
	result := make([]assertions.Assertion, len(typeAssertions))
	for i, a := range typeAssertions {
		result[i] = assertions.Assertion{
			Field: a.Field,
			Op:    a.Op,
			Value: a.Value,
		}
	}
	return result
}
