package curl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/pkg/types"
)

func ExecuteCurl(request *types.SavedRequest, vars map[string]string) (*types.ExecutionHistory, error) {
	clientReq, err := buildClientRequest(request, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	start := time.Now()
	resp, err := client.Execute(clientReq)
	duration := time.Since(start)

	if err != nil {
		return &types.ExecutionHistory{
			ID:         generateID(),
			RequestID:  request.ID,
			Response:   err.Error(),
			StatusCode: 0,
			DurationMs: duration.Milliseconds(),
			SizeBytes:  0,
			Timestamp:  time.Now().Unix(),
		}, nil
	}

	return &types.ExecutionHistory{
		ID:         generateID(),
		RequestID:  request.ID,
		Response:   string(resp.Body),
		StatusCode: resp.StatusCode,
		DurationMs: duration.Milliseconds(),
		SizeBytes:  resp.Size,
		Timestamp:  time.Now().Unix(),
	}, nil
}

func buildClientRequest(request *types.SavedRequest, vars map[string]string) (client.Request, error) {
	url, err := template.Substitute(request.URL, vars)
	if err != nil {
		return client.Request{}, fmt.Errorf("failed to substitute URL variables: %w", err)
	}

	headers := make([]client.Header, 0, len(request.Headers))
	for _, h := range request.Headers {
		key, err := template.Substitute(h.Key, vars)
		if err != nil {
			return client.Request{}, fmt.Errorf("failed to substitute header key: %w", err)
		}
		value, err := template.Substitute(h.Value, vars)
		if err != nil {
			return client.Request{}, fmt.Errorf("failed to substitute header value: %w", err)
		}
		headers = append(headers, client.Header{Key: key, Value: value})
	}

	var body string
	if request.Body != "" {
		body, err = template.Substitute(request.Body, vars)
		if err != nil {
			return client.Request{}, fmt.Errorf("failed to substitute body: %w", err)
		}
	}

	method := request.Method
	if method == "" {
		method = "GET"
	}

	return client.Request{
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    body,
	}, nil
}

func BuildCurlCommand(request *types.SavedRequest, vars map[string]string) ([]string, error) {
	url, err := template.Substitute(request.URL, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute URL variables: %w", err)
	}

	args := []string{
		"-s",
		"-w", "\n%{http_code}",
		"-o", "-",
	}

	if request.Method != "" && request.Method != "GET" {
		args = append(args, "-X", request.Method)
	}

	for _, header := range request.Headers {
		key, err := template.Substitute(header.Key, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute header key: %w", err)
		}
		value, err := template.Substitute(header.Value, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute header value: %w", err)
		}
		args = append(args, "-H", fmt.Sprintf("%s: %s", key, value))
	}

	if request.Body != "" {
		body, err := template.Substitute(request.Body, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute body: %w", err)
		}
		args = append(args, "-d", body)
	}

	args = append(args, url)

	return args, nil
}

func parseStatusCode(output string) int {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return 0
	}

	lastLine := lines[len(lines)-1]
	statusCode, err := strconv.Atoi(strings.TrimSpace(lastLine))
	if err != nil {
		return 0
	}

	return statusCode
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func ExecuteCurlWithOutput(request *types.SavedRequest, vars map[string]string) (string, int, int64, error) {
	clientReq, err := buildClientRequest(request, vars)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to build request: %w", err)
	}

	start := time.Now()
	resp, err := client.Execute(clientReq)
	duration := time.Since(start)

	if err != nil {
		return err.Error(), 0, duration.Milliseconds(), nil
	}

	return string(resp.Body), resp.StatusCode, duration.Milliseconds(), nil
}

var statusCodePattern = regexp.MustCompile(`\d{3}`)

func ParseStatusCodeFromResponse(response string) int {
	lines := strings.Split(response, "\n")
	for i := 0; i < len(lines) && i < 5; i++ {
		if matches := statusCodePattern.FindAllString(lines[i], 1); len(matches) > 0 {
			if code, err := strconv.Atoi(matches[0]); err == nil && code >= 100 && code < 600 {
				return code
			}
		}
	}
	return 0
}
