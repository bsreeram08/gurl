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

### show

Show requests in a collection.

```bash
gurl collection show [name]
```

Aliases: `view`, `info`

### run

Run every request in a collection.

```bash
gurl collection run [name] [flags]
```

Common flags:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--env` | `-e` | none | Environment to use |
| `--var` | `-v` | none | Variable as `key=value` |
| `--data` | `-D` | none | CSV or JSON data file |
| `--bail` | `-b` | `false` | Stop on first failure |
| `--assert-bail` | | `false` | Stop on first assertion failure only |
| `--persist` | | `false` | Persist extracted/script variables back to the selected environment |
| `--dry-run` | | `false` | Preview collection execution without sending requests |
| `--reporter` | `-R` | none | Reporter to use: `junit`, `json`, `html`, or `console` |
| `--reporter-output` | `-O` | none | Output directory for reporter files |

### remove

Delete a collection.

```bash
gurl collection remove [name]
```

Aliases: `rm`, `delete`, `del`

### export

Export a collection with passphrase-encrypted collection secrets.

```bash
gurl collection export [name] --passphrase "$TEAM_SECRET" --output collection.gurl
```

### import

Import a collection export and re-encrypt secrets with the local collection key.

```bash
gurl collection import collection.gurl --passphrase "$TEAM_SECRET"
```

For CI, set `GURL_IMPORT_PASSPHRASE` instead of passing `--passphrase`.

### unlock

Unlock a passphrase-protected file-backed collection after cloning shared project files.

```bash
gurl collection unlock [name] --passphrase "$TEAM_SECRET"
```

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

### Show a collection

```bash
gurl collection show "api-v2"
```

Shows each request in the collection with method and URL.

### Run a collection dry run

```bash
gurl collection run "checkout-flow" --env staging --dry-run
```

Prints request order, planned variable sources, extraction rules, and unresolved placeholders without sending HTTP requests.

### Persist extracted flow variables

```bash
gurl collection run "checkout-flow" --env staging --persist
```

Writes only extracted variables and script-set variables back to the selected environment. `--persist` and `--dry-run` cannot be used together.

### Stop on assertion failures only

```bash
gurl collection run "checkout-flow" --assert-bail
```

Stops after the first assertion failure. Other request failures are reported, but they do not trigger assertion bail mode.

### Delete a collection

```bash
gurl collection remove "old-api"
```

Deletes the "old-api" collection.

### Share a collection with secrets

```bash
gurl collection export "checkout-flow" --passphrase "$TEAM_SECRET" --output checkout-flow.gurl
gurl collection import checkout-flow.gurl --passphrase "$TEAM_SECRET"
```

The export encrypts collection secrets with the passphrase. Import decrypts those values and stores them with the local `.gurl/collections/<collection>/collection.key`.

## See also

- [`gurl save`](save) - Save a request to a collection
- [`gurl list`](list) - List requests in a collection
- [`gurl env`](env) - Manage environments
