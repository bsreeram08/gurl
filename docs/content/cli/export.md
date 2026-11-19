---
title: "apigent export"
description: "Export requests to a file"
---


Write one request, one collection or every request to a file (or stdout) in apigent's native format.

## Synopsis

```bash
apigent export (--name <request> | --collection <name> | --all) [--output <file>] [--passphrase <pass>]
```

A collection export includes the collection's variables and request membership. Collection secrets are encrypted with `--passphrase` and can be imported with [`apigent import`]({{< relref "/cli/import" >}}) or `apigent collection import`.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | none | Export one request by name |
| `--collection` | `-c` | none | Export an entire collection |
| `--all` | `-a` | `false` | Export all requests |
| `--output` | `-o` | stdout | Output file |
| `--format` | `--fmt` | `apigent` | Export format (only `apigent` is supported) |
| `--passphrase` | | none | Passphrase for encrypting collection secrets |

## Aliases

- `exp`

## Examples

### Export one request to stdout

```bash
apigent export --name get-user
```

### Share a collection with its secrets

```bash
apigent export --collection payments-api --passphrase "$TEAM_SECRET" --output payments-api.gurl
```

### Back up everything

```bash
apigent export --all --output backup.gurl
```

## Related commands

- [`apigent import`]({{< relref "/cli/import" >}}) - Import requests and collection exports
- [`apigent share`]({{< relref "/cli/share" >}}) - Encrypt an export to one person's public key instead of a shared passphrase
- [`apigent collection`]({{< relref "/cli/collection" >}}) - `collection export` / `collection import`
- [Credential storage]({{< relref "/security/credential-storage" >}}) - How passphrase exports are encrypted
