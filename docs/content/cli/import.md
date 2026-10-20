---
title: "gurl import"
description: "Import requests from OpenAPI, Insomnia, Bruno, Postman, HAR or gurl exports"
---


Import requests from OpenAPI, Insomnia, Bruno, Postman, HAR or a native gurl export. The format is detected from the file unless you force it.

## Synopsis

```bash
gurl import <path> [flags]
gurl import --list
```

Supported inputs (`gurl import --list`):

| Extension | Format |
|-----------|--------|
| `.yaml`, `.yml` | OpenAPI/Swagger 3.x |
| `.json` | Insomnia, Postman, OpenAPI, HAR |
| `.bru` | Bruno |
| `.har` | HAR (HTTP Archive) |
| `.gurl` | Native gurl request or collection export |

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Overwrite existing requests with the same name |
| `--format` | `--fmt` | auto-detected | Force a specific format |
| `--list` | `-l` | `false` | List all supported formats |
| `--passphrase` | | none | Passphrase for decrypting native collection exports (or set `GURL_IMPORT_PASSPHRASE`) |

## Aliases

- `imp`

## Examples

### Import an OpenAPI spec

```bash
gurl import ./api.yaml
```

### Re-import a Postman collection, overwriting existing requests

```bash
gurl import ./my-collection.json --force
```

### Import a teammate's collection export

```bash
gurl import ./payments.gurl --passphrase "$TEAM_SECRET"
```

## Related commands

- [`gurl export`]({{< relref "/cli/export" >}}) - Export requests
- [`gurl collection`]({{< relref "/cli/collection" >}}) - `collection import` for collection directories and `.env` files
- [`gurl detect`]({{< relref "/cli/detect" >}}) - Parse a single curl command
