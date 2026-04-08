---
title: "gurl history"
description: "Show execution history for a request"
---

# gurl history

Show the execution history for a specific saved request.

## Usage

```bash
gurl history [name] [flags]
```

## Description

The `history` command displays past executions of a request, including timestamps, status codes, response times, and response sizes. This helps you track how an API behaves over time.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-n` | `10` | Number of history entries to show |
| `--limit` | `-l` | `10` | Number of history entries to show |

## Aliases

- `hist`
- `h`

## Examples

### View history for a request

```bash
gurl history "users"
```

Shows the last 10 executions of the "users" request.

### Limit history entries

```bash
gurl history "users" --limit 5
```

Shows only the last 5 executions.

## See also

- [`gurl run`](run) - Execute a request
- [`gurl timeline`](timeline) - Show global timeline
- [`gurl diff`](diff) - Compare responses
