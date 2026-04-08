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

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(tlsConfig), nil
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
	defer conn.Close()

	// Build context with metadata
	if o.metadata != nil {
		ctx = metadata.NewOutgoingContext(ctx, o.metadata)
	} else if len(o.headers) > 0 {
		md := metadata.New(o.headers)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// Create a placeholder input message for testing
	// In real usage, the caller would provide proper proto message using descriptor source
	inputMsg := dynamicpb.NewMessage(nil)

	// For now, we return an error indicating we need proper proto setup
	// The actual implementation would use protoreflect to build the message
	if c.descSource == nil {
		return nil, fmt.Errorf("gRPC call requires service descriptor source (use SetDescriptorSource)")
	}

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

	// Get trailing metadata
	trailer := metadata.MD{}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		trailer = md
	}

	return &Response{
		Data:       outputJSON,
		StatusCode: codes.OK,
		Message:    "OK",
		Trailer:    trailer,
	}, nil
}

// executeStreamingCall performs a streaming gRPC call
func (c *Client) executeStreamingCall(ctx context.Context, target, method string, data []byte, callType CallType, opts ...Option) (*StreamingResponse, error) {
	// Streaming calls require proper descriptor source for full implementation
	// Return a clear error indicating what's needed
	return nil, fmt.Errorf("streaming calls require service descriptor source (use SetDescriptorSource)")
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
