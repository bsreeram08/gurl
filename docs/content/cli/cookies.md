---
title: "valetkey cookies"
description: "Manage stored cookies"
---


List, delete and clear the cookies valetkey has stored from responses.

## Synopsis

```bash
valetkey cookies <subcommand> [arguments]
```

## Subcommands

### list

List all stored cookies (domain, name, value; long values are truncated).

```bash
valetkey cookies list
```

None.

Aliases: `ls`, `l`

### clear

Delete every stored cookie.

```bash
valetkey cookies clear
```

None.

Aliases: `cl`, `clean`

### delete

Delete one cookie by domain and name.

```bash
valetkey cookies delete <domain> <name>
```

None.

Aliases: `rm`, `del`

## Aliases

- `cookie`
- `ck`

## Examples

### See what is stored

```bash
valetkey cookies list
```

### Drop a session cookie

```bash
valetkey cookies delete api.example.com sid
```

### Start fresh

```bash
valetkey cookies clear
```

## Related commands

- [`valetkey run`]({{< relref "/cli/run" >}}) - Execute a saved request
- [Header redaction]({{< relref "/security/header-redaction" >}}) - `Cookie` and `Set-Cookie` masking
