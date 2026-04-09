---
title: "Installation"
weight: 1
---

## Prerequisites

Gurl requires no runtime dependencies — it ships as a single static binary. Go is only needed if you are building from source (Go 1.21+).

## Homebrew (macOS / Linux)

```bash
brew tap bsreeram08/gurl https://github.com/bsreeram08/gurl
brew install gurl
```

Update with:

```bash
brew upgrade gurl
```

## One-liner Install

```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/gurl/master/scripts/install.sh | bash
```

This detects your platform, downloads the correct binary from the [latest release](https://github.com/bsreeram08/gurl/releases/latest), and installs it to `/usr/local/bin`.

## Pre-built Binaries

Download directly from [GitHub Releases](https://github.com/bsreeram08/gurl/releases/latest):

### macOS (Apple Silicon)

```bash
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-darwin-arm64.tar.gz
tar -xzf gurl-darwin-arm64.tar.gz
sudo mv gurl /usr/local/bin/gurl
```

### Linux (x86_64)

```bash
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-linux-amd64.tar.gz
tar -xzf gurl-linux-amd64.tar.gz
sudo mv gurl /usr/local/bin/gurl
```

### Linux (ARM64)

```bash
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-linux-arm64.tar.gz
tar -xzf gurl-linux-arm64.tar.gz
sudo mv gurl /usr/local/bin/gurl
```

## Install via Go

For users with Go 1.21+ installed:

```bash
go install github.com/sreeram/gurl/cmd/gurl@latest
```

This installs the `gurl` binary to `$GOPATH/bin` or `$HOME/go/bin`.

## Build from Source

```bash
git clone https://github.com/bsreeram08/gurl.git
cd gurl
go build -o gurl ./cmd/gurl
sudo mv gurl /usr/local/bin/gurl
```

## Shell Completion

Enable tab completion for faster command entry.

### Bash

Add to your `~/.bashrc`:

```bash
source <(gurl completion bash)
```

### Zsh

Add to your `~/.zshrc`:

```bash
source <(gurl completion zsh)
```

### Fish

Add to your `~/.config/fish/config.fish`:

```bash
gurl completion fish | source
```

## First-run Setup

On first launch, Gurl creates the local data directory:

```
~/.local/share/gurl/
```

This stores:

- `gurl.db` — Request collections, environments, and history (LevelDB)
- `plugins/` — Custom middleware and formatters

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

```bash
gurl --version
```

You should see the version number and build information.

## Update

Gurl can self-update to the latest release:

```bash
gurl update
```

## Next Steps

Continue to the [Quick Start guide](/docs/quickstart/) to save and run your first request.
