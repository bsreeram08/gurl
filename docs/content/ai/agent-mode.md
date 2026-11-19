---
title: "Agent mode"
description: "One switch for AI agents: JSON output, no prompts, strict secret masking."
weight: 2
---

Agent mode makes apigent safe and predictable for an AI agent: every command that can print JSON does, nothing ever prompts, and secrets are masked everywhere, including response bodies.

## Turn it on

Use any one of these:

```bash
apigent --agent run "get-user" --env staging      # flag, in any position
APIGENT_AGENT=1 apigent run "get-user"               # environment variable
ln -s "$(command -v apigent)" ~/.local/bin/agentapigent
agentapigent run "get-user"                       # binary named agentapigent
```

The symlink is the most robust choice: give the agent `agentapigent` and it cannot forget the flag.

## Step 1: read `apigent docs`

The first command an agent should run:

```bash
agentapigent docs            # JSON: {"commands":[...],"sections":{...}}
apigent docs                 # the same reference as compact text (~25KB)
apigent docs run             # one command
```

It lists every command and subcommand with arguments and flags (type, default, aliases), generated from the binary itself, plus short sections on placeholders, scope precedence, extract rules, `--persist`, agent mode, the `run --json` shape, exit codes, `exec` and `diag`. See [`apigent docs`]({{< relref "/cli/docs" >}}).

## What agent mode changes

| Behaviour | In agent mode |
|-----------|---------------|
| Output | `--json` is on for every command that has it: `run`, `run --dry-run`, `diag`, `list`, `show`, `env list`/`show`, `history`, `timeline`, `diff`, `collection list`/`show`, `docs` |
| Prompts | `--no-input` is on: commands never prompt; they fail and name the flag or argument to pass |
| Errors | Every error is printed on stdout as `{"ok":false,"error":"..."}` (and on stderr), usage errors included; help text is not printed on a usage error |
| Masking | Secrets are masked in response bodies, response headers and errors too (`run`, `diag`, `diff`), and in the output of `exec` child processes |
| Refused (exit 2) | `--reveal`, `paste`, `codegen --clipboard`, `run --output`; `apigent shell` refuses to start |

## No prompts: `--no-input`

`--no-input` (or `APIGENT_NO_INPUT=1`) is implied by `--agent` and by `--json`. A command that would ask fails and says what to pass instead:

```json
{"ok":false,"error":"request name is required: apigent run <name> (list names with: apigent list --json)"}
```

Other examples: saving into a collection that doesn't exist, `collection delete` without `--force`, a missing passphrase (`pass --passphrase or set APIGENT_IMPORT_PASSPHRASE`), `env set KEY=-` on a terminal.

## Running requests: `run --json`

stdout carries one JSON object per execution and nothing else (NDJSON for `--chain` steps and `--data` rows):

```json
{"id":"e7770aa1-94f2-4b4a-9e3e-8605e5c1a0cc","short_id":"e7770aa1","request":"get-user","environment":"dev",
 "ok":true,"method":"GET","url":"http://127.0.0.1:18731/users?api_key=*****","status":200,"status_text":"OK",
 "duration_ms":4,"size_bytes":89,"response_headers":{"Content-Type":"application/json","Set-Cookie":"*****"},
 "body":{"id":"u-123","auth":"Bearer *****"},"extracted":{"uid":"u-123"}}
```

(Wrapped for reading; real output is one line.) The agent checks `ok`, reads `body`, and on failure reads `error`, `failure_phase` and `problems`, then calls `apigent diag <short_id>`. The full field list, problem kinds and failure phases are in [JSON output]({{< relref "/ai/json-output" >}}).

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success. A non-2xx response is still 0 unless `--fail` is passed (`ok` is false in the JSON). |
| 1 | Error: request or environment not found, resolution problems, transport error, script or extraction error, failed assertion, non-2xx with `--fail`, dry run with problems |
| 2 | Usage: missing argument with `--no-input`, conflicting flags (`--persist --dry-run`), `--reveal` / `--output` / `paste` / `codegen --clipboard` in agent mode, `shell` with `--no-input` |
| other | `apigent exec` exits with the child command's code |

## Strict masking

Outside agent mode, apigent already masks secrets in everything it shows or stores about a request: saved requests, dry runs, execution records, and the URL, headers and errors of `run --json`. Response bodies are shown as received, because they are your API's data.

Agent mode also masks the body, response headers and errors, and `exec` output. "Secret" here means:

- variables marked secret in an environment or collection, including values reached through an alias;
- variables whose names look sensitive (`*key*`, `*token*`, `*secret*`, `*password*`, `*signature*`, `Authorization`, `Cookie`, …);
- literal credentials written into the saved request: sensitive headers, sensitive query parameters (`?api_key=…`), auth params;
- the URL-escaped and JSON-escaped spellings of all of the above.

Values shorter than 4 characters are not replaced inside free text. Details: [Secret masking]({{< relref "/security/header-redaction" >}}).

## A recommended agent loop

1. `agentapigent docs` — learn the commands.
2. `agentapigent list`, `agentapigent env list`, `agentapigent collection list` — discover what exists.
3. `agentapigent show <request>` or `agentapigent collection show <collection>` — inspect before running.
4. `agentapigent run <request> --env <env> --dry-run` — check that every placeholder resolves.
5. `agentapigent run <request> --env <env>` — run the smallest request that answers the task.
6. On failure: `agentapigent diag <short_id>`.
7. Use `--persist` only when the task explicitly needs extracted values saved.

{{< callout type="warning" >}}
Keep secrets out of commands: an agent should never pass a token with `--var` (the token is then in its context anyway). Have a human store it once in an environment or collection: `apigent env set prod API_KEY=- --secret API_KEY`.
{{< /callout >}}

## See also

- [JSON output]({{< relref "/ai/json-output" >}})
- [Memory safety for AI agents]({{< relref "/ai/memory-safety" >}})
- [`apigent exec`]({{< relref "/cli/exec" >}}) — hand an environment to a script without printing it
