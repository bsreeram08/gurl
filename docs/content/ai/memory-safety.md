---
title: "Memory safety for AI agents"
description: "Why credentials must stay out of LLM context windows, and the layers gurl uses to keep them out."
weight: 4
---

Everything that enters an LLM context window may be stored, logged, or reused in ways you don't control. If a credential touches a context window, treat it as exposed. gurl's job is to make sure the agent never needs to see one.

## The risks

- **Context exposure.** A key pasted into a prompt, or printed by a command the agent ran, stays in the conversation.
- **History and logs.** Conversations and tool output are often retained.
- **Accidental output.** Models repeat what they've seen, for example in generated code or debugging output.
- **Echoed secrets.** APIs often echo credentials back: in a redirect URL, an error message, a debug field of the response body.

## The layers

**1. Requests are referenced by name.** The agent runs `agentgurl run "get-users" --env prod`. The command contains a request name and an environment name, never a value.

**2. Secrets are encrypted at rest.** Environment and collection secrets are encrypted with AES-256-GCM and decrypted only at send time. See [Credential storage]({{< relref "/security/credential-storage" >}}).

**3. Auth is applied by gurl.** Saved auth configs (`--auth bearer --auth-param token='{{API_KEY}}'`) are filled and applied to the outgoing request inside gurl.

**4. Everything gurl shows or stores is masked.** `show`, `list --json`, `codegen`, `paste`, `env show`, dry runs, `run --json` URLs/headers/errors and execution records replace secret values with `*****`, including URL- and JSON-escaped copies. See [Secret masking]({{< relref "/security/header-redaction" >}}).

**5. Agent mode masks the rest.** With [`--agent`]({{< relref "/ai/agent-mode" >}}), secrets are also masked in response bodies, response headers, errors and `gurl exec` child output; `--reveal`, `paste`, `codegen --clipboard` and `run --output` are refused.

```bash
$ agentgurl env show prod
{"name":"prod","variables":{"API_KEY":"*****","BASE_URL":"https://api.example.com"},"secrets":["API_KEY"], ...}

$ agentgurl run "whoami" --env prod
{"ok":true,"status":200,"body":{"token_echo":"*****"}, ...}
```

**6. Failures are diagnosable without secrets.** Execution records never contain secret values, so `gurl diag <id>` can show the agent what changed since the last success, comparing secret variables by origin only.

## Before and after

```bash
# Before: the key is in the command, the shell history and the agent's context
curl -H "Authorization: Bearer sk-live-abc123def456" https://api.example.com/users
```

```bash
# After: set up once by a human
gurl env create prod --var BASE_URL=https://api.example.com --var API_KEY=- --secret API_KEY
gurl save "get-users" '{{BASE_URL}}/users' --auth bearer --auth-param token='{{API_KEY}}'

# The agent runs
agentgurl run "get-users" --env prod
```

## Limits

- A secret has to be known to gurl to be masked: marked secret, sensitively named, or written into a credential header, sensitive query parameter or auth param. An unrelated value in a response body that happens to be sensitive is not masked.
- Values shorter than 4 characters are not replaced inside free text.
- Outside agent mode, `run` prints response bodies as received. Give agents `agentgurl`, not `gurl`.
