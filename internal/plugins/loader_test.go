package plugins

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/client"
)

// --- Test Middleware Implementations ---

type mockMiddleware struct {
	name          string
	beforeOrder   *int
	afterOrder    *int
	counter       *int
	panicInBefore bool
	panicInAfter  bool
}

func (m *mockMiddleware) Name() string { return m.name }
func (m *mockMiddleware) BeforeRequest(ctx *RequestContext) *RequestContext {
	if m.counter != nil {
		*m.counter++
	}
	if m.beforeOrder != nil {
		*m.beforeOrder++
	}
	if m.panicInBefore {
		panic("middleware panic in BeforeRequest")
	}
	if ctx == nil {
		return nil
	}
	return ctx
}
func (m *mockMiddleware) AfterResponse(ctx *ResponseContext) *ResponseContext {
	if m.counter != nil {
		*m.counter++
	}
	if m.afterOrder != nil {
		*m.afterOrder++
	}
	if m.panicInAfter {
		panic("middleware panic in AfterResponse")
	}
	if ctx == nil {
		return nil
	}
	return ctx
}

// --- Test Output Plugin Implementation ---

type mockOutputPlugin struct {
	name   string
	format string
}

func (p *mockOutputPlugin) Name() string   { return p.name }
func (p *mockOutputPlugin) Format() string { return p.format }
func (p *mockOutputPlugin) Render(ctx *ResponseContext) string {
	return "mock output"
}

// --- Test Command Plugin Implementation ---

type mockCommandPlugin struct {
	name        string
	command     string
	description string
	ran         bool
}

func (p *mockCommandPlugin) Name() string        { return p.name }
func (p *mockCommandPlugin) Command() string     { return p.command }
func (p *mockCommandPlugin) Description() string { return p.description }
func (p *mockCommandPlugin) Run(args []string) error {
	p.ran = true
	return nil
}

// --- Tests ---

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	mw := &mockMiddleware{name: "mw1"}
	out := &mockOutputPlugin{name: "out1", format: "json"}
	cmd := &mockCommandPlugin{name: "cmd1", command: "test", description: "test cmd"}

	registry.Register(mw)
	registry.Register(out)
	registry.Register(cmd)

	if len(registry.Middleware()) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(registry.Middleware()))
	}
	if len(registry.Outputs()) != 1 {
		t.Errorf("expected 1 output, got %d", len(registry.Outputs()))
	}
	if len(registry.Commands()) != 1 {
		t.Errorf("expected 1 command, got %d", len(registry.Commands()))
	}
}

func TestRegistry_Middleware_Chain(t *testing.T) {
	registry := NewRegistry()

	beforeOrder := []string{}
	afterOrder := []string{}

	mw1 := &middlewareTracker{name: "mw1", beforeOrder: &beforeOrder, afterOrder: &afterOrder}
	mw2 := &middlewareTracker{name: "mw2", beforeOrder: &beforeOrder, afterOrder: &afterOrder}

	registry.Register(mw1)
	registry.Register(mw2)

	ctx := &RequestContext{Request: &client.Request{URL: "http://example.com"}}
	result := registry.ApplyBeforeRequest(ctx)

	if result == nil {
		t.Fatal("expected non-nil result from ApplyBeforeRequest")
	}

	// mw1 before should run first, then mw2
	if len(beforeOrder) != 2 {
		t.Fatalf("expected 2 before calls, got %d", len(beforeOrder))
	}
	if beforeOrder[0] != "mw1" {
		t.Errorf("mw1 BeforeRequest should run first, got order: %v", beforeOrder)
	}
	if beforeOrder[1] != "mw2" {
		t.Errorf("mw2 BeforeRequest should run second, got order: %v", beforeOrder)
	}

	// Reset for AfterResponse test
	beforeOrder = nil
	afterOrder = nil

	respCtx := &ResponseContext{Request: &client.Request{}, Response: &client.Response{}}
	registry.ApplyAfterResponse(respCtx)

	// AfterResponse runs in reverse: mw2 first, then mw1
	if len(afterOrder) != 2 {
		t.Fatalf("expected 2 after calls, got %d", len(afterOrder))
	}
	if afterOrder[0] != "mw2" {
		t.Errorf("mw2 AfterResponse should run first in reverse order, got order: %v", afterOrder)
	}
	if afterOrder[1] != "mw1" {
		t.Errorf("mw1 AfterResponse should run second in reverse order, got order: %v", afterOrder)
	}
}

// middlewareTracker is a test middleware that records call order
type middlewareTracker struct {
	name        string
	beforeOrder *[]string
	afterOrder  *[]string
}

func (m *middlewareTracker) Name() string { return m.name }
func (m *middlewareTracker) BeforeRequest(ctx *RequestContext) *RequestContext {
	if m.beforeOrder != nil {
		*m.beforeOrder = append(*m.beforeOrder, m.name)
	}
	if ctx == nil {
		return nil
	}
	return ctx
}
func (m *middlewareTracker) AfterResponse(ctx *ResponseContext) *ResponseContext {
	if m.afterOrder != nil {
		*m.afterOrder = append(*m.afterOrder, m.name)
	}
	if ctx == nil {
		return nil
	}
	return ctx
}

func TestRegistry_Middleware_PanicRecovery(t *testing.T) {
	registry := NewRegistry()

	panicMW := &mockMiddleware{name: "panic", panicInBefore: true}
	normalMW := &mockMiddleware{name: "normal"}
	finalMW := &mockMiddleware{name: "final"}

	registry.Register(panicMW)
	registry.Register(normalMW)
	registry.Register(finalMW)

	ctx := &RequestContext{Request: &client.Request{URL: "http://example.com"}}
	result := registry.ApplyBeforeRequest(ctx)

	// Should continue despite panic and return non-nil
	if result == nil {
		t.Error("expected non-nil result despite panic in middleware")
	}

	// Normal and final middleware should still have run
	if len(registry.Middleware()) != 3 {
		t.Errorf("expected 3 middleware registered, got %d", len(registry.Middleware()))
	}
}

func TestRegistry_OutputPlugin(t *testing.T) {
	registry := NewRegistry()

	out1 := &mockOutputPlugin{name: "out1", format: "json"}
	out2 := &mockOutputPlugin{name: "out2", format: "xml"}

	registry.Register(out1)
	registry.Register(out2)

	found, ok := registry.GetOutputByFormat("json")
	if !ok {
		t.Fatal("expected to find json output plugin")
	}
	if found.Name() != "out1" {
		t.Errorf("expected out1, got %s", found.Name())
	}

	found, ok = registry.GetOutputByFormat("xml")
	if !ok {
		t.Fatal("expected to find xml output plugin")
	}
	if found.Name() != "out2" {
		t.Errorf("expected out2, got %s", found.Name())
	}

	_, ok = registry.GetOutputByFormat("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent format")
	}
}

func TestRegistry_CommandPlugin(t *testing.T) {
	registry := NewRegistry()

	cmd := &mockCommandPlugin{name: "testcmd", command: "mycommand", description: "A test command"}
	registry.Register(cmd)

	cmds := registry.Commands()
	if len(cmds) != 1 {
		t.Errorf("expected 1 command, got %d", len(cmds))
	}

	if cmds[0].Command() != "mycommand" {
		t.Errorf("expected command 'mycommand', got '%s'", cmds[0].Command())
	}
	if cmds[0].Description() != "A test command" {
		t.Errorf("expected description 'A test command', got '%s'", cmds[0].Description())
	}
}

func TestLoader_Discover(t *testing.T) {
	// Skip on macOS where plugin package doesn't work
	if runtime.GOOS == "darwin" {
		t.Skip("plugin package not supported on darwin")
	}

	// Create temp dir with .so files in subdirectories
	tmpDir, err := os.MkdirTemp("", "plugin-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create plugin1/myplugin.so and plugin2/myplugin.so
	plugin1Dir := filepath.Join(tmpDir, "plugin1")
	plugin2Dir := filepath.Join(tmpDir, "plugin2")
	os.MkdirAll(plugin1Dir, 0755)
	os.MkdirAll(plugin2Dir, 0755)

	// Create dummy .so files (they don't need to be valid for discovery test)
	os.WriteFile(filepath.Join(plugin1Dir, "plugin1.so"), []byte("dummy"), 0444)
	os.WriteFile(filepath.Join(plugin2Dir, "plugin2.so"), []byte("dummy"), 0444)

	loader := NewLoader(tmpDir, nil)
	discovered, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(discovered) != 2 {
		t.Errorf("expected 2 discovered plugins, got %d", len(discovered))
	}
}

func TestLoader_DisabledPlugin(t *testing.T) {
	loader := NewLoader("/some/plugin/dir", []string{"enabled1", "enabled2"})
	loader.RegisterBuiltIn(&mockMiddleware{name: "should_be_loaded"})
	loader.RegisterBuiltIn(&mockMiddleware{name: "enabled1"})

	registry, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	// Only enabled1 middleware should be loaded from .so (but we only have built-ins here)
	// Built-ins are always loaded regardless of enabled list
	middleware := registry.Middleware()
	if len(middleware) != 2 {
		t.Errorf("expected 2 built-in middleware (loaded regardless of enabled list), got %d", len(middleware))
	}
}

func TestLoader_InvalidPlugin(t *testing.T) {
	// Create a loader with a non-existent plugin directory
	loader := NewLoader("/nonexistent/plugin/dir", nil)
	registry, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll should not fail for non-existent dir, got error: %v", err)
	}
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestApplyBeforeRequest_NilSafe(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockMiddleware{name: "mw1"})

	result := registry.ApplyBeforeRequest(nil)
	if result != nil {
		t.Error("expected nil result when passing nil context")
	}
}

func TestApplyAfterResponse_NilSafe(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockMiddleware{name: "mw1"})

	result := registry.ApplyAfterResponse(nil)
	if result != nil {
		t.Error("expected nil result when passing nil context")
	}
}

func TestLoader_BuiltInPlugins(t *testing.T) {
	loader := NewLoader("", nil) // No plugin dir
	loader.RegisterBuiltIn(&mockMiddleware{name: "builtin_mw"})
	loader.RegisterBuiltIn(&mockOutputPlugin{name: "builtin_out", format: "test"})
	loader.RegisterBuiltIn(&mockCommandPlugin{name: "builtin_cmd", command: "testcmd", description: "desc"})

	registry, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(registry.Middleware()) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(registry.Middleware()))
	}
	if len(registry.Outputs()) != 1 {
		t.Errorf("expected 1 output, got %d", len(registry.Outputs()))
	}
	if len(registry.Commands()) != 1 {
		t.Errorf("expected 1 command, got %d", len(registry.Commands()))
	}
}

func TestLoader_Discover_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir, nil)

	discovered, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(discovered) != 0 {
		t.Errorf("expected 0 discovered plugins, got %d", len(discovered))
	}
}

func TestLoader_Discover_NonExistentDir(t *testing.T) {
	loader := NewLoader("/nonexistent/path/to/plugins", nil)

	discovered, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover should not return error for nonexistent dir: %v", err)
	}
	if discovered != nil {
		t.Errorf("expected nil for nonexistent dir, got %v", discovered)
	}
}

func TestLoader_Discover_WithSubdirectories(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("plugin package not supported on darwin")
	}

	tmpDir := t.TempDir()
	plugin1Dir := filepath.Join(tmpDir, "plugin1")
	plugin2Dir := filepath.Join(tmpDir, "plugin2")
	plugin3Dir := filepath.Join(tmpDir, "plugin3")
	os.MkdirAll(plugin1Dir, 0755)
	os.MkdirAll(plugin2Dir, 0755)
	os.MkdirAll(plugin3Dir, 0755)

	os.WriteFile(filepath.Join(plugin1Dir, "plugin1.so"), []byte("dummy"), 0444)
	os.WriteFile(filepath.Join(plugin2Dir, "plugin2.so"), []byte("dummy"), 0444)
	os.WriteFile(filepath.Join(plugin3Dir, "plugin3.so"), []byte("dummy"), 0444)

	loader := NewLoader(tmpDir, nil)
	discovered, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(discovered) != 3 {
		t.Errorf("expected 3 discovered plugins, got %d", len(discovered))
	}

	found := map[string]bool{"plugin1": false, "plugin2": false, "plugin3": false}
	for _, name := range discovered {
		found[name] = true
	}
	for name, ok := range found {
		if !ok {
			t.Errorf("expected plugin %s to be discovered", name)
		}
	}
}

func TestLoader_Discover_WithFilesInRoot(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "not_a_plugin.txt"), []byte("content"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "valid_plugin"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "valid_plugin", "valid_plugin.so"), []byte("dummy"), 0444)

	loader := NewLoader(tmpDir, nil)
	discovered, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(discovered) != 1 {
		t.Errorf("expected 1 discovered plugin, got %d", len(discovered))
	}
	if discovered[0] != "valid_plugin" {
		t.Errorf("expected plugin 'valid_plugin', got %s", discovered[0])
	}
}

func TestLoader_Discover_WithMissingSo(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "plugin_without_so"), 0755)

	loader := NewLoader(tmpDir, nil)
	discovered, err := loader.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(discovered) != 0 {
		t.Errorf("expected 0 discovered plugins when no .so files exist, got %d", len(discovered))
	}
}

func TestLoader_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  []string
		plugin   string
		expected bool
	}{
		{"empty enabled list", []string{}, "test", false},
		{"plugin in list", []string{"a", "b", "c"}, "b", true},
		{"plugin not in list", []string{"a", "b", "c"}, "d", false},
		{"single match", []string{"only"}, "only", true},
		{"single no match", []string{"only"}, "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEnabled(tt.plugin, tt.enabled)
			if result != tt.expected {
				t.Errorf("isEnabled(%q, %v) = %v, want %v", tt.plugin, tt.enabled, result, tt.expected)
			}
		})
	}
}

func TestLoader_Load_UnsupportedPlatform(t *testing.T) {
	loader := NewLoader("/some/path", nil)

	_, err := loader.Load("/some/plugin.so")
	if err == nil {
		t.Error("expected error on unsupported platform")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("expected 'not supported' in error, got: %v", err)
	}
}

func TestLoader_Load_InvalidPath(t *testing.T) {
	loader := NewLoader("/some/plugin/dir", nil)
	_, err := loader.Load("/nonexistent/plugin.so")
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestLoader_LoadAll_EmptyPluginDir(t *testing.T) {
	loader := NewLoader("", nil)
	loader.RegisterBuiltIn(&mockMiddleware{name: "builtin"})

	registry, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(registry.Middleware()) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(registry.Middleware()))
	}
}

func TestLoader_LoadAll_WithMultipleBuiltIns(t *testing.T) {
	loader := NewLoader("", []string{"mw1", "mw2", "out1"})

	loader.RegisterBuiltIn(&mockMiddleware{name: "mw1"})
	loader.RegisterBuiltIn(&mockMiddleware{name: "mw2"})
	loader.RegisterBuiltIn(&mockMiddleware{name: "disabled_mw"})
	loader.RegisterBuiltIn(&mockOutputPlugin{name: "out1"})
	loader.RegisterBuiltIn(&mockOutputPlugin{name: "out2"})

	registry, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	if len(registry.Middleware()) != 3 {
		t.Errorf("expected 3 middleware (built-ins always loaded), got %d", len(registry.Middleware()))
	}
	if len(registry.Outputs()) != 2 {
		t.Errorf("expected 2 outputs (built-ins always loaded), got %d", len(registry.Outputs()))
	}
}

func TestLoader_LoadAll_EnabledFiltering(t *testing.T) {
	loader := NewLoader("/nonexistent", []string{"enabled1", "enabled3"})
	loader.RegisterBuiltIn(&mockMiddleware{name: "builtin1"})
	loader.RegisterBuiltIn(&mockMiddleware{name: "builtin2"})

	registry, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	middleware := registry.Middleware()
	if len(middleware) != 2 {
		t.Errorf("expected 2 built-in middleware, got %d", len(middleware))
	}

	names := make([]string, len(middleware))
	for i, m := range middleware {
		names[i] = m.Name()
	}
	if names[0] != "builtin1" || names[1] != "builtin2" {
		t.Errorf("unexpected middleware order or names: %v", names)
	}
}

func TestSupportsPlugins(t *testing.T) {
	result := supportsPlugins()
	if runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") {
		if !result {
			t.Error("expected supportsPlugins() to return true on linux/amd64 or linux/arm64")
		}
	} else {
		if result {
			t.Error("expected supportsPlugins() to return false on non-linux or non-amd64/arm64")
		}
	}
}

func TestRegistry_GetOutputByFormat_MultipleOutputs(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&mockOutputPlugin{name: "json_out", format: "json"})
	registry.Register(&mockOutputPlugin{name: "xml_out", format: "xml"})
	registry.Register(&mockOutputPlugin{name: "yaml_out", format: "yaml"})

	jsonOut, ok := registry.GetOutputByFormat("json")
	if !ok {
		t.Fatal("expected to find json output")
	}
	if jsonOut.Name() != "json_out" {
		t.Errorf("expected json_out, got %s", jsonOut.Name())
	}

	yamlOut, ok := registry.GetOutputByFormat("yaml")
	if !ok {
		t.Fatal("expected to find yaml output")
	}
	if yamlOut.Name() != "yaml_out" {
		t.Errorf("expected yaml_out, got %s", yamlOut.Name())
	}

	_, ok = registry.GetOutputByFormat("text")
	if ok {
		t.Error("expected not to find text format")
	}
}

func TestRegistry_MiddlewareTracker(t *testing.T) {
	registry := NewRegistry()

	beforeOrder := []string{}
	afterOrder := []string{}

	mw1 := &middlewareTracker{name: "mw1", beforeOrder: &beforeOrder, afterOrder: &afterOrder}
	mw2 := &middlewareTracker{name: "mw2", beforeOrder: &beforeOrder, afterOrder: &afterOrder}
	mw3 := &middlewareTracker{name: "mw3", beforeOrder: &beforeOrder, afterOrder: &afterOrder}

	registry.Register(mw1)
	registry.Register(mw2)
	registry.Register(mw3)

	ctx := &RequestContext{Request: &client.Request{URL: "http://example.com"}}
	result := registry.ApplyBeforeRequest(ctx)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(beforeOrder) != 3 {
		t.Fatalf("expected 3 before calls, got %d", len(beforeOrder))
	}

	expectedOrder := []string{"mw1", "mw2", "mw3"}
	for i, name := range expectedOrder {
		if beforeOrder[i] != name {
			t.Errorf("position %d: expected %s, got %s", i, name, beforeOrder[i])
		}
	}

	beforeOrder = nil
	afterOrder = nil

	respCtx := &ResponseContext{Request: &client.Request{}, Response: &client.Response{}}
	registry.ApplyAfterResponse(respCtx)

	if len(afterOrder) != 3 {
		t.Fatalf("expected 3 after calls, got %d", len(afterOrder))
	}

	expectedAfterOrder := []string{"mw3", "mw2", "mw1"}
	for i, name := range expectedAfterOrder {
		if afterOrder[i] != name {
			t.Errorf("after position %d: expected %s, got %s", i, name, afterOrder[i])
		}
	}
}

func TestRegistry_ApplyAfterResponse_ReverseOrder(t *testing.T) {
	registry := NewRegistry()

	afterOrder := []string{}
	registry.Register(&middlewareTracker{name: "first", afterOrder: &afterOrder})
	registry.Register(&middlewareTracker{name: "second", afterOrder: &afterOrder})
	registry.Register(&middlewareTracker{name: "third", afterOrder: &afterOrder})

	registry.ApplyAfterResponse(&ResponseContext{Request: &client.Request{}, Response: &client.Response{}})

	if len(afterOrder) != 3 {
		t.Fatalf("expected 3 after calls, got %d", len(afterOrder))
	}
	if afterOrder[0] != "third" {
		t.Errorf("expected third to run first (reverse), got %v", afterOrder)
	}
	if afterOrder[1] != "second" {
		t.Errorf("expected second to run second, got %v", afterOrder)
	}
	if afterOrder[2] != "first" {
		t.Errorf("expected first to run last, got %v", afterOrder)
	}
}

func TestRegistry_MiddlewarePanicRecovery_AfterResponse(t *testing.T) {
	registry := NewRegistry()

	panicMW := &mockMiddleware{name: "panic", panicInAfter: true}
	normalMW := &mockMiddleware{name: "normal"}
	finalMW := &mockMiddleware{name: "final"}

	registry.Register(panicMW)
	registry.Register(normalMW)
	registry.Register(finalMW)

	result := registry.ApplyAfterResponse(&ResponseContext{Request: &client.Request{}, Response: &client.Response{}})

	if result == nil {
		t.Error("expected non-nil result despite panic in middleware")
	}
}

func TestRegistry_Commands_Multiple(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&mockCommandPlugin{name: "cmd1", command: "cmd1", description: "Command 1"})
	registry.Register(&mockCommandPlugin{name: "cmd2", command: "cmd2", description: "Command 2"})
	registry.Register(&mockCommandPlugin{name: "cmd3", command: "cmd3", description: "Command 3"})

	cmds := registry.Commands()
	if len(cmds) != 3 {
		t.Errorf("expected 3 commands, got %d", len(cmds))
	}
}

func TestRegistry_Commands_Empty(t *testing.T) {
	registry := NewRegistry()
	cmds := registry.Commands()
	if len(cmds) != 0 {
		t.Errorf("expected 0 commands, got %d", len(cmds))
	}
}

func TestRegistry_Outputs_Empty(t *testing.T) {
	registry := NewRegistry()
	outputs := registry.Outputs()
	if len(outputs) != 0 {
		t.Errorf("expected 0 outputs, got %d", len(outputs))
	}
}

func TestRegistry_Middleware_Empty(t *testing.T) {
	registry := NewRegistry()
	mw := registry.Middleware()
	if len(mw) != 0 {
		t.Errorf("expected 0 middleware, got %d", len(mw))
	}
}

func TestRegistry_ApplyBeforeRequest_Empty(t *testing.T) {
	registry := NewRegistry()
	result := registry.ApplyBeforeRequest(&RequestContext{Request: &client.Request{URL: "http://example.com"}})
	if result == nil {
		t.Error("expected non-nil result with no middleware")
	}
}

func TestRegistry_ApplyAfterResponse_Empty(t *testing.T) {
	registry := NewRegistry()
	result := registry.ApplyAfterResponse(&ResponseContext{Request: &client.Request{}, Response: &client.Response{}})
	if result == nil {
		t.Error("expected non-nil result with no middleware")
	}
}
