---
title: "valetkey delete"
description: "Remove a saved request"
---


Permanently remove a saved request. This cannot be undone.

## Synopsis

```bash
valetkey delete <name>
```

## Flags

None.

## Aliases

- `rm`
- `del`

## Examples

### Delete a request

```bash
valetkey delete "old-request"
```

### Check the name first

```bash
valetkey list --pattern old
valetkey delete "old-request"
```

## Related commands

- [`valetkey rename`]({{< relref "/cli/rename" >}}) - Rename a request instead
- [`valetkey list`]({{< relref "/cli/list" >}}) - List saved requests
- [`valetkey collection`]({{< relref "/cli/collection" >}}) - `collection delete` removes a whole collection
