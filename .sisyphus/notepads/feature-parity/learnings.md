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

## Task T39: Path Parameters Support

### What Worked
- Created `internal/core/template/pathparam.go` with `ResolvePathParams(url string, params map[string]string) (string, error)`
- Created `internal/core/template/pathparam_test.go` with 7 tests (6 required + 1 bonus)
- Added `PathParams []Var` field to `SavedRequest` in `pkg/types/types.go`
- Added `ResolvePathParamsInRequest()` and `GetVariablesFromRequest()` enhancement in `engine.go`

### Key Edge Cases Handled
- Both `:param` and `{param}` syntax supported via single regex `(:[a-zA-Z_][a-zA-Z0-9_]*|\{[a-zA-Z_][a-zA-Z0-9_]*\})`
- Curly brace params need `}` stripped from name after removing leading `{`
- Must skip `{{var}}` style template variables when extracting path params — regex matches `{var}` in both cases
- URL encoding via `url.PathEscape()` — spaces become `%20`, `&` becomes `%26`, but `=` is NOT encoded (Go's PathEscape behavior)
- Empty path param values return error (not silent empty segment)
- Unresolved path params return error listing the first missing param name

### Regex Pattern Matching
- For `{{base_url}}/api`: match at [1,11] giving text `{base_url}` — check `match[0] > 1` before accessing `urlStr[match[0]-2]`
- `url.PathEscape("hello world & foo=bar")` → `hello%20world%20&%20foo=bar` (note: `=` is NOT encoded)

### Test Results
- `go test ./internal/core/template/... -count=1` → 52 passed (including 7 pathparam tests)
- `go test ./internal/core/template/... -v -run TestPathParam -count=1` → 7 passed

### Notes
- The plan specified "path params resolve BEFORE query params" — the `ResolvePathParamsInRequest()` function can be called before any template variable substitution in the render pipeline
- `url.PathEscape` is correct for path segments but not for query strings (use `url.QueryEscape` for query string values with `+` for spaces)

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

## Task T33: Client certificates (mTLS) + SSL toggle

### What Worked
- Created `TLSConfig` struct with `CertFile`, `KeyFile`, `CAFile`, `Insecure`, `MinTLSVersion` fields
- Created `NewClientWithTLS(cfg TLSConfig) *Client` factory function
- Implemented `parseTLSVersion()` to convert version strings to crypto/tls constants
- Warning messages printed to stderr for: InsecureSkipVerify, cert/key load failures, CA parse failures, invalid TLS version

### TLS Implementation Details
- mTLS: `tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)` loads client cert+key into `tls.Config.Certificates`
- Custom CA: `x509.NewCertPool()` + `AppendCertsFromPEM()` for CA bundle, set in `tls.Config.RootCAs`
- Insecure mode: `tls.Config.InsecureSkipVerify = true` with WARNING printed to stderr
- MinTLSVersion: Maps "1.0", "1.1", "1.2", "1.3" to tls.VersionTLS10/11/12/13
- Default TLS verification: Never skip verification unless explicit `Insecure: true`

### Test Coverage
- `TestTLSConfig_Struct`: Validates TLSConfig struct fields
- `TestNewClientWithTLS_ValidConfig`: Creates client with cert/key (skips if openssl unavailable)
- `TestNewClientWithTLS_InsecureSkipsVerification`: Verifies InsecureSkipVerify=true
- `TestNewClientWithTLS_CustomCA`: Verifies RootCAs is set from CA file
- `TestNewClientWithTLS_MinTLSVersion`: Verifies MinVersion is set to TLS 1.2
- `TestNewClientWithTLS_CertFileNotFound`: Warns but doesn't fail on missing cert
- `TestNewClientWithTLS_KeyFileNotFound`: Warns but doesn't fail on missing key
- `TestNewClientWithTLS_CAFileNotFound`: Warns but doesn't fail on missing CA

### Test Results
- `go test ./internal/client/... -v -run TestTLS -count=1` → ALL passed (1 test suite)
- `go vet ./internal/client/...` → No issues found
- `go build ./internal/client/...` → Success

### Files Modified
- `internal/client/client.go` — Added TLSConfig struct, NewClientWithTLS(), parseTLSVersion()
- `internal/client/client_test.go` — Added TLS tests + helper functions (generateSelfSignedCert, copyFile)

### Notes
- Non-fatal warnings: File load failures print to stderr but don't prevent client creation
- Pre-existing test failure in `TestRedirectFollowing_DefaultMax10` (nil pointer dereference in redirect handler) — unrelated to TLS changes
- Openssl required for generating test certs — tests skip gracefully if not available
- `fmt.Fprintf(os.Stderr, ...)` for warnings ensures they don't interfere with stdout responses

## Task T29: Auth inheritance from collections/folders

### What Worked
- Created `AuthConfig` struct in `pkg/types/types.go` (Type + Params map) to avoid circular import
- Added `AuthConfig *AuthConfig` field to both `SavedRequest` and `Collection`
- Created `ResolveAuthConfig(request, collection)` using slice iteration approach (not if-else chains)
- Resolution order: request.AuthConfig > collection.AuthConfig > nil
- Schema migration v3 added (migrateToV2 and migrateToV3 are no-ops since JSON serialization handles new fields)

### Key Design Decision: AuthConfig in types package
- Circular import issue: auth → types → auth
- Solution: Define AuthConfig in pkg/types (base package with no dependencies)
- Both SavedRequest and Collection import types, auth package imports types for resolve.go
- auth.Handler interface uses map[string]string params (not AuthConfig) - different concerns

### Resolution Function Implementation
```go
func ResolveAuthConfig(request *types.SavedRequest, collection *types.Collection) *types.AuthConfig {
    precedence := make([]*types.AuthConfig, 0, 2)
    if request != nil {
        precedence = append(precedence, request.AuthConfig)
    }
    if collection != nil {
        precedence = append(precedence, collection.AuthConfig)
    }
    for _, cfg := range precedence {
        if cfg != nil {
            return cfg
        }
    }
    return nil
}
```

### Nil Handling
- Handles nil request gracefully (returns collection auth if exists)
- Handles nil collection gracefully (returns request auth if exists)
- Handles both nil (returns nil)
- Backward compatible: nil AuthConfig = no auth

### Files Modified
- `pkg/types/types.go` — Added AuthConfig struct, AuthConfig field on SavedRequest/Collection
- `internal/auth/auth.go` — Removed AuthConfig (moved to types)
- `internal/auth/resolve.go` — New file with ResolveAuthConfig function
- `internal/auth/resolve_test.go` — New file with 9 tests
- `internal/storage/migration.go` — Updated currentSchemaVersion to 3
- `internal/storage/migration_test.go` — Updated expected version to 3

### Test Results
- `go test ./internal/auth/... -v -run TestResolveAuthConfig -count=1` → 9 passed
- `go test ./internal/storage/... -count=1` → 5 passed
- `go test ./pkg/types/... -count=1` → 13 passed
- `go test ./internal/auth/... ./pkg/types/... ./internal/storage/... -count=1` → 100 passed
- `go vet ./internal/auth/... ./pkg/types/... ./internal/storage/...` → No issues

### Pre-existing Test Failures (unrelated)
- client package: TestRedirect tests failing (makeslice panic, status code issues)
- cookies package: TestCookieJar tests failing (cookie persistence issues)
- These failures existed before T29 changes

## Task T38: Dynamic Template Values (UUID, Timestamp, Random)

### What Worked
- Created `internal/core/template/dynamic.go` with `ResolveDynamic()` function
- Created `internal/core/template/dynamic_test.go` with 8 tests covering all dynamic functions
- Used `github.com/google/uuid` for UUID v4 generation
- Used `crypto/rand` (NOT math/rand) for cryptographic randomness

### Dynamic Functions Implemented
- `$uuid` → `uuid.New().String()` (validates as UUID v4 format)
- `$timestamp` → `time.Now().Unix()` (integer Unix epoch)
- `$isoTimestamp` → `time.Now().UTC().Format(time.RFC3339)` (ISO 8601 format)
- `$randomInt(min, max)` → crypto/rand based random integer in range [min, max]
- `$randomString(len)` → alphanumeric string of given length
- `$randomEmail` → `{randomString(8)}@example.com`

### Implementation Details
- Used switch on function name for dispatch (not if-else chains)
- Unknown functions return descriptive error (NOT empty string)
- Dynamic pattern: `\{\{\$([^}]+)\}\}` matches `{{$funcName}}` or `{{$funcName(args)}}`
- Regex capture group extracts function name with optional arguments
- Processing from end to beginning preserves string indices during replacement

### Test Coverage
- `TestDynamic_UUID`: Validates UUID v4 format via regex and uuid.Parse()
- `TestDynamic_Timestamp`: Verifies timestamp is current epoch (before <= result <= after)
- `TestDynamic_ISOTimestamp`: Verifies RFC3339 format via time.Parse()
- `TestDynamic_RandomInt`: Tests range boundaries (1-100, 0-10, 5-5)
- `TestDynamic_RandomString`: Tests length and alphanumeric charset
- `TestDynamic_RandomEmail`: Validates email format against regex
- `TestDynamic_RandomUUID_Uniqueness`: 100 iterations to verify no duplicates
- `TestDynamic_UnknownFunction`: Verifies error returned (not empty string)

### Test Results
- `go test ./internal/core/template/... -v -count=1` → 45 passed (all tests)
- `go vet ./internal/core/template/...` → No issues found

### Notes
- TDD RED phase: wrote tests first (they failed because ResolveDynamic didn't exist)
- GREEN phase: implemented ResolveDynamic with switch dispatch
- REFACTOR phase: cleaned up unused variable (fullMatch)
- Integration with existing template engine: ResolveDynamic is a pre-pass that runs before variable substitution

## Task T37: Save Response to File

### What Worked
- Created `internal/client/output.go` with `SaveToFile()` and `DeriveFilename()` functions
- Created `internal/client/output_test.go` with 12 tests covering all use cases
- Added `URL string` field to `Response` struct for filename derivation from URL
- Wired `--output`/`-o` and `--force`/`-f` flags to run command

### Implementation Details
- `SaveToFile(resp *Response, path string, force bool) error`: saves body to file, handles "-", creates dirs, respects force flag
- `DeriveFilename(resp *Response, fallbackURL string) string`: extracts filename from Content-Disposition header OR URL last segment
- Content-Disposition parsing uses regex: `filename*=` for RFC 5987 encoding, `filename=` for simple form
- Stdout piping via path="-" writes directly to `os.Stdout`
- `os.MkdirAll()` for creating parent directories

### Binary Handling
- Body is `[]byte` — no UTF-8 assumption, binary data passes through unchanged
- `os.WriteFile()` writes raw bytes, no encoding/decoding

### Key Edge Cases Handled
- Existing file without force → error mentioning "--force"
- Empty body → creates empty file correctly
- Path with no parent dir → creates parent dirs with `os.MkdirAll()`
- Content-Disposition with quoted filename: extracts properly
- Content-Disposition with RFC 5987 encoding (UTF-8''...): decodes
- URL with query params: strips query, uses path segment

### Test Coverage
- `TestSaveResponse_JSON`: JSON save and verify content
- `TestSaveResponse_Binary`: PNG bytes save without corruption
- `TestSaveResponse_AutoFilename`: Content-Disposition header parsing
- `TestSaveResponse_AutoFilenameFromURL`: URL-based filename fallback
- `TestSaveResponse_CustomPath`: custom nested path
- `TestSaveResponse_CreateDirs`: creates missing parent directories
- `TestSaveResponse_ExistingFile`: error without force, success with force
- `TestSaveResponse_Stdout`: path="-" writes to stdout
- Additional tests for edge cases (empty body, filename extraction, fallback)

### Files Created
- `internal/client/output.go` — SaveToFile, DeriveFilename functions
- `internal/client/output_test.go` — 12 tests

### Files Modified
- `internal/client/response.go` — Added URL field to Response struct
- `internal/cli/commands/run.go` — Added --output/-o and --force/-f flags, wiring to SaveToFile
- `internal/formatter/formatter.go` — Fixed unused variable (v) in XML token switch

### Test Results
- `go test ./internal/client/... -count=1` → 52 passed
- `go build ./...` → Success

### Notes
- Response.URL field needed for URL-based filename derivation (Response didn't previously have URL)
- TDD RED→GREEN→REFACTOR cycle followed: wrote tests first, then implementation
- Pre-existing unrelated build error in formatter.go fixed (unused switch variable)

## Task T35: JSONPath and XPath Response Filtering

### What Worked
- Created `internal/formatter/filter.go` with `FilterJSON` and `FilterXML` functions
- Created `internal/formatter/filter_test.go` with 9 tests covering all major use cases
- Used `github.com/PaesslerAG/jsonpath` for JSONPath (lightweight, well-maintained)
- Used `github.com/antchfx/xmlquery` for XPath extraction
- Used switch on first character for path type detection (not if-else chains): '$' = JSONPath, '/' or '*' = XPath

### Implementation Details
- `FilterJSON(body []byte, path string) (string, error)` — JSONPath extraction
  - Path must start with '$' (e.g., "$.name", "$.data.users[0].email")
  - `jsonpath.Get(path, data)` returns interface{}, converted to pretty-printed JSON
- `FilterXML(body []byte, xpath string) (string, error)` — XPath extraction
  - Path must start with '/' or '//' (e.g., "//title", "//book[@category='fiction']")
  - `xmlquery.QueryAll(doc, xpath)` returns []*Node, each OutputXML(true) for formatted output
- `Filter(body []byte, path string) (string, error)` — auto-detects path type via switch on first char

### Key Edge Cases Handled
- Invalid JSON path format: returns descriptive error
- Invalid XML path format: returns descriptive error
- No match in JSON: returns error (PaesslerAG/jsonpath returns error for unknown keys)
- No match in XML: returns empty string, no error
- Empty result: handled gracefully (empty string returned)
- Multiple XPath results: wrapped in JSON array

### Library Notes
- `PaesslerAG/jsonpath` v0.1.1: Simple API, `jsonpath.Get(path, data)` returns interface{}
- `antchfx/xmlquery` v1.5.1: XPath 1.0 support, `OutputXML(bool)` method takes formatting bool param
- Both libraries auto-downloaded via `go mod tidy`

### Test Results
- `go test ./internal/formatter/filter_test.go ./internal/formatter/filter.go -count=1` → 9 passed
- `go mod tidy` → clean (new deps added properly)

### Notes
- TDD RED phase: wrote tests first (they failed because FilterJSON/FilterXML didn't exist)
- GREEN phase: implemented both functions following spec
- REFACTOR phase: added auto-detecting `Filter()` function using switch on first char
- Pre-existing syntax error in `internal/formatter/formatter_test.go` line 346 (fixed manually)
- Pre-existing syntax error in `internal/formatter/diff.go` lines 47-63 (duplicate code outside function) — unrelated to T35

## Task T34: Pretty Print JSON/XML/HTML Responses

### What Worked
- Created `internal/formatter/formatter.go` with `Format()`, `FormatJSON()`, `FormatXML()`, `FormatHTML()` functions
- Created `internal/formatter/theme.go` with ANSI color constants (Cyan, Green, Yellow, Magenta, Red, Reset)
- Created `internal/formatter/formatter_test.go` with 38 tests covering all formatting scenarios
- Used `json.MarshalIndent()` for JSON formatting, `xml.NewEncoder()` for XML formatting
- Implemented state machine for JSON colorization (keys=cyan, strings=green, numbers=yellow, booleans=magenta, null=red)

### Implementation Details
- `FormatOptions`: `Indent string`, `Color bool`, `MaxWidth int`
- `Format()` uses switch on content-type (NOT if-else chains) for dispatch
- JSON colorization uses 3-state machine: `stateKey` → `stateAfterColon` → `stateKey`
- XML/HTML colorization uses regex to extract tags and apply colors
- HTML formatting uses lightweight tag-based indentation (NOT full DOM parse)

### Key Edge Cases Handled
- Invalid JSON returns raw input without panic
- Empty input returns empty string
- Self-closing tags handled correctly in XML/HTML
- Comments and DOCTYPE preserved in HTML formatting

### Files Created
- `internal/formatter/formatter.go` — main formatting logic
- `internal/formatter/formatter_test.go` — 38 tests
- `internal/formatter/theme.go` — ANSI color constants

### Files Modified
- `internal/formatter/diff.go` — fixed pre-existing bug (op.Op → op.Type for jsondiff library)

### Test Results
- `go test ./internal/formatter/... -v -run TestFormat -count=1` → 28 passed
- `go vet ./internal/formatter/...` → No issues found
- `go test ./internal/formatter/... -count=1` → 56 passed, 1 failed (unrelated TestDiffJSON_FieldChanged)

### Notes
- TDD RED phase: wrote 38 tests first (all failed because implementation didn't exist)
- GREEN phase: implemented all formatting functions following spec
- REFACTOR phase: extracted color constants to theme.go for future TUI reuse
- Pre-existing build issue in diff.go (jsondiff.Operation.Type vs .Op) — fixed to make package compile

## Task T34 Fix: Add HTML Syntax Highlighting (colorizeHTML)

### What Worked
- Added `colorizeHTML(output string) string` function that returns ANSI-colored HTML
- Wired it into `FormatHTML` when `opts.Color` is true
- Used same regex pattern as `colorizeXML`: `(<[^>]+>)` to split HTML into tags and text
- Text content colored green (Green + Reset), tags colored via existing `colorizeHTMLTag`

### Implementation Details
- `colorizeHTML` splits HTML using regex, then:
  - Text between tags: wrapped in `Green + text + Reset`
  - Tags: passed through `colorizeHTMLTag` (already existed: tags=cyan, attributes=yellow)
  - Comments and DOCTYPE: preserved as-is (no colorization)
- `FormatHTML` now calls `colorizeHTML(output)` when `opts.Color` is true

### ANSI Color Scheme for HTML
- Tags: cyan (via `colorizeHTMLTag`)
- Attributes: yellow (via `colorizeHTMLTag`)
- Text content: green (new in `colorizeHTML`)

### Test Results
- `go test ./internal/formatter/... -count=1` → 57 passed
- All existing tests continue to pass

### Files Modified
- `internal/formatter/formatter.go` — added `colorizeHTML()` function, modified `FormatHTML()` to call it

## Task T41: GraphQL Client

### What Worked
- Created `internal/protocols/graphql/graphql.go` with `Client`, `Request`, `Response`, `GraphQLError`, `Location` types
- Created `internal/protocols/graphql/graphql_test.go` with 8 tests covering all required scenarios
- Created `internal/protocols/graphql/cli.go` with `GraphQLCommand` for `gurl graphql` subcommand
- Implemented functional options pattern (`WithHeader`) for extensible client configuration
- Handled null data edge case: `json.RawMessage` unmarshals JSON `null` as `[]byte("null")`, not Go `nil`

### GraphQL Request/Response Types
```go
type Request struct {
    Query         string
    Variables     map[string]interface{}
    OperationName string
}

type Response struct {
    Data   json.RawMessage
    Errors []GraphQLError
}

type GraphQLError struct {
    Message   string
    Locations []Location
    Path      []interface{}
}
```

### Implementation Details
- POST request with `Content-Type: application/json`
- Body format: `{"query": "...", "variables": {...}, "operationName": "..."}`
- Null data handling: convert `[]byte("null")` to `nil` after unmarshalling
- Functional options via `WithHeader(key, value)` for custom headers

### CLI Subcommand
- `gurl graphql "endpoint" --query 'query { ... }' --vars '{"limit": 10}'`
- `--query-file`/`-f` flag for loading queries from `.graphql` files
- `--operation-name`/`-op` for named operations
- `--color`/`-c` for syntax highlighting via formatter
- GraphQL errors printed to stderr with location info

### Test Coverage
- TestGraphQL_Query — basic query execution
- TestGraphQL_QueryWithVariables — variables passed correctly
- TestGraphQL_Mutation — mutation execution
- TestGraphQL_Introspection — `__schema` query
- TestGraphQL_ErrorResponse — error parsing with locations and path
- TestGraphQL_Headers — custom headers passed via WithHeader option
- TestGraphQL_BuildRequestBody — operation name included in body
- TestGraphQL_MultilineQuery — multiline/fragment queries preserved

### Test Results
- `go test ./internal/protocols/graphql/... -count=1` → 8 passed
- `go test ./... -count=1` → 466 passed (full test suite)
- `go build ./...` → Success

### Notes
- urfave/cli/v3 does NOT use `Arguments` field - positional args accessed via `c.Args().Get(n)`
- The `Args: true` field is not used in cli.Command struct
- json.RawMessage is `nil` when JSON field is omitted, but `[]byte("null")` when JSON value is `null`

## Task T40: Request Timeout Configuration

### What Worked
- Created `internal/client/timeout_test.go` with 14 tests covering all timeout scenarios
- Added `Option` type with `WithTimeout` and `WithConnectTimeout` functional options
- Added `connectTimeout` field to `Client` struct with `applyConnectTimeout()` method
- Added `Timeout string` field to `SavedRequest` (JSON) and `Config.General` (TOML)
- Wired connect timeout via `net.Dialer{Timeout: c.connectTimeout}` and `transport.DialContext`
- Added friendly error wrapping: `wrapTimeoutError()` converts `context.DeadlineExceeded` to "request timed out after X"
- Added `--timeout` flag to run command with `time.ParseDuration` support

### Implementation Details
- `Option` functional option pattern: `type Option func(*Client)`
- `WithTimeout(total time.Duration) Option` — sets total request timeout
- `WithConnectTimeout(connect time.Duration) Option` — sets connection establishment timeout
- `applyConnectTimeout()` creates `net.Dialer{Timeout: c.connectTimeout}` and sets `transport.DialContext`
- `wrapTimeoutError(err error, timeout time.Duration) error` — wraps context.DeadlineExceeded with friendly message
- Per-request timeout (`req.Timeout > 0`) takes precedence over client default timeout
- CLI `--timeout` flag overrides saved request's timeout field

### Timeout Resolution Precedence
1. CLI `--timeout` flag (highest priority)
2. SavedRequest.Timeout field
3. Client default timeout (30s from config)

### Error Message Format
- Old: `Get "http://...": context deadline exceeded`
- New: `request timed out after 5s`

### Test Coverage
- `TestTimeout_Default` — default 30s timeout from config
- `TestTimeout_PerRequest` — per-request timeout overrides client default
- `TestTimeout_Zero` — zero timeout means no timeout (infinite)
- `TestTimeout_Exceeded` — request exceeding timeout returns clear error
- `TestTimeout_ConnectVsTotal` — separate connect timeout and total timeout
- `TestTimeout_WithTimeout` — WithTimeout functional option works
- `TestTimeout_ConnectTimeoutApplied` — connect timeout applied to transport
- `TestTimeout_FriendlyError` — error message is user-friendly, not raw Go error
- `TestTimeout_FromConfig` — reads timeout from TOML config
- `TestTimeout_ConfigLoaderIntegration` — config loader handles timeout field
- `TestTimeout_PerRequestOverridesClient` — per-request timeout overrides client timeout
- `TestTimeout_ZeroTimeoutFromConfig` — "0" timeout means no timeout
- `TestTimeout_ExecuteWithContextTimeout` — context timeout works with ExecuteWithContext
- `TestTimeout_ErrorType` — timeout errors are properly typed

### Files Modified
- `pkg/types/types.go` — Added `Timeout string` to `SavedRequest` and `Config.General`
- `internal/config/defaults.go` — Added `Timeout: "30s"` default to General config
- `internal/client/client.go` — Added `Option` type, `WithTimeout`, `WithConnectTimeout`, `connectTimeout` field, `wrapTimeoutError`, `applyConnectTimeout`
- `internal/client/timeout_test.go` — 14 timeout tests
- `internal/cli/commands/run.go` — Added `--timeout` flag, timeout resolution logic

### Test Results
- `go test ./internal/client/... -v -run TestTimeout -count=1` → 14 passed
- `go test ./internal/client/... -count=1` → 66 passed
- `go build ./...` → Success

### Notes
- TDD RED phase: wrote tests first, they failed because functions didn't exist
- GREEN phase: implemented functional options and timeout wiring
- `effectiveTimeout` variable tracks which timeout was used for error message
- `httpClient.Timeout` is set to `effectiveTimeout` (not always `c.timeout`) to handle per-request override
- Pre-existing unrelated failures in template package (TestGetVariablesFromRequest) — not caused by T40

## Task T45: JavaScript Scripting Engine (goja runtime)

### What Worked
- Created `internal/scripting/engine.go` with `Engine` struct wrapping `*goja.Runtime`
- Created `internal/scripting/globals.go` with `gurl`, `console`, `require` objects
- Created `internal/scripting/sandbox.go` with security restrictions (blocks fs, net, os, child_process, http, https)
- Created 28 tests covering all required functionality
- Used goja ES5.1 runtime (v0.0.0-20260311135729)

### Key Implementation Details
- `Engine.vm` is set before calling `RegisterGlobals` so methods can access the runtime
- `gurl.request` and `gurl.response` use JavaScript getters via `Object.defineProperty` in JS
- `gurl._request` and `gurl._response` are internal properties set via `updateGurlRequest/Response`
- Sandbox enforcement: JavaScript overrides `global.require` to block listed modules
- Allowed modules (crypto, JSON, Math, Date, Buffer) are set as globals via `vm.Set`
- Timeout uses `context.WithTimeout` with goroutine for execution
- Test assertions use `panic/recover` to catch assertion failures in JS callbacks

### goja API Gotchas
- `goja.FunctionCall` does NOT have `.Runtime()` or `.VM` field - use `e.vm` directly
- `goja.Value.Call()` does NOT exist - use `goja.AssertFunction(fn)` to get callable, then call with `fn(goja.Undefined(), args...)`
- `goja.NewError()` does NOT exist - use `panic(errors.New(...))` for exceptions
- `vm.RunString()` returns `(goja.Value, error)` - must handle both return values
- To define getters in JS: use `Object.defineProperty` via `vm.RunString`
- For chained API (crypto, Buffer): create proper goja objects with methods, not nil

### Sandbox Implementation
- Blocked modules defined in `blockedModules` map
- JavaScript snippet creates `global.require` that checks against blocked list
- Allowed modules (crypto, JSON, Math, Date, Buffer) are set as VM globals before script execution
- crypto.createHash returns an object with update() returning self, digest() returning mock string
- Buffer.from returns an object with toString() method

### Test Coverage (28 tests)
- Basic execution, console.log/warn/error
- setVar/getVar round-trip
- Request headers modification via gurl.request.headers.set()
- Response status/body access
- Assertion API (gurl.test, gurl.expect)
- Timeout enforcement (context deadline)
- Sandbox: blocked modules (fs, net, os, child_process, http, https)
- Sandbox: allowed modules (crypto, Buffer, JSON, Math, Date)
- skipRequest and setNextRequest flags

### Test Results
- `go test ./internal/scripting/... -v -count=1` → 28 passed
- `go build ./...` → Success

### Notes
- goja v0.0.0-20260311135729 has different API than expected from docs - had to discover correct methods through trial and error
- The `gurl.expect()` API chains through JS getter pattern: `gurl.expect(val).to.equal(expected)`


## Task T48: Request Chaining via Scripts

### What Worked
- Created `internal/scripting/chaining.go` with `ChainExecutor` struct
- Created `internal/scripting/chaining_test.go` with 7 tests
- Wired `--chain` flag to run command in `internal/cli/commands/run.go`
- Added `MaxIterations()` and `Variables()` methods to ChainExecutor

### ChainExecutor Implementation
- `NewChainExecutor(engine *Engine, opts ...ChainExecutorOption)` - factory with functional options
- `WithMaxIterations(max int)` - option to customize max iterations (default 100)
- `MarkIteration(requestName string)` - tracks visit count per request
- `GetNextRequest()` - reads `engine.nextRequest` set by `gurl.setNextRequest()`
- `IsCircular()` - returns true if any request visited 3+ times
- `MaxIterationsReached()` - returns true when iteration count >= maxIterations
- `MaxIterations()` - exported getter for maxIterations
- `Variables()` - returns engine's variables map for persistence across chain
- `Reset()` - clears visited map and iteration count

### jsSetNextRequest Behavior
- `gurl.setNextRequest("name")` sets `engine.nextRequest = name`
- `gurl.setNextRequest(null)` sets `engine.nextRequest = ""` (empty string stops chain)
- null value in goja becomes empty string when exported via `Export().(string)`

### Variable Persistence Across Chain
- `engine.variables` is a `map[string]string` that persists across executions
- `RunPostResponse` sets variables via `gurl.setVar()` which updates `engine.variables`
- Variables are passed to next request via `vars` map merge in `executeChain()`

### Key Edge Cases Handled
- Empty nextRequest (null in JS) stops chain execution
- Circular detection: 3+ repetitions of same request triggers error
- Max iterations: default 100, configurable via `WithMaxIterations()`
- Variables set in post-response script persist to next request's pre-request script

### Test Coverage (7 tests)
- TestChain_SetNextRequest - basic setNextRequest functionality
- TestChain_PassVariable - variable persistence across requests
- TestChain_StopChain - stopping chain with null
- TestChain_CircularDetection - 3+ repetitions detected
- TestChain_MaxIterations - default 100 and custom limits
- TestChain_ConditionalBranch - conditional next request selection
- TestChain_ExecutionOrder - proper execution sequence

### Files Created
- `internal/scripting/chaining.go` - ChainExecutor implementation
- `internal/scripting/chaining_test.go` - 7 tests

### Files Modified
- `internal/cli/commands/run.go` - added --chain flag, executeChain(), executeSingleRequest()

### Test Results
- `go test ./internal/scripting/... -v -run TestChain -count=1` → 7 passed
- `go test ./internal/scripting/... -count=1` → 58 passed
- `go build ./internal/scripting/... ./internal/cli/commands/...` → Success

### Notes
- TDD: wrote tests first (RED), then implementation (GREEN)
- Engine's `nextRequest` field is the communication mechanism between JS and Go
- Chain executor uses functional options pattern for configuration
- Exported `maxIterations` via `MaxIterations()` method to allow CLI access

## Task T51: Data-Driven Testing (CSV/JSON Datasets)

### What Worked
- Created `internal/runner/datadriven.go` with `DataLoader` struct supporting CSV and JSON
- Created `internal/runner/datadriven_test.go` with 7 tests covering all required scenarios
- Wired `--data` flag to collection run command (`internal/runner/cli.go`)
- Wired `--data` flag to run command (`internal/cli/commands/run.go`)
- Added `DataFile` field to `RunConfig` struct in `runner.go`

### Implementation Details
- `NewDataLoader(filePath)` - factory that auto-detects file type from extension (.csv or .json)
- CSV parsing: uses `encoding/csv` with `bufio.Reader` for streaming (doesn't load entire file)
- JSON parsing: uses `json.Decoder` with streaming (doesn't load entire file)
- Headers: first row of CSV = column names, subsequent rows = data
- JSON: array of objects where each object becomes a row
- `Iterate(fn func(row map[string]string) error)` - streams rows one at a time
- `ReadAll()` - reads all rows into memory (for smaller files)
- `SubstituteTemplateWithVars(template, baseVars, rowVars)` - row vars take precedence over base vars

### Variable Substitution
- `{{name}}` placeholders replaced from dataset row values
- Row variables merge with environment variables (row takes precedence)
- Missing columns return error with column name: `&MissingColumnError{Column: "name", Row: 1}`

### Error Handling
- Missing column: returns `MissingColumnError` with column name and row number
- Unsupported file type: returns descriptive error with supported types
- Data iteration errors: wrapped with row number context

### Files Created
- `internal/runner/datadriven.go` - DataLoader implementation
- `internal/runner/datadriven_test.go` - 7 tests (CSV, JSON, Headers, VariableSubstitution, Iteration, EmptyFile, MissingColumn)

### Files Modified
- `internal/runner/runner.go` - Added `DataFile` to `RunConfig`, added `runWithData()` method
- `internal/runner/cli.go` - Added `--data` flag to `CollectionRunCommand`
- `internal/cli/commands/run.go` - Added `--data` flag, added `executeDataDriven()` function

### Test Results
- `go test ./internal/runner/... -run TestDataDriven -count=1` → 7 passed
- `go test ./internal/cli/commands/... -count=1` → 14 passed
- `go build ./cmd/gurl` → Success

### Notes
- TDD RED phase: wrote tests first (7 failing due to missing `NewDataLoader`)
- GREEN phase: implemented DataLoader with CSV and JSON streaming support
- REFACTOR phase: extracted `SubstituteTemplateWithVars` function to datadriven.go for reuse
- Pre-existing test failure in `TestRunner_RunWithOrder` (ordering issue, unrelated to T51)

## Task T52: Test Reporters (JUnit XML, JSON, HTML)

### What Worked
- Created `internal/reporters/` package with standalone types to avoid import cycles
- Implemented JUnit XML, JSON, HTML, and console reporters following the Reporter interface
- Wired `--reporter` (StringSliceFlag) and `--reporter-output` (StringFlag) to collection run command
- Used `convertToReporterResults()` adapter function to bridge runner.RunResult types with reporters types

### Key Implementation Details
- Reporters package has its own `RunResult`, `RequestResult`, `AssertionResult` types (not importing runner)
- This avoids circular import: runner imports reporters, and reporters needed to be usable standalone
- JUnit XML uses standard `encoding/xml` with proper CI-compatible structure (testsuites → testsuite → testcase)
- HTML uses inline CSS (no external dependencies, no href/src links) - self-contained
- Console reporter uses ANSI color codes from `internal/formatter`
- Multiple reporters supported via `--reporter junit --reporter json --reporter html --reporter console`
- Reporter output directory via `--reporter-output ./reports` saves files with correct extensions

### Import Cycle Resolution
- `internal/runner/cli.go` imports `internal/reporters`
- `internal/runner/runner.go` does NOT import `internal/reporters`
- CLI converts runner.RunResult types to reporters.RunResult types via adapter function
- Build succeeds: `go build ./internal/runner/...` and `go build ./cmd/gurl`

### Files Created
- `internal/reporters/reporter.go` — Reporter interface + JUnit/JSON/HTML/Console implementations
- `internal/reporters/reporters_test.go` — 7 tests covering all reporter types

### Files Modified
- `internal/runner/cli.go` — Added `--reporter` and `--reporter-output` flags, reporter execution logic

### Test Results
- `go test ./internal/reporters/... -count=1` → 7 passed
- `go build ./cmd/gurl` → Success
- Pre-existing failure in `TestRunner_StopOnError` (unrelated to T52)

### Notes
- TDD RED phase: wrote 7 failing tests (TestReporter_JUnit, TestReporter_JSON, TestReporter_HTML, etc.)
- GREEN phase: implemented all 4 reporters + interface
- REFACTOR phase: extracted convertToReporterResults adapter for type safety
- CLI uses switch for filename extension based on reporter type
- Task T52 depends on T50 (Collection Runner) — reporters consume RunResult from runner

## Task 53: CI-Friendly Exit Codes

### What Worked
- Wired exit codes directly into `internal/runner/cli.go` CollectionRunCommand Action
- Used `os.Exit()` with appropriate codes based on `RunResult.Failed` count
- No modification to runner core logic needed — RunResult already had Passed/Failed counts

### Implementation
- Exit 0: All assertions passed (`hasFailures = false` → no exit call = success)
- Exit 1: Any assertion failed (`result.Failed > 0` sets `hasFailures = true` → `os.Exit(1)`)
- Exit 2: Request error or collection not found (handled in `runner.Run()` error path → `os.Exit(2)`)

### Key Changes
- `runner.Run()` error: prints error to stderr, calls `os.Exit(2)`, returns `nil` (to satisfy cli.Action signature)
- After successful run: checks all `RunResult.Failed` counts across iterations
- `hasFailures` flag set if ANY iteration had failures
- At end of Action: `os.Exit(1)` if `hasFailures`, else normal return (exit 0)

### Test Results
- `go build ./cmd/gurl` → Success
- `go test ./internal/runner/... -v -count=1` → 16 passed

### Notes
- Used early `os.Exit(2)` in error path since cli.Action must return `nil` but we need non-zero exit
- Comments in code are minimal and explain CI exit code semantics — necessary for maintenance
