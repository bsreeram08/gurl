---
title: "gurl auth"
description: "Discover supported authentication types"
---

# gurl auth

Discover the auth handlers built into gurl and see the parameters each handler accepts.

## Usage

```bash
gurl auth list
gurl auth info <type>
```

## Description

Use `gurl auth` when you need the exact value for `--auth` or the supported `--auth-param key=value` pairs for a handler. `--auth-param` can be repeated on `gurl save` and `gurl edit`.

`gurl run` uses the auth config already saved on the request. Auth parameters are template-substituted during execution, after the request URL, headers, and body are prepared and before the auth handler updates the outgoing request.

## Built-in auth types

Use the CLI type names in the first column with `--auth`.

| CLI type | Use it for |
|----------|------------|
| `basic` | HTTP Basic auth with `username` and `password` |
| `bearer` | Bearer tokens in the `Authorization` header |
| `apikey` | API key auth in a header or query parameter |
| `oauth1` | OAuth 1.0 signed requests |
| `oauth2` | OAuth 2.0 `auth_code` or client credentials flow |
| `awsv4` | AWS SigV4 request signing |
| `digest` | Digest `Authorization` headers from supplied challenge values |
| `ntlm` | NTLM negotiate or challenge response headers |

## Commands

### List auth types

```bash
gurl auth list
```

Prints the built-in auth types:

```text
Built-in auth types:
apikey
awsv4
basic
bearer
digest
ntlm
oauth1
oauth2
```

### Show handler parameters

```bash
gurl auth info bearer
gurl auth info oauth2
gurl auth info awsv4
```

The output marks each parameter as required or optional, shows defaults where they exist, and marks secret values.

## Examples

### Bearer token

```bash
gurl save "profile" https://api.example.com/me \
  --auth bearer \
  --auth-param token='{{token}}'

gurl run "profile" --var token=abc123
```

### Basic auth

```bash
gurl save "admin" https://api.example.com/admin \
  --auth basic \
  --auth-param username='{{user}}' \
  --auth-param password='{{password}}'
```

### API key header

```bash
gurl save "search" https://api.example.com/search \
  --auth apikey \
  --auth-param header=X-API-Key \
  --auth-param value='{{api_key}}'
```

### API key query parameter

```bash
gurl save "legacy-search" https://api.example.com/search \
  --auth apikey \
  --auth-param in=query \
  --auth-param key='{{api_key}}' \
  --auth-param param_name=api_key
```

### OAuth 2 client credentials

```bash
gurl save "service-token" https://api.example.com/service \
  --auth oauth2 \
  --auth-param flow=client_credentials \
  --auth-param client_id='{{client_id}}' \
  --auth-param client_secret='{{client_secret}}' \
  --auth-param token_url='https://auth.example.com/oauth/token' \
  --auth-param scope='read:orders'
```

### AWS SigV4

```bash
gurl save "aws-api" https://example.execute-api.us-east-1.amazonaws.com/prod/items \
  --auth awsv4 \
  --auth-param access_key='{{AWS_ACCESS_KEY_ID}}' \
  --auth-param secret_key='{{AWS_SECRET_ACCESS_KEY}}' \
  --auth-param region=us-east-1 \
  --auth-param service=execute-api
```

## Changing saved auth

Use `gurl edit` to replace or clear saved auth:

```bash
gurl edit "profile" --auth bearer --auth-param token='{{new_token}}'
gurl edit "profile" --auth none
```

`--auth none` clears auth from the saved request and cannot be combined with `--auth-param`.

## See also

- [`gurl save`](save) - Save auth with a request
- [`gurl run`](run) - Execute a request with saved auth
- [`gurl edit`](edit) - Update or clear saved auth
