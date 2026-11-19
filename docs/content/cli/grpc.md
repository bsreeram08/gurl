---
title: "valetkey grpc"
description: "Execute a gRPC request"
---


Call a gRPC method, or list a server's services through reflection.

## Synopsis

```bash
valetkey grpc <host:port> --service <name> --method <name> [--data <json> | --data-file <file>] [flags]
valetkey grpc <host:port> --list
```

The target (`host:port`) is required. `--service` and `--method` are required for calls.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--service` | `-s` | none | gRPC service name (e.g., `UserService`) |
| `--method` | `-m` | none | gRPC method name (e.g., `GetUser`) |
| `--data` | `-d` | none | JSON request data |
| `--data-file` | `-f` | none | Path to a file containing JSON request data |
| `--call-type` | `--ct` | `unary` | Call type: `unary`, `server-streaming`, `client-streaming`, `bidirectional` |
| `--list` | `-l` | `false` | List available services via reflection |
| `--metadata` | `--mdata` | none | Comma-separated `key:value` pairs for gRPC metadata |
| `--format` | `--fmt` | `auto` | Output format: `auto`, `json`, `table` |
| `--color` | `-c` | `false` | Enable syntax highlighting |
| `--insecure` | `-k` | `false` | Skip TLS verification (testing only) |
| `--cacert` | | none | CA certificate file |
| `--cert` | | none | Client certificate file |
| `--key` | | none | Client key file |
| `--server-name` | | none | Server name for SNI |

## Aliases

- `grpc`

## Examples

### Discover services

```bash
valetkey grpc localhost:50051 --list
```

### Unary call with metadata

```bash
valetkey grpc localhost:50051 -s UserService -m GetUser \
  -d '{"id":"u-123"}' --metadata 'authorization:Bearer dev-token'
```

### Server-streaming call over mutual TLS

```bash
valetkey grpc api.example.com:443 -s Events -m Subscribe --call-type server-streaming \
  --cacert ca.pem --cert client.pem --key client-key.pem
```

## Related commands

- [`valetkey graphql`]({{< relref "/cli/graphql" >}}) - GraphQL queries
- [`valetkey ws`]({{< relref "/cli/ws" >}}) - WebSocket connections
- [`valetkey sse`]({{< relref "/cli/sse" >}}) - Server-Sent Events
