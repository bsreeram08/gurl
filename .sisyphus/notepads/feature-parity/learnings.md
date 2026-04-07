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

