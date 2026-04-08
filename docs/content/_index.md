---
title: "Gurl CLI"
description: "The terminal-first API client"
layout: home
---

Gurl is the API client that lives in your terminal. Built for developers who prefer the command line, it handles every auth type, every protocol, and every workflow without leaving the terminal.

## The terminal-first API client. Like Postman, for your terminal.

Whether you are debugging an API at 2am or scripting a CI pipeline, Gurl gives you the full power of an API client without leaving your terminal.

## Features

### 17 CLI Commands

Every command you need for API workflows: save, run, list, edit, delete, duplicate, export, import, and more.

### 9 Auth Handlers

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
| Hawk | Message authentication |

### 4 Protocol Clients

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

## What Gurl Replaces

| Paid Tool | Gurl |
|-----------|------|
| Postman | Local-first, no account required |
| Insomnia | Open source, lighter weight |
| Bruno | Git-friendly collection format |
| Yaak | Faster, scriptable |
