package curl

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/pkg/types"
)

// ExecuteCurl executes a saved request with variable substitution
func ExecuteCurl(request *types.SavedRequest, vars map[string]string) (*types.ExecutionHistory, error) {
	// Build curl command from template
	cmd, err := BuildCurlCommand(request, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to build curl command: %w", err)
	}

	// Execute the command
	start := time.Now()
	result, err := exec.Command("curl", cmd...).Output()
	duration := time.Since(start)

	if err != nil {
		// Check if it's a curl execution error
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &types.ExecutionHistory{
				ID:         generateID(),
				RequestID:  request.ID,
				Response:   string(exitErr.Stderr),
				StatusCode: 0,
				DurationMs: duration.Milliseconds(),
				SizeBytes:  int64(len(result)),
				Timestamp:  time.Now().Unix(),
			}, nil
		}
		return nil, fmt.Errorf("curl execution failed: %w", err)
	}

	// Parse status code from output
	statusCode := parseStatusCode(string(result))

	// Create execution history
	history := &types.ExecutionHistory{
		ID:         generateID(),
		RequestID:  request.ID,
		Response:   string(result),
		StatusCode: statusCode,
		DurationMs: duration.Milliseconds(),
		SizeBytes:  int64(len(result)),
		Timestamp:  time.Now().Unix(),
	}

	return history, nil
}

// BuildCurlCommand builds the curl command arguments from a saved request
func BuildCurlCommand(request *types.SavedRequest, vars map[string]string) ([]string, error) {
	// Substitute variables in URL
	url, err := template.Substitute(request.URL, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute URL variables: %w", err)
	}

	args := []string{
		"-s", // Silent mode
		"-w", "\n%{http_code}", // Write out status code
		"-o", "-", // Output to stdout
	}

	// Add method if not GET
	if request.Method != "" && request.Method != "GET" {
		args = append(args, "-X", request.Method)
	}

	// Add headers
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

	// Add body if present
	if request.Body != "" {
		body, err := template.Substitute(request.Body, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute body: %w", err)
		}
		args = append(args, "-d", body)
	}

	// Add URL
	args = append(args, url)

	return args, nil
}

// parseStatusCode extracts the HTTP status code from curl output
// curl outputs the body followed by the status code on a new line
func parseStatusCode(output string) int {
	// The status code is at the end after \n%{http_code}
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return 0
	}

	// Last line should be the status code
	lastLine := lines[len(lines)-1]
	statusCode, err := strconv.Atoi(strings.TrimSpace(lastLine))
	if err != nil {
		return 0
	}

	return statusCode
}

// generateID generates a simple ID for execution history
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ExecuteCurlWithOutput is like ExecuteCurl but returns combined stdout/stderr
func ExecuteCurlWithOutput(request *types.SavedRequest, vars map[string]string) (string, int, int64, error) {
	// Build curl command
	cmdArgs, err := BuildCurlCommand(request, vars)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to build curl command: %w", err)
	}

	// Execute
	start := time.Now()
	cmd := exec.Command("curl", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	duration := time.Since(start)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(exitErr.Stderr), 0, duration.Milliseconds(), nil
		}
		return "", 0, duration.Milliseconds(), fmt.Errorf("curl execution failed: %w", err)
	}

	output := stdout.String()
	statusCode := parseStatusCode(output)

	return output, statusCode, duration.Milliseconds(), nil
}

// parseHTTPStatusCode uses a simpler regex to extract status code
var statusCodePattern = regexp.MustCompile(`\d{3}`)

// ParseStatusCodeFromResponse tries to extract HTTP status code from response
func ParseStatusCodeFromResponse(response string) int {
	// Look for status code pattern in first few lines
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
