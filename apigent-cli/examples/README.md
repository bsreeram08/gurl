# apigent CLI Examples

Real-world examples for using apigent.

## Quick Examples

### Save and Run
```bash
# Save a health check
apigent save "health" https://api.example.com/health

# Run it
apigent run "health"

# List all
apigent list
```

### With Variables
```bash
# Save with variable placeholder
apigent save "get user" https://api.example.com/users/{{userId}}

# Run with variable
apigent run "get user" --var userId=123
```

### Import and Use
```bash
# Import from OpenAPI spec
apigent import openapi ./openapi.yaml --collection myapi

# List imported
apigent list --collection myapi

# Run any
apigent run "list users" --collection myapi
```

## See Also

- `SKILL.md` - Full command reference
- `workflows/` - Common API workflows
