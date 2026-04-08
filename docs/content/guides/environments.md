---
title: "Environments"
description: "Manage variables and secrets across deployment stages"
weight: 1
---

# Environments

Environments let you manage variables and secrets for different deployment stages. Each environment contains a set of key-value pairs that are substituted into requests at runtime.

## Creating Environments

Create an environment with `gurl env create`:

```bash
gurl env create dev --var "BASE_URL=https://dev.api.com" --var "TIMEOUT=30"
gurl env create staging --var "BASE_URL=https://staging.api.com" --var "TIMEOUT=60"
gurl env create prod --var "BASE_URL=https://api.example.com" --var "TIMEOUT=120"
```

For secrets, use `--secret` to encrypt the value:

```bash
gurl env create prod --secret "API_KEY=sk-live-xxxxx" --secret "WEBHOOK_SECRET=whsec_xxxxx"
```

## Switching Environments

Set the active environment with `gurl env switch`:

```bash
gurl env switch staging
```

View the current environment:

```bash
gurl env current
# Output: staging
```

List all environments:

```bash
gurl env list
# dev
# staging (active)
# prod
```

## Using Environments in Requests

Reference variables in your TOML request files using template syntax:

```toml
# requests/user.toml
[request]
method = "GET"
url = "{{BASE_URL}}/users"

[headers]
Authorization = "Bearer {{API_KEY}}"
X-Request-Timeout = "{{TIMEOUT}}"
```

Run with a specific environment:

```bash
gurl run "user" --env dev
```

Variable substitution replaces `{{VAR_NAME}}` with the corresponding value from the active environment.

## Variable Precedence

Variables are resolved in this order (highest to lowest):

| Source | Example | Override Flag |
|--------|---------|---------------|
| CLI variables | `--var "BASE_URL=https://custom.com"` | `--var` |
| Active environment | `gurl env switch prod` | `--env` |
| Template defaults | `{{BASE_URL default "https://api.com"}}` | none |

## Importing from .env Files

Import variables from a `.env` file:

```bash
gurl env import --file .env.production
gurl env import --file .env.staging --env staging
```

The import command reads key-value pairs from the file and adds them to the specified environment (or creates a new one).

## Secret Encryption

Secrets are encrypted at rest using AES-256-GCM. The encryption key is derived from your machine's credentials via the keychain.

> [!WARNING]
> Secrets are decrypted only when needed and only in memory. Never share your `gurl.db` file, as it contains encrypted secrets that could be brute-forced given sufficient time.

## Environment Management Commands

| Command | Description |
|---------|-------------|
| `gurl env create <name>` | Create a new environment |
| `gurl env list` | List all environments |
| `gurl env switch <name>` | Switch active environment |
| `gurl env current` | Show current environment |
| `gurl env delete <name>` | Delete an environment |
| `gurl env export <name>` | Export environment variables |
| `gurl env import --file <path>` | Import from .env file |
| `gurl env set <key> <value>` | Set a variable in active env |
| `gurl env unset <key>` | Remove a variable |
