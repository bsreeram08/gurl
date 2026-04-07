package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Location represents a location in a GraphQL query
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message   string        `json:"message"`
	Locations []Location    `json:"locations,omitempty"`
	Path      []interface{} `json:"path,omitempty"`
}

// Request represents a GraphQL request
type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// Response represents a GraphQL response
type Response struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// Option is a functional option for GraphQL requests
type Option func(*options)

type options struct {
	headers map[string]string
}

func WithHeader(key, value string) Option {
	return func(o *options) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// Client wraps an HTTP client for GraphQL requests
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new GraphQL client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

// Execute sends a GraphQL request to the specified endpoint
func (c *Client) Execute(ctx context.Context, endpoint string, req Request, opts ...Option) (*Response, error) {
	// Apply options
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	// Ensure httpClient is initialized
	if c.httpClient == nil {
		c.httpClient = &http.Client{}
	}

	// Build request body
	bodyMap := map[string]interface{}{
		"query": req.Query,
	}
	if req.Variables != nil {
		bodyMap["variables"] = req.Variables
	}
	if req.OperationName != "" {
		bodyMap["operationName"] = req.OperationName
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Apply custom headers
	for k, v := range o.headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse GraphQL response
	var graphqlResp Response
	if err := json.Unmarshal(respBody, &graphqlResp); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	// Handle null data (when data is null in JSON, RawMessage becomes []byte("null"))
	if bytes.Equal(graphqlResp.Data, []byte("null")) {
		graphqlResp.Data = nil
	}

	return &graphqlResp, nil
}
