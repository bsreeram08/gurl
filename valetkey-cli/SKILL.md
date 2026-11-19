---
name: valetkey-cli
description: AI agent skill for using the valetkey CLI (formerly gurl). Run saved HTTP requests by name without seeing secrets, save and manage requests, import from OpenAPI, Insomnia, Bruno, Postman.
---

# valetkey CLI Skill

This skill helps AI agents use the valetkey CLI tool effectively.

## Rules for agents

- Run with `--agent` (or `VALETKEY_AGENT=1`): JSON output, no prompts, secrets masked everywhere.
- Run requests by name: `valetkey --agent run <name> --env <env>`. Never put a live token in `--var`; ask a human to store it with `valetkey env set <env> KEY=- --secret KEY`.
- Start with `valetkey docs` (or `valetkey docs <command>`) for exact flags.
- If the user has set up `valetkey mcp`, prefer its tools: each saved request is a tool whose arguments are the placeholders the environment leaves open.

## When to Use

Use when:
- User wants to save HTTP requests
- User wants to replay API calls
- User wants to import from OpenAPI/Insomnia/Bruno
- User wants to manage API collections
- User wants to update valetkey to the latest version
- Agent needs to make HTTP requests programmatically

## Installation

```bash
# Install valetkey CLI
go install github.com/bsreeram08/valetkey/cmd/valetkey@latest

# macOS
brew install valetkey

# Linux
curl -sL https://raw.githubusercontent.com/bsreeram08/valetkey/master/scripts/install.sh | bash
```

## Core Commands

### save - Save a request
```bash
# Basic GET
valetkey save "ping google" https://google.com

# POST with JSON body
valetkey save "create order" -X POST https://api.example.com/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 123, "items": [{"sku": "ABC"}]}'

# With headers and collection
valetkey save "get user" https://api.example.com/users/123 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" \
  --collection myapi \
  --tag auth

# Using --curl flag (full curl command as string)
valetkey save --curl 'curl -X POST https://api.example.com/orders -H "Content-Type: application/json" -d "{\"id\":1}"' \
  --name "create order" \
  --collection orders

# Save with folder structure
valetkey save "list users" https://api.example.com/api/v2/users \
  --folder api/v2 \
  --collection myapi
```

### run - Execute a saved request
```bash
# By name
valetkey run "ping google"

# With variables
valetkey run "create order" --var customerId=456
valetkey run "get user" --var userId=789

# With environment
valetkey run "get user" --env production --var userId=123

# Use cached response (if available and fresh)
valetkey run "ping google" --cache

# With timeout
valetkey run "slow endpoint" --timeout 30s

# Data-driven iteration (CSV or JSON data file)
valetkey run "create order" --data ./orders.csv

# Assertions
valetkey run "ping google" --assert status=200 --assert "body contains OK"

# Request chaining (setNextRequest)
valetkey run "login" --chain

# Output to file
valetkey run "get report" --output ./report.json
```

### list - Show saved requests
```bash
valetkey list                      # All requests (table format)
valetkey list --json              # JSON output for scripting
valetkey list --collection orders   # Filter by collection
valetkey list --tag auth           # Filter by tag
valetkey list "order*"            # Pattern match
valetkey list --sort name          # Sort by name (name|updated|collection)
valetkey list --limit 10           # Limit results
```

### import - Import from external formats
```bash
# OpenAPI/Swagger
valetkey import openapi ./openapi.yaml --collection myapi

# Insomnia
valetkey import insomnia ./insomnia-export.json

# Bruno
valetkey import bruno ./bruno-requests/

# Postman
valetkey import postman ./collection.json

# HAR (HTTP Archive)
valetkey import har ./requests.har

# Import .env variables into environment
valetkey env import ./vars.env
```

## Advanced Commands

```bash
# History
valetkey history "create order"      # Show history for one request
valetkey timeline                     # Global execution timeline

# Diff responses
valetkey diff "create order"         # Compare last 2 executions

# Edit in TUI
valetkey edit "create order"

# Show request details
valetkey show "create order"
valetkey info "create order"

# Delete/Rename
valetkey delete "old request"
valetkey rename "old name" "new name"

# Export requests
valetkey export "create order"        # Export one request
valetkey export --collection orders  # Export entire collection

# Copy as curl to clipboard
valetkey paste "create order"

# Parse curl from stdin or file
valetkey detect < curl-command.txt
valetkey parse ./curl-commands.txt
```

### env - Environment management
```bash
valetkey env list                    # List all environments
valetkey env create production        # Create environment
valetkey env set API_URL=https://...  # Set variable
valetkey env switch production        # Activate environment
valetkey env import .env             # Import from .env file
valetkey env delete staging          # Delete environment
```

### collection - Collection management
```bash
valetkey collection list             # List all collections
valetkey collection run orders        # Run all requests in a collection
valetkey collection add payments       # Create collection
valetkey collection rename old new     # Rename
valetkey collection remove staging    # Delete
```

### sequence - Request execution order
```bash
valetkey sequence set "req1" "req2" "req3"  # Define order
valetkey sequence list                       # Show current order
```

### update - Update valetkey
```bash
valetkey update    # Check and install latest version
```

### codegen - Generate code from a request
```bash
# Generate code in a language
valetkey codegen "get user" --lang python
valetkey codegen "create order" --lang go
valetkey codegen "api call" --lang javascript
valetkey codegen "api call" --lang curl

# Copy to clipboard
valetkey codegen "api call" --lang python --clipboard
```

### graphql - Execute GraphQL queries
```bash
valetkey graphql https://api.example.com/graphql --query '{ users { id name } }'
```

### tui - Interactive TUI
```bash
valetkey tui    # Launch interactive UI
valetkey ui     # Same
```

## Common Workflows

### API Testing Workflow
```bash
# 1. Save endpoints
valetkey save "list products" GET https://api.example.com/products
valetkey save "get product" GET https://api.example.com/products/123
valetkey save "create product" POST https://api.example.com/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Widget", "price": 9.99}'

# 2. Run them
valetkey run "list products"
valetkey run "get product" --var id=456

# 3. Check history
valetkey history "create product"
```

### Environment-based Workflow
```bash
# 1. Create environments
valetkey env create staging
valetkey env create production

# 2. Set variables per environment
valetkey env switch staging
valetkey env set BASE_URL=https://staging.api.example.com
valetkey env set API_KEY=staging-key

valetkey env switch production
valetkey env set BASE_URL=https://api.example.com
valetkey env set API_KEY=prod-key

# 3. Run with environment (use $VAR in saved requests)
valetkey run "api call" --env production
```

### Bulk Import from OpenAPI
```bash
# 1. Export from your API docs
# 2. Import with collection name
valetkey import openapi ./api-spec.yaml --collection myapi

# 3. List imported
valetkey list --collection myapi

# 4. Run any imported request
valetkey run "list products" --collection myapi
```

## Agent Integration Examples

### Bash Script
```bash
#!/bin/bash
# Make API call and capture response
RESPONSE=$(valetkey run "get user" --var userId=123 --format json)
echo "$RESPONSE" | jq '.data.name'
```

### Python Script
```python
import subprocess
import json

def run_valetkey(name, **vars):
    cmd = ["valetkey", "run", name, "--format", "json"]
    for k, v in vars.items():
        cmd.extend(["--var", f"{k}={v}"])
    result = subprocess.run(cmd, capture_output=True, text=True)
    return json.loads(result.stdout)

user = run_valetkey("get user", userId=123)
print(user["data"]["name"])
```

### Node.js Script
```javascript
import { execSync } from 'child_process';

function runGurl(name, vars = {}) {
  let cmd = `valetkey run "${name}" --format json`;
  for (const [k, v] of Object.entries(vars)) {
    cmd += ` --var ${k}=${v}`;
  }
  return JSON.parse(execSync(cmd, { encoding: 'utf-8' }));
}

const user = runGurl("get user", { userId: 123 });
console.log(user.data.name);
```

## Configuration

Create `~/.valetkeyrc`:
```toml
[general]
history_depth = 100
auto_template = true

[output]
default_format = "auto"
syntax_highlight = true

[cache]
ttl_seconds = 300
```

## Tips for Agents

1. **Use descriptive names**: "create-user" not "api1"
2. **Organize with collections**: `--collection stripe` or `--collection github`
3. **Parameterize with --var**: Reuse requests for different IDs
4. **Import from OpenAPI**: Bulk import saves time
5. **Use --format json**: For machine-readable output
6. **Check history**: `valetkey timeline` shows all executions
7. **Paste for sharing**: `valetkey paste` copies as curl command
8. **Environments**: Use `--env` to switch between staging/prod
9. **Assertions**: Use `--assert` to validate responses in tests
10. **Update regularly**: Run `valetkey update` to get latest version

## Error Handling

```bash
# Request not found
valetkey: request not found: "name"

# Missing variable
valetkey: Missing variable: userId
Usage: valetkey run "name" --var userId=123

# Network error
valetkey: Failed to connect: connection refused

# Assertion failed
valetkey: Assertion failed: expected status=200, got 404
```

## Environment Variables

```bash
VALETKEY_DB_PATH=~/.local/share/valetkey/valetkey.db
VALETKEY_CONFIG_PATH=./.valetkeyrc
```
