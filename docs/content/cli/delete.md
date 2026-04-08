---
title: "gurl delete"
description: "Remove a saved request"
---

# gurl delete

Remove a saved request from gurl's storage.

## Usage

```bash
gurl delete [name]
```

## Description

The `delete` command permanently removes a saved request. This action cannot be undone.

## Flags

None.

## Aliases

- `rm`
- `del`
- `d`

## Examples

### Delete a request

```bash
gurl delete "old-request"
```

Removes the request named "old-request".

## See also

- [`gurl save`](save) - Save a new request
- [`gurl list`](list) - List saved requests
- [`gurl rename`](rename) - Rename a request
