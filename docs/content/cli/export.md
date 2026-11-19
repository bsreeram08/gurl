---
title: "valetkey export"
description: "Export requests to a file"
---


Write one request, one collection or every request to a file (or stdout) in valetkey's native format.

## Synopsis

```bash
valetkey export (--name <request> | --collection <name> | --all) [--output <file>] [--passphrase <pass>]
```

A collection export includes the collection's variables and request membership. Collection secrets are encrypted with `--passphrase` and can be imported with [`valetkey import`]({{< relref "/cli/import" >}}) or `valetkey collection import`.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | none | Export one request by name |
| `--collection` | `-c` | none | Export an entire collection |
| `--all` | `-a` | `false` | Export all requests |
| `--output` | `-o` | stdout | Output file |
| `--format` | `--fmt` | `valetkey` | Export format (only `valetkey` is supported) |
| `--passphrase` | | none | Passphrase for encrypting collection secrets |

## Aliases

- `exp`

## Examples

### Export one request to stdout

```bash
valetkey export --name get-user
```

### Share a collection with its secrets

```bash
valetkey export --collection payments-api --passphrase "$TEAM_SECRET" --output payments-api.gurl
```

### Back up everything

```bash
valetkey export --all --output backup.gurl
```

## Related commands

- [`valetkey import`]({{< relref "/cli/import" >}}) - Import requests and collection exports
- [`valetkey share`]({{< relref "/cli/share" >}}) - Encrypt an export to one person's public key instead of a shared passphrase
- [`valetkey collection`]({{< relref "/cli/collection" >}}) - `collection export` / `collection import`
- [Credential storage]({{< relref "/security/credential-storage" >}}) - How passphrase exports are encrypted
