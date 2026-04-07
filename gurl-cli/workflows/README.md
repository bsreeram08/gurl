# Common API Workflows

Copy-paste workflows for common API tasks.

## Stripe Payment Workflow

```bash
# 1. Save Stripe endpoints
gurl save "stripe list customers" \
  -H "Authorization: Bearer $STRIPE_KEY" \
  https://api.stripe.com/v1/customers

gurl save "stripe create payment" -X POST \
  -H "Authorization: Bearer $STRIPE_KEY" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "amount=2000&currency=usd&source=tok_visa" \
  https://api.stripe.com/v1/charges

# 2. Run
gurl run "stripe list customers"
gurl run "stripe create payment"
```

## GitHub API Workflow

```bash
# 1. Save GitHub endpoints
gurl save "gh list repos" \
  -H "Authorization: token $GH_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/user/repos

gurl save "gh create issue" -X POST \
  -H "Authorization: token $GH_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  -H "Content-Type: application/json" \
  -d '{"title": "Bug: Login broken", "body": "Steps to reproduce..."}' \
  https://api.github.com/repos/owner/repo/issues

# 2. Run
gurl run "gh list repos"
gurl run "gh create issue"
```

## REST API CRUD Workflow

```bash
# 1. Save CRUD endpoints
gurl save "api list" GET https://api.example.com/items
gurl save "api get" GET https://api.example.com/items/{{id}}
gurl save "api create" POST https://api.example.com/items \
  -H "Content-Type: application/json" \
  -d '{"name": "{{name}}", "qty": {{qty}}}'
gurl save "api update" PUT https://api.example.com/items/{{id}} \
  -H "Content-Type: application/json" \
  -d '{"name": "{{name}}"}'
gurl save "api delete" DELETE https://api.example.com/items/{{id}}

# 2. Use with variables
gurl run "api get" --var id=123
gurl run "api create" --var name=Widget --var qty=5
gurl run "api update" --var id=123 --var name=UpdatedName
gurl run "api delete" --var id=123
```

## Health Check Dashboard

```bash
# 1. Save multiple health endpoints
gurl save "health api" https://api.example.com/health
gurl save "health db" https://api.example.com/health/db
gurl save "health cache" https://api.example.com/health/cache

# 2. Run all
gurl run "health api"
gurl run "health db"
gurl run "health cache"

# 3. Check timeline
gurl timeline
```
