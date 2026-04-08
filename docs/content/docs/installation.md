---
title: "Installation"
weight: 1
---

## Prerequisites

Gurl requires Go 1.21 or later. Most users install via Homebrew, but there are other options.

## Install via Homebrew

The recommended installation method for macOS and Linux:

```bash
brew install gurl
```

Update with:

```bash
brew upgrade gurl
```

## Install via Go

For users with Go installed:

```bash
go install github.com/sreeram/terminal-curl/cmd/gurl@latest
```

This installs the `gurl` binary to `$GOPATH/bin` or `$HOME/go/bin`.

## Install from Source

Clone the repository and build:

```bash
git clone https://github.com/sreeram/terminal-curl.git
cd terminal-curl
go build ./cmd/gurl
```

The binary is created as `./gurl` in the project root. Move it to a directory in your PATH:

```bash
mv ./gurl /usr/local/bin/gurl
```

## Binary Releases

Download pre-built binaries from the [GitHub Releases page](https://github.com/sreeram/terminal-curl/releases).

Each release includes:

- macOS (Apple Silicon and Intel)
- Linux (x86 and ARM)
- Windows

Extract the archive and move the binary to your PATH.

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

- `gurl.db` - Request collections and environments (LMDB)
- `logs/` - Request and response logs
- `plugins/` - Custom middleware and formatters

Configuration is stored separately:

- `~/.config/gurl/config.toml` (preferred)
- `~/.gurlrc` (fallback)

> [!TIP]
> Create a config file to set defaults for your environment:

```toml
[defaults]
timeout = 30
follow_redirects = true
verify_ssl = true

[output]
format = "table"
color = true
```

## Verify Installation

Check that Gurl is installed correctly:

```bash
gurl --version
```

You should see the version number and build information.

## Next Steps

Now that Gurl is installed, continue to the [Quick Start guide](/docs/quickstart/) to save and run your first request.
