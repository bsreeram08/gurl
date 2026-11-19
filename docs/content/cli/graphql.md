---
title: "apigent graphql"
description: "Execute a GraphQL query"
---


Send a GraphQL query or mutation to an endpoint, inline or from a file.

## Synopsis

```bash
apigent graphql <endpoint> (--query <query> | --query-file <file>) [flags]
```

The endpoint argument is required, and so is one of `--query` or `--query-file`.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--query` | `-q` | none | GraphQL query string |
| `--query-file` | `-f` | none | Path to a `.graphql` file containing the query |
| `--vars` | `-v` | none | JSON object with query variables (e.g., `'{"limit": 10}'`) |
| `--operation-name` | `--op` | none | GraphQL operation name |
| `--format` | `--fmt` | `auto` | Output format: `auto`, `json`, `table` |
| `--color` | `-c` | `false` | Enable syntax highlighting |

## Aliases

- `gql`

## Examples

### Inline query

```bash
apigent graphql https://api.example.com/graphql --query '{ users { id name } }'
```

### Query with variables

```bash
apigent graphql https://api.example.com/graphql \
  -q 'query($id: ID!) { user(id: $id) { name } }' \
  --vars '{"id":"123"}'
```

### Named operation from a file

```bash
apigent graphql https://api.example.com/graphql --query-file ./queries.graphql --op GetUser
```

## Related commands

- [`apigent grpc`]({{< relref "/cli/grpc" >}}) - gRPC requests
- [`apigent ws`]({{< relref "/cli/ws" >}}) - WebSocket connections
- [`apigent sse`]({{< relref "/cli/sse" >}}) - Server-Sent Events
- [`apigent save`]({{< relref "/cli/save" >}}) - Save an HTTP request (GraphQL over POST works as a saved request too)
