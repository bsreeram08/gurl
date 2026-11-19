---
title: "apigent codegen"
description: "Generate client code from a saved request"
---


Turn a saved request into Go, Python, JavaScript or curl code. Literal secrets are masked by default.

## Synopsis

```bash
apigent codegen <name> --lang <go|python|javascript|curl> [flags]
```

Literal secret header values, sensitive query parameters and secret auth params are masked as `*****`. With `--reveal` they are included, except `Authorization`, which always becomes a placeholder (`<your-token-here>`). `-l curl` quotes each header as one argument: `-H 'Content-Type: application/json'`.

`--clipboard` and `--reveal` are refused in [agent mode]({{< relref "/ai/agent-mode" >}}); there, codegen prints masked code to stdout.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--lang` | `-l` | none | Language to generate: `go`, `python`, `javascript`, `curl` |
| `--clipboard` | `-c` | `false` | Copy the output to the clipboard instead of printing it |
| `--reveal` | | `false` | Include secret header and auth values instead of masking them (Authorization always becomes a placeholder) |

## Aliases

- `cg`

## Examples

### Python client code

```bash
apigent codegen "create-order" --lang python
```

### A curl command with secrets masked

```bash
apigent codegen literal -l curl
```

```text
curl -X 'GET' -H 'Authorization: <your-token-here>' 'http://127.0.0.1:18731/x?token=*****'
```

### Copy JavaScript to the clipboard

```bash
apigent codegen "create-order" -l javascript --clipboard
```

## Related commands

- [`apigent paste`]({{< relref "/cli/paste" >}}) - Copy a request as curl
- [`apigent show`]({{< relref "/cli/show" >}}) - Show a saved request
- [`apigent export`]({{< relref "/cli/export" >}}) - Export requests to a file
