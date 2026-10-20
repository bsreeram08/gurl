---
title: "gurl edit"
description: "Edit a saved request"
---


Change fields of a saved request from the command line. Only the fields you pass are changed; gurl prints the list of changes.

## Synopsis

```bash
gurl edit <name> [flags]
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--method` | `-X` | none | Change the HTTP method |
| `--url` | `-u` | none | Change the request URL |
| `--header` | `-H` | none | Add a header `'Key: Value'` (repeatable) |
| `--remove-header` | `--rm-header` | none | Remove a header by key (repeatable) |
| `--body`, `--data` | `-d` | none | Change the request body |
| `--collection` | `-c` | none | Move to a collection |
| `--tag` | `-t` | none | Add a tag (repeatable) |
| `--pre-script` | `--pre` | none | Set the pre-request script |
| `--post-script` | `--post` | none | Set the post-response script |
| `--run-if` | | none | Set a conditional run expression (`VAR == VALUE` or `VAR != VALUE`) |
| `--extract` | | none | Add or replace an extraction rule `VAR=METHOD:EXPR` (repeatable) |
| `--remove-extract` | | none | Remove an extraction rule by variable name (repeatable) |
| `--assert` | `-a` | none | Add an assertion `field=op=value` (repeatable) |
| `--auth` | | none | Auth type: `basic`, `bearer`, `apikey`, `oauth1`, `oauth2`, `awsv4`, `digest`, `ntlm`, `none` |
| `--auth-param` | | none | Auth parameter `KEY=VALUE` (repeatable) |

## Aliases

- `e`

## Examples

### Point a request at a new URL

```bash
gurl edit "list-users" --url '{{BASE_URL}}/v2/users'
```

### Add flow metadata for chaining

```bash
gurl edit "get-profile" \
  --run-if "token != ''" \
  --extract userId=jsonpath:'$.id' \
  --post-script "gurl.setVariable('seenProfile', 'true')"
```

### Replace or clear saved auth

```bash
gurl edit "profile" --auth bearer --auth-param token='{{token}}'
gurl edit "profile" --auth none
```

`--auth none` clears auth and cannot be combined with `--auth-param`.

## Related commands

- [`gurl show`]({{< relref "/cli/show" >}}) - See what is saved
- [`gurl save`]({{< relref "/cli/save" >}}) - Save a new request
- [`gurl extract`]({{< relref "/cli/extract" >}}) - Test extraction rules
- [Chaining and extraction]({{< relref "/guides/chaining" >}}) - Guide
