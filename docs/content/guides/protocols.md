---
title: "Other protocols"
description: "Call GraphQL, gRPC, WebSocket and SSE endpoints from the terminal."
weight: 7
aliases: ["/protocols/", "/protocols/graphql/", "/protocols/grpc-and-websocket/"]
---

Saved requests are HTTP. For other protocols gurl has one-shot commands that take the endpoint as an argument. They are not saved, recorded, or filled from environments.

## GraphQL

```bash
gurl graphql https://api.example.com/graphql --query '{ users { name email } }'

gurl graphql https://api.example.com/graphql \
  --query 'query($limit: Int!) { users(limit: $limit) { name } }' \
  --vars '{"limit": 10}'

gurl graphql https://api.example.com/graphql --query-file queries/get-users.graphql --op GetUsers
```

To keep a GraphQL call with auth and environments, save it as an ordinary POST request with a JSON body (`{"query": "...", "variables": {...}}`) and run it with `gurl run`. Flags: [`gurl graphql`]({{< relref "/cli/graphql" >}}).

## gRPC

```bash
gurl grpc localhost:50051 --list                                    # services via reflection
gurl grpc localhost:50051 --service UserService --method GetUser --data '{"id": 1}'
```

TLS options: `--cacert`, `--cert`, `--key`, `--server-name`, or `--insecure` for testing. Flags: [`gurl grpc`]({{< relref "/cli/grpc" >}}).

## WebSocket

```bash
gurl ws wss://echo.example.com --send '{"type":"ping"}'     # send one message and exit
gurl ws wss://echo.example.com --pretty --timestamp         # interactive (stdin/stdout)
```

Flags: [`gurl ws`]({{< relref "/cli/ws" >}}).

## Server-Sent Events

```bash
gurl sse https://api.example.com/events --event order.created --json
```

Flags: [`gurl sse`]({{< relref "/cli/sse" >}}).
