---
title: "apigent import"
description: "Import requests from OpenAPI, Insomnia, Bruno, Postman, HAR or apigent exports"
---


Import requests from OpenAPI, Insomnia, Bruno, Postman, HAR or a native apigent export. The format is detected from the file unless you force it with `--format`: an extension used by one format (`.bru`, `.har`, `.gurl`) decides on its own, and for `.json` the document's top-level keys decide (`openapi`, Postman `info` + `item`, Insomnia `_type`/`resources`, HAR `log`, Yaak `collections`).

## Synopsis

```bash
apigent import <path> [flags]
apigent import --list
```

Supported inputs (`apigent import --list`):

| Extension | Format |
|-----------|--------|
| `.yaml`, `.yml` | OpenAPI/Swagger 3.x |
| `.json` | OpenAPI, Postman, Insomnia, HAR, Yaak (detected from the content) |
| `.bru` | Bruno |
| `.har` | HAR (HTTP Archive) |
| `.gurl` | Native apigent request or collection export |

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Overwrite existing requests with the same name |
| `--format` | `--fmt` | detected | Import as `openapi`, `postman`, `insomnia`, `har`, `bruno`, `yaak` or `gurl` |
| `--list` | `-l` | `false` | List all supported formats |
| `--base-url` | | spec server | OpenAPI only: URL to use instead of the spec's first server, e.g. `'{{BASE_URL}}'` |
| `--collection` | `-c` | none | Put the requests in this collection, created if missing (in the `.apigent/` project when there is one) |
| `--passphrase` | | none | Passphrase for decrypting native collection exports (or set `APIGENT_IMPORT_PASSPHRASE`) |

## Aliases

- `imp`

## Examples

### Import an OpenAPI spec

```bash
apigent import ./api.yaml
```

### Import an OpenAPI spec into a project collection that follows the environment

```bash
apigent init
apigent import ./openapi.json --collection my-api --base-url '{{BASE_URL}}' --force
apigent env create local --var BASE_URL=http://localhost:3000 --var token=- --secret token
apigent run getUserById --env local --var id=42
```

An imported OpenAPI request is runnable as is: path parameters and required query parameters are `{{name}}` placeholders, bearer auth uses `{{token}}`, and JSON bodies are filled from the schema's examples, defaults and types.

### Re-import a Postman collection, overwriting existing requests

```bash
apigent import ./my-collection.json --force
```

### Import a teammate's collection export

```bash
apigent import ./payments.gurl --passphrase "$TEAM_SECRET"
```

## Related commands

- [`apigent export`]({{< relref "/cli/export" >}}) - Export requests
- [`apigent collection`]({{< relref "/cli/collection" >}}) - `collection import` for collection directories and `.env` files
- [`apigent detect`]({{< relref "/cli/detect" >}}) - Parse a single curl command
