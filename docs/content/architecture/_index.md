---
title: "Architecture"
description: "How gurl is built: layers, storage, request lifecycle, plugins."
weight: 6
---

For contributors and plugin authors. You don't need this page to use gurl.

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
│  Middleware │ Output │ Command │ Auth               │
├─────────────────────────────────────────────────────┤
│         TUI (bubbletea) — not functional             │
│  Sidebar │ Request Builder │ Response Viewer       │
└─────────────────────────────────────────────────────┘
```

### Layer Descriptions

**CLI Layer** - Uses urfave/cli for command-line argument parsing. Handles all user-facing commands like `gurl save`, `gurl run`, and `gurl edit`.

**Core Services Layer** - Contains the business logic. Request resolution (`internal/resolve`) turns a saved request plus its scopes into the request to send, or into the list of resolution problems; it is shared by `run`, `collection run`, the shell and the TUI. Masking rules (`internal/redact`) are shared by every command that shows or stores a request, and by execution records. Auth registry manages API keys, Bearer tokens, and basic auth. Formatter provides response rendering in multiple formats.

**Storage Layer** - goleveldb-backed persistence for requests, collections, environments, and execution history. Provides ACID-like transactions for data integrity.

**HTTP Client Layer** - Wraps Go's net/http with TLS configuration, timeout handling, proxy support, and cookie jar management.

**Protocol Handlers Layer** - Extends beyond HTTP to support GraphQL, gRPC, WebSocket, and Server-Sent Events. Each handler translates between protocol-specific formats and Gurl's internal request/response model.

**Plugin System Layer** - Four plugin types: MiddlewarePlugin runs before/after requests, OutputPlugin formats responses, CommandPlugin adds new CLI commands, and AuthPlugin provides custom authentication handlers (added in v0.3.0). All plugins are discovered from `~/.config/gurl/plugins/`.

**TUI Layer** - Built with bubbletea. The code exists but is not functional in the current release. The intended design is a full-screen three-pane workspace for browsing requests, editing them, and inspecting responses.

## Request Lifecycle

1. CLI parses command and loads request from storage
2. Pre-request scripting executes (JavaScript/goja)
3. Request resolution fills placeholders from the scopes (environment < collection < data row < CLI < runtime) and applies auth; any problem stops the request here
4. Middleware plugins run their `ApplyBeforeRequest` hooks
5. Protocol handler sends the request
6. Middleware plugins run their `ApplyAfterResponse` hooks
7. Post-response scripting executes
8. Assertions are evaluated
9. Output plugin formats and displays the response
10. The execution is recorded, with secrets masked, whether it succeeded or failed at any earlier step (inspect with `gurl diag`)
