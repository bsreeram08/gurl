---
title: "gurl paste"
description: "Copy request as curl to clipboard"
---

# gurl paste

Copy a saved request to the clipboard as a curl command.

## Usage

```bash
gurl paste [name]
```

## Description

The `paste` command copies the specified request to your clipboard as a fully-formed curl command. This is useful for sharing requests or running them directly with curl.

## Flags

None.

## Aliases

- `clip`
- `copy`

## Examples

### Copy request as curl

```bash
gurl paste "my-request"
```

Copies "my-request" as a curl command to the clipboard.

### Paste and run

```bash
gurl paste "my-request" | bash
```

Copies the request and pipes it to bash for execution.

## See also

- [`gurl save`](save) - Save a new request
- [`gurl export`](export) - Export requests to file
- [`gurl codegen`](codegen) - Generate code in other languages
