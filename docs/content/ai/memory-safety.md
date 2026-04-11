---
title: "Memory Safety for AI Agents"
description: "How Gurl prevents credential leakage in AI context windows"
weight: 2
---

LLM context windows are not ephemeral. Everything that enters them may be stored, logged, or processed in ways you do not control. Understanding this is the foundation of safe AI API access.

## The Problem with AI + APIs

**Context window exposure**

When you paste an API key into a prompt, it is in the context permanently for that conversation. If the conversation is stored for history, logging, or training, the key persists.

**Conversation history risks**

Many AI platforms retain conversation data. A key that appears in one request may be present in every subsequent request in that session.

**Accidental credential output**

AI models can accidentally include credentials in their responses, especially when generating code examples or debugging output.

**Token caching opacity**

When an AI manages OAuth tokens directly, it may cache them in ways that are invisible to you. A cached token in an AI context is indistinguishable from a leaked token.

> [!IMPORTANT]
> If a credential touches an LLM context window, assume it is exposed. No amount of trust in the model changes this.

## Gurl's Solution

Gurl provides five layers of protection:

**1. Encrypted Environment Secrets**

Secrets stored with AES-256-GCM are decrypted only at request execution time. The decrypted value exists only in memory and is overwritten immediately after use.

```
gurl env create prod --secret "API_KEY=sk-live-abc123def456"
# Stored encrypted. AI sees only the environment name "prod".
```

**2. Auth Handler Abstraction**

Eight auth handlers inject credentials at the HTTP transport layer. The calling process never builds a credential string that could appear in output.

Handlers: Bearer, Basic, Digest, API Key, OAuth1, OAuth2, AWS Sigv4, NTLM.

**3. Header Redaction**

Even in logs, credentials show as `[REDACTED]`. This means audit logs do not become a credential leak vector.

**4. Variable Masking**

When displaying environment variables, secrets show as `*****`. The AI can reference a secret by name without seeing its value.

```
$ gurl env list prod
prod:
  API_KEY=*****
  BASE_URL=https://api.example.com
```

**5. No Prompt Leakage**

`gurl run "request-name"` returns only the HTTP response. Headers with auth are not included in command output unless explicitly requested.

## Practical Example

Before and after comparison:

```bash
# BEFORE: AI sees your API key
curl -H "Authorization: Bearer sk-live-abc123def456" https://api.example.com/users

# AI context window now contains: "sk-live-abc123def456"
# This key is in the conversation history indefinitely.
```

```bash
# AFTER: AI never sees your API key
gurl save "get-users" https://api.example.com/users \
  --auth bearer \
  --env prod

gurl env create prod --secret "API_KEY=sk-live-abc123def456"

# Now the AI only needs:
gurl run "get-users" --env prod

# AI context window contains: "200 OK {users: [...]}"
# The API key never entered the context.
```

## Comparison

| What the AI sees | Direct curl | gurl |
|-----------------|-------------|------|
| API key | `sk-live-abc123def456` | Nothing |
| Auth header | Full `Authorization: Bearer ...` | Nothing |
| Response | Full response with any auth info | Clean response |
| Log files | Full headers | Redacted headers |

## Security Model Summary

Gurl's approach is simple: credentials live in Gurl's memory, not in the AI's memory. The AI executes commands, Gurl handles secrets.

This is not about trusting the AI less. It is about making the credential lifecycle explicable and auditable. When a request fails, you can inspect the logs. When you want to revoke a key, you do it in one place. The AI does not hold state.

The result is an AI agent that can call APIs safely, without you having to trust that every context window it has ever touched is secure.
