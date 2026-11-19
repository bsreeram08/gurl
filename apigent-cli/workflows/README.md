# Common API Workflows

Copy-paste workflows for common API tasks.

## Stripe Payment Workflow

```bash
# 1. Save Stripe endpoints
apigent save "stripe list customers" \
  -H "Authorization: Bearer $STRIPE_KEY" \
  https://api.stripe.com/v1/customers

apigent save "stripe create payment" -X POST \
  -H "Authorization: Bearer $STRIPE_KEY" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "amount=2000&currency=usd&source=tok_visa" \
  https://api.stripe.com/v1/charges

# 2. Run
apigent run "stripe list customers"
apigent run "stripe create payment"
```

## GitHub API Workflow

```bash
# 1. Save GitHub endpoints
apigent save "gh list repos" \
  -H "Authorization: token $GH_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/user/repos

apigent save "gh create issue" -X POST \
  -H "Authorization: token $GH_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  -H "Content-Type: application/json" \
  -d '{"title": "Bug: Login broken", "body": "Steps to reproduce..."}' \
  https://api.github.com/repos/owner/repo/issues

# 2. Run
apigent run "gh list repos"
apigent run "gh create issue"
```

## REST API CRUD Workflow

```bash
# 1. Save CRUD endpoints
apigent save "api list" GET https://api.example.com/items
apigent save "api get" GET https://api.example.com/items/{{id}}
apigent save "api create" POST https://api.example.com/items \
  -H "Content-Type: application/json" \
  -d '{"name": "{{name}}", "qty": {{qty}}}'
apigent save "api update" PUT https://api.example.com/items/{{id}} \
  -H "Content-Type: application/json" \
  -d '{"name": "{{name}}"}'
apigent save "api delete" DELETE https://api.example.com/items/{{id}}

# 2. Use with variables
apigent run "api get" --var id=123
apigent run "api create" --var name=Widget --var qty=5
apigent run "api update" --var id=123 --var name=UpdatedName
apigent run "api delete" --var id=123
```

## Health Check Dashboard

```bash
# 1. Save multiple health endpoints
apigent save "health api" https://api.example.com/health
apigent save "health db" https://api.example.com/health/db
apigent save "health cache" https://api.example.com/health/cache

# 2. Run all
apigent run "health api"
apigent run "health db"
apigent run "health cache"

# 3. Check timeline
apigent timeline
```
