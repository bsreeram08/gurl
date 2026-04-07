# Gurl - Smart Curl Saver & API Companion

**GURL** = **G**url's **U**niversal **R**equest **L**ibrary

One-liner: Your curl history, organized and reusable with variable templates.

## Features
- Save and name curl requests
- Execute by name with variable substitution
- Auto-detect variables (IDs, UUIDs)
- Import from OpenAPI, Insomnia, Bruno, Postman, HAR
- Execution history and timeline
- TOML configuration
- Plugin system
- Agent-friendly API

## Installation

### One-liner Install (Recommended)
```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/gurl/master/scripts/install.sh | bash
```

### Pre-built Binaries
Download from [GitHub Releases](https://github.com/bsreeram08/gurl/releases):
```bash
# Linux (amd64)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-linux-amd64
chmod +x gurl-linux-amd64
sudo mv gurl-linux-amd64 /usr/local/bin/gurl

# macOS (Apple Silicon)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-darwin-arm64
chmod +x gurl-darwin-arm64
sudo mv gurl-darwin-arm64 /usr/local/bin/gurl

# Windows
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-windows-amd64.exe
```

### Build from Source
```bash
# Clone the repository
git clone https://github.com/bsreeram08/gurl.git
cd gurl

# Build
go build -o gurl ./cmd/gurl

# Install globally
sudo mv gurl /usr/local/bin/

# Verify
gurl --version
```

### Homebrew (macOS/Linux)
```bash
brew tap bsreeram08/gurl
brew install gurl
```

## Quick Start
```bash
# Save a request
gurl save "ping google" https://google.com

# Run it
gurl run "ping google"

# List all
gurl list
```

## All Commands
| Command | Description |
|---------|-------------|
| save | Save a curl request with a name |
| run | Execute a saved request |
| list | List all saved requests |
| detect | Parse curl from stdin and save |
| history | Show execution history |
| timeline | Global execution timeline |
| diff | Compare responses |
| edit | Edit a saved request |
| delete | Delete a saved request |
| rename | Rename a saved request |
| export | Export requests to JSON |
| import | Import from OpenAPI/Insomnia/etc |
| paste | Copy as curl command |
| collection | Manage collections |

## Configuration
See `.scurlrc` or `~/.gurlrc`:
```toml
[general]
history_depth = 100
auto_template = true

[output]
default_format = "auto"
```

## Import Formats
```bash
gurl import openapi ./api.yaml --collection myapi
gurl import insomnia ./insomnia.json
gurl import bruno ./requests/
gurl import postman ./collection.json
gurl import har ./requests.har
```

## Contributing
See CONTRIBUTING.md

## License
MIT
