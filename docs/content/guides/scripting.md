---
title: "Scripting"
description: "JavaScript pre/post-request scripts for complex API workflows"
weight: 2
---

# Scripting

Gurl supports JavaScript pre-request and post-response scripts powered by the goja runtime (ECMAScript 5.1+). Scripts let you implement dynamic behavior that would be difficult or impossible with static configuration.

## Script Types

### Pre-Request Scripts

Execute before the request is sent. Use to:
- Modify headers dynamically
- Generate authentication tokens
- Set request body based on conditions
- Skip the request entirely

### Post-Response Scripts

Execute after the response is received. Use to:
- Extract data from responses
- Validate response content
- Set variables for subsequent requests
- Chain requests together

## Script API

### Request Object

The `request` object provides access to the outgoing request:

```javascript
// Get a header value
var authHeader = request.headers.get("Authorization");

// Set a header value
request.headers.set("X-Custom-Header", "value");

// Get the request body as string
var body = request.body();

// Set the request body
request.setBody('{"updated": true}');

// Get a variable
var apiKey = gurl.getVariable("API_KEY");
```

### Response Object

The `response` object provides access to the received response:

```javascript
// Get the status code
var status = response.status;

// Get a header value
var contentType = response.header("Content-Type");

// Get the body as text
var text = response.text();

// Get the body as parsed JSON
var data = response.json();

// Get response time in milliseconds
var elapsed = response.time;
```

### Global Functions

```javascript
// Set a variable for subsequent requests
gurl.setVariable("AUTH_TOKEN", "abc123");

// Skip the current request
gurl.skipRequest();

// Make the next request use specific data
gurl.setNextRequest({
    method: "POST",
    url: "https://api.example.com/logout"
});

// Log to the console (printed after script execution)
console.log("Debug: " + value);
```

## Example: Extract Auth Token

A common pattern is to log in, extract the token, and use it in subsequent requests.

```toml
# requests/login.toml
[request]
method = "POST"
url = "{{BASE_URL}}/auth/login"
body = '{"username": "{{USERNAME}}", "password": "{{PASSWORD}}"}'

[headers]
Content-Type = "application/json"

[script]
post-response = """
var data = response.json();
if (data.token) {
    gurl.setVariable("AUTH_TOKEN", data.token);
    console.log("Token extracted: " + data.token);
}
"""
```

Then use the token in subsequent requests:

```toml
# requests/profile.toml
[request]
method = "GET"
url = "{{BASE_URL}}/users/me"

[headers]
Authorization = "Bearer {{AUTH_TOKEN}}"
```

## Example: Dynamic Timestamp Header

```javascript
// Pre-request script to add a timestamp
var timestamp = Math.floor(Date.now() / 1000);
request.headers.set("X-Timestamp", timestamp.toString());

// Sign the request if a secret is available
var secret = gurl.getVariable("SIGNING_SECRET");
if (secret) {
    var body = request.body();
    var signature = hmacSha256(secret, body + timestamp);
    request.headers.set("X-Signature", signature);
}
```

## Example: Conditional Request

Skip a request based on a condition:

```javascript
var skip = gurl.getVariable("SKIP_HEALTH_CHECK");
if (skip === "true") {
    gurl.skipRequest();
}
```

## Example: Response Validation

Validate that a response contains expected data:

```javascript
var data = response.json();

if (data.status !== "success") {
    throw new Error("API returned error status: " + data.status);
}

if (!data.users || data.users.length === 0) {
    throw new Error("Expected users array to be non-empty");
}

console.log("Validation passed. Found " + data.users.length + " users");
```

## Script Storage

Scripts can be defined inline in TOML request files:

```toml
[script]
pre-request = "..."
post-response = "..."
```

Or stored in separate `.js` files and referenced:

```toml
[script]
pre-request-file = "./scripts/auth.js"
post-response-file = "./scripts/validate.js"
```

> [!TIP]
> Keep scripts small and focused. Complex logic belongs in your application code; scripts should only handle API-specific transformations.
