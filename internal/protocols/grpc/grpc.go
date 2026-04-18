package grpc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// CallType represents the type of gRPC call
type CallType int

const (
	CallTypeUnary CallType = iota
	CallTypeServerStreaming
	CallTypeClientStreaming
	CallTypeBidirectionalStreaming
)

// String returns a human-readable name for the call type
func (ct CallType) String() string {
	switch ct {
	case CallTypeUnary:
		return "UNARY"
	case CallTypeServerStreaming:
		return "SERVER_STREAMING"
	case CallTypeClientStreaming:
		return "CLIENT_STREAMING"
	case CallTypeBidirectionalStreaming:
		return "BIDIRECTIONAL_STREAMING"
	default:
		return "UNKNOWN"
	}
}

// TLSConfig holds TLS configuration for gRPC connections
type TLSConfig struct {
	CertFile      string
	KeyFile       string
	CAFile        string
	Insecure      bool
	MinTLSVersion string
	ServerName    string // For SNI
}

// Option is a functional option for configuring the Client
type Option func(*options)

type options struct {
	headers   map[string]string
	tlsConfig *TLSConfig
	metadata  metadata.MD
}

// WithHeader adds a header to the gRPC request
func WithHeader(key, value string) Option {
	return func(o *options) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// WithMetadata adds metadata to the gRPC request
func WithMetadata(md metadata.MD) Option {
	return func(o *options) {
		o.metadata = md
	}
}

// WithTLS sets TLS configuration
func WithTLS(cfg TLSConfig) Option {
	return func(o *options) {
		o.tlsConfig = &cfg
	}
}

// Response represents a gRPC response
type Response struct {
	Data       json.RawMessage
	StatusCode codes.Code
	Message    string
	Headers    metadata.MD
	Trailer    metadata.MD
}

// StreamEvent represents an event in a streaming RPC
type StreamEvent struct {
	Type      string // "recv" or "send"
	Data      json.RawMessage
	Timestamp string
}

// Client wraps a gRPC client connection
type Client struct {
	conn       *grpc.ClientConn
	tlsConfig  *TLSConfig
	descSource ServiceDescriptorSource
}

// ServiceDescriptorSource is an interface for getting service descriptors
type ServiceDescriptorSource interface {
	GetServiceDescriptor(name string) interface{}
	ListServices() []string
}

// NewClient creates a new gRPC client
func NewClient() *Client {
	return &Client{}
}

// NewClientWithTLS creates a new gRPC client with TLS configuration
func NewClientWithTLS(cfg TLSConfig) *Client {
	return &Client{
		tlsConfig: &cfg,
	}
}

// SetDescriptorSource sets the service descriptor source for dynamic invocation
func (c *Client) SetDescriptorSource(src ServiceDescriptorSource) {
	c.descSource = src
}

// dial establishes a gRPC connection
func (c *Client) dial(ctx context.Context, target string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	// Configure transport credentials
	if c.tlsConfig != nil && c.tlsConfig.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else if c.tlsConfig != nil {
		// Build TLS config
		creds, err := buildTLSCredentials(c.tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		// Default to insecure
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return grpc.NewClient(target, opts...)
}

// buildTLSCredentials builds gRPC TLS credentials from config
func buildTLSCredentials(cfg *TLSConfig) (credentials.TransportCredentials, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
		ServerName:         cfg.ServerName,
	}

	// Apply MinTLSVersion if specified
	if cfg.MinTLSVersion != "" {
		version, ok := tlsVersionToValue(cfg.MinTLSVersion)
		if !ok {
			return nil, fmt.Errorf("invalid MinTLSVersion: %s", cfg.MinTLSVersion)
		}
		tlsConfig.MinVersion = version
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if cfg.CAFile != "" {
		// Load CA certificate for server verification
		// This would require reading the CA file and setting RootCAs
	}

	return credentials.NewTLS(tlsConfig), nil
}

// tlsVersionToValue converts a TLS version string to its numeric value
func tlsVersionToValue(version string) (uint16, bool) {
	switch version {
	case "1.0":
		return tls.VersionTLS10, true
	case "1.1":
		return tls.VersionTLS11, true
	case "1.2":
		return tls.VersionTLS12, true
	case "1.3":
		return tls.VersionTLS13, true
	default:
		return 0, false
	}
}

// ExecuteUnary performs a unary gRPC call
func (c *Client) ExecuteUnary(ctx context.Context, target, method string, data []byte, opts ...Option) (*Response, error) {
	return c.executeCall(ctx, target, method, data, CallTypeUnary, opts...)
}

// ExecuteServerStreaming performs a server streaming gRPC call
func (c *Client) ExecuteServerStreaming(ctx context.Context, target, method string, data []byte, opts ...Option) (*StreamingResponse, error) {
	return c.executeStreamingCall(ctx, target, method, data, CallTypeServerStreaming, opts...)
}

// ExecuteClientStreaming performs a client streaming gRPC call
func (c *Client) ExecuteClientStreaming(ctx context.Context, target, method string, data []byte, opts ...Option) (*StreamingResponse, error) {
	return c.executeStreamingCall(ctx, target, method, data, CallTypeClientStreaming, opts...)
}

// ExecuteBidirectionalStreaming performs a bidirectional streaming gRPC call
func (c *Client) ExecuteBidirectionalStreaming(ctx context.Context, target, method string, data []byte, opts ...Option) (*StreamingResponse, error) {
	return c.executeStreamingCall(ctx, target, method, data, CallTypeBidirectionalStreaming, opts...)
}

// StreamingResponse represents a streaming gRPC response
type StreamingResponse struct {
	Events []*StreamEvent
}

// executeCall performs the actual gRPC call
func (c *Client) executeCall(ctx context.Context, target, method string, data []byte, callType CallType, opts ...Option) (*Response, error) {
	// Apply options
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	conn, err := c.dial(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	// Only defer close if dial succeeded - conn is valid only if err is nil
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	// Build context with metadata
	if o.metadata != nil {
		ctx = metadata.NewOutgoingContext(ctx, o.metadata)
	} else if len(o.headers) > 0 {
		md := metadata.New(o.headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// For now, we return an error indicating we need proper proto setup
	// The actual implementation would use protoreflect to build the message
	if c.descSource == nil {
		return nil, fmt.Errorf("gRPC call requires service descriptor source (use SetDescriptorSource)")
	}

	// Create input message from descriptor source
	// The descSource should provide the method descriptor with proper input/output types
	serviceName, _ := ParseMethod(method)
	desc := c.descSource.GetServiceDescriptor(serviceName)
	if desc == nil {
		return nil, fmt.Errorf("service descriptor source not found for %s (ensure SetDescriptorSource is properly configured)", serviceName)
	}

	// Use dynamicpb for message creation if we have a file descriptor
	inputMsg := dynamicpb.NewMessage(nil)

	// Use conn.Invoke for unary calls
	var outputMsg proto.Message
	err = conn.Invoke(ctx, method, inputMsg, &outputMsg, grpc.EmptyCallOption{})
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return &Response{
				StatusCode: codes.Unknown,
				Message:    err.Error(),
			}, err
		}
		return &Response{
			StatusCode: st.Code(),
			Message:    st.Message(),
		}, err
	}

	// Marshal output to JSON
	outputJSON, err := json.Marshal(outputMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &Response{
		Data:       outputJSON,
		StatusCode: codes.OK,
		Message:    "OK",
		Trailer:    metadata.MD{}, // Trailers only available via proper call options
	}, nil
}

// executeStreamingCall performs a streaming gRPC call.
// NOTE: Full streaming RPC support requires a service descriptor source that provides
// proto file descriptors (FileDescriptor messages). Without descriptors, streaming
// calls cannot properly encode/decode messages. Set a descriptor source via
// SetDescriptorSource() before using streaming calls.
//
// Supported streaming types:
// - CallTypeServerStreaming: Server sends multiple responses to a single client request
// - CallTypeClientStreaming: Client sends multiple requests, server sends single response
// - CallTypeBidirectionalStreaming: Both client and server send multiple messages
func (c *Client) executeStreamingCall(ctx context.Context, target, method string, data []byte, callType CallType, opts ...Option) (*StreamingResponse, error) {
	if c.descSource == nil {
		return nil, fmt.Errorf("streaming calls require service descriptor source (use SetDescriptorSource)")
	}

	// Apply options
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	conn, err := c.dial(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	// Build context with metadata
	if o.metadata != nil {
		ctx = metadata.NewOutgoingContext(ctx, o.metadata)
	} else if len(o.headers) > 0 {
		md := metadata.New(o.headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// Parse method to get service and method names
	serviceName, methodName := ParseMethod(method)
	desc := c.descSource.GetServiceDescriptor(serviceName)
	if desc == nil {
		return nil, fmt.Errorf("service descriptor source not found for %s", serviceName)
	}

	// Get the method descriptor from the service
	// The actual implementation would need to look up the method from the descriptor
	// and create the appropriate stream using grpc.NewClientStream or similar

	// For now, return a clear error that this needs proper descriptor-based implementation
	return nil, fmt.Errorf("streaming RPC for %s/%s requires proto descriptor with method definitions; current descriptor source does not provide sufficient information for stream setup", serviceName, methodName)
}

// StatusCodeToString converts a gRPC status code to human-readable string
func StatusCodeToString(code codes.Code) string {
	switch code {
	case codes.OK:
		return "OK"
	case codes.Canceled:
		return "CANCELED"
	case codes.Unknown:
		return "UNKNOWN"
	case codes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case codes.DeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case codes.NotFound:
		return "NOT_FOUND"
	case codes.AlreadyExists:
		return "ALREADY_EXISTS"
	case codes.PermissionDenied:
		return "PERMISSION_DENIED"
	case codes.ResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case codes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case codes.Aborted:
		return "ABORTED"
	case codes.OutOfRange:
		return "OUT_OF_RANGE"
	case codes.Unimplemented:
		return "UNIMPLEMENTED"
	case codes.Internal:
		return "INTERNAL"
	case codes.Unavailable:
		return "UNAVAILABLE"
	case codes.DataLoss:
		return "DATA_LOSS"
	case codes.Unauthenticated:
		return "UNAUTHENTICATED"
	default:
		return fmt.Sprintf("CODE_%d", int(code))
	}
}

// ParseMethod parses a fully-qualified method name into service and method names
func ParseMethod(fullMethod string) (service, method string) {
	// Expected format: /service.name/MethodName
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.SplitN(fullMethod, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fullMethod
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
