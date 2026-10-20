---
title: "gurl detect"
description: "Parse a curl command from stdin or a file"
---


Parse a curl command from stdin or a file and save it as a named request.

## Synopsis

```bash
gurl detect [--file <path>] [--name <name>] [--collection <name>]
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | stdin | Read the curl command from a file instead of stdin |
| `--name` | `-n` | none | Name for the detected request |
| `--collection` | `-c` | none | Assign to a collection |

## Aliases

- `parse`

## Examples

### Parse a curl command from stdin

```bash
echo 'curl -X POST https://api.example.com/orders -H "Content-Type: application/json" -d "{}"' \
  | gurl detect --name create-order
```

### Parse a command saved in a file

```bash
gurl detect --file ./request.txt --name "my-request" --collection api
```

### Paste from the clipboard (macOS)

```bash
pbpaste | gurl detect --name from-browser
```

## Related commands

- [`gurl save`]({{< relref "/cli/save" >}}) - Save a request from flags or `--curl`
- [`gurl import`]({{< relref "/cli/import" >}}) - Import whole collections
- [`gurl edit`]({{< relref "/cli/edit" >}}) - Edit the saved request
