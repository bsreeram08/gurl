---
title: "gurl delete"
description: "Remove a saved request"
---


Permanently remove a saved request. This cannot be undone.

## Synopsis

```bash
gurl delete <name>
```

## Flags

None.

## Aliases

- `rm`
- `del`

## Examples

### Delete a request

```bash
gurl delete "old-request"
```

### Check the name first

```bash
gurl list --pattern old
gurl delete "old-request"
```

## Related commands

- [`gurl rename`]({{< relref "/cli/rename" >}}) - Rename a request instead
- [`gurl list`]({{< relref "/cli/list" >}}) - List saved requests
- [`gurl collection`]({{< relref "/cli/collection" >}}) - `collection delete` removes a whole collection
