---
title: "gRPC & WebSocket"
description: "Work with gRPC and WebSocket APIs"
weight: 2
---

# gRPC & WebSocket

Gurl supports gRPC and WebSocket protocols alongside HTTP for APIs that require these communication patterns.

## gRPC

Gurl provides a dedicated `gurl grpc` command for gRPC services.

### Listing Services

Use server reflection to list available services and methods:

```bash
gurl grpc "localhost:50051" --list
```

Output:

```
Service: user.v1
  GetUser
  ListUsers
  CreateUser
  UpdateUser
  DeleteUser

Service: auth.v1
  Login
  RefreshToken
  Logout
```

### Unary Calls

Call a simple request-response method:

```bash
gurl grpc "localhost:50051" --method "user.v1.GetUser" --data '{"id": 42}'
```

### Streaming Calls

For server-streaming methods, Gurl streams output as messages arrive:

```bash
gurl grpc "localhost:50051" --method "user.v1.WatchUsers" --data '{}'
# Streams events until interrupted
# Message 1: {"user": {"id": 1, "name": "Alice"}}
# Message 2: {"user": {"id": 2, "name": "Bob"}}
```

### Dynamic Messages

Gurl uses protobuf reflection to decode messages at runtime. No `.proto` files required when server reflection is enabled.

If server reflection is unavailable, specify a proto file:

```bash
gurl grpc "localhost:50051" --method "user.v1.GetUser" --data '{"id": 42}' --proto ./user.proto
```

### With TLS

Connect to a gRPC server with TLS:

```bash
gurl grpc "localhost:50051" --method "user.v1.GetUser" --data '{"id": 42}' --tls
```

### With Authentication

Pass metadata (headers) to include auth tokens:

```bash
gurl grpc "localhost:50051" \
  --method "user.v1.GetUser" \
  --data '{"id": 42}' \
  --metadata "authorization:Bearer $TOKEN"
```

## WebSocket

Gurl supports WebSocket connections for real-time communication.

### Interactive Mode

Open an interactive session:

```bash
gurl ws "ws://localhost:8080/socket"
```

In interactive mode, type messages and press Enter to send. Type `exit` to close.

### One-Shot Mode

Send a single message without interactive mode:

```bash
gurl ws "ws://echo.websocket.org" --send '{"action": "ping"}'
```

The response is printed to stdout:

```json
{"action": "pong", "timestamp": 1699578245}
```

### With Headers

Pass custom headers during the WebSocket handshake:

```bash
gurl ws "wss://api.example.com/socket" \
  --header "Authorization: Bearer $TOKEN" \
  --send '{"event": "subscribe", "channel": "updates"}'
```

### Subprotocols

Specify a subprotocol:

```bash
gurl ws "ws://localhost:8080/socket" --subprotocol "graphql-ws" --send '{"type": "connection_init"}'
```

## Server-Sent Events (SSE)

Gurl can consume SSE streams for real-time updates.

### Basic Usage

Connect to an SSE endpoint:

```bash
gurl sse "https://api.example.com/events"
```

Gurl streams events as they arrive:

```
event: update
data: {"id": 1, "status": "pending"}

event: update
data: {"id": 1, "status": "complete"}
```

### Event Filtering

Filter for specific event types:

```bash
gurl sse "https://api.example.com/events" --event "update"
```

Multiple `--event` flags can be specified to accept multiple event types:

```bash
gurl sse "https://api.example.com/events" --event "update" --event "delete"
```

### Timeout

Set a timeout for SSE connections:

```bash
gurl sse "https://api.example.com/events" --timeout 60
```

After 60 seconds, the connection closes and Gurl exits.

## Protocol Selection

Gurl auto-detects the protocol from the URL scheme, but you can override:

| Scheme | Protocol |
|--------|----------|
| `http://`, `https://` | HTTP |
| `grpc://`, `grpcs://` | gRPC |
| `ws://`, `wss://` | WebSocket |
| `sse://` | Server-Sent Events |

> [!WARNING]
> gRPC and WebSocket support require server reflection to be enabled on your gRPC server. Check your server documentation to enable reflection.
