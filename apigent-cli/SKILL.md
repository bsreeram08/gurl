---
name: apigent-cli
description: AI agent skill for using the apigent CLI (formerly gurl). Run saved HTTP requests by name without seeing secrets, save and manage requests, import from OpenAPI, Insomnia, Bruno, Postman.
---

# apigent CLI Skill

This skill helps AI agents use the apigent CLI tool effectively.

## Rules for agents

- Run with `--agent` (or `APIGENT_AGENT=1`): JSON output, no prompts, secrets masked everywhere.
- Run requests by name: `apigent --agent run <name> --env <env>`. Never put a live token in `--var`; ask a human to store it with `apigent env set <env> KEY=- --secret KEY`.
- Start with `apigent docs` (or `apigent docs <command>`) for exact flags.
- If the user has set up `apigent mcp`, prefer its tools: each saved request is a tool whose arguments are the placeholders the environment leaves open.

## When to Use

Use when:
- User wants to save HTTP requests
- User wants to replay API calls
- User wants to import from OpenAPI/Insomnia/Bruno
- User wants to manage API collections
- User wants to update apigent to the latest version
- Agent needs to make HTTP requests programmatically

## Installation

```bash
# Install apigent CLI
go install github.com/bsreeram08/apigent/cmd/apigent@latest

# macOS
brew install apigent

# Linux
curl -sL https://raw.githubusercontent.com/bsreeram08/apigent/master/scripts/install.sh | bash
```

## Core Commands

### save - Save a request
```bash
# Basic GET
apigent save "ping google" https://google.com

# POST with JSON body
apigent save "create order" -X POST https://api.example.com/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 123, "items": [{"sku": "ABC"}]}'

# With headers and collection
apigent save "get user" https://api.example.com/users/123 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" \
  --collection myapi \
  --tag auth

# Using --curl flag (full curl command as string)
apigent save --curl 'curl -X POST https://api.example.com/orders -H "Content-Type: application/json" -d "{\"id\":1}"' \
  --name "create order" \
  --collection orders

# Save with folder structure
apigent save "list users" https://api.example.com/api/v2/users \
  --folder api/v2 \
  --collection myapi
```

### run - Execute a saved request
```bash
# By name
apigent run "ping google"

# With variables
apigent run "create order" --var customerId=456
apigent run "get user" --var userId=789

# With environment
apigent run "get user" --env production --var userId=123

# Use cached response (if available and fresh)
apigent run "ping google" --cache

# With timeout
apigent run "slow endpoint" --timeout 30s

# Data-driven iteration (CSV or JSON data file)
apigent run "create order" --data ./orders.csv

# Assertions
apigent run "ping google" --assert status=200 --assert "body contains OK"

# Request chaining (setNextRequest)
apigent run "login" --chain

# Output to file
apigent run "get report" --output ./report.json
```

### list - Show saved requests
```bash
apigent list                      # All requests (table format)
apigent list --json              # JSON output for scripting
apigent list --collection orders   # Filter by collection
apigent list --tag auth           # Filter by tag
apigent list "order*"            # Pattern match
apigent list --sort name          # Sort by name (name|updated|collection)
apigent list --limit 10           # Limit results
```

### import - Import from external formats
```bash
# OpenAPI/Swagger
apigent import openapi ./openapi.yaml --collection myapi

# Insomnia
apigent import insomnia ./insomnia-export.json

# Bruno
apigent import bruno ./bruno-requests/

# Postman
apigent import postman ./collection.json

# HAR (HTTP Archive)
apigent import har ./requests.har

# Import .env variables into environment
apigent env import ./vars.env
```

## Advanced Commands

```bash
# History
apigent history "create order"      # Show history for one request
apigent timeline                     # Global execution timeline

# Diff responses
apigent diff "create order"         # Compare last 2 executions

# Edit in TUI
apigent edit "create order"

# Show request details
apigent show "create order"
apigent info "create order"

# Delete/Rename
apigent delete "old request"
apigent rename "old name" "new name"

# Export requests
apigent export "create order"        # Export one request
apigent export --collection orders  # Export entire collection

# Copy as curl to clipboard
apigent paste "create order"

# Parse curl from stdin or file
apigent detect < curl-command.txt
apigent parse ./curl-commands.txt
```

### env - Environment management
```bash
apigent env list                    # List all environments
apigent env create production        # Create environment
apigent env set API_URL=https://...  # Set variable
apigent env switch production        # Activate environment
apigent env import .env             # Import from .env file
apigent env delete staging          # Delete environment
```

### collection - Collection management
```bash
apigent collection list             # List all collections
apigent collection run orders        # Run all requests in a collection
apigent collection add payments       # Create collection
apigent collection rename old new     # Rename
apigent collection remove staging    # Delete
```

### sequence - Request execution order
```bash
apigent sequence set "req1" "req2" "req3"  # Define order
apigent sequence list                       # Show current order
```

### update - Update apigent
```bash
apigent update    # Check and install latest version
```

### codegen - Generate code from a request
```bash
# Generate code in a language
apigent codegen "get user" --lang python
apigent codegen "create order" --lang go
apigent codegen "api call" --lang javascript
apigent codegen "api call" --lang curl

# Copy to clipboard
apigent codegen "api call" --lang python --clipboard
```

### graphql - Execute GraphQL queries
```bash
apigent graphql https://api.example.com/graphql --query '{ users { id name } }'
```

### tui - Interactive TUI
```bash
apigent tui    # Launch interactive UI
apigent ui     # Same
```

## Common Workflows

### API Testing Workflow
```bash
# 1. Save endpoints
apigent save "list products" GET https://api.example.com/products
apigent save "get product" GET https://api.example.com/products/123
apigent save "create product" POST https://api.example.com/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Widget", "price": 9.99}'

# 2. Run them
apigent run "list products"
apigent run "get product" --var id=456

# 3. Check history
apigent history "create product"
```

### Environment-based Workflow
```bash
# 1. Create environments
apigent env create staging
apigent env create production

# 2. Set variables per environment
apigent env switch staging
apigent env set BASE_URL=https://staging.api.example.com
apigent env set API_KEY=staging-key

apigent env switch production
apigent env set BASE_URL=https://api.example.com
apigent env set API_KEY=prod-key

# 3. Run with environment (use $VAR in saved requests)
apigent run "api call" --env production
```

### Bulk Import from OpenAPI
```bash
# 1. Export from your API docs
# 2. Import with collection name
apigent import openapi ./api-spec.yaml --collection myapi

# 3. List imported
apigent list --collection myapi

# 4. Run any imported request
apigent run "list products" --collection myapi
```

## Agent Integration Examples

### Bash Script
```bash
#!/bin/bash
# Make API call and capture response
RESPONSE=$(apigent run "get user" --var userId=123 --format json)
echo "$RESPONSE" | jq '.data.name'
```

### Python Script
```python
import subprocess
import json

def run_apigent(name, **vars):
    cmd = ["apigent", "run", name, "--format", "json"]
    for k, v in vars.items():
        cmd.extend(["--var", f"{k}={v}"])
    result = subprocess.run(cmd, capture_output=True, text=True)
    return json.loads(result.stdout)

user = run_apigent("get user", userId=123)
print(user["data"]["name"])
```

### Node.js Script
```javascript
import { execSync } from 'child_process';

function runGurl(name, vars = {}) {
  let cmd = `apigent run "${name}" --format json`;
  for (const [k, v] of Object.entries(vars)) {
    cmd += ` --var ${k}=${v}`;
  }
  return JSON.parse(execSync(cmd, { encoding: 'utf-8' }));
}

const user = runGurl("get user", { userId: 123 });
console.log(user.data.name);
```

## Configuration

Create `~/.apigentrc`:
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
6. **Check history**: `apigent timeline` shows all executions
7. **Paste for sharing**: `apigent paste` copies as curl command
8. **Environments**: Use `--env` to switch between staging/prod
9. **Assertions**: Use `--assert` to validate responses in tests
10. **Update regularly**: Run `apigent update` to get latest version

## Error Handling

```bash
# Request not found
apigent: request not found: "name"

# Missing variable
apigent: Missing variable: userId
Usage: apigent run "name" --var userId=123

# Network error
apigent: Failed to connect: connection refused

# Assertion failed
apigent: Assertion failed: expected status=200, got 404
```

## Environment Variables

```bash
APIGENT_DB_PATH=~/.local/share/apigent/apigent.db
APIGENT_CONFIG_PATH=./.apigentrc
```
