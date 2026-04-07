package types

import (
	"time"

	"github.com/google/uuid"
)

// Header represents an HTTP header
type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Var represents a variable in a template
type Var struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Example string `json:"example"`
}

// ParsedCurl represents a parsed curl command
type ParsedCurl struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

// SavedRequest represents a saved curl request
type SavedRequest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	CurlCmd      string   `json:"curl_cmd"`
	URL          string   `json:"url"`
	Method       string   `json:"method"`
	Headers      []Header `json:"headers"`
	Body         string   `json:"body,omitempty"`
	Variables    []Var    `json:"variables,omitempty"`
	Collection   string   `json:"collection,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	OutputFormat string   `json:"output_format"`
	CreatedAt    int64    `json:"created_at"`
	UpdatedAt    int64    `json:"updated_at"`
}

// ExecutionHistory represents an execution record
type ExecutionHistory struct {
	ID         string `json:"id"`
	RequestID  string `json:"request_id"`
	Response   string `json:"response"`
	StatusCode int    `json:"status_code"`
	DurationMs int64  `json:"duration_ms"`
	SizeBytes  int64  `json:"size_bytes"`
	Timestamp  int64  `json:"timestamp"`
}

// NewExecutionHistory creates a new execution history entry
func NewExecutionHistory(requestID string, response string, statusCode int, durationMs int64, sizeBytes int64) *ExecutionHistory {
	return &ExecutionHistory{
		ID:         uuid.New().String(),
		RequestID:  requestID,
		Response:   response,
		StatusCode: statusCode,
		DurationMs: durationMs,
		SizeBytes:  sizeBytes,
		Timestamp:  time.Now().Unix(),
	}
}

// Collection represents a collection of requests
type Collection struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// Config represents the application configuration
type Config struct {
	General struct {
		HistoryDepth   int    `toml:"history_depth"`
		AutoTemplate   bool   `toml:"auto_template"`
		CompletionMode string `toml:"completion_mode"`
	} `toml:"general"`

	Output struct {
		DefaultFormat   string `toml:"default_format"`
		SyntaxHighlight bool   `toml:"syntax_highlight"`
		JSONPretty      bool   `toml:"json_pretty"`
	} `toml:"output"`

	Cache struct {
		TTLSeconds int `toml:"ttl_seconds"`
	} `toml:"cache"`

	Detect struct {
		ExtractVariables bool `toml:"extract_variables"`
		SuggestMerge     bool `toml:"suggest_merge"`
		PromptTemplates  bool `toml:"prompt_templates"`
	} `toml:"detect"`

	UI struct {
		TUIOnDecisions    bool `toml:"tui_on_decisions"`
		TUIThresholdLines int  `toml:"tui_threshold_lines"`
	} `toml:"ui"`

	Plugins struct {
		Enabled []string `toml:"enabled"`
	} `toml:"plugins"`
}

// HistoryEntry is an alias for ExecutionHistory for compatibility
type HistoryEntry = ExecutionHistory

// ListOptions represents filtering options for list command
type ListOptions struct {
	Pattern    string
	Collection string
	Tag        string
	JSON       bool
	Format     string // "table" or "list"
	Limit      int
	Sort       string // "name", "updated", "collection"
}

// NewSavedRequest creates a new SavedRequest with generated ID and timestamps
func NewSavedRequest(name, url, method string) *SavedRequest {
	now := time.Now().Unix()
	return &SavedRequest{
		ID:           uuid.New().String(),
		Name:         name,
		URL:          url,
		Method:       method,
		Headers:      []Header{},
		OutputFormat: "auto",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// AddHeader adds a header to the request
func (r *SavedRequest) AddHeader(key, value string) {
	r.Headers = append(r.Headers, Header{Key: key, Value: value})
}

// AddTag adds a tag to the request
func (r *SavedRequest) AddTag(tag string) {
	r.Tags = append(r.Tags, tag)
}

func ParsedCurlToSavedRequest(parsed ParsedCurl) SavedRequest {
	var headers []Header
	if parsed.Headers != nil {
		headers = make([]Header, 0, len(parsed.Headers))
		for k, v := range parsed.Headers {
			headers = append(headers, Header{Key: k, Value: v})
		}
	}

	return SavedRequest{
		URL:     parsed.URL,
		Method:  parsed.Method,
		Headers: headers,
		Body:    parsed.Body,
	}
}

func SavedRequestToParsedCurl(req SavedRequest) ParsedCurl {
	var headers map[string]string
	if req.Headers != nil {
		headers = make(map[string]string, len(req.Headers))
		for _, h := range req.Headers {
			headers[h.Key] = h.Value
		}
	}

	return ParsedCurl{
		URL:     req.URL,
		Method:  req.Method,
		Headers: headers,
		Body:    req.Body,
	}
}
