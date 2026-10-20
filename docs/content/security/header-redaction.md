---
title: "Secret masking"
description: "What gurl masks, where, and how --reveal and agent mode change it."
weight: 2
---

gurl replaces secret values with `*****` in everything it prints or stores about a request. One set of rules (the `internal/redact` package) is shared by every command, so `show`, `list --json`, `codegen`, `paste`, dry runs, `run --json` and execution records all mask the same way.

## What counts as a secret

A value is masked when any of these is true:

| Rule | Examples |
|------|----------|
| Its variable is marked secret in an environment or collection | `gurl env set prod API_KEY=- --secret API_KEY` |
| Its name looks sensitive | `Authorization`, `Proxy-Authorization`, `Cookie`, `Set-Cookie`, or any name containing `key`, `secret`, `token`, `password`, `signature` |
| It is a literal value under a sensitive header | `-H 'Authorization: Bearer sk-live-…'` saved in the request |
| It is a literal value of a sensitive query parameter | `https://api.example.com/v1?api_key=sk-live-…` |
| It is a literal sensitive auth param | `--auth-param token=…`, the API key auth `value`, `auth_code` |
| It was extracted or set by a script under a sensitive or secret name | `--extract access_token=jsonpath:$.token` |

Every occurrence is replaced, not only the field it came from: if a secret appears in the URL, the body, an error message or an assertion message, it is masked there too, including its URL-escaped (`%2B`, `+`) and JSON-escaped (`\"`, `&`) spellings. A secret overridden by a higher scope but still reachable through an alias is masked as well.

Values that still contain a `{{placeholder}}` only name a variable and stay visible, for example `Authorization: Bearer {{API_KEY}}`. Values shorter than 4 characters are not replaced inside free text, because that would garble unrelated text.

## Where masking applies

| Output | Masked by default | Notes |
|--------|-------------------|-------|
| `gurl show`, `show --json`, `show --format curl` | yes | `--reveal` shows literal values |
| `gurl list --json` | yes | `--reveal` shows literal values |
| `gurl codegen` | yes | `--reveal` includes literals; `Authorization` always becomes `<your-token-here>` |
| `gurl paste` | yes | `--reveal` includes literals |
| `gurl env show`, `env show --json` | yes | `--reveal` shows plaintext |
| `gurl collection show --json` | yes | |
| `gurl run --dry-run` | yes | auth is never applied in a dry run |
| `gurl run --json` | URL, request headers, response headers, errors | body shown as received, except in agent mode |
| `gurl run` (plain) | errors | the body is your API's response, printed as received |
| Execution records (`history`, `diag`, `diff`, `timeline`) | yes, before saving | response bodies stored as received |
| [Agent mode]({{< relref "/ai/agent-mode" >}}) | everything above, plus response bodies, response headers, errors and `gurl exec` child output | `--reveal` refused |

```text
$ gurl show literal
Name:       literal
Method:     GET
URL:        http://127.0.0.1:18731/x?token=*****

Headers:
  Authorization: *****

$ gurl codegen literal -l curl
curl -X 'GET' -H 'Authorization: <your-token-here>' 'http://127.0.0.1:18731/x?token=*****'
```

## Execution records never store secrets

Every run is recorded so you can [diagnose failures]({{< relref "/guides/diagnosing" >}}). Before a record is saved, secret variable values, literal credential headers and sensitive query parameters are masked, every occurrence of those values is scrubbed from the URL, request headers, request body, error and assertion messages, and sensitive response headers are masked. `gurl diag` therefore compares secret variables by origin only. Response bodies are stored as received.

## --reveal

`--reveal` (on `show`, `list`, `codegen`, `paste`, `env show`) prints the literal values for a human at a terminal.

{{< callout type="warning" >}}
Revealed values end up in terminal scrollback, clipboard history (`paste --reveal`) and generated files (`codegen --reveal`). Never use `--reveal` in CI or in a session an AI agent can read.
{{< /callout >}}

It is refused in agent mode with exit code 2:

```json
{"ok":false,"error":"--reveal is not allowed in agent mode"}
```

## Request logging

The built-in logging plugin also replaces `Authorization`, `Cookie`, `Set-Cookie` and `Proxy-Authorization` header values with `[REDACTED]` in log output. The built-in user-agent plugin sets `User-Agent: gurl/<version>` when a request has none.

## See also

- [Credential storage]({{< relref "/security/credential-storage" >}}) — how secrets are encrypted at rest
- [Memory safety for AI agents]({{< relref "/ai/memory-safety" >}})
