---
title: "gurl sequence"
description: "Manage request sequences"
---

# gurl sequence

Manage sequences for chaining multiple requests together.

## Usage

```bash
gurl sequence [subcommand] [flags]
```

## Description

Sequences allow you to define an ordered list of requests that run in sequence. The output of one request can be used as input for the next, enabling complex workflows and data-driven testing.

## Subcommands

### set

Add a request to a sequence.

```bash
gurl sequence set [sequence-name] [request-names...]
```

Aliases: `s`

### list

List all sequences.

```bash
gurl sequence list
```

Aliases: `ls`, `l`

## Examples

### Create a sequence

```bash
gurl sequence set "user-workflow" "login" "get-profile" "get-orders"
```

Creates a sequence called "user-workflow" that runs login, get-profile, and get-orders in order.

### List sequences

```bash
gurl sequence list
```

Shows all sequences and their requests.

## See also

- [`gurl run`](run) - Run a sequence
- [`gurl save`](save) - Save requests
- [`gurl env`](env) - Manage environments
