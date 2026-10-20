---
title: "gurl rename"
description: "Rename a saved request"
---


Give a saved request a new name. Scripts that call `gurl.setNextRequest("old-name")` are not updated, so change them too.

## Synopsis

```bash
gurl rename <old-name> <new-name>
```

## Flags

None.

## Aliases

- `mv`
- `ren`

## Examples

### Rename a request

```bash
gurl rename "get user" "get-user"
```

### Rename, then confirm

```bash
gurl rename "orders" "list-orders"
gurl show "list-orders"
```

## Related commands

- [`gurl edit`]({{< relref "/cli/edit" >}}) - Change other fields
- [`gurl delete`]({{< relref "/cli/delete" >}}) - Remove a request
- [`gurl collection`]({{< relref "/cli/collection" >}}) - `collection rename` renames a collection
