---
title: "gurl codegen"
description: "Generate code from a saved request"
---

# gurl codegen

Generate code in various programming languages from a saved request.

## Usage

```bash
gurl codegen [name] [flags]
```

## Description

The `codegen` command converts a saved request into code for various languages. This helps you integrate API calls into your codebase without manual translation.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--lang` | `-l` | none | Target language: `go`, `python`, `javascript`, or `curl` |
| `--clipboard` | `-c` | `false` | Copy output to clipboard instead of printing |

## Aliases

- `cg`

## Examples

### Generate Go code

```bash
gurl codegen "my-request" --lang go
```

Generates Go code for the request.

### Generate Python code

```bash
gurl codegen "api-users" --lang python
```

Generates Python requests library code.

### Generate JavaScript code

```bash
gurl codegen "create-user" --lang javascript
```

Generates JavaScript fetch code.

### Copy to clipboard

```bash
gurl codegen "my-request" --lang curl --clipboard
```

Generates curl command and copies to clipboard.

## See also

- [`gurl paste`](paste) - Copy as curl to clipboard
- [`gurl save`](save) - Save a new request
- [`gurl export`](export) - Export requests to file
