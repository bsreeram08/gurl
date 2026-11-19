---
title: "apigent cookies"
description: "Manage stored cookies"
---


List, delete and clear the cookies apigent has stored from responses.

## Synopsis

```bash
apigent cookies <subcommand> [arguments]
```

## Subcommands

### list

List all stored cookies (domain, name, value; long values are truncated).

```bash
apigent cookies list
```

None.

Aliases: `ls`, `l`

### clear

Delete every stored cookie.

```bash
apigent cookies clear
```

None.

Aliases: `cl`, `clean`

### delete

Delete one cookie by domain and name.

```bash
apigent cookies delete <domain> <name>
```

None.

Aliases: `rm`, `del`

## Aliases

- `cookie`
- `ck`

## Examples

### See what is stored

```bash
apigent cookies list
```

### Drop a session cookie

```bash
apigent cookies delete api.example.com sid
```

### Start fresh

```bash
apigent cookies clear
```

## Related commands

- [`apigent run`]({{< relref "/cli/run" >}}) - Execute a saved request
- [Header redaction]({{< relref "/security/header-redaction" >}}) - `Cookie` and `Set-Cookie` masking
