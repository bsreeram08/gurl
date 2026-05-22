---
title: "Plugin System"
description: "Extensible middleware, output, command, and auth plugins"
weight: 2
---

# Plugin System

Gurl's plugin system lets you extend functionality through middleware, output, command, and auth plugin interfaces. Go plugins are discovered from `~/.config/gurl/plugins/` and loaded into the plugin registry on startup.

## Plugin Interfaces

### MiddlewarePlugin

Runs before and after each request. Use for logging, timing, retry logic, and request modification.

```go
type MiddlewarePlugin interface {
    Name() string
    BeforeRequest(ctx *RequestContext) *RequestContext
    AfterResponse(ctx *ResponseContext) *ResponseContext
}
```

### OutputPlugin

Formats and displays responses. Gurl includes built-in output plugins, but custom plugins can provide specialized formatting.

```go
type OutputPlugin interface {
    Name() string
    Format() string
    Render(ctx *ResponseContext) string
}
```

### CommandPlugin

Adds new CLI commands. Use for project-specific workflows or integrations.

```go
type CommandPlugin interface {
    Name() string
    Command() string
    Description() string
    Run(args []string) error
}
```

### AuthPlugin

Applies authentication to an outgoing request. Auth plugins expose parameter metadata through the same `auth.ParamDef` shape used by built-in handlers.

```go
type AuthPlugin interface {
    Name() string
    Params() []auth.ParamDef
    Apply(req *client.Request, params map[string]string) error
}
```

## Plugin Registry

The registry stores each plugin by the interfaces it implements.

```go
type Registry struct {
    middleware []MiddlewarePlugin
    outputs    []OutputPlugin
    commands   []CommandPlugin
    auths      []AuthPlugin
}

func (r *Registry) Register(plugin interface{}) {
    switch p := plugin.(type) {
    case MiddlewarePlugin:
        r.middleware = append(r.middleware, p)
    case OutputPlugin:
        r.outputs = append(r.outputs, p)
    case CommandPlugin:
        r.commands = append(r.commands, p)
    case AuthPlugin:
        r.auths = append(r.auths, p)
    }
}
```

Auth plugins are stored separately from middleware and can be listed or looked up by name before their `Apply` method updates a request.

## Request execution touchpoints

During a request run, extensions can affect different parts of execution:

1. **Middleware plugins** - Can inspect or change request context before the request and response context after the response
2. **Auth handlers** - Apply saved auth settings after templates are substituted and before the request is sent
3. **Output plugins** - Render the response when their format is selected

Custom middleware inserts into this chain via the registry.

## Built-in Output Plugins

| Plugin | Description | Example Output |
|--------|-------------|----------------|
| pretty | Formatted JSON/XML with syntax highlighting | Colored, indented output |
| slack | Slack-compatible attachment format | Block kit payload |
| markdown | Markdown code blocks | ```json ...``` |
| csv | Tabular data extraction | Comma-separated values |
| minimal | Single-line summary | `200 OK - 123ms - 2.1KB` |

Select an output plugin with the `--format` flag:

```bash
gurl run "api" --format slack
```

## Plugin Discovery

On startup, Gurl discovers shared objects at `~/.config/gurl/plugins/<name>/<name>.so`. If an enabled-plugin list is configured, only matching names are loaded. Each plugin file must export a `Plugin` symbol:

```go
// plugin.go
package main

import "github.com/sreeram/gurl/..."

var Plugin = MyMiddlewarePlugin{}
```

## Security

Plugins run with the user's permissions. Gurl applies these safeguards:

- If an enabled list is configured, plugins not on that list are skipped
- Middleware `BeforeRequest` and `AfterResponse` calls are wrapped in panic recovery
- Plugin discovery treats a missing plugin directory as empty
- Individual plugin load failures print a warning to stderr and do not stop other plugins from loading

## Creating a Custom Plugin

Custom plugins are currently low-level Go plugins, not a polished external SDK. A plugin must export a `Plugin` symbol whose value implements at least one known plugin interface. This example imports gurl's `internal/...` packages, so build it under the gurl module tree or another module path that Go permits to import those internal packages.

Minimal middleware plugin:

```go
package main

import (
    "github.com/sreeram/gurl/internal/client"
    "github.com/sreeram/gurl/internal/plugins"
)

type HeaderPlugin struct{}

func (HeaderPlugin) Name() string { return "example-header" }

func (HeaderPlugin) BeforeRequest(ctx *plugins.RequestContext) *plugins.RequestContext {
    if ctx != nil && ctx.Request != nil {
        ctx.Request.Headers = append(ctx.Request.Headers, client.Header{
            Key:   "X-Gurl-Plugin",
            Value: "example-header",
        })
    }
    return ctx
}

func (HeaderPlugin) AfterResponse(ctx *plugins.ResponseContext) *plugins.ResponseContext {
    return ctx
}

var Plugin = HeaderPlugin{}
```

Build and place the shared object where the loader expects it:

```bash
mkdir -p ~/.config/gurl/plugins/example-header
go build -buildmode=plugin -o ~/.config/gurl/plugins/example-header/example-header.so ./plugin.go
```

The loader discovers enabled plugins at `~/.config/gurl/plugins/<name>/<name>.so`, looks up the exported `Plugin` symbol, and accepts values that implement `MiddlewarePlugin`, `OutputPlugin`, `CommandPlugin`, or `AuthPlugin`.

## Current support

The registry and loader can categorize values that implement `MiddlewarePlugin`, `OutputPlugin`, `CommandPlugin`, or `AuthPlugin`. Auth plugins are stored separately from middleware and can be listed or looked up through the registry.

External plugin packaging is still low-level Go plugin support. The loader expects plugins under `~/.config/gurl/plugins/<name>/<name>.so` and a `Plugin` symbol that implements at least one known plugin interface. Go plugin loading is only supported on Linux amd64 and Linux arm64.
