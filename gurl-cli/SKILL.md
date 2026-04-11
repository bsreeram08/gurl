---
name: gurl-cli
description: AI agent skill for using the Gurl CLI tool. Save, run, and manage HTTP requests with variable templates. Import from OpenAPI, Insomnia, Bruno, Postman.
---

# Gurl CLI Skill

**GURL** = **G**url's **U**niversal **R**equest **L**ibrary

This skill helps AI agents use the Gurl CLI tool effectively.

## When to Use

Use when:
- User wants to save HTTP requests
- User wants to replay API calls
- User wants to import from OpenAPI/Insomnia/Bruno
- User wants to manage API collections
- User wants to update gurl to the latest version
- Agent needs to make HTTP requests programmatically

## Installation

```bash
# Install gurl CLI
go install github.com/bsreeram08/gurl@latest

# macOS
brew install gurl

# Linux
curl -sL https://raw.githubusercontent.com/bsreeram08/gurl/master/scripts/install.sh | bash
```

## Core Commands

### save - Save a request
```bash
# Basic GET
gurl save "ping google" https://google.com

# POST with JSON body
gurl save "create order" -X POST https://api.example.com/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 123, "items": [{"sku": "ABC"}]}'

# With headers and collection
gurl save "get user" https://api.example.com/users/123 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" \
  --collection myapi \
  --tag auth

# Using --curl flag (full curl command as string)
gurl save --curl 'curl -X POST https://api.example.com/orders -H "Content-Type: application/json" -d "{\"id\":1}"' \
  --name "create order" \
  --collection orders

# Save with folder structure
gurl save "list users" https://api.example.com/api/v2/users \
  --folder api/v2 \
  --collection myapi
```

### run - Execute a saved request
```bash
# By name
gurl run "ping google"

# With variables
gurl run "create order" --var customerId=456
gurl run "get user" --var userId=789

# With environment
gurl run "get user" --env production --var userId=123

# Use cached response (if available and fresh)
gurl run "ping google" --cache

# With timeout
gurl run "slow endpoint" --timeout 30s

# Data-driven iteration (CSV or JSON data file)
gurl run "create order" --data ./orders.csv

# Assertions
gurl run "ping google" --assert status=200 --assert "body contains OK"

# Request chaining (setNextRequest)
gurl run "login" --chain

# Output to file
gurl run "get report" --output ./report.json
```

### list - Show saved requests
```bash
gurl list                      # All requests (table format)
gurl list --json              # JSON output for scripting
gurl list --collection orders   # Filter by collection
gurl list --tag auth           # Filter by tag
gurl list "order*"            # Pattern match
gurl list --sort name          # Sort by name (name|updated|collection)
gurl list --limit 10           # Limit results
```

### import - Import from external formats
```bash
# OpenAPI/Swagger
gurl import openapi ./openapi.yaml --collection myapi

# Insomnia
gurl import insomnia ./insomnia-export.json

# Bruno
gurl import bruno ./bruno-requests/

# Postman
gurl import postman ./collection.json

# HAR (HTTP Archive)
gurl import har ./requests.har

# Import .env variables into environment
gurl env import ./vars.env
```

## Advanced Commands

```bash
# History
gurl history "create order"      # Show history for one request
gurl timeline                     # Global execution timeline

# Diff responses
gurl diff "create order"         # Compare last 2 executions

# Edit in TUI
gurl edit "create order"

# Show request details
gurl show "create order"
gurl info "create order"

# Delete/Rename
gurl delete "old request"
gurl rename "old name" "new name"

# Export requests
gurl export "create order"        # Export one request
gurl export --collection orders  # Export entire collection

# Copy as curl to clipboard
gurl paste "create order"

# Parse curl from stdin or file
gurl detect < curl-command.txt
gurl parse ./curl-commands.txt
```

### env - Environment management
```bash
gurl env list                    # List all environments
gurl env create production        # Create environment
gurl env set API_URL=https://...  # Set variable
gurl env switch production        # Activate environment
gurl env import .env             # Import from .env file
gurl env delete staging          # Delete environment
```

### collection - Collection management
```bash
gurl collection list             # List all collections
gurl collection run orders        # Run all requests in a collection
gurl collection add payments       # Create collection
gurl collection rename old new     # Rename
gurl collection remove staging    # Delete
```

### sequence - Request execution order
```bash
gurl sequence set "req1" "req2" "req3"  # Define order
gurl sequence list                       # Show current order
```

### update - Update gurl
```bash
gurl update    # Check and install latest version
```

### codegen - Generate code from a request
```bash
# Generate code in a language
gurl codegen "get user" --lang python
gurl codegen "create order" --lang go
gurl codegen "api call" --lang javascript
gurl codegen "api call" --lang curl

# Copy to clipboard
gurl codegen "api call" --lang python --clipboard
```

### graphql - Execute GraphQL queries
```bash
gurl graphql https://api.example.com/graphql --query '{ users { id name } }'
```

### tui - Interactive TUI
```bash
gurl tui    # Launch interactive UI
gurl ui     # Same
```

## Common Workflows

### API Testing Workflow
```bash
# 1. Save endpoints
gurl save "list products" GET https://api.example.com/products
gurl save "get product" GET https://api.example.com/products/123
gurl save "create product" POST https://api.example.com/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Widget", "price": 9.99}'

# 2. Run them
gurl run "list products"
gurl run "get product" --var id=456

# 3. Check history
gurl history "create product"
```

### Environment-based Workflow
```bash
# 1. Create environments
gurl env create staging
gurl env create production

# 2. Set variables per environment
gurl env switch staging
gurl env set BASE_URL=https://staging.api.example.com
gurl env set API_KEY=staging-key

gurl env switch production
gurl env set BASE_URL=https://api.example.com
gurl env set API_KEY=prod-key

# 3. Run with environment (use $VAR in saved requests)
gurl run "api call" --env production
```

### Bulk Import from OpenAPI
```bash
# 1. Export from your API docs
# 2. Import with collection name
gurl import openapi ./api-spec.yaml --collection myapi

# 3. List imported
gurl list --collection myapi

# 4. Run any imported request
gurl run "list products" --collection myapi
```

## Agent Integration Examples

### Bash Script
```bash
#!/bin/bash
# Make API call and capture response
RESPONSE=$(gurl run "get user" --var userId=123 --format json)
echo "$RESPONSE" | jq '.data.name'
```

### Python Script
```python
import subprocess
import json

def run_gurl(name, **vars):
    cmd = ["gurl", "run", name, "--format", "json"]
    for k, v in vars.items():
        cmd.extend(["--var", f"{k}={v}"])
    result = subprocess.run(cmd, capture_output=True, text=True)
    return json.loads(result.stdout)

user = run_gurl("get user", userId=123)
print(user["data"]["name"])
```

### Node.js Script
```javascript
import { execSync } from 'child_process';

function runGurl(name, vars = {}) {
  let cmd = `gurl run "${name}" --format json`;
  for (const [k, v] of Object.entries(vars)) {
    cmd += ` --var ${k}=${v}`;
  }
  return JSON.parse(execSync(cmd, { encoding: 'utf-8' }));
}

const user = runGurl("get user", { userId: 123 });
console.log(user.data.name);
```

## Configuration

Create `~/.gurlrc`:
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
6. **Check history**: `gurl timeline` shows all executions
7. **Paste for sharing**: `gurl paste` copies as curl command
8. **Environments**: Use `--env` to switch between staging/prod
9. **Assertions**: Use `--assert` to validate responses in tests
10. **Update regularly**: Run `gurl update` to get latest version

## Error Handling

```bash
# Request not found
gurl: request not found: "name"

# Missing variable
gurl: Missing variable: userId
Usage: gurl run "name" --var userId=123

# Network error
gurl: Failed to connect: connection refused

# Assertion failed
gurl: Assertion failed: expected status=200, got 404
```

## Environment Variables

```bash
GURL_DB_PATH=~/.local/share/gurl/gurl.db
GURL_CONFIG_PATH=./.gurlrc
```
