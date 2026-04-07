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
