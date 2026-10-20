---
title: "gurl timeline"
description: "Show the global execution timeline"
---


Show recent executions across all requests, newest first.

## Synopsis

```bash
gurl timeline [flags]
```

Each entry carries the execution id, so any row can be opened with `gurl diag <id>`. Failed attempts without a response are included with their error.

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
gurl timeline --since 24h
```

### Only requests matching a name

```bash
gurl timeline --filter users
```

### JSON for scripts

```bash
gurl timeline --json --limit 2
```

The JSON is an array of the same objects as [`gurl history --json`]({{< relref "/cli/history" >}}): `id`, `short_id`, `request`, `time`, `ok`, `status`, `duration_ms`, `size_bytes`, and `error` / `failure_phase` for failures.

## Related commands

- [`gurl history`]({{< relref "/cli/history" >}}) - History for one request
- [`gurl diag`]({{< relref "/cli/diag" >}}) - Inspect one execution
- [`gurl diff`]({{< relref "/cli/diff" >}}) - Compare responses
