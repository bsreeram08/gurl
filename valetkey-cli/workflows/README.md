# Common API Workflows

Copy-paste workflows for common API tasks.

## Stripe Payment Workflow

```bash
# 1. Save Stripe endpoints
valetkey save "stripe list customers" \
  -H "Authorization: Bearer $STRIPE_KEY" \
  https://api.stripe.com/v1/customers

valetkey save "stripe create payment" -X POST \
  -H "Authorization: Bearer $STRIPE_KEY" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "amount=2000&currency=usd&source=tok_visa" \
  https://api.stripe.com/v1/charges

# 2. Run
valetkey run "stripe list customers"
valetkey run "stripe create payment"
```

## GitHub API Workflow

```bash
# 1. Save GitHub endpoints
valetkey save "gh list repos" \
  -H "Authorization: token $GH_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/user/repos

valetkey save "gh create issue" -X POST \
  -H "Authorization: token $GH_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  -H "Content-Type: application/json" \
  -d '{"title": "Bug: Login broken", "body": "Steps to reproduce..."}' \
  https://api.github.com/repos/owner/repo/issues

# 2. Run
valetkey run "gh list repos"
valetkey run "gh create issue"
```

## REST API CRUD Workflow

```bash
# 1. Save CRUD endpoints
valetkey save "api list" GET https://api.example.com/items
valetkey save "api get" GET https://api.example.com/items/{{id}}
valetkey save "api create" POST https://api.example.com/items \
  -H "Content-Type: application/json" \
  -d '{"name": "{{name}}", "qty": {{qty}}}'
valetkey save "api update" PUT https://api.example.com/items/{{id}} \
  -H "Content-Type: application/json" \
  -d '{"name": "{{name}}"}'
valetkey save "api delete" DELETE https://api.example.com/items/{{id}}

# 2. Use with variables
valetkey run "api get" --var id=123
valetkey run "api create" --var name=Widget --var qty=5
valetkey run "api update" --var id=123 --var name=UpdatedName
valetkey run "api delete" --var id=123
```

## Health Check Dashboard

```bash
# 1. Save multiple health endpoints
valetkey save "health api" https://api.example.com/health
valetkey save "health db" https://api.example.com/health/db
valetkey save "health cache" https://api.example.com/health/cache

# 2. Run all
valetkey run "health api"
valetkey run "health db"
valetkey run "health cache"

# 3. Check timeline
valetkey timeline
```
