---
title: "gurl init"
description: "Initialize a gurl project"
---


Create a `.gurl/` directory so collections and environments can live in your repository, next to the code that uses them.

## Synopsis

```bash
gurl init [--project-dir <path>]
```

`gurl init` writes `.gurl/` (with `.gurl/.gitignore`, which keeps local `collection.key` files out of git). After that, gurl walks up parent directories to find `.gurl/`; in CI, point at it with `GURL_PROJECT_DIR` or the global `--project-dir` flag.

```text
.gurl/
  collections/
    users-api/
      collection.json
      list-users.json
      collection.key   # local only, gitignored
  environments/
    staging.json
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--project-dir` | | current directory | Project root for `.gurl` file storage |

## Examples

### Start a project in the current repo

```bash
gurl init
```

### Move an existing collection into the project and commit it

```bash
gurl collection migrate "payments-api"
git add .gurl
git commit -m "add payments API collection"
```

### Initialize another directory

```bash
gurl init --project-dir ./services/billing
```

## Related commands

- [`gurl collection`]({{< relref "/cli/collection" >}}) - `collection migrate`, `export`, `unlock`
- [Credential storage]({{< relref "/security/credential-storage" >}}) - How project secrets are encrypted
