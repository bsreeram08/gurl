---
title: "gurl import"
description: "Import from external formats"
---

# gurl import

Import requests from external formats including HAR, OpenAPI, and common collection formats.

## Usage

```bash
gurl import [flags]
```

## Description

The `import` command converts requests from common API formats into gurl's format. It supports HAR files, OpenAPI specifications, and several collection formats.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Overwrite existing requests with the same name |
| `--format` | | none | Specify format explicitly (har, openapi, insomnia, postman, bruno) |
| `--list` | `-l` | `false` | List available collections/requests before importing |

## Aliases

- `imp`

## Examples

### Import from file

```bash
gurl import --format openapi ./api.yaml
```

Imports all endpoints from an OpenAPI spec.

### Import with overwrite

```bash
gurl import --format har ./requests.har --force
```

Imports and overwrites existing requests with the same names.

### List before importing

```bash
gurl import --format openapi ./api.yaml --list
```

Lists all endpoints in the OpenAPI file before importing.

### Import HAR file

```bash
gurl import --format har ./requests.har
```

Imports all requests from a HAR file.

## See also

- [`gurl export`](export) - Export requests to file
- [`gurl save`](save) - Save a new request
- [`gurl detect`](detect) - Parse curl commands
