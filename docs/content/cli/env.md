---
title: "gurl env"
description: "Manage environments and their variables"
---


Manage environments: named sets of variables (base URLs, keys, tokens) that fill placeholders at send time. Secret values are encrypted at rest and masked in every output unless you pass `--reveal`.

## Synopsis

```bash
gurl env <subcommand> [flags] [arguments]
```

## Subcommands

### create

Create a new environment.

```bash
gurl env create <name> [--var KEY=VALUE ...] [--secret KEY ...]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--var` | `-v` | none | Variable as `KEY=VALUE` (repeatable). `KEY=-` reads the value from stdin. |
| `--secret` | `-s` | none | Mark a variable as secret by its key name (repeatable) |

Aliases: `new`, `add`

### list

List all environments.

```bash
gurl env list [--json]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--json` | | `false` | Output as JSON: `[{"name","active","variables"}]` |

Aliases: `ls`, `l`

### switch

Make an environment the active one. `run`, `exec` and `collection run` use the active environment when `--env` is not passed.

```bash
gurl env switch <name>
```

None.

Aliases: `use`, `activate`

### delete

Delete an environment.

```bash
gurl env delete <name>
```

None.

Aliases: `rm`, `remove`, `del`

### show

Show an environment's variables. Secret values print as `*****` unless `--reveal` is passed.

```bash
gurl env show <name> [--json] [--reveal]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--json` | | `false` | Output as JSON: `{"name","id","variables":{},"secrets":[]}` |
| `--reveal` | | `false` | Show secret values instead of masking them (refused in agent mode) |

Aliases: `display`, `view`

### set

Set one or more variables in an existing environment.

```bash
gurl env set <env> KEY=VALUE [KEY=VALUE ...]
gurl env set <env> KEY VALUE
gurl env set <env> KEY=-
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--var` | `-v` | none | Variable as `KEY=VALUE` (repeatable); value `-` reads from stdin |
| `--secret` | `-s` | none | Mark a variable as secret by its key name (repeatable) |

`KEY=-` reads the value from stdin; on a terminal it prompts without echo, so the secret never lands in shell history. The form `gurl env set <env> KEY -` is rejected with `use KEY=- to read the value from stdin`, because flags after a lone `-` would be silently dropped. With `--no-input`, `KEY=-` on a terminal fails instead of prompting.

### unset

Remove variables from an environment.

```bash
gurl env unset <env> --var KEY [--var KEY ...]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--var` | `-v` | none | Variable key to remove (repeatable) |

### import

Import variables from a `.env` file.

```bash
gurl env import <env> --file <path>
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | none | Path to the `.env` file |

Aliases: `imp`

## Aliases

None for `env` itself. Subcommand aliases are listed above.

## Examples

### Create an environment with a secret read from stdin

```bash
gurl env create prod --var BASE_URL=https://api.example.com --var API_KEY=- --secret API_KEY
```

gurl prompts for `API_KEY` without echo. In scripts, pipe the value in: `printf %s "$KEY" | gurl env set prod API_KEY=- --secret API_KEY`.

### Alias one key to another

```bash
gurl env set nets-beta 'API_KEY={{SA_KEY}}'
```

An environment value that refers to another key of the same environment is an alias: requests that use `{{API_KEY}}` get the value of `SA_KEY`. Aliases are resolved once, never re-expanded. See [Variables and placeholders]({{< relref "/guides/variables" >}}).

### Inspect an environment safely

```bash
gurl env show dev
```

```text
Environment: dev
Variables:
  API_KEY = {{SA_KEY}}
  BASE = http://127.0.0.1:18731
  SA_KEY = *****
```

```bash
gurl env show dev --json      # secrets masked
gurl env show dev --reveal    # plaintext, for humans only
```

## Related commands

- [`gurl exec`]({{< relref "/cli/exec" >}}) - Run a command with an environment's variables injected
- [`gurl run`]({{< relref "/cli/run" >}}) - Execute a request with an environment
- [`gurl collection`]({{< relref "/cli/collection" >}}) - Collection-scoped variables
- [Environments and secrets]({{< relref "/guides/environments" >}}) - Guide
- [Credential storage]({{< relref "/security/credential-storage" >}}) - How secrets are encrypted
