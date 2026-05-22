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
| `--run-if` | | none | Conditional expression for running this request |
| `--extract` | | none | Add or replace extraction rule as `VAR_NAME=METHOD:EXPRESSION` |
| `--remove-extract` | | none | Remove extraction rule by variable name |
| `--assert` | `-a` | none | Add an assertion |
| `--auth` | | none | Authentication type: `basic`, `bearer`, `apikey`, `oauth1`, `oauth2`, `awsv4`, `digest`, `ntlm`, or `none` |
| `--auth-param` | | none | Authentication parameter as `key=value` (can be specified multiple times) |

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

### Add flow metadata

```bash
gurl edit "get-profile" \
  --run-if "token != ''" \
  --extract userId=jsonpath:$.id \
  --post-script "gurl.setVariable('seenProfile', 'true')"
```

Adds a run condition, an extraction rule, and a post-response script. `run-if` supports simple `VAR == VALUE` and `VAR != VALUE` checks.

### Update saved auth

```bash
gurl edit "profile" \
  --auth bearer \
  --auth-param token='{{token}}'
```

Replaces the request's saved auth config with bearer auth. Each `--auth-param` must be a `key=value` pair and may be repeated.

### Clear saved auth

```bash
gurl edit "profile" --auth none
```

Clears the request's saved auth config. `--auth none` cannot be combined with `--auth-param`.

## See also

- [`gurl save`](save) - Save a new request
- [`gurl run`](run) - Execute a request
- [`gurl auth`](auth) - Discover auth handlers and parameters
- [`gurl list`](list) - List saved requests
