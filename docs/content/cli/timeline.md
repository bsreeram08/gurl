---
title: "valetkey timeline"
description: "Show the global execution timeline"
---


Show recent executions across all requests, newest first.

## Synopsis

```bash
valetkey timeline [flags]
```

Each entry carries the execution id, so any row can be opened with `valetkey diag <id>`. Failed attempts without a response are included with their error.

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
valetkey timeline --since 24h
```

### Only requests matching a name

```bash
valetkey timeline --filter users
```

### JSON for scripts

```bash
valetkey timeline --json --limit 2
```

The JSON is an array of the same objects as [`valetkey history --json`]({{< relref "/cli/history" >}}): `id`, `short_id`, `request`, `time`, `ok`, `status`, `duration_ms`, `size_bytes`, and `error` / `failure_phase` for failures.

## Related commands

- [`valetkey history`]({{< relref "/cli/history" >}}) - History for one request
- [`valetkey diag`]({{< relref "/cli/diag" >}}) - Inspect one execution
- [`valetkey diff`]({{< relref "/cli/diff" >}}) - Compare responses
