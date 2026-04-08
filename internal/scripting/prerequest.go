package scripting

import (
	"github.com/sreeram/gurl/internal/client"
)

func RunPreRequest(engine *Engine, script string, req *client.Request) (*client.Request, error) {
	scriptReq := &ScriptRequest{
		Method:  req.Method,
		URL:     req.URL,
		Headers: make([]Header, len(req.Headers)),
		Body:    req.Body,
	}
	for i, h := range req.Headers {
		scriptReq.Headers[i] = Header{Key: h.Key, Value: h.Value}
	}

	engine.PrepareRequest(scriptReq)

	_, err := engine.Execute(script)
	if err != nil {
		return req, err
	}

	if engine.skipRequest {
		result := *req
		result.Headers = make([]client.Header, len(scriptReq.Headers))
		for i, h := range scriptReq.Headers {
			result.Headers[i] = client.Header{Key: h.Key, Value: h.Value}
		}
		result.URL = scriptReq.URL
		result.Body = scriptReq.Body
		return &result, nil
	}

	result := *req
	result.Method = scriptReq.Method
	result.URL = scriptReq.URL
	result.Body = scriptReq.Body
	result.Headers = make([]client.Header, len(scriptReq.Headers))
	for i, h := range scriptReq.Headers {
		result.Headers[i] = client.Header{Key: h.Key, Value: h.Value}
	}

	return &result, nil
}
