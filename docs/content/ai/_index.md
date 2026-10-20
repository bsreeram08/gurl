---
title: "For AI agents"
description: "Let an AI agent call your APIs by name, with JSON output and without ever seeing a credential."
weight: 3
---

An agent that calls an API directly has to hold the credential, which puts it in the model's context window, its logs and its history. With gurl, the agent runs a saved request by name; gurl fills in the secrets at send time and masks them in everything it prints.

```bash
agentgurl run "get-user" --env staging
```

```json
{"ok":true,"status":200,"url":"https://api.example.com/users?api_key=*****","body":{"id":"u-123","auth":"Bearer *****"}, ...}
```

## Start here

1. **[Agent mode]({{< relref "/ai/agent-mode" >}})** — turn on JSON output, no prompts and strict masking with `--agent`, `GURL_AGENT=1` or an `agentgurl` symlink. Run `gurl docs` first.
2. **[JSON output]({{< relref "/ai/json-output" >}})** — the `run --json` object, dry runs, errors, history, diff and exit codes.
3. **[Memory safety]({{< relref "/ai/memory-safety" >}})** — why credentials must stay out of context windows, and what gurl does about it.

## Set up once, as a human

The agent should only ever see names. Store secrets yourself, once:

```bash
gurl collection create "payments-api" --var BASE_URL=https://api.example.com --secret API_KEY=sk-live-...
gurl save "list-payments" '{{BASE_URL}}/payments' --collection payments-api \
  --auth bearer --auth-param token='{{API_KEY}}'

# The agent only needs:
agentgurl run "list-payments"
```

For environments, read the secret from stdin so it never lands in shell history: `gurl env set prod API_KEY=- --secret API_KEY`.

## What the agent sees, and what it doesn't

| | Direct curl | gurl in agent mode |
|---|---|---|
| API key in the command | yes | no — only a request name and environment name |
| Credential headers in output | yes | `*****` |
| Secrets echoed back in the response body | yes | `*****` |
| Secrets in error messages | yes | `*****` |
| Record of what happened | no | every run recorded with secrets masked; `gurl diag <id>` |
