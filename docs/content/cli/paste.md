---
title: "apigent paste"
description: "Copy a request as a curl command to the clipboard"
---


Copy a saved request to the clipboard as a curl command, with literal secrets masked.

## Synopsis

```bash
apigent paste <name> [--reveal]
```

Literal secret header values, sensitive query parameters and secret auth params are masked as `*****` unless you pass `--reveal`. `paste` is refused in [agent mode]({{< relref "/ai/agent-mode" >}}) because it writes to the clipboard; use `apigent show <name> --format curl` instead.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--reveal` | | `false` | Include secret header and auth values instead of masking them |

## Aliases

- `clip`
- `copy`

## Examples

### Copy a request to share

```bash
apigent paste "create-order"
```

### Copy with real credentials (for your own terminal only)

```bash
apigent paste "create-order" --reveal
```

## Related commands

- [`apigent codegen`]({{< relref "/cli/codegen" >}}) - Generate code in other languages
- [`apigent show`]({{< relref "/cli/show" >}}) - Print the request as curl (`--format curl`)
- [`apigent export`]({{< relref "/cli/export" >}}) - Export requests to a file
