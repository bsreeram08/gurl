---
title: "apigent collection"
description: "Manage collections, their variables, and collection runs"
---


Group related requests into a collection with its own variables and encrypted secrets, run the whole collection, and share it through git or passphrase-encrypted exports.

## Synopsis

```bash
apigent collection <subcommand> [flags] [arguments]
```

Collection variables sit above environment variables in scope precedence (environment < collection < data row < CLI < runtime), and may reference environment keys (`API_URL={{BASE_URL}}/v1`).

## Subcommands

### run

Run every request in a collection in sequence order.

```bash
apigent collection run <name> [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--env` | `-e` | none | Environment to use |
| `--bail` | `-b` | `false` | Stop on the first failure |
| `--assert-bail` | | `false` | Stop on the first assertion failure only |
| `--persist` | | `false` | Write extracted and script-set variables back to their variable origin |
| `--dry-run` | | `false` | Preview the run without sending requests |
| `--verbose` | | `false` | Print request and response headers and bodies for each request |
| `--iterations` | `-n` | `1` | Number of times to run the collection |
| `--delay` | `-d` | `0s` | Delay between requests (e.g., `100ms`, `1s`) |
| `--var` | `-v` | none | Variable as `key=value` (repeatable) |
| `--data` | `-D` | none | CSV or JSON data file for data-driven iteration |
| `--reporter` | `-R` | none | Reporter(s): `junit`, `json`, `html`, `console` |
| `--reporter-output` | `-O` | none | Output directory for reporter files |
| `--ci` | | `false` | CI mode: treat skipped requests as failures (exit 1) |

Each execution is recorded. For every failed request, the id is printed on stderr: `[FAIL] failing: HTTP 500 (id: 51be33a5) — inspect with: apigent diag 51be33a5`. `--persist` and `--dry-run` cannot be used together.

Aliases: `r`

### show

Show a collection's variables (secrets masked) and requests.

```bash
apigent collection show <name> [--json]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--json` | `-j` | `false` | JSON output (secrets masked): `{"name","id","variables":{},"secrets":[],"requests":[{"name","method","url"}]}` |

Aliases: `view`, `info`

### list

List all collections with request and variable counts.

```bash
apigent collection list [--json]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--json` | `-j` | `false` | JSON output: `[{"name","requests","variables","updated"}]` |

Aliases: `ls`, `l`

### create

Create a collection, optionally with variables and secrets.

```bash
apigent collection create <name> [--var KEY=VALUE ...] [--secret KEY=VALUE ...]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--var` | `-v` | none | Variable as `KEY=VALUE` (repeatable) |
| `--secret` | `-s` | none | Secret variable as `KEY=VALUE` (repeatable), encrypted at rest |

Aliases: `add`, `new`

### set-var

Set one variable in a collection.

```bash
apigent collection set-var <collection> KEY=VALUE [--secret]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--secret` | | `false` | Store the variable as a secret |

Aliases: `set`

### unset-var

Remove one variable from a collection.

```bash
apigent collection unset-var <collection> KEY
```

None.

Aliases: `unset`

### migrate

Move a database-backed collection into project file storage (`.apigent/collections/<name>/`).

```bash
apigent collection migrate <name> [--passphrase <pass>]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project-dir` | | none | Project root for `.apigent` file storage |
| `--passphrase` | | none | Write collection secrets with passphrase-derived encryption |

With `--passphrase`, secrets are written to `collection.json` with passphrase-derived encryption and no local `collection.key` is left behind; teammates unlock with the passphrase.

### unlock

Unlock a passphrase-protected file collection after cloning. The derived key is cached in the OS keychain.

```bash
apigent collection unlock <name> [--passphrase <pass>]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--passphrase` | | none | Collection import passphrase |

### export

Export a collection with passphrase-encrypted secrets.

```bash
apigent collection export <name> --passphrase <pass> [--output <file>]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | stdout | Output file |
| `--passphrase` | | none | Passphrase for encrypting collection secrets |

### import

Import a collection export, a collection directory, or `.env` variables.

```bash
apigent collection import <file-or-directory> [--passphrase <pass>] [--force]
apigent collection import <collection> --file .env
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Overwrite existing collection/request records |
| `--file` | | none | Path to a `.env` file for collection variable import |
| `--passphrase` | | none | Passphrase for decrypting collection secrets (or set `APIGENT_IMPORT_PASSPHRASE`) |

`.env` imports create the collection if needed and store values as plain (non-secret) variables. Without a passphrase and with `--no-input`, an encrypted import fails with `passphrase required: pass --passphrase or set APIGENT_IMPORT_PASSPHRASE`.

### remove

Remove a collection label but keep its requests.

```bash
apigent collection remove <name>
```

None.

Aliases: `rm`

### delete

Delete a collection and all of its requests. Interactive terminals ask for confirmation; with `--no-input` (or `--json`/`--agent`) it fails unless `--force` is passed.

```bash
apigent collection delete <name> [--force]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | | `false` | Delete without interactive confirmation |

Aliases: `del`

### rename

Rename a collection.

```bash
apigent collection rename <old-name> <new-name>
```

None.

Aliases: `mv`, `ren`

## Aliases

- `collections`
- `col`
- `c`

## Examples

### Create a collection with a secret and run it

```bash
apigent collection create payments-api \
  --var BASE_URL=https://api.example.com \
  --secret API_KEY=sk-live-...

apigent save "list-payments" '{{BASE_URL}}/payments' \
  --collection payments-api \
  --auth bearer --auth-param token='{{API_KEY}}'

apigent collection run payments-api --env staging --bail
```

### Preview a run, then persist what it extracts

```bash
apigent collection run checkout-flow --env staging --dry-run
apigent collection run checkout-flow --env staging --persist
```

### Share a collection with its secrets

```bash
apigent collection export payments-api --passphrase "$TEAM_SECRET" --output payments-api.gurl
apigent collection import payments-api.gurl --passphrase "$TEAM_SECRET"
```

## Related commands

- [`apigent sequence`]({{< relref "/cli/sequence" >}}) - Set execution order
- [`apigent init`]({{< relref "/cli/init" >}}) - Project file storage
- [`apigent env`]({{< relref "/cli/env" >}}) - Environment variables
- [`apigent diag`]({{< relref "/cli/diag" >}}) - Inspect a failed request
- [Collections and data-driven runs]({{< relref "/guides/collection-runner" >}}) - Guide
- [Credential storage]({{< relref "/security/credential-storage" >}}) - How collection secrets are encrypted
