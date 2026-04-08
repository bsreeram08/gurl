package scripting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dop251/goja"
	"github.com/sreeram/gurl/internal/client"
)

type PostResponseResult struct {
	Assertions []AssertionResult
	Variables  map[string]string
	Logs       []string
}

type AssertionResult struct {
	Name   string
	Passed bool
	Error  string
}

func RunPostResponse(engine *Engine, script string, resp *client.Response) (*PostResponseResult, error) {
	scriptResp := &ScriptResponse{
		Status:     resp.StatusCode,
		Body:       resp.Body,
		StatusText: http.StatusText(resp.StatusCode),
		Headers:    resp.Headers,
		Time:       resp.Duration,
		Size:       resp.Size,
	}

	engine.PrepareResponse(scriptResp)

	if engine.vm != nil {
		engine.addResponseMethods(engine.vm)
	}

	result, err := engine.Execute(script)

	postRespResult := &PostResponseResult{
		Assertions: make([]AssertionResult, len(engine.testResults)),
		Variables:  engine.variables,
		Logs:       splitLines(engine.outputBuffer),
	}

	for i, tr := range engine.testResults {
		postRespResult.Assertions[i] = AssertionResult{
			Name:   tr.Name,
			Passed: tr.Passed,
			Error:  tr.Error,
		}
	}

	if result != nil && result.Error != nil && err == nil {
		err = result.Error
	}

	return postRespResult, err
}

func (e *Engine) addResponseMethods(vm *goja.Runtime) {
	gurl := vm.Get("gurl").ToObject(vm)
	responseObj := gurl.Get("response").ToObject(vm)

	responseObj.Set("json", e.jsResponseJSON)
	responseObj.Set("text", e.jsResponseText)

	headersObj := responseObj.Get("headers").ToObject(vm)
	headersObj.Set("get", e.jsResponseHeadersGet)
}

func (e *Engine) jsResponseJSON(call goja.FunctionCall) goja.Value {
	if e.response == nil || len(e.response.Body) == 0 {
		return goja.Null()
	}

	var data interface{}
	if err := json.Unmarshal(e.response.Body, &data); err != nil {
		panic(fmt.Sprintf("Failed to parse JSON: %v", err))
	}

	return e.vm.ToValue(data)
}

func (e *Engine) jsResponseText(call goja.FunctionCall) goja.Value {
	if e.response == nil {
		return goja.Null()
	}
	return e.vm.ToValue(string(e.response.Body))
}

func (e *Engine) jsResponseHeadersGet(call goja.FunctionCall) goja.Value {
	if e.response == nil {
		return goja.Undefined()
	}

	keyVal := call.Argument(0).Export()
	key, ok := keyVal.(string)
	if !ok {
		return goja.Undefined()
	}

	if values, exists := e.response.Headers[key]; exists && len(values) > 0 {
		return e.vm.ToValue(values[0])
	}

	for k, values := range e.response.Headers {
		if strings.EqualFold(k, key) && len(values) > 0 {
			return e.vm.ToValue(values[0])
		}
	}

	return goja.Undefined()
}

func splitLines(output string) []string {
	if output == "" {
		return nil
	}
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) == 0 {
		return nil
	}
	return lines
}
