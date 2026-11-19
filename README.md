# apigent

**Saved API requests your AI agents can run without ever seeing the keys.**

apigent is a local-first library of named HTTP requests, like Insomnia or Postman for the terminal, built so that AI agents can use your APIs safely. You save requests and keep secrets in environments. An agent runs a request by name, through the CLI or as an MCP tool, and gets back JSON with every secret masked. The agent gets the API, never the keys.

> **Formerly gurl.** The `gurl` command, `GURL_*` variables, `~/.local/share/gurl` and `.gurl/` projects all keep working. See [Coming from gurl](#coming-from-gurl).

[![Latest release](https://img.shields.io/github/v/release/bsreeram08/apigent)](https://github.com/bsreeram08/apigent/releases/latest)
[![Build](https://github.com/bsreeram08/apigent/actions/workflows/build.yml/badge.svg)](https://github.com/bsreeram08/apigent/actions/workflows/build.yml)
[![Container image](https://img.shields.io/badge/ghcr.io-bsreeram08%2Fapigent-blue?logo=docker)](https://github.com/bsreeram08/apigent/pkgs/container/apigent)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Docs:** <https://bsreeram08.github.io/apigent/>

## Why apigent

- **Agents get the request, not the key.** A saved request is the unit of permission: the agent can run the requests you saved, against the environment you chose, and nothing else. Secrets are filled in inside apigent and masked everywhere the agent can look, including response bodies that echo them back.
- **An MCP server in the binary.** `apigent mcp --env dev` turns every saved request into a tool for Claude, Cursor or any MCP client. A tool call can only fill placeholders your environment leaves empty, so an agent cannot point `BASE_URL` at another host and walk off with your token.
- **Fails loudly, not silently.** An unresolved `{{placeholder}}` stops the run instead of being sent. Every run is recorded, and `apigent diag <id>` shows what was sent and what changed since the last success.
- **Built for agents and scripts.** `apigent docs` is one compact reference generated from the binary. `--agent`, `--json` and stable exit codes make every command safe to drive.
- **Local-first and git-friendly.** One binary, no account, no hosted proxy. Collections and environments can live in a `.apigent/` directory next to your code.

## Give an agent your API in one minute

```bash
apigent env create dev --var BASE_URL=https://api.example.com --var API_KEY=- --secret API_KEY
apigent save get-user '{{BASE_URL}}/users/{{id}}' -H 'Authorization: Bearer {{API_KEY}}'
```

Then add the server to your MCP client:

```json
{ "mcpServers": { "my-api": { "command": "apigent", "args": ["mcp", "--env", "dev"] } } }
```

The agent sees one tool, `get-user`, with a single argument, `id`. It never sees `API_KEY`, and it cannot change `BASE_URL`. Use `--collection <name>` to expose only part of your library.

## Install

### Homebrew (macOS, Linux)

```bash
brew tap bsreeram08/apigent https://github.com/bsreeram08/apigent
brew install apigent
```

### Install script

```bash
curl -sL https://raw.githubusercontent.com/bsreeram08/apigent/master/scripts/install.sh | bash
```

This installs to `~/.local/bin` (set `INSTALL_DIR` to change it).

### Binary download

Each [release](https://github.com/bsreeram08/apigent/releases/latest) has binaries for `darwin-arm64`, `darwin-amd64`, `linux-amd64` and `linux-arm64`, plus `SHA256SUMS`:

```bash
curl -Lo apigent https://github.com/bsreeram08/apigent/releases/latest/download/apigent-darwin-arm64
chmod +x apigent && sudo mv apigent /usr/local/bin/apigent
```

### Docker

The public, multi-arch (`linux/amd64`, `linux/arm64`) image is published on every release. `latest` follows the newest stable release, and each release also has its own tag, such as `v0.7.0`.

```bash
docker run --rm ghcr.io/bsreeram08/apigent:latest --version
docker run --rm -v "$PWD:/work" -w /work ghcr.io/bsreeram08/apigent:latest list
```

To keep the request database and secret keys between runs, mount them:

```bash
docker run --rm \
  -v "$PWD:/work" -w /work \
  -v "$HOME/.local/share/apigent:/home/apigent/.local/share/apigent" \
  -v "$HOME/.config/apigent:/home/apigent/.config/apigent" \
  ghcr.io/bsreeram08/apigent:latest list
```

### Build from source

Requires Go 1.25 or newer.

```bash
git clone https://github.com/bsreeram08/apigent.git && cd apigent
go build -o apigent ./cmd/apigent
```

### Upgrade

```bash
apigent update
```

## Quickstart

```bash
# 1. Save a request and run it
apigent save health https://api.example.com/health
apigent run health

# 2. Create an environment. KEY=- reads the value from stdin (no echo on a terminal)
apigent env create dev --var BASE_URL=https://dev.example.com --var API_KEY=- --secret API_KEY

# 3. Save a request with placeholders and run it with the environment
apigent save get-user '{{BASE_URL}}/users/{{id}}' -H 'Authorization: Bearer {{API_KEY}}'
apigent run get-user --env dev --var id=42

# 4. Preview without sending (secrets masked)
apigent run get-user --env dev --var id=42 --dry-run

# 5. Group requests into a collection and run them all
apigent collection create users-api --var 'API={{BASE_URL}}/v1'
apigent save list-users '{{API}}/users' --collection users-api
apigent collection run users-api --env dev

# 6. When something fails, apigent prints a record id. Inspect it:
#    request failed: HTTP 500 (id: 078b9896) — inspect with: apigent diag 078b9896
apigent diag 078b9896
```

## Feature tour

### Environments and secrets

```bash
apigent env create prod --var BASE_URL=https://api.example.com --var API_KEY=- --secret API_KEY
apigent env show prod              # API_KEY = *****
apigent env show prod --reveal     # prints the value (refused in agent mode)
apigent env switch prod            # make it the active environment
apigent env import prod --file .env     # add variables from a .env file

# Alias: one set of requests works across environments that name keys differently
apigent env set partner-beta 'API_KEY={{SA_KEY}}'
```

Secret values are encrypted at rest with AES-256-GCM. `env show`, `show`, `list --json`, `cookies list`, `codegen` and `paste` mask secret values, credential headers and cookie values unless you pass `--reveal`.

### Placeholders

```bash
# Typed placeholders keep a saved JSON body valid: "{{amount:num}}" sends 5, not "5"
apigent save create-order '{{BASE_URL}}/orders' -X POST \
  -d '{"amount":"{{amount:num}}","ref":"{{$uuid}}"}'
apigent run create-order --env dev --var amount=5
```

- `{{name}}` works in the URL, headers, body and auth params.
- Types: `num`, `bool`, `raw`.
- Dynamic values: `{{$uuid}}`, `{{$timestamp}}`, `{{$isoTimestamp}}`, `{{$randomInt(1,100)}}`, `{{$randomString(8)}}`, `{{$randomEmail}}`.
- An unresolved placeholder fails the run with a resolution problem instead of being sent.
- Scope precedence, lowest first: environment < collection < data row (`--data`) < CLI (`--var`) < runtime (values extracted or set by a script earlier in the run).

### Chaining, extract and `--persist`

```bash
apigent save login '{{BASE_URL}}/auth/login' -X POST -d '{"user":"{{user}}"}' \
  --extract token=jsonpath:$.token \
  --post-script "apigent.setNextRequest('profile')"

apigent run login --env dev --chain --persist --assert "status=200"

# Try extraction rules on a saved response without sending anything
apigent extract 'id=jsonpath:$.data.items[0].id' --file response.json
```

Extract methods are `jsonpath:`, `jq:` (an alias of jsonpath), `header:` and `regex:` (first capture group). Extracted values are dropped after the run unless you pass `--persist`, which writes each value back to its variable origin (the environment or the request's collection).

### Collections and data-driven runs

```bash
apigent collection run checkout --env staging --dry-run       # order and resolved URLs, nothing sent
apigent collection run checkout --env staging --data rows.csv  # one iteration per CSV/JSON row
apigent collection run checkout --env staging --bail --reporter junit --reporter-output out/

apigent init                          # create .apigent/ in this repo
apigent collection migrate checkout   # move the collection into .apigent/ to commit it
```

`.apigent/.gitignore` keeps the local `collection.key` files out of git. To share secrets, use `apigent collection export --passphrase` and `apigent collection import --passphrase`. In CI, set `APIGENT_IMPORT_PASSPHRASE` instead.

### Diagnosing failures and history

Every `run`, `collection run` and shell send is recorded, including attempts that never got a response. Dry runs are not recorded.

```bash
apigent history get-user              # record ids, statuses and durations
apigent diag 078b9896                 # any unique id prefix works; --full, --json
apigent diff get-user                 # compare the last two responses
apigent timeline --since 24h
```

`diag` shows the request as sent, where each variable came from, the response, assertions, and what changed since the last successful run. Secret values are masked before records are saved.

### exec

```bash
apigent exec --env prod -- node e2e.mjs   # the environment's variables arrive as process.env.API_KEY etc.
```

apigent never prints the values, and the command's exit code is passed through.

### Code generation, import and export

```bash
apigent codegen get-user --lang python    # go, python, javascript, curl; secrets masked
apigent import ./postman_collection.json  # OpenAPI, Insomnia, Postman, Bruno, HAR, .gurl (auto-detected)
apigent export --collection users-api -o users-api.gurl
apigent share -c users-api --to github:alice -o users-api.gurl.age   # encrypted to alice's SSH keys
apigent receive users-api.gurl.age                                  # alice: decrypt, preview, import
echo 'curl -X POST https://api.example.com/x -d "{}"' | apigent detect --name x
```

### Other protocols

```bash
apigent graphql https://api.example.com/graphql --query '{ me { id } }'
apigent grpc localhost:50051 --list
apigent ws wss://echo.example.com --send '{"ping":1}'
apigent sse https://api.example.com/events --event update
```

Authentication is saved with the request: `--auth basic|bearer|apikey|oauth1|oauth2|awsv4|digest|ntlm` plus repeated `--auth-param KEY=VALUE`. Run `apigent auth list` and `apigent auth info <type>` for the parameters.

## For AI agents

**Start with `apigent docs`.** It prints a compact reference of every command, flag and concept, generated from the binary (under 32KB). `apigent docs run` narrows it to one command, and `apigent docs --json` returns it as structured data.

Turn on agent mode in any of three ways:

```bash
apigent --agent run get-user --env dev       # flag, in any position
APIGENT_AGENT=1 apigent run get-user --env dev  # environment variable
ln -s "$(command -v apigent)" ~/.local/bin/agentapigent && agentapigent run get-user --env dev
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

A failed run has `error`, `failure_phase` and, for resolution problems, a `problems` list (`missing`, `leftover_placeholder`, `bad_type`, `unknown_dynamic`, `auth`). Pass `short_id` to `apigent diag <id> --json`.

| Exit code | Meaning |
|-----------|---------|
| 0 | Success. A non-2xx response is still 0 unless `--fail` is passed (`ok` is false in JSON). |
| 1 | Error: not found, resolution problems, transport, script, extraction or assertion failure, non-2xx with `--fail`. |
| 2 | Usage: missing argument with `--no-input`, conflicting flags, `--reveal` in agent mode. |

`collection run` uses its own codes: 0 all passed, 1 a request failed (or was skipped, with `--ci`), 2 runtime error, 3 empty or missing collection. `exec` exits with the command's code.

Work with names, not secrets: have a human create environments with `KEY=-`, then let the agent run `apigent run <name> --env <env>`. Never pass live tokens through `--var`.

## Commands

| Command | What it does |
|---------|--------------|
| `init` | Create a `.apigent/` project directory |
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
| `import` / `export` | Import from OpenAPI, Insomnia, Postman, Bruno, HAR; export to the `.gurl` file format |
| `share` / `receive` | Encrypt requests to someone's public key (`github:<user>`, SSH, age) and import them |
| `paste` | Copy a request as curl to the clipboard |
| `graphql` / `grpc` / `ws` / `sse` | GraphQL, gRPC, WebSocket and Server-Sent Events |
| `cookies` | Manage stored cookies |
| `shell` | Interactive shell |
| `mcp` | Serve saved requests as MCP tools over stdio |
| `migrate` | Move data and projects from gurl locations to apigent ones |
| `docs` | Compact reference of every command and concept |
| `update` | Update apigent to the latest release |

Run `apigent <command> --help` or `apigent docs <command>` for flags.

## Coming from gurl

apigent is the new name of gurl. Nothing has to change on day one:

| Before | Now | The old one still works? |
|--------|-----|--------------------------|
| `gurl`, `agentgurl` | `apigent`, `agentapigent` | Yes: Homebrew and `install.sh` keep `gurl` as a link to `apigent` |
| `GURL_AGENT`, `GURL_DB_PATH`, ... | `APIGENT_AGENT`, `APIGENT_DB_PATH`, ... | Yes, read when the new one is unset |
| `~/.local/share/gurl`, `~/.config/gurl`, `~/.gurl/plugins` | `~/.local/share/apigent`, `~/.config/apigent`, `~/.apigent/plugins` | Yes, used until you migrate |
| `.gurl/` project directory, `.gurlrc` | `.apigent/`, `.apigentrc` | Yes, discovered when there is no `.apigent/` |
| `gurl.setVar(...)` in scripts | `apigent.setVar(...)` | Yes, both names are the same object |
| `ghcr.io/bsreeram08/gurl` | `ghcr.io/bsreeram08/apigent` | Published under both names for now |

When you are ready, `apigent migrate --dry-run` shows what would move and `apigent migrate` moves it (data, config, plugins and the current project's `.gurl/`). It renames directories without copying or deleting anything, and skips any whose new name already exists. After migrating, releases older than the rename no longer find your data. If you mount the data directory into the container, mount `~/.local/share/apigent` at `/home/apigent/.local/share/apigent`.

Unchanged on purpose: exported `.gurl` files, `gurl-share` bundles and their signatures, and encrypted values already on disk. They all keep working in both directions.

## Links

- Documentation: <https://bsreeram08.github.io/apigent/>
- [CHANGELOG](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- License: [MIT](LICENSE)
