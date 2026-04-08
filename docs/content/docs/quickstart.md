---
title: "Quick Start"
weight: 2
---

Get started with Gurl in 5 minutes. This guide walks through the core workflow: saving a request, running it, and generating code.

## Step 1: Save a Request

Save your first request with the `save` command:

```bash
gurl save "users" https://jsonplaceholder.typicode.com/users
```

This saves a GET request to the users endpoint. Gurl creates the collection automatically if it does not exist.

You can also save requests with common options inline:

```bash
gurl save "user-by-id" https://jsonplaceholder.typicode.com/users/1
```

## Step 2: Run the Request

Execute the saved request with the `run` command:

```bash
gurl run "users"
```

Gurl sends the request and displays the response with status code, headers, and body.

To run with verbose output:

```bash
gurl run "users" -v
```

## Step 3: Save with curl Syntax

Import an existing curl command directly:

```bash
gurl save "create-user" --curl "curl -X POST -H 'Content-Type: application/json' -d '{\"name\":\"John\"}' https://jsonplaceholder.typicode.com/users"
```

Gurl parses the curl syntax and saves it as a proper request.

## Step 4: List Saved Requests

See all saved requests:

```bash
gurl list
```

Output shows request name, method, URL, and which environment is active.

To list requests in a specific collection:

```bash
gurl list --collection "my-api"
```

## Step 5: Use Environments

Environments let you swap variables between contexts (local, dev, staging, prod).

Create an environment:

```bash
gurl env create dev --var "BASE_URL=https://dev.api.com" --secret "API_KEY=sk-test-123"
```

Run a request with an environment:

```bash
gurl run "users" --env dev
```

Variables in the URL are substituted automatically:

```bash
gurl save "users" "{{.BASE_URL}}/users"
gurl run "users" --env dev
```

> [!TIP]
> Secrets are encrypted with AES-256-GCM and never appear in logs or generated code.

List all environments:

```bash
gurl env list
```

Show environment variables:

```bash
gurl env show dev
```

## Step 6: Generate Code

Generate client code from any saved request:

```bash
gurl codegen "users" --lang python
```

Supported languages:

- `python` - requests library
- `javascript` - fetch API
- `go` - net/http package
- `curl` - shell curl command

Add the request to the generated code:

```bash
gurl codegen "create-user" --lang python --output api_client.py
```

## Common Workflows

### Chain Requests with Scripts

Add a pre-request script to inject dynamic values:

```bash
gurl edit "auth" --script "
request.headers['X-Request-ID'] = uuid()
"
```

### Run a Collection

Execute all requests in a collection:

```bash
gurl collection run "my-api" --env staging
```

### Export and Import

Export a collection to share:

```bash
gurl export "my-api" --format json --output my-api.json
```

Import elsewhere:

```bash
gurl import my-api.json
```

## Next Steps

- Explore [all CLI commands](/cli/) for advanced usage
- Learn about [authentication handlers](/docs/authentication/) for API security
- Set up [plugins](/docs/plugins/) to extend Gurl
