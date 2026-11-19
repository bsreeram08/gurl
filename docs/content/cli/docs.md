---
title: "apigent docs"
description: "Print a compact reference of every command, flag and concept"
---


Print one compact reference of every command, flag and concept. This is the first command an AI agent should run.

## Synopsis

```bash
apigent docs [command [subcommand]] [flags]
```

`apigent docs` is generated from the binary's own command tree at runtime, so it always matches the installed version. It lists every command and subcommand with its arguments and flags (type, default, aliases), followed by short sections on:

- agent mode
- placeholders (typed and dynamic) and environment aliases
- scope precedence
- extract rules
- `--persist`
- the `run --json` shape
- exit codes
- `apigent exec` and `apigent diag`

`apigent docs <command>` narrows the output to one command (for example `apigent docs run` or `apigent docs env set`).

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
apigent docs
```

### One command

```bash
apigent docs run
```

```text
apigent run  — Execute a saved request (aliases: r, execute)
  --env, -e <string>  Environment name to use for variable substitution
  --var, -v <[]string>  Variable substitution (--var key=value)
  ...
  --fail  Exit non-zero when the response status is not 2xx (body is still printed)
  --json, -j  JSON object per execution (NDJSON for --chain/--data), nothing else on stdout
```

### JSON for agents

```bash
apigent docs --json
```

## Related commands

- [Agent mode]({{< relref "/ai/agent-mode" >}}) - How agents should drive apigent
- [CLI Reference]({{< relref "/cli" >}}) - All commands
