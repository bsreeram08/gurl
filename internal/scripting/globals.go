package scripting

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/dop251/goja"
)

func RegisterGlobals(vm *goja.Runtime, e *Engine) {
	vm.Set("gurl", e.vm.NewObject())
	vm.Set("console", e.vm.NewObject())

	e.setupGurlObject(vm)
	e.setupConsoleObject(vm)
}

func (e *Engine) setupGurlObject(vm *goja.Runtime) {
	gurl := vm.Get("gurl").ToObject(vm)

	gurl.Set("setVar", e.jsSetVar)
	gurl.Set("getVar", e.jsGetVar)
	gurl.Set("test", e.jsTest)
	gurl.Set("skipRequest", e.jsSkipRequest)
	gurl.Set("setNextRequest", e.jsSetNextRequest)
	gurl.Set("expect", e.jsExpect)
	gurl.Set("setRequestURL", e.jsSetRequestURL)
	gurl.Set("setRequestBody", e.jsSetRequestBody)

	vm.RunString(`
		Object.defineProperty(gurl, 'request', {
			get: function() {
				return gurl._request || {};
			}
		});
		Object.defineProperty(gurl, 'response', {
			get: function() {
				return gurl._response || {};
			}
		});
	`)

	gurl.Get("request").ToObject(vm).Set("headers", e.vm.NewObject())
	gurl.Get("response").ToObject(vm).Set("headers", e.vm.NewObject())
}

func (e *Engine) setupConsoleObject(vm *goja.Runtime) {
	console := vm.Get("console").ToObject(vm)
	console.Set("log", e.jsConsoleLog)
	console.Set("warn", e.jsConsoleWarn)
	console.Set("error", e.jsConsoleError)
}

func (e *Engine) jsSetVar(call goja.FunctionCall) goja.Value {
	nameVal := call.Argument(0).Export()
	name, _ := nameVal.(string)
	valueVal := call.Argument(1).Export()
	value, _ := valueVal.(string)
	if e.variables == nil {
		e.variables = make(map[string]string)
	}
	e.variables[name] = value
	return goja.Undefined()
}

func (e *Engine) jsGetVar(call goja.FunctionCall) goja.Value {
	nameVal := call.Argument(0).Export()
	name, _ := nameVal.(string)
	if e.variables != nil {
		if val, ok := e.variables[name]; ok {
			return e.vm.ToValue(val)
		}
	}
	if e.envStorage != nil {
		if envName, err := e.envStorage.GetActiveEnv(); err == nil && envName != "" {
			if envObj, err := e.envStorage.GetEnvByName(envName); err == nil {
				if val, ok := envObj.GetVariable(name); ok {
					if !e.AllowSecretAccess && envObj.IsSecret(name) {
						panic(errors.New("access to secret variable '" + name + "' is denied"))
					}
					return e.vm.ToValue(val)
				}
			}
		}
	}
	return goja.Null()
}

func (e *Engine) jsRequest(call goja.FunctionCall) goja.Value {
	obj := e.vm.NewObject()

	url := ""
	method := ""
	body := ""

	if e.request != nil {
		url = e.request.URL
		method = e.request.Method
		body = e.request.Body
	}

	obj.Set("url", url)
	obj.Set("method", method)
	obj.Set("body", body)

	headersObj := e.vm.NewObject()
	if e.request != nil {
		for _, h := range e.request.Headers {
			headersObj.Set(h.Key, h.Value)
		}
	}
	headersObj.Set("set", e.jsRequestHeadersSet)
	headersObj.Set("get", e.jsRequestHeadersGet)
	headersObj.Set("remove", e.jsRequestHeadersRemove)
	obj.Set("headers", headersObj)

	return obj
}

func (e *Engine) jsRequestHeadersSet(call goja.FunctionCall) goja.Value {
	keyVal := call.Argument(0).Export()
	key, _ := keyVal.(string)
	valueVal := call.Argument(1).Export()
	value, _ := valueVal.(string)
	if e.request != nil {
		found := false
		for i := range e.request.Headers {
			if e.request.Headers[i].Key == key {
				e.request.Headers[i].Value = value
				found = true
				break
			}
		}
		if !found {
			e.request.Headers = append(e.request.Headers, Header{Key: key, Value: value})
		}
	}
	return goja.Undefined()
}

func (e *Engine) jsRequestHeadersGet(call goja.FunctionCall) goja.Value {
	keyVal := call.Argument(0).Export()
	key, _ := keyVal.(string)
	if e.request != nil {
		for _, h := range e.request.Headers {
			if h.Key == key {
				return e.vm.ToValue(h.Value)
			}
		}
	}
	return goja.Null()
}

func (e *Engine) jsRequestHeadersRemove(call goja.FunctionCall) goja.Value {
	keyVal := call.Argument(0).Export()
	key, _ := keyVal.(string)
	if e.request != nil {
		filtered := make([]Header, 0)
		for _, h := range e.request.Headers {
			if h.Key != key {
				filtered = append(filtered, h)
			}
		}
		e.request.Headers = filtered
	}
	return goja.Undefined()
}

func (e *Engine) jsSetRequestURL(call goja.FunctionCall) goja.Value {
	val := call.Argument(0).Export()
	if v, ok := val.(string); ok && e.request != nil {
		parsed, err := url.Parse(v)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			panic(errors.New("invalid URL: must be a valid http/https URL"))
		}
		e.request.URL = v
	}
	return goja.Undefined()
}

func (e *Engine) jsSetRequestBody(call goja.FunctionCall) goja.Value {
	val := call.Argument(0).Export()
	if v, ok := val.(string); ok && e.request != nil {
		e.request.Body = v
	}
	return goja.Undefined()
}

func (e *Engine) jsResponse(call goja.FunctionCall) goja.Value {
	obj := e.vm.NewObject()

	status := 0
	statusText := ""
	body := ""

	if e.response != nil {
		status = e.response.Status
		statusText = e.response.StatusText
		body = string(e.response.Body)
	}

	obj.Set("status", status)
	obj.Set("statusText", statusText)
	obj.Set("body", body)
	obj.Set("time", int64(e.response.Time))
	obj.Set("size", e.response.Size)

	headersObj := e.vm.NewObject()
	if e.response != nil {
		for k, v := range e.response.Headers {
			if len(v) > 0 {
				headersObj.Set(k, v[0])
			}
		}
	}
	obj.Set("headers", headersObj)

	return obj
}

func (e *Engine) jsTest(call goja.FunctionCall) goja.Value {
	nameVal := call.Argument(0).Export()
	name, _ := nameVal.(string)
	fn := call.Argument(1)

	testPassed := true
	var testErrMsg string

	if fnObj, ok := goja.AssertFunction(fn); ok {
		func() {
			defer func() {
				if r := recover(); r != nil {
					testPassed = false
					if err, ok := r.(error); ok {
						testErrMsg = err.Error()
					} else {
						testErrMsg = fmt.Sprintf("%v", r)
					}
				}
			}()
			_, _ = fnObj(goja.Undefined())
		}()
	}

	e.testResults = append(e.testResults, TestResult{
		Name:   name,
		Passed: testPassed,
		Error:  testErrMsg,
	})

	return goja.Undefined()
}

func (e *Engine) jsSkipRequest(call goja.FunctionCall) goja.Value {
	e.skipRequest = true
	return goja.Undefined()
}

func (e *Engine) jsSetNextRequest(call goja.FunctionCall) goja.Value {
	nameVal := call.Argument(0).Export()
	name, _ := nameVal.(string)
	e.nextRequest = name
	return goja.Undefined()
}

func (e *Engine) jsConsoleLog(call goja.FunctionCall) goja.Value {
	args := call.Arguments
	output := ""
	for i, arg := range args {
		if i > 0 {
			output += " "
		}
		output += arg.String()
	}
	e.outputBuffer += output + "\n"
	return goja.Undefined()
}

func (e *Engine) jsConsoleWarn(call goja.FunctionCall) goja.Value {
	args := call.Arguments
	output := ""
	for i, arg := range args {
		if i > 0 {
			output += " "
		}
		output += arg.String()
	}
	e.outputBuffer += output + "\n"
	return goja.Undefined()
}

func (e *Engine) jsConsoleError(call goja.FunctionCall) goja.Value {
	args := call.Arguments
	output := ""
	for i, arg := range args {
		if i > 0 {
			output += " "
		}
		output += arg.String()
	}
	e.outputBuffer += output + "\n"
	return goja.Undefined()
}

func (e *Engine) jsRequire(call goja.FunctionCall) goja.Value {
	moduleVal := call.Argument(0).Export()
	module, _ := moduleVal.(string)
	switch module {
	case "crypto", "JSON", "Math", "Date", "Buffer":
		return goja.Undefined()
	}
	panic(errors.New(fmt.Sprintf("Access to module '%s' is not allowed", module)))
}

func (e *Engine) jsExpect(call goja.FunctionCall) goja.Value {
	val := call.Argument(0).Export()
	expectObj := e.vm.NewObject()
	expectVal := val

	expectObj.Set("to", e.vm.NewObject())

	to := expectObj.Get("to").ToObject(e.vm)
	to.Set("equal", func(call goja.FunctionCall) goja.Value {
		expected := call.Argument(0).Export()
		actual := expectVal

		switch actualVal := actual.(type) {
		case int64:
			if expectedInt, ok := expected.(int64); ok {
				if actualVal != expectedInt {
					panic(errors.New(fmt.Sprintf("Expected %d but got %d", expectedInt, actualVal)))
				}
			}
		case string:
			if expectedStr, ok := expected.(string); ok {
				if actualVal != expectedStr {
					panic(errors.New(fmt.Sprintf("Expected '%s' but got '%s'", expectedStr, actualVal)))
				}
			}
		}
		return goja.Undefined()
	})

	return expectObj
}
