---
title: "Installation"
description: "Homebrew, install script, container image, pre-built binaries, or build from source."
weight: 1
---

gurl ships as a single static binary with no runtime dependencies. Pick one install method, then [verify](#verify-installation) and continue to the [Quickstart]({{< relref "/docs/quickstart" >}}).

## Install

{{< tabs >}}

{{< tab name="Homebrew" >}}

macOS and Linux:

```bash
brew tap bsreeram08/gurl https://github.com/bsreeram08/gurl
brew install gurl
```

{{< /tab >}}

{{< tab name="Script" >}}

```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/gurl/master/scripts/install.sh | bash
```

Detects your platform, downloads the matching binary from the [latest release](https://github.com/bsreeram08/gurl/releases/latest), and installs it to `/usr/local/bin`.

{{< /tab >}}

{{< tab name="Binary" >}}

Download from [GitHub Releases](https://github.com/bsreeram08/gurl/releases/latest):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-darwin-arm64.tar.gz
tar -xzf gurl-darwin-arm64.tar.gz && sudo mv gurl /usr/local/bin/gurl

# Linux (x86_64)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-linux-amd64.tar.gz
tar -xzf gurl-linux-amd64.tar.gz && sudo mv gurl /usr/local/bin/gurl

# Linux (ARM64)
curl -LO https://github.com/bsreeram08/gurl/releases/latest/download/gurl-linux-arm64.tar.gz
tar -xzf gurl-linux-arm64.tar.gz && sudo mv gurl /usr/local/bin/gurl
```

Each release also includes raw binaries and `SHA256SUMS`.

{{< /tab >}}

{{< tab name="Docker" >}}

Every release tag publishes a multi-arch image (`linux/amd64`, `linux/arm64`) to GitHub Container Registry: `ghcr.io/bsreeram08/gurl:latest` (newest stable release; pre-releases don't move it) and one tag per release, matching the [GitHub release](https://github.com/bsreeram08/gurl/releases) tag, e.g. `v0.7.0`.

```bash
docker run --rm ghcr.io/bsreeram08/gurl:latest --version
docker run --rm -v "$PWD:/work" -w /work ghcr.io/bsreeram08/gurl:latest list
docker run --rm ghcr.io/bsreeram08/gurl:v0.7.0 --version     # pinned, reproducible
```

Keep the database, project files and key material across runs by mounting the host directories:

```bash
docker run --rm \
  -v "$PWD:/work" \
  -v "$HOME/.local/share/gurl:/home/gurl/.local/share/gurl" \
  -v "$HOME/.config/gurl:/home/gurl/.config/gurl" \
  -w /work \
  ghcr.io/bsreeram08/gurl:latest list
```

The package is public. If GitHub still shows it as private, log in with a token that has `read:packages`: `echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin`.

{{< /tab >}}

{{< tab name="Source" >}}

Requires Go 1.25.0+.

```bash
git clone https://github.com/bsreeram08/gurl.git
cd gurl
go build -o gurl ./cmd/gurl
sudo mv gurl /usr/local/bin/gurl
```

{{< /tab >}}

{{< /tabs >}}

## Verify installation

```bash
gurl --version
gurl docs run      # prints the flags of `gurl run`
```

## Shell completion

```bash
# Bash (~/.bashrc)
source <(gurl completion bash)

# Zsh (~/.zshrc)
source <(gurl completion zsh)

# Fish (~/.config/fish/config.fish)
gurl completion fish | source
```

## Where gurl keeps data

| Path | Contents |
|------|----------|
| `~/.local/share/gurl/gurl.db` | Saved requests, environments, collections and execution records (LevelDB) |
| `~/.local/share/gurl/.secret-key` | Machine key for encrypted secrets (mode 0600) |
| `.gurl/` in a project | Project-backed collections and environments, created by `gurl init` |

Configuration is loaded from the first file found:

1. Path set in `$GURL_CONFIG_PATH`
2. `.gurlrc` (current directory)
3. `~/.gurlrc`
4. `~/.config/gurl/config.toml`

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

## Update

| Installed with | Update with |
|----------------|-------------|
| Homebrew | `brew upgrade gurl` |
| Install script or pre-built binary | `gurl update` |
| Container image | `docker pull ghcr.io/bsreeram08/gurl:latest` (or bump your pinned tag) |
| Source | `git pull && go build -o gurl ./cmd/gurl` |

`gurl update` checks the latest release metadata and replaces the binary with the matching macOS or Linux asset for your CPU architecture. See [`gurl update`]({{< relref "/cli/update" >}}).

## Next steps

Continue to the [Quickstart]({{< relref "/docs/quickstart" >}}). AI agents should run `gurl docs` first; see [Agent mode]({{< relref "/ai/agent-mode" >}}).
