---
title: "valetkey auth"
description: "Discover supported authentication types"
---


List the built-in auth handlers and the parameters each one accepts, so you know what to pass to `--auth` and `--auth-param` on `valetkey save` and `valetkey edit`.

## Synopsis

```bash
valetkey auth list
valetkey auth info <type>
```

`valetkey run` applies the auth saved on the request. Auth params may contain placeholders (`token='{{API_KEY}}'`); they are filled during request resolution, then the handler updates the outgoing request. Literal secret auth params are masked by `show`, `list --json`, `codegen` and `paste`.

| Type | Use it for |
|------|------------|
| `basic` | HTTP Basic auth with `username` and `password` |
| `bearer` | Bearer tokens in the `Authorization` header |
| `apikey` | API key in a header or query parameter |
| `oauth1` | OAuth 1.0 signed requests |
| `oauth2` | OAuth 2.0 `auth_code` or client credentials flow |
| `awsv4` | AWS SigV4 request signing |
| `digest` | Digest `Authorization` headers from supplied challenge values |
| `ntlm` | NTLM negotiate or challenge response headers |

## Subcommands

### list

List the built-in auth types.

```bash
valetkey auth list
```

None.

### info

Show the parameters for one auth type, marked required/optional and secret, with defaults.

```bash
valetkey auth info <type>
```

None.

## Examples

### See what a handler needs

```bash
valetkey auth info bearer
```

```text
Auth type: bearer
Parameters:
  token	required,secret	Bearer token value
```

### Save bearer auth that reads a secret from the environment

```bash
valetkey save "profile" '{{BASE_URL}}/me' \
  --auth bearer \
  --auth-param token='{{API_KEY}}'

valetkey run "profile" --env prod
```

### OAuth 2 client credentials

```bash
valetkey save "service-token" https://api.example.com/service \
  --auth oauth2 \
  --auth-param flow=client_credentials \
  --auth-param client_id='{{client_id}}' \
  --auth-param client_secret='{{client_secret}}' \
  --auth-param token_url='https://auth.example.com/oauth/token' \
  --auth-param scope='read:orders'
```

To change or clear saved auth, use `valetkey edit <name> --auth ...` or `valetkey edit <name> --auth none`.

## Related commands

- [`valetkey save`]({{< relref "/cli/save" >}}) - Save auth with a request
- [`valetkey edit`]({{< relref "/cli/edit" >}}) - Update or clear saved auth
- [`valetkey run`]({{< relref "/cli/run" >}}) - Execute a request with saved auth
- [Credential storage]({{< relref "/security/credential-storage" >}}) - Where auth params are stored
