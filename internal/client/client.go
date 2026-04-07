package client

import (
	"context"
	"net/http"
	"time"
)

type Client struct {
	transport *http.Transport
	timeout   time.Duration
}

func NewClient() *Client {
	return &Client{
		transport: &http.Transport{
			DisableKeepAlives: true,
		},
		timeout: defaultTimeout,
	}
}

func (c *Client) Execute(req Request) (Response, error) {
	return c.ExecuteWithContext(context.Background(), req)
}

func (c *Client) ExecuteWithContext(ctx context.Context, req Request) (Response, error) {
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, nil)
	if err != nil {
		return Response{}, err
	}

	for _, h := range req.Headers {
		httpReq.Header.Set(h.Key, h.Value)
	}

	if req.Body != "" {
		httpReq.Body = http.NoBody
	}

	client := &http.Client{
		Transport: c.transport,
		Timeout:   c.timeout,
	}

	start := time.Now()
	httpResp, err := client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return Response{}, err
	}
	defer httpResp.Body.Close()

	body := make([]byte, 0, 1024)
	buf := make([]byte, 4096)
	for {
		n, readErr := httpResp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if readErr != nil {
			break
		}
	}

	return Response{
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       body,
		Duration:   duration,
		Size:       int64(len(body)),
	}, nil
}
