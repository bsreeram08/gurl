---
title: "gurl detect"
description: "Parse curl from stdin or file"
---

# gurl detect

Parse curl command syntax from stdin or a file and convert it to a saved request.

## Usage

```bash
gurl detect [flags]
```

## Description

The `detect` command reads a curl command from stdin or a file and parses it into a saved request. This is useful when you have curl commands from other sources and want to save them in gurl.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | none | Path to file containing curl command |
| `--name` | `-n` | none | Name for the saved request |
| `--collection` | `-c` | none | Collection to assign the request to |

## Aliases

- `parse`
- `d`

## Examples

### Parse from stdin

```bash
curl -s https://api.example.com/users | gurl detect
```

Parses curl output from stdin and prompts for a name.

### Parse from file

```bash
gurl detect --file ./curl-commands.txt
```

Reads curl commands from a file and saves them.

### Parse and assign collection

```bash
cat ./request.txt | gurl detect --name "my-request" --collection "api"
```

Parses stdin and saves as "my-request" in the "api" collection.

## See also

- [`gurl save`](save) - Save a new request
- [`gurl edit`](edit) - Edit a saved request
- [`gurl import`](import) - Import from external formats
