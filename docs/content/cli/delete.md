---
title: "apigent delete"
description: "Remove a saved request"
---


Permanently remove a saved request. This cannot be undone.

## Synopsis

```bash
apigent delete <name>
```

## Flags

None.

## Aliases

- `rm`
- `del`

## Examples

### Delete a request

```bash
apigent delete "old-request"
```

### Check the name first

```bash
apigent list --pattern old
apigent delete "old-request"
```

## Related commands

- [`apigent rename`]({{< relref "/cli/rename" >}}) - Rename a request instead
- [`apigent list`]({{< relref "/cli/list" >}}) - List saved requests
- [`apigent collection`]({{< relref "/cli/collection" >}}) - `collection delete` removes a whole collection
