---
title: "Scripting & assertions"
description: "Check responses with --assert, or run JavaScript before and after a request."
weight: 6
aliases: ["/guides/assertions/"]
---

Assertions check a response without code; scripts run JavaScript before a request is sent and after its response arrives. Start with assertions and reach for a script only when you need logic.

## Assertions

Pass one or more `--assert` flags to `run` (all must pass), or save them on a request with `gurl edit --assert`:

```bash
gurl run "get-user" --assert "status=200" --assert "body.$.email contains @"
gurl edit "get-user" --assert "status=200"
```

A failed assertion makes `run` exit 1, records the execution with failure phase `assertion`, and prints the failure notice with the execution id. With `--json`, results are in `"assertions":[{"name","passed","message"}]`.

### Syntax

Comparison operators attach directly to the field; word operators are surrounded by spaces:

```text
status=200            status>=200          time<1000
body contains success body not_contains error
body matches u-\d+    headers.X-Request-Id exists
```

| Operator | Meaning |
|----------|---------|
| `=`, `!=` | Equal, not equal |
| `<`, `>`, `<=`, `>=` | Numeric comparison |
| `contains`, `not_contains` | Substring match |
| `matches` | Regular expression match |
| `exists` | Field is present and non-empty (no value) |

| Field | Value |
|-------|-------|
| `status` (or `status_code`) | HTTP status code |
| `headers.NAME` | Response header, e.g. `headers.Content-Type contains json` |
| `body` | Whole response body as text |
| `body.$.path` | JSONPath into the body, e.g. `body.$.data.items[0].id = 7` (`body.data.id` also works) |
| `extract:NAME` | A value extracted or set by a script, e.g. `extract:orderId != ''` |
| `time` | Response time in milliseconds |
| `size` | Response size in bytes |

An assertion that can't be parsed (`invalid assertion format`) stops the run with exit code 1 before anything is sent.

### Examples

```bash
# Validation error is returned as expected
gurl run "invalid-order" --assert "status=400" --assert "body contains validation"

# Header present and fast enough
gurl run "health" --assert "headers.X-Request-Id exists" --assert "time<500"

# Extraction worked
gurl run "login" --assert "extract:token != ''"
```

## Scripts

Scripts are JavaScript (the goja runtime, ECMAScript 5.1+) saved on the request as inline source:

```bash
gurl save "login" '{{BASE_URL}}/login' -X POST --extract 'token=jsonpath:$.token' \
  --post-script 'gurl.setNextRequest("profile")'

gurl edit "profile" --pre-script 'gurl.request.headers.set("X-Request-Time", String(Date.now()))'
```

- **Pre-request scripts** run before the request is sent: add headers, rewrite the URL or body, or skip the request.
- **Post-response scripts** run after the response arrives: read the body, set variables, add tests, choose the next request.

### Script API

Everything hangs off the global `gurl` object:

| API | Description |
|-----|-------------|
| `gurl.request.headers.get(name)` / `.set(name, value)` / `.remove(name)` | Read or change request headers (pre-request) |
| `gurl.request.url`, `gurl.request.body`, `gurl.request.method` | Read or assign the URL, body and method |
| `gurl.response.status`, `.statusText`, `.time`, `.size`, `.body` | Response fields (post-response) |
| `gurl.response.json()`, `gurl.response.text()` | Parsed JSON body, raw text body |
| `gurl.response.headers.get(name)` | Response header |
| `gurl.setVar(name, value)`, `gurl.getVar(name)` | Set or read a runtime variable |
| `gurl.setNextRequest(name)` | Run this saved request next (with `run --chain` or in a collection run) |
| `gurl.skipRequest()` | Skip the current request |
| `gurl.test(name, fn)`, `gurl.expect(value).to.equal(expected)` | Named checks inside a script |
| `console.log`, `console.warn`, `console.error` | Logging |

Variables set with `gurl.setVar` are runtime values: the highest scope, available to later requests in the same run, and written back only with `--persist` (which refuses values containing `{{`). See [Variables & placeholders]({{< relref "/guides/variables" >}}).

### Example: capture a value and pick the next request

```javascript
var data = gurl.response.json();
if (data.requiresProfile) {
  gurl.setVar("USER_ID", String(data.userId));
  gurl.setNextRequest("get-profile");
}
```

### Example: validate inside a script

```javascript
var data = gurl.response.json();
gurl.test("has users", function () {
  gurl.expect(data.status).to.equal("success");
});
```

### Example: skip a request

```javascript
if (gurl.getVar("SKIP_HEALTH_CHECK") === "true") {
  gurl.skipRequest();
}
```

### Sandbox

Scripts cannot read secret variables unless explicitly allowed, and URL changes are limited to `http` and `https`. See [Security]({{< relref "/security" >}}).

> [!TIP]
> Keep scripts small. If a script grows past a few lines, the logic probably belongs in your application or test code; use [`gurl exec`]({{< relref "/cli/exec" >}}) to hand it an environment instead.

## See also

- [Chaining & extraction]({{< relref "/guides/chaining" >}})
- [`gurl run`]({{< relref "/cli/run" >}}), [`gurl edit`]({{< relref "/cli/edit" >}})
