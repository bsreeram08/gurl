package scripting

import (
	"context"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/sreeram/gurl/internal/env"
)

type EngineOption func(*Engine)

type Engine struct {
	vm                *goja.Runtime
	envStorage        *env.EnvStorage
	timeout           time.Duration
	outputBuffer      string
	testResults       []TestResult
	skipRequest       bool
	nextRequest       string
	request           *ScriptRequest
	response          *ScriptResponse
	variables         map[string]string
	AllowSecretAccess bool
	pool              *RuntimePool
}

type ScriptRequest struct {
	Method  string
	URL     string
	Headers []Header
	Body    string
}

type Header struct {
	Key   string
	Value string
}

type ScriptResponse struct {
	Status     int
	Body       []byte
	StatusText string
	Headers    map[string][]string
	Time       time.Duration
	Size       int64
}

type Result struct {
	Value  interface{}
	Error  error
	Output string
}

type TestResult struct {
	Name   string
	Passed bool
	Error  string
}

// RuntimePool manages a pool of reusable goja.Runtime instances
type RuntimePool struct {
	pool sync.Pool
}

func NewRuntimePool() *RuntimePool {
	p := &RuntimePool{
		pool: sync.Pool{
			New: func() interface{} {
				vm := goja.New()
				registerSandboxRestricted(vm)
				return vm
			},
		},
	}
	return p
}

func (p *RuntimePool) Get() *goja.Runtime {
	return p.pool.Get().(*goja.Runtime)
}

func (p *RuntimePool) Put(vm *goja.Runtime) {
	// Clear any pending interrupt and reset for reuse
	vm.ClearInterrupt()
	// Re-run sandbox setup to re-initialize restricted modules
	registerSandboxRestricted(vm)
	p.pool.Put(vm)
}

func NewEngine(envStorage *env.EnvStorage, opts ...EngineOption) *Engine {
	eng := &Engine{
		envStorage: envStorage,
		timeout:    5 * time.Second,
		pool:       NewRuntimePool(),
	}
	for _, opt := range opts {
		opt(eng)
	}
	return eng
}

func (e *Engine) getRuntime() *goja.Runtime {
	return e.pool.Get()
}

func (e *Engine) putRuntime(vm *goja.Runtime) {
	e.pool.Put(vm)
}

func (e *Engine) Execute(script string) (*Result, error) {
	e.outputBuffer = ""
	e.testResults = nil
	e.skipRequest = false
	e.nextRequest = ""

	vm := e.getRuntime()
	e.vm = vm

	RegisterGlobals(vm, e)
	e.updateGurlRequest(vm)
	e.updateGurlResponse(vm)

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	done := make(chan struct{})
	var result *Result
	var execErr error

	go func() {
		defer close(done)
		res, err := vm.RunString(script)
		if err != nil {
			execErr = err
			result = &Result{Error: err}
			return
		}
		result = &Result{
			Value:  res.Export(),
			Output: e.outputBuffer,
		}
	}()

	select {
	case <-done:
		cancel()
		e.putRuntime(vm)
		return result, execErr
	case <-ctx.Done():
		cancel()
		vm.Interrupt("timeout exceeded")
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		e.putRuntime(vm)
		return &Result{Error: ctx.Err()}, ctx.Err()
	}
}

func (e *Engine) PrepareRequest(req *ScriptRequest) {
	e.request = req
	if e.vm != nil {
		e.updateGurlRequest(e.vm)
	}
}

func (e *Engine) PrepareResponse(resp *ScriptResponse) {
	e.response = resp
	if e.vm != nil {
		e.updateGurlResponse(e.vm)
	}
}

func (e *Engine) updateGurlRequest(vm *goja.Runtime) {
	if e.request == nil {
		return
	}
	gurl := vm.Get("gurl").ToObject(vm)

	requestObj := vm.NewObject()

	headersObj := vm.NewObject()
	for _, h := range e.request.Headers {
		headersObj.Set(h.Key, h.Value)
	}
	headersObj.Set("set", e.jsRequestHeadersSet)
	headersObj.Set("get", e.jsRequestHeadersGet)
	headersObj.Set("remove", e.jsRequestHeadersRemove)
	requestObj.Set("headers", headersObj)

	gurl.Set("_request", requestObj)

	vm.RunString(`
		Object.defineProperty(gurl._request, 'url', {
			get: function() { return gurl._request._url; },
			set: function(v) { gurl._request._url = v; gurl.setRequestURL(v); },
			enumerable: true,
			configurable: true
		});
		Object.defineProperty(gurl._request, 'body', {
			get: function() { return gurl._request._body; },
			set: function(v) { gurl._request._body = v; gurl.setRequestBody(v); },
			enumerable: true,
			configurable: true
		});
		Object.defineProperty(gurl._request, 'method', {
			get: function() { return gurl._request._method; },
			set: function(v) { gurl._request._method = v; },
			enumerable: true,
			configurable: true
		});
	`)

	requestObj.Set("_url", e.request.URL)
	requestObj.Set("_body", e.request.Body)
	requestObj.Set("_method", e.request.Method)
}

func (e *Engine) updateGurlResponse(vm *goja.Runtime) {
	if e.response == nil {
		return
	}
	gurl := vm.Get("gurl").ToObject(vm)

	responseObj := vm.NewObject()
	responseObj.Set("status", e.response.Status)
	responseObj.Set("statusText", e.response.StatusText)
	responseObj.Set("body", string(e.response.Body))
	responseObj.Set("time", int64(e.response.Time))
	responseObj.Set("size", e.response.Size)

	headersObj := vm.NewObject()
	for k, v := range e.response.Headers {
		if len(v) > 0 {
			headersObj.Set(k, v[0])
		}
	}
	headersObj.Set("get", e.jsResponseHeadersGet)
	responseObj.Set("headers", headersObj)

	responseObj.Set("json", e.jsResponseJSON)
	responseObj.Set("text", e.jsResponseText)

	gurl.Set("_response", responseObj)
}

func WithTimeout(timeout time.Duration) EngineOption {
	return func(e *Engine) {
		e.timeout = timeout
	}
}
