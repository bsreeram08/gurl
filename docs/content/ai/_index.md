---
title: "AI Integration"
description: "Why Gurl is the safest API client for AI agents"
weight: 1
---

When an AI agent needs to interact with an API, it faces a fundamental problem: the agent must handle credentials, which means those credentials end up in its context window.

Gurl solves this by acting as a credential firewall between the AI agent and the API.

## The Problem

When you let an AI agent call an API directly, the credential lifecycle becomes opaque:

1. The AI sees your API key every time you send it a prompt
2. The key appears in conversation history
3. History may be stored, logged, or used for training
4. The AI might accidentally output the key in its response

A token that touches an LLM context window is effectively exposed. It does not matter that the model is trustworthy. The attack surface is the context.

```
Traditional AI Workflow:
  AI Agent → [sees API key in prompt] → Makes HTTP request with key → [sees key in response logs]
```

The key is visible at every step: in the prompt, in the model reasoning, in the response, and in any logs.

## Gurl's Solution

When you use Gurl, the AI agent runs a single command:

```bash
gurl run "my-api" --env production
```

Gurl handles credential injection internally. The AI never sees the actual token, password, or API key. It only sees the response.

```
Gurl Workflow:
  AI Agent → `gurl run "my-api" --env production` → Gurl injects credentials internally → Returns only the response
  AI Agent never sees the API key. Ever.
```

## Key Benefits

**Memory safety**

Credentials never enter the AI context window. Secrets are decrypted only at request execution time and exist only in Gurl's memory.

**Audit trail**

Every request is logged with redacted credentials. You can replay requests without re-entering credentials.

**Reproducibility**

Saved requests can be re-run without re-entering credentials. The AI does not need to store or remember API keys.

**Environment isolation**

Switch between dev, staging, and production environments without exposing credentials to the AI.

**Token lifecycle**

OAuth2 token refresh happens automatically. The AI does not manage token state or handle refresh logic.

## How It Works

**Environment system with encrypted secrets**

The AI references a named environment, not a raw credential value. When `gurl env create prod --secret "API_KEY=sk-xxx"` runs, the key is encrypted with AES-256-GCM and stored locally. The AI never sees the plaintext value.

**Auth handler abstraction**

Eight auth handlers cover common patterns: Bearer, Basic, Digest, API Key, OAuth1, OAuth2, AWS Sigv4, NTLM. Each handler injects credentials at the HTTP transport layer.

**Scripting engine for complex flows**

Pre-request and post-request scripts handle scenarios like signing requests, extracting tokens from responses, or chaining authentication. Scripts run inside Gurl, not in the AI context.

**Collection runner for test suites**

You can define a collection of requests that share auth and environment. The AI runs the collection without ever handling credentials directly.

## Example

A typical workflow with Gurl:

```bash
# Save a request, attaching auth config but not the raw credential
gurl save "get-users" https://api.example.com/users --auth bearer

# Create a production environment with the secret key
gurl env create prod --secret "API_KEY=sk-live-abc123def456"

# The AI agent only needs this:
gurl run "get-users" --env prod
# Response: 200 OK {"users": [...]}
```

The AI sees only the command and the response. It never sees `sk-live-abc123def456`.
