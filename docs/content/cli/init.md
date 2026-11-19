---
title: "apigent init"
description: "Initialize a apigent project"
---


Create a `.apigent/` directory so collections and environments can live in your repository, next to the code that uses them.

## Synopsis

```bash
apigent init [--project-dir <path>]
```

`apigent init` writes `.apigent/` (with `.apigent/.gitignore`, which keeps local `collection.key` files out of git). After that, apigent walks up parent directories to find `.apigent/`; in CI, point at it with `APIGENT_PROJECT_DIR` or the global `--project-dir` flag.

```text
.apigent/
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
| `--project-dir` | | current directory | Project root for `.apigent` file storage |

## Examples

### Start a project in the current repo

```bash
apigent init
```

### Move an existing collection into the project and commit it

```bash
apigent collection migrate "payments-api"
git add .apigent
git commit -m "add payments API collection"
```

### Initialize another directory

```bash
apigent init --project-dir ./services/billing
```

## Related commands

- [`apigent collection`]({{< relref "/cli/collection" >}}) - `collection migrate`, `export`, `unlock`
- [Credential storage]({{< relref "/security/credential-storage" >}}) - How project secrets are encrypted
