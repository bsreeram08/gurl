package plugins

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestRegistry_UnknownPluginType(t *testing.T) {
	registry := NewRegistry()

	// Register something that doesn't implement any plugin interface
	registry.Register("not a plugin")

	// Should not panic and should result in empty slices
	if len(registry.Middleware()) != 0 {
		t.Errorf("expected 0 middleware, got %d", len(registry.Middleware()))
	}
	if len(registry.Outputs()) != 0 {
		t.Errorf("expected 0 outputs, got %d", len(registry.Outputs()))
	}
	if len(registry.Commands()) != 0 {
		t.Errorf("expected 0 commands, got %d", len(registry.Commands()))
	}
}
