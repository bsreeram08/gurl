---
title: "valetkey detect"
description: "Parse a curl command from stdin or a file"
---


Parse a curl command from stdin or a file and save it as a named request.

## Synopsis

```bash
valetkey detect [--file <path>] [--name <name>] [--collection <name>]
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
  | valetkey detect --name create-order
```

### Parse a command saved in a file

```bash
valetkey detect --file ./request.txt --name "my-request" --collection api
```

### Paste from the clipboard (macOS)

```bash
pbpaste | valetkey detect --name from-browser
```

## Related commands

- [`valetkey save`]({{< relref "/cli/save" >}}) - Save a request from flags or `--curl`
- [`valetkey import`]({{< relref "/cli/import" >}}) - Import whole collections
- [`valetkey edit`]({{< relref "/cli/edit" >}}) - Edit the saved request
