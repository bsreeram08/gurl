---
title: "Gurl CLI"
description: "curl for the AI era — secure, logged, and agent-friendly"
layout: home
---

Gurl is curl for the AI era. It gives AI agents a secure, observable, and easily configurable tool for making HTTP requests — with built-in credential isolation, request logging, auth handling, and scripting.

## curl for the AI era.

AI agents need to call APIs. Gurl makes that safe and auditable — credentials never touch agent context, every request is logged, and collections are trivially configurable. Works just as well for humans at the terminal.

## Features

### 21 CLI Commands

Every command you need for API workflows: save, run, list, edit, delete, duplicate, export, import, and more.

### 8 Auth Handlers

Programmatic credential injection for every auth scheme:

| Auth Type | Use Case |
|-----------|----------|
| Basic | Username/password |
| Bearer | Token-based |
| Digest | Challenge-response |
| API Key | Header or query |
| OAuth 1.0 | Signature-based |
| OAuth 2.0 | Token flow |
| AWS SigV4 | AWS API Gateway |
| NTLM | Windows integrated |

### 5 Protocol Clients

Support for modern API protocols:

- **HTTP** - Full request/response with multipart support
- **GraphQL** - Query, mutation, subscription
- **gRPC** - Proto-based RPC with reflection
- **WebSocket** - Real-time bidirectional
- **SSE** - Server-sent events

### Scripting Engine

Extend requests with JavaScript using the goja runtime. Modify headers, transform responses, or chain dependent requests with full access to the request and response objects.

### Collection Runner

Run collections with data-driven testing. Feed CSV or JSON test data, assert on response status, body, and headers.

### Interactive TUI

Built with bubbletea, the Gurl interface is a first-class terminal citizen. Navigate requests, inspect responses, and manage environments without losing your terminal workflow.

### Plugin System

Extend Gurl with middleware for request/response transformation and custom output formatters. Plugins hook into the request lifecycle.

### Multi-language Code Generation

Generate idiomatic client code from saved requests:

```bash
gurl codegen "my-request" --lang python
gurl codegen "my-request" --lang javascript
gurl codegen "my-request" --lang go
gurl codegen "my-request" --lang curl
```

## Quick Start

```bash
brew tap bsreeram08/gurl https://github.com/bsreeram08/gurl
brew install gurl
gurl save "my-api" https://api.example.com/users -H "Authorization: Bearer $TOKEN"
gurl run "my-api"
```

Save any request, run it instantly, generate code, and share with your team.

## Built for AI Agents

They never touch your credentials.

Gurl is designed for AI agent workflows where security matters. Credentials never appear in logs, never leak into prompts, and never get exposed to model providers.

**Security model:**

- Environment secrets encrypted at rest with AES-256-GCM
- Auth handlers inject credentials programmatically
- Header redaction in all log output
- No credentials in generated code without explicit flags

See the [AI integration page](/ai/) for how Gurl works with AI agents.

## Why Gurl

- **Local-first** — no account, no cloud sync, no lock-in
- **Git-friendly** — collections are just data, export and version them
- **Scriptable** — JavaScript pre/post-request hooks, chain requests, CI-ready exit codes
- **Open source** — inspect and extend everything
