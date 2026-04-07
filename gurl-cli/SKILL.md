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

# With headers
gurl save "get user" https://api.example.com/users/123 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json"
```

### run - Execute a request
```bash
# By name
gurl run "ping google"
gurl "ping google"  # shorthand

# With variables
gurl run "create order" --var customerId=456
gurl run "get user" --var userId=789

# Force fresh response
gurl run "ping google" --no-cache
```

### list - Show saved requests
```bash
gurl list                      # All requests
gurl list --json               # JSON output for scripting
gurl list --collection orders   # Filter by collection
gurl list --tag auth           # Filter by tag
gurl list "order*"             # Pattern match
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
```

## Advanced Commands

```bash
# History
gurl history "create order"      # Show history for one request
gurl timeline                     # Global execution timeline

# Diff responses
gurl diff "create order"         # Compare last 2 executions

# Edit
gurl edit "create order"         # Edit in TUI

# Delete/Rename
gurl delete "old request"
gurl rename "old name" "new name"

# Export/Import
gurl export "create order" > order.json
gurl export --collection orders > orders.json
gurl import order.json

# Copy as curl
gurl paste "create order"        # Copy to clipboard
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

## Error Handling

```bash
# Request not found
gurl: request not found: "name"

# Missing variable
gurl: Missing variable: userId
Usage: gurl run "name" --var userId=123

# Network error
gurl: Failed to connect: connection refused
```

## Environment Variables

```bash
GURL_DB_PATH=~/.local/share/gurl/gurl.db
GURL_CONFIG_PATH=./.gurlrc
GURL_TOKEN=your-api-token  # Use in requests with $VAR
```
