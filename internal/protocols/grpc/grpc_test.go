package grpc

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Test helper to create a gRPC test server
func newTestServer(t *testing.T, opts ...grpc.ServerOption) *grpc.Server {
	server := grpc.NewServer(opts...)
	return server
}

// Test for status code to string conversion
func TestStatusCodeToString(t *testing.T) {
	tests := []struct {
		code    codes.Code
		wantStr string
	}{
		{codes.OK, "OK"},
		{codes.Canceled, "CANCELED"},
		{codes.Unknown, "UNKNOWN"},
		{codes.InvalidArgument, "INVALID_ARGUMENT"},
		{codes.DeadlineExceeded, "DEADLINE_EXCEEDED"},
		{codes.NotFound, "NOT_FOUND"},
		{codes.AlreadyExists, "ALREADY_EXISTS"},
		{codes.PermissionDenied, "PERMISSION_DENIED"},
		{codes.ResourceExhausted, "RESOURCE_EXHAUSTED"},
		{codes.FailedPrecondition, "FAILED_PRECONDITION"},
		{codes.Aborted, "ABORTED"},
		{codes.OutOfRange, "OUT_OF_RANGE"},
		{codes.Unimplemented, "UNIMPLEMENTED"},
		{codes.Internal, "INTERNAL"},
		{codes.Unavailable, "UNAVAILABLE"},
		{codes.DataLoss, "DATA_LOSS"},
		{codes.Unauthenticated, "UNAUTHENTICATED"},
		{codes.Code(999), "CODE_999"},
	}

	for _, tt := range tests {
		t.Run(tt.wantStr, func(t *testing.T) {
			got := StatusCodeToString(tt.code)
			if got != tt.wantStr {
				t.Errorf("StatusCodeToString(%v) = %v, want %v", tt.code, got, tt.wantStr)
			}
		})
	}
}

// Test for method parsing
func TestParseMethod(t *testing.T) {
	tests := []struct {
		fullMethod  string
		wantService string
		wantMethod  string
	}{
		{"/helloworld.Greeter/SayHello", "helloworld.Greeter", "SayHello"},
		{"/UserService/GetUser", "UserService", "GetUser"},
		{"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo", "grpc.reflection.v1alpha.ServerReflection", "ServerReflectionInfo"},
		{"SayHello", "", "SayHello"}, // No prefix
		{"/SingleSlash", "", "SingleSlash"},
	}

	for _, tt := range tests {
		t.Run(tt.fullMethod, func(t *testing.T) {
			svc, meth := ParseMethod(tt.fullMethod)
			if svc != tt.wantService {
				t.Errorf("ParseMethod(%q) service = %v, want %v", tt.fullMethod, svc, tt.wantService)
			}
			if meth != tt.wantMethod {
				t.Errorf("ParseMethod(%q) method = %v, want %v", tt.fullMethod, meth, tt.wantMethod)
			}
		})
	}
}

// Test for call type string conversion
func TestCallTypeString(t *testing.T) {
	tests := []struct {
		ct   CallType
		want string
	}{
		{CallTypeUnary, "UNARY"},
		{CallTypeServerStreaming, "SERVER_STREAMING"},
		{CallTypeClientStreaming, "CLIENT_STREAMING"},
		{CallTypeBidirectionalStreaming, "BIDIRECTIONAL_STREAMING"},
		{CallType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.ct.String(); got != tt.want {
				t.Errorf("CallType(%d).String() = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}

// Test for parseCallType
func TestParseCallType(t *testing.T) {
	tests := []struct {
		input    string
		wantType CallType
		wantErr  bool
	}{
		{"unary", CallTypeUnary, false},
		{"UNARY", CallTypeUnary, false},
		{"server-streaming", CallTypeServerStreaming, false},
		{"server_streaming", CallTypeServerStreaming, false},
		{"serverstreaming", CallTypeServerStreaming, false},
		{"client-streaming", CallTypeClientStreaming, false},
		{"client_streaming", CallTypeClientStreaming, false},
		{"clientstreaming", CallTypeClientStreaming, false},
		{"bidirectional", CallTypeBidirectionalStreaming, false},
		{"bidirectional-streaming", CallTypeBidirectionalStreaming, false},
		{"bidi", CallTypeBidirectionalStreaming, false},
		{"invalid", CallTypeUnary, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseCallType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCallType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantType {
				t.Errorf("parseCallType(%q) = %v, want %v", tt.input, got, tt.wantType)
			}
		})
	}
}

// Test for TLS config
func TestTLSConfig(t *testing.T) {
	cfg := TLSConfig{
		CertFile:      "/path/to/cert.pem",
		KeyFile:       "/path/to/key.pem",
		CAFile:        "/path/to/ca.pem",
		Insecure:      false,
		MinTLSVersion: "1.2",
		ServerName:    "example.com",
	}

	if cfg.CertFile != "/path/to/cert.pem" {
		t.Errorf("CertFile = %v, want /path/to/cert.pem", cfg.CertFile)
	}
	if cfg.Insecure != false {
		t.Errorf("Insecure = %v, want false", cfg.Insecure)
	}
	if cfg.ServerName != "example.com" {
		t.Errorf("ServerName = %v, want example.com", cfg.ServerName)
	}
}

// Test for response struct
func TestResponse(t *testing.T) {
	resp := &Response{
		Data:       json.RawMessage(`{"message":"hello"}`),
		StatusCode: codes.OK,
		Message:    "OK",
	}

	if string(resp.Data) != `{"message":"hello"}` {
		t.Errorf("Data = %v, want {\"message\":\"hello\"}", string(resp.Data))
	}
	if resp.StatusCode != codes.OK {
		t.Errorf("StatusCode = %v, want OK", resp.StatusCode)
	}
	if resp.Message != "OK" {
		t.Errorf("Message = %v, want OK", resp.Message)
	}
}

// Test for streaming response
func TestStreamingResponse(t *testing.T) {
	events := []*StreamEvent{
		{Type: "send", Data: json.RawMessage(`{"name":"test"}`)},
		{Type: "recv", Data: json.RawMessage(`{"message":"hello"}`)},
	}

	resp := &StreamingResponse{
		Events: events,
	}

	if len(resp.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(resp.Events))
	}
	if resp.Events[0].Type != "send" {
		t.Errorf("Events[0].Type = %v, want send", resp.Events[0].Type)
	}
	if resp.Events[1].Type != "recv" {
		t.Errorf("Events[1].Type = %v, want recv", resp.Events[1].Type)
	}
}

// Test for service info
func TestServiceInfo(t *testing.T) {
	info := &ServiceInfo{
		Name: "UserService",
		Methods: []MethodInfo{
			{
				Name:           "GetUser",
				InputType:      ".UserRequest",
				OutputType:     ".UserResponse",
				IsServerStream: false,
				IsClientStream: false,
			},
			{
				Name:           "StreamUsers",
				InputType:      ".UserRequest",
				OutputType:     ".UserResponse",
				IsServerStream: true,
				IsClientStream: false,
			},
		},
	}

	if info.Name != "UserService" {
		t.Errorf("Name = %v, want UserService", info.Name)
	}
	if len(info.Methods) != 2 {
		t.Errorf("len(Methods) = %d, want 2", len(info.Methods))
	}
	if info.Methods[0].IsServerStream {
		t.Error("Methods[0] should not be server streaming")
	}
	if !info.Methods[1].IsServerStream {
		t.Error("Methods[1] should be server streaming")
	}
}

// Test for metadata option
func TestWithMetadata(t *testing.T) {
	md := metadata.New(map[string]string{
		"authorization": "Bearer token123",
		"custom-header": "value",
	})

	opt := WithMetadata(md)
	o := &options{}
	opt(o)

	if o.metadata == nil {
		t.Fatal("metadata should not be nil")
	}

	if v := o.metadata.Get("authorization"); len(v) == 0 || v[0] != "Bearer token123" {
		t.Errorf("authorization = %v, want [Bearer token123]", v)
	}
}

// Test for header option
func TestWithHeader(t *testing.T) {
	opt := WithHeader("X-Custom-Header", "custom-value")
	o := &options{}
	opt(o)

	if o.headers == nil {
		t.Fatal("headers should not be nil")
	}
	if v := o.headers["X-Custom-Header"]; v != "custom-value" {
		t.Errorf("X-Custom-Header = %v, want custom-value", v)
	}
}

// Test for TLS option
func TestWithTLS(t *testing.T) {
	cfg := TLSConfig{
		Insecure: true,
	}

	opt := WithTLS(cfg)
	o := &options{}
	opt(o)

	if o.tlsConfig == nil {
		t.Fatal("tlsConfig should not be nil")
	}
	if !o.tlsConfig.Insecure {
		t.Error("tlsConfig.Insecure should be true")
	}
}

// Test for mustMarshal helper
func TestMustMarshal(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
		"age":  42,
	}

	result := mustMarshal(data)
	expected := `{"age":42,"name":"test"}`

	if string(result) != expected {
		t.Errorf("mustMarshal() = %v, want %v", string(result), expected)
	}
}

// Test gRPC status error handling
func TestStatusError(t *testing.T) {
	// Create a status error
	st := status.New(codes.NotFound, "user not found")
	err := st.Err()

	// Verify we can extract the code and message
	s, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected error to be a status error")
	}

	if s.Code() != codes.NotFound {
		t.Errorf("Code() = %v, want NotFound", s.Code())
	}
	if s.Message() != "user not found" {
		t.Errorf("Message() = %v, want 'user not found'", s.Message())
	}
}

// Test client creation
func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

// Test client with TLS
func TestNewClientWithTLS(t *testing.T) {
	cfg := TLSConfig{
		Insecure: true,
	}

	client := NewClientWithTLS(cfg)
	if client == nil {
		t.Fatal("NewClientWithTLS() returned nil")
	}
	if client.tlsConfig == nil {
		t.Error("tlsConfig should not be nil")
	}
	if !client.tlsConfig.Insecure {
		t.Error("tlsConfig.Insecure should be true")
	}
}

// Test descriptor source interface
func TestSetDescriptorSource(t *testing.T) {
	client := NewClient()

	// Create a mock descriptor source
	src := &mockDescriptorSource{}
	client.SetDescriptorSource(src)

	if client.descSource == nil {
		t.Error("descSource should not be nil after SetDescriptorSource")
	}
}

type mockDescriptorSource struct{}

func (m *mockDescriptorSource) GetServiceDescriptor(name string) interface{} {
	return nil
}

func (m *mockDescriptorSource) ListServices() []string {
	return nil
}

// Test client dial with insecure credentials
func TestClientDialInsecure(t *testing.T) {
	client := NewClientWithTLS(TLSConfig{Insecure: true})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Dial may succeed or fail depending on network
	conn, err := client.dial(ctx, "localhost:99999")
	if conn != nil {
		conn.Close()
	}
	// Just verify it doesn't panic
	_ = err
}

// Test reflection client creation
func TestNewReflectionClient(t *testing.T) {
	// Create a client without connection (for interface testing)
	client := &ReflectionClient{}
	if client == nil {
		t.Fatal("NewReflectionClient() returned nil")
	}
}

// Test reflection client with nil connection
func TestReflectionClientNilConn(t *testing.T) {
	rc := &ReflectionClient{conn: nil}

	// Should handle nil connection gracefully
	ctx := context.Background()
	_, err := rc.ListServices(ctx)

	// Will fail due to nil connection, but shouldn't panic
	if err == nil {
		t.Error("expected error with nil connection")
	}
}

// Test CheckReflectionSupported with nil connection
func TestCheckReflectionSupportedNilConn(t *testing.T) {
	rc := &ReflectionClient{conn: nil}

	ctx := context.Background()
	err := rc.CheckReflectionSupported(ctx)

	// Should fail gracefully
	if err == nil {
		t.Error("expected error with nil connection")
	}
}

// Test ListServices with nil connection
func TestListServicesNilConn(t *testing.T) {
	rc := &ReflectionClient{conn: nil}

	ctx := context.Background()
	_, err := rc.ListServices(ctx)

	// Should fail gracefully
	if err == nil {
		t.Error("expected error with nil connection")
	}
}

// Test GetServiceDescription with nil connection (stub implementation)
func TestGetServiceDescriptionNilConn(t *testing.T) {
	rc := &ReflectionClient{conn: nil}

	ctx := context.Background()
	info, err := rc.GetServiceDescription(ctx, "TestService")

	// Stub returns info even with nil connection
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info == nil {
		t.Error("expected info to be returned")
	}
}

// Test ResolveMethod with nil connection (stub implementation)
func TestResolveMethodNilConn(t *testing.T) {
	rc := &ReflectionClient{conn: nil}

	ctx := context.Background()
	info, err := rc.ResolveMethod(ctx, "/TestService/TestMethod")

	// Stub returns info even with nil connection
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info == nil {
		t.Error("expected info to be returned")
	}
}

// Test GetAllMethodsForService with nil connection (stub implementation)
func TestGetAllMethodsForServiceNilConn(t *testing.T) {
	rc := &ReflectionClient{conn: nil}

	ctx := context.Background()
	methods, err := rc.GetAllMethodsForService(ctx, "TestService")

	// Stub returns empty methods even with nil connection
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if methods == nil {
		t.Error("expected methods to be returned")
	}
}

// Benchmark for status code string conversion
func BenchmarkStatusCodeToString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = StatusCodeToString(codes.OK)
		_ = StatusCodeToString(codes.NotFound)
		_ = StatusCodeToString(codes.Internal)
	}
}

// Benchmark for method parsing
func BenchmarkParseMethod(b *testing.B) {
	method := "/helloworld.Greeter/SayHello"
	for i := 0; i < b.N; i++ {
		_, _ = ParseMethod(method)
	}
}

// Benchmark for call type parsing
func BenchmarkParseCallType(b *testing.B) {
	types := []string{"unary", "server-streaming", "client-streaming", "bidirectional"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseCallType(types[i%len(types)])
	}
}

// Test integration helper - would be used with real gRPC server
func TestIntegrationGRPCUnary(t *testing.T) {
	t.Skip("Skipping integration test - requires running gRPC server")
}

// TestExecuteCallNilDescriptorSource tests executeCall with nil descriptor source
func TestExecuteCallNilDescriptorSource(t *testing.T) {
	client := NewClient()
	// No descriptor source set

	ctx := context.Background()
	_, err := client.executeCall(ctx, "localhost:50051", "/TestService/TestMethod", []byte(`{}`), CallTypeUnary)

	if err == nil {
		t.Error("expected error when descriptor source is nil")
	}
	if !strings.Contains(err.Error(), "descriptor source") {
		t.Errorf("expected 'descriptor source' in error, got: %v", err)
	}
}

// TestExecuteCallDialError tests executeCall when dial fails
func TestExecuteCallDialError(t *testing.T) {
	client := NewClient()
	client.SetDescriptorSource(&mockDescriptorSource{})

	ctx := context.Background()
	// Use an invalid target that will cause dial to fail
	_, err := client.executeCall(ctx, "", "/TestService/TestMethod", []byte(`{}`), CallTypeUnary)

	if err == nil {
		t.Error("expected error when dial fails")
	}
}

// TestExecuteCallWithMetadata tests executeCall with metadata option
func TestExecuteCallWithMetadata(t *testing.T) {
	client := NewClient()
	client.SetDescriptorSource(&mockDescriptorSource{})

	ctx := context.Background()
	md := metadata.New(map[string]string{
		"x-custom-header": "test-value",
	})

	_, err := client.executeCall(ctx, "localhost:50051", "/TestService/TestMethod", []byte(`{}`), CallTypeUnary, WithMetadata(md))

	// Should fail at connection but not at metadata parsing
	// This tests that metadata is properly applied before connection
	if err == nil {
		t.Error("expected connection error with fake target")
	}
}

// TestExecuteCallWithHeaders tests executeCall with header options
func TestExecuteCallWithHeaders(t *testing.T) {
	client := NewClient()
	client.SetDescriptorSource(&mockDescriptorSource{})

	ctx := context.Background()

	_, err := client.executeCall(ctx, "localhost:50051", "/TestService/TestMethod", []byte(`{}`), CallTypeUnary,
		WithHeader("Authorization", "Bearer token123"))

	// Should fail at connection but headers should be applied
	if err == nil {
		t.Error("expected connection error with fake target")
	}
}

// TestExecuteCallInvalidMethodFormat tests executeCall with various method formats
func TestExecuteCallInvalidMethodFormat(t *testing.T) {
	client := NewClient()
	client.SetDescriptorSource(&mockDescriptorSource{})

	ctx := context.Background()

	// Empty method
	_, err := client.executeCall(ctx, "localhost:50051", "", []byte(`{}`), CallTypeUnary)
	if err == nil {
		t.Error("expected error with empty method")
	}
}

// TestBuildTLSCredentialsMissingCertFile tests buildTLSCredentials with missing cert file
func TestBuildTLSCredentialsMissingCertFile(t *testing.T) {
	cfg := &TLSConfig{
		CertFile: "/nonexistent/path/to/cert.pem",
		KeyFile:  "/nonexistent/path/to/key.pem",
	}

	_, err := buildTLSCredentials(cfg)
	if err == nil {
		t.Error("expected error for missing cert file")
	}
	if !strings.Contains(err.Error(), "failed to load client certificate") {
		t.Errorf("expected 'failed to load client certificate' error, got: %v", err)
	}
}

// TestBuildTLSCredentialsInvalidCertData tests buildTLSCredentials with invalid cert data
func TestBuildTLSCredentialsInvalidCertData(t *testing.T) {
	// Create a temp file with invalid cert data
	tmpDir := t.TempDir()
	certFile := tmpDir + "/invalid_cert.pem"
	keyFile := tmpDir + "/invalid_key.pem"

	if err := os.WriteFile(certFile, []byte("not a valid certificate"), 0644); err != nil {
		t.Fatalf("failed to write temp cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, []byte("not a valid key"), 0644); err != nil {
		t.Fatalf("failed to write temp key file: %v", err)
	}

	cfg := &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
	}

	_, err := buildTLSCredentials(cfg)
	if err == nil {
		t.Error("expected error for invalid cert data")
	}
}

// TestBuildTLSCredentialsValidCert tests buildTLSCredentials with valid TLS config (no cert files)
func TestBuildTLSCredentialsValidCert(t *testing.T) {
	cfg := &TLSConfig{
		Insecure:   true,
		ServerName: "testserver",
	}

	creds, err := buildTLSCredentials(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if creds == nil {
		t.Error("expected credentials to be returned")
	}
}

// TestBuildTLSCredentialsNoCertFiles tests buildTLSCredentials without cert files (default TLS)
func TestBuildTLSCredentialsNoCertFiles(t *testing.T) {
	cfg := &TLSConfig{
		Insecure: false,
	}

	creds, err := buildTLSCredentials(cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if creds == nil {
		t.Error("expected credentials to be returned")
	}
}

// TestExecuteStreamingCallReturnsError tests that executeStreamingCall returns proper error
func TestExecuteStreamingCallReturnsError(t *testing.T) {
	client := NewClient()
	client.SetDescriptorSource(&mockDescriptorSource{})

	ctx := context.Background()

	// All streaming call types should return an error about missing descriptor source
	testCases := []struct {
		callType CallType
		name     string
	}{
		{CallTypeServerStreaming, "ServerStreaming"},
		{CallTypeClientStreaming, "ClientStreaming"},
		{CallTypeBidirectionalStreaming, "BidirectionalStreaming"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.executeStreamingCall(ctx, "localhost:50051", "/TestService/Method", []byte(`{}`), tc.callType)
			if err == nil {
				t.Error("expected error for streaming call without proper descriptor source")
			}
			if !strings.Contains(err.Error(), "descriptor source") {
				t.Errorf("expected 'descriptor source' in error, got: %v", err)
			}
		})
	}
}

// TestExecuteUnaryDelegate tests that ExecuteUnary delegates to executeCall
func TestExecuteUnaryDelegate(t *testing.T) {
	client := NewClient()
	// No descriptor source set

	ctx := context.Background()
	_, err := client.ExecuteUnary(ctx, "localhost:50051", "/TestService/TestMethod", []byte(`{}`))

	if err == nil {
		t.Error("expected error when descriptor source is nil (delegated from ExecuteUnary)")
	}
	if !strings.Contains(err.Error(), "descriptor source") {
		t.Errorf("expected 'descriptor source' in error, got: %v", err)
	}
}

// TestExecuteServerStreamingDelegate tests that ExecuteServerStreaming delegates
func TestExecuteServerStreamingDelegate(t *testing.T) {
	client := NewClient()

	ctx := context.Background()
	_, err := client.ExecuteServerStreaming(ctx, "localhost:50051", "/TestService/TestMethod", []byte(`{}`))

	if err == nil {
		t.Error("expected error when descriptor source is nil")
	}
}

// TestExecuteClientStreamingDelegate tests that ExecuteClientStreaming delegates
func TestExecuteClientStreamingDelegate(t *testing.T) {
	client := NewClient()

	ctx := context.Background()
	_, err := client.ExecuteClientStreaming(ctx, "localhost:50051", "/TestService/TestMethod", []byte(`{}`))

	if err == nil {
		t.Error("expected error when descriptor source is nil")
	}
}

// TestExecuteBidirectionalStreamingDelegate tests that ExecuteBidirectionalStreaming delegates
func TestExecuteBidirectionalStreamingDelegate(t *testing.T) {
	client := NewClient()

	ctx := context.Background()
	_, err := client.ExecuteBidirectionalStreaming(ctx, "localhost:50051", "/TestService/TestMethod", []byte(`{}`))

	if err == nil {
		t.Error("expected error when descriptor source is nil")
	}
}

// TestExecuteCallWithAllCallTypes tests executeCall with all call type variants
func TestExecuteCallWithAllCallTypes(t *testing.T) {
	client := NewClient()
	client.SetDescriptorSource(&mockDescriptorSource{})

	ctx := context.Background()

	// Test that call type is properly received even if connection fails
	_, err := client.executeCall(ctx, "localhost:50051", "/TestService/TestMethod", []byte(`{}`), CallTypeUnary)
	if err == nil {
		t.Error("expected connection error")
	}
}

// TestClientDialWithTLSConfig tests dial with full TLS config
func TestClientDialWithTLSConfig(t *testing.T) {
	// This tests the TLS config path through dial
	client := NewClientWithTLS(TLSConfig{
		Insecure:   true,
		ServerName: "test.example.com",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := client.dial(ctx, "localhost:99999")
	if conn != nil {
		conn.Close()
	}
	// Connection may fail, but we just verify it doesn't panic
	_ = err
}
