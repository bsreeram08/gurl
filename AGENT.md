# AGENT.md - gurl Development Guide

## Project Overview

**gurl** is a smart curl saver and API companion built in Go. It replaces chaotic `Ctrl+R` curl history with an intelligent, named request library.

- **Language**: Go 1.21+
- **Module**: `github.com/sreeram/gurl`
- **Tech Stack**: Go + bubbletea (TUI) + LMDB (storage) + urfave/cli (CLI)
- **PRD**: See `PRD.md` for full specification

## Project Structure

```
gurl/
├── cmd/gurl/main.go           # Entry point
├── internal/
│   ├── cli/                    # CLI framework
│   │   ├── parser.go          # Argument parsing
│   │   └── commands/          # Command handlers
│   ├── core/
│   │   ├── curl/              # Curl parsing & execution
│   │   ├── storage/           # LMDB database
│   │   ├── template/         # Variable substitution
│   │   └── formatter/         # Output formatting
│   ├── config/                # TOML config loader
│   ├── plugins/               # Plugin system
│   ├── tui/                  # Bubbletea TUI
│   └── agent/                 # Agent/programmatic API
├── pkg/types/                 # Shared types
└── completions/              # Shell completions
```

## Key Technologies

### CLI Framework
- **urfave/cli/v3** - CLI argument parsing and command structure

### TUI Framework
- **charmbracelet/bubbletea** - Elm-style TUI framework
- **charmbracelet/lipgloss** - Terminal styling

### Storage
- **LMDB** via Go wrapper - Fast embedded key-value store
- Database location: `~/.local/share/gurl/gurl.db`

### Configuration
- **go-toml/v2** - TOML config file parsing
- Config locations checked in order:
  1. `./.gurlrc`
  2. `~/.gurlrc`
  3. `~/.config/gurl/config.toml`

## Commands to Implement

### Phase 1 (Core Foundation)
- [ ] `gurl save <name> <url> [options]` - Save a request
- [ ] `gurl run <name> [--var key=value]` - Execute a request
- [ ] `gurl list [--pattern] [--collection] [--tag] [--json]` - List requests
- [ ] `gurl delete <name>` - Delete a request
- [ ] `gurl rename <old> <new>` - Rename a request

### Phase 2 (Detect & Templates)
- [ ] `gurl detect` - Parse curl from stdin
- [ ] Variable extraction ({{var}})
- [ ] Template engine

### Phase 3 (History & Timeline)
- [ ] `gurl history <name>` - Per-request history
- [ ] `gurl timeline` - Global timeline
- [ ] `gurl diff <name>` - Compare responses

### Phase 4 (Collections)
- [ ] `gurl collection add|list|remove|rename`
- [ ] Tag management
- [ ] Auto-grouping by endpoint

### Phase 5-8 (Advanced)
- Output formatting, export/import, plugin system, agent API

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

### Database Keys (LMDB)
```
request:{uuid}              → JSON request
history:{requestId}:{ts}    → JSON history
idx:name:{name}             → requestId
idx:collection:{name}       → [requestId, ...]
idx:tag:{tag}               → [requestId, ...]
```

### Config Format (TOML)
```toml
[general]
history_depth = 100
auto_template = true
completion_mode = "both"

[output]
default_format = "auto"
syntax_highlight = true
```

## Running & Testing

```bash
# Build
go build -o gurl ./cmd/gurl

# Run
./gurl --help
./gurl save "test" https://example.com
./gurl list

# Test
go test ./...

# Tidy dependencies
go mod tidy
```

## Environment Variables

- `GURL_CONFIG_PATH` - Override config file path
- `GURL_DB_PATH` - Override database path

## Important Notes for AI Agents

1. **Always read PRD.md first** before implementing any feature
2. **Follow Go conventions** from golang-pro skill
3. **Use bubbletea for TUI** - never implement raw terminal I/O
4. **LMDB for storage** - single binary, no external DB needed
5. **TOML for config** - no YAML or JSON for user config
6. **Plugin system** - make everything extensible
7. **Agent-friendly** - include programmatic API for other tools to use

## Skills Available

This project has these skills loaded:
- `golang-pro` - Go best practices and patterns
- `golang-testing` - Testing strategies for Go
- `building-tui-apps` - TUI development with bubbletea
- `cli-developer` - CLI development patterns

Activate a skill with: `@skill <skill-name>`
