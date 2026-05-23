---
title: "gRPC, WebSocket & SSE"
description: "Protocol client libraries available in gurl"
weight: 2
---

# gRPC, WebSocket & SSE

> **Status:** gurl has internal client libraries for gRPC, WebSocket, and SSE, but these are not yet exposed as standalone CLI commands (`gurl grpc`, `gurl ws`, `gurl sse` do not exist). The protocol handlers are used internally by the runner and scripting engine. This page documents the intended CLI interface for a future release.

The protocol implementations live in `internal/protocols/` and include:

- **gRPC** — unary and streaming calls with protobuf reflection
- **WebSocket** — interactive and one-shot modes with reconnection
- **SSE** — server-sent event streaming with filtering

GraphQL is the only non-HTTP protocol currently exposed as a CLI command (`gurl graphql`).
