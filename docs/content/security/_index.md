---
title: "Security"
description: "How Gurl keeps your credentials safe"
weight: 1
---

Gurl is designed around a simple principle: your credentials should never leave your machine.

When you use an API client directly, your secrets pass through memory, logs, and context windows where they can be exposed. Gurl treats credential security as a first-class concern, not an afterthought.

## Core Security Properties

**Credentials never leave your machine**

All secrets are stored locally using machine-specific encryption. When you run a request, Gurl injects credentials at the HTTP layer without ever exposing them to the calling process or its output streams.

**AES-256-GCM encryption for environment secrets**

Environment variables marked as secret are encrypted with AES-256-GCM before storage. The encryption key is generated on your machine and never leaves it.

**Auth handlers inject credentials programmatically**

Rather than building credential strings that appear in logs or command history, Gurl's auth handlers inject values directly into HTTP headers at the transport layer.

**Sensitive header redaction in logs**

When logging HTTP requests and responses, Gurl automatically redacts sensitive headers including `Authorization`, `Cookie`, `Set-Cookie`, and `Proxy-Authorization`.

**Machine-specific encryption key**

A 256-bit encryption key is generated on first launch using `crypto/rand` and stored with owner-only permissions. Encrypted values are useless on any other machine.

## Security Architecture

Gurl's security model separates concerns across three systems:

| Layer | What it handles | Where secrets live |
|-------|------------------|-------------------|
| Environment secrets | Encrypted storage and retrieval | AES-256-GCM at rest |
| Auth handlers | Credential injection at request time | In-memory only |
| Request logging | Redacted output for auditability | Logs with `[REDACTED]` placeholders |

This separation means that even if logs are compromised or an AI agent is inspecting the process, credentials remain protected.

## Code Generation Safety

All code generators (curl, Python, JavaScript, Go) properly escape string values before interpolation. This prevents shell injection when generating curl commands and code injection in generated scripts. Header values, request bodies, and URLs are sanitized for each target language's escaping rules.

## Path Protection

Export and import commands validate file paths to prevent directory traversal attacks. Paths containing `..` sequences are rejected. Temporary files used during self-update are created with `os.CreateTemp` for unpredictable names.

## Scripting Sandbox

The JavaScript scripting runtime restricts secret variable access by default. Scripts cannot read environment secrets unless explicitly granted access via the `AllowSecretAccess` flag. URL modifications in scripts are validated to only allow `http` and `https` schemes.
