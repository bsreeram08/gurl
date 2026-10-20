---
title: "gurl diff"
description: "Compare responses for a request"
---


Compare the response bodies of the most recent executions of one request.

## Synopsis

```bash
gurl diff <name> [flags]
```

JSON bodies are compared structurally (RFC 6902 changes); other bodies get a text diff. `diff` always compares the newest entries. In [agent mode]({{< relref "/ai/agent-mode" >}}) it masks secrets using the secrets of both compared runs.

To see why one run failed, including variable and header changes since the last success, use [`gurl diag`]({{< relref "/cli/diag" >}}).

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-n`, `-l` | `2` | Number of responses to compare |
| `--json` | `-j` | `false` | JSON output |

## Examples

### Compare the last two runs

```bash
gurl diff get-user
```

### Compare the last three runs

```bash
gurl diff get-user --limit 3
```

### JSON for scripts

```bash
gurl diff get-user --json
```

```json
{
  "request": "get-user",
  "before": {"id": "699120a5-...", "short_id": "699120a5", "time": "2026-10-20T18:41:41+05:30", "status": 200},
  "after":  {"id": "e7770aa1-...", "short_id": "e7770aa1", "time": "2026-10-20T18:41:41+05:30", "status": 200},
  "format": "json",
  "identical": true
}
```

When the bodies differ, the object also carries `changes` (RFC 6902 operations, when `format` is `json`) or `text` (a character diff, when `format` is `text`). `diff --json` compares the newest execution (`after`) with the one before it (`before`).

## Related commands

- [`gurl diag`]({{< relref "/cli/diag" >}}) - Inspect one execution against the last success
- [`gurl history`]({{< relref "/cli/history" >}}) - List executions and ids
- [`gurl timeline`]({{< relref "/cli/timeline" >}}) - Executions across all requests
