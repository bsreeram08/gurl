---
title: "valetkey docs"
description: "Print a compact reference of every command, flag and concept"
---


Print one compact reference of every command, flag and concept. This is the first command an AI agent should run.

## Synopsis

```bash
valetkey docs [command [subcommand]] [flags]
```

`valetkey docs` is generated from the binary's own command tree at runtime, so it always matches the installed version. It lists every command and subcommand with its arguments and flags (type, default, aliases), followed by short sections on:

- agent mode
- placeholders (typed and dynamic) and environment aliases
- scope precedence
- extract rules
- `--persist`
- the `run --json` shape
- exit codes
- `valetkey exec` and `valetkey diag`

`valetkey docs <command>` narrows the output to one command (for example `valetkey docs run` or `valetkey docs env set`).

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--json` | `-j` | `false` | JSON output |

The JSON shape is:

```json
{"commands":[{"path","usage","args","flags":[{"name","aliases","type","default","usage"}]}],
 "sections":{...}}
```

## Examples

### Whole reference

```bash
valetkey docs
```

### One command

```bash
valetkey docs run
```

```text
valetkey run  — Execute a saved request (aliases: r, execute)
  --env, -e <string>  Environment name to use for variable substitution
  --var, -v <[]string>  Variable substitution (--var key=value)
  ...
  --fail  Exit non-zero when the response status is not 2xx (body is still printed)
  --json, -j  JSON object per execution (NDJSON for --chain/--data), nothing else on stdout
```

### JSON for agents

```bash
valetkey docs --json
```

## Related commands

- [Agent mode]({{< relref "/ai/agent-mode" >}}) - How agents should drive valetkey
- [CLI Reference]({{< relref "/cli" >}}) - All commands
