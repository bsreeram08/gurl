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
