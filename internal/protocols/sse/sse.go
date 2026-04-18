package sse

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Event struct {
	ID    string
	Type  string
	Data  string
	Retry int
}

type Option func(*options)

type options struct {
	headers         map[string]string
	eventTypes      []string
	lastEventID     string
	timeout         time.Duration
	maxScanTokenSize int // 0 means use default 1MB limit
}

func WithHeader(key, value string) Option {
	return func(o *options) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

func WithEventType(eventType string) Option {
	return func(o *options) {
		o.eventTypes = append(o.eventTypes, eventType)
	}
}

func WithLastEventID(id string) Option {
	return func(o *options) {
		o.lastEventID = id
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

// WithMaxScanTokenSize sets the maximum size for a single token (line) in the SSE stream.
// Default is 1MB. Setting this to a smaller value may cause truncation of long lines.
// Setting to 0 uses the default 1MB limit.
func WithMaxScanTokenSize(size int) Option {
	return func(o *options) {
		o.maxScanTokenSize = size
	}
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Connect(ctx context.Context, url string, opts ...Option) (<-chan Event, <-chan error, error) {
	o := &options{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(o)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")

	if o.lastEventID != "" {
		req.Header.Set("Last-Event-ID", o.lastEventID)
	}

	for k, v := range o.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}

	eventChan := make(chan Event, 100)
	errorChan := make(chan error, 1)

	go c.readEvents(resp.Body, eventChan, errorChan, o.eventTypes, o.lastEventID, o.maxScanTokenSize)

	return eventChan, errorChan, nil
}

func (c *Client) readEvents(body io.ReadCloser, eventChan chan<- Event, errorChan chan<- error, filterTypes []string, lastEventID string, maxScanTokenSize int) {
	defer body.Close()
	defer close(eventChan)
	defer close(errorChan)

	scanner := bufio.NewScanner(body)

	// Configure scanner buffer size
	const defaultMaxScanTokenSize = 1024 * 1024 // 1MB
	if maxScanTokenSize <= 0 {
		maxScanTokenSize = defaultMaxScanTokenSize
	}
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	var currentEvent Event
	currentEvent.ID = lastEventID
	var dataBuilder strings.Builder
	var hasFields bool

	for scanner.Scan() {
		line := scanner.Text()

		if len(line) >= maxScanTokenSize {
			select {
			case errorChan <- fmt.Errorf("SSE line exceeded maximum token size of %d bytes; consider using WithMaxScanTokenSize to increase limit", maxScanTokenSize):
			default:
			}
		}

		if line == "" {
			if hasFields {
				if dataBuilder.Len() > 0 {
					currentEvent.Data = dataBuilder.String()
					dataBuilder.Reset()
				}

				if len(filterTypes) == 0 || contains(filterTypes, currentEvent.Type) {
					select {
					case eventChan <- currentEvent:
					default:
						fmt.Printf("SSE warning: event channel full, dropping event (ID=%s, Type=%s)\n", currentEvent.ID, currentEvent.Type)
					}
				}

				if currentEvent.ID != "" {
					lastEventID = currentEvent.ID
				}
				currentEvent = Event{ID: lastEventID}
				hasFields = false
			}
			continue
		}

		var field, value string
		if idx := strings.Index(line, ":"); idx != -1 {
			field = strings.TrimSpace(line[:idx])
			value = strings.TrimSpace(line[idx+1:])
		} else {
			continue
		}

		hasFields = true

		switch field {
		case "data":
			if dataBuilder.Len() > 0 {
				dataBuilder.WriteString("\n")
			}
			dataBuilder.WriteString(value)

		case "event":
			currentEvent.Type = value

		case "id":
			currentEvent.ID = value

		case "retry":
			if retryMs, err := strconv.ParseInt(value, 10, 64); err == nil {
				currentEvent.Retry = int(retryMs)
			}
		}
	}

	if dataBuilder.Len() > 0 {
		currentEvent.Data = dataBuilder.String()
	}
	if currentEvent.Data != "" && (len(filterTypes) == 0 || contains(filterTypes, currentEvent.Type)) {
		select {
		case eventChan <- currentEvent:
		default:
			fmt.Printf("SSE warning: event channel full, dropping final event (ID=%s, Type=%s)\n", currentEvent.ID, currentEvent.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case errorChan <- fmt.Errorf("error reading SSE stream: %w", err):
		default:
		}
	}
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func (c *Client) ConnectWithTimeout(ctx context.Context, url string, timeout time.Duration, opts ...Option) (<-chan Event, <-chan error, error) {
	opts = append(opts, WithTimeout(timeout))
	return c.Connect(ctx, url, opts...)
}
