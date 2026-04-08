---
title: "Credential Storage"
description: "How Gurl stores and encrypts your API credentials"
weight: 2
---

Gurl separates secret storage from general configuration. Secrets are encrypted at rest using AES-256-GCM with a machine-specific key. General settings live in a plain TOML config file with no credentials.

## Environment Secrets (Encrypted)

Environment variables marked `--secret` are encrypted before storage. This applies to API keys, OAuth tokens, passwords, and any other value you want protected.

**Algorithm**: AES-256-GCM

**Machine key**: 256-bit key generated via `crypto/rand`, stored at `~/.local/share/gurl/.secret-key` with permissions 0600

**Nonce**: 12 bytes, randomly generated per encryption operation

> [!IMPORTANT]
> Only values marked `--secret` get encrypted. Regular environment variables are stored as plaintext. Mark anything sensitive.

**How it works**:

1. First launch generates a 256-bit key using `crypto/rand` and saves it to `~/.local/share/gurl/.secret-key`
2. File permissions are set to 0600 (owner read/write only)
3. When you create a secret: `gurl env create prod --secret "API_KEY=sk-xxx"`, the value is encrypted with AES-256-GCM and stored
4. At request execution time, the key is loaded, the value is decrypted, and it is injected into HTTP headers by an auth handler
5. The decrypted value lives only in memory during the request

**Machine dependency**: Because the key is stored locally, encrypted values cannot be decrypted on a different machine. If you copy your Gurl data to another machine, secrets are unreadable.

## Auth Credentials in Requests

Saved requests store auth configuration in an LMDB database at `~/.local/share/gurl/gurl.db`.

**Database**: LMDB (memory-mapped, transactional)

**Auth params**: Username, password, tokens stored as JSON in `SavedRequest.AuthConfig`

**Permissions**: Database file has owner-only permissions by default

**Future**: Keychain integration for macOS and Windows is under consideration.

## Config File

The TOML config file at `~/.config/gurl/config.toml` or `~/.gurlrc` contains general settings only.

**Stored settings**:

- `history_depth` (number of requests to keep in history)
- `output_format` (default output format)
- `timeout` (request timeout in seconds)
- `log_level` (logging verbosity)

**Never stored in config**:

- API keys
- OAuth tokens
- Passwords
- Any credential values

**Custom path**: Set `GURL_CONFIG_PATH` environment variable to use a different location.

## Security Comparison

| Concern | Gurl CLI | AI Agent Direct Handling |
|---------|----------|-------------------------|
| Credentials at rest | Env secrets encrypted AES-256-GCM | AI might store in plaintext context |
| Credentials in transit | Applied to headers by auth handlers | AI might log/expose in prompts |
| Logs | Sensitive headers redacted | AI might output full headers |
| Machine key | Per-machine 256-bit key (0600) | AI has no equivalent |
| Token caching | OAuth2 tokens cached in memory with TTL | AI might expose cached tokens |

The key difference is that Gurl treats credential handling as a separate concern from request execution. Auth handlers inject credentials at the HTTP transport layer, so the calling process never sees the raw values.
