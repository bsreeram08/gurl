---
title: "valetkey import"
description: "Import requests from OpenAPI, Insomnia, Bruno, Postman, HAR or valetkey exports"
---


Import requests from OpenAPI, Insomnia, Bruno, Postman, HAR or a native valetkey export. The format is detected from the file unless you force it.

## Synopsis

```bash
valetkey import <path> [flags]
valetkey import --list
```

Supported inputs (`valetkey import --list`):

| Extension | Format |
|-----------|--------|
| `.yaml`, `.yml` | OpenAPI/Swagger 3.x |
| `.json` | Insomnia, Postman, OpenAPI, HAR |
| `.bru` | Bruno |
| `.har` | HAR (HTTP Archive) |
| `.gurl` | Native valetkey request or collection export |

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Overwrite existing requests with the same name |
| `--format` | `--fmt` | auto-detected | Force a specific format |
| `--list` | `-l` | `false` | List all supported formats |
| `--passphrase` | | none | Passphrase for decrypting native collection exports (or set `VALETKEY_IMPORT_PASSPHRASE`) |

## Aliases

- `imp`

## Examples

### Import an OpenAPI spec

```bash
valetkey import ./api.yaml
```

### Re-import a Postman collection, overwriting existing requests

```bash
valetkey import ./my-collection.json --force
```

### Import a teammate's collection export

```bash
valetkey import ./payments.gurl --passphrase "$TEAM_SECRET"
```

## Related commands

- [`valetkey export`]({{< relref "/cli/export" >}}) - Export requests
- [`valetkey collection`]({{< relref "/cli/collection" >}}) - `collection import` for collection directories and `.env` files
- [`valetkey detect`]({{< relref "/cli/detect" >}}) - Parse a single curl command
