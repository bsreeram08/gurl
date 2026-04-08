---
title: "Assertions"
description: "Validate API responses with declarative assertions"
weight: 3
---

# Assertions

Assertions let you validate API responses without writing scripts. Gurl evaluates assertions after receiving a response and before displaying output. Failed assertions cause a non-zero exit code.

## Syntax

Each assertion follows the format `field operator value`:

```bash
gurl run "api" --assert "status=200" --assert "body.path(\"$.name\") contains John"
```

## Supported Operators

| Operator | Description |
|----------|-------------|
| `=` | Equal (exact match) |
| `!=` | Not equal |
| `<` | Less than (numeric comparison) |
| `>` | Greater than (numeric comparison) |
| `<=` | Less than or equal |
| `>=` | Greater than or equal |
| `contains` | Substring match |
| `not_contains` | No substring match |
| `matches` | Regex match |
| `exists` | Field exists (no value needed) |

## Assertable Fields

| Field | Description | Examples |
|-------|-------------|----------|
| `status` | HTTP status code | `status=200`, `status>=200` |
| `header.NAME` | Response header | `header.Content-Type contains json` |
| `body` | Response body as text | `body contains "success"` |
| `body.path(X)` | JSONPath expression | `body.path($.data.id) exists` |

## JSONPath Examples

```javascript
// Assert a nested value equals expected
body.path($.data.users[0].name) = "Alice"

// Assert array length
body.path($.data.users.length()) > 0

// Assert object exists
body.path($.data) exists
```

## CLI Usage

Pass assertions with `--assert`:

```bash
gurl run "get-user" --assert "status=200"
gurl run "get-users" --assert "status=200" --assert "body.path($.users.length()) > 0"
gurl run "search" --assert "body contains results"
gurl run "create" --assert "status=201" --assert "header.Content-Type contains json"
```

Multiple assertions are ANDed together. All must pass for the overall assertion to pass.

## TOML Configuration

Define assertions in request TOML files:

```toml
# requests/user.toml
[request]
method = "GET"
url = "{{BASE_URL}}/users/1"

[assertions]
status = 200
header.Content-Type = "application/json"
body.path($.id) = 1
```

### Inline Assertions

For simple response validation directly in TOML:

```toml
[assertions]
status = 200
"body contains \"email\"" = true
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All assertions passed |
| 1 | One or more assertions failed |
| 2 | Assertion error (invalid syntax, missing field, etc.) |

## Examples

### Validate User Exists

```bash
gurl run "get-user" --assert "status=200" --assert "body.path($.email) contains @"
```

### Validate Error Response

```bash
gurl run "invalid" --assert "status=400" --assert "body contains \"validation error\""
```

### Validate Array Response

```bash
gurl run "list-users" --assert "status=200" --assert "body.path($.data.length()) >= 5"
```

### Validate Header Presence

```bash
gurl run "api" --assert "header.X-Request-Id exists"
```

> [!WARNING]
> Assertions are evaluated before output formatting. If you use `--format minimal`, you may not see assertion failures in the output. Check the exit code in scripts and CI pipelines.
