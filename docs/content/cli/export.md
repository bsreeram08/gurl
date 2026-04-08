---
title: "gurl export"
description: "Export requests to file"
---

# gurl export

Export saved requests to a file in various formats.

## Usage

```bash
gurl export [flags]
```

## Description

The `export` command writes one or more requests to a file. You can export individual requests, entire collections, or all requests.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | none | Export a specific request by name |
| `--collection` | `-c` | none | Export an entire collection |
| `--all` | `-a` | `false` | Export all requests |
| `--output` | `-o` | stdout | Output file path |

## Aliases

- `exp`

## Examples

### Export a single request

```bash
gurl export --name "my-request" --output my-request.gurl
```

Exports the "my-request" to a file.

### Export a collection

```bash
gurl export --collection "api" --output api-collection.gurl
```

Exports all requests in the "api" collection.

### Export all requests

```bash
gurl export --all --output backup.gurl
```

Exports all saved requests to a backup file.

## See also

- [`gurl import`](import) - Import requests from external formats
- [`gurl save`](save) - Save a new request
- [`gurl paste`](paste) - Copy as curl to clipboard
