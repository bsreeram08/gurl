---
title: "gurl diff"
description: "Compare responses for a request"
---

# gurl diff

Compare responses from different executions of the same request.

## Usage

```bash
gurl diff [name] [flags]
```

## Description

The `diff` command shows the differences between multiple executions of a request. This helps identify changes in API responses over time or across different environments.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-n` | `2` | Number of executions to compare |
| `--limit` | `-l` | `2` | Number of executions to compare |

## Aliases

- `d`

## Examples

### Compare last two executions

```bash
gurl diff "users"
```

Compares the two most recent executions of the "users" request.

### Compare more executions

```bash
gurl diff "api-users" --limit 3
```

Compares the three most recent executions.

## See also

- [`gurl history`](history) - View execution history
- [`gurl run`](run) - Execute a request
- [`gurl timeline`](timeline) - Show global timeline
