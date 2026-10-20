---
title: "gurl list"
description: "List saved requests"
---


List saved requests, optionally filtered by text, collection or tag.

## Synopsis

```bash
gurl list [flags]
```

`--pattern` matches a substring of the request name, URL or any header value, so you can find requests by merchant ID, host or similar. Names are shown in full whenever stdout is not a terminal (or with `--wide`), so piped output is never truncated.

`--json` prints the saved requests with literal secrets masked: values of credential headers (`Authorization`, `Cookie`, names containing `key`, `token`, `secret`, `password`, `signature`), sensitive query parameters such as `?api_key=...`, and secret auth params show as `*****`. Values that are still placeholders (`Bearer {{API_KEY}}`) stay visible. Pass `--reveal` to see the literals.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--pattern` | `-p` | none | Filter by substring of name, URL or header value |
| `--collection` | `-c` | none | Filter by collection |
| `--tag` | `-t` | none | Filter by tag |
| `--json` | `-j` | `false` | JSON output (secrets masked) |
| `--reveal` | | `false` | Show literal secret header values in `--json` output (refused in agent mode) |
| `--wide` | `-w` | `false` | Never truncate names (default when stdout is not a terminal) |
| `--format` | `-f` | `table` | Output format: `table` or `list` |
| `--limit` | `-n` | none | Limit the number of results |
| `--sort` | `-s` | `updated` | Sort by `name`, `updated` or `collection` |

## Aliases

- `ls`
- `l`

## Examples

### Find a request by a header value

```bash
gurl list --pattern M-4242
```

### Requests in one collection, sorted by name

```bash
gurl list --collection payments-api --sort name
```

### Machine-readable list

```bash
gurl list --json | jq -r '.[].name'
```

## Related commands

- [`gurl show`]({{< relref "/cli/show" >}}) - Show one request in full
- [`gurl save`]({{< relref "/cli/save" >}}) - Save a new request
- [`gurl collection`]({{< relref "/cli/collection" >}}) - Manage collections
- [Header redaction]({{< relref "/security/header-redaction" >}}) - What gets masked
