# Gurl — Smart Curl Saver & API Companion

**GURL** = **G**url's **U**niversal **R**equest **L**ibrary

[![Latest Release](https://img.shields.io/github/v/release/bsreeram08/gurl)](https://github.com/bsreeram08/gurl/releases/latest)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-blue)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Save named curl requests, replay them with variable substitution, manage environments, assert on responses, and generate client code — all from the terminal.

## Features

- **Named requests** — save any curl command with a memorable name, run it forever
- **Variable templates** — auto-detect IDs/UUIDs, substitute at runtime with `--var`
- **Environments** — swap base URLs, secrets, and tokens between dev/staging/prod
- **Import** — OpenAPI, Insomnia, Bruno, Postman, HAR
- **Auth handlers** — Basic, Bearer, API Key, Digest, OAuth 1/2, AWS SigV4, NTLM
- **Protocols** — HTTP, GraphQL, gRPC, WebSocket, SSE
- **Scripting** — JavaScript pre/post-request hooks via goja runtime
- **Assertions** — assert on status, headers, and body (JSON path, XPath)
- **Collection runner** — data-driven testing with CSV/JSON input
- **Code generation** — generate curl, Go, Python, JavaScript from any saved request
- **Interactive TUI** — full bubbletea interface for browsing and running requests
- **Execution history** — per-request history + global timeline + diff between runs
- **Plugin system** — middleware and custom output formatters

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap bsreeram08/gurl https://github.com/bsreeram08/gurl
brew install gurl
```

### One-liner

```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/gurl/master/scripts/install.sh | bash
```

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/bsreeram08/gurl/releases/latest) (v0.1.15):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-darwin-arm64.tar.gz
tar -xzf gurl-darwin-arm64.tar.gz && sudo mv gurl /usr/local/bin/gurl

# Linux (amd64)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-linux-amd64.tar.gz
tar -xzf gurl-linux-amd64.tar.gz && sudo mv gurl /usr/local/bin/gurl

# Linux (arm64)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-linux-arm64.tar.gz
tar -xzf gurl-linux-arm64.tar.gz && sudo mv gurl /usr/local/bin/gurl
```

### Build from Source

```bash
git clone https://github.com/bsreeram08/gurl.git
cd gurl
go build -o gurl ./cmd/gurl
sudo mv gurl /usr/local/bin/
gurl --version
```

## Quick Start

```bash
# Save a request
gurl save "health" https://api.example.com/health

# Save with full curl flags
gurl save "create-order" https://api.example.com/orders \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {{token}}" \
  -d '{"customer_id": "{{customerId}}"}'

# Or pipe a raw curl string
echo 'curl -X POST https://api.example.com/orders -d "{}"' | gurl save "create-order"

# Run it
gurl run "create-order" --var token=abc123 --var customerId=42

# List all saved requests
gurl list
```

## All Commands

| Command | Description |
|---------|-------------|
| `save` | Save a request (flags or raw curl string) |
| `run` | Execute a saved request with variable substitution |
| `list` | List saved requests (filter by collection, tag, pattern) |
| `detect` | Parse curl from stdin interactively (TUI) |
| `edit` | Edit a saved request in TUI form |
| `delete` | Delete a saved request |
| `rename` | Rename a saved request |
| `show` | Show full request details |
| `history` | Show execution history for a request |
| `timeline` | Global execution timeline across all requests |
| `diff` | Compare last two responses for a request |
| `env` | Manage environments (create, list, show, switch) |
| `collection` | Manage collections |
| `sequence` | Run multiple requests in sequence |
| `graphql` | Execute a GraphQL query |
| `export` | Export requests to JSON |
| `import` | Import from OpenAPI/Insomnia/Bruno/Postman/HAR |
| `paste` | Copy request as curl command to clipboard |
| `codegen` | Generate code (curl, Go, Python, JavaScript) |
| `tui` | Launch full interactive TUI |
| `update` | Self-update to latest release |

## Environments

```bash
# Create environments
gurl env create dev --var "BASE_URL=https://dev.api.com" --secret "API_KEY=sk-dev-123"
gurl env create prod --var "BASE_URL=https://api.com" --secret "API_KEY=sk-prod-456"

# Run with an environment
gurl run "create-order" --env prod

# Switch default environment
gurl env use dev
```

Secrets are encrypted at rest with AES-256-GCM and never appear in logs or generated code.

## Import

```bash
gurl import openapi ./api.yaml --collection myapi
gurl import insomnia ./insomnia.json
gurl import bruno ./requests/
gurl import postman ./collection.json
gurl import har ./requests.har
```

## Code Generation

```bash
gurl codegen "create-order" --lang python
gurl codegen "create-order" --lang javascript
gurl codegen "create-order" --lang go
gurl codegen "create-order" --lang curl
```

## Configuration

`~/.config/gurl/config.toml` or `~/.gurlrc`:

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

## Agent Skills

For AI coding agents (Cursor, Claude Code, etc.):

```bash
npx skills add https://github.com/bsreeram08/gurl
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT
