package runner

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/assertions"
	"github.com/sreeram/gurl/internal/auth"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/core/curl"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/extract"
	"github.com/sreeram/gurl/internal/scripting"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

type RunConfig struct {
	CollectionName string
	Environment    string
	Iterations     int
	Bail           bool
	AssertBail     bool
	DryRun         bool
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
	RequestName         string
	StatusCode          int
	Passed              bool
	Skipped             bool
	Error               string
	Duration            time.Duration
	AssertionResults    []assertions.Result
	ExtractedVars       map[string]string
	DirtyVars           map[string]string
	DryRun              bool
	PlannedMethod       string
	PlannedURL          string
	PlannedExtracts     []types.Extract
	PlannedVarSources   map[string]string
	DryRunWarnings      []string
	SkipReason          string
	FailurePhase        string
	NextRequestOverride string
}

type SingleRequestExecution struct {
	Result   *RequestResult
	Request  *client.Request
	Response *client.Response
}

const (
	SkipReasonRunIf  = "run_if"
	SkipReasonScript = "script"
	SkipReasonBail   = "bail"

	FailurePhaseRunIf              = "run_if"
	FailurePhaseRequestBuild       = "request_build"
	FailurePhaseHTTP               = "http"
	FailurePhasePreRequestScript   = "pre_request_script"
	FailurePhasePostResponseScript = "post_response_script"
	FailurePhaseAssertion          = "assertion"
	FailurePhaseNextRequest        = "next_request"
)

type EnvProvider interface {
	GetEnvByName(name string) (*env.Environment, error)
}

type Runner struct {
	db         storage.DB
	envStorage EnvProvider
	client     *client.Client
	eval       *assertions.Evaluator
	extractor  *extract.Extractor
	auth       *auth.Registry
}

func NewRunner(db storage.DB, envStorage EnvProvider) *Runner {
	return &Runner{
		db:         db,
		envStorage: envStorage,
		client:     client.NewClient(),
		eval:       assertions.NewEvaluator(),
		extractor:  extract.NewExtractor(),
		auth:       auth.BuiltinRegistry(),
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

	envVars := make(map[string]string)
	if config.Environment != "" && r.envStorage != nil {
		env, err := r.envStorage.GetEnvByName(config.Environment)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: environment %q not found: %v\n", config.Environment, err)
		} else {
			for k, v := range env.Variables {
				envVars[k] = v
			}
		}
	}

	baseVars := copyStringMap(envVars)
	for k, v := range config.Vars {
		baseVars[k] = v
	}

	results := make([]RunResult, 0)

	if config.DataFile != "" {
		dataResults, err := r.runWithData(ctx, requests, envVars, config)
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

func (r *Runner) runWithData(ctx context.Context, requests []*types.SavedRequest, envVars map[string]string, config RunConfig) ([]RunResult, error) {
	loader, err := NewDataLoader(config.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load data file: %w", err)
	}

	iteration := 1
	var results []RunResult

	err = loader.Iterate(func(rowVars map[string]string) error {
		mergedVars := make(map[string]string)
		for k, v := range envVars {
			mergedVars[k] = v
		}
		for k, v := range rowVars {
			mergedVars[k] = v
		}
		for k, v := range config.Vars {
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
	runningVars := copyStringMap(vars)
	extractedVars := make(map[string]string)
	plannedVarSources := make(map[string]string)
	requestIndex := requestIndexByName(requests)
	visited := make(map[string]bool, len(requests))
	result := RunResult{
		CollectionName: config.CollectionName,
		Iteration:      iteration,
		RequestResults: make([]*RequestResult, 0, len(requests)),
		Total:          len(requests), // Total = full collection size (includes skipped on bail)
	}

	for i := 0; i < len(requests); {
		req := requests[i]
		if visited[req.Name] {
			reqResult := &RequestResult{
				RequestName:  req.Name,
				Error:        fmt.Sprintf("next request loop detected: %q would be revisited", req.Name),
				FailurePhase: FailurePhaseNextRequest,
			}
			result.RequestResults = append(result.RequestResults, reqResult)
			result.Failed++
			break
		}
		visited[req.Name] = true

		reqResult := r.runRequest(ctx, req, runningVars, extractedVars, config.DryRun, plannedVarSources)
		nextIndex := i + 1
		stop := false
		if reqResult.NextRequestOverride != "" {
			targetName := reqResult.NextRequestOverride
			targetIndex, ok := requestIndex[targetName]
			if !ok {
				failNextRequestOverride(reqResult, fmt.Sprintf("next request %q not found in collection %q", targetName, config.CollectionName))
				stop = true
			} else if visited[targetName] {
				failNextRequestOverride(reqResult, fmt.Sprintf("next request loop detected: %q would revisit an already executed request", targetName))
				stop = true
			} else {
				nextIndex = targetIndex
			}
		}

		result.RequestResults = append(result.RequestResults, reqResult)
		if config.DryRun {
			step := len(result.RequestResults)
			for _, plannedExtract := range reqResult.PlannedExtracts {
				plannedVarSources[plannedExtract.Name] = fmt.Sprintf("from step %d extraction", step)
			}
		}

		if reqResult.Passed {
			result.Passed++
		} else if reqResult.Skipped {
			result.Skipped++
		} else {
			result.Failed++
		}

		if shouldStopForBail(config, reqResult) && (config.Bail || reqResult.NextRequestOverride == "" || stop) {
			appendBailSkippedResults(&result, requests, i+1)
			break
		}

		if stop {
			break
		}

		if config.Delay > 0 && nextIndex < len(requests) {
			select {
			case <-ctx.Done():
				return result
			case <-time.After(config.Delay):
			}
		}

		i = nextIndex
	}

	result.Duration = time.Since(start)
	return result
}

func shouldStopForBail(config RunConfig, result *RequestResult) bool {
	if result.Passed || result.Skipped {
		return false
	}
	if config.Bail {
		return true
	}
	return config.AssertBail && result.FailurePhase == FailurePhaseAssertion
}

func appendBailSkippedResults(result *RunResult, requests []*types.SavedRequest, start int) {
	remaining := len(requests) - start
	result.Skipped += remaining
	for j := start; j < len(requests); j++ {
		result.RequestResults = append(result.RequestResults, &RequestResult{
			RequestName: requests[j].Name,
			Skipped:     true,
			SkipReason:  SkipReasonBail,
		})
	}
}

func (r *Runner) runRequest(ctx context.Context, req *types.SavedRequest, vars map[string]string, extractedVars map[string]string, dryRun bool, plannedVarSources map[string]string) *RequestResult {
	return r.runRequestLifecycle(ctx, req, vars, extractedVars, dryRun, plannedVarSources).Result
}

func (r *Runner) RunSavedRequest(ctx context.Context, req *types.SavedRequest, vars map[string]string) *SingleRequestExecution {
	runningVars := copyStringMap(vars)
	extractedVars := make(map[string]string)
	return r.runRequestLifecycle(ctx, req, runningVars, extractedVars, false, nil)
}

func (r *Runner) runRequestLifecycle(ctx context.Context, req *types.SavedRequest, vars map[string]string, extractedVars map[string]string, dryRun bool, plannedVarSources map[string]string) *SingleRequestExecution {
	start := time.Now()
	result := &RequestResult{
		RequestName: req.Name,
	}
	execution := &SingleRequestExecution{Result: result}

	dirtyVars := make(map[string]string)
	effectiveReq := req

	if req.RunIf != "" {
		ok, err := evaluateRunIf(req.RunIf, vars)
		if err != nil {
			result.Error = err.Error()
			result.FailurePhase = FailurePhaseRunIf
			result.Duration = time.Since(start)
			return execution
		}
		if !ok {
			result.Skipped = true
			result.SkipReason = SkipReasonRunIf
			result.Duration = time.Since(start)
			return execution
		}
	}

	if req.PreScript != "" {
		engine := scripting.NewEngine(nil)
		engine.SetVariables(vars)
		rawReq := &client.Request{
			Method:  req.Method,
			URL:     req.URL,
			Headers: convertHeaders(req.Headers),
			Body:    req.Body,
		}
		modifiedReq, err := scripting.RunPreRequest(engine, req.PreScript, rawReq)
		if err != nil {
			result.Error = fmt.Sprintf("pre-request script failed: %v", err)
			result.FailurePhase = FailurePhasePreRequestScript
			result.Duration = time.Since(start)
			return execution
		}
		mergeVars(vars, extractedVars, dirtyVars, engine.DirtyVariables())

		if engine.SkipRequested() {
			result.Skipped = true
			result.SkipReason = engine.SkipReason()
			result.DirtyVars = nonEmptyMap(dirtyVars)
			result.Duration = time.Since(start)
			return execution
		}

		copy := *req
		copy.Method = modifiedReq.Method
		copy.URL = modifiedReq.URL
		copy.Body = modifiedReq.Body
		copy.Headers = convertClientHeadersToTypes(modifiedReq.Headers)
		effectiveReq = &copy
	}

	if dryRun {
		planned := planDryRunRequest(effectiveReq, vars, plannedVarSources)
		result.DryRun = true
		result.Passed = true
		result.PlannedMethod = planned.method
		result.PlannedURL = planned.url
		result.PlannedExtracts = planned.extracts
		result.PlannedVarSources = planned.varSources
		result.DryRunWarnings = planned.warnings
		result.DirtyVars = nonEmptyMap(dirtyVars)
		result.Duration = time.Since(start)
		return execution
	}

	authRegistry := r.auth
	if authRegistry == nil {
		authRegistry = auth.BuiltinRegistry()
	}
	clientReq, err := curl.BuildClientRequestWithAuth(effectiveReq, vars, authRegistry)
	if err != nil {
		result.Error = fmt.Sprintf("variable substitution failed: %v", err)
		result.FailurePhase = FailurePhaseRequestBuild
		result.DirtyVars = nonEmptyMap(dirtyVars)
		return execution
	}
	execution.Request = &clientReq

	if effectiveReq.Timeout != "" {
		if d, err := time.ParseDuration(effectiveReq.Timeout); err == nil && d > 0 {
			clientReq.Timeout = d
		}
	}

	resp, err := r.client.ExecuteWithContext(ctx, clientReq)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		result.FailurePhase = FailurePhaseHTTP
		result.DirtyVars = nonEmptyMap(dirtyVars)
		return execution
	}
	execution.Response = &resp

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(start)

	if len(effectiveReq.Extracts) > 0 {
		extracted, err := r.extractor.Extract(resp.Body, resp.Headers, effectiveReq.Extracts)
		if err != nil {
			result.Error = fmt.Sprintf("extraction failed: %v", err)
			result.DirtyVars = nonEmptyMap(dirtyVars)
			return execution
		}
		result.ExtractedVars = extracted
		mergeVars(vars, extractedVars, dirtyVars, extracted)
	}

	if effectiveReq.PostScript != "" {
		engine := scripting.NewEngine(nil)
		engine.SetVariables(vars)
		postResult, err := scripting.RunPostResponse(engine, effectiveReq.PostScript, &resp)
		if err != nil {
			result.Error = fmt.Sprintf("post-response script failed: %v", err)
			result.FailurePhase = FailurePhasePostResponseScript
			result.DirtyVars = nonEmptyMap(dirtyVars)
			return execution
		}
		mergeVars(vars, extractedVars, dirtyVars, postResult.Variables)
		result.NextRequestOverride = engine.NextRequest()
	}
	result.DirtyVars = nonEmptyMap(dirtyVars)

	if result.StatusCode < 200 || result.StatusCode >= 300 {
		result.Passed = false
		return execution
	}

	if len(effectiveReq.Assertions) > 0 {
		assertResults := r.eval.Evaluate(&resp, convertAssertions(effectiveReq.Assertions), extractedVars)
		result.AssertionResults = assertResults

		allPassed := true
		for _, ar := range assertResults {
			if !ar.Passed {
				allPassed = false
				break
			}
		}
		if !allPassed {
			result.FailurePhase = FailurePhaseAssertion
		}
		result.Passed = allPassed && result.Error == ""
	} else {
		result.Passed = result.Error == ""
	}

	return execution
}

type dryRunPlan struct {
	method     string
	url        string
	extracts   []types.Extract
	varSources map[string]string
	warnings   []string
}

func planDryRunRequest(req *types.SavedRequest, vars map[string]string, plannedVarSources map[string]string) dryRunPlan {
	plan := dryRunPlan{
		method:     req.Method,
		url:        resolveDryRunTemplate(req.URL, vars),
		extracts:   append([]types.Extract(nil), req.Extracts...),
		varSources: make(map[string]string),
	}
	if plan.method == "" {
		plan.method = "GET"
	}

	usedVars := extractTemplateVars(req.URL)
	for _, header := range req.Headers {
		usedVars = append(usedVars, extractTemplateVars(header.Value)...)
	}
	usedVars = append(usedVars, extractTemplateVars(req.Body)...)

	warningsByVar := make(map[string]bool)
	for _, name := range usedVars {
		if _, ok := vars[name]; ok {
			continue
		}
		if source := plannedVarSources[name]; source != "" {
			plan.varSources[name] = source
			continue
		}
		warningsByVar[name] = true
	}

	if len(warningsByVar) > 0 {
		names := make([]string, 0, len(warningsByVar))
		for name := range warningsByVar {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			plan.warnings = append(plan.warnings, fmt.Sprintf("unresolved {{%s}}", name))
		}
	}

	return plan
}

func resolveDryRunTemplate(value string, vars map[string]string) string {
	resolved := value
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		resolved = strings.ReplaceAll(resolved, "{{"+key+"}}", vars[key])
	}
	return resolved
}

func requestIndexByName(requests []*types.SavedRequest) map[string]int {
	index := make(map[string]int, len(requests))
	for i, req := range requests {
		index[req.Name] = i
	}
	return index
}

func failNextRequestOverride(result *RequestResult, message string) {
	result.Passed = false
	result.Skipped = false
	result.Error = message
	result.FailurePhase = FailurePhaseNextRequest
}

func mergeVars(vars map[string]string, extractedVars map[string]string, dirtyVars map[string]string, updates map[string]string) {
	for key, value := range updates {
		vars[key] = value
		extractedVars[key] = value
		dirtyVars[key] = value
	}
}

func nonEmptyMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	return copyStringMap(source)
}

func CollectDirtyVarsForPersist(results []RunResult) map[string]string {
	dirty := make(map[string]string)
	for _, runResult := range results {
		for _, requestResult := range runResult.RequestResults {
			for key, value := range requestResult.DirtyVars {
				dirty[key] = value
			}
		}
	}
	return dirty
}

func copyStringMap(source map[string]string) map[string]string {
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
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

func convertClientHeadersToTypes(headers []client.Header) []types.Header {
	if headers == nil {
		return nil
	}
	result := make([]types.Header, len(headers))
	for i, h := range headers {
		result[i] = types.Header{Key: h.Key, Value: h.Value}
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
