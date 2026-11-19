---
title: "valetkey init"
description: "Initialize a valetkey project"
---


Create a `.valetkey/` directory so collections and environments can live in your repository, next to the code that uses them.

## Synopsis

```bash
valetkey init [--project-dir <path>]
```

`valetkey init` writes `.valetkey/` (with `.valetkey/.gitignore`, which keeps local `collection.key` files out of git). After that, valetkey walks up parent directories to find `.valetkey/`; in CI, point at it with `VALETKEY_PROJECT_DIR` or the global `--project-dir` flag.

```text
.valetkey/
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
| `--project-dir` | | current directory | Project root for `.valetkey` file storage |

## Examples

### Start a project in the current repo

```bash
valetkey init
```

### Move an existing collection into the project and commit it

```bash
valetkey collection migrate "payments-api"
git add .valetkey
git commit -m "add payments API collection"
```

### Initialize another directory

```bash
valetkey init --project-dir ./services/billing
```

## Related commands

- [`valetkey collection`]({{< relref "/cli/collection" >}}) - `collection migrate`, `export`, `unlock`
- [Credential storage]({{< relref "/security/credential-storage" >}}) - How project secrets are encrypted
