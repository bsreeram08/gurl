---
title: "valetkey shell"
description: "Launch the typed interactive shell"
---


A line-oriented REPL for browsing, editing and sending requests, without fullscreen keybindings or terminal key conflicts.

## Synopsis

```bash
valetkey shell [--env <name>] [--request <name>] [--format <auto|json|table>]
```

`valetkey shell` is for humans. It refuses to start with `--no-input` or `--agent` (exit code 2): `valetkey shell is interactive and cannot run with --no-input or --agent; use valetkey run <name> instead`.

Every `send` is recorded like a `run`, so failures can be inspected with [`valetkey diag`]({{< relref "/cli/diag" >}}).

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
valetkey shell --env staging --request "create-order"
```

### Send with one-off values

```text
valetkey> open create-order
valetkey[create-order]> send amount=10 currency=USD
```

### Build and save a new request

```text
valetkey> new health check
valetkey[health check]> set url https://api.example.com/health
valetkey[health check*]> send
valetkey[health check*]> save
```

## Related commands

- [`valetkey run`]({{< relref "/cli/run" >}}) - Non-interactive execution
- [`valetkey list`]({{< relref "/cli/list" >}}) - List saved requests
- [`valetkey env`]({{< relref "/cli/env" >}}) - Manage environments
