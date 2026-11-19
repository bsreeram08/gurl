---
title: "Installation"
description: "Homebrew, install script, container image, pre-built binaries, or build from source."
weight: 1
---

valetkey ships as a single static binary with no runtime dependencies. Pick one install method, then [verify](#verify-installation) and continue to the [Quickstart]({{< relref "/docs/quickstart" >}}).

## Install

{{< tabs >}}

{{< tab name="Homebrew" >}}

macOS and Linux:

```bash
brew tap bsreeram08/valetkey https://github.com/bsreeram08/valetkey
brew install valetkey
```

{{< /tab >}}

{{< tab name="Script" >}}

```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/valetkey/master/scripts/install.sh | bash
```

Detects your platform, downloads the matching binary from the [latest release](https://github.com/bsreeram08/valetkey/releases/latest), and installs it to `/usr/local/bin`.

{{< /tab >}}

{{< tab name="Binary" >}}

Download from [GitHub Releases](https://github.com/bsreeram08/valetkey/releases/latest):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/bsreeram08/valetkey/releases/latest/download/valetkey-darwin-arm64.tar.gz
tar -xzf valetkey-darwin-arm64.tar.gz && sudo mv valetkey /usr/local/bin/valetkey

# Linux (x86_64)
curl -LO https://github.com/bsreeram08/valetkey/releases/latest/download/valetkey-linux-amd64.tar.gz
tar -xzf valetkey-linux-amd64.tar.gz && sudo mv valetkey /usr/local/bin/valetkey

# Linux (ARM64)
curl -LO https://github.com/bsreeram08/valetkey/releases/latest/download/valetkey-linux-arm64.tar.gz
tar -xzf valetkey-linux-arm64.tar.gz && sudo mv valetkey /usr/local/bin/valetkey
```

Each release also includes raw binaries and `SHA256SUMS`.

{{< /tab >}}

{{< tab name="Docker" >}}

Every release tag publishes a multi-arch image (`linux/amd64`, `linux/arm64`) to GitHub Container Registry: `ghcr.io/bsreeram08/valetkey:latest` (newest stable release; pre-releases don't move it) and one tag per release, matching the [GitHub release](https://github.com/bsreeram08/valetkey/releases) tag, e.g. `v0.7.0`.

```bash
docker run --rm ghcr.io/bsreeram08/valetkey:latest --version
docker run --rm -v "$PWD:/work" -w /work ghcr.io/bsreeram08/valetkey:latest list
docker run --rm ghcr.io/bsreeram08/valetkey:v0.7.0 --version     # pinned, reproducible
```

Keep the database, project files and key material across runs by mounting the host directories:

```bash
docker run --rm \
  -v "$PWD:/work" \
  -v "$HOME/.local/share/valetkey:/home/valetkey/.local/share/valetkey" \
  -v "$HOME/.config/valetkey:/home/valetkey/.config/valetkey" \
  -w /work \
  ghcr.io/bsreeram08/valetkey:latest list
```

The package is public. If GitHub still shows it as private, log in with a token that has `read:packages`: `echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin`.

{{< /tab >}}

{{< tab name="Source" >}}

Requires Go 1.25.0+.

```bash
git clone https://github.com/bsreeram08/valetkey.git
cd valetkey
go build -o valetkey ./cmd/valetkey
sudo mv valetkey /usr/local/bin/valetkey
```

{{< /tab >}}

{{< /tabs >}}

## Verify installation

```bash
valetkey --version
valetkey docs run      # prints the flags of `valetkey run`
```

## Shell completion

```bash
# Bash (~/.bashrc)
source <(valetkey completion bash)

# Zsh (~/.zshrc)
source <(valetkey completion zsh)

# Fish (~/.config/fish/config.fish)
valetkey completion fish | source
```

## Where valetkey keeps data

| Path | Contents |
|------|----------|
| `~/.local/share/valetkey/valetkey.db` | Saved requests, environments, collections and execution records (LevelDB) |
| `~/.local/share/valetkey/.secret-key` | Machine key for encrypted secrets (mode 0600) |
| `.valetkey/` in a project | Project-backed collections and environments, created by `valetkey init` |

Configuration is loaded from the first file found:

1. Path set in `$VALETKEY_CONFIG_PATH`
2. `.valetkeyrc` (current directory)
3. `~/.valetkeyrc`
4. `~/.config/valetkey/config.toml`

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
| Homebrew | `brew upgrade valetkey` |
| Install script or pre-built binary | `valetkey update` |
| Container image | `docker pull ghcr.io/bsreeram08/valetkey:latest` (or bump your pinned tag) |
| Source | `git pull && go build -o valetkey ./cmd/valetkey` |

`valetkey update` checks the latest release metadata and replaces the binary with the matching macOS or Linux asset for your CPU architecture. See [`valetkey update`]({{< relref "/cli/update" >}}).

## Next steps

Continue to the [Quickstart]({{< relref "/docs/quickstart" >}}). AI agents should run `valetkey docs` first; see [Agent mode]({{< relref "/ai/agent-mode" >}}).
