---
title: "gurl sequence"
description: "Manage request execution order in a collection"
---


Set the order in which `gurl collection run` executes a collection's requests.

## Synopsis

```bash
gurl sequence <subcommand> [arguments]
```

## Subcommands

### set

Set the execution order number for a request (`0` or higher; `0` means automatic).

```bash
gurl sequence set <request> <order>
```

None.

Aliases: `s`

### list

List a collection's requests in execution order.

```bash
gurl sequence list <collection>
```

None.

Aliases: `ls`, `l`

## Aliases

- `seq`

## Examples

### Order a checkout flow

```bash
gurl sequence set create-order 1
gurl sequence set capture-payment 2
gurl sequence set get-receipt 3
```

### Check the order

```bash
gurl sequence list checkout-flow
```

```text
┌─ checkout-flow Sequence ───────────────────────────┐
│  #   REQUEST                    ORDER                │
├─────────────────────────────────────────────────────┤
│  1   create-order             1
│  2   capture-payment          2
│  3   get-receipt              3
└─────────────────────────────────────────────────────┘
```

### Run the flow in that order

```bash
gurl collection run checkout-flow --env staging
```

## Related commands

- [`gurl collection`]({{< relref "/cli/collection" >}}) - `collection run`
- [Collections and data-driven runs]({{< relref "/guides/collection-runner" >}}) - Guide
