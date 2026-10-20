---
title: "Variables & placeholders"
description: "How {{placeholders}} are filled: scopes, typed and dynamic placeholders, aliases, failures, and --persist."
weight: 2
---

A placeholder is a `{{name}}` marker in a saved request's URL, headers, body or auth params. Request resolution replaces every placeholder with a value at send time, or stops the request and lists every problem. Nothing with an unresolved placeholder is ever sent.

## Where values come from: scopes

Each value comes from one scope. When the same name is set in several scopes, the higher one wins:

| Scope (lowest first) | Set with |
|----------------------|----------|
| 1. Environment | `--env NAME` or `gurl env switch NAME` |
| 2. Collection | `gurl collection create/set-var` (the request's collection) |
| 3. Data row | `--data rows.csv` / `rows.json` |
| 4. CLI | `--var key=value` |
| 5. Runtime | values extracted from a response or set by a script earlier in the same run |

The scope a variable's final value came from is its **variable origin**. [`gurl diag`]({{< relref "/cli/diag" >}}) shows the origin of each variable in a recorded execution, and `--persist` uses it to decide where to write values back.

Collection values may reference environment keys, so one collection can serve several environments:

```bash
gurl env create staging --var BASE_URL=https://staging.example.com
gurl collection create "users-api" --var 'API_URL={{BASE_URL}}/v1'
```

## Placeholder forms

| Form | Meaning | Example |
|------|---------|---------|
| `{{name}}` | Plain value, inserted as text | `{{BASE_URL}}/users/{{id}}` |
| `{{name:num}}` | Must be a number | `{"amount": {{amount:num}}}` |
| `{{name:bool}}` | Must be `true` or `false` | `{"live": {{live:bool}}}` |
| `{{name:raw}}` | Inserted as-is (e.g. a JSON fragment) | `{"meta": {{meta:raw}}}` |
| `{{$fn}}` | Dynamic function, computed per send | `{{$uuid}}` |

### Typed placeholders keep JSON bodies valid

A bare `{{amount:num}}` makes the stored body invalid JSON. Quote it instead: in a JSON body, the quoted form `"{{amount:num}}"` (likewise `"{{x:bool}}"` and `"{{x:raw}}"`) drops the quotes when it is filled.

```bash
gurl save "charge" '{{BASE_URL}}/charges' -X POST \
  -H 'Content-Type: application/json' \
  -d '{"amount":"{{amount:num}}","capture":"{{capture:bool}}"}'

gurl run "charge" --env dev --var amount=5 --var capture=true --dry-run
```

```text
POST https://dev.api.example.com/charges
Content-Type: application/json

{"amount":5,"capture":true}
```

A value of the wrong type is a resolution problem (`variable "amount" must be a number`); the value itself is never echoed in the error.

### Dynamic functions

| Function | Value |
|----------|-------|
| `{{$uuid}}` | Random UUID |
| `{{$timestamp}}` | Unix timestamp (seconds) |
| `{{$isoTimestamp}}` | ISO 8601 timestamp |
| `{{$randomInt(1,100)}}` | Random integer in the range |
| `{{$randomString(8)}}` | Random string of the given length |
| `{{$randomEmail}}` | Random email address |

Dynamic functions work in `run`, `collection run`, the shell and auth params. A dry run leaves them in place. An unknown function (`{{$nope}}`) is a resolution problem.

### Environment aliases

An environment value can name another key of the same environment: `API_KEY={{SA_KEY}}`. See [Environments & secrets]({{< relref "/guides/environments#alias-keys-across-environments" >}}).

## When resolution fails

Every problem is collected and reported at once, and the request is not sent:

```text
Error: variable substitution failed: missing required variables: BASE, API_KEY
```

| Problem kind | Cause |
|--------------|-------|
| `missing` | No scope has a value for `{{name}}` |
| `leftover_placeholder` | A value still contains a placeholder, e.g. `--var 'API_KEY={{other}}'`. Values are filled in one pass and never re-expanded. |
| `bad_type` | A typed placeholder got the wrong kind of value |
| `unknown_dynamic` | `{{$fn}}` is not a known function |
| `auth` | The auth config cannot be resolved |

With `run --json`, these appear as structured `problems` entries (`kind`, `variable`, `location`, `message`). Use `--dry-run` to check a request without sending it; it exits 1 when there are problems.

## Keeping runtime values: --persist

Values extracted from a response (`--extract`) or set by a script (`gurl.setVar`) live only for the current run. Without `--persist`, `run` drops them and prints a note such as `note: uid not saved; add --persist to use them in later runs`.

With `--persist`, each extracted or script-set value is written back to its variable origin:

- a variable that came from the environment goes to that environment (`--env` or the active one);
- otherwise it goes to the request's collection, or to the environment if the request has no collection.

`--persist` refuses:

- a value that contains `{{` (`refusing to persist "tpl": value contains "{{"`), so placeholders never get stored as values;
- a locked (passphrase-protected, not unlocked) collection, instead of overwriting it;
- `--dry-run` (exit code 2).

CLI `--var` values and data-row values are never written back unless an extraction or a script changes them.

## See also

- [Chaining & extraction]({{< relref "/guides/chaining" >}})
- [`gurl run`]({{< relref "/cli/run" >}}) and [`gurl docs`]({{< relref "/cli/docs" >}}) (its "Placeholders" and "Scope precedence" sections are the short version of this page)
