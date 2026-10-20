---
title: "Chaining & extraction"
description: "Pull a token or id out of one response and use it in the next request."
weight: 3
---

Extraction rules capture values from a response; chaining runs the next request with those values. Together they turn "log in, then call the API with the token" into one command.

## 1. Write and test an extraction rule

A rule has the form `VAR=METHOD:EXPR`:

| Method | Captures | Example |
|--------|----------|---------|
| `jsonpath` | A JSONPath value (array indexes supported) | `orderId=jsonpath:$.data.items[0].id` |
| `jq` | Alias of `jsonpath` | `token=jq:$.token` |
| `header` | A response header | `location=header:Location` |
| `regex` | The first capture group | `n=regex:"amount":(\d+)` |

Test it against a sample response first. No request is sent:

```bash
echo '{"token":"t-123","user":{"id":42}}' | gurl extract 'token=jsonpath:$.token' 'uid=jsonpath:$.user.id'
```

```text
token = t-123
uid = 42
```

See [`gurl extract`]({{< relref "/cli/extract" >}}).

## 2. Save the rule on the request

```bash
gurl save "login" '{{BASE_URL}}/auth/login' -X POST \
  -H 'Content-Type: application/json' \
  -d '{"user":"{{USER}}","password":"{{PASSWORD}}"}' \
  --extract 'token=jsonpath:$.token'

# or add it to an existing request
gurl edit "login" --extract 'token=jsonpath:$.token'
```

Run it with `--json` to see what was extracted:

```bash
gurl run "login" --env dev --json | jq .extracted
```

## 3. Chain to the next request

A post-response script picks the next request with `gurl.setNextRequest`. Run the first request with `--chain`:

```bash
gurl edit "login" --post-script 'gurl.setNextRequest("profile")'
gurl save "profile" '{{BASE_URL}}/me' -H 'Authorization: Bearer {{token}}'

gurl run "login" --env dev --chain
```

Extracted and script-set values are runtime values: the highest scope, available to every later request, assertion, script and `run-if` check in the same run. With `--json`, each step prints one line (NDJSON).

Skip a step conditionally with a saved `run-if` expression (`VAR == VALUE` or `VAR != VALUE`; missing variables count as empty):

```bash
gurl edit "profile" --run-if "token != ''"
```

## 4. Keep values for later runs (optional)

Runtime values are dropped at the end of the run, and `run` prints a note naming them. Add `--persist` to write them back to their variable origin:

```bash
gurl run "login" --env dev --chain --persist
```

`--persist` refuses values that contain `{{` and locked collections. Details: [Variables & placeholders]({{< relref "/guides/variables#keeping-runtime-values---persist" >}}).

## Assert on extracted values

```bash
gurl run "login" --env dev --assert "extract:token != ''"
```

## See also

- [Collections & data-driven runs]({{< relref "/guides/collection-runner" >}}) — chains across a whole collection
- [Scripting & assertions]({{< relref "/guides/scripting" >}}) — the script API
