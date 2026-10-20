---
title: "Collections & data-driven runs"
description: "Run a group of requests in order, feed CSV/JSON rows, and produce CI reports."
weight: 4
---

A collection groups saved requests and carries its own variables and secrets. `gurl collection run` executes its requests in order, once per iteration or data row, and reports pass/fail for each.

## Build a collection

```bash
gurl collection create "checkout" --var CURRENCY=EUR --secret API_KEY=sk-test-...
gurl save "create-order" '{{BASE_URL}}/orders' -X POST -c checkout \
  --extract 'orderId=jsonpath:$.id'
gurl save "capture-payment" '{{BASE_URL}}/orders/{{orderId}}/capture' -X POST -c checkout
gurl collection show "checkout"          # requests and variables (secrets masked)
```

Collection variables sit between environment and data-row values in [scope precedence]({{< relref "/guides/variables#where-values-come-from-scopes" >}}), and may reference environment keys (`API_URL={{BASE_URL}}/v1`).

## Set the order

```bash
gurl sequence set "create-order" 1
gurl sequence set "capture-payment" 2
gurl sequence list "checkout"
```

## Preview, then run

```bash
gurl collection run "checkout" --env staging --dry-run
gurl collection run "checkout" --env staging
```

A dry run prints the request order and each resolved request without sending anything, running post-response scripts, recording history, writing reports or persisting variables.

A failed request prints its execution id on stderr, ready for [`gurl diag`]({{< relref "/cli/diag" >}}):

```text
  [FAIL] capture-payment: HTTP 500 (id: 01dda65f) — inspect with: gurl diag 01dda65f
```

`gurl collection run` exits 1 when any request fails.

## Flow variables across requests

Extraction rules and scripts share one map of runtime values. A value extracted by one request is available to later requests' placeholders, assertions, scripts and `run-if` checks:

```bash
gurl edit "create-order" --post-script "gurl.setNextRequest('capture-payment')"
gurl edit "capture-payment" --run-if "orderId != ''"
```

Write them back to their variable origin at the end of the run with `--persist` (not combinable with `--dry-run`):

```bash
gurl collection run "checkout" --env staging --persist
```

See [Chaining & extraction]({{< relref "/guides/chaining" >}}).

## Data-driven runs

Feed rows from CSV or JSON; each row is one iteration, and its columns become data-row variables:

```csv
email,name
alice@example.com,Alice
bob@example.com,Bob
```

```bash
gurl collection run "signup" --data users.csv --env staging
gurl run "create-user" --data users.json      # a single request works too
```

JSON data is an array of objects: `[{"email": "alice@example.com", "name": "Alice"}]`.

Repeat without data using `--iterations 3`, and pace requests with `--delay 200ms`.

## Stop early

| Flag | Effect |
|------|--------|
| `--bail` | Stop on the first failure of any kind; remaining requests are skipped |
| `--assert-bail` | Stop on the first failed assertion only |
| `--ci` | CI mode: treat skipped requests as failures (exit 1) |

## Reports for CI

Reporters are `console`, `junit`, `json` and `html`; repeat `--reporter` for several. Files go to `--reporter-output DIR` (as `report.xml`, `report.json`, …). Create the directory first:

```bash
mkdir -p reports
gurl collection run "api-tests" --env ci --reporter junit --reporter json --reporter-output reports --ci
```

```yaml
# GitHub Actions
- name: Run API tests
  run: |
    mkdir -p reports
    gurl collection run "api-tests" --env ci --reporter junit --reporter-output reports --bail
- name: Publish test results
  uses: dorny/test-reporter@v1
  with:
    name: API tests
    path: reports/report.xml
    reporter: java-junit
```

Add `--verbose` to print request and response headers and bodies for each request.

## File-backed reloads

For project-backed collections (`gurl init`, `gurl collection migrate`), a long collection run watches the collection files. The current iteration always uses the snapshot it started with; changes (debounced) apply from the next iteration or data row.

## See also

- [`gurl collection`]({{< relref "/cli/collection" >}}) — every subcommand and flag, including sharing collections through git with passphrase-encrypted secrets
- [`gurl sequence`]({{< relref "/cli/sequence" >}})
