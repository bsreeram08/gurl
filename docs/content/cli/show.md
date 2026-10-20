---
title: "gurl show"
description: "Show details of a saved request"
---


Print everything stored on a saved request: method, URL, headers, body, auth, extraction rules, scripts, collection and tags. Literal secrets are masked by default.

## Synopsis

```bash
gurl show <name> [flags]
```

Values of credential headers, sensitive query parameters (for example `?token=...`) and secret auth params print as `*****`. Values that are placeholders such as `Bearer {{API_KEY}}` stay visible, because they only name a variable. Pass `--reveal` to print the literals; `--reveal` is refused in agent mode.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | `pretty` | Output format: `pretty`, `json`, or `curl` |
| `--json` | | `false` | Output as JSON (same as `--format json`) |
| `--reveal` | | `false` | Show secret header and auth values instead of masking them |

## Aliases

- `inspect`
- `view`
- `info`

## Examples

### Inspect a request before running it

```bash
gurl show literal
```

```text
Name:       literal
Method:     GET
URL:        http://127.0.0.1:18731/x?token=*****

Headers:
  Authorization: *****
```

### JSON for scripts

```bash
gurl show "create-order" --json | jq .headers
```

### Reconstruct the curl command

```bash
gurl show "create-order" --format curl
```

Prints an equivalent `curl` command, masked the same way. For a copy-paste-ready command use [`gurl codegen -l curl`]({{< relref "/cli/codegen" >}}).

## Related commands

- [`gurl run`]({{< relref "/cli/run" >}}) - Execute a saved request (`--dry-run` previews the resolved request)
- [`gurl edit`]({{< relref "/cli/edit" >}}) - Edit a saved request
- [`gurl list`]({{< relref "/cli/list" >}}) - List saved requests
- [Header redaction]({{< relref "/security/header-redaction" >}}) - What gets masked
