---
title: "JSON output"
description: "Every --json shape: run, dry runs, errors, history, diff, and listing commands."
weight: 3
---

Every command an agent or script needs has `--json`. In [agent mode]({{< relref "/ai/agent-mode" >}}) it is on automatically. Empty fields are omitted, and `<`, `>` and `&` are not escaped.

## run

`gurl run --json` prints one object per execution on stdout and nothing else: one line for a single run, NDJSON (one line per step or row) for `--chain` and `--data`. With `--persist`, `persisted` is on the last line.

| Field | Meaning |
|-------|---------|
| `id`, `short_id` | Execution record id; pass to `gurl diag` |
| `request`, `collection`, `environment`, `row` | What ran (`row` for `--data` iterations) |
| `ok` | No error, no failed assertion, and a 2xx status |
| `skipped`, `skip_reason` | The request was skipped (e.g. `run-if`) |
| `method`, `url` | The request as sent (secrets masked) |
| `status`, `status_text`, `duration_ms`, `size_bytes` | Response summary |
| `response_headers` | Object of header → value; sensitive headers masked |
| `body` | Parsed JSON when the body is valid JSON, otherwise a string |
| `error`, `failure_phase` | Why the attempt failed |
| `problems` | Resolution problems: `[{"kind","variable","location","message"}]` |
| `assertions` | `[{"name","passed","message"}]` |
| `extracted` | Values captured by `--extract` rules |
| `persisted` | `{"environment","variables","collections":[{"name","variables"}]}` |

Failure phases: `request_build`, `http`, `extraction`, `assertion`, `status`, `pre_request_script`, `post_response_script`, `run_if`, `next_request`.

Problem kinds: `missing`, `leftover_placeholder`, `bad_type`, `unknown_dynamic`, `auth`.

A request that could not be resolved:

```json
{"id":"5a9eefb8-87bc-401d-94b7-b80c9a07cf18","short_id":"5a9eefb8","request":"get-user","ok":false,
 "method":"GET","url":"{{BASE}}/users?api_key={{API_KEY}}","duration_ms":0,"size_bytes":0,
 "error":"variable substitution failed: missing required variables: BASE, API_KEY","failure_phase":"request_build",
 "problems":[{"kind":"missing","variable":"BASE","location":"url","message":"missing required variable \"BASE\""},
             {"kind":"missing","variable":"API_KEY","location":"header Authorization","message":"missing required variable \"API_KEY\""}]}
```

A failed assertion:

```json
{"ok":false,"status":200,"failure_phase":"assertion",
 "assertions":[{"name":"status = 201","passed":false,"message":"FAIL: status = 201 (actual: 200)"}], ...}
```

Masking matches execution records (URL, request headers, errors, response headers). Outside agent mode the body is shown as received; in agent mode secrets are masked in the body too.

## run --dry-run --json

The planned request; nothing is sent or recorded. Exits 1 when `problems` is non-empty.

```json
{"dry_run":true,"request":"get-user","environment":"dev","ok":true,"method":"GET",
 "url":"http://127.0.0.1:18731/users?api_key=*****",
 "headers":[{"key":"Authorization","value":"*****"},{"key":"X-Merchant","value":"M-4242"}],
 "extracts":[{"name":"uid","source":"jsonpath:$.id"}]}
```

## Errors

With `--json` (or in agent mode), an error that is not part of an execution object — unknown request or environment, conflicting flags, a missing argument with `--no-input` — prints on stdout:

```json
{"ok":false,"error":"request not found: nope"}
```

The message also goes to stderr.

## history and timeline

```bash
gurl history "get-user" --json
gurl timeline --since 24h --json
```

An array, newest first:

```json
[{"id":"e7770aa1-…","short_id":"e7770aa1","request":"get-user","time":"2026-10-20T18:41:41+05:30",
  "ok":true,"status":200,"duration_ms":0,"size_bytes":89},
 {"id":"3baa2a2c-…","short_id":"3baa2a2c","request":"get-user","time":"…","ok":false,
  "duration_ms":0,"size_bytes":0,"error":"variable substitution failed: …","failure_phase":"request_build"}]
```

## diff

```bash
gurl diff "get-user" --json
```

```json
{"request":"get-user",
 "before":{"id":"…","short_id":"699120a5","time":"…","status":200},
 "after":{"id":"…","short_id":"e7770aa1","time":"…","status":200},
 "format":"json","identical":true}
```

When the responses differ, JSON bodies are compared as structured RFC 6902 changes; other bodies as a text diff.

## diag

`gurl diag <id> --json` prints `{"record": {...}, "compared_with": ...}`: the stored execution record (method, URL, headers, body, `vars` with `name`/`origin`/`value`, response headers, error, failure phase, assertions) and the comparison with the last success. See [`gurl diag`]({{< relref "/cli/diag" >}}).

## Listing and inspecting

| Command | Shape |
|---------|-------|
| `gurl list --json` | Array of saved requests; literal credential headers and sensitive query params masked |
| `gurl show <name> --json` | One saved request, masked the same way |
| `gurl env list --json` | `[{"name","active","variables"}]` (variables = count) |
| `gurl env show <name> --json` | `{"name","id","variables":{...},"secrets":[...]}`, secrets masked |
| `gurl collection list --json` | `[{"name","requests","variables","updated"}]` (counts) |
| `gurl collection show <name> --json` | `{"name","id","variables":{...},"secrets":[...],"requests":[{"name","method","url"}]}`, secrets masked |
| `gurl docs --json` | `{"commands":[{"path","usage","args","flags":[...]}],"sections":{...}}` |

`--reveal` (where supported) shows the masked values, and is refused in agent mode.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success. A non-2xx response is still 0 unless `--fail` is passed. |
| 1 | Not found, resolution problems, transport error, script or extraction error, failed assertion, non-2xx with `--fail`, dry run with problems |
| 2 | Usage: missing argument with `--no-input`, conflicting flags, refused in agent mode |
