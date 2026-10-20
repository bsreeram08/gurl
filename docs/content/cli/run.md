---
title: "gurl run"
description: "Execute a saved request"
---


Execute a previously saved request by name.

## Synopsis

```bash
gurl run <name> [flags]
```

The `run` command executes a request that was previously saved with `gurl save`. You can pick an environment, override variables, set timeouts, validate responses with assertions, iterate over a data file, and follow request chains.

Before anything is sent, gurl performs **request resolution**: every `{{placeholder}}` in the URL, headers, body and auth params is filled from the scopes (environment < collection < data row < CLI `--var` < runtime). If any placeholder cannot be filled, the request is not sent and every **resolution problem** is reported at once, for example `missing required variables: BASE, API_KEY`. See [Environments and variables]({{< relref "/guides/environments" >}}) for the placeholder rules.

If the saved request has auth settings, `gurl run` applies them after the placeholders are filled.

### Output streams

- The response body goes to stdout.
- Status lines, notes (for example `note: uid not saved; add --persist to use them in later runs`), persist summaries, assertion summaries and `Response saved` go to stderr, so `gurl run ... | jq .` sees only the body.
- With `--json`, stdout carries only JSON (see below).

### Execution records and the failure notice

Every execution is recorded, including attempts that never got a response (resolution problems, connection errors, timeouts, script errors), assertion failures and non-2xx responses. Dry runs are not recorded. When a run fails, gurl names the record on stderr:

```text
request failed: HTTP 500 (id: bbd47546) — inspect with: gurl diag bbd47546
```

Pass that id to [`gurl diag`]({{< relref "/cli/diag" >}}). Execution records never contain secrets; see [Header Redaction]({{< relref "/security/header-redaction" >}}).

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--env` | `-e` | none | Environment to use for variable substitution. An unknown environment is an error. |
| `--var` | `-v` | none | Variable as `key=value` (repeatable). Highest scope except runtime values. |
| `--format` | `-f` | `auto` | Output format: `auto`, `json`, or `table` |
| `--output` | `-o` | none | Write the response to a file (`-` for stdout). Refused in agent mode. |
| `--force` | | `false` | Overwrite the output file if it exists |
| `--timeout` | | none | Request timeout (e.g., `5s`, `1m`, `30s`) |
| `--chain` | `--ch` | `false` | Enable request chaining via `setNextRequest` |
| `--persist` | | `false` | Write extracted and script-set variables back to their variable origin |
| `--dry-run` | | `false` | Resolve the request and print it without sending it |
| `--assert` | `-a` | none | Assertion to evaluate (repeatable), e.g. `status=200` |
| `--fail` | | `false` | Exit non-zero when the response status is not 2xx (the body is still printed) |
| `--data` | `-d` | none | CSV or JSON data file for data-driven iteration |
| `--json` | `-j` | `false` | One JSON object per execution (NDJSON for `--chain`/`--data`), nothing else on stdout |

`--persist` and `--dry-run` cannot be used together (exit code 2).

## Aliases

- `r`
- `execute`

## Examples

### Basic run

```bash
gurl run "users"
```

### Run with an environment and variables

```bash
gurl run "users" --env production
gurl run "users" --var "page=2" --var "limit=50"
```

### Run a request with saved auth

```bash
gurl save "profile" https://api.example.com/me \
  --auth bearer \
  --auth-param token='{{token}}'

gurl run "profile" --var token=abc123
```

The saved `bearer` auth handler adds the `Authorization: Bearer abc123` header during execution. You don't need to resupply `--auth` when running the request.

### Fail on non-2xx responses

```bash
gurl run "create-order" --env prod --fail | jq .
```

By default a non-2xx response still exits 0. With `--fail` it exits 1; the body is still printed to stdout.

### Preview without sending (dry run)

```bash
gurl run "get-user" --env dev --dry-run
```

```text
GET http://127.0.0.1:18731/users?api_key=*****
Authorization: *****
X-Merchant: M-4242
extract: uid <- jsonpath:$.id
```

The dry run shows the request after pre-request scripts run, with secrets masked. It never applies auth (applying auth could contact a token endpoint), leaves `{{$uuid}}`-style dynamic functions in place, sends nothing, records nothing, and exits 1 when the request has resolution problems.

### Machine-readable output

```bash
gurl run "get-user" --env dev --json
```

stdout gets one JSON object per execution (one line). Empty fields are omitted:

```json
{"id":"50488e65-9cb3-4cbe-a5b2-769258964b6e","short_id":"50488e65","request":"get-user","environment":"dev",
 "ok":true,"method":"GET","url":"http://127.0.0.1:18731/users?api_key=*****","status":200,"status_text":"OK",
 "duration_ms":4,"size_bytes":89,"response_headers":{"Content-Type":"application/json","Set-Cookie":"*****"},
 "body":{"id":"u-123"},"extracted":{"uid":"u-123"}}
```

(Wrapped here for reading; the real output is one line.)

The full set of fields is:

```text
id short_id request collection environment row ok skipped skip_reason method url
status status_text duration_ms size_bytes response_headers body error failure_phase
problems[{kind,variable,location,message}] assertions[{name,passed,message}]
extracted persisted{environment,variables,collections[{name,variables}]}
```

- `ok` is true when there was no error, no failed assertion and the status was 2xx.
- `body` is the parsed JSON value when the body is valid JSON, otherwise a string.
- A failed attempt has `error` and `failure_phase`: `request_build`, `http`, `extraction`, `assertion`, `status`, `pre_request_script`, `post_response_script`, `run_if` or `next_request`.
- Requests that cannot be resolved list structured `problems`. Problem kinds are `missing`, `leftover_placeholder`, `bad_type`, `unknown_dynamic` and `auth`:

```json
{"ok":false,"failure_phase":"request_build","error":"variable substitution failed: missing required variables: BASE, API_KEY",
 "problems":[{"kind":"missing","variable":"BASE","location":"url","message":"missing required variable \"BASE\""}, ...]}
```

- `--chain` steps and `--data` rows each print one line (NDJSON). With `--persist`, `persisted` is on the last line.
- Masking matches execution records: secret values are masked in the URL, request data, response headers and errors. The body is shown as received, except in [agent mode]({{< relref "/ai/agent-mode" >}}), where secrets are masked in the body too.
- The failure notice is not printed with `--json`; the record id is in the object (`id`, `short_id`).
- With `--json`, errors before any execution (unknown request or environment, conflicting flags) print `{"ok":false,"error":"..."}` on stdout, as well as the message on stderr.

`--dry-run --json` prints the planned request instead:

```json
{"dry_run":true,"request":"get-user","environment":"dev","ok":true,"method":"GET",
 "url":"http://127.0.0.1:18731/users?api_key=*****",
 "headers":[{"key":"Authorization","value":"*****"},{"key":"X-Merchant","value":"M-4242"}],
 "extracts":[{"name":"uid","source":"jsonpath:$.id"}]}
```

### Run with timeout and assertion

```bash
gurl run "create-user" --timeout 10s --assert "status=201"
```

### Run a chained request

```bash
gurl run "login" --chain --env staging
```

Enables request chaining for saved post-response scripts that call `gurl.setNextRequest(...)`. Extracted variables and variables set by scripts become available to later requests in the chain.

### Persist flow variables

```bash
gurl run "login" --chain --persist --env staging
```

Without `--persist`, extracted and script-set variables are dropped at the end of the run and `run` prints a note naming them. With `--persist`, each one is written back to its variable origin: values that came from the environment go to the environment (`--env` or the active one), others to the request's collection, otherwise to the environment. CLI variables and data-row values are not written back unless extraction or a script changes them.

`--persist` refuses to write a value that contains `{{` (for example `refusing to persist "tpl": value contains "{{"`), and returns an error for a locked collection instead of overwriting it.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success. A non-2xx response is still 0 unless `--fail` is passed. |
| 1 | Request or environment not found, resolution problems, transport error, script or extraction error, failed assertion, non-2xx with `--fail`, dry run with problems |
| 2 | Usage: missing request name with `--no-input`, conflicting flags (`--persist --dry-run`), `--reveal` or `--output` in agent mode |

## Related commands

- [`gurl diag`]({{< relref "/cli/diag" >}}) - Inspect a recorded execution
- [`gurl save`]({{< relref "/cli/save" >}}) - Save a new request
- [`gurl extract`]({{< relref "/cli/extract" >}}) - Test extraction rules offline
- [`gurl history`]({{< relref "/cli/history" >}}) - View execution history
- [`gurl env`]({{< relref "/cli/env" >}}) - Manage environments
- [Agent mode]({{< relref "/ai/agent-mode" >}}) - JSON output and strict masking for AI agents
