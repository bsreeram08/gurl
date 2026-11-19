---
title: "valetkey exec"
description: "Run a command with an environment's variables injected"
---


Run another command with an environment's variables in its process environment, without printing them.

## Synopsis

```bash
valetkey exec [--env NAME] [--var KEY=VALUE ...] -- <command> [args...]
```

`valetkey exec` hands an environment to another tool: a test runner, a script, or a plain `curl`. Each variable of the environment (aliases resolved, secrets decrypted) is set in the child process's environment. valetkey itself never prints the values.

Everything after `--` is the command to run. The exit code of `valetkey exec` is the command's exit code.

In [agent mode]({{< relref "/ai/agent-mode" >}}), injected secret values are also masked in the child's output, so a command that echoes `$API_KEY` prints `*****`.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--env` | `-e` | active environment | Environment to load |
| `--var` | `-v` | none | Override a variable as `KEY=VALUE` (repeatable) |

## Examples

### Run a script with an environment

```bash
valetkey exec --env prod -- node e2e.mjs
```

The script reads `process.env.API_KEY`, `process.env.BASE_URL`, and so on.

### Use variables in a shell command

```bash
valetkey exec --env prod -- sh -c 'curl -H "Authorization: Bearer $API_KEY" "$BASE_URL/v1/charges"'
```

Single quotes keep your shell from expanding `$API_KEY` before valetkey injects it.

### Override one variable

```bash
valetkey exec --env staging --var REGION=eu -- ./smoke-test.sh
```

## Related commands

- [`valetkey env`]({{< relref "/cli/env" >}}) - Manage environments
- [Environments and variables]({{< relref "/guides/environments" >}}) - Aliases and scopes
- [Agent mode]({{< relref "/ai/agent-mode" >}}) - Strict masking for AI agents
