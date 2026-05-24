# Changelog

## [v0.4.0] - 2026-05-24

### Added

- First-class collections with scoped variables and encrypted collection secrets.
- Collection variable precedence: CLI variables override collection variables, which override environment variables.
- `gurl collection create`, `set-var`, `unset-var`, `show`, `delete`, `migrate`, `export`, `import`, and `unlock` workflows.
- Git-friendly project storage under `.gurl/collections/` and `.gurl/environments/`, with project discovery via parent-directory walk-up, `GURL_PROJECT_DIR`, or `--project-dir`.
- File-backed collections as one `collection.json` plus one JSON file per request for reviewable diffs.
- File-backed environment storage with encrypted secrets.
- Per-collection encryption keys for DB-backed and file-backed collections.
- Passphrase-protected collection export/import and passphrase-protected project migration.
- OS keychain caching for passphrase-protected file-backed collections, with local-key fallback when keychain storage is unavailable.
- Collection directory import and native top-level collection export/import.
- `.env` import into collection variables with `gurl collection import <name> --file <path>`.
- File watching for long-running file-backed collection runs and shell sessions.
- README guidance for AI agents using gurl safely without exposing credentials.

### Fixed

- `gurl save --collection` now guards against typo-created collections and preserves real storage lookup failures.
- `--persist` routes dirty variables back to their origin: environment variables to environments, collection variables to collections, and new variables to the active collection when present.
- Cross-collection chains refresh collection variable context on each request step.
- File-backed deletion removes migrated DB shadow records instead of allowing deleted requests to reappear.
- Locked collection handling now fails with actionable errors instead of silently consuming ciphertext.
- Encrypted-value detection now validates ciphertext payloads before skipping encryption.

## [v0.3.1] - 2026-05-23

### Fixed

- Digest auth handler now rejects unsupported algorithm values instead of silently falling back to MD5.
- OAuth 1.0a handler now propagates URL parse errors instead of producing incorrect signatures silently.
- Auth plugin dispatch now has panic recovery, matching the existing middleware plugin safety pattern.

### Added

- `Description()` method on all auth handlers, enabling `gurl auth list` to show one-line descriptions alongside type names.
- `ApplyAuth` dispatch method on plugin registry with panic recovery for third-party auth plugins.
- Tests for duplicate handler registration, `type=none` no-op, explicit-none inheritance prevention, and plugin panic recovery.

## [v0.3.0] - 2026-05-23

### Added

- `gurl auth list` and `gurl auth info <type>` now help you discover the built-in auth handlers and their parameters without leaving the terminal.
- `gurl save` and `gurl edit` can persist auth settings with `--auth` and repeated `--auth-param key=value` flags, so saved requests can run later without resupplying credentials.
- Built-in auth handlers now cover Basic, Bearer, API key, OAuth 1, OAuth 2 client credentials, AWS SigV4, Digest, and NTLM request signing flows.

### Changed

- Saved auth parameters are template-aware. Values like `{{token}}`, `{{AWS_SECRET_ACCESS_KEY}}`, or `{{client_secret}}` are substituted at execution time before auth is applied.
- The plugin system now has an `AuthPlugin` interface and auth registry support for handlers that apply credentials to outgoing requests.

## [v0.2.2] - 2026-05-22

### Fixed

- `gurl update` now falls back to GitHub's public latest-release redirect when release metadata is temporarily unavailable, so current installs can report `Already up to date!` instead of a transient HTTP error.

## [v0.2.1] - 2026-05-22

### Fixed

- `gurl update` now validates the latest release metadata before it downloads anything, so malformed release tags fail safely instead of producing broken asset URLs.
- The latest release includes the full macOS and Linux asset set for amd64 and arm64, with raw binaries, tarballs, and `SHA256SUMS`.

## [v0.2.0] - 2026-05-22

### Added

- **Request chaining and flow control**
  - `gurl run <name> --chain` now follows post-response `gurl.setNextRequest(...)` decisions through the same lifecycle used by collection runs.
  - Saved requests can use `run-if` expressions to skip work when a variable does not match the current flow state.
  - Extracted variables and script-set variables flow into later requests in the same chain or collection run.
- **Saved extraction and scripts**
  - `gurl save` and `gurl edit` can persist `--extract`, `--pre-script`, and `--post-script` metadata on a request.
  - `gurl edit` can add `--run-if` conditions and remove extraction rules.
- **Flow variable persistence**
  - `gurl run --persist` and `gurl collection run --persist` can write extracted and script-set variables back to the selected environment.
  - Input variables from CLI flags, data rows, and environments are not persisted unless extraction or a script changes them.
- **Collection dry runs**
  - `gurl collection run --dry-run` previews request order, variable sources, planned extractions, and unresolved placeholders without sending requests.
- **Assertion bail mode**
  - `gurl collection run --assert-bail` stops only on assertion failures, while normal `--bail` still stops on any request failure.

### Fixed

- Chained runs now execute pre-scripts, extraction, post-scripts, assertions, history writes, and persistence in the same order as collection runs.
- Chained `setNextRequest` routing is honored before assertion bail handling, so intentional cleanup or follow-up requests can still run.
- Dirty variables are preserved across terminal errors where persistence still needs to happen.

## [v0.1.23] - 2026-05-21

### Added
- **P0: Response Variable Extraction** (`--extract`)
  - New `Extract` type and `SavedRequest.Extracts` field
  - `internal/extract` package with support for `jsonpath:`, `header:`, `regex:`, and `jq:`
- **P2: Assertions on Extracted Values**
  - New `extract:varName` syntax in `--assert` (e.g. `--assert "extract:orderId != ''"`)
  - Works with all existing operators
- **Collection environment/secret foundation**
  - Extended `types.Collection` with `Variables` + `SecretKeys`

### Changed
- `assertions.Evaluator.Evaluate()` now accepts `extractedVars` map

### Docs
- Updated README.md and AGENT.md

Refs: API flow testing PRD

## [v0.1.22] - 2026-04-18

### Bug Fixes

- **storage schema version** (`internal/storage/db.go`) — Schema version now initialized on fresh DB; iterator in ListRequests no longer skips the first key
- **SSE stream body close race** (`internal/protocols/sse/sse.go`) — Removed `defer resp.Body.Close()` that killed the goroutine's body read; moved close into goroutine. Removed over-aggressive event ID deduplication. Fixed event dispatch for type-only events (SSE spec compliance)
- **SSE test channel races** (`internal/protocols/sse/sse_test.go`) — Fixed 3 tests with broken select patterns across events/errors channels
- **gRPC error messages** (`internal/protocols/grpc/grpc.go`) — Error messages now contain "descriptor source" substring expected by tests
- **gRPC test expectations** (`internal/protocols/grpc/grpc_test.go`) — Dead connection tests now correctly expect errors
- **Path validation on macOS** (`internal/cli/commands/pathutil.go`) — Resolves symlinks on both path and allowed directory before comparing (fixes /var → /private/var)
- **Importers auth encoding** (`internal/importers/bruno.go`, `postman.go`) — basicAuth returns plaintext per test contract instead of base64
- **Postman URL building** (`internal/importers/postman.go`) — Fixed extra space in urlToString path construction; fixed basic auth to use plaintext
- **OpenAPI body/schema extraction** (`internal/importers/openapi.go`) — extractBody now falls back to text/plain content types. schemaToExample uses raw values for string examples. Path parameters preserved as templates instead of substituted
- **Cookie jar init order** (`internal/cookies/jar.go`) — loadFromDB runs before cleanExpired so correct error message is returned
- **Cookie DeleteCookie iterator** (`internal/cookies/jar.go`) — Same Seek+Next iterator bug as storage — first cookie was skipped
- **GraphQL error handling** (`internal/protocols/graphql/graphql.go`) — Errors returned in response struct, not as Go error return
- **TLS insecure flag** (`internal/client/client.go`) — Removed GURL_TLS_INSECURE_OK env var gate that blocked legitimate Insecure: true config
- **Scripting sandbox** (`internal/scripting/sandbox.go`) — eval blocked via vm.Set() with Go panic; Function blocked via JS-level override for goja constructor compatibility

### Test Suite

- **1339/1339 tests passing** across 25 packages (was 1320 passed, 27 failed before this release)

## [v0.1.19] - 2026-04-11

### TUI Upgrade

- **Bubbletea v2** (`internal/tui/`) — Upgraded from Bubbletea v1 to v2 (charm.land/bubbletea/v2). The TUI now uses the Cursed Renderer for better performance and stability
- **Viewport API fix** — `viewport.New()` now uses functional options (`WithWidth`/`WithHeight`) per Bubbletea v2 API
- **Key handling fix** — `tea.KeyRunes` replaced with `msg.Text != ""` for printable character detection; `tea.KeyCtrlJ` replaced with string matching (`keyStr == "ctrl+j"`)
- **View model change** — All `View()` methods now return `tea.View` (struct) instead of `string`
- **Alt screen/mouse API** — `tea.WithAltScreen()` option removed; set `v.AltScreen = true` and `v.MouseMode` directly on the `tea.View` struct
- **SearchModal mutation bug** — Fixed `View()` method mutating live state (`sm.results` truncation moved to local `displayResults` variable)
- **Test updates** — All tests updated for Bubbletea v2 API (`KeyMsg` → `KeyPressMsg`, view content accessed via `.Content` field)

## [Unreleased] - Security Hardening

Full codebase audit fixing 21 critical issues and 30+ robustness improvements across 35 files.

### Security Fixes

- **Shell injection in code generation** (`internal/codegen/`) — All string values are now properly escaped for their target language (POSIX shell, Python, JavaScript, Go) before being interpolated into generated code
- **Shell injection in `show` command** (`internal/cli/commands/show.go`) — Curl format output now escapes header values and body
- **Path traversal in `export`** (`internal/cli/commands/export.go`) — Output paths are validated to reject `..` sequences and prevent writing outside intended directories
- **Path traversal in `import`** (`internal/cli/commands/import.go`) — Import paths are sanitized before reading
- **Predictable temp file in `update`** (`internal/cli/commands/update.go`) — Replaced `os.Create` with `os.CreateTemp` for unique temp files
- **Secret access in scripting** (`internal/scripting/globals.go`) — `jsGetVar` now denies access to secret variables by default; `AllowSecretAccess` flag controls access

### Concurrency Fixes

- **WebSocket I/O race** (`internal/protocols/websocket/ws.go`) — Added `ioMu sync.Mutex` wrapping all conn I/O operations (Send, Receive, Close) to prevent concurrent write panics
- **TUI data races** (`internal/tui/app.go`, `internal/tui/responseviewer.go`) — Added `sync.Mutex` to RunnerModal and StreamingViewer for concurrent result/message appends
- **Cookie jar concurrent modification** (`internal/cookies/jar.go`) — Changed from modify-during-iterate to collect-then-delete pattern

### Resource Leak Fixes

- **WebSocket timer leak** (`internal/protocols/websocket/ws.go`) — Replaced `time.After(backoff)` with `time.NewTimer` + proper `Stop()` cleanup on context cancellation
- **Scripting engine deadlock** (`internal/scripting/engine.go`) — Added secondary 5s timeout after `vm.Interrupt()` to prevent indefinite hangs on timeout

### Error Handling Fixes

- **Regex argument swap** (`internal/assertions/engine.go`) — Fixed reversed arguments in `regexp.MatchString` calls
- **URL parse error** (`internal/auth/awsv4.go`) — Early return on `url.Parse` failure instead of nil dereference
- **Template substitution error** (`internal/runner/runner.go`) — Error from `template.Substitute` on body now propagated instead of discarded
- **Duration parse error** (`internal/runner/runner.go`) — `time.ParseDuration` error now checked before using the result
- **Context propagation** (`internal/runner/runner.go`) — `client.Execute` replaced with `ExecuteWithContext(ctx, ...)` so cancellations reach the HTTP layer
- **SSE blocking send** (`internal/protocols/sse/sse.go`) — Error channel send now uses non-blocking select/default pattern

### Correctness Fixes

- **SOCKS5 proxy auth** (`internal/client/proxy.go`) — Proxy URL credentials now extracted and passed to SOCKS5 dialer (previously always `nil`)
- **WebSocket stale connection** (`internal/protocols/websocket/ws.go`) — Local `conn` variable updated after reconnect in `ReceiveMultiple`
- **OAuth2 HTTP client** (`internal/auth/oauth2.go`) — Replaced bare `http.Post` with `&http.Client{Timeout: 30s}`
- **Formatter backslash counting** (`internal/formatter/formatter.go`) — Fixed off-by-one in escaped quote detection during JSON colorization
- **API key unknown `in` value** (`internal/auth/apikey.go`) — Added default case for unknown credential locations
- **Mock crypto** (`internal/scripting/sandbox.go`) — `crypto.createHash().digest()` now panics instead of returning fake "mockdigest" value
- **Plugin type validation** (`internal/plugins/loader.go`) — Loaded plugins are validated to implement at least one plugin interface

### Robustness Improvements

- **Package-level regex compilation** (`internal/auth/digest.go`, `internal/core/curl/detector.go`) — `regexp.MustCompile` calls moved to package-level vars to avoid recompilation on every call
- **Deterministic template substitution** (`internal/runner/datadriven.go`) — Map iteration in `substituteTemplate` now sorted by key for consistent output
- **Environment lookup warning** (`internal/runner/runner.go`) — Logs warning when referenced environment variable is not found
- **String escaping in all generators** (`internal/codegen/*.go`) — Python, JavaScript, and Go generators now escape special characters in interpolated values
- **URL validation in scripting** (`internal/scripting/globals.go`) — `jsSetRequestURL` validates URL scheme is http/https
- **Variables() returns copy** (`internal/scripting/chaining.go`) — Prevents external mutation of internal state
- **Stack traces in plugin recovery** (`internal/plugins/registry.go`) — Panic recovery now includes `debug.Stack()` for debugging
- **WebSocket signal buffer** (`internal/protocols/websocket/interactive.go`) — Signal channel buffer increased from 1 to 2
- **TLS cert error propagation** (`internal/protocols/websocket/cli.go`) — `NewDialerWithTLS` now returns TLS errors instead of silently failing
- **Blocked array JSON encoding** (`internal/scripting/sandbox.go`) — Replaced manual string concatenation with `json.Marshal`
- **Dead code removal** (`internal/cli/commands/env.go`) — Removed unused `isSecretVariable` function
- **Custom contains replaced** (`internal/cli/commands/timeline.go`) — Replaced hand-rolled `contains()` with `strings.Contains`
