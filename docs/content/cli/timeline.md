---
title: "gurl timeline"
description: "Show global execution timeline"
---

# gurl timeline

Show a global timeline of all request executions across all requests.

## Usage

```bash
gurl timeline [flags]
```

## Description

The `timeline` command displays executions from all requests in chronological order. Use it to get an overview of API activity or investigate issues across multiple endpoints.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--since` | `-s` | none | Show entries from this time ago (e.g., `24h`, `7d`) |
| `--filter` | `-f` | none | Filter by pattern |
| `--limit` | `-n` | `50` | Number of entries to show |
| `--limit` | `-l` | `50` | Number of entries to show |

## Aliases

- `tl`
- `log`

## Examples

### View recent timeline

```bash
gurl timeline
```

Shows the last 50 executions across all requests.

### View last 24 hours

```bash
gurl timeline --since 24h
```

Shows executions from the past 24 hours.

### Filter by pattern

```bash
gurl timeline --filter "users"
```

Shows only executions involving "users" requests.

## See also

- [`gurl history`](history) - View history for a specific request
- [`gurl run`](run) - Execute a request
- [`gurl diff`](diff) - Compare responses
