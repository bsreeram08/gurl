---
title: "apigent shell"
description: "Launch the typed interactive shell"
---


A line-oriented REPL for browsing, editing and sending requests, without fullscreen keybindings or terminal key conflicts.

## Synopsis

```bash
apigent shell [--env <name>] [--request <name>] [--format <auto|json|table>]
```

`apigent shell` is for humans. It refuses to start with `--no-input` or `--agent` (exit code 2): `apigent shell is interactive and cannot run with --no-input or --agent; use apigent run <name> instead`.

Every `send` is recorded like a `run`, so failures can be inspected with [`apigent diag`]({{< relref "/cli/diag" >}}).

### Shell commands

| Command | What it does |
|---------|--------------|
| `help [command]` | Show shell help, or detailed help for one command |
| `list [pattern]` | List saved requests, optionally filtered by text |
| `open [name]` | Open a request; without a name, shows a numbered selector |
| `new [name]` | Start a new draft request |
| `show` | Display the current draft, headers, auth and body |
| `set <field> <value>` | Set `method`, `url`, `body`, `collection`, `folder` or `timeout` |
| `header add <key> <value>` / `header remove <key>` / `header clear` | Manage headers |
| `query add <key> <value>` / `query remove <key>` / `query clear` | Manage URL query parameters |
| `auth none` / `auth basic <user> <pass>` / `auth bearer <token>` / `auth apikey [header] <value>` | Configure auth |
| `env list` / `env use <name>` / `env show` | Inspect or switch environments |
| `save [name]` | Save the current draft |
| `send [key=value ...]` | Send the current draft; missing variables are prompted for |
| `close` | Close the current draft |
| `quit` | Exit the shell |

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | `auto` | Response output format: `auto`, `json`, `table` |
| `--env` | `-e` | none | Active environment for this shell session |
| `--request` | `-r` | none | Open a request immediately by name |

## Aliases

- `sh`
- `repl`

## Examples

### Open a request against staging

```bash
apigent shell --env staging --request "create-order"
```

### Send with one-off values

```text
apigent> open create-order
apigent[create-order]> send amount=10 currency=USD
```

### Build and save a new request

```text
apigent> new health check
apigent[health check]> set url https://api.example.com/health
apigent[health check*]> send
apigent[health check*]> save
```

## Related commands

- [`apigent run`]({{< relref "/cli/run" >}}) - Non-interactive execution
- [`apigent list`]({{< relref "/cli/list" >}}) - List saved requests
- [`apigent env`]({{< relref "/cli/env" >}}) - Manage environments
