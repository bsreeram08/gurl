# scurl - Smart Curl Saver & API Companion

## 1. Concept & Vision

**scurl** replaces your chaotic `Ctrl+R` curl history with an intelligent, named request library. It's the tool you reach for when you're tired of typing `curl -X POST https://api.example.com/orders/12345/items` for the 47th time only to realize you meant `12346`.

The experience: Pipe any curl command through `scurl detect`, give it a memorable name like "add-item-to-order", and run it forever with `scurl run "add-item-to-order"`. It's Insomnia for the terminal—agent-friendly, plugin-extensible, and designed for the person who lives in the CLI.

**Core Philosophy**: Local-first, config-driven, plugin-based. No cloud dependencies by default. Every behavior is tunable via TOML config. Every extension point is a plugin.

---

## 2. Design Language

### Aesthetic Direction
Terminal-native with JSON-render UI for structured output. Think `btm` meets `lazygit`—information density with clear visual hierarchy. Monospace typography throughout.

### Color Palette
```
Background:   #1E1E2E (dark) / #FFFFFF (light)
Surface:      #313244 (dark) / #F5F5F5 (light)
Primary:      #89B4FA (blue accent)
Success:      #A6E3A1 (green)
Warning:      #F9E2AF (yellow)
Error:        #F38BA8 (red)
Text Primary: #CDD6F4 (dark) / #1E1E2E (light)
Text Muted:   #A6ADC8
```

### Typography
- **Primary**: System monospace (`JetBrains Mono` if available)
- **Fallback**: `Courier New`, `monospace`
- All output aligned to 80-char width where practical

### Spatial System
- 2-space indentation for nested output
- Consistent padding: 1 char between logical sections
- Status lines at top, content in middle, hints at bottom

### Motion Philosophy
Minimal animation. Instant feedback for commands. TUI transitions only where they aid comprehension (e.g., progress indicators).

### Visual Assets
- ASCII box-drawing characters for tables and frames
- Unicode symbols for status indicators (✓ ✗ ●)
- No external images or icons

---

## 3. Layout & Structure

### CLI Interface (Default)
```
┌─────────────────────────────────────────────────────┐
│ scurl v0.1.0                                        │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Usage: scurl <command> [options]                   │
│                                                     │
│  Commands:                                          │
│    save      Save a curl request with a name       │
│    run       Execute a saved request               │
│    detect    Parse curl from stdin/file             │
│    list      List saved requests                   │
│    history   Show execution history                │
│    timeline  Global execution timeline              │
│    diff      Compare responses                      │
│    edit      Edit a saved request                  │
│    delete    Remove a saved request                 │
│    rename    Rename a saved request                 │
│    export    Export requests to file                │
│    import    Import requests from file              │
│    paste     Copy request as curl command           │
│    collection Manage collections                    │
│                                                     │
│  Options:                                          │
│    -h, --help     Show this help                    │
│    -v, --version  Show version                      │
│    --config      Path to config file               │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### TUI Interface (On-Demand)
Launched via:
- `scurl detect` (complex decision flows)
- `scurl edit <name>` (request editing)
- `scurl --tui` (full interactive mode)
- Any command where config specifies `tui_on_decisions: true`

TUI uses **bubbletea** (Elm-style) for interactive elements.

### Response Output
```
● GET https://api.example.com/health
  Status: 200 OK (142ms) | Size: 1.2KB

┌─ Headers ─────────────────────────────────────────┐
│ content-type: application/json                    │
│ x-request-id: abc-123                             │
└───────────────────────────────────────────────────┘

┌─ Body ────────────────────────────────────────────┐
│ {                                                 │
│   "status": "healthy",                            │
│   "version": "2.1.0"                             │
│ }                                                 │
└───────────────────────────────────────────────────┘
```

### JSON-Render Dashboard (for multi-request views)
```
┌─ API Health Dashboard ──────────────────────────────┐
│                                                     │
│  ● GET /health           200 (45ms)   healthy      │
│  ● GET /metrics          200 (89ms)   uptime: 99.9%│
│  ✗ GET /readiness        503 (12ms)   degraded     │
│                                                     │
│  Summary: 2/3 passing | Last check: 2 min ago      │
└─────────────────────────────────────────────────────┘
```

---

## 4. Features & Interactions

### 4.1 Save Command

**Purpose**: Save a curl command with a memorable name.

**Usage**:
```bash
scurl save "health check" https://api.example.com/health
scurl save "create order" -X POST https://api.example.com/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 123, "items": [{"sku": "ABC"}]}'
scurl save "get user" https://api.example.com/users/{{userId}} \
  --var userId=12345
```

**Behavior**:
1. Parse URL, method, headers, body from arguments
2. Extract variables automatically (IDs, UUIDs in URL path)
3. Normalize URL (remove trailing slashes, query param ordering)
4. Store in LMDB with name as primary key
5. Create indices for collection, tags, endpoint

**Options**:
- `--collection <name>` - Assign to collection
- `--tag <name>` - Add tag (can repeat)
- `--format <auto|json|table>` - Output format preference
- `--description <text>` - Human-readable description

**Edge Cases**:
- Duplicate name: Prompt to rename or overwrite
- Malformed curl: Show parse error with suggestions
- Missing URL: Show error "URL required"

---

### 4.2 Run Command

**Purpose**: Execute a saved request by name.

**Usage**:
```bash
scurl run "health check"
scurl "health check"              # shorthand
scurl run "get user" --var userId=67890
scurl run "create order" --format json
scurl run "create order" --cache  # use cached response if fresh
```

**Behavior**:
1. Look up request by name
2. Substitute variables from `--var` flags
3. Run curl with all options
4. Record execution in history
5. Format and display response
6. Update cache if enabled

**Output Format** (per saved request preference):
- `auto`: Detect JSON → pretty-print, else show raw
- `json`: Force JSON formatting
- `table`: Status-line format

**Error Handling**:
- Request not found: Show "No saved request named 'X'. Try `scurl list`."
- Variable missing: Show "Missing variable: userId. Usage: --var userId=123"
- Network error: Show error with curl exit code

---

### 4.3 Detect Command

**Purpose**: Parse curl from stdin or file, save with interactive decisions.

**Usage**:
```bash
echo "curl -X POST https://api.example.com/orders/123" | scurl detect
cat request.curl | scurl detect
scurl detect --file request.curl
```

**Flow** (TUI mode):
```
┌─ Detected Request ──────────────────────────────────────┐
│                                                             │
│  URL:    https://api.example.com/orders/123               │
│  Method: POST                                              │
│  Headers: Content-Type: application/json                   │
│  Body:    {"item": "widget"}                              │
│                                                             │
│  Detected variables: [{{orderId}} = "123"]                 │
│                                                             │
│  Name this request: [get-order                             ]
│  Collection: [personal        ] (optional)                 │
│  Tags: [api, example       ] (comma-separated)            │
│                                                             │
│  [ Save ]  [ Save as Template ]  [ Cancel ]               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Decision Points**:
1. Parse curl → show extracted components
2. Auto-detect variables (IDs, UUIDs) → offer to template
3. Check for similar existing requests → offer to merge/compare
4. Prompt for name, collection, tags
5. Ask: Save as-is or as template?

---

### 4.4 List Command

**Purpose**: Show saved requests with filtering.

**Usage**:
```bash
scurl list
scurl list "order*"
scurl list --collection orders
scurl list --tag auth
scurl list --json
scurl list --format table
```

**Output**:
```
┌─ Saved Requests ─────────────────────────────────────────┐
│  NAME                  COLLECTION    TAGS      UPDATED  │
├─────────────────────────────────────────────────────────┤
│  health check          monitoring    GET        2d ago  │
│  create order          orders        POST       5m ago  │
│  get user              users         GET        1h ago  │
│  get order             orders        GET        3d ago  │
└─────────────────────────────────────────────────────────┘
  4 requests | Showing 4 of 47
```

**Options**:
- `--json` - JSON output for scripting
- `--format table|list` - Output format
- `--limit <n>` - Limit results
- `--sort name|updated|collection` - Sort order

---

### 4.5 History Command

**Purpose**: Show execution history for a specific request.

**Usage**:
```bash
scurl history "create order"
scurl history "create order" --limit 10
```

**Output**:
```
┌─ History: create order ─────────────────────────────────┐
│  #   STATUS   DURATION   SIZE     TIMESTAMP             │
├──────────────────────────────────────────────────────────┤
│  1   201      245ms      1.2KB    2024-01-15 10:30:45   │
│  2   201      198ms      1.2KB    2024-01-15 09:15:22   │
│  3   500      89ms       0.3KB    2024-01-14 16:45:01   │
└──────────────────────────────────────────────────────────┘
```

---

### 4.6 Timeline Command

**Purpose**: Global view of all request executions.

**Usage**:
```bash
scurl timeline
scurl timeline --since 24h
scurl timeline --filter "order*"
```

**Output**:
```
┌─ Execution Timeline (Last 24h) ─────────────────────────┐
│  10:30:45  ● create order        201  245ms            │
│  09:15:22  ● create order        201  198ms            │
│  08:00:00  ● health check        200   45ms            │
│  16:45:01  ✗ create order        500   89ms            │
└─────────────────────────────────────────────────────────┘
```

---

### 4.7 Diff Command

**Purpose**: Compare last two responses for a request.

**Usage**:
```bash
scurl diff "create order"
```

**Output**:
```
┌─ Diff: create order (Last 2 executions) ────────────────┐
│                                                             │
│  - Response #3 (500 error at 16:45:01)                    │
│  + Response #4 (201 success at 10:30:45)                 │
│                                                             │
│  - "error": "Invalid customer_id"                         │
│  + "order_id": "ORD-456789"                              │
│  + "status": "pending"                                   │
│                                                             │
└───────────────────────────────────────────────────────────┘
```

---

### 4.8 Collection Management

**Purpose**: Organize requests into collections.

**Usage**:
```bash
scurl collection list
scurl collection add orders
scurl collection add users
scurl collection remove old-api
scurl collection rename old-api new-api
scurl save "list orders" --collection orders
```

**Output**:
```
┌─ Collections ────────────────────────────────────────────┐
│  NAME          REQUESTS   UPDATED                        │
├──────────────────────────────────────────────────────────┤
│  orders        12         5m ago                        │
│  users         8          1h ago                        │
│  monitoring    5          2d ago                        │
│  auth          3          1w ago                        │
└──────────────────────────────────────────────────────────┘
```

---

### 4.9 Export/Import

**Purpose**: Share requests and backup collections.

**Usage**:
```bash
scurl export "create order" > order.json
scurl export --collection orders > orders-backup.json
scurl export --all > full-backup.json
scurl import order.json
scurl paste "create order"   # copy as curl to clipboard
```

**Export Format**:
```json
{
  "version": "1.0",
  "exported_at": "2024-01-15T10:30:00Z",
  "requests": [
    {
      "name": "create order",
      "curl_cmd": "curl -X POST https://api.example.com/orders...",
      "collection": "orders",
      "tags": ["api", "example"]
    }
  ]
}
```

---

### 4.10 Edit Command

**Purpose**: Modify a saved request (TUI mode).

**Usage**:
```bash
scurl edit "create order"
```

**TUI Flow**:
- Edit URL, method, headers, body in form
- Add/remove tags and collection
- Preview changes before saving

---

## 5. Component Inventory

### 5.1 CLI Parser
Built with **mri** (lightweight Go equivalent: `urfave/cli` or native flag parsing).

**States**:
- Idle: Parse command and arguments
- Error: Show usage for unrecognized commands
- Help: Display command-specific help

### 5.2 LMDB Storage
Single database file at `~/.local/share/scurl/scurl.db`.

**Keys**:
```
request:{uuid}           → JSON request definition
history:{requestId}:{ts} → JSON history entry
idx:name:{name}          → requestId (unique)
idx:collection:{name}   → [requestId, ...]
idx:tag:{name}           → [requestId, ...]
idx:endpoint:{norm}     → [requestId, ...]
config                   → JSON config
```

**States**:
- Read: Fetch by key or scan by index
- Write: Put with transaction
- Error: Handle corruption, show recovery options

### 5.3 Curl Parser
Parse curl command-line into structured request.

**States**:
- Success: Return ParsedCurl struct
- Partial: Return with warnings (e.g., unknown flag)
- Error: Show parse error with position

**Auto-detection**:
- URL pattern matching for variables
- Header normalization
- JSON body detection

### 5.4 Template Engine
Substitute `{{variable}}` in URLs, headers, body.

**States**:
- Valid: All variables resolved
- Missing: Show missing variable names
- Extra: Warn about unused variables

### 5.5 Output Formatter
Format responses for display.

**Modes**:
- Auto: Detect JSON, fallback to raw
- JSON: Pretty-print with syntax highlight
- Table: Compact status-line format

### 5.6 Plugin Loader
Load plugins from `~/.config/scurl/plugins/`.

**Plugin Types**:
- Middleware: Transform requests/responses
- Output: Custom response formatters
- Command: New subcommands

### 5.7 TUI Launcher
Launch bubbletea TUI when decisions needed.

**Triggers**:
- `scurl detect` (interactive flow)
- `scurl edit` (form-based editing)
- `scurl --tui` (full interactive mode)
- Config: `tui_on_decisions: true`

### 5.8 Shell Completion
Generate completion scripts.

**Supported Shells**:
- bash
- zsh
- fish

**Completions**:
- Command names
- Saved request names
- Variable names for `--var`

---

## 6. Technical Approach

### Language & Runtime
- **Language**: Go 1.21+
- **Build**: `go build` → single static binary
- **Size Target**: < 10MB binary

### Key Libraries
```go
// CLI
github.com/urfave/cli/v3          // CLI framework

// Storage
github.com/syndtr/goleveldb/leveldb  // LMDB-like (or wrapper)

// TUI
github.com/charmbracelet/bubbletea  // TUI framework
github.com/charmbracelet/lipgloss   // Styling

// Config
github.com/pelletier/go-toml/v2     // TOML parsing

// HTTP (curl wrapper)
os/exec + net/http                  // Execute curl, parse output

// UI/Output
github.com/gookit/filter            // JSON handling
github.com/mattn/go-runewidth       // Unicode width

// Completion
github.com/urfave/cli/v3/complete   // Shell completions
```

### Project Structure
```
scurl/
├── cmd/
│   └── scurl/
│       └── main.go           # Entry point
├── internal/
│   ├── cli/
│   │   ├── parser.go        # CLI argument parsing
│   │   └── commands/        # Command handlers
│   │       ├── save.go
│   │       ├── run.go
│   │       ├── detect.go
│   │       ├── list.go
│   │       ├── history.go
│   │       ├── timeline.go
│   │       ├── diff.go
│   │       ├── edit.go
│   │       ├── delete.go
│   │       ├── rename.go
│   │       ├── export.go
│   │       ├── import.go
│   │       ├── paste.go
│   │       └── collection.go
│   ├── core/
│   │   ├── curl/
│   │   │   ├── parser.go    # Parse curl command
│   │   │   ├── executor.go  # Execute curl
│   │   │   └── detector.go  # Extract variables
│   │   ├── storage/
│   │   │   ├── db.go        # LMDB wrapper
│   │   │   ├── history.go   # History management
│   │   │   └── indices.go   # Index management
│   │   ├── template/
│   │   │   ├── engine.go    # Variable substitution
│   │   │   └── resolver.go  # Resolve variables
│   │   └── formatter/
│   │       ├── auto.go      # Auto-detect format
│   │       ├── json.go      # JSON formatter
│   │       ├── table.go     # Table formatter
│   │       └── jsonrender.go # Terminal UI components
│   ├── config/
│   │   ├── loader.go        # TOML config loader
│   │   └── defaults.go      # Default config
│   ├── plugins/
│   │   ├── loader.go        # Plugin loader
│   │   ├── middleware.go    # Middleware interface
│   │   ├── output.go        # Output plugin interface
│   │   └── commands.go      # Command plugin interface
│   ├── tui/
│   │   ├── launcher.go      # Launch TUI
│   │   ├── detect.go        # Detect flow TUI
│   │   ├── edit.go          # Edit flow TUI
│   │   └── components/      # TUI components
│   └── agent/
│       ├── api.go          # Programmatic API
│       └── rcfile.go       # Generate .scurlrc
├── pkg/
│   └── types/               # Shared types
├── completions/            # Shell completion scripts
├── config.go               # Default config file
├── go.mod
├── go.sum
└── README.md
```

### Data Model (LMDB)

**Database**: `~/.local/share/scurl/scurl.db`

```
# Request definitions
request:{uuid} → {
  "id": "uuid",
  "name": "create order",
  "curl_cmd": "curl -X POST ...",
  "url": "https://api.example.com/orders",
  "method": "POST",
  "headers": {"Content-Type": "application/json"},
  "body": "{\"customer_id\": 123}",
  "variables": [{"name": "customerId", "pattern": "\\d+", "example": "123"}],
  "collection": "orders",
  "tags": ["api", "production"],
  "output_format": "auto",
  "created_at": 1705312200,
  "updated_at": 1705312200
}

# Execution history (per request)
history:{requestId}:{timestamp} → {
  "id": "uuid",
  "request_id": "uuid",
  "response": "...",
  "status_code": 201,
  "duration_ms": 245,
  "size_bytes": 1234,
  "timestamp": 1705312245
}

# Indices
idx:name:{name} → requestId
idx:collection:{collection} → [requestId, ...]
idx:tag:{tag} → [requestId, ...]
idx:endpoint:{normalizedUrl} → [requestId, ...]
```

### Config File (TOML)

Location: `~/.scurlrc` or `./.scurlrc`

```toml
[general]
history_depth = 100          # Max history entries per request
auto_template = true         # Auto-detect variables in URLs
completion_mode = "both"     # shell, inline, both, none

[autocomplete]
enabled = true              # Enable shell completion
inline_enabled = true       # Enable inline suggestions (requires shell support)

[output]
default_format = "auto"      # auto, json, table
syntax_highlight = true     # Syntax highlight JSON
json_pretty = true          # Pretty-print JSON

[cache]
ttl_seconds = 300           # Cache TTL (0 = disabled)

[detect]
extract_variables = true    # Auto-extract IDs/UUIDs as variables
suggest_merge = true        # Suggest merging similar requests
prompt_templates = true     # Ask to create templates

[ui]
tui_on_decisions = true     # Launch TUI for complex decisions
tui_threshold_lines = 100    # Auto-TUI for responses > 100 lines

[plugins]
enabled = []                # List of enabled plugins
```

### Plugin System

**Plugin Location**: `~/.config/scurl/plugins/`

**Plugin Interface**:
```go
// Middleware - transform requests/responses
type MiddlewarePlugin interface {
    Name() string
    BeforeRequest(ctx *RequestContext) *RequestContext
    AfterResponse(ctx *ResponseContext) *ResponseContext
}

// Output - custom formatters
type OutputPlugin interface {
    Name() string
    Format() string // "slack", "notion", etc.
    Render(ctx *ResponseContext) string
}

// Command - new subcommands
type CommandPlugin interface {
    Name() string
    Command() string
    Description() string
    Run(args []string) error
}
```

**Example Plugin**:
```go
// ~/.config/scurl/plugins/auth-encrypt/main.go
package main

import "scurl/plugin"

func main() {
    plugin.Register(&AuthEncryptPlugin{})
}

type AuthEncryptPlugin struct{}

func (p *AuthEncryptPlugin) Name() string { return "auth-encrypt" }

func (p *AuthEncryptPlugin) BeforeRequest(ctx *plugin.RequestContext) *plugin.RequestContext {
    ctx.Request.Headers["Authorization"] = encrypt(ctx.Request.Headers["Authorization"])
    return ctx
}
```

### Agent Integration

**Programmatic API**:
```go
import "scurl"

func main() {
    // Run saved request
    resp, err := scurl.Run("create order", scurl.WithVars(map[string]string{
        "customerId": "12345",
    }))
    
    // Save new request
    err = scurl.Save("health check", "https://api.example.com/health", nil)
    
    // List requests
    requests, err := scurl.List(scurl.ListOptions{
        Collection: "orders",
    })
}
```

**Agent Config File**:
```bash
# Generate .scurlrc for agent
scurl agent-init > .scurlrc

# Agent queries requests
scurl list --json | jq '.[] | select(.name | contains("order"))'
```

---

## 7. Implementation Phases

### Phase 1: Core Foundation
- [ ] Project setup, Go dependencies
- [ ] LMDB storage layer
- [ ] CLI framework with commands
- [ ] `scurl save` - basic save
- [ ] `scurl run` - basic execute
- [ ] `scurl list` - list with filters

### Phase 2: Detect & Templates
- [ ] Curl parser
- [ ] Variable extraction
- [ ] `scurl detect` pipe flow
- [ ] TUI launcher (bubbletea)

### Phase 3: History & Timeline
- [ ] Execution history storage
- [ ] `scurl history`
- [ ] `scurl timeline`
- [ ] `scurl diff`

### Phase 4: Collections & Organization
- [ ] Collection management
- [ ] Tag system
- [ ] Auto-grouping by endpoint

### Phase 5: Output & UI
- [ ] JSON auto-formatting
- [ ] JSON-render components
- [ ] Configurable output formats

### Phase 6: Sharing & Sync
- [ ] Export/Import
- [ ] Paste as curl
- [ ] Git-compatible collection format

### Phase 7: Plugin System
- [ ] Plugin loader
- [ ] Middleware system
- [ ] Output plugins
- [ ] Command plugins

### Phase 8: Agent Integration
- [ ] Programmatic API (Go library)
- [ ] Shell completions
- [ ] .scurlrc generator

---

## 8. Success Metrics

- **Speed**: `scurl run "name"` completes in < 50ms (excluding network)
- **Storage**: 1000 requests + history < 50MB LMDB file
- **CLI**: All commands return in < 100ms
- **Parse**: Detect curl command in < 10ms
- **Binary**: Final binary < 10MB

---

## 9. Out of Scope (v1)

- GUI/Desktop app
- Web dashboard
- Cloud sync
- Team collaboration
- OpenAPI/Swagger import
- Request chaining (run A, use response in B)
- GraphQL support (explicit)
