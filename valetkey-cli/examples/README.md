# valetkey CLI Examples

Real-world examples for using valetkey.

## Quick Examples

### Save and Run
```bash
# Save a health check
valetkey save "health" https://api.example.com/health

# Run it
valetkey run "health"

# List all
valetkey list
```

### With Variables
```bash
# Save with variable placeholder
valetkey save "get user" https://api.example.com/users/{{userId}}

# Run with variable
valetkey run "get user" --var userId=123
```

### Import and Use
```bash
# Import from OpenAPI spec
valetkey import openapi ./openapi.yaml --collection myapi

# List imported
valetkey list --collection myapi

# Run any
valetkey run "list users" --collection myapi
```

## See Also

- `SKILL.md` - Full command reference
- `workflows/` - Common API workflows
