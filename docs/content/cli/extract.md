---
title: "gurl extract"
description: "Test extraction rules against a sample response"
---


Test extraction rules against a sample response. No request is sent.

## Synopsis

```bash
gurl extract [flags] VAR=METHOD:EXPR [VAR=METHOD:EXPR...]
```

`gurl extract` reads a response body from `--file` or stdin and prints what each rule extracts. Use it to get a rule right before saving it on a request with `gurl save --extract` or `gurl edit --extract`.

Rules have the form `VAR=METHOD:EXPR`:

| Method | Expression | Example |
|--------|------------|---------|
| `jsonpath` | JSONPath, including array indexes | `orderId=jsonpath:$.data.items[0].id` |
| `jq` | Alias of `jsonpath` | `id=jq:$.id` |
| `header` | Response header name (pass headers with `-H`) | `loc=header:Location` |
| `regex` | Regular expression; the first capture group is the value | `n=regex:"amount":(\d+)` |

JSONPath extraction keeps large numbers exact (`1000000`, not `1e+06`).

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | stdin | Response body file |
| `--header` | `-H` | none | Response header `'Key: Value'` for header rules (repeatable) |

## Examples

### JSONPath from stdin

```bash
echo '{"data":{"id":"ord_1"}}' | gurl extract 'orderId=jsonpath:$.data.id'
```

```text
orderId = ord_1
```

### Several rules against a file

```bash
gurl extract -f resp.json -H 'Location: /orders/9' \
  'loc=header:Location' \
  'first=jsonpath:$.data.items[0].id' \
  'n=regex:"amount":(\d+)'
```

### Save the rule once it works

```bash
gurl edit "create-order" --extract 'orderId=jsonpath:$.data.id'
```

## Related commands

- [`gurl save`]({{< relref "/cli/save" >}}) - Save a request with `--extract` rules
- [`gurl edit`]({{< relref "/cli/edit" >}}) - Add or replace extraction rules
- [`gurl run`]({{< relref "/cli/run" >}}) - Extracted values and `--persist`
