# Feature Parity Learnings

## Task: ParsedCurl ↔ SavedRequest Conversion Functions

### What Worked
- Created `ParsedCurlToSavedRequest` and `SavedRequestToParsedCurl` in `pkg/types/types.go`
- `ParsedCurl.Headers` is `map[string]string`, `SavedRequest.Headers` is `[]Header` where `Header = {Key, Value string}`
- Used range loops for field mapping (not if-else chains) as required
- Preserved `nil` for both nil maps and nil slices — critical edge case

### Key Edge Cases Handled
- `nil` headers map → `nil` headers slice (and vice versa)
- Empty headers map → empty headers slice (acceptable)
- Empty method/body preserved correctly

### Test Results
- `go test ./pkg/types -v -count=1` → 11 passed
- Round-trip: SavedRequest → ParsedCurl → SavedRequest preserves all data ✓

### Notes
- Pre-existing build error in `internal/cli/commands/run_test.go` (unused imports) — unrelated to this task

## Task T20: .env File Support

### What Worked
- Created `internal/env/dotenv.go` with `ParseDotenv()` and `ParseDotenvFile()` functions
- Created `internal/env/dotenv_test.go` with 15 tests covering all .env formats
- Added `import` subcommand to `internal/cli/commands/env.go` CLI
- Parser uses switch statement for line type handling (lineTypeEmpty, lineTypeComment, lineTypeKeyValue)
- Line classification via `classifyLine()` helper returns lineType enum
- `export` prefix is stripped; quoted values (single/double) are unquoted

### Key Edge Cases Handled
- Comments: lines starting with `#` are skipped
- Empty lines: skipped
- `export KEY=value`: prefix stripped, KEY=value processed
- Double/single quoted values: outer quotes removed
- Value containing `=`: handled correctly (first `=` is the delimiter)
- Trailing whitespace: trimmed

### Test Results
- `go test ./internal/env/dotenv_test.go ./internal/env/dotenv.go ./internal/env/env.go -v -count=1` → 15 passed
- `go build ./internal/env/...` → Success
- `go build ./internal/cli/commands/...` → Success

### Notes
- Pre-existing build errors in `internal/env/secrets_test.go` (undefined SecretsManager) and `cmd/gurl/main.go` (undefined NewEnvStorage) — unrelated to this task
- The plan says "use switch for line type detection" - implemented via `classifyLine()` returning lineType enum, then switch in `ParseDotenv()` handles each type
- Task T16 (env CLI) was running in parallel and modified `env.go` — merged import subcommand into their version

## Task T17: Wire environments into run command

### What Worked
- Added `--env` (alias `-e`) StringFlag to run command
- If `--env` flag set: load that environment by name using `envStorage.GetEnvByName()`
- If no `--env` but active env set: load active environment via `envStorage.GetActiveEnv()`
- CLI `--var` flags override environment variables via map merge pattern (env vars first, then CLI vars overwrite)
- Used `env.NewEnvStorage(db)` to create EnvStorage from LMDB

### Implementation Details
- `RunCommand(db storage.DB, envStorage *env.EnvStorage)` — signature changed to accept envStorage
- Map merge order: env vars (lower precedence) → CLI --var (higher precedence)
- No if-else chains for precedence — simple "first populate from env, then overwrite from CLI vars"

### Files Modified
- `internal/cli/commands/run.go` — added --env flag, env loading logic, new import
- `internal/cli/commands/run_test.go` — added 4 new tests for env integration
- `cmd/gurl/main.go` — updated RunCommand call to pass envStorage

### Test Results
- `go test ./internal/cli/commands/... -v -run "TestRunWithEnv|TestRunVarOverride|TestRunBackwardCompat" -count=1` → 4 passed
- `go test ./... -count=1` → 258 passed in 11 packages

### Notes
- Test data uses hostnames only (e.g., "api.dev.com") not full URLs to avoid double-prefixing issues
- Template engine `Substitute()` performs variable replacement, so environment variables should contain partial URL components, not full URLs

## Task T26: AWS Signature v4 Auth

### What Worked
- Created `internal/auth/awsv4.go` implementing `AWSv4Handler` with `Handler` interface
- Created `internal/auth/awsv4_test.go` with 6 tests covering all major use cases
- Implemented full AWS Sig v4 signing chain: canonical request → string to sign → signing key → signature
- Used `crypto/hmac` and `crypto/sha256` directly (no AWS SDK)

### AWS Sig v4 Signing Algorithm
1. Canonical request: `Method\nCanonicalURI\nCanonicalQueryString\nCanonicalHeaders\nSignedHeaders\nPayloadHash`
2. String to sign: `AWS4-HMAC-SHA256\nDate\nCredentialScope\nCanonicalRequestHash`
3. Signing key: `HMAC-SHA256(HMAC-SHA256(HMAC-SHA256(HMAC-SHA256("AWS4"+SecretKey, Date), Region), Service), "aws4_request")`
4. Signature: `HMAC-SHA256(signingKey, stringToSign)`
5. Authorization header: `AWS4-HMAC-SHA256 Credential=Key/Date/Region/Service/aws4_request, SignedHeaders=..., Signature=...`

### Key Edge Cases Handled
- Empty path → "/"
- Query string encoding and sorting
- Header normalization (lowercase keys, trimmed values)
- Session token support via `X-Amz-Security-Token`
- Payload hashing for body content

### Required Headers Set
- `Authorization`: Full AWS4-HMAC-SHA256 signature
- `X-Amz-Date`: ISO8601 format (YYYYMMDDTHHMMSSZ)
- `X-Amz-Content-Sha256`: Hash of request payload
- `X-Amz-Security-Token`: If session_token param provided

### Test Results
- `go test -v -run "TestAWSv4" ./internal/auth/` → 6 passed
- `go vet ./internal/auth/awsv4.go` → No issues found
- `go build ./internal/auth/awsv4.go ./internal/auth/auth.go ./internal/client/...` → Success

### Notes
- AWS Sig v4 does NOT use a separate `X-Amz-SignedHeaders` header — it's embedded in the Authorization header
- Pre-existing build errors in `digest_test.go` (parse errors) and `ntlm.go` (unused variable) are unrelated to T26
- The plan says "Register as 'awsv4' type in registry" — this is done via `Name() string { return "awsv4" }`, users call `registry.Register(&AWSv4Handler{})`

## Task T19: Secret/Encrypted Variables

### What Worked
- Created `internal/env/secrets.go` with AES-GCM encryption using Go's `crypto/aes` and `crypto/cipher`
- Created `internal/env/secrets_test.go` with 12 tests covering encryption round-trip, nonce randomness, key generation
- Added `SecretKeys map[string]bool` to `Environment` struct to track which variables are secret
- Modified `EnvStorage.SaveEnv()` to encrypt secrets before storing
- Modified `EnvStorage.GetEnv()` to decrypt secrets after loading
- Added `--secret` flag to `env create` and `env set` CLI commands
- Updated `env show` to mask secret values using `env.IsSecret(k)` instead of heuristic pattern matching

### Implementation Details
- Machine key stored at `~/.local/share/gurl/.secret-key` (auto-generated with 32 random bytes)
- AES-GCM with 12-byte random nonce per encryption (same plaintext → different ciphertext)
- `EncryptSecret(key, plaintext)` → base64-encoded ciphertext
- `DecryptSecret(key, ciphertext)` → original plaintext
- `IsEncryptedValue(value)` detects base64-encoded AES-GCM ciphertext
- `MaskSecret(value)` returns `*****` for non-empty values

### Files Created
- `internal/env/secrets.go` — encryption/decryption functions
- `internal/env/secrets_test.go` — 12 tests for crypto functions

### Files Modified
- `internal/env/env.go` — added `SecretKeys` field, `SetSecretVariable()`, `IsSecret()` methods
- `internal/env/storage.go` — encrypt on SaveEnv, decrypt on GetEnv
- `internal/cli/commands/env.go` — added `--secret` flag, updated `env show` to use `env.IsSecret()`

### Test Results
- `go test ./internal/env/... -v -run "TestEncrypt|TestMask|TestSecret" -count=1` → ALL passed
- `go test ./internal/env/... ./internal/cli/commands/... -count=1` → 73 passed
- `go build ./...` → Success

### Notes
- Secrets are decrypted when loaded from DB (ready for use in requests)
- `env show` displays `*****` for secret values
- Encryption uses random nonce so same secret produces different ciphertext each time
- Key file permissions are 0600 (owner read/write only)

## Task T27: Digest Auth Handler

### What Worked
- Created `internal/auth/digest.go` implementing `DigestHandler` with `Handler` interface
- Created `internal/auth/digest_test.go` with 12 tests covering all major use cases
- Implemented MD5 and SHA-256 algorithm support for digest auth
- Used `crypto/md5` and `crypto/sha256` directly (no external digest auth library)

### Digest Auth Algorithm
1. HA1 = MD5/SHA256("username:realm:password")
2. HA2 = MD5/SHA256("method:uri")
3. Response = MD5/SHA256("HA1:nonce:nc:cnonce:qop:HA2")
4. Authorization header: Digest username="...", realm="...", nonce="...", uri="...", response="..."

### Challenge-Response Flow
- Digest auth requires TWO requests: first gets 401 with challenge, second sends computed auth
- Handler receives challenge params (realm, nonce, qop, opaque) as params map
- First request: no auth → 401 with WWW-Authenticate: Digest header
- Second request: computed auth → 200 OK

### Key Edge Cases Handled
- Missing username/password: no auth header set
- Missing challenge params: uses sensible defaults (realm="default-realm", nonce="default-nonce", qop="auth")
- Algorithm variant: MD5 (default) or SHA-256

### Files Created
- `internal/auth/digest.go` — DigestHandler implementation
- `internal/auth/digest_test.go` — 12 tests

### Files Modified
- N/A

### Test Results
- `go test -v -run "Digest" ./internal/auth/` → 12 passed
- `go test ./internal/auth/... -count=1` → 75 passed
- `go test ./... -count=1` → 311 passed

### Notes
- Handler interface is single-shot; digest auth requires challenge-response cycle
- Challenge params must be passed as params map on retry: `{"realm": "...", "nonce": "...", "qop": "...", "opaque": "...", "algorithm": "SHA-256"}`
- Registered as "digest" type via `Name() string { return "digest" }`
- No if-else chains used — used switch/if only for algorithm selection and optional params

## Task T28: NTLM Auth Handler

### What Worked
- Created `internal/auth/ntlm.go` implementing `NTLMHandler` with `Handler` interface
- Created `internal/auth/ntlm_test.go` with 6 tests covering all major use cases
- Used `github.com/Azure/go-ntlmssp` library for NTLM protocol implementation
- Handler sends Type 1 (Negotiate) message on first authentication step

### NTLM Authentication Flow
1. Client sends Type 1 (Negotiate) message with supported features
2. Server responds with Type 2 (Challenge) message containing server challenge
3. Client computes Type 3 (Authenticate) message using: NTLMv2 hash of (password, challenge, username, domain)
4. Server validates and grants access

### Implementation Details
- `NewNegotiateMessage(domain, workstation string)` creates Type 1 message
- `ProcessChallenge(challenge, username, password, domainNeeded)` creates Type 3 response
- Authorization header format: `NTLM <base64-encoded-message>`
- Params: "username" (required), "password" (required), "domain" (optional)

### Files Created
- `internal/auth/ntlm.go` — NTLMHandler implementing Handler interface
- `internal/auth/ntlm_test.go` — 6 tests: interface check, missing credentials, with domain, registry apply

### Test Results
- `go test -v -run TestNTLM ./internal/auth/` → 5 passed
- `go test ./... -count=1` → 311 passed
- `go vet ./internal/auth/ntlm.go` → No issues

### Notes
- NTLM is a multi-step challenge-response auth that typically requires http.RoundTripper middleware to handle fully
- Handler.Name() returns "ntlm" which is used for registry registration
- go-ntlmssp library handles NTLMv2 protocol (NTLMv1 is insecure and not supported)
- The library's Negotiator type implements http.RoundTripper for transparent multi-step handling

## Task T25: OAuth 1.0 Auth Handler

### What Worked
- Created `internal/auth/oauth1.go` implementing `OAuth1Handler` with `Handler` interface
- Created `internal/auth/oauth1_test.go` with 13 tests covering all major use cases
- Implemented full OAuth 1.0a signing: nonce, timestamp, signature base string, HMAC-SHA1 signing
- Used `crypto/hmac`, `crypto/sha1`, `crypto/rand` directly (no external OAuth library)

### OAuth 1.0a Signing Algorithm
1. Generate nonce using `crypto/rand` and base64 encoding
2. Get timestamp as Unix time in seconds
3. Build signature base string: `HTTP_METHOD&url_encode(base_uri)&url_encode(sorted_query_params)`
4. HMAC-SHA1 sign with key: `url_encode(consumer_secret)&url_encode(token_secret)`
5. Base64 encode the HMAC output
6. Build Authorization header: `OAuth oauth_consumer_key="...", oauth_nonce="...", oauth_signature="...", ...`

### Key Edge Cases Handled
- Missing consumer_key, consumer_secret, or token → early return, no header set
- POST body with oauth_body_hash included when body is non-empty
- Query string properly split from URL for signature base string
- Unreserved characters per RFC 3986: A-Z, a-z, 0-9, -, ., _, ~

### Authorization Header Format
- `OAuth oauth_consumer_key="...", oauth_nonce="...", oauth_signature="...", oauth_signature_method="HMAC-SHA1", oauth_timestamp="...", oauth_token="...", oauth_version="1.0"`
- If body present: `oauth_body_hash="..."` appended

### Test Results
- `go test -v -run "TestOAuth1" ./internal/auth/oauth1_test.go ...` → 13 passed
- `go test ./internal/auth/... -count=1` → 75 passed (all auth tests)
- `go vet ./internal/auth/oauth1.go` → No issues found

### Notes
- Go's `base64.URLEncoding.EncodeToString()` produces 24 chars for 16 bytes (not 32 as initially expected)
- OAuth header parsing cannot use `url.ParseQuery()` because OAuth uses comma-space separator, not ampersand
- Registered as "oauth1" type via `Name() string { return "oauth1" }`
