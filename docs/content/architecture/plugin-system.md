---
title: "Plugin System"
description: "Extensible middleware, output, and command plugins"
weight: 2
---

# Plugin System

Gurl's plugin system lets you extend functionality through three plugin interfaces. Plugins are discovered from `~/.config/gurl/plugins/` and loaded on startup.

## Plugin Interfaces

### MiddlewarePlugin

Runs before and after each request. Use for logging, timing, retry logic, and request modification.

```go
type MiddlewarePlugin interface {
    Register(reg *MiddlewareRegistry)
    ApplyBeforeRequest(req *Request) error
    ApplyAfterResponse(req *Request, resp *Response) error
}
```

### OutputPlugin

Formats and displays responses. Gurl includes built-in output plugins, but custom plugins can provide specialized formatting.

```go
type OutputPlugin interface {
    Name() string
    Format(resp *Response) string
}
```

### CommandPlugin

Adds new CLI commands. Use for project-specific workflows or integrations.

```go
type CommandPlugin interface {
    Register(cmd *CommandRegistry)
    Run(args []string) error
}
```

## Plugin Registry

The registry pattern manages plugin lifecycle and execution order.

```go
type MiddlewareRegistry struct {
    plugins []MiddlewarePlugin
}

func (r *MiddlewareRegistry) Register(p MiddlewarePlugin) {
    r.plugins = append(r.plugins, p)
}
```

## Middleware Chain

Built-in middleware executes in this order:

1. **Timing** - Measures request duration
2. **User-Agent** - Sets default User-Agent header
3. **Retry-401** - Automatically retry with fresh credentials on 401
4. **Logging** - Records request/response details

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

On startup, Gurl scans `~/.config/gurl/plugins/` for `.so` files. Each plugin file must export the `PluginSymbol` function:

```go
// plugin.go
package main

import "github.com/sreeram/gurl/..."

func PluginSymbol() interface{} {
    return &MyMiddlewarePlugin{}
}
```

## Security

Plugins run with the user's permissions. Gurl applies these safeguards:

- Only explicitly enabled plugins are loaded
- Each plugin call is wrapped in panic recovery
- A failing plugin does not block request execution
- Plugin logs are written to `~/.local/share/gurl/plugin.log`

## Creating a Custom Plugin

### Example: Request Signing Middleware

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "github.com/sreeram/gurl"
)

type SigningPlugin struct{}

func (p *SigningPlugin) Register(reg *gurl.MiddlewareRegistry) {
    reg.Register(p)
}

func (p *SigningPlugin) ApplyBeforeRequest(req *gurl.Request) error {
    secret := req.GetVariable("SIGNING_SECRET")
    if secret == "" {
        return nil
    }

    body := req.Body()
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(body)
    signature := hex.EncodeToString(h.Sum(nil))

    req.Headers.Set("X-Signature", signature)
    return nil
}

func (p *SigningPlugin) ApplyAfterResponse(req *gurl.Request, resp *gurl.Response) error {
    return nil
}

func PluginSymbol() interface{} {
    return &SigningPlugin{}
}
```

Compile and install:

```bash
go build -o ~/.config/gurl/plugins/signing.so ./plugin.go
```

Enable in `~/.config/gurl/config.toml`:

```toml
[plugins]
enabled = ["signing"]
```

> [!TIP]
> Use the `gurl plugin list` command to see all discovered and enabled plugins.
