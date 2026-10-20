---
title: "gurl save"
description: "Save a request under a name"
---


Store an HTTP request under a name so you can run it later with `gurl run <name>`. URL, headers, body and auth params may contain `{{placeholders}}` that are filled at send time.

## Synopsis

```bash
gurl save <name> <url> [flags]
gurl save <name> --curl "<curl command>"
gurl save --name <name> --curl "<curl command>"
```

To parse a curl command from stdin, use [`gurl detect`]({{< relref "/cli/detect" >}}).

The collection passed with `--collection` must already exist. In an interactive terminal gurl offers to create it; with `--no-input` (or `--json`/`--agent`) it fails and names the command to run: `collection "x" does not exist; create it first with 'gurl collection create x'`.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--collection` | `-c` | none | Assign to a collection |
| `--folder` | `-F` | none | Assign to a folder (e.g., `api/v2/users`) |
| `--tag` | `-t` | none | Add a tag (repeatable) |
| `--format` | `-f` | `auto` | Output format preference: `auto`, `json`, `table` |
| `--description` | | none | Human-readable description |
| `--name` | | none | Name for the request (used with `--curl`) |
| `--curl` | | none | Full curl command as a string |
| `--request` | `-X` | `GET` | HTTP method |
| `--header` | `-H` | none | HTTP header `'Key: Value'` (repeatable) |
| `--data`, `--body` | `-d` | none | Request body |
| `--extract` | | none | Extraction rule `VAR=METHOD:EXPR` (repeatable). METHOD: `jsonpath`, `header`, `regex`, `jq` |
| `--pre-script` | `--pre` | none | Pre-request script |
| `--post-script` | `--post` | none | Post-response script |
| `--auth` | | none | Auth type: `basic`, `bearer`, `apikey`, `oauth1`, `oauth2`, `awsv4`, `digest`, `ntlm`, `none` |
| `--auth-param` | | none | Auth parameter `KEY=VALUE` (repeatable) |

## Aliases

- `s`

## Examples

### Save a JSON POST with placeholders

```bash
gurl save "create-order" '{{BASE_URL}}/orders' \
  -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {{API_KEY}}" \
  -d '{"customer_id":"{{customerId}}","amount":"{{amount:num}}"}'
```

The quoted typed placeholder `"{{amount:num}}"` keeps the stored body valid JSON and sends a number. See [Variables and placeholders]({{< relref "/guides/variables" >}}).

### Save from a curl command

```bash
gurl save "me" --curl "curl -H 'Accept: application/json' https://api.example.com/me"
```

### Save auth and an extraction rule

```bash
gurl save "login" '{{BASE_URL}}/auth/login' \
  -X POST \
  --collection payments-api \
  --auth basic --auth-param username='{{USER}}' --auth-param password='{{PASSWORD}}' \
  --extract token=jsonpath:'$.token'
```

Use [`gurl auth info <type>`]({{< relref "/cli/auth" >}}) to see every parameter a handler accepts, and [`gurl extract`]({{< relref "/cli/extract" >}}) to test a rule against a sample response first.

## Related commands

- [`gurl run`]({{< relref "/cli/run" >}}) - Execute a saved request
- [`gurl edit`]({{< relref "/cli/edit" >}}) - Change a saved request
- [`gurl detect`]({{< relref "/cli/detect" >}}) - Save a curl command from stdin
- [`gurl auth`]({{< relref "/cli/auth" >}}) - Auth handlers and parameters
