---
title: "gurl rename"
description: "Rename a saved request"
---

# gurl rename

Rename a saved request to a new name.

## Usage

```bash
gurl rename [old-name] [new-name]
```

## Description

The `rename` command changes the name of a saved request. All references to the old name remain valid until you update them.

## Flags

None.

## Aliases

- `mv`
- `ren`

## Examples

### Rename a request

```bash
gurl rename "old-name" "new-name"
```

Changes the request name from "old-name" to "new-name".

## See also

- [`gurl save`](save) - Save a new request
- [`gurl delete`](delete) - Remove a request
- [`gurl edit`](edit) - Edit a request
