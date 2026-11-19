---
title: "apigent rename"
description: "Rename a saved request"
---


Give a saved request a new name. Scripts that call `apigent.setNextRequest("old-name")` are not updated, so change them too.

## Synopsis

```bash
apigent rename <old-name> <new-name>
```

## Flags

None.

## Aliases

- `mv`
- `ren`

## Examples

### Rename a request

```bash
apigent rename "get user" "get-user"
```

### Rename, then confirm

```bash
apigent rename "orders" "list-orders"
apigent show "list-orders"
```

## Related commands

- [`apigent edit`]({{< relref "/cli/edit" >}}) - Change other fields
- [`apigent delete`]({{< relref "/cli/delete" >}}) - Remove a request
- [`apigent collection`]({{< relref "/cli/collection" >}}) - `collection rename` renames a collection
