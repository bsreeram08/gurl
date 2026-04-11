# Changelog

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
