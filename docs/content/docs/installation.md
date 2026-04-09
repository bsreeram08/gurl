---
title: "Installation"
weight: 1
---

## Prerequisites

Gurl requires Go 1.21 or later to build from source. Most users install via Homebrew or a pre-built binary.

## Install via Homebrew

The recommended installation method for macOS and Linux:

```bash
brew tap bsreeram08/gurl
brew install gurl
```

Update with:

```bash
brew upgrade gurl
```

## Install via Script

One-liner install using the official install script:

```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/gurl/master/scripts/install.sh | bash
```

## Install from Source

Clone the repository and build:

```bash
git clone https://github.com/bsreeram08/gurl.git
cd gurl
go build -o gurl ./cmd/gurl
```

The binary is created as `./gurl` in the project root. Move it to a directory in your PATH:

```bash
sudo mv ./gurl /usr/local/bin/gurl
```

## Binary Releases

Download pre-built binaries from the [GitHub Releases page](https://github.com/bsreeram08/gurl/releases).

Each release includes:

- macOS (Apple Silicon and Intel)
- Linux (amd64 and ARM)
- Windows (amd64)

Download the appropriate binary, make it executable, and move it to your PATH:

```bash
# Linux (amd64)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-linux-amd64
chmod +x gurl-linux-amd64
sudo mv gurl-linux-amd64 /usr/local/bin/gurl

# macOS (Apple Silicon)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-darwin-arm64
chmod +x gurl-darwin-arm64
sudo mv gurl-darwin-arm64 /usr/local/bin/gurl
```

## First-run Setup

On first launch, Gurl creates the local data directory:

```
~/.local/share/gurl/
```

This stores:

- `gurl.db` - Request collections and environments (LevelDB)
- `plugins/` - Custom middleware and formatters

Configuration is loaded from the first file found in this order:

1. Path set in `$GURL_CONFIG_PATH`
2. `.gurlrc` (current directory)
3. `~/.gurlrc`
4. `~/.config/gurl/config.toml`

Example config file:

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

## Verify Installation

Check that Gurl is installed correctly:

```bash
gurl --version
```

You should see the version number and build information.

## Next Steps

Now that Gurl is installed, continue to the [Quick Start guide](/docs/quickstart/) to save and run your first request.
