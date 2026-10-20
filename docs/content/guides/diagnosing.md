---
title: "Diagnosing failures"
description: "Find out why a run failed and what changed since it last worked, with history, diag and diff."
weight: 5
---

Every execution is recorded, so when a request fails you don't need to re-run it with extra flags: look it up by id and compare it with the last time it worked.

## What gets recorded

Every execution of `run` (single requests, `--chain` steps, `--data` rows), `collection run` and shell `send`, including attempts that never got a response:

- resolution problems (missing variables, bad types)
- connection errors and timeouts
- pre-request and post-response script errors
- failed assertions and non-2xx responses

Dry runs are not recorded. Records store the method, URL, headers, body, variables with their origins, response headers, error, failure phase and assertion results, with secrets masked before saving. Response bodies are stored as received.

## 1. Get the execution id

A failed `run` prints it on stderr:

```text
request failed: HTTP 500 (id: 078b9896) — inspect with: gurl diag 078b9896
```

Other ways to find it:

```bash
gurl history "health"                 # newest first, with short ids and errors
gurl history "health" --json          # [{"id","short_id","request","time","ok","status",...}]
gurl timeline --since 1h              # across all requests
gurl run "health" --json | jq -r .short_id
```

## 2. Inspect it

```bash
gurl diag 078b9896        # any unique prefix works; or: gurl diag --id 078b9896
```

`diag` shows:

- **Summary**: result, status, failure phase (`request_build`, `http`, `extraction`, `assertion`, `status`, `pre_request_script`, `post_response_script`, `run_if`, `next_request`), duration, size
- **Request** as sent
- **Variables**, each with its variable origin (`environment`, `collection`, `data`, `cli`, or a runtime value) — secret values show as `*****`
- **Response** headers and body (first 4KB; `--full` for all of it)
- **Assertions**
- **Compared with last success**: what changed since the last successful run of the same request — status, duration, URL, headers and body, variable values and origins, response headers and top-level JSON fields

`--json` prints the record and the comparison as JSON.

## 3. Read the failure phase

| Phase | Usually means | Next step |
|-------|---------------|-----------|
| `request_build` | A placeholder couldn't be filled | Check the problems list; `gurl run <name> --dry-run` |
| `http` | Connection error or timeout | Host, network, `--timeout` |
| `status` | Non-2xx response | Response body in `diag`; compare with last success |
| `assertion` | An `--assert` failed | Assertion messages in `diag` |
| `extraction` | An `--extract` rule found nothing | Test the rule with [`gurl extract`]({{< relref "/cli/extract" >}}) on the stored body |
| `pre_request_script` / `post_response_script` | A script threw | Script error in `diag` |

## Compare the latest responses

```bash
gurl diff "health"                 # last two responses
gurl diff "health" --json          # structured RFC 6902 changes, or a text diff
```

## Secrets in records

Records never contain secrets: secret and credential-looking values are masked before saving, so `diag` compares secret variables by origin only ("came from `environment` both times"), never by value. In [agent mode]({{< relref "/ai/agent-mode" >}}), `diag` and `diff` also mask secrets in response bodies and headers, using the secrets of both compared runs.

## See also

- [`gurl diag`]({{< relref "/cli/diag" >}}), [`gurl history`]({{< relref "/cli/history" >}}), [`gurl timeline`]({{< relref "/cli/timeline" >}}), [`gurl diff`]({{< relref "/cli/diff" >}})
