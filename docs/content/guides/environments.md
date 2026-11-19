---
title: "Environments & secrets"
description: "Keep hosts and credentials out of saved requests and switch between dev, staging and prod."
weight: 1
---

An environment is a named set of variables, such as `BASE_URL` and `API_KEY`, that fills the `{{placeholders}}` in saved requests at send time. Keep one environment per stage and the same saved requests work against all of them.

## Create an environment

```bash
apigent env create dev --var BASE_URL=https://dev.api.example.com --var TIMEOUT=30
apigent env create prod --var BASE_URL=https://api.example.com
```

## Store a secret without typing it on the command line

Mark a variable secret with `--secret KEY`. Pass the value as `KEY=-` to read it from stdin; on a terminal apigent prompts without echo, so the value never lands in shell history:

```bash
apigent env create prod --var BASE_URL=https://api.example.com --var API_KEY=- --secret API_KEY
# or later:
apigent env set prod API_KEY=- --secret API_KEY
printf '%s' "$PROD_KEY" | apigent env set prod API_KEY=- --secret API_KEY
```

`apigent env set prod KEY -` (a lone `-`) is rejected: use `KEY=-`.

Secret values are encrypted at rest with AES-256-GCM (see [Credential storage]({{< relref "/security/credential-storage" >}})) and shown as `*****`:

```bash
apigent env show prod            # secrets masked
apigent env show prod --json     # {"name","id","variables":{...},"secrets":[...]}, masked
apigent env show prod --reveal   # plaintext; refused in agent mode
apigent env list --json          # [{"name","active","variables"}]
```

{{< callout type="warning" >}}
`--reveal` prints secrets in plaintext to your terminal and its scrollback. Don't use it while screen-sharing, in CI logs, or in a session an AI agent can read. It is refused in [agent mode]({{< relref "/ai/agent-mode" >}}).
{{< /callout >}}

## Use an environment

Reference variables as `{{NAME}}` in the URL, headers, body or auth params of a saved request:

```bash
apigent save "list-charges" '{{BASE_URL}}/v1/charges' -H 'Authorization: Bearer {{API_KEY}}'

apigent run "list-charges" --env prod     # one run
apigent env switch prod                   # make prod the active environment
apigent run "list-charges"
```

`--env` with an unknown name is an error; apigent never silently runs without variables.

## Alias keys across environments

Two environments may name the same credential differently. An alias is an environment value that refers to another key of the same environment:

```bash
apigent env set nets-beta 'API_KEY={{SA_KEY}}'
```

Requests that use `{{API_KEY}}` now get the value of `SA_KEY` in `nets-beta`. Aliases resolve one level only (no re-expansion). When a request uses the alias, the resolved value is masked like the secret it points to.

## Hand an environment to another tool

[`apigent exec`]({{< relref "/cli/exec" >}}) runs a command with the environment's variables in its process environment, without printing them:

```bash
apigent exec --env prod -- node e2e.mjs
apigent exec --env prod -- sh -c 'curl -H "Authorization: Bearer $API_KEY" "$BASE_URL/v1/charges"'
```

## Import from a .env file

```bash
apigent env import prod --file .env.production
```

## Environment commands

| Command | Description |
|---------|-------------|
| `apigent env create <name>` | Create an environment (`--var KEY=VALUE`, `--var KEY=-`, `--secret KEY`) |
| `apigent env list` | List environments (`--json`) |
| `apigent env show <name>` | Show variables, secrets masked (`--json`, `--reveal`) |
| `apigent env set <name> KEY=VALUE` | Set variables (`KEY=-` reads stdin) |
| `apigent env unset <name> --var KEY` | Remove a variable |
| `apigent env switch <name>` | Make an environment active |
| `apigent env import <name> --file .env` | Import a `.env` file |
| `apigent env delete <name>` | Delete an environment |

Full flags: [`apigent env`]({{< relref "/cli/env" >}}). Next: [Variables & placeholders]({{< relref "/guides/variables" >}}) explains how environment values combine with collection, data-row and `--var` values.
