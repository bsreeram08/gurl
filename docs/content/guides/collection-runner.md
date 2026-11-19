---
title: "Collections & data-driven runs"
description: "Run a group of requests in order, feed CSV/JSON rows, and produce CI reports."
weight: 4
---

A collection groups saved requests and carries its own variables and secrets. `apigent collection run` executes its requests in order, once per iteration or data row, and reports pass/fail for each.

## Build a collection

```bash
apigent collection create "checkout" --var CURRENCY=EUR --secret API_KEY=sk-test-...
apigent save "create-order" '{{BASE_URL}}/orders' -X POST -c checkout \
  --extract 'orderId=jsonpath:$.id'
apigent save "capture-payment" '{{BASE_URL}}/orders/{{orderId}}/capture' -X POST -c checkout
apigent collection show "checkout"          # requests and variables (secrets masked)
```

Collection variables sit between environment and data-row values in [scope precedence]({{< relref "/guides/variables#where-values-come-from-scopes" >}}), and may reference environment keys (`API_URL={{BASE_URL}}/v1`).

## Set the order

```bash
apigent sequence set "create-order" 1
apigent sequence set "capture-payment" 2
apigent sequence list "checkout"
```

## Preview, then run

```bash
apigent collection run "checkout" --env staging --dry-run
apigent collection run "checkout" --env staging
```

A dry run prints the request order and each resolved request without sending anything, running post-response scripts, recording history, writing reports or persisting variables.

A failed request prints its execution id on stderr, ready for [`apigent diag`]({{< relref "/cli/diag" >}}):

```text
  [FAIL] capture-payment: HTTP 500 (id: 01dda65f) — inspect with: apigent diag 01dda65f
```

`apigent collection run` exits 1 when any request fails.

## Flow variables across requests

Extraction rules and scripts share one map of runtime values. A value extracted by one request is available to later requests' placeholders, assertions, scripts and `run-if` checks:

```bash
apigent edit "create-order" --post-script "apigent.setNextRequest('capture-payment')"
apigent edit "capture-payment" --run-if "orderId != ''"
```

Write them back to their variable origin at the end of the run with `--persist` (not combinable with `--dry-run`):

```bash
apigent collection run "checkout" --env staging --persist
```

Assertions can use those values too, and a request that asserts its status may expect an error:

```bash
apigent save "read-order" '{{BASE_URL}}/orders/{{orderId}}' -c checkout
apigent edit "read-order" -a 'status=200' -a 'body.data.id={{orderId}}'
apigent save "locked-account" '{{BASE_URL}}/billing' -c checkout
apigent edit "locked-account" -a 'status=402' -a 'body.errorCodes[0]=ACCESS_ENDED'
```

Without a status assertion, any non-2xx response fails the request.

See [Chaining & extraction]({{< relref "/guides/chaining" >}}).

## Data-driven runs

Feed rows from CSV or JSON; each row is one iteration, and its columns become data-row variables:

```csv
email,name
alice@example.com,Alice
bob@example.com,Bob
```

```bash
apigent collection run "signup" --data users.csv --env staging
apigent run "create-user" --data users.json      # a single request works too
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
apigent collection run "api-tests" --env ci --reporter junit --reporter json --reporter-output reports --ci
```

```yaml
# GitHub Actions
- name: Run API tests
  run: |
    mkdir -p reports
    apigent collection run "api-tests" --env ci --reporter junit --reporter-output reports --bail
- name: Publish test results
  uses: dorny/test-reporter@v1
  with:
    name: API tests
    path: reports/report.xml
    reporter: java-junit
```

Add `--verbose` to print request and response headers and bodies for each request.

## File-backed reloads

For project-backed collections (`apigent init`, `apigent collection migrate`), a long collection run watches the collection files. The current iteration always uses the snapshot it started with; changes (debounced) apply from the next iteration or data row.

## See also

- [`apigent collection`]({{< relref "/cli/collection" >}}) — every subcommand and flag, including sharing collections through git with passphrase-encrypted secrets
- [`apigent sequence`]({{< relref "/cli/sequence" >}})
