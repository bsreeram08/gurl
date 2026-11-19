---
title: "valetkey rename"
description: "Rename a saved request"
---


Give a saved request a new name. Scripts that call `valetkey.setNextRequest("old-name")` are not updated, so change them too.

## Synopsis

```bash
valetkey rename <old-name> <new-name>
```

## Flags

None.

## Aliases

- `mv`
- `ren`

## Examples

### Rename a request

```bash
valetkey rename "get user" "get-user"
```

### Rename, then confirm

```bash
valetkey rename "orders" "list-orders"
valetkey show "list-orders"
```

## Related commands

- [`valetkey edit`]({{< relref "/cli/edit" >}}) - Change other fields
- [`valetkey delete`]({{< relref "/cli/delete" >}}) - Remove a request
- [`valetkey collection`]({{< relref "/cli/collection" >}}) - `collection rename` renames a collection
