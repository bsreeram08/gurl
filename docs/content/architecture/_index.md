---
title: "Architecture"
description: "How Gurl CLI is built"
weight: 1
---

# Architecture Overview

Gurl CLI is a Go binary designed for single-file distribution. The architecture follows a layered approach where each layer depends only on the layers below it.

## Data Storage

Gurl stores data in two locations:

| Location | Path | Purpose |
|----------|------|---------|
| Database | `~/.local/share/gurl/gurl.db` | Requests, collections, environments, history |
| Config | `~/.config/gurl/config.toml` | User preferences, plugin settings |

The database uses goleveldb (LevelDB port) for high-performance key-value storage. Configuration uses TOML for human-readable settings.

## 9-Layer Architecture

```
┌─────────────────────────────────────────────────────┐
│                    CLI (urfave/cli)                 │
│  save run list edit delete env collection codegen... │
├─────────────────────────────────────────────────────┤
│              Core Services                           │
│  Template Engine │ Auth Registry │ Formatter        │
│  Curl Parser     │ Scripting     │ Assertions       │
├─────────────────────────────────────────────────────┤
│              Storage (goleveldb)                    │
│  Requests │ Collections │ Environments │ History     │
├─────────────────────────────────────────────────────┤
│              HTTP Client (net/http)                 │
│  TLS │ Timeouts │ Proxies │ Cookie Jar              │
├─────────────────────────────────────────────────────┤
│              Protocol Handlers                       │
│  HTTP │ GraphQL │ gRPC │ WebSocket │ SSE             │
├─────────────────────────────────────────────────────┤
│              Plugin System                           │
│  Middleware │ Output │ Command                      │
├─────────────────────────────────────────────────────┤
│              TUI (bubbletea)                        │
│  Sidebar │ Request Builder │ Response Viewer       │
└─────────────────────────────────────────────────────┘
```

### Layer Descriptions

**CLI Layer** - Uses urfave/cli for command-line argument parsing. Handles all user-facing commands like `gurl save`, `gurl run`, and `gurl edit`.

**Core Services Layer** - Contains the business logic. Template engine handles variable substitution. Auth registry manages API keys, Bearer tokens, and basic auth. Formatter provides response rendering in multiple formats.

**Storage Layer** - goleveldb-backed persistence for requests, collections, environments, and execution history. Provides ACID-like transactions for data integrity.

**HTTP Client Layer** - Wraps Go's net/http with TLS configuration, timeout handling, proxy support, and cookie jar management.

**Protocol Handlers Layer** - Extends beyond HTTP to support GraphQL, gRPC, WebSocket, and Server-Sent Events. Each handler translates between protocol-specific formats and Gurl's internal request/response model.

**Plugin System Layer** - Three plugin types: MiddlewarePlugin runs before/after requests, OutputPlugin formats responses, CommandPlugin adds new CLI commands. All plugins are discovered from `~/.config/gurl/plugins/`.

**TUI Layer** - Built with bubbletea for interactive terminal UI. Provides a full-screen three-pane workspace for browsing requests, editing them, and inspecting responses.

## Request Lifecycle

1. CLI parses command and loads request from storage
2. Environment variables are substituted
3. Pre-request scripting executes (JavaScript/goja)
4. Middleware plugins run their `ApplyBeforeRequest` hooks
5. Protocol handler sends the request
6. Middleware plugins run their `ApplyAfterResponse` hooks
7. Post-response scripting executes
8. Assertions are evaluated
9. Output plugin formats and displays the response
10. Results are written to history
