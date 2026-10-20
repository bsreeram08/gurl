# gurl

A terminal library of named HTTP requests, run against environments and collections, with variables filled in at send time.

[![Latest release](https://img.shields.io/github/v/release/bsreeram08/gurl)](https://github.com/bsreeram08/gurl/releases/latest)
[![Build](https://github.com/bsreeram08/gurl/actions/workflows/build.yml/badge.svg)](https://github.com/bsreeram08/gurl/actions/workflows/build.yml)
[![Container image](https://img.shields.io/badge/ghcr.io-bsreeram08%2Fgurl-blue?logo=docker)](https://github.com/bsreeram08/gurl/pkgs/container/gurl)

**Docs:** <https://bsreeram08.github.io/gurl/>

## Why gurl

- **Named, replayable requests.** Save a request once as `get-user` and run it by name instead of digging through shell history.
- **Secrets stay out of sight.** Secret values are encrypted at rest, masked in every output and execution record, and can be handed to other tools with `gurl exec` without being printed.
- **Fails loudly, not silently.** An unresolved `{{placeholder}}` stops the run instead of being sent. Every run is recorded, and `gurl diag <id>` shows what was sent and what changed since the last success.
- **Built for scripts and AI agents.** `gurl docs` is a single reference generated from the binary. `--json`, `--agent` and stable exit codes make the output easy to parse.
- **Local-first and git-friendly.** One binary, no account. Collections and environments can live in a `.gurl/` directory next to your code.

## Install

### Homebrew (macOS, Linux)

```bash
brew tap bsreeram08/gurl https://github.com/bsreeram08/gurl
brew install gurl
```

### Install script

```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/gurl/master/scripts/install.sh | bash
```

This installs to `~/.local/bin` (set `INSTALL_DIR` to change it).

### Binary download

Each [release](https://github.com/bsreeram08/gurl/releases/latest) has binaries for `darwin-arm64`, `darwin-amd64`, `linux-amd64` and `linux-arm64`, plus `SHA256SUMS`:

```bash
curl -Lo gurl https://github.com/bsreeram08/gurl/releases/latest/download/gurl-darwin-arm64
chmod +x gurl && sudo mv gurl /usr/local/bin/gurl
```

### Docker

The public, multi-arch (`linux/amd64`, `linux/arm64`) image is published on every release. `latest` follows the newest stable release, and each release also has its own tag, such as `v0.7.0`.

```bash
docker run --rm ghcr.io/bsreeram08/gurl:latest --version
docker run --rm -v "$PWD:/work" -w /work ghcr.io/bsreeram08/gurl:latest list
```

To keep the request database and secret keys between runs, mount them:

```bash
docker run --rm \
  -v "$PWD:/work" -w /work \
  -v "$HOME/.local/share/gurl:/home/gurl/.local/share/gurl" \
  -v "$HOME/.config/gurl:/home/gurl/.config/gurl" \
  ghcr.io/bsreeram08/gurl:latest list
```

### Build from source

Requires Go 1.25 or newer.

```bash
git clone https://github.com/bsreeram08/gurl.git && cd gurl
go build -o gurl ./cmd/gurl
```

### Upgrade

```bash
gurl update
```

## Quickstart

```bash
# 1. Save a request and run it
gurl save health https://api.example.com/health
gurl run health

# 2. Create an environment. KEY=- reads the value from stdin (no echo on a terminal)
gurl env create dev --var BASE_URL=https://dev.example.com --var API_KEY=- --secret API_KEY

# 3. Save a request with placeholders and run it with the environment
gurl save get-user '{{BASE_URL}}/users/{{id}}' -H 'Authorization: Bearer {{API_KEY}}'
gurl run get-user --env dev --var id=42

# 4. Preview without sending (secrets masked)
gurl run get-user --env dev --var id=42 --dry-run

# 5. Group requests into a collection and run them all
gurl collection create users-api --var 'API={{BASE_URL}}/v1'
gurl save list-users '{{API}}/users' --collection users-api
gurl collection run users-api --env dev

# 6. When something fails, gurl prints a record id. Inspect it:
#    request failed: HTTP 500 (id: 078b9896) — inspect with: gurl diag 078b9896
gurl diag 078b9896
```

## Feature tour

### Environments and secrets

```bash
gurl env create prod --var BASE_URL=https://api.example.com --var API_KEY=- --secret API_KEY
gurl env show prod              # API_KEY = *****
gurl env show prod --reveal     # prints the value (refused in agent mode)
gurl env switch prod            # make it the active environment
gurl env import prod --file .env     # add variables from a .env file

# Alias: one set of requests works across environments that name keys differently
gurl env set partner-beta 'API_KEY={{SA_KEY}}'
```

Secret values are encrypted at rest with AES-256-GCM. `env show`, `show`, `list --json`, `codegen` and `paste` mask secret values and credential headers unless you pass `--reveal`.

### Placeholders

```bash
# Typed placeholders keep a saved JSON body valid: "{{amount:num}}" sends 5, not "5"
gurl save create-order '{{BASE_URL}}/orders' -X POST \
  -d '{"amount":"{{amount:num}}","ref":"{{$uuid}}"}'
gurl run create-order --env dev --var amount=5
```

- `{{name}}` works in the URL, headers, body and auth params.
- Types: `num`, `bool`, `raw`.
- Dynamic values: `{{$uuid}}`, `{{$timestamp}}`, `{{$isoTimestamp}}`, `{{$randomInt(1,100)}}`, `{{$randomString(8)}}`, `{{$randomEmail}}`.
- An unresolved placeholder fails the run with a resolution problem instead of being sent.
- Scope precedence, lowest first: environment < collection < data row (`--data`) < CLI (`--var`) < runtime (values extracted or set by a script earlier in the run).

### Chaining, extract and `--persist`

```bash
gurl save login '{{BASE_URL}}/auth/login' -X POST -d '{"user":"{{user}}"}' \
  --extract token=jsonpath:$.token \
  --post-script "gurl.setNextRequest('profile')"

gurl run login --env dev --chain --persist --assert "status=200"

# Try extraction rules on a saved response without sending anything
gurl extract 'id=jsonpath:$.data.items[0].id' --file response.json
```

Extract methods are `jsonpath:`, `jq:` (an alias of jsonpath), `header:` and `regex:` (first capture group). Extracted values are dropped after the run unless you pass `--persist`, which writes each value back to its variable origin (the environment or the request's collection).

### Collections and data-driven runs

```bash
gurl collection run checkout --env staging --dry-run       # order and resolved URLs, nothing sent
gurl collection run checkout --env staging --data rows.csv  # one iteration per CSV/JSON row
gurl collection run checkout --env staging --bail --reporter junit --reporter-output out/

gurl init                          # create .gurl/ in this repo
gurl collection migrate checkout   # move the collection into .gurl/ to commit it
```

`.gurl/.gitignore` keeps the local `collection.key` files out of git. To share secrets, use `gurl collection export --passphrase` and `gurl collection import --passphrase`. In CI, set `GURL_IMPORT_PASSPHRASE` instead.

### Diagnosing failures and history

Every `run`, `collection run` and shell send is recorded, including attempts that never got a response. Dry runs are not recorded.

```bash
gurl history get-user              # record ids, statuses and durations
gurl diag 078b9896                 # any unique id prefix works; --full, --json
gurl diff get-user                 # compare the last two responses
gurl timeline --since 24h
```

`diag` shows the request as sent, where each variable came from, the response, assertions, and what changed since the last successful run. Secret values are masked before records are saved.

### exec

```bash
gurl exec --env prod -- node e2e.mjs   # the environment's variables arrive as process.env.API_KEY etc.
```

gurl never prints the values, and the command's exit code is passed through.

### Code generation, import and export

```bash
gurl codegen get-user --lang python    # go, python, javascript, curl; secrets masked
gurl import ./postman_collection.json  # OpenAPI, Insomnia, Postman, Bruno, HAR, .gurl (auto-detected)
gurl export --collection users-api -o users-api.gurl
echo 'curl -X POST https://api.example.com/x -d "{}"' | gurl detect --name x
```

### Other protocols

```bash
gurl graphql https://api.example.com/graphql --query '{ me { id } }'
gurl grpc localhost:50051 --list
gurl ws wss://echo.example.com --send '{"ping":1}'
gurl sse https://api.example.com/events --event update
```

Authentication is saved with the request: `--auth basic|bearer|apikey|oauth1|oauth2|awsv4|digest|ntlm` plus repeated `--auth-param KEY=VALUE`. Run `gurl auth list` and `gurl auth info <type>` for the parameters.

## For AI agents

**Start with `gurl docs`.** It prints a compact reference of every command, flag and concept, generated from the binary (under 32KB). `gurl docs run` narrows it to one command, and `gurl docs --json` returns it as structured data.

Turn on agent mode in any of three ways:

```bash
gurl --agent run get-user --env dev       # flag, in any position
GURL_AGENT=1 gurl run get-user --env dev  # environment variable
ln -s "$(command -v gurl)" ~/.local/bin/agentgurl && agentgurl run get-user --env dev
```

Agent mode:

- turns on `--json` wherever a command supports it, and `--no-input`, so a command never prompts and instead fails naming what to pass;
- refuses `--reveal`, `paste` and `codegen --clipboard` (exit 2);
- masks secret values in response bodies, response headers and errors as well as in requests;
- prints every error on stdout as `{"ok":false,"error":"..."}`.

`run --json` prints one object per execution (NDJSON for `--chain` and `--data`):

```json
{"id":"77f8d53b-...","short_id":"77f8d53b","request":"get-user","environment":"dev","ok":true,
 "method":"GET","url":"https://dev.example.com/users/7","status":200,"duration_ms":4,
 "body":{"id":"u-123","auth":"Bearer *****"}}
```

A failed run has `error`, `failure_phase` and, for resolution problems, a `problems` list (`missing`, `leftover_placeholder`, `bad_type`, `unknown_dynamic`, `auth`). Pass `short_id` to `gurl diag <id> --json`.

| Exit code | Meaning |
|-----------|---------|
| 0 | Success. A non-2xx response is still 0 unless `--fail` is passed (`ok` is false in JSON). |
| 1 | Error: not found, resolution problems, transport, script, extraction or assertion failure, non-2xx with `--fail`. |
| 2 | Usage: missing argument with `--no-input`, conflicting flags, `--reveal` in agent mode. |

`collection run` uses its own codes: 0 all passed, 1 a request failed (or was skipped, with `--ci`), 2 runtime error, 3 empty or missing collection. `exec` exits with the command's code.

Work with names, not secrets: have a human create environments with `KEY=-`, then let the agent run `gurl run <name> --env <env>`. Never pass live tokens through `--var`.

## Commands

| Command | What it does |
|---------|--------------|
| `init` | Create a `.gurl/` project directory |
| `save` | Save a request (flags or a raw curl string) |
| `run` | Execute a saved request |
| `list` | List saved requests |
| `show` | Show a saved request (secrets masked) |
| `edit` | Edit a saved request |
| `rename` / `delete` | Rename or remove a saved request |
| `detect` | Parse a curl command from stdin or a file |
| `history` | Execution history for a request |
| `timeline` | Execution timeline across all requests |
| `diff` | Compare responses for a request |
| `diag` | Inspect one execution and compare it with the last success |
| `env` | Manage environments |
| `exec` | Run a command with an environment's variables injected |
| `collection` | Manage and run collections |
| `sequence` | Set execution order within a collection |
| `extract` | Test extraction rules against a sample response |
| `auth` | List authentication types and their parameters |
| `codegen` | Generate go, python, javascript or curl code |
| `import` / `export` | Import from OpenAPI, Insomnia, Postman, Bruno, HAR; export to gurl format |
| `paste` | Copy a request as curl to the clipboard |
| `graphql` / `grpc` / `ws` / `sse` | GraphQL, gRPC, WebSocket and Server-Sent Events |
| `cookies` | Manage stored cookies |
| `shell` | Interactive shell |
| `docs` | Compact reference of every command and concept |
| `update` | Update gurl to the latest release |

Run `gurl <command> --help` or `gurl docs <command>` for flags.

## Links

- Documentation: <https://bsreeram08.github.io/gurl/>
- [CHANGELOG](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- License: MIT
