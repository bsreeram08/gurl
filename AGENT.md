# AGENT.md - gurl Development Guide

## Project Overview

**gurl** is a smart curl saver and API companion built in Go. It replaces chaotic `Ctrl+R` curl history with an intelligent, named request library.

- **Language**: Go 1.26+
- **Module**: `github.com/sreeram/gurl`
- **Tech Stack**: Go + Bubbletea v2 (TUI) + goleveldb (storage) + urfave/cli/v3 (CLI)
- **Meaning**: "Gurl = Go URL" — the name is a play on "go" (the verb) and "url"

## Project Structure

```
gurl/
├── cmd/gurl/main.go           # Entry point, version var
├── internal/
│   ├── cli/commands/          # CLI command handlers (urfave/cli/v3)
│   ├── auth/                  # Auth: Basic, Bearer, API Key, Digest, OAuth1/2, AWSv4, NTLM
│   ├── client/                # HTTP client with middleware pipeline
│   ├── codegen/              # Code generation: curl, Go, Python, JavaScript
│   ├── config/               # TOML config loader
│   ├── cookies/             # Cookie jar management
│   ├── core/
│   │   ├── curl/             # Curl command parsing and execution
│   │   └── template/         # Variable substitution templates
│   ├── env/                  # Environment variable storage (dev/staging/prod)
│   ├── formatter/            # Output formatters (JSON, XML, body-only, etc.)
│   ├── history/             # Execution history tracking
│   ├── importers/           # Import from Postman, OpenAPI, Insomnia, Bruno, HAR
│   ├── plugins/             # Plugin system and builtin middleware/output plugins
│   ├── protocols/           # HTTP, GraphQL, gRPC, WebSocket, SSE
│   ├── reporters/            # Output reporters
│   ├── runner/              # Request execution engine (with scripting + assertions)
│   ├── scripting/           # JavaScript pre/post-request hooks (goja runtime)
│   ├── storage/             # goleveldb key-value store
│   ├── tui/                 # Bubbletea v2 interactive TUI
│   └── assertions/          # Response assertion engine
├── pkg/types/                # Shared types
├── Formula/                 # Homebrew formula
└── scripts/                  # Install and release scripts
```

## Key Technologies

### CLI Framework
- **urfave/cli/v3** - CLI argument parsing and command structure

### TUI Framework
- **charm.land/bubbletea/v2** - Elm-style TUI framework (Bubbletea v2 with Cursed Renderer)
- **charm.land/lipgloss/v2** - Terminal styling
- **charm.land/bubbles/v2** - TUI components (textinput, spinner, viewport, etc.)

### Storage
- **syndtr/goleveldb** - Fast embedded key-value store (LevelDB port)
- Database location: `~/.local/share/gurl/gurl.db`

### Configuration
- **go-toml/v2** - TOML config file parsing
- Config locations checked in order:
  1. `./.gurlrc`
  2. `~/.gurlrc`
  3. `~/.config/gurl/config.toml`

## Implemented Commands

All Phase 1-5 commands are implemented. See `gurl --help` for the full list:

| Command | Description |
|---------|-------------|
| `save` | Save a request (flags or raw curl string) |
| `run` | Execute a saved request with variable substitution |
| `list` | List saved requests |
| `delete` | Delete a saved request |
| `rename` | Rename a request |
| `detect` | Parse curl from stdin interactively |
| `edit` | Edit a request in TUI form |
| `history` | Execution history for a request |
| `timeline` | Global execution timeline |
| `diff` | Compare last two responses |
| `env` | Manage environments |
| `collection` | Manage collections |
| `sequence` | Run multiple requests in sequence |
| `graphql` | Execute a GraphQL query |
| `export` | Export requests to JSON |
| `import` | Import from OpenAPI/Insomnia/Bruno/Postman/HAR |
| `paste` | Copy request as curl to clipboard |
| `codegen` | Generate code (curl, Go, Python, JavaScript) |
| `tui` | Launch interactive TUI |
| `update` | Self-update to latest release |

## Code Conventions

### Go Style
- Use `pkg/` for public packages, `internal/` for private
- Interfaces first, then implementations
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Context propagation for cancellation

### Type Definitions (pkg/types/types.go)
```go
type SavedRequest struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    CurlCmd      string    `json:"curl_cmd"`
    URL          string    `json:"url"`
    Method       string    `json:"method"`
    Headers      []Header  `json:"headers"`
    Body         string    `json:"body,omitempty"`
    Variables    []Var     `json:"variables,omitempty"`
    Collection   string    `json:"collection,omitempty"`
    Tags         []string  `json:"tags,omitempty"`
    OutputFormat string    `json:"output_format"`
    CreatedAt    int64     `json:"created_at"`
    UpdatedAt    int64     `json:"updated_at"`
}
```

### Database Keys (goleveldb)
```
request:{uuid}              → JSON request
history:{requestId}:{ts}  → JSON history
idx:name:{name}           → requestId
idx:collection:{name}      → [requestId, ...]
idx:tag:{tag}             → [requestId, ...]
```

### Config Format (TOML)
```toml
[general]
history_depth = 100
auto_template = true
timeout = "30s"

[output]
default_format = "auto"
syntax_highlight = true
json_pretty = true
```

## Running & Testing

```bash
# Build (version from git tag)
make build VERSION=$(git describe --tags)
./gurl --version

# Build dev version
go build -o gurl ./cmd/gurl

# Run
./gurl --help
./gurl save "test" https://example.com
./gurl list

# Test
go test ./...

# Tidy dependencies
go mod tidy

# Build with version
go build -ldflags="-X main.version=v0.1.19" -o gurl ./cmd/gurl
```

## Environment Variables

- `GURL_CONFIG_PATH` - Override config file path
- `GURL_DB_PATH` - Override database path

## Important Notes for AI Agents

1. **Use Bubbletea v2 for TUI** - never implement raw terminal I/O
2. **goleveldb for storage** - single binary, no external DB needed
3. **TOML for config** - no YAML or JSON for user config
4. **Plugin system** - make everything extensible via `internal/plugins/`
5. **Context propagation** - all I/O operations must respect context cancellation
6. **Bubbletea v2 API** - uses `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`. View() methods return `tea.View` (struct), not `string`. Alt screen via `v.AltScreen = true` on the View struct
7. **Secrets** - environment secrets are AES-256-GCM encrypted at rest
