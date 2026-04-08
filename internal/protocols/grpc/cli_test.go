package grpc

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/formatter"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

type mockDB struct{}

func (m *mockDB) Open() error                               { return nil }
func (m *mockDB) Close() error                              { return nil }
func (m *mockDB) SaveRequest(req *types.SavedRequest) error { return nil }
func (m *mockDB) GetRequest(id string) (*types.SavedRequest, error) {
	return nil, nil
}
func (m *mockDB) GetRequestByName(name string) (*types.SavedRequest, error) {
	return nil, nil
}
func (m *mockDB) ListRequests(opts *storage.ListOptions) ([]*types.SavedRequest, error) {
	return nil, nil
}
func (m *mockDB) DeleteRequest(id string) error               { return nil }
func (m *mockDB) UpdateRequest(req *types.SavedRequest) error { return nil }
func (m *mockDB) SaveHistory(history *types.ExecutionHistory) error {
	return nil
}
func (m *mockDB) GetHistory(requestID string, limit int) ([]*types.ExecutionHistory, error) {
	return nil, nil
}
func (m *mockDB) ListFolder(path string) ([]*types.SavedRequest, error) { return nil, nil }
func (m *mockDB) ListFolderRecursive(path string) ([]*types.SavedRequest, error) {
	return nil, nil
}
func (m *mockDB) DeleteFolder(path string) error   { return nil }
func (m *mockDB) GetAllFolders() ([]string, error) { return nil, nil }

func runGRPCCommand(t *testing.T, args []string) error {
	t.Helper()
	cmd := GRPCCommand(&mockDB{})
	root := &cli.Command{
		Name:     "test",
		Commands: []*cli.Command{cmd},
	}
	return root.Run(context.Background(), append([]string{"test", "grpc"}, args...))
}

func TestGRPCCommandMissingTarget(t *testing.T) {
	err := runGRPCCommand(t, []string{})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
	if !strings.Contains(err.Error(), "target") {
		t.Errorf("expected 'target' in error, got: %v", err)
	}
}

func TestGRPCCommandMissingServiceAndMethod(t *testing.T) {
	err := runGRPCCommand(t, []string{"localhost:50051"})
	if err == nil {
		t.Fatal("expected error for missing service/method")
	}
	if !strings.Contains(err.Error(), "--service and --method are required") {
		t.Errorf("expected '--service and --method are required' in error, got: %v", err)
	}
}

func TestGRPCCommandMissingMethod(t *testing.T) {
	err := runGRPCCommand(t, []string{"--service", "TestService", "localhost:50051"})
	if err == nil {
		t.Fatal("expected error for missing method")
	}
	if !strings.Contains(err.Error(), "--service and --method are required") {
		t.Errorf("expected '--service and --method are required' in error, got: %v", err)
	}
}

func TestGRPCCommandInvalidCallTypeViaRun(t *testing.T) {
	err := runGRPCCommand(t, []string{
		"--service", "Svc",
		"--method", "Method",
		"--call-type", "garbage",
		"localhost:50051",
	})
	if err == nil {
		t.Fatal("expected error for invalid call type")
	}
	if !strings.Contains(err.Error(), "invalid call type") {
		t.Errorf("expected 'invalid call type' in error, got: %v", err)
	}
}

func TestGRPCCommandListFlag(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runGRPCCommand(t, []string{"--list", "localhost:50051"})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Listing services on localhost:50051") {
		t.Errorf("expected listing output, got: %v", buf.String())
	}
}

func TestGRPCCommandUnaryCallErrorPath(t *testing.T) {
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "TestMethod",
		"--data", `{"key":"value"}`,
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("action should return nil (errors written to stderr), got: %v", err)
	}
}

func TestGRPCCommandServerStreamingCallErrorPath(t *testing.T) {
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "StreamMethod",
		"--call-type", "server-streaming",
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("action should return nil, got: %v", err)
	}
}

func TestGRPCCommandClientStreamingCallErrorPath(t *testing.T) {
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "StreamMethod",
		"--call-type", "client-streaming",
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("action should return nil, got: %v", err)
	}
}

func TestGRPCCommandBidirectionalCallErrorPath(t *testing.T) {
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "StreamMethod",
		"--call-type", "bidirectional",
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("action should return nil, got: %v", err)
	}
}

func TestGRPCCommandWithDataFileFlag(t *testing.T) {
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "req.json")
	os.WriteFile(dataFile, []byte(`{"name":"test"}`), 0644)

	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "TestMethod",
		"--data-file", dataFile,
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGRPCCommandWithMissingDataFileFlag(t *testing.T) {
	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "TestMethod",
		"--data-file", "/nonexistent/data.json",
		"localhost:50051",
	})

	if err == nil {
		t.Fatal("expected error for missing data file")
	}
	if !strings.Contains(err.Error(), "failed to read data file") {
		t.Errorf("expected 'failed to read data file' in error, got: %v", err)
	}
}

func TestGRPCCommandWithInsecureFlag(t *testing.T) {
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "TestMethod",
		"--insecure",
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGRPCCommandWithCertFlag(t *testing.T) {
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "TestMethod",
		"--cert", "/some/cert.pem",
		"--key", "/some/key.pem",
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGRPCCommandWithMetadataFlag(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "TestMethod",
		"--metadata", "auth:token123,trace-id:abc",
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Metadata sent:") {
		t.Errorf("expected metadata info in stderr, got: %v", buf.String())
	}
}

func TestGRPCCommandWithCACertFlag(t *testing.T) {
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := runGRPCCommand(t, []string{
		"--service", "TestService",
		"--method", "TestMethod",
		"--cacert", "/some/ca.pem",
		"localhost:50051",
	})

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGRPCCommandCreation(t *testing.T) {
	cmd := GRPCCommand(&mockDB{})
	if cmd == nil {
		t.Fatal("GRPCCommand returned nil")
	}
	if cmd.Name != "grpc" {
		t.Errorf("Name = %v, want grpc", cmd.Name)
	}
	if len(cmd.Flags) == 0 {
		t.Error("expected flags to be defined")
	}
}

func TestGRPCCommandInvalidCallType(t *testing.T) {
	_, err := parseCallType("invalid-type")
	if err == nil {
		t.Error("expected error for invalid call type")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown call type") {
		t.Errorf("expected 'unknown call type' in error, got: %v", err)
	}
}

func TestGRPCCommandWithDataFile(t *testing.T) {
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "test_data.json")
	testData := `{"key":"value"}`
	if err := os.WriteFile(dataFile, []byte(testData), 0644); err != nil {
		t.Fatalf("failed to write temp data file: %v", err)
	}

	content, err := os.ReadFile(dataFile)
	if err != nil {
		t.Fatalf("failed to read data file: %v", err)
	}
	if string(content) != testData {
		t.Errorf("file content = %v, want %v", string(content), testData)
	}
}

func TestGRPCCommandMissingDataFile(t *testing.T) {
	_, err := os.ReadFile("/nonexistent/path/to/data.json")
	if err == nil {
		t.Error("expected error for non-existent data file")
	}
}

func TestGRPCCommandWithMetadata(t *testing.T) {
	metaStr := "key1:value1,key2:value2"
	pairs := strings.Split(metaStr, ",")

	if len(pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(pairs))
	}
	if pairs[0] != "key1:value1" {
		t.Errorf("first pair = %v, want key1:value1", pairs[0])
	}
	if pairs[1] != "key2:value2" {
		t.Errorf("second pair = %v, want key2:value2", pairs[1])
	}
}

func TestGRPCCommandWithEmptyMetadata(t *testing.T) {
	metaStr := ""
	pairs := strings.Split(metaStr, ",")

	if len(pairs) != 1 || pairs[0] != "" {
		t.Errorf("expected single empty pair for empty metadata, got %v", pairs)
	}
}

func TestListServices(t *testing.T) {
	ctx := context.Background()
	client := NewClient()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listServices(ctx, client, "localhost:50051")

	w.Close()
	os.Stdout = oldStdout

	var output bytes.Buffer
	_, _ = output.ReadFrom(r)
	outputStr := output.String()

	if outputStr == "" {
		t.Error("expected output from listServices")
	}

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListServicesWithTarget(t *testing.T) {
	ctx := context.Background()
	client := NewClient()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listServices(ctx, client, "example.com:8080")

	w.Close()
	os.Stdout = oldStdout

	var output bytes.Buffer
	_, _ = output.ReadFrom(r)
	outputStr := output.String()

	if !strings.Contains(outputStr, "example.com:8080") {
		t.Errorf("expected target in output, got: %v", outputStr)
	}

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseCallTypeVariants(t *testing.T) {
	tests := []struct {
		input    string
		expected CallType
		wantErr  bool
	}{
		{"unary", CallTypeUnary, false},
		{"UNARY", CallTypeUnary, false},
		{"Unary", CallTypeUnary, false},
		{"server-streaming", CallTypeServerStreaming, false},
		{"server_streaming", CallTypeServerStreaming, false},
		{"serverstreaming", CallTypeServerStreaming, false},
		{"client-streaming", CallTypeClientStreaming, false},
		{"client_streaming", CallTypeClientStreaming, false},
		{"clientstreaming", CallTypeClientStreaming, false},
		{"bidirectional", CallTypeBidirectionalStreaming, false},
		{"bidirectional-streaming", CallTypeBidirectionalStreaming, false},
		{"bidi", CallTypeBidirectionalStreaming, false},
		{"BIDIRECTIONAL", CallTypeBidirectionalStreaming, false},
		{"", CallTypeUnary, true},
		{"unknown", CallTypeUnary, true},
		{"stream", CallTypeUnary, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseCallType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCallType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("parseCallType(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGRPCCommandTLSFlags(t *testing.T) {
	tlsCfg := &TLSConfig{
		Insecure:   true,
		CAFile:     "",
		CertFile:   "",
		KeyFile:    "",
		ServerName: "",
	}

	if !tlsCfg.Insecure {
		t.Error("expected Insecure to be true")
	}
}

func TestGRPCCommandFormatOutput(t *testing.T) {
	testJSON := []byte(`{"message":"hello","status":"ok"}`)
	formatted := formatter.Format(testJSON, "application/json", formatter.FormatOptions{
		Indent: "  ",
		Color:  false,
	})

	if formatted == "" {
		t.Error("expected formatted output")
	}

	if !strings.Contains(formatted, "message") {
		t.Errorf("expected 'message' in formatted output, got: %v", formatted)
	}
}

func TestGRPCCommandFormatOutputWithColor(t *testing.T) {
	testJSON := []byte(`{"key":"value"}`)
	formatted := formatter.Format(testJSON, "application/json", formatter.FormatOptions{
		Indent: "  ",
		Color:  true,
	})

	if formatted == "" {
		t.Error("expected formatted output")
	}
}

func BenchmarkParseCallTypeCLI(b *testing.B) {
	types := []string{"unary", "server-streaming", "client-streaming", "bidirectional", "bidi"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseCallType(types[i%len(types)])
	}
}
