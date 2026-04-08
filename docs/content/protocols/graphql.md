---
title: "GraphQL"
description: "Execute GraphQL queries from the terminal"
weight: 1
---

# GraphQL

Gurl provides first-class support for GraphQL APIs with a dedicated `gurl graphql` command.

## Basic Usage

Execute a GraphQL query against an endpoint:

```bash
gurl graphql "https://api.example.com/graphql" --query 'query { users { name email } }'
```

## With Variables

Pass variables using `--vars`:

```bash
gurl graphql "https://api.example.com/graphql" \
  --query 'query($limit: Int!) { users(limit: $limit) { name email } }' \
  --vars '{"limit": 10}'
```

## From a File

For complex queries, store them in `.graphql` files:

```bash
gurl graphql "https://api.example.com/graphql" --query-file queries/get-users.graphql
```

Example `queries/get-users.graphql`:

```graphql
query GetUsers($limit: Int!, $offset: Int) {
  users(limit: $limit, offset: $offset) {
    id
    name
    email
    createdAt
  }
}
```

## Introspection

Query the schema directly:

```bash
gurl graphql "https://api.example.com/graphql" --introspect
```

This fetches the full schema and outputs it as JSON.

Query a specific type:

```bash
gurl graphql "https://api.example.com/graphql" --query '{ __type(name: "User") { name fields { name type { name kind } } } }'
```

## Mutations

Use mutations to create, update, or delete data:

```bash
gurl graphql "https://api.example.com/graphql" \
  --query 'mutation CreateUser($input: CreateUserInput!) { createUser(input: $input) { id email } }' \
  --vars '{"input": {"email": "alice@example.com", "name": "Alice"}}'
```

## Response Handling

By default, responses are formatted with syntax highlighting:

```bash
gurl graphql "https://api.example.com/graphql" --query '{ users { name } }'
# Output:
# {
#   "data": {
#     "users": [
#       { "name": "Alice" },
#       { "name": "Bob" }
#     ]
#   }
# }
```

### Error Display

GraphQL errors are displayed in a structured format, not buried in JSON:

```
ERROR: Validation failed
  - Field "email" of required type "String!" was not provided
  - Argument "limit" expected type "Int!" but got "String"
```

> [!TIP]
> Use `--format minimal` for quieter output in scripts: `gurl graphql "..." --query "..." --format minimal`

## GraphQL with Headers

Pass custom headers:

```bash
gurl graphql "https://api.example.com/graphql" \
  --header "Authorization: Bearer $TOKEN" \
  --header "X-Client-Version: 2.0" \
  --query '{ users { name } }'
```

## Environment Variables

Reference environment variables in queries:

```bash
gurl env switch staging
gurl graphql "{{GRAPHQL_ENDPOINT}}" --query 'query { users { name } }'
```
