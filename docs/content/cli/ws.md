---
title: "gurl ws"
description: "Open a WebSocket connection"
---


Connect to a WebSocket endpoint, either interactively (stdin to the socket, messages to stdout) or to send one message and exit.

## Synopsis

```bash
gurl ws <url> [flags]
gurl ws <url> --send '<message>'
```

The URL is required. Without `--send`, the connection is interactive.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--header` | `-H` | none | Custom header (`key:value` format) |
| `--send` | `-s` | none | Send a message and exit (one-shot mode) |
| `--interactive` | `-i` | `false` | Interactive mode (stdin/stdout); the default when `--send` is not given |
| `--pretty` | `-p` | `false` | Pretty-print JSON messages |
| `--color` | `-c` | `false` | Enable syntax highlighting |
| `--timestamp` | `-t` | `false` | Show timestamps in output |
| `--timeout` | `-T` | `30` | Connection timeout in seconds |
| `--reconnect` | `-r` | `false` | Reconnect automatically on disconnect |
| `--max-retries` | | `3` | Maximum reconnection attempts |
| `--insecure` | `-k` | `false` | Skip TLS verification (testing only) |
| `--cacert` | | none | CA certificate file |
| `--cert` | | none | Client certificate file |
| `--key` | | none | Client key file |
| `--server-name` | | none | Server name for SNI |

## Aliases

- `websocket`

## Examples

### Interactive session

```bash
gurl ws wss://echo.example.com/socket --pretty --timestamp
```

### Send one message and exit

```bash
gurl ws wss://api.example.com/ws --send '{"type":"ping"}'
```

### Authenticated connection that reconnects

```bash
gurl ws wss://api.example.com/stream -H 'Authorization:Bearer dev-token' --reconnect --max-retries 5
```

## Related commands

- [`gurl sse`]({{< relref "/cli/sse" >}}) - Server-Sent Events
- [`gurl grpc`]({{< relref "/cli/grpc" >}}) - gRPC requests
- [`gurl graphql`]({{< relref "/cli/graphql" >}}) - GraphQL queries
