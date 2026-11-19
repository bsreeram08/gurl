---
title: "apigent timeline"
description: "Show the global execution timeline"
---


Show recent executions across all requests, newest first.

## Synopsis

```bash
apigent timeline [flags]
```

Each entry carries the execution id, so any row can be opened with `apigent diag <id>`. Failed attempts without a response are included with their error.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--since` | `-s` | none | Show entries since a duration ago (e.g., `24h`, `7d`) |
| `--filter` | `-f` | none | Filter by request name pattern |
| `--limit` | `-n`, `-l` | `50` | Number of entries to show (the newest) |
| `--json` | `-j` | `false` | JSON output |

## Aliases

- `tl`
- `log`

## Examples

### Last 24 hours

```bash
apigent timeline --since 24h
```

### Only requests matching a name

```bash
apigent timeline --filter users
```

### JSON for scripts

```bash
apigent timeline --json --limit 2
```

The JSON is an array of the same objects as [`apigent history --json`]({{< relref "/cli/history" >}}): `id`, `short_id`, `request`, `time`, `ok`, `status`, `duration_ms`, `size_bytes`, and `error` / `failure_phase` for failures.

## Related commands

- [`apigent history`]({{< relref "/cli/history" >}}) - History for one request
- [`apigent diag`]({{< relref "/cli/diag" >}}) - Inspect one execution
- [`apigent diff`]({{< relref "/cli/diff" >}}) - Compare responses
