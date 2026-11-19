---
title: "Quickstart"
description: "Save a request, run it, move its values into an environment, and group it in a collection."
weight: 2
aliases: ["/docs/quick-start/"]
---

Five minutes, five steps: save → run → environment → collection → inspect. Every command below works against the public `jsonplaceholder.typicode.com` API.

{{% steps %}}

### Save a request

```bash
apigent save "list-users" https://jsonplaceholder.typicode.com/users
```

A saved request has a name, a method, a URL, and optional headers, body, auth and extraction rules. You can also paste a curl command:

```bash
apigent save "create-user" --curl "curl -X POST -H 'Content-Type: application/json' -d '{\"name\":\"Ada\"}' https://jsonplaceholder.typicode.com/users"
```

Check what you saved:

```bash
apigent list
apigent show "create-user"
```

### Run it

```bash
apigent run "list-users"
```

The response body goes to stdout; status lines and notes go to stderr, so piping works:

```bash
apigent run "list-users" | jq '.[0].email'
```

Add assertions, or make non-2xx responses fail the command:

```bash
apigent run "list-users" --assert "status=200" --fail
```

### Move values into an environment

Hard-coded hosts and tokens don't belong in saved requests. Use `{{placeholders}}` and put the values in an environment:

```bash
apigent env create dev \
  --var BASE_URL=https://jsonplaceholder.typicode.com \
  --var API_KEY=- --secret API_KEY          # KEY=- reads the value from stdin, without echo

apigent save "get-user" '{{BASE_URL}}/users/{{id}}' -H 'Authorization: Bearer {{API_KEY}}'

apigent run "get-user" --env dev --var id=1
```

Secret values are encrypted at rest and shown as `*****` by `apigent env show dev`. If a placeholder can't be filled, apigent refuses to send the request and names every missing variable. See [Variables & placeholders]({{< relref "/guides/variables" >}}).

### Group requests in a collection

```bash
apigent collection create "users-api" --var id=1
apigent edit "get-user" --collection users-api
apigent edit "list-users" --collection users-api

apigent collection run "users-api" --env dev
```

Collection variables sit above environment variables and below `--var`. A collection run executes its requests in order (see [`apigent sequence`]({{< relref "/cli/sequence" >}})).

### Inspect what happened

Every run is recorded, including failures:

```bash
apigent history "get-user"          # short ids, status, duration
apigent diag <id>                   # request as sent, variable origins, response, changes since last success
```

{{% /steps %}}

## Next steps

- [Environments & secrets]({{< relref "/guides/environments" >}}) and [Variables & placeholders]({{< relref "/guides/variables" >}})
- [Chaining & extraction]({{< relref "/guides/chaining" >}}) — log in once, reuse the token
- [Agent mode]({{< relref "/ai/agent-mode" >}}) — let an AI agent run your requests without seeing secrets
- [CLI reference]({{< relref "/cli" >}}) — every command and flag
