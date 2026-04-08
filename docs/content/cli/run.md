---
title: "gurl run"
description: "Execute a saved request"
---

# gurl run

Execute a previously saved request by name.

## Usage

```bash
gurl run [name] [flags]
```

## Description

The `run` command executes a request that was previously saved with `gurl save`. You can override variables, switch environments, set timeouts, and validate responses with assertions.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--env` | `-e` | none | Environment to use |
| `--var` | `-v` | none | Variable as `key=value` (can be specified multiple times) |
| `--format` | `-f` | `auto` | Output format: `auto`, `json`, or `table` |
| `--cache` | `-c` | `false` | Use cached response |
| `--output` | `-o` | none | Write response to file |
| `--force` | | `false` | Overwrite output file if it exists |
| `--timeout` | | `30s` | Request timeout (e.g., `5s`, `1m`, `30s`) |
| `--chain` | `-ch` | `false` | Enable request chaining |
| `--assert` | `-a` | none | Assertion to validate response |
| `--data` | `-d` | none | Data file for iteration |

## Aliases

- `r`
- `execute`

## Examples

### Basic run

```bash
gurl run "users"
```

Executes the "users" request.

### Run with environment

```bash
gurl run "users" --env production
```

Executes the request using the "production" environment variables.

### Run with variables

```bash
gurl run "users" --var "page=2" --var "limit=50"
```

Overrides the `page` and `limit` variables for this execution.

### Run with JSON output

```bash
gurl run "api-users" --format json
```

Outputs the response in JSON format regardless of auto-detection.

### Run with timeout and assertion

```bash
gurl run "create-user" --timeout 10s --assert "status=201"
```

Sets a 10-second timeout and asserts the response status is 201.

## See also

- [`gurl save`](save) - Save a new request
- [`gurl history`](history) - View execution history
- [`gurl env`](env) - Manage environments
