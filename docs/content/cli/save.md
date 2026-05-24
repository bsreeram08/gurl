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
| `--extract` | | none | Add extraction rule as `VAR_NAME=METHOD:EXPRESSION` |
| `--pre-script` | `--pre` | none | Set pre-request script |
| `--post-script` | `--post` | none | Set post-response script |
| `--auth` | | none | Authentication type: `basic`, `bearer`, `apikey`, `oauth1`, `oauth2`, `awsv4`, `digest`, `ntlm`, or `none` |
| `--auth-param` | | none | Authentication parameter as `key=value` (can be specified multiple times) |

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

The collection must already exist. In an interactive terminal, gurl asks whether to create a missing collection. In non-interactive scripts and CI, create it first with `gurl collection create`.

### Save extraction and scripts

```bash
gurl save "login" https://api.example.com/auth/login \
  -X POST \
  --extract token=jsonpath:$.token \
  --pre-script "request.headers.set('X-Client', 'gurl')" \
  --post-script "gurl.setNextRequest('profile')"
```

Stores extraction and script metadata with the request. Later runs can use extracted or script-set variables in assertions, request templates, and chained requests.

### Save with bearer auth

```bash
gurl save "profile" https://api.example.com/me \
  --auth bearer \
  --auth-param token='{{token}}'
```

Stores bearer auth with the request. Later, `gurl run "profile" --var token=abc123` substitutes the token and applies the `Authorization` header.

### Save with API key auth

```bash
gurl save "search" https://api.example.com/search \
  --auth apikey \
  --auth-param header=X-API-Key \
  --auth-param value='{{api_key}}'
```

This stores a header-based API key. For the legacy query parameter form, use `--auth-param in=query`, `--auth-param key='{{api_key}}'`, and optionally `--auth-param param_name=api_key`.

### Save with OAuth 2 client credentials

```bash
gurl save "service-token" https://api.example.com/service \
  --auth oauth2 \
  --auth-param flow=client_credentials \
  --auth-param client_id='{{client_id}}' \
  --auth-param client_secret='{{client_secret}}' \
  --auth-param token_url='https://auth.example.com/oauth/token' \
  --auth-param scope='read:orders'
```

The `oauth2` handler fetches an access token and applies it as a bearer token. Use `gurl auth info oauth2` to see every supported parameter.

### Save with AWS SigV4

```bash
gurl save "aws-api" https://example.execute-api.us-east-1.amazonaws.com/prod/items \
  --auth awsv4 \
  --auth-param access_key='{{AWS_ACCESS_KEY_ID}}' \
  --auth-param secret_key='{{AWS_SECRET_ACCESS_KEY}}' \
  --auth-param region=us-east-1 \
  --auth-param service=execute-api
```

The `awsv4` handler signs the outgoing request and adds the AWS authorization headers during execution.

## See also

- [`gurl run`](run) - Execute a saved request
- [`gurl auth`](auth) - Discover auth handlers and parameters
- [`gurl list`](list) - List saved requests
- [`gurl edit`](edit) - Edit a saved request
