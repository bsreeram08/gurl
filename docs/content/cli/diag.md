---
title: "gurl diag"
description: "Inspect one recorded execution"
---


Inspect one execution: the request as sent, the variables and where each came from, the response, assertions, and what changed since the last successful run of the same request.

## Synopsis

```bash
gurl diag <id> [flags]
gurl diag --id <id> [flags]
```

Every execution of `run` (single requests, `--chain` steps and `--data` rows), `collection run` and shell `send` is recorded, including failures before a response: resolution problems, connection errors and timeouts, script errors, assertion failures and non-2xx responses. Dry runs are not recorded.

A failed `run` prints the record id on stderr:

```text
request failed: HTTP 500 (id: 7e0d3985) — inspect with: gurl diag 7e0d3985
```

`collection run` prints the id next to each failed request, `gurl history <name>` lists ids, and `run --json` includes `id` and `short_id` in the JSON object. Any unique prefix of an id works.

`diag` prints:

- **Summary**: request, id, time, environment, result, status, failure phase, duration and size
- **Request**: method, URL, headers and body as sent
- **Variables**: each variable with its variable origin (`environment`, `collection`, data row, CLI, runtime) and value; secrets show as `*****`
- **Response**: status, headers and body (the first 4KB unless `--full`)
- **Assertions**: each result
- **Compared with last success**: status, duration, URL, headers and body, variable values and origins, response headers and top-level JSON fields that changed

Records never contain secrets: secret and credential-looking values are masked before saving, so secret variables are compared by origin only. Response bodies are stored as received. In [agent mode]({{< relref "/ai/agent-mode" >}}), `diag` also masks secrets in the response body and headers, using the secrets of both compared runs.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--id` | | none | Execution id or unique prefix (as printed by `run` and `history`) |
| `--full` | | `false` | Print the whole response body instead of the first 4KB |
| `--json` | | `false` | Print the execution record and comparison as JSON |

## Examples

### Inspect a failure

```bash
gurl diag 7e0d3985
```

```text
Summary
  Request:        failing
  ID:             7e0d3985 (7e0d3985-dfb2-4cb5-b886-bfd754e15e24)
  Environment:    dev
  Result:         FAILED: HTTP 500
  Status:         500 Internal Server Error
  Failure phase:  status

Request
  GET http://127.0.0.1:18731/fail

Variables
  API_KEY  environment  *****
  BASE     environment  http://127.0.0.1:18731
  SA_KEY   environment  *****  (secret)

Response
  Status: 500 Internal Server Error
  Content-Type: application/json
  Set-Cookie: *****
  ...

Compared with last success
  No earlier successful execution of this request.
```

### JSON for scripts and agents

```bash
gurl diag 7e0d3985 --json
```

### Full response body

```bash
gurl diag --id 7e0d3985 --full
```

## Related commands

- [`gurl run`]({{< relref "/cli/run" >}}) - Execute a request
- [`gurl history`]({{< relref "/cli/history" >}}) - List execution ids for a request
- [`gurl diff`]({{< relref "/cli/diff" >}}) - Compare the latest responses
