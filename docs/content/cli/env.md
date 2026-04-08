---
title: "gurl env"
description: "Manage environment variables"
---

# gurl env

Manage environment variables for use in requests.

## Usage

```bash
gurl env [subcommand] [flags]
```

## Description

The `env` command group manages environments and variables. Environments allow you to define different variable sets for development, staging, production, etc.

## Subcommands

### create

Create a new environment.

```bash
gurl env create [name] [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--var` | `-v` | none | Variable as `KEY=VALUE` |
| `--secret` | `-s` | none | Secret variable (value prompted) |

### list

List all environments.

```bash
gurl env list
```

Aliases: `ls`, `l`

### switch

Switch the active environment.

```bash
gurl env switch [name]
```

Aliases: `use`, `activate`

### delete

Delete an environment.

```bash
gurl env delete [name]
```

Aliases: `rm`, `del`

### show

Display environment variables.

```bash
gurl env show [name]
```

Aliases: `display`, `view`

### set

Set a variable in an environment.

```bash
gurl env set [name] [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--var` | `-v` | none | Variable as `KEY=VALUE` |
| `--secret` | `-s` | none | Secret variable |

### unset

Remove a variable from an environment.

```bash
gurl env unset [name] [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--var` | `-v` | none | Variable name to remove |

### import

Import variables from a file.

```bash
gurl env import [name] [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | none | File to import from |

## Examples

### Create an environment

```bash
gurl env create production
```

Creates a new environment called "production".

### Add variables

```bash
gurl env create staging --var "API_URL=https://staging.example.com" --var "DEBUG=false"
```

Creates staging environment with variables.

### Set a secret

```bash
gurl env set production --secret "API_KEY"
```

Sets a secret variable (prompts for value).

### Switch environment

```bash
gurl env switch production
```

Makes "production" the active environment.

### Import from .env file

```bash
gurl env import production --file .env.production
```

Imports variables from a .env file.

## See also

- [`gurl run`](run) - Execute a request with environment
- [`gurl collection`](collection) - Manage collections
- [`gurl sequence`](sequence) - Manage sequences
