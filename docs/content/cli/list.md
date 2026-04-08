---
title: "gurl list"
description: "List saved requests"
---

# gurl list

List all saved requests, optionally filtered by pattern, collection, or tag.

## Usage

```bash
gurl list [flags]
```

## Description

The `list` command displays saved requests with their names, URLs, methods, and collections. Use filters to narrow down results when you have many requests.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--pattern` | `-p` | none | Filter by name pattern |
| `--collection` | `-c` | none | Filter by collection name |
| `--tag` | `-t` | none | Filter by tag |
| `--json` | `-j` | `false` | Output as JSON |
| `--format` | `-f` | `table` | Output format: `table` or `list` |
| `--limit` | `-n` | none | Limit number of results |
| `--sort` | `-s` | `name` | Sort by: `name`, `updated`, or `collection` |

## Aliases

- `ls`
- `l`

## Examples

### List all requests

```bash
gurl list
```

Displays all saved requests in a table.

### Filter by collection

```bash
gurl list --collection "api"
```

Shows only requests in the "api" collection.

### Filter by tag

```bash
gurl list --tag "production"
```

Shows requests tagged with "production".

### JSON output

```bash
gurl list --json
```

Outputs all requests as JSON for programmatic use.

### Sort by last updated

```bash
gurl list --sort updated
```

Lists requests sorted by most recently updated.

## See also

- [`gurl save`](save) - Save a new request
- [`gurl delete`](delete) - Remove a request
- [`gurl collection`](collection) - Manage collections
