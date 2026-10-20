---
title: "gurl cookies"
description: "Manage stored cookies"
---


List, delete and clear the cookies gurl has stored from responses.

## Synopsis

```bash
gurl cookies <subcommand> [arguments]
```

## Subcommands

### list

List all stored cookies (domain, name, value; long values are truncated).

```bash
gurl cookies list
```

None.

Aliases: `ls`, `l`

### clear

Delete every stored cookie.

```bash
gurl cookies clear
```

None.

Aliases: `cl`, `clean`

### delete

Delete one cookie by domain and name.

```bash
gurl cookies delete <domain> <name>
```

None.

Aliases: `rm`, `del`

## Aliases

- `cookie`
- `ck`

## Examples

### See what is stored

```bash
gurl cookies list
```

### Drop a session cookie

```bash
gurl cookies delete api.example.com sid
```

### Start fresh

```bash
gurl cookies clear
```

## Related commands

- [`gurl run`]({{< relref "/cli/run" >}}) - Execute a saved request
- [Header redaction]({{< relref "/security/header-redaction" >}}) - `Cookie` and `Set-Cookie` masking
