---
title: "Installation"
description: "Homebrew, install script, container image, pre-built binaries, or build from source."
weight: 1
---

apigent ships as a single static binary with no runtime dependencies. Pick one install method, then [verify](#verify-installation) and continue to the [Quickstart]({{< relref "/docs/quickstart" >}}).

## Install

{{< tabs >}}

{{< tab name="Homebrew" >}}

macOS and Linux:

```bash
brew tap bsreeram08/apigent https://github.com/bsreeram08/apigent
brew install apigent
```

{{< /tab >}}

{{< tab name="Script" >}}

```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/apigent/master/scripts/install.sh | bash
```

Detects your platform, downloads the matching binary from the [latest release](https://github.com/bsreeram08/apigent/releases/latest), and installs it to `/usr/local/bin`.

{{< /tab >}}

{{< tab name="Binary" >}}

Download from [GitHub Releases](https://github.com/bsreeram08/apigent/releases/latest):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/bsreeram08/apigent/releases/latest/download/apigent-darwin-arm64.tar.gz
tar -xzf apigent-darwin-arm64.tar.gz && sudo mv apigent /usr/local/bin/apigent

# Linux (x86_64)
curl -LO https://github.com/bsreeram08/apigent/releases/latest/download/apigent-linux-amd64.tar.gz
tar -xzf apigent-linux-amd64.tar.gz && sudo mv apigent /usr/local/bin/apigent

# Linux (ARM64)
curl -LO https://github.com/bsreeram08/apigent/releases/latest/download/apigent-linux-arm64.tar.gz
tar -xzf apigent-linux-arm64.tar.gz && sudo mv apigent /usr/local/bin/apigent
```

Each release also includes raw binaries and `SHA256SUMS`.

{{< /tab >}}

{{< tab name="Docker" >}}

Every release tag publishes a multi-arch image (`linux/amd64`, `linux/arm64`) to GitHub Container Registry: `ghcr.io/bsreeram08/apigent:latest` (newest stable release; pre-releases don't move it) and one tag per release, matching the [GitHub release](https://github.com/bsreeram08/apigent/releases) tag, e.g. `v0.7.0`.

```bash
docker run --rm ghcr.io/bsreeram08/apigent:latest --version
docker run --rm -v "$PWD:/work" -w /work ghcr.io/bsreeram08/apigent:latest list
docker run --rm ghcr.io/bsreeram08/apigent:v0.7.0 --version     # pinned, reproducible
```

Keep the database, project files and key material across runs by mounting the host directories:

```bash
docker run --rm \
  -v "$PWD:/work" \
  -v "$HOME/.local/share/apigent:/home/apigent/.local/share/apigent" \
  -v "$HOME/.config/apigent:/home/apigent/.config/apigent" \
  -w /work \
  ghcr.io/bsreeram08/apigent:latest list
```

The package is public. If GitHub still shows it as private, log in with a token that has `read:packages`: `echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin`.

{{< /tab >}}

{{< tab name="Source" >}}

Requires Go 1.25.0+.

```bash
git clone https://github.com/bsreeram08/apigent.git
cd apigent
go build -o apigent ./cmd/apigent
sudo mv apigent /usr/local/bin/apigent
```

{{< /tab >}}

{{< /tabs >}}

## Verify installation

```bash
apigent --version
apigent docs run      # prints the flags of `apigent run`
```

## Shell completion

```bash
# Bash (~/.bashrc)
source <(apigent completion bash)

# Zsh (~/.zshrc)
source <(apigent completion zsh)

# Fish (~/.config/fish/config.fish)
apigent completion fish | source
```

## Where apigent keeps data

| Path | Contents |
|------|----------|
| `~/.local/share/apigent/apigent.db` | Saved requests, environments, collections and execution records (LevelDB) |
| `~/.local/share/apigent/.secret-key` | Machine key for encrypted secrets (mode 0600) |
| `.apigent/` in a project | Project-backed collections and environments, created by `apigent init` |

Configuration is loaded from the first file found:

1. Path set in `$APIGENT_CONFIG_PATH`
2. `.apigentrc` (current directory)
3. `~/.apigentrc`
4. `~/.config/apigent/config.toml`

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
| Homebrew | `brew upgrade apigent` |
| Install script or pre-built binary | `apigent update` |
| Container image | `docker pull ghcr.io/bsreeram08/apigent:latest` (or bump your pinned tag) |
| Source | `git pull && go build -o apigent ./cmd/apigent` |

`apigent update` checks the latest release metadata and replaces the binary with the matching macOS or Linux asset for your CPU architecture. See [`apigent update`]({{< relref "/cli/update" >}}).

## Next steps

Continue to the [Quickstart]({{< relref "/docs/quickstart" >}}). AI agents should run `apigent docs` first; see [Agent mode]({{< relref "/ai/agent-mode" >}}).
