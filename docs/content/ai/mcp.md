---
title: "MCP server"
description: "Serve saved requests as MCP tools. Agents call them by name and never see a credential."
weight: 4
---

`apigent mcp` starts a [Model Context Protocol](https://modelcontextprotocol.io) server on stdin/stdout. Every saved request becomes a tool. Calls run in [agent mode]({{< relref "/ai/agent-mode" >}}) and return the same masked JSON as `run --json`.

## Set up

Store the secrets yourself, once:

```bash
apigent env create dev --var BASE_URL=https://api.example.com --var API_KEY=- --secret API_KEY
apigent save get-user '{{BASE_URL}}/users/{{id}}' -H 'Authorization: Bearer {{API_KEY}}'
```

Then add the server to your MCP client's configuration:

```json
{
  "mcpServers": {
    "my-api": { "command": "apigent", "args": ["mcp", "--env", "dev"] }
  }
}
```

| Flag | Meaning |
|------|---------|
| `--env`, `-e` | The environment every call runs against. Fixed for the server's lifetime. Default: the active environment. |
| `--collection`, `-c` | Expose only the requests in this collection. |

## What the agent sees

The agent gets one tool per saved request. Tool names use only letters, digits, `_` and `-`, so `get user` becomes `get_user`. The tool's arguments are the request's **fillable placeholders**: the placeholders that neither the environment nor the collection defines. For `get-user` above, that is just `id`.

```json
{"name": "get-user", "arguments": {"id": "42"}}
```

The result is the [`run --json` object]({{< relref "/ai/json-output" >}}), with `isError` set when `ok` is false. Secrets are masked everywhere in it, including response bodies that echo them back:

```json
{"ok":true,"status":200,"url":"https://api.example.com/users/42","body":{"id":"42","auth":"Bearer *****"}}
```

## Why an agent can't redirect your secrets

If a tool call could set any variable, an agent could pass `BASE_URL=https://attacker.example`. The request would then go out with your `Authorization` header to a host the agent controls. apigent refuses any argument that the environment or collection already defines, and nothing is sent:

```text
argument "BASE_URL" is not allowed: only placeholders the environment leaves undefined can be set (allowed: id)
```

So define every host in the environment or collection, never leave it as a placeholder for the agent to fill.

## Under the hood

Each call runs `apigent --agent run <request> --env <env> --var ...` in a child process. It gets exactly the CLI's guarantees: one resolution path, masked output, an execution record you can inspect with `apigent diag <id>`, and no way to reveal a secret. Calls are handled one at a time.
