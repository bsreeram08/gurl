---
title: "gurl graphql"
description: "Execute a GraphQL query"
---

# gurl graphql

Execute a GraphQL query against an endpoint.

## Usage

```bash
gurl graphql [flags]
```

## Description

The `graphql` command sends GraphQL queries to an endpoint. You can provide the query inline, from a file, with variables, and control output formatting.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--query` | `-q` | none | GraphQL query string |
| `--query-file` | `-f` | none | Path to file containing GraphQL query |
| `--vars` | `-v` | none | Variables as JSON |
| `--operation-name` | | none | GraphQL operation name |
| `--format` | | `auto` | Output format: `auto`, `json`, or `table` |
| `--color` | `-c` | `true` | Enable colored output |

## Aliases

- `gql`

## Examples

### Simple query

```bash
gurl graphql --query "{ users { id name } }" https://api.example.com/graphql
```

Sends a simple GraphQL query.

### Query from file

```bash
gurl graphql --query-file ./query.gql https://api.example.com/graphql
```

Reads the query from a file.

### Query with variables

```bash
gurl graphql -q "query($id: ID!) { user(id: $id) { name } }" --vars '{"id":"123"}' https://api.example.com/graphql
```

Sends a query with variables.

### Named operation

```bash
gurl graphql --query-file ./queries.gql --operation-name "GetUser" https://api.example.com/graphql
```

Executes a specific named operation from a file with multiple queries.

## See also

- [`gurl run`](run) - Execute a saved request
- [`gurl save`](save) - Save a new request
- [`gurl import`](import) - Import from OpenAPI
