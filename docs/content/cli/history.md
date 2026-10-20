---
title: "gurl history"
description: "Show execution history for a request"
---


List recent executions of one saved request, newest first, with the execution id you pass to `gurl diag`.

## Synopsis

```bash
gurl history <name> [flags]
```

Each row shows the short id, status, duration, size and time, followed by the start of the response body. Attempts that never got a response (resolution problems, connection errors, timeouts, script errors) show `ERR` and the error instead. Execution records never contain secrets.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-n`, `-l` | `10` | Number of entries to show (the newest) |
| `--json` | `-j` | `false` | JSON output |

## Aliases

- `hist`
- `h`

## Examples

### Recent runs of a request

```bash
gurl history get-user --limit 5
```

```text
│  #   ID         STATUS   DURATION   SIZE     TIMESTAMP
│  1   e7770aa1   200      0ms        89B      2026-10-20 18:41:41
│       RESP: {"id": "u-123", ...
│  2   3baa2a2c   ERR      0ms        0B       2026-10-20 18:41:41
│       ERROR: variable substitution failed: missing required variables: BASE, API_KEY
```

### JSON for scripts

```bash
gurl history get-user --json
```

```json
[
  {"id": "e7770aa1-94f2-4b4a-9e3e-8605e5c1a0cc", "short_id": "e7770aa1", "request": "get-user",
   "time": "2026-10-20T18:41:41+05:30", "ok": true, "status": 200, "duration_ms": 0, "size_bytes": 89},
  {"id": "3baa2a2c-...", "short_id": "3baa2a2c", "request": "get-user", "time": "...", "ok": false,
   "duration_ms": 0, "size_bytes": 0, "error": "variable substitution failed: ...", "failure_phase": "request_build"}
]
```

### Inspect one entry

```bash
gurl diag 3baa2a2c
```

## Related commands

- [`gurl diag`]({{< relref "/cli/diag" >}}) - Inspect one execution
- [`gurl timeline`]({{< relref "/cli/timeline" >}}) - Executions across all requests
- [`gurl diff`]({{< relref "/cli/diff" >}}) - Compare the latest responses
- [Diagnosing failures]({{< relref "/guides/diagnosing" >}}) - Guide
