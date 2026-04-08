package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var ErrTooManyRedirects = errors.New("too many redirects")

// wrapTimeoutError wraps context deadline exceeded errors with a friendly message
func wrapTimeoutError(err error, timeout time.Duration) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("request timed out after %v", timeout)
	}
	return err
}

// Option is a functional option for configuring the Client
type Option func(*Client)

// WithTimeout sets the total request timeout
func WithTimeout(total time.Duration) Option {
	return func(c *Client) {
		c.timeout = total
	}
}

// WithConnectTimeout sets the connection establishment timeout
func WithConnectTimeout(connect time.Duration) Option {
	return func(c *Client) {
		c.connectTimeout = connect
		c.applyConnectTimeout()
	}
}

// applyConnectTimeout applies the connect timeout to the transport's DialContext
func (c *Client) applyConnectTimeout() {
	if c.transport == nil {
		c.transport = &http.Transport{}
	}
	dialer := &net.Dialer{
		Timeout: c.connectTimeout,
	}
	c.transport.DialContext = dialer.DialContext
}

type statusCapturingTransport struct {
	http.RoundTripper
	onResponse func(statusCode int)
}

func (t *statusCapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.RoundTripper.RoundTrip(req)
	if err == nil && resp != nil {
		t.onResponse(resp.StatusCode)
	}
	return resp, err
}

type TLSConfig struct {
	CertFile      string
	KeyFile       string
	CAFile        string
	Insecure      bool
	MinTLSVersion string
}

type Client struct {
	transport      *http.Transport
	timeout        time.Duration
	connectTimeout time.Duration
	proxyConfig    *proxyConfig
	Jar            http.CookieJar
}

func NewClient() *Client {
	return &Client{
		transport: &http.Transport{
			DisableKeepAlives: true,
		},
		timeout: defaultTimeout,
	}
}

func NewClientWithTLS(cfg TLSConfig) *Client {
	tlsConfig := &tls.Config{}

	if cfg.Insecure {
		tlsConfig.InsecureSkipVerify = true
		fmt.Fprintf(os.Stderr, "WARNING: TLS verification disabled. This is insecure and should only be used for testing.\n")
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Failed to load client certificate: %v\n", err)
		} else {
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Failed to read CA file: %v\n", err)
		} else {
			caCertPool := x509.NewCertPool()
			if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
				fmt.Fprintf(os.Stderr, "WARNING: Failed to parse CA certificate\n")
			} else {
				tlsConfig.RootCAs = caCertPool
			}
		}
	}

	if cfg.MinTLSVersion != "" {
		version, err := parseTLSVersion(cfg.MinTLSVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Invalid TLS version '%s': %v\n", cfg.MinTLSVersion, err)
		} else {
			tlsConfig.MinVersion = version
		}
	}

	return &Client{
		transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:   tlsConfig,
		},
		timeout: defaultTimeout,
	}
}

func parseTLSVersion(version string) (uint16, error) {
	switch version {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported TLS version: %s", version)
	}
}

func (c *Client) Execute(req Request) (Response, error) {
	return c.ExecuteWithContext(context.Background(), req)
}

func (c *Client) ExecuteWithContext(ctx context.Context, req Request) (Response, error) {
	// Determine effective timeout: per-request overrides client default
	effectiveTimeout := c.timeout
	if req.Timeout > 0 {
		effectiveTimeout = req.Timeout
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
		httpReq.Body = io.NopCloser(strings.NewReader(req.Body))
		httpReq.ContentLength = int64(len(req.Body))
	}

	maxRedirects := req.MaxRedirects
	if maxRedirects == 0 {
		maxRedirects = DefaultMaxRedirects
	}

	var redirectHops []RedirectHop
	var redirectChain []string

	httpClient := &http.Client{
		Transport: c.transport,
		Timeout:   effectiveTimeout,
	}

	if req.ProxyURL != "" || len(req.NoProxy) > 0 {
		httpClient = c.buildClientWithProxy(req)
	}

	if maxRedirects < 0 {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if maxRedirects > 0 {
		redirectHops = make([]RedirectHop, 0, maxRedirects)
		redirectChain = make([]string, 0, maxRedirects)
		lastStatusCode := 0
		origTransport := httpClient.Transport
		httpClient.Transport = &statusCapturingTransport{
			RoundTripper: origTransport,
			onResponse: func(statusCode int) {
				lastStatusCode = statusCode
			},
		}
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 {
				prevReq := via[len(via)-1]
				statusCode := lastStatusCode
				if prevReq.Response != nil {
					statusCode = prevReq.Response.StatusCode
				}
				hop := RedirectHop{
					URL:        prevReq.URL.String(),
					StatusCode: statusCode,
				}
				redirectHops = append(redirectHops, hop)
				redirectChain = append(redirectChain, req.URL.String())
			}
			if len(via) >= maxRedirects {
				return ErrTooManyRedirects
			}
			return nil
		}
	}

	start := time.Now()
	httpResp, err := httpClient.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, ErrTooManyRedirects) && len(redirectHops) > 0 {
			lastHop := redirectHops[len(redirectHops)-1]
			headers := http.Header{}
			if httpResp != nil {
				headers = httpResp.Header
			}
			return Response{
				StatusCode: lastHop.StatusCode,
				Headers:    headers,
				Body:       []byte{},
				Duration:   duration,
				Size:       0,
				Redirects:  redirectHops,
			}, nil
		}
		return Response{}, wrapTimeoutError(err, effectiveTimeout)
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
		Redirects:  redirectHops,
	}, nil
}
