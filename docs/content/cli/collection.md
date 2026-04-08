---
title: "gurl collection"
description: "Manage request collections"
---

# gurl collection

Manage collections for organizing saved requests.

## Usage

```bash
gurl collection [subcommand] [flags]
```

## Description

Collections group related requests together. Use collections to organize requests by project, API, or any other logical grouping.

## Subcommands

### list

List all collections.

```bash
gurl collection list
```

Aliases: `ls`, `l`

### add

Create a new collection.

```bash
gurl collection add [name]
```

Aliases: `create`, `new`

### remove

Delete a collection.

```bash
gurl collection remove [name]
```

Aliases: `rm`, `delete`, `del`

### rename

Rename a collection.

```bash
gurl collection rename [old-name] [new-name]
```

Aliases: `mv`, `ren`

## Examples

### List collections

```bash
gurl collection list
```

Shows all collections and request counts.

### Create a collection

```bash
gurl collection add "api-v2"
```

Creates a new collection called "api-v2".

### Rename a collection

```bash
gurl collection rename "api" "api-v3"
```

Renames "api" to "api-v3".

### Delete a collection

```bash
gurl collection remove "old-api"
```

Deletes the "old-api" collection.

## See also

- [`gurl save`](save) - Save a request to a collection
- [`gurl list`](list) - List requests in a collection
- [`gurl env`](env) - Manage environments
