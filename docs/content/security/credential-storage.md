---
title: "Credential Storage"
description: "How Gurl stores and encrypts your API credentials"
weight: 2
---

Gurl separates secret storage from general configuration. Secrets are encrypted at rest using AES-256-GCM with a machine-specific key. General settings live in a plain TOML config file with no credentials.

## Environment Secrets (Encrypted)

Environment variables marked `--secret` are encrypted before storage in both the default database and file-backed project environments under `.gurl/environments/`. This applies to API keys, OAuth tokens, passwords, and any other value you want protected.

**Algorithm**: AES-256-GCM

**Machine key**: 256-bit key generated via `crypto/rand`, stored at `~/.local/share/gurl/.secret-key` with permissions 0600

**Nonce**: 12 bytes, randomly generated per encryption operation

> [!IMPORTANT]
> Only values marked `--secret` get encrypted. Regular environment variables are stored as plaintext. Mark anything sensitive.

**How it works**:

1. First launch generates a 256-bit key using `crypto/rand` and saves it to `~/.local/share/gurl/.secret-key`
2. File permissions are set to 0600 (owner read/write only)
3. When you create a secret: `gurl env create prod --secret "API_KEY=sk-xxx"`, the value is encrypted with AES-256-GCM and stored
4. At request execution time, the key is loaded, the value is decrypted, and templates such as `{{API_KEY}}` can be used by request fields or auth parameters
5. The decrypted value lives only in memory during the request

**Machine dependency**: Because the key is stored locally, encrypted values cannot be decrypted on a different machine. If you copy your Gurl data to another machine, secrets are unreadable.

## Collection Secrets (Encrypted)

File-backed collections store variables in `.gurl/collections/<collection>/collection.json`. Variables marked as collection secrets are encrypted with a per-collection AES-256-GCM key before they are written.

**Canonical local key**: `.gurl/collections/<collection>/collection.key`

**Git behavior**: `gurl init` writes `.gurl/.gitignore` so `collection.key` files stay local. Commit `collection.json` and request files, but do not commit local collection keys.

**Clone/share behavior**:

1. Local use creates `collection.key` automatically when a collection secret is saved.
2. A clone without `collection.key` can read non-secret collection variables and request files, but encrypted collection secrets remain locked.
3. To share secrets, export the collection with a passphrase: `gurl collection export <name> --passphrase ... --output team.gurl`.
4. The receiver imports or unlocks with the passphrase. Gurl decrypts the passphrase-protected values and re-encrypts them with that machine's local `collection.key`.

Passphrase exports use PBKDF2-SHA256 with a per-export salt and AES-256-GCM for secret values. `--passphrase` is available for interactive use, and `GURL_IMPORT_PASSPHRASE` can provide the passphrase in CI.

## Auth Credentials in Requests

Saved requests store auth configuration in a goleveldb database at `~/.local/share/gurl/gurl.db`. Save or edit auth with `--auth` and repeated `--auth-param key=value` flags.

**Database**: goleveldb (LevelDB port, memory-mapped, transactional)

**Auth params**: Values are stored as JSON in `SavedRequest.AuthConfig`. If you put a literal password or token in `--auth-param`, it is saved in the request database. Prefer templates that point at encrypted environment secrets, such as `--auth-param token='{{API_TOKEN}}'`.

**Permissions**: Database file has owner-only permissions by default

At execution time, gurl substitutes templates in auth parameters before the auth handler applies credentials to the outgoing request. Saved auth can be replaced with `gurl edit <name> --auth ...` or cleared with `gurl edit <name> --auth none`.

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

The key difference is that Gurl treats credential handling as a separate concern from request execution. Auth handlers apply credentials at the HTTP request layer, and templated auth params let saved requests refer to encrypted environment secrets instead of storing raw secret values in the request.
