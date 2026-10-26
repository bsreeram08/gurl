---
title: "CLI reference"
description: "Every gurl command, its flags and examples"
aliases: ["/cli/tui/"]
weight: 4
---

One page per command, in alphabetical order. Every page has the same layout: synopsis, flags, aliases, examples and related commands. Flag tables are checked against `gurl docs --json` from the binary; run [`gurl docs`]({{< relref "/cli/docs" >}}) for the reference that matches your installed version.

## Commands

| Command | What it does |
|---------|--------------|
| [`auth`]({{< relref "/cli/auth" >}}) | List auth handlers and their parameters |
| [`codegen`]({{< relref "/cli/codegen" >}}) | Generate Go, Python, JavaScript or curl code from a request |
| [`collection`]({{< relref "/cli/collection" >}}) | Manage collections, their variables, and collection runs |
| [`cookies`]({{< relref "/cli/cookies" >}}) | List, delete and clear stored cookies |
| [`delete`]({{< relref "/cli/delete" >}}) | Remove a saved request |
| [`detect`]({{< relref "/cli/detect" >}}) | Parse a curl command from stdin or a file |
| [`diag`]({{< relref "/cli/diag" >}}) | Inspect one recorded execution |
| [`diff`]({{< relref "/cli/diff" >}}) | Compare the latest responses of a request |
| [`docs`]({{< relref "/cli/docs" >}}) | Print a compact reference of every command and concept |
| [`edit`]({{< relref "/cli/edit" >}}) | Change fields of a saved request |
| [`env`]({{< relref "/cli/env" >}}) | Manage environments and their variables |
| [`exec`]({{< relref "/cli/exec" >}}) | Run a command with an environment's variables injected |
| [`export`]({{< relref "/cli/export" >}}) | Export requests to a file |
| [`extract`]({{< relref "/cli/extract" >}}) | Test extraction rules against a sample response |
| [`graphql`]({{< relref "/cli/graphql" >}}) | Execute a GraphQL query |
| [`grpc`]({{< relref "/cli/grpc" >}}) | Execute a gRPC request |
| [`history`]({{< relref "/cli/history" >}}) | Show execution history for a request |
| [`import`]({{< relref "/cli/import" >}}) | Import from OpenAPI, Insomnia, Bruno, Postman, HAR or gurl exports |
| [`init`]({{< relref "/cli/init" >}}) | Initialize a project (`.gurl/`) |
| [`list`]({{< relref "/cli/list" >}}) | List saved requests |
| [`paste`]({{< relref "/cli/paste" >}}) | Copy a request as curl to the clipboard |
| [`receive`]({{< relref "/cli/receive" >}}) | Decrypt a share bundle and import it |
| [`rename`]({{< relref "/cli/rename" >}}) | Rename a saved request |
| [`run`]({{< relref "/cli/run" >}}) | Execute a saved request |
| [`save`]({{< relref "/cli/save" >}}) | Save a request under a name |
| [`sequence`]({{< relref "/cli/sequence" >}}) | Set execution order within a collection |
| [`share`]({{< relref "/cli/share" >}}) | Encrypt requests to a recipient's public key |
| [`shell`]({{< relref "/cli/shell" >}}) | Launch the typed interactive shell |
| [`show`]({{< relref "/cli/show" >}}) | Show a saved request |
| [`sse`]({{< relref "/cli/sse" >}}) | Connect to a Server-Sent Events endpoint |
| [`timeline`]({{< relref "/cli/timeline" >}}) | Show executions across all requests |
| [`update`]({{< relref "/cli/update" >}}) | Update gurl to the latest release |
| [`ws`]({{< relref "/cli/ws" >}}) | Open a WebSocket connection |

`gurl tui` is a hidden, non-functional stub; use `gurl shell` for interactive use.

## Global flags

These work with every command, in any position.

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | | JSON output, no prompts, strict masking (or `GURL_AGENT=1`, or run the binary as `agentgurl`). See [Agent mode]({{< relref "/ai/agent-mode" >}}). |
| `--no-input` | | Never prompt; fail and name what to pass instead (or `GURL_NO_INPUT=1`) |
| `--project-dir` | | Project root containing `.gurl` file storage (or `GURL_PROJECT_DIR`) |
| `--version` | `-v` | Print the version |

## Exit codes

`0` success, `1` error, `2` usage error; `gurl exec` exits with its command's code. Details and the JSON error shape are in [JSON output]({{< relref "/ai/json-output" >}}).
