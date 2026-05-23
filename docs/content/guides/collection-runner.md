---
title: "Collection Runner"
description: "Run entire API test suites with data-driven testing"
weight: 4
---

# Collection Runner

The collection runner executes multiple requests in sequence, supporting data-driven testing and various output formats for CI integration.

## Running a Collection

Run all requests in a collection:

```bash
gurl collection run "my-api"
```

Run a specific request from a collection:

```bash
gurl collection run "my-api" --request "get-users"
```

## Sequencing Requests

By default, requests run in the order they were saved. Control execution order explicitly:

```bash
gurl sequence set "login" 1
gurl sequence set "get-users" 2
gurl sequence set "update-user" 3
gurl sequence set "logout" 4
```

View the current sequence:

```bash
gurl sequence list "my-api"
```

### Conditional Execution

Skip requests based on conditions:

```bash
gurl collection run "my-api" --skip "update-user" --skip-if "SKIP_WRITES=true"
```

Saved requests can also carry a `run-if` expression:

```bash
gurl edit "update-user" --run-if "SKIP_WRITES != true"
```

`run-if` supports simple `VAR == VALUE` and `VAR != VALUE` checks. Missing variables are treated as empty strings.

## Request Chaining and Flow Variables

Post-response scripts can route the next request in a collection:

```javascript
var data = response.json();
gurl.setVariable("orderId", data.id);
gurl.setNextRequest("capture-payment");
```

Extraction rules and scripts share the same flow variable map. A value extracted by one request can be used by later request templates, assertions, scripts, and `run-if` checks.

```bash
gurl save "create-order" https://api.example.com/orders \
  --extract orderId=jsonpath:$.id \
  --post-script "gurl.setNextRequest('capture-payment')"

gurl edit "capture-payment" --run-if "orderId != ''"
```

Persist only the extracted or script-set values back to the selected environment:

```bash
gurl collection run "checkout-flow" --env staging --persist
```

CLI variables, environment inputs, and data-row values are not written back unless extraction or a script changes the same key.

## File-Backed Reloads

When a project uses file-backed collections, long-running collection runs watch the collection files while the run is active. Changes are debounced before reload so partially written files are not read.

The active iteration always uses a stable snapshot. If request files, collection variables, or sequence order change during an iteration, the current iteration continues with the snapshot it started with. Future iterations, including future data rows in a data-driven run, reload the collection after the debounced change event. Added, removed, renamed, or reordered requests affect only those later iterations.

## Dry Run

Preview a collection flow without sending requests:

```bash
gurl collection run "checkout-flow" --env staging --dry-run
```

Dry runs print request order, planned extraction sources, variable sources from earlier steps, and unresolved placeholders. They do not send HTTP requests, run post-response scripts, save history, write reports, or persist variables.

## Data-Driven Testing

Run a request with data from a CSV or JSON file. Each row iteration substitutes variables and runs the request.

### CSV Format

Create a `users.csv` file:

```csv
email,name,role
alice@example.com,Alice,admin
bob@example.com,Bob,viewer
charlie@example.com,Charlie,editor
```

Run with data:

```bash
gurl run "create-user" --data users.csv
```

This runs `create-user` three times, once for each row.

### JSON Dataset

Create a `users.json` file:

```json
[
  {"email": "alice@example.com", "name": "Alice", "role": "admin"},
  {"email": "bob@example.com", "name": "Bob", "role": "viewer"}
]
```

Run with data:

```bash
gurl run "create-user" --data users.json
```

## Reporters

Gurl supports multiple reporter formats for different use cases.

### Console Reporter (Default)

Human-readable output with color coding:

```
PASS  get-users       200 OK  45ms
PASS  create-user     201 Created  120ms
FAIL  update-user     400 Bad Request  32ms
```

### JUnit XML Reporter

For CI integration with GitHub Actions, Jenkins, etc.:

```bash
gurl collection run "my-api" --reporter junit --output test-results.xml
```

Generates JUnit-compatible XML:

```xml
<testsuite name="my-api" tests="3" failures="1">
  <testcase name="get-users" classname="my-api" time="0.045"/>
  <testcase name="create-user" classname="my-api" time="0.120"/>
  <testcase name="update-user" classname="my-api" time="0.032">
    <failure message="expected 200, got 400"/>
  </testcase>
</testsuite>
```

### JSON Reporter

Machine-readable output for custom tooling:

```bash
gurl collection run "my-api" --reporter json --output results.json
```

### HTML Reporter

Visual HTML report with timing charts:

```bash
gurl collection run "my-api" --reporter html --output report.html
```

## CI Integration

### GitHub Actions

```yaml
- name: Run API Tests
  run: |
    gurl collection run "api-tests" --reporter junit --output results.xml --bail
  env:
    CI: true

- name: Publish Test Results
  uses: dorny/test-reporter@v1
  with:
    name: API Tests
    path: results.xml
    reporter: java-junit
```

### Bail Mode

Stop on first failure:

```bash
gurl collection run "my-api" --bail
```

This is useful in CI to fail fast and avoid unnecessary API calls.

### Assertion Bail Mode

Stop only when an assertion fails:

```bash
gurl collection run "my-api" --assert-bail
```

This lets the run continue through transport or HTTP failures while still stopping on the first failed assertion.

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All requests passed |
| 1 | One or more assertions failed |
| 2 | Error (network, parsing, etc.) |

## Report Generation

Generate a report after a collection run:

```bash
gurl collection run "my-api" --reporter html --output report.html
open report.html
```

> [!TIP]
> Use environment variables to switch between staging and production endpoints in CI: `gurl env switch $TARGET_ENV && gurl collection run "api-tests"`.
