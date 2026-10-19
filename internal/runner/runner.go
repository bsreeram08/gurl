package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/sreeram/gurl/internal/assertions"
	"github.com/sreeram/gurl/internal/auth"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/extract"
	"github.com/sreeram/gurl/internal/resolve"
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
	// Verbose prints each request/response exchange to VerboseOut as the
	// collection runs. Display only: nothing extra is stored or reported.
	Verbose bool
	// VerboseOut receives the verbose output. Defaults to os.Stdout.
	VerboseOut io.Writer
}

var collectionRunWatchOptions = storage.CollectionWatchOptions{}

type runCollectionState struct {
	requests             []*types.SavedRequest
	scopes               resolve.Scopes
	defaultPersistOrigin resolve.Origin
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
	RequestName       string
	StatusCode        int
	Passed            bool
	Skipped           bool
	Error             string
	Duration          time.Duration
	AssertionResults  []assertions.Result
	ExtractedVars     map[string]string
	DirtyVars         map[string]string
	DirtyVarOrigins   map[string]resolve.Origin
	DryRun            bool
	PlannedMethod     string
	PlannedURL        string
	PlannedExtracts   []types.Extract
	PlannedVarSources map[string]string
	DryRunWarnings    []string
	// DryRunProblems are the resolution problems of a dry run other than
	// missing variables (those are DryRunWarnings).
	DryRunProblems      resolve.Problems
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

	runEnv := r.loadRunEnv(config)
	state, err := r.loadCollectionRunState(config, runEnv)
	if err != nil {
		return nil, err
	}

	watcher := r.startCollectionRunWatcher(ctx, config.CollectionName)
	defer watcher.stop()

	reloadState := func() error {
		next, err := r.loadCollectionRunState(config, runEnv)
		if err != nil {
			return err
		}
		*state = *next
		return nil
	}
	beforeIteration := func() error {
		if !watcher.changed() {
			return nil
		}
		return reloadState()
	}
	waitDelay := func(delay time.Duration) error {
		return watcher.wait(ctx, delay)
	}
	results := make([]RunResult, 0)

	if config.DataFile != "" {
		dataResults, err := r.runWithData(ctx, state, config, beforeIteration, waitDelay)
		if err != nil {
			return nil, err
		}
		results = append(results, dataResults...)
	} else {
		for iter := 0; iter < config.Iterations; iter++ {
			if iter > 0 {
				if err := beforeIteration(); err != nil {
					return results, err
				}
			}
			result := r.runIteration(ctx, state.requests, state.scopes, state.defaultPersistOrigin, config, iter+1)
			results = append(results, result)

			if config.Delay > 0 && iter < config.Iterations-1 {
				if err := waitDelay(config.Delay); err != nil {
					return results, ctx.Err()
				}
			}
		}
	}

	return results, nil
}

type collectionRunWatcher struct {
	watcher      *storage.CollectionWatcher
	reloadNeeded bool
}

func (r *Runner) startCollectionRunWatcher(ctx context.Context, collection string) *collectionRunWatcher {
	runWatcher := &collectionRunWatcher{}
	if collection == "" {
		return runWatcher
	}
	store, ok := r.db.(storage.CollectionWatcherStore)
	if !ok {
		return runWatcher
	}
	watcher, err := store.WatchCollection(ctx, collection, collectionRunWatchOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: file watcher disabled for collection %q: %v\n", collection, err)
		return runWatcher
	}
	runWatcher.watcher = watcher
	return runWatcher
}

func (w *collectionRunWatcher) stop() {
	if w == nil || w.watcher == nil {
		return
	}
	w.watcher.Stop()
}

func (w *collectionRunWatcher) changed() bool {
	if w == nil || w.watcher == nil {
		return false
	}
	if w.watcher.Changed() {
		w.reloadNeeded = true
	}
	if !w.reloadNeeded {
		return false
	}
	w.reloadNeeded = false
	return true
}

func (w *collectionRunWatcher) wait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	if w == nil || w.watcher == nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			return nil
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-w.watcher.Events():
			if !ok {
				w.watcher = nil
				continue
			}
			w.reloadNeeded = true
		case <-timer.C:
			return nil
		}
	}
}

// loadRunEnv returns the run's environment, or nil when none is selected or
// it cannot be loaded.
func (r *Runner) loadRunEnv(config RunConfig) *env.Environment {
	if config.Environment == "" || r.envStorage == nil {
		return nil
	}
	e, err := r.envStorage.GetEnvByName(config.Environment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: environment %q not found: %v\n", config.Environment, err)
		return nil
	}
	return e
}

func (r *Runner) loadCollectionRunState(config RunConfig, runEnv *env.Environment) (*runCollectionState, error) {
	requests, err := r.db.ListRequests(&storage.ListOptions{
		Collection: config.CollectionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list collection: %w", err)
	}
	if len(requests) == 0 {
		return nil, &EmptyCollectionError{Collection: config.CollectionName}
	}

	return &runCollectionState{
		requests:             sortBySequence(requests),
		scopes:               ScopesFor(r.db, runEnv, config.CollectionName, config.Vars),
		defaultPersistOrigin: persistOriginFor(config.CollectionName),
	}, nil
}

// ScopesFor builds the resolution scopes for running requests against e (nil
// for no environment), the named collection (empty for none) and the
// --var values.
func ScopesFor(db storage.DB, e *env.Environment, collectionName string, cliVars map[string]string) resolve.Scopes {
	s := resolve.Scopes{CLI: cliVars}
	if e != nil {
		s.Env = e.ResolvedVariables()
		s.EnvSecrets = e.SecretKeys
	}
	if collectionName == "" {
		return s
	}
	if store, ok := db.(storage.CollectionStore); ok {
		if collection, err := store.GetCollectionByName(collectionName); err == nil && collection != nil {
			s.Collection = collection.Variables
			s.CollectionSecrets = collection.SecretKeys
		}
	}
	return s
}

// persistOriginFor is the origin given to new extracted or script variables:
// they persist to the collection when the request belongs to one.
func persistOriginFor(collectionName string) resolve.Origin {
	if collectionName != "" {
		return resolve.OriginCollection
	}
	return resolve.OriginEnvironment
}

func (r *Runner) runWithData(ctx context.Context, state *runCollectionState, config RunConfig, beforeIteration func() error, waitDelay func(time.Duration) error) ([]RunResult, error) {
	loader, err := NewDataLoader(config.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load data file: %w", err)
	}

	iteration := 1
	var results []RunResult

	err = loader.Iterate(func(rowVars map[string]string) error {
		if iteration > 1 {
			if err := beforeIteration(); err != nil {
				return err
			}
		}
		scopes := state.scopes
		scopes.Data = rowVars
		result := r.runIteration(ctx, state.requests, scopes, state.defaultPersistOrigin, config, iteration)
		results = append(results, result)
		iteration++

		if config.Delay > 0 {
			return waitDelay(config.Delay)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("data iteration failed: %w", err)
	}

	return results, nil
}

func (r *Runner) runIteration(ctx context.Context, requests []*types.SavedRequest, scopes resolve.Scopes, defaultPersistOrigin resolve.Origin, config RunConfig, iteration int) RunResult {
	start := time.Now()
	scopes.Runtime = copyRuntime(scopes.Runtime)
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

		execution := r.runRequestLifecycle(ctx, req, scopes, defaultPersistOrigin, extractedVars, config.DryRun, plannedVarSources)
		reqResult := execution.Result
		printVerboseExchange(config, execution, req.Name)
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

// printVerboseExchange writes one request/response pair when --verbose is set.
// A dry run has nothing on the wire to show.
func printVerboseExchange(config RunConfig, execution *SingleRequestExecution, name string) {
	if !config.Verbose || config.DryRun {
		return
	}
	out := config.VerboseOut
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintf(out, "\n─── %s ───\n", name)
	fmt.Fprint(out, FormatVerboseExchange(execution.Request, execution.Response))
}

// RunSavedRequest runs req with vars as its only (environment) scope.
func (r *Runner) RunSavedRequest(ctx context.Context, req *types.SavedRequest, vars map[string]string) *SingleRequestExecution {
	scopes := resolve.Scopes{Env: vars, Runtime: make(map[string]resolve.Var)}
	return r.runRequestLifecycle(ctx, req, scopes, resolve.OriginEnvironment, make(map[string]string), false, nil)
}

// RunSavedRequestInScopes runs req resolved against scopes. Variables that
// scripts or extractions create persist to req's collection, or to the
// environment when req has none.
func (r *Runner) RunSavedRequestInScopes(ctx context.Context, req *types.SavedRequest, scopes resolve.Scopes) *SingleRequestExecution {
	scopes.Runtime = copyRuntime(scopes.Runtime)
	return r.runRequestLifecycle(ctx, req, scopes, persistOriginFor(req.Collection), make(map[string]string), false, nil)
}

// PlanSavedRequest runs pre-request scripts and resolves the resulting request
// as a preview without sending it. Missing variables are reported as
// DryRunWarnings, other resolution problems as DryRunProblems, and the
// previewed request as the execution's Request.
func (r *Runner) PlanSavedRequest(ctx context.Context, req *types.SavedRequest, scopes resolve.Scopes) *SingleRequestExecution {
	scopes.Runtime = copyRuntime(scopes.Runtime)
	return r.runRequestLifecycle(ctx, req, scopes, persistOriginFor(req.Collection), make(map[string]string), true, nil)
}

// runRequestLifecycle runs one request. scopes.Runtime must be non-nil; it is
// updated with extracted and script variables so later requests see them.
func (r *Runner) runRequestLifecycle(ctx context.Context, req *types.SavedRequest, scopes resolve.Scopes, defaultPersistOrigin resolve.Origin, extractedVars map[string]string, dryRun bool, plannedVarSources map[string]string) *SingleRequestExecution {
	start := time.Now()
	result := &RequestResult{
		RequestName: req.Name,
	}
	execution := &SingleRequestExecution{Result: result}

	dirtyVars := make(map[string]string)
	dirtyVarOrigins := make(map[string]resolve.Origin)
	effectiveReq := req

	if req.RunIf != "" {
		ok, err := evaluateRunIf(req.RunIf, varValues(scopes))
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
		engine.SetVariables(varValues(scopes))
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
		recordUpdates(scopes, defaultPersistOrigin, extractedVars, dirtyVars, dirtyVarOrigins, engine.DirtyVariables())

		if engine.SkipRequested() {
			result.Skipped = true
			result.SkipReason = engine.SkipReason()
			result.DirtyVars = nonEmptyMap(dirtyVars)
			result.DirtyVarOrigins = nonEmptyOriginMap(dirtyVarOrigins)
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
		preview, problems := resolve.Resolve(effectiveReq, scopes, resolve.Options{Auth: r.auth, Preview: true})
		execution.Request = &preview.Request
		result.DryRun = true
		result.Passed = true
		result.PlannedMethod = preview.Request.Method
		result.PlannedURL = preview.Request.URL
		result.PlannedExtracts = append([]types.Extract(nil), effectiveReq.Extracts...)
		result.PlannedVarSources, result.DryRunWarnings = planMissingVars(problems.Missing(), plannedVarSources)
		result.DryRunProblems = problems.Others()
		result.DirtyVars = nonEmptyMap(dirtyVars)
		result.DirtyVarOrigins = nonEmptyOriginMap(dirtyVarOrigins)
		result.Duration = time.Since(start)
		return execution
	}

	resolved, problems := resolve.Resolve(effectiveReq, scopes, resolve.Options{Auth: r.auth})
	if err := problems.Err(); err != nil {
		result.Error = fmt.Sprintf("variable substitution failed: %v", err)
		result.FailurePhase = FailurePhaseRequestBuild
		result.DirtyVars = nonEmptyMap(dirtyVars)
		result.DirtyVarOrigins = nonEmptyOriginMap(dirtyVarOrigins)
		return execution
	}
	clientReq := resolved.Request
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
		result.DirtyVarOrigins = nonEmptyOriginMap(dirtyVarOrigins)
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
			result.DirtyVarOrigins = nonEmptyOriginMap(dirtyVarOrigins)
			return execution
		}
		result.ExtractedVars = extracted
		recordUpdates(scopes, defaultPersistOrigin, extractedVars, dirtyVars, dirtyVarOrigins, extracted)
	}

	if effectiveReq.PostScript != "" {
		engine := scripting.NewEngine(nil)
		engine.SetVariables(varValues(scopes))
		postResult, err := scripting.RunPostResponse(engine, effectiveReq.PostScript, &resp)
		if err != nil {
			result.Error = fmt.Sprintf("post-response script failed: %v", err)
			result.FailurePhase = FailurePhasePostResponseScript
			result.DirtyVars = nonEmptyMap(dirtyVars)
			result.DirtyVarOrigins = nonEmptyOriginMap(dirtyVarOrigins)
			return execution
		}
		recordUpdates(scopes, defaultPersistOrigin, extractedVars, dirtyVars, dirtyVarOrigins, postResult.Variables)
		result.NextRequestOverride = engine.NextRequest()
	}
	result.DirtyVars = nonEmptyMap(dirtyVars)
	result.DirtyVarOrigins = nonEmptyOriginMap(dirtyVarOrigins)

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

// planMissingVars splits the missing variables of a dry-run step into those
// an earlier step's extraction will provide (with their source) and
// "unresolved {{name}}" warnings, sorted by name.
func planMissingVars(missing []string, plannedVarSources map[string]string) (map[string]string, []string) {
	sources := make(map[string]string)
	var unresolved []string
	for _, name := range missing {
		if source := plannedVarSources[name]; source != "" {
			sources[name] = source
			continue
		}
		unresolved = append(unresolved, name)
	}
	sort.Strings(unresolved)
	var warnings []string
	for _, name := range unresolved {
		warnings = append(warnings, fmt.Sprintf("unresolved {{%s}}", name))
	}
	return sources, warnings
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

// recordUpdates stores extracted or script variables as runtime values. Each
// keeps the origin of the variable it replaces, or defaultPersistOrigin when
// it is new, so --persist writes it back where it came from.
func recordUpdates(scopes resolve.Scopes, defaultPersistOrigin resolve.Origin, extractedVars map[string]string, dirtyVars map[string]string, dirtyOrigins map[string]resolve.Origin, updates map[string]string) {
	current := resolve.Vars(scopes)
	for key, value := range updates {
		origin := defaultPersistOrigin
		if existing, ok := current[key]; ok {
			origin = existing.Origin
		}
		scopes.Runtime[key] = resolve.Var{Value: value, Origin: origin}
		extractedVars[key] = value
		dirtyVars[key] = value
		dirtyOrigins[key] = origin
	}
}

// varValues returns the merged variable values of scopes.
func varValues(scopes resolve.Scopes) map[string]string {
	vars := resolve.Vars(scopes)
	values := make(map[string]string, len(vars))
	for key, v := range vars {
		values[key] = v.Value
	}
	return values
}

func copyRuntime(source map[string]resolve.Var) map[string]resolve.Var {
	runtime := make(map[string]resolve.Var, len(source))
	for key, v := range source {
		runtime[key] = v
	}
	return runtime
}

func nonEmptyMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	return copyStringMap(source)
}

func nonEmptyOriginMap(source map[string]resolve.Origin) map[string]resolve.Origin {
	if len(source) == 0 {
		return nil
	}
	origins := make(map[string]resolve.Origin, len(source))
	for key, origin := range source {
		origins[key] = origin
	}
	return origins
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

func CollectDirtyVarsWithOrigins(results []RunResult) (map[string]string, map[string]resolve.Origin) {
	dirty := make(map[string]string)
	origins := make(map[string]resolve.Origin)
	for _, runResult := range results {
		for _, requestResult := range runResult.RequestResults {
			for key, value := range requestResult.DirtyVars {
				dirty[key] = value
				if origin, ok := requestResult.DirtyVarOrigins[key]; ok {
					origins[key] = origin
				}
			}
		}
	}
	return dirty, origins
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
