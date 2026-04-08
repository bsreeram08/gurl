---
title: "gurl edit"
description: "Edit a saved request"
---

# gurl edit

Edit a saved request using your default editor or command-line flags.

## Usage

```bash
gurl edit [name] [flags]
```

## Description

The `edit` command modifies a saved request. Without flags, it opens the request in your default editor. With flags, it updates specific fields directly.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--method` | `-X` | none | HTTP method |
| `--url` | `-u` | none | Request URL |
| `--header` | `-H` | none | Add HTTP header |
| `--remove-header` | | none | Remove a header by name |
| `--body` | `-d` | none | Request body |
| `--collection` | `-c` | none | Move to collection |
| `--tag` | `-t` | none | Add a tag |
| `--pre-script` | | none | Script to run before request |
| `--post-script` | | none | Script to run after request |
| `--assert` | `-a` | none | Add an assertion |

## Aliases

- `e`

## Examples

### Edit in default editor

```bash
gurl edit "my-request"
```

Opens "my-request" in your default editor.

### Change URL

```bash
gurl edit "my-request" --url https://api.new-example.com/users
```

Updates the URL for the request.

### Update method and headers

```bash
gurl edit "api-users" -X POST -H "Authorization: Bearer new-token"
```

Changes the method to POST and updates the Authorization header.

### Add pre-script

```bash
gurl edit "auth-request" --pre-script "./auth-script.sh"
```

Adds a pre-request script to the "auth-request".

## See also

- [`gurl save`](save) - Save a new request
- [`gurl run`](run) - Execute a request
- [`gurl list`](list) - List saved requests
