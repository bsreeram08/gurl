---
title: "gurl sse"
description: "Connect to a Server-Sent Events endpoint"
---


Connect to a Server-Sent Events (SSE) endpoint and print events as they arrive.

## Synopsis

```bash
gurl sse <url> [flags]
```

The URL is required.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--header` | `-H` | none | Custom header (`key:value` format) |
| `--event` | `-e` | none | Only show events of this type |
| `--last-event-id` | `-L` | none | `Last-Event-ID` header value, to resume after a reconnect |
| `--timeout` | `-T` | `30` | Connection timeout in seconds |
| `--json` | `-j` | `false` | Parse and pretty-print JSON data |
| `--color` | `-c` | `false` | Enable syntax highlighting for JSON output |
| `--timestamp` | `-t` | `false` | Show a timestamp for each event |
| `--raw` | `-r` | `false` | Output raw data without formatting |

## Aliases

- `events`
- `sse`

## Examples

### Follow a stream

```bash
gurl sse https://api.example.com/events --json --timestamp
```

### Only one event type

```bash
gurl sse https://api.example.com/events --event order.created
```

### Resume from the last event you saw

```bash
gurl sse https://api.example.com/events -H 'Authorization:Bearer dev-token' --last-event-id 1042
```

## Related commands

- [`gurl ws`]({{< relref "/cli/ws" >}}) - WebSocket connections
- [`gurl grpc`]({{< relref "/cli/grpc" >}}) - gRPC requests
- [`gurl graphql`]({{< relref "/cli/graphql" >}}) - GraphQL queries
