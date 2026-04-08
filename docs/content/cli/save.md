---
title: "gurl save"
description: "Save a curl request with a name"
---

# gurl save

Save a curl request with a memorable name for later use.

## Usage

```bash
gurl [command] [arguments] [flags]
```

```bash
gurl save [name] [url] [flags]
```

## Description

The `save` command stores a curl request under a given name. You can specify the URL directly, pass a full curl command with `--curl`, or pipe curl output from another program.

After saving, use `gurl run [name]` to execute the request.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--collection` | `-c` | none | Assign request to a collection |
| `--folder` | `-F` | none | Assign request to a folder |
| `--tag` | `-t` | none | Add a tag (can be specified multiple times) |
| `--format` | `-f` | `auto` | Output format: `auto`, `json`, or `table` |
| `--description` | | none | Human-readable description |
| `--curl` | | none | Full curl command to parse |
| `--request` | `-X` | `GET` | HTTP method |
| `--header` | `-H` | none | HTTP header (can be specified multiple times) |
| `--data` | `-d` | none | Request body |
| `--body` | | none | Request body (alias for `--data`) |

## Aliases

- `s`

## Examples

### Basic save

```bash
gurl save "users" https://api.example.com/users
```

Saves a GET request to fetch users.

### Save with method and body

```bash
gurl save "create-user" -X POST -H "Content-Type: application/json" -d '{"name":"John"}' https://api.example.com/users
```

Saves a POST request that creates a user with JSON data.

### Save from curl command

```bash
gurl save "auth-test" --curl "curl -H 'Authorization: Bearer token' https://api.example.com/me"
```

Parses the curl command and saves it as a named request.

### Pipe curl output

```bash
curl -s https://api.example.com/health | gurl save "health-check"
```

Pipes the curl output and saves it as a health check request.

### Save with collection and tags

```bash
gurl save "api-users" https://api.example.com/users -c "api" -t "production" -t "v2"
```

Saves the request to the "api" collection with "production" and "v2" tags.

## See also

- [`gurl run`](run) - Execute a saved request
- [`gurl list`](list) - List saved requests
- [`gurl edit`](edit) - Edit a saved request
