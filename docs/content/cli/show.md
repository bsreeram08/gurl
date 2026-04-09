---
title: "gurl show"
description: "Show details of a saved request"
---

# gurl show

Display the full details of a saved request.

## Usage

```bash
gurl show [name] [flags]
```

## Description

The `show` command prints all stored properties of a named request: URL, method, headers, body, variables, assertions, timeout, collection, folder, and tags. It is useful for inspecting what was saved before running or editing a request.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | `pretty` | Output format: `pretty`, `json`, or `curl` |

## Aliases

- `inspect`
- `view`
- `info`

## Examples

### Pretty-print a request

```bash
gurl show "users"
```

Prints the request in a human-readable format with labeled fields.

### Output as JSON

```bash
gurl show "users" --format json
```

Returns the full request object as indented JSON. Useful for scripting or piping to `jq`.

### Output as curl command

```bash
gurl show "create-user" --format curl
```

Reconstructs and prints the equivalent `curl` command. Useful for sharing or debugging outside of Gurl.

## See also

- [`gurl run`](run) - Execute a saved request
- [`gurl edit`](edit) - Edit a saved request
- [`gurl list`](list) - List all saved requests
