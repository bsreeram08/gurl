package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
)

// Loader discovers and loads plugins from .so files or built-in registration.
type Loader struct {
	pluginDir string
	enabled   []string
	// builtInPlugins allows registering plugins via code (for testing/cross-platform)
	builtInPlugins []interface{}
}

// NewLoader creates a new Loader with the specified plugin directory and enabled list.
func NewLoader(pluginDir string, enabled []string) *Loader {
	return &Loader{
		pluginDir:      pluginDir,
		enabled:        enabled,
		builtInPlugins: []interface{}{},
	}
}

// RegisterBuiltIn adds a plugin that was registered via code rather than loaded from .so.
// This is useful for testing and for cross-platform compatibility where the plugin
// package doesn't work.
func (l *Loader) RegisterBuiltIn(plugin interface{}) {
	l.builtInPlugins = append(l.builtInPlugins, plugin)
}

// Discover finds all .so files in pluginDir/*/ subdirectories.
// Returns a list of plugin names (directory names) that contain .so files.
func (l *Loader) Discover() ([]string, error) {
	if l.pluginDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No plugin directory, not an error
		}
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	var plugins []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginName := entry.Name()
		pluginPath := filepath.Join(l.pluginDir, pluginName, pluginName+".so")
		if _, err := os.Stat(pluginPath); err == nil {
			plugins = append(plugins, pluginName)
		}
	}
	return plugins, nil
}

// Load loads a plugin from the specified .so file path.
// Uses plugin.Open() and plugin.Lookup("Plugin") to load the plugin symbol.
func (l *Loader) Load(path string) (interface{}, error) {
	// Check if we're on a platform that supports plugins
	if !supportsPlugins() {
		return nil, fmt.Errorf("plugin loading not supported on %s", runtime.GOOS)
	}

	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	symbol, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup Plugin symbol in %s: %w", path, err)
	}

	// plugin.Lookup returns a Symbol (which is interface{}), but the actual
	// plugin exports a value like `var Plugin = MyPlugin{}`. The Symbol is
	// a pointer to that value, so we need to dereference it for the type
	// switch in Registry.Register to match correctly.
	deref, ok := symbol.(interface{ Elem() interface{} })
	if ok {
		return deref.Elem(), nil
	}
	return symbol, nil
}

// LoadAll discovers plugins, filters by enabled list, loads each .so, and
// registers them into a new Registry. Also includes any built-in plugins
// registered via RegisterBuiltIn.
func (l *Loader) LoadAll() (*Registry, error) {
	registry := NewRegistry()

	// First, register all built-in plugins (they work on all platforms)
	for _, plugin := range l.builtInPlugins {
		registry.Register(plugin)
	}

	// Then load .so plugins if on supported platform
	if !supportsPlugins() {
		return registry, nil
	}

	discovered, err := l.Discover()
	if err != nil {
		return nil, err
	}

	for _, name := range discovered {
		// Skip if enabled list is specified and this plugin is not in it
		if len(l.enabled) > 0 && !isEnabled(name, l.enabled) {
			continue
		}

		pluginPath := filepath.Join(l.pluginDir, name, name+".so")
		loaded, err := l.Load(pluginPath)
		if err != nil {
			// Log error but continue loading other plugins
			fmt.Fprintf(os.Stderr, "Warning: failed to load plugin %s: %v\n", name, err)
			continue
		}

		registry.Register(loaded)
	}

	return registry, nil
}

// supportsPlugins returns true if the current platform supports plugin loading.
func supportsPlugins() bool {
	// Go's plugin package only works on linux/amd64 and linux/arm64
	return runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")
}

// isEnabled checks if a plugin name is in the enabled list.
func isEnabled(name string, enabled []string) bool {
	for _, e := range enabled {
		if e == name {
			return true
		}
	}
	return false
}
