---
title: "Header Redaction"
description: "How Gurl prevents credential leakage in logs"
weight: 3
---

Gurl's logging middleware automatically redacts sensitive headers from all log output. This ensures credentials never appear in logs even if an auth handler injects them into a request.

## Redacted Headers

The following headers are redacted in all log output:

- `Authorization`
- `Cookie`
- `Set-Cookie`
- `Proxy-Authorization`

Values are replaced with `[REDACTED]` in all log entries including request logs, response logs, and saved request history.

## Implementation

Header redaction uses a map-based lookup for case-insensitive matching. This avoids hardcoded if-else chains and makes the redaction list easy to extend.

```go
var sensitiveHeaders = map[string]bool{
    "authorization":       true,
    "cookie":              true,
    "set-cookie":          true,
    "proxy-authorization": true,
}

func redactHeader(name string) string {
    if sensitiveHeaders[strings.ToLower(name)] {
        return "[REDACTED]"
    }
    return name
}
```

The lookup is case-insensitive, so `authorization`, `Authorization`, and `AUTHORIZATION` all match.

## User-Agent Header

Gurl automatically sets the `User-Agent` header to `gurl/<version>` on all requests. This identifies your requests in server logs without exposing any credential information.

## Disabling Middleware

You can disable specific middleware via the config file if you need fine-grained control over what is logged:

```toml
[logging]
# Disable header redaction (not recommended)
redact_headers = true

# Set log level: debug, info, warn, error
level = "info"
```

> [!IMPORTANT]
> Disabling header redaction means Authorization, Cookie, and similar headers will appear in logs. Only disable if you have a specific auditing need and control access to those logs.

## Redaction in Practice

When a request with Bearer token authentication is logged, the output looks like this:

```
--> GET /api/users HTTP/1.1
    Host: api.example.com
    Authorization: [REDACTED]
    User-Agent: gurl/1.0.0

<-- 200 OK
    Content-Type: application/json
    Set-Cookie: [REDACTED]
    Body: {"users": [...]}
```

The response body is preserved intact. Only the headers containing credentials are modified.
