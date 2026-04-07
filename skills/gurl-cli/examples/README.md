# Gurl CLI Examples

Real-world examples for using Gurl.

## Quick Examples

### Save and Run
```bash
# Save a health check
gurl save "health" https://api.example.com/health

# Run it
gurl run "health"

# List all
gurl list
```

### With Variables
```bash
# Save with variable placeholder
gurl save "get user" https://api.example.com/users/{{userId}}

# Run with variable
gurl run "get user" --var userId=123
```

### Import and Use
```bash
# Import from OpenAPI spec
gurl import openapi ./openapi.yaml --collection myapi

# List imported
gurl list --collection myapi

# Run any
gurl run "list users" --collection myapi
```

## See Also

- `SKILL.md` - Full command reference
- `workflows/` - Common API workflows
