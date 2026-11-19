---
title: "apigent sequence"
description: "Manage request execution order in a collection"
---


Set the order in which `apigent collection run` executes a collection's requests.

## Synopsis

```bash
apigent sequence <subcommand> [arguments]
```

## Subcommands

### set

Set the execution order number for a request (`0` or higher; `0` means automatic).

```bash
apigent sequence set <request> <order>
```

None.

Aliases: `s`

### list

List a collection's requests in execution order.

```bash
apigent sequence list <collection>
```

None.

Aliases: `ls`, `l`

## Aliases

- `seq`

## Examples

### Order a checkout flow

```bash
apigent sequence set create-order 1
apigent sequence set capture-payment 2
apigent sequence set get-receipt 3
```

### Check the order

```bash
apigent sequence list checkout-flow
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
apigent collection run checkout-flow --env staging
```

## Related commands

- [`apigent collection`]({{< relref "/cli/collection" >}}) - `collection run`
- [Collections and data-driven runs]({{< relref "/guides/collection-runner" >}}) - Guide
