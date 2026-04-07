# Gurl: Feature Parity with Insomnia/Bruno/Postman/Yaak

## TL;DR

> **Quick Summary**: Bring Gurl CLI to feature parity with the 4 major API clients by fixing broken foundations (curl parser, save command, response storage), migrating from curl shelling to Go's `net/http`, then layering environments, auth (all types), protocol support (GraphQL/gRPC/WebSocket/SSE), JS scripting, assertions, collection runner, TUI (bubbletea), and plugin system.
> 
> **Deliverables**:
> - Fixed curl parser (shell tokenization, all edge cases)
> - Unified HTTP client (`internal/client`) using `net/http` — no more shelling to curl
> - Full environment system (global/collection/folder scoping, secrets, .env files)
> - All auth types: Basic, Bearer, API Key, OAuth 1/2, AWS Sig v4, Digest, NTLM
> - GraphQL, gRPC, WebSocket, SSE protocol handlers
> - JavaScript pre/post-request scripting (goja runtime)
> - Assertion engine + collection runner with CI exit codes + reporters
> - Cookie jar, proxy config, mTLS, redirect handling
> - Pretty printing, JSONPath filtering, response body diff
> - Interactive TUI (bubbletea) for request building + response viewing
> - Plugin system for extensibility
> - Naming fix: scurl → gurl throughout
> 
> **Estimated Effort**: XL (50+ tasks across 12 phases)
> **Parallel Execution**: YES — up to 7 tasks per wave
> **Critical Path**: Phase 0 (rename) → Phase 1 (client+parser) → Phase 2 (storage+response) → Phase 3 (environments) → Phase 4 (auth) → Phase 5 (cookies/proxy/TLS) → Phase 7 (protocols) → Phase 8 (scripting) → Phase 9 (assertions/runner) → Phase 10 (TUI) → Phase 11 (plugins)

---

## Context

### Original Request
User wants Gurl CLI to achieve feature parity with Insomnia, Bruno, Postman, and Yaak — the 4 major API client tools.

### Interview Summary
**Key Discussions**:
- Priority: Foundations first (fix curl parser, save command, response storage before new features)
- Scripting: JavaScript via goja runtime (like competitors)
- Protocols: ALL — GraphQL, gRPC, WebSocket, SSE
- TUI: In parallel with CLI features (bubbletea)
- Auth: All types — Basic, Bearer, OAuth 1/2, API Key, AWS Sig v4, Digest, NTLM
- Testing: TDD (red-green-refactor)

**Research Findings**:
- Gap analysis across 4 competitors identified 50+ missing features
- All 4 competitors use `net/http` equivalent (not curl shelling) — critical architecture gap
- Gurl's unique advantages: TUI (no competitor has CLI TUI), self-update, shell completions, imports from ALL 4 formats
- Two storage packages exist; `internal/core/storage/` is dead code (InMemoryDB stub, incompatible interface)
- Curl parser uses regex — fundamentally can't handle shell quoting; needs full rewrite
- `run.go` and `executor.go` BOTH shell out to system `curl` — must migrate to `net/http`

### Metis Review
**Identified Gaps** (addressed):
- **curl→net/http migration is MANDATORY**: Both `run.go` and `executor.go` shell out to curl. Adding auth, cookies, TLS, proxy, response capture is impossible without switching to `net/http`. → Phase 1 is the migration.
- **`ParsedCurl` ↔ `SavedRequest` type mismatch**: Parser outputs flat map headers, storage uses Header structs. No conversion exists. → Phase 1 includes conversion functions.
- **DB schema versioning needed**: Adding environments, auth, cookies changes the storage schema. No migration system exists. → Phase 2 adds schema versioning.
- **Delete `internal/core/storage/`**: Dead code with incompatible interface. Only `internal/storage/` (LevelDB) is used. → Phase 0 cleanup.
- **scurl→gurl rename must be Phase 0**: Mixing rename with feature work risks hard-to-debug issues. → Dedicated atomic commit.
- **goja ES5.1 limitation**: Scripts won't support `async/await`, `fetch()`, Promises natively. → Document as limitation, consider transpiler later.
- **Dependency budget concern**: gRPC adds ~15MB. → Accept for now, consider build tags for optional features later.
- **TUI must consume `internal/client`**: Don't let TUI build its own HTTP execution. → Phase 10 depends on Phase 1 client.

---

## Work Objectives

### Core Objective
Transform Gurl from a broken curl-wrapper skeleton into a fully-featured CLI API client matching the core capabilities of Insomnia, Bruno, Postman, and Yaak.

### Concrete Deliverables
- `internal/client/` — unified HTTP client package using `net/http`
- `internal/core/curl/parser.go` — rewritten parser using shell tokenization
- `internal/env/` — environment management (global, collection, folder scoping)
- `internal/auth/` — all auth type handlers
- `internal/client/graphql.go`, `grpc.go`, `websocket.go`, `sse.go` — protocol handlers
- `internal/scripting/` — JavaScript runtime via goja
- `internal/assertions/` — assertion engine
- `internal/runner/` — collection runner with reporters
- `internal/cookies/` — cookie jar management
- `internal/tui/` — bubbletea terminal UI
- `internal/plugins/` — plugin system
- Updated `save`, `run`, `edit`, `detect`, `diff`, `history`, `timeline` commands
- New commands: `env`, `auth`, `test`, `runner`

### Definition of Done
- [ ] `go test ./... -count=1` — ALL tests pass, zero failures
- [ ] `go build ./cmd/gurl` — compiles without errors
- [ ] `gurl save "test" --curl "curl -X POST -H 'Content-Type: application/json' -d '{"key":"value"}' https://httpbin.org/post"` — parses correctly
- [ ] `gurl run test` — executes via net/http, stores response body in history
- [ ] `gurl env create dev --var "BASE_URL=https://dev.api.com"` — creates environment
- [ ] `gurl run test --env dev` — uses environment variables
- [ ] `gurl run test --auth basic --user admin:pass` — authenticates
- [ ] `gurl diff test` — shows response body diff between last two runs
- [ ] All 4 competitor import formats still work

### Must Have
- net/http based execution (no curl shelling for core features)
- Response body stored in history
- Multiple environments with variable scoping
- At minimum: Basic, Bearer, API Key, OAuth 2.0 auth
- GraphQL requests
- Cookie jar management
- Pretty-printed JSON/XML output
- Collection runner with exit codes for CI
- TDD — every feature has tests first

### Must NOT Have (Guardrails)
- NO if-else-if-else chains (use switch/early-return/map patterns per AGENT.md)
- NO `interface{}` abstraction layers when concrete types work
- NO "utils" or "helpers" packages — code where it's used
- NO cloud sync, team collaboration, or account requirements
- NO GUI — CLI + TUI only
- NO mock servers or API design editor features
- NO monitors or scheduled runs
- NO telemetry or analytics
- NO configuration options for features that don't exist yet
- NO over-documentation — Go doc comments on exported types only
- NO mixing feature additions with refactoring in same commits
- NO adding dependencies without justification

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (`go test`, 134 passing tests)
- **Automated tests**: YES (TDD — red-green-refactor)
- **Framework**: `go test` (standard library) + `testing` package
- **If TDD**: Each task follows RED (failing test) → GREEN (minimal impl) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **CLI commands**: Use Bash — run gurl commands, assert exit codes + output
- **HTTP client**: Use Bash — run against `httpbin.org` or local `httptest` server
- **TUI**: Use interactive_bash (tmux) — launch TUI, send keystrokes, validate display
- **Tests**: Use Bash — `go test ./path/... -v -run TestName`

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — Phase 0+1, start immediately):
├── Task 1: scurl→gurl rename across all files [quick]
├── Task 2: Delete dead code (internal/core/storage/) [quick]
├── Task 3: Rewrite curl parser with shell tokenization [deep]
├── Task 4: Create internal/client HTTP client (net/http) [deep]
├── Task 5: Add ParsedCurl↔SavedRequest conversion [quick]
├── Task 6: Fix tags flag (StringFlag→StringSliceFlag) [quick]
├── Task 7: Fix --var flag (accept multiple values) [quick]

Wave 2 (Storage + Response — Phase 2, after Wave 1):
├── Task 8: Add DB schema versioning + migration framework [unspecified-high]
├── Task 9: Store response body in history [unspecified-high]
├── Task 10: Migrate run command to internal/client [deep]
├── Task 11: Migrate executor.go to internal/client [deep]
├── Task 12: Implement detect command (was stub) [quick]
├── Task 13: Fix export timestamp (was hardcoded) [quick]
├── Task 14: Fix save command — accept --curl, -X, -H, -d, stdin [unspecified-high]

Wave 3 (Environments + Auth foundations — Phase 3+4 start):
├── Task 15: Environment system (internal/env/) [deep]
├── Task 16: Environment CLI commands (env create/list/switch/delete) [unspecified-high]
├── Task 17: Wire environments into run command [unspecified-high]
├── Task 18: Variable scoping (folder→collection→global) [deep]
├── Task 19: Secret/encrypted variables [unspecified-high]
├── Task 20: .env file support [quick]
├── Task 21: Auth framework (internal/auth/) + Basic auth [unspecified-high]
├── Task 22: Bearer token auth [quick]
├── Task 23: API Key auth (header/query param) [quick]

Wave 4 (Auth continued + HTTP features — Phase 4+5):
├── Task 24: OAuth 2.0 (authorization code + client credentials) [deep]
├── Task 25: OAuth 1.0 [unspecified-high]
├── Task 26: AWS Signature v4 [unspecified-high]
├── Task 27: Digest auth [unspecified-high]
├── Task 28: NTLM auth [unspecified-high]
├── Task 29: Auth inheritance from folders [unspecified-high]
├── Task 30: Cookie jar management [unspecified-high]
├── Task 31: Redirect handling (follow/max config) [quick]
├── Task 32: Proxy configuration [unspecified-high]
├── Task 33: Client certificates (mTLS) + SSL toggle [unspecified-high]

Wave 5 (Response handling + Protocols start — Phase 6+7 start):
├── Task 34: Pretty print JSON/XML/HTML responses [unspecified-high]
├── Task 35: JSONPath/XPath response filtering [unspecified-high]
├── Task 36: Response diff (body comparison between runs) [unspecified-high]
├── Task 37: Save response to file [quick]
├── Task 38: Dynamic template values (UUID, timestamp, random) [unspecified-high]
├── Task 39: Path parameters support [quick]
├── Task 40: Request timeout configuration [quick]
├── Task 41: GraphQL client (internal/client/graphql.go) [deep]

Wave 6 (Protocols continued + Scripting — Phase 7+8):
├── Task 42: gRPC client (internal/client/grpc.go) [deep]
├── Task 43: WebSocket client (internal/client/websocket.go) [deep]
├── Task 44: SSE client (internal/client/sse.go) [unspecified-high]
├── Task 45: JavaScript scripting engine (internal/scripting/) [deep]
├── Task 46: Pre-request script execution [unspecified-high]
├── Task 47: Post-response script execution [unspecified-high]
├── Task 48: Request chaining via scripts [unspecified-high]

Wave 7 (Assertions + Runner + Edit — Phase 9):
├── Task 49: Assertion engine (internal/assertions/) [deep]
├── Task 50: Collection runner (internal/runner/) [deep]
├── Task 51: Data-driven testing (CSV/JSON iteration) [unspecified-high]
├── Task 52: Test reporters (JUnit, JSON, HTML) [unspecified-high]
├── Task 53: CI exit codes [quick]
├── Task 54: Edit command — interactive request editing [unspecified-high]
├── Task 55: Nested folders in collections [unspecified-high]
├── Task 56: Request sequencing [quick]

Wave 8 (TUI — Phase 10):
├── Task 57: TUI foundation — bubbletea setup + main layout [deep]
├── Task 58: TUI request list panel [visual-engineering]
├── Task 59: TUI request builder (method, URL, headers, body) [visual-engineering]
├── Task 60: TUI response viewer (pretty print, headers, timing) [visual-engineering]
├── Task 61: TUI environment switcher [visual-engineering]
├── Task 62: TUI keyboard shortcuts + help [visual-engineering]

Wave 9 (Plugins + Code Gen — Phase 11):
├── Task 63: Plugin system architecture (internal/plugins/) [deep]
├── Task 64: Template function plugins [unspecified-high]
├── Task 65: Auth plugins [unspecified-high]
├── Task 66: Multi-language code generation [unspecified-high]

Wave FINAL (Verification — after ALL tasks):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
├── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: T1→T4→T10→T15→T17→T21→T24→T30→T41→T45→T49→T50→T57→T63→F1-F4→user okay
Parallel Speedup: ~65% faster than sequential
Max Concurrent: 7 (Waves 1, 3, 4)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | — | ALL | 1 |
| 2 | — | 8 | 1 |
| 3 | 1 | 5, 12, 14 | 1 |
| 4 | 1 | 10, 11, 21, 30-33, 41-44 | 1 |
| 5 | 3, 4 | 10, 14 | 1 |
| 6 | 1 | — | 1 |
| 7 | 1 | — | 1 |
| 8 | 2 | 9, 15 | 2 |
| 9 | 4, 8 | 36 | 2 |
| 10 | 4, 5 | 17 | 2 |
| 11 | 4 | — | 2 |
| 12 | 3 | — | 2 |
| 13 | 1 | — | 2 |
| 14 | 3, 5 | — | 2 |
| 15 | 8 | 16, 17, 18, 19, 20, 45 | 3 |
| 16 | 15 | — | 3 |
| 17 | 10, 15 | — | 3 |
| 18 | 15 | 29 | 3 |
| 19 | 15 | — | 3 |
| 20 | 15 | — | 3 |
| 21 | 4, 15 | 22-29 | 3 |
| 22 | 21 | — | 3 |
| 23 | 21 | — | 3 |
| 24 | 21 | — | 4 |
| 25 | 21 | — | 4 |
| 26 | 21 | — | 4 |
| 27 | 21 | — | 4 |
| 28 | 21 | — | 4 |
| 29 | 18, 21 | — | 4 |
| 30 | 4 | — | 4 |
| 31 | 4 | — | 4 |
| 32 | 4 | — | 4 |
| 33 | 4 | — | 4 |
| 34 | 9 | 60 | 5 |
| 35 | 9 | — | 5 |
| 36 | 9 | — | 5 |
| 37 | 9 | — | 5 |
| 38 | 15 | — | 5 |
| 39 | 4 | — | 5 |
| 40 | 4 | — | 5 |
| 41 | 4 | — | 5 |
| 42 | 4 | — | 6 |
| 43 | 4 | — | 6 |
| 44 | 4 | — | 6 |
| 45 | 15, 4 | 46, 47, 48 | 6 |
| 46 | 45 | — | 6 |
| 47 | 45 | — | 6 |
| 48 | 45 | — | 6 |
| 49 | 9, 15 | 50 | 7 |
| 50 | 49 | 51, 52, 53 | 7 |
| 51 | 50 | — | 7 |
| 52 | 50 | — | 7 |
| 53 | 50 | — | 7 |
| 54 | 14, 15 | — | 7 |
| 55 | 8 | — | 7 |
| 56 | 55 | — | 7 |
| 57 | 4, 15 | 58-62 | 8 |
| 58 | 57 | — | 8 |
| 59 | 57 | — | 8 |
| 60 | 34, 57 | — | 8 |
| 61 | 15, 57 | — | 8 |
| 62 | 57 | — | 8 |
| 63 | 4, 15 | 64, 65 | 9 |
| 64 | 63 | — | 9 |
| 65 | 63, 21 | — | 9 |
| 66 | 4 | — | 9 |

### Agent Dispatch Summary

- **Wave 1**: 7 tasks — T1,T2,T6,T7 → `quick`; T3,T4 → `deep`; T5 → `quick`
- **Wave 2**: 7 tasks — T8,T9,T14 → `unspecified-high`; T10,T11 → `deep`; T12,T13 → `quick`
- **Wave 3**: 9 tasks — T15,T18 → `deep`; T16,T17,T19 → `unspecified-high`; T20 → `quick`; T21 → `unspecified-high`; T22,T23 → `quick`
- **Wave 4**: 10 tasks — T24 → `deep`; T25-T28 → `unspecified-high`; T29-T33 → `unspecified-high`; T31 → `quick`
- **Wave 5**: 8 tasks — T34-T36,T38 → `unspecified-high`; T37,T39,T40 → `quick`; T41 → `deep`
- **Wave 6**: 7 tasks — T42,T43,T45 → `deep`; T44,T46-T48 → `unspecified-high`
- **Wave 7**: 8 tasks — T49,T50 → `deep`; T51,T52,T54,T55 → `unspecified-high`; T53,T56 → `quick`
- **Wave 8**: 6 tasks — T57 → `deep`; T58-T62 → `visual-engineering`
- **Wave 9**: 4 tasks — T63 → `deep`; T64-T66 → `unspecified-high`
- **FINAL**: 4 tasks — F1 → `oracle`; F2 → `unspecified-high`; F3 → `unspecified-high`; F4 → `deep`

---

## TODOs

- [x] 1. Rename scurl→gurl across all files, paths, configs, env vars

  **What to do**:
  - RED: Write a test that asserts DB path defaults to `~/.local/share/gurl/gurl.db` (not scurl)
  - GREEN: Use `ast_grep_search` and `grep` to find ALL occurrences of "scurl" across the codebase
  - Replace all: DB path `~/.local/share/scurl/scurl.db` → `~/.local/share/gurl/gurl.db`
  - Env var `SCURL_DB_PATH` → `GURL_DB_PATH`
  - Config file `.scurlrc` → `.gurlrc`
  - All string literals, comments, and docs referencing "scurl"
  - REFACTOR: Verify `go test ./... -count=1` passes with renamed paths

  **Must NOT do**:
  - Change any behavior — only rename strings
  - Use if-else chains for path detection

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`cli-developer`]
    - `cli-developer`: CLI path conventions and naming

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 2)
  - **Parallel Group**: Wave 1
  - **Blocks**: ALL subsequent tasks
  - **Blocked By**: None

  **References**:
  - `internal/storage/db.go` — Contains hardcoded `scurl` DB path
  - `internal/cli/commands/detect.go` — References "scurl" in strings
  - `PRD.md` — Title says "scurl"
  - `cmd/gurl/main.go` — App name registration

  **Acceptance Criteria**:
  - [ ] `grep -r "scurl" --include="*.go" .` returns zero matches
  - [ ] `go test ./... -count=1` — ALL PASS

  **QA Scenarios**:
  ```
  Scenario: No scurl references remain in Go source
    Tool: Bash (grep)
    Preconditions: All renames applied
    Steps:
      1. Run: grep -rn "scurl" --include="*.go" .
      2. Assert: exit code 1 (no matches)
    Expected Result: Zero occurrences of "scurl" in any .go file
    Failure Indicators: Any line containing "scurl" in grep output
    Evidence: .sisyphus/evidence/task-1-no-scurl-refs.txt

  Scenario: Build and tests pass after rename
    Tool: Bash
    Preconditions: All renames applied
    Steps:
      1. Run: go build ./cmd/gurl
      2. Assert: exit code 0
      3. Run: go test ./... -count=1
      4. Assert: exit code 0, "FAIL" not in output
    Expected Result: Clean build and all existing tests pass
    Evidence: .sisyphus/evidence/task-1-build-tests.txt
  ```

  **Commit**: YES
  - Message: `rename: scurl→gurl across all paths, configs, env vars, docs`
  - Files: ALL files with scurl references
  - Pre-commit: `go test ./... -count=1`

- [x] 2. Delete dead code: internal/core/storage/

  **What to do**:
  - Verify `internal/core/storage/` is not imported by any production code (only tests if any)
  - Delete the entire `internal/core/storage/` directory
  - If any tests depend on it, migrate them to use the real `internal/storage` package with test helpers
  - Verify build and tests still pass

  **Must NOT do**:
  - Delete `internal/storage/` (the real LevelDB storage)
  - Break any existing tests

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 1)
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 8 (schema versioning)
  - **Blocked By**: None

  **References**:
  - `internal/core/storage/db.go` — InMemoryDB stub (134 lines) with incompatible interface
  - `internal/storage/db.go` — Real LevelDB storage (463 lines) used by all commands
  - `internal/cli/commands/mock_db_test.go` — Test mocks (may reference core/storage)

  **Acceptance Criteria**:
  - [ ] `internal/core/storage/` directory does not exist
  - [ ] `go build ./cmd/gurl` succeeds
  - [ ] `go test ./... -count=1` — ALL existing tests pass

  **QA Scenarios**:
  ```
  Scenario: Dead code removed, build passes
    Tool: Bash
    Preconditions: Directory deleted
    Steps:
      1. Run: ls internal/core/storage/ 2>&1
      2. Assert: "No such file or directory" in output
      3. Run: go build ./cmd/gurl
      4. Assert: exit code 0
      5. Run: go test ./... -count=1
      6. Assert: exit code 0
    Expected Result: Directory gone, project builds and tests pass
    Evidence: .sisyphus/evidence/task-2-dead-code-removed.txt
  ```

  **Commit**: YES
  - Message: `cleanup: remove dead internal/core/storage — incompatible InMemoryDB stub`
  - Files: `internal/core/storage/`
  - Pre-commit: `go test ./... -count=1`

- [x] 3. Rewrite curl parser using shell tokenization

  **What to do**:
  - RED: Keep all 18 existing test cases. Add 20+ new edge cases: shell quoting (`$'...'`, double quotes with escapes), `--data-raw`, `--data-urlencode`, `-F` multipart, `@` file references, `-X POST`, unquoted headers, URLs with query params and fragments, multiline bodies, `--compressed`, `--location`, `-k`/`--insecure`, `-L`, `--max-redirs`, `--connect-timeout`, `--cookie`, `--cookie-jar`, `-u` user auth
  - GREEN: Replace regex-based parsing with shell tokenization approach:
    1. Tokenize using `github.com/kballard/go-shellquote` or equivalent Go shell lexer
    2. Parse tokens using a flag-based state machine (switch statement, NO if-else chains)
    3. Map curl flags to `ParsedCurl` struct fields
  - REFACTOR: All 38+ tests pass, parser handles real-world curl commands

  **Must NOT do**:
  - Use regex for parsing (the whole point is to replace it)
  - Use if-else-if-else chains — use switch statements with explicit cases
  - Break existing passing tests

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`cli-developer`]
    - `cli-developer`: CLI argument parsing patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 4, different packages)
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 5, 12, 14
  - **Blocked By**: Task 1 (rename first)

  **References**:
  - `internal/core/curl/parser.go` — Current regex-based parser to REPLACE
  - `internal/core/curl/parser_test.go` — 18 existing tests (10 failing) — keep ALL, add more
  - `pkg/types/types.go:ParsedCurl` — Output struct the parser must populate
  - `github.com/kballard/go-shellquote` — Recommended tokenization library

  **Acceptance Criteria**:
  - [ ] `go test ./internal/core/curl/... -v -count=1` — ALL 38+ tests pass, 0 failures
  - [ ] Parser correctly handles: `-X POST -H 'Content-Type: application/json' -d '{"key":"value"}' https://example.com`
  - [ ] Parser correctly handles: shell quoting with escaped characters
  - [ ] No regex in parser.go (grep -c "regexp" parser.go returns 0)

  **QA Scenarios**:
  ```
  Scenario: Complex curl command parses correctly
    Tool: Bash (go test)
    Preconditions: Parser rewritten
    Steps:
      1. Run: go test ./internal/core/curl/... -v -count=1 -run TestParseCurl
      2. Assert: exit code 0
      3. Assert: output contains "PASS"
      4. Assert: output does NOT contain "FAIL"
    Expected Result: All 38+ test cases pass including edge cases
    Evidence: .sisyphus/evidence/task-3-parser-tests.txt

  Scenario: No regex remains in parser
    Tool: Bash (grep)
    Preconditions: Parser rewritten
    Steps:
      1. Run: grep -c "regexp" internal/core/curl/parser.go
      2. Assert: output is "0"
    Expected Result: Zero regex imports/usage in parser
    Evidence: .sisyphus/evidence/task-3-no-regex.txt
  ```

  **Commit**: YES (multiple TDD commits)
  - Message: `parser: rewrite curl parser using shell tokenization — regex can't handle quoting`
  - Files: `internal/core/curl/parser.go`, `internal/core/curl/parser_test.go`
  - Pre-commit: `go test ./internal/core/curl/... -v -count=1`

- [x] 4. Create unified HTTP client (internal/client/) using net/http

  **What to do**:
  - RED: Write tests for HTTP client: GET, POST with body, custom headers, status code capture, response body capture, response time tracking, error handling
  - GREEN: Create `internal/client/` package with:
    - `client.go` — `Client` struct wrapping `*http.Client` with configurable transport
    - `request.go` — `Request` struct with method, URL, headers, body, auth, timeout
    - `response.go` — `Response` struct with status, headers, body bytes, duration, size
    - `execute.go` — `Execute(Request) (Response, error)` using `net/http`
  - REFACTOR: Clean interfaces, proper error types, zero curl dependency

  **Must NOT do**:
  - Shell out to curl
  - Use `interface{}` — use concrete types
  - Create a "utils" package
  - Add auth/proxy/TLS features yet — bare HTTP client only

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Idiomatic Go HTTP client patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 3, different packages)
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 10, 11, 21, 30-33, 41-44 (everything that makes HTTP calls)
  - **Blocked By**: Task 1 (rename first)

  **References**:
  - `internal/core/curl/executor.go` — Current curl execution (shells out) — to be REPLACED
  - `internal/cli/commands/run.go` — Current run command (shells out) — will migrate in Task 10
  - `pkg/types/types.go:SavedRequest` — Request data model client must accept
  - `pkg/types/types.go:ExecutionHistory` — History model client must populate
  - `net/http` standard library — Foundation for the client

  **Acceptance Criteria**:
  - [ ] `go test ./internal/client/... -v -count=1` — ALL tests pass
  - [ ] Client executes GET request to httptest server, captures status+body+headers+duration
  - [ ] Client executes POST with JSON body, captures response
  - [ ] Client respects timeout setting
  - [ ] No `exec.Command("curl"` in the new client package

  **QA Scenarios**:
  ```
  Scenario: HTTP client GET request works
    Tool: Bash (go test)
    Preconditions: Client package created
    Steps:
      1. Run: go test ./internal/client/... -v -count=1 -run TestExecuteGET
      2. Assert: exit code 0, output contains "PASS"
    Expected Result: GET request to httptest server returns correct status, body, headers
    Evidence: .sisyphus/evidence/task-4-client-get.txt

  Scenario: HTTP client POST with body works
    Tool: Bash (go test)
    Preconditions: Client package created
    Steps:
      1. Run: go test ./internal/client/... -v -count=1 -run TestExecutePOST
      2. Assert: exit code 0, output contains "PASS"
    Expected Result: POST with JSON body sends correctly, captures response body
    Evidence: .sisyphus/evidence/task-4-client-post.txt
  ```

  **Commit**: YES (TDD commits)
  - Message: `client: add internal/client with net/http executor — replace curl shelling`
  - Files: `internal/client/client.go`, `internal/client/request.go`, `internal/client/response.go`, `internal/client/execute.go`, `internal/client/client_test.go`
  - Pre-commit: `go test ./internal/client/... -v -count=1`

- [x] 5. Add ParsedCurl↔SavedRequest conversion

  **What to do**:
  - RED: Write tests for conversion: ParsedCurl with method, headers (map), body, URL → SavedRequest with Method, Headers ([]Header), Body, URL; and reverse
  - GREEN: Create conversion functions in `internal/core/curl/` or `pkg/types/`:
    - `ParsedCurlToSavedRequest(parsed ParsedCurl) SavedRequest`
    - `SavedRequestToParsedCurl(req SavedRequest) ParsedCurl`
  - REFACTOR: Handle edge cases (nil maps, empty fields)

  **Must NOT do**:
  - Modify the ParsedCurl or SavedRequest types (yet)
  - Use if-else chains for field mapping

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on Tasks 3+4)
  - **Parallel Group**: Wave 1 (end)
  - **Blocks**: Tasks 10, 14
  - **Blocked By**: Tasks 3 (parser types), 4 (client types)

  **References**:
  - `pkg/types/types.go:ParsedCurl` — Source type (flat map headers)
  - `pkg/types/types.go:SavedRequest` — Target type ([]Header structs)
  - `pkg/types/types.go:Header` — Header struct definition

  **Acceptance Criteria**:
  - [ ] `go test ./... -run TestConversion -v -count=1` — ALL pass
  - [ ] Round-trip: SavedRequest → ParsedCurl → SavedRequest preserves all data

  **QA Scenarios**:
  ```
  Scenario: Conversion preserves all request data
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./... -run TestConversion -v -count=1
      2. Assert: exit code 0, "PASS" in output
    Expected Result: All conversion tests pass including round-trip
    Evidence: .sisyphus/evidence/task-5-conversion.txt
  ```

  **Commit**: YES
  - Message: `types: add ParsedCurl↔SavedRequest conversion — bridge parser and storage`
  - Pre-commit: `go test ./... -count=1`

- [x] 6. Fix tags flag: StringFlag→StringSliceFlag

  **What to do**:
  - RED: Write test that saves a request with multiple tags and verifies all are stored
  - GREEN: In `save.go` and any command using `--tag`, change `cli.StringFlag` to `cli.StringSliceFlag`
  - Update tag handling to accept multiple `--tag` values
  - REFACTOR: Verify existing tag functionality still works

  **Must NOT do**:
  - Change storage format for tags (already supports []string)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: None
  - **Blocked By**: Task 1

  **References**:
  - `internal/cli/commands/save.go` — Tag flag definition
  - `pkg/types/types.go:SavedRequest.Tags` — Already `[]string`

  **Acceptance Criteria**:
  - [ ] `gurl save test --tag api --tag auth https://example.com` stores both tags
  - [ ] `go test ./... -count=1` passes

  **QA Scenarios**:
  ```
  Scenario: Multiple tags accepted
    Tool: Bash
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestSaveMultipleTags -count=1
      2. Assert: exit code 0
    Expected Result: Request saved with both tags
    Evidence: .sisyphus/evidence/task-6-multi-tags.txt
  ```

  **Commit**: YES
  - Message: `save: fix tag flag to accept multiple values — was StringFlag, now StringSliceFlag`
  - Pre-commit: `go test ./... -count=1`

- [x] 7. Fix --var flag: accept multiple values

  **What to do**:
  - RED: Write test that runs a request with `--var KEY1=val1 --var KEY2=val2` and both are substituted
  - GREEN: In `run.go`, change `--var` from `cli.StringFlag` to `cli.StringSliceFlag`
  - Update template variable injection to handle multiple vars
  - REFACTOR: Verify template engine handles multiple variables

  **Must NOT do**:
  - Change the template engine itself (it already supports multiple vars)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: None
  - **Blocked By**: Task 1

  **References**:
  - `internal/cli/commands/run.go` — Var flag definition and usage
  - `internal/core/template/engine.go` — Template substitution (already supports multiple vars)

  **Acceptance Criteria**:
  - [ ] Multiple --var flags all substitute correctly
  - [ ] `go test ./... -count=1` passes

  **QA Scenarios**:
  ```
  Scenario: Multiple vars substituted
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestRunMultipleVars -count=1
      2. Assert: exit code 0
    Expected Result: Both variables substituted in request
    Evidence: .sisyphus/evidence/task-7-multi-vars.txt
  ```

  **Commit**: YES
  - Message: `run: fix var flag to accept multiple values — was StringFlag, now StringSliceFlag`
  - Pre-commit: `go test ./... -count=1`

- [x] 8. Add DB schema versioning + migration framework

  **What to do**:
  - RED: Write tests that verify: (1) a new DB gets schema version "1" written, (2) a migration function can bump version, (3) opening an old DB triggers migration
  - GREEN: Add to `internal/storage/`:
    - A `schema_version` key in LevelDB (stored as simple JSON: `{"version": 1}`)
    - `GetSchemaVersion() (int, error)` — reads current version, returns 0 if key missing (legacy DB)
    - `MigrateIfNeeded() error` — checks version, runs pending migrations in order
    - A `migrations` map: `map[int]func(db *leveldb.DB) error` — each migration bumps version after success
    - Version 1 migration: no-op (current schema is version 1)
  - REFACTOR: Clean up, ensure Open() calls MigrateIfNeeded() automatically

  **Must NOT do**:
  - Use if-else chains for migration dispatch — use a map or switch
  - Change existing data format — version 1 = current format
  - Add complex migration framework — keep it simple (map of version→func)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Idiomatic Go patterns for DB migration

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on Task 2)
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 9, 15
  - **Blocked By**: Task 2 (dead code removed first)

  **References**:
  - `internal/storage/db.go:16-27` — DB interface (add migration methods here)
  - `internal/storage/db.go:38-64` — LMDB struct + NewLMDB/Open (wire migration into Open)
  - `internal/storage/db.go:66-76` — Open() method (call MigrateIfNeeded after open)
  - `pkg/types/types.go` — Types that define current schema (SavedRequest, ExecutionHistory, Collection)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/storage/... -v -run TestSchema -count=1` — ALL pass
  - [ ] New DB gets version 1 written on first Open()
  - [ ] Legacy DB (no version key) migrates to version 1 on Open()
  - [ ] `go test ./... -count=1` — ALL existing tests pass

  **QA Scenarios**:
  ```
  Scenario: New DB gets schema version on first open
    Tool: Bash (go test)
    Preconditions: Fresh test DB
    Steps:
      1. Run: go test ./internal/storage/... -v -run TestSchemaVersionNewDB -count=1
      2. Assert: exit code 0, "PASS" in output
    Expected Result: GetSchemaVersion returns 1 after first Open
    Evidence: .sisyphus/evidence/task-8-schema-new-db.txt

  Scenario: Legacy DB migrates on open
    Tool: Bash (go test)
    Preconditions: DB without schema_version key
    Steps:
      1. Run: go test ./internal/storage/... -v -run TestSchemaVersionLegacyDB -count=1
      2. Assert: exit code 0, "PASS" in output
    Expected Result: Open() detects no version, runs migration, version becomes 1
    Evidence: .sisyphus/evidence/task-8-schema-legacy.txt
  ```

  **Commit**: YES (TDD commits)
  - Message: `storage: add schema versioning + migration framework — enables safe schema evolution`
  - Files: `internal/storage/db.go`, `internal/storage/migration.go`, `internal/storage/migration_test.go`
  - Pre-commit: `go test ./internal/storage/... -count=1`

- [x] 9. Store response body in execution history

  **What to do**:
  - RED: Write test that executes a request via `internal/client`, stores the full response (body, headers, status, duration, size) in ExecutionHistory, and retrieves it
  - GREEN: Update the execution flow to:
    - After `internal/client.Execute()` returns a Response, populate `ExecutionHistory.Response` with the full body
    - Populate `StatusCode`, `SizeBytes` from the Response
    - Ensure `db.SaveHistory()` persists the full response body
    - Ensure `db.GetHistory()` returns it
  - REFACTOR: Verify that `history` and `timeline` commands now display response bodies

  **Must NOT do**:
  - Change the ExecutionHistory struct fields (they already have Response, StatusCode, SizeBytes)
  - Truncate response body — store full body (future optimization can add limits)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go HTTP response handling patterns

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on Tasks 4, 8)
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 34, 35, 36
  - **Blocked By**: Tasks 4 (HTTP client), 8 (schema versioning)

  **References**:
  - `pkg/types/types.go:48-56` — ExecutionHistory struct (has Response string, StatusCode, SizeBytes fields — currently unpopulated by run.go)
  - `pkg/types/types.go:59-69` — NewExecutionHistory constructor (already accepts all fields)
  - `internal/cli/commands/run.go:78-84` — Current history recording (only sets RequestID, DurationMs, Timestamp — ignores Response/StatusCode/SizeBytes)
  - `internal/core/curl/executor.go:17-60` — ExecuteCurl function (captures response but run.go doesn't use it)
  - `internal/storage/db.go:25-26` — SaveHistory/GetHistory DB methods

  **Acceptance Criteria**:
  - [ ] `go test ./... -run TestResponseStorage -v -count=1` — ALL pass
  - [ ] After running a request, `db.GetHistory(reqID, 1)` returns entry with non-empty Response body
  - [ ] `history` command shows response body preview
  - [ ] `go test ./... -count=1` — ALL pass

  **QA Scenarios**:
  ```
  Scenario: Response body stored after execution
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./... -v -run TestResponseStorage -count=1
      2. Assert: exit code 0, "PASS" in output
    Expected Result: ExecutionHistory.Response contains the HTTP response body
    Evidence: .sisyphus/evidence/task-9-response-storage.txt

  Scenario: History shows response data
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./... -v -run TestHistoryWithResponse -count=1
      2. Assert: exit code 0
    Expected Result: Retrieved history entry has StatusCode > 0 and non-empty Response
    Evidence: .sisyphus/evidence/task-9-history-response.txt
  ```

  **Commit**: YES
  - Message: `history: store full response body in execution history — enables diff and filtering`
  - Files: `internal/cli/commands/run.go`, `internal/cli/commands/history.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 10. Migrate run command to internal/client

  **What to do**:
  - RED: Write integration test that uses `run` command action function → verifies it calls `internal/client.Execute()` instead of `exec.Command("curl")`
  - GREEN: Rewrite `run.go` Action to:
    1. Retrieve SavedRequest from DB
    2. Convert to `client.Request` using conversion functions (Task 5)
    3. Apply variable substitution via template engine
    4. Call `internal/client.Execute(request)` to get `Response`
    5. Create `ExecutionHistory` from `Response` (with full body, status, duration, size)
    6. Save history via `db.SaveHistory()`
    7. Print response to stdout (respecting --format flag)
  - Remove all `exec.Command("curl"...)` code from run.go
  - REFACTOR: Ensure `--var`, `--format`, `--cache` flags still work

  **Must NOT do**:
  - Shell out to curl (the whole point of this task)
  - Remove the `--var` or `--format` flags
  - Break the `--cache` flag (can leave as no-op for now)
  - Use if-else chains for format dispatch

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`golang-pro`, `cli-developer`]
    - `golang-pro`: Idiomatic Go HTTP client integration
    - `cli-developer`: CLI command pattern rewiring

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 11, different files)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 17 (wire environments into run)
  - **Blocked By**: Tasks 4 (client), 5 (conversion)

  **References**:
  - `internal/cli/commands/run.go:1-89` — FULL FILE to rewrite (currently shells to curl on line 68)
  - `internal/client/` — New HTTP client package (created in Task 4)
  - `internal/core/template/engine.go` — Template substitution (keep using this)
  - `pkg/types/types.go:48-56` — ExecutionHistory (populate all fields)
  - `pkg/types/types.go:31-45` — SavedRequest (source of request data)

  **Acceptance Criteria**:
  - [ ] `grep -c "exec.Command" internal/cli/commands/run.go` returns 0
  - [ ] `go test ./internal/cli/commands/... -v -run TestRun -count=1` — ALL pass
  - [ ] `gurl run <name>` executes and prints response body
  - [ ] Response body + status stored in history

  **QA Scenarios**:
  ```
  Scenario: Run command uses net/http client
    Tool: Bash (grep + go test)
    Steps:
      1. Run: grep -c "exec.Command" internal/cli/commands/run.go
      2. Assert: output is "0"
      3. Run: go test ./internal/cli/commands/... -v -run TestRun -count=1
      4. Assert: exit code 0
    Expected Result: No curl shelling in run.go, all run tests pass
    Evidence: .sisyphus/evidence/task-10-run-migration.txt

  Scenario: Run stores full response in history
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestRunStoresResponse -count=1
      2. Assert: exit code 0
    Expected Result: After run, history entry has Response, StatusCode, SizeBytes populated
    Evidence: .sisyphus/evidence/task-10-run-response.txt
  ```

  **Commit**: YES (TDD commits)
  - Message: `run: migrate to internal/client — unify execution path, capture full response`
  - Files: `internal/cli/commands/run.go`, `internal/cli/commands/run_test.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 11. Migrate executor.go to internal/client

  **What to do**:
  - RED: Write test that verifies `ExecuteCurl()` and `ExecuteCurlWithOutput()` use `internal/client` instead of `exec.Command("curl")`
  - GREEN: Rewrite `internal/core/curl/executor.go`:
    1. `ExecuteCurl()` → build a `client.Request` from `SavedRequest`, call `client.Execute()`, convert `Response` to `ExecutionHistory`
    2. `ExecuteCurlWithOutput()` → same but return the string output, status code, duration
    3. `BuildCurlCommand()` → keep as utility for `paste` command (converts SavedRequest to curl CLI args) but do NOT use it for execution
    4. Keep `parseStatusCode()` as utility for backward compat
  - REFACTOR: Remove `os/exec` import, remove direct curl invocation

  **Must NOT do**:
  - Delete `BuildCurlCommand()` — it's needed by `paste` command for "copy as curl"
  - Shell out to curl for execution
  - Break the `paste` command

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go package refactoring patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 10, different files)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Task 4 (client)

  **References**:
  - `internal/core/curl/executor.go:1-180` — FULL FILE to rewrite (3 exec.Command("curl") calls)
  - `internal/client/` — New HTTP client (Task 4)
  - `internal/core/template/engine.go` — Keep using for variable substitution
  - `internal/cli/commands/paste.go` — Uses BuildCurlCommand (must not break)

  **Acceptance Criteria**:
  - [ ] `grep -c 'exec.Command("curl"' internal/core/curl/executor.go` returns 0
  - [ ] `BuildCurlCommand()` still works (paste command depends on it)
  - [ ] `go test ./internal/core/curl/... -v -count=1` — ALL pass
  - [ ] `go test ./... -count=1` — ALL pass

  **QA Scenarios**:
  ```
  Scenario: Executor no longer shells to curl
    Tool: Bash (grep)
    Steps:
      1. Run: grep -c 'exec.Command("curl"' internal/core/curl/executor.go
      2. Assert: output is "0"
    Expected Result: No curl shell-outs in executor
    Evidence: .sisyphus/evidence/task-11-no-curl-exec.txt

  Scenario: BuildCurlCommand still works for paste
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/core/curl/... -v -run TestBuildCurlCommand -count=1
      2. Assert: exit code 0
    Expected Result: Paste-related curl string building still functions
    Evidence: .sisyphus/evidence/task-11-build-curl-cmd.txt
  ```

  **Commit**: YES
  - Message: `executor: migrate to internal/client — remove curl shelling, keep BuildCurlCommand for paste`
  - Files: `internal/core/curl/executor.go`, `internal/core/curl/executor_test.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 12. Implement detect command (read curl from stdin/file, parse, save)

  **What to do**:
  - RED: Write tests: (1) detect reads curl from stdin, parses via parser, saves request with --name, (2) detect reads from --file flag, (3) detect without --name auto-generates name from URL
  - GREEN: Replace the stub in `detect.go` Action:
    1. Read input: if `--file` flag set, read file contents; else read from `os.Stdin`
    2. Parse input using `curl.Parse()` (rewritten parser from Task 3)
    3. Convert ParsedCurl → SavedRequest using conversion function (Task 5)
    4. Apply --name, --collection flags to the SavedRequest
    5. Save via `db.SaveRequest()`
    6. Print confirmation with parsed details
  - REFACTOR: Handle edge cases (empty stdin, invalid curl, pipe detection)

  **Must NOT do**:
  - Use if-else chains for input source — use switch or early return
  - Show the "under development" placeholder message
  - Reference "scurl" anywhere

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`cli-developer`]
    - `cli-developer`: Stdin reading and CLI input patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 10, 11, 13, 14)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Task 3 (parser)

  **References**:
  - `internal/cli/commands/detect.go:1-50` — FULL FILE to rewrite (current stub with TODO)
  - `internal/core/curl/parser.go` — Parse() function (rewritten in Task 3)
  - `pkg/types/types.go:23-28` — ParsedCurl output from parser
  - `pkg/types/types.go:31-45` — SavedRequest to save to DB

  **Acceptance Criteria**:
  - [ ] `go test ./internal/cli/commands/... -v -run TestDetect -count=1` — ALL pass
  - [ ] Piping `echo "curl -X POST https://example.com" | gurl detect --name test` saves correctly
  - [ ] `gurl detect --file curl.txt --name test` reads file and saves
  - [ ] No "under development" text in detect.go

  **QA Scenarios**:
  ```
  Scenario: Detect parses curl from stdin
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestDetectFromStdin -count=1
      2. Assert: exit code 0
    Expected Result: Curl parsed from stdin and saved as request
    Evidence: .sisyphus/evidence/task-12-detect-stdin.txt

  Scenario: Detect reads from file
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestDetectFromFile -count=1
      2. Assert: exit code 0
    Expected Result: Curl read from file, parsed, saved
    Evidence: .sisyphus/evidence/task-12-detect-file.txt
  ```

  **Commit**: YES
  - Message: `detect: implement curl parsing from stdin/file — replace stub`
  - Files: `internal/cli/commands/detect.go`, `internal/cli/commands/detect_test.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 13. Fix export timestamp (use actual time instead of hardcoded)

  **What to do**:
  - RED: Write test that calls export, verifies `exported_at` is a valid ISO8601 timestamp within last 5 seconds (not "2024-01-01T00:00:00Z")
  - GREEN: In `export.go` line 74, replace `"2024-01-01T00:00:00Z"` with `time.Now().UTC().Format(time.RFC3339)`
  - REFACTOR: Verify export output has correct timestamp

  **Must NOT do**:
  - Change the export JSON structure
  - Break existing import compatibility

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with all other Wave 2 tasks)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Task 1 (rename)

  **References**:
  - `internal/cli/commands/export.go:74` — The hardcoded timestamp `"2024-01-01T00:00:00Z"` (line 74)
  - `internal/cli/commands/export.go:68-76` — Export data struct definition

  **Acceptance Criteria**:
  - [ ] `go test ./internal/cli/commands/... -v -run TestExportTimestamp -count=1` — PASS
  - [ ] Export JSON `exported_at` field is current time, not hardcoded
  - [ ] `grep "2024-01-01" internal/cli/commands/export.go` returns no matches

  **QA Scenarios**:
  ```
  Scenario: Export uses real timestamp
    Tool: Bash (grep + go test)
    Steps:
      1. Run: grep -c "2024-01-01" internal/cli/commands/export.go
      2. Assert: output is "0"
      3. Run: go test ./internal/cli/commands/... -v -run TestExportTimestamp -count=1
      4. Assert: exit code 0
    Expected Result: No hardcoded date, test verifies current timestamp
    Evidence: .sisyphus/evidence/task-13-export-timestamp.txt
  ```

  **Commit**: YES
  - Message: `export: use actual timestamp — was hardcoded to 2024-01-01`
  - Files: `internal/cli/commands/export.go`, `internal/cli/commands/export_test.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 14. Fix save command — accept --curl, -X, -H, -d flags, stdin

  **What to do**:
  - RED: Write tests: (1) `save "name" --curl "curl -X POST -H 'CT: app/json' -d '{}' https://ex.com"` parses and saves with method, headers, body; (2) `save "name" -X POST -H "Auth: Bearer tok" -d '{"a":1}' https://ex.com` saves with individual flags; (3) piping curl to save via stdin
  - GREEN: Rewrite `save.go` Action:
    1. Add flags: `--curl` (full curl string), `-X` (method), `-H` (header, repeatable StringSliceFlag), `-d` (body)
    2. If `--curl` provided: parse using `curl.Parse()`, convert to SavedRequest
    3. If individual flags (-X, -H, -d) provided: construct SavedRequest directly
    4. If neither: read from stdin, parse as curl
    5. Apply --collection, --tag, --description, --format to the SavedRequest
    6. Save via `db.SaveRequest()`
  - REFACTOR: Handle all input modes cleanly

  **Must NOT do**:
  - Use if-else-if-else chains for input mode detection — use switch with early returns
  - Break existing `save "name" <url>` syntax (backward compat)
  - Ignore headers/body from curl commands

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`cli-developer`]
    - `cli-developer`: CLI flag patterns and input mode handling

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 10, 11, 12, 13)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Tasks 3 (parser), 5 (conversion)

  **References**:
  - `internal/cli/commands/save.go:1-67` — FULL FILE to rewrite (currently only accepts name + URL as GET)
  - `internal/core/curl/parser.go` — Parse() for --curl mode (Task 3)
  - `pkg/types/types.go:127-140` — NewSavedRequest constructor
  - `pkg/types/types.go:142-149` — AddHeader/AddTag helper methods
  - `internal/cli/commands/detect.go` — Similar stdin-reading pattern (Task 12)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/cli/commands/... -v -run TestSave -count=1` — ALL pass
  - [ ] `save "test" --curl "curl -X POST -H 'CT: app/json' -d '{}' https://ex.com"` saves with POST method, header, body
  - [ ] `save "test" -X POST -H "Auth: tok" https://ex.com` saves with individual flags
  - [ ] `save "test" https://ex.com` still works (backward compat)
  - [ ] `go test ./... -count=1` — ALL pass

  **QA Scenarios**:
  ```
  Scenario: Save with --curl flag parses full command
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestSaveWithCurl -count=1
      2. Assert: exit code 0
    Expected Result: SavedRequest has correct method, headers, body, URL from parsed curl
    Evidence: .sisyphus/evidence/task-14-save-curl.txt

  Scenario: Save with individual flags
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestSaveWithFlags -count=1
      2. Assert: exit code 0
    Expected Result: -X, -H, -d flags correctly populate SavedRequest
    Evidence: .sisyphus/evidence/task-14-save-flags.txt

  Scenario: Backward compat — save name url still works
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestSaveBackwardCompat -count=1
      2. Assert: exit code 0
    Expected Result: save "name" "url" saves as GET with no headers/body
    Evidence: .sisyphus/evidence/task-14-save-compat.txt
  ```

  **Commit**: YES (TDD commits)
  - Message: `save: accept --curl, -X, -H, -d flags + stdin — was only name+URL as GET`
  - Files: `internal/cli/commands/save.go`, `internal/cli/commands/save_test.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 15. Environment system (internal/env/)

  **What to do**:
  - RED: Write tests: (1) create environment with name + variables, (2) get environment by name, (3) list environments, (4) update variables, (5) delete environment, (6) active environment selection
  - GREEN: Create `internal/env/` package:
    - `env.go` — `Environment` struct: `{ID, Name, Variables map[string]string, IsActive bool, CreatedAt, UpdatedAt}`
    - `store.go` — Environment storage using LevelDB (via DB interface extension or separate prefix):
      - `SaveEnvironment(env *Environment) error`
      - `GetEnvironment(name string) (*Environment, error)`
      - `ListEnvironments() ([]*Environment, error)`
      - `DeleteEnvironment(name string) error`
      - `SetActiveEnvironment(name string) error`
      - `GetActiveEnvironment() (*Environment, error)`
    - Use LevelDB key prefix `env:` to namespace environment entries
  - REFACTOR: Add schema version 2 migration that creates env key prefix

  **Must NOT do**:
  - Store environments in separate files (use LevelDB like requests)
  - Use YAML/JSON config files for environments (TOML only for user config per AGENT.md)
  - Create a separate database — reuse existing LevelDB with key prefixes

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`golang-pro`, `cli-developer`]
    - `golang-pro`: Idiomatic Go data layer patterns
    - `cli-developer`: Environment management patterns in CLI tools

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on Task 8)
  - **Parallel Group**: Wave 3
  - **Blocks**: Tasks 16, 17, 18, 19, 20, 45
  - **Blocked By**: Task 8 (schema versioning for migration)

  **References**:
  - `internal/storage/db.go:16-27` — DB interface to extend with env methods (or separate env store)
  - `internal/storage/db.go:86-130` — SaveRequest pattern (follow same LevelDB batch + prefix pattern)
  - `pkg/types/types.go:80-111` — Config struct (environments complement config)
  - Competitor patterns: Bruno stores envs as `.bru` files, Insomnia uses JSON DB, Postman uses cloud+JSON

  **Acceptance Criteria**:
  - [ ] `go test ./internal/env/... -v -count=1` — ALL pass
  - [ ] Can create, get, list, update, delete environments
  - [ ] Can set and get active environment
  - [ ] Environment variables are key-value string maps

  **QA Scenarios**:
  ```
  Scenario: CRUD operations on environments
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/env/... -v -run TestEnvironmentCRUD -count=1
      2. Assert: exit code 0
    Expected Result: Create, read, update, delete all work on environments
    Evidence: .sisyphus/evidence/task-15-env-crud.txt

  Scenario: Active environment selection
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/env/... -v -run TestActiveEnvironment -count=1
      2. Assert: exit code 0
    Expected Result: SetActive/GetActive correctly persists and retrieves active env
    Evidence: .sisyphus/evidence/task-15-env-active.txt
  ```

  **Commit**: YES (TDD commits)
  - Message: `env: add environment system with CRUD + active selection — foundation for variable scoping`
  - Files: `internal/env/env.go`, `internal/env/store.go`, `internal/env/store_test.go`
  - Pre-commit: `go test ./internal/env/... -count=1`

- [x] 16. Environment CLI commands (env create/list/switch/delete/show)

  **What to do**:
  - RED: Write tests for each subcommand: `env create`, `env list`, `env switch`, `env delete`, `env show`
  - GREEN: Create `internal/cli/commands/env.go` with subcommands:
    - `env create <name> --var KEY=VALUE --var KEY2=VALUE2` — creates environment with initial vars
    - `env list` — shows all environments with active marker
    - `env switch <name>` — sets active environment
    - `env delete <name>` — deletes environment (with confirmation if active)
    - `env show <name>` — shows environment variables (masks secrets)
    - `env set <name> --var KEY=VALUE` — adds/updates a variable
    - `env unset <name> --var KEY` — removes a variable
  - Register all subcommands in `cmd/gurl/main.go`
  - REFACTOR: Consistent output formatting

  **Must NOT do**:
  - Use if-else chains for subcommand dispatch — urfave/cli handles this
  - Store secrets in plain text (mark with `--secret` flag, store encrypted in Task 19)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`cli-developer`]
    - `cli-developer`: CLI subcommand design patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 17-23 once Task 15 done)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Task 15 (env store)

  **References**:
  - `internal/env/` — Environment store (Task 15)
  - `cmd/gurl/main.go` — Register new `env` command
  - `internal/cli/commands/collection.go` — Similar subcommand pattern to follow (if exists)
  - `internal/cli/commands/save.go` — Flag definition patterns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/cli/commands/... -v -run TestEnv -count=1` — ALL pass
  - [ ] `gurl env create dev --var BASE_URL=https://dev.api.com` creates environment
  - [ ] `gurl env list` shows all environments
  - [ ] `gurl env switch dev` sets dev as active

  **QA Scenarios**:
  ```
  Scenario: Full env workflow
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestEnvCreate -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/cli/commands/... -v -run TestEnvList -count=1
      4. Assert: exit code 0
    Expected Result: Environment created and visible in list
    Evidence: .sisyphus/evidence/task-16-env-commands.txt
  ```

  **Commit**: YES
  - Message: `env: add CLI commands (create/list/switch/delete/show/set/unset)`
  - Files: `internal/cli/commands/env.go`, `internal/cli/commands/env_test.go`, `cmd/gurl/main.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 17. Wire environments into run command

  **What to do**:
  - RED: Write test: `run "test" --env dev` resolves variables from "dev" environment before executing
  - GREEN: In `run.go`:
    1. Add `--env` flag (StringFlag)
    2. If `--env` set: load that environment; else load active environment (if any)
    3. Merge environment variables with `--var` overrides (CLI vars take precedence)
    4. Pass merged vars to template substitution
  - REFACTOR: Ensure no env = no change in behavior (backward compat)

  **Must NOT do**:
  - Use if-else chains for variable precedence — use a simple map merge (env vars first, then CLI vars overwrite)
  - Make --env required (optional, uses active env by default)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`cli-developer`]
    - `cli-developer`: CLI flag integration patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 18-23)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Tasks 10 (run uses client), 15 (env system)

  **References**:
  - `internal/cli/commands/run.go` — Add --env flag, load env vars before template sub
  - `internal/env/` — GetEnvironment, GetActiveEnvironment
  - `internal/core/template/engine.go` — Substitute(template, vars) — vars is map[string]string

  **Acceptance Criteria**:
  - [ ] `go test ./internal/cli/commands/... -v -run TestRunWithEnv -count=1` — PASS
  - [ ] `--var` flags override environment variables
  - [ ] No `--env` flag = uses active environment if set
  - [ ] No active environment and no `--env` = runs without env (backward compat)

  **QA Scenarios**:
  ```
  Scenario: Run with explicit environment
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestRunWithEnv -count=1
      2. Assert: exit code 0
    Expected Result: Request URL/headers/body have env variables substituted
    Evidence: .sisyphus/evidence/task-17-run-env.txt

  Scenario: CLI vars override env vars
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestRunVarOverridesEnv -count=1
      2. Assert: exit code 0
    Expected Result: --var value used instead of environment value for same key
    Evidence: .sisyphus/evidence/task-17-var-override.txt
  ```

  **Commit**: YES
  - Message: `run: wire environment variables into request execution — --env flag + active env`
  - Files: `internal/cli/commands/run.go`, `internal/cli/commands/run_test.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 18. Variable scoping (request→folder→collection→environment→global)

  **What to do**:
  - RED: Write test: variables resolve in order: request-level > folder-level > collection-level > environment-level > global, with closest scope winning
  - GREEN: Create variable resolution function in `internal/env/`:
    - `ResolveVariables(request *SavedRequest, env *Environment, globalVars map[string]string) map[string]string`
    - Scoping order (later overrides earlier): global → environment → collection → folder → request → CLI --var
    - For now: implement environment + global + CLI levels (folder/collection scoping deferred to Task 55 nested folders)
  - REFACTOR: Wire into run command's variable resolution

  **Must NOT do**:
  - Implement folder-level variables yet (no nested folders until Task 55)
  - Use if-else chains for scope resolution — use a loop over ordered scopes

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go map merging and scoping patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 16, 17, 19-23)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 29 (auth inheritance uses same scoping)
  - **Blocked By**: Task 15 (env system)

  **References**:
  - `internal/env/env.go` — Environment with Variables map
  - `internal/core/template/engine.go` — Template substitution consumes map[string]string
  - `pkg/types/types.go:39` — SavedRequest.Variables ([]Var — request-level vars)
  - Bruno's approach: env vars → collection vars → request vars → runtime vars

  **Acceptance Criteria**:
  - [ ] `go test ./internal/env/... -v -run TestVariableScoping -count=1` — ALL pass
  - [ ] Environment var overridden by CLI var for same key
  - [ ] Missing key in request scope falls back to environment scope

  **QA Scenarios**:
  ```
  Scenario: Variable scoping precedence
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/env/... -v -run TestVariableScoping -count=1
      2. Assert: exit code 0
    Expected Result: Closest scope wins, fallback to parent scopes works
    Evidence: .sisyphus/evidence/task-18-var-scoping.txt
  ```

  **Commit**: YES
  - Message: `env: add variable scoping resolution (env→global→cli) — closest scope wins`
  - Files: `internal/env/resolve.go`, `internal/env/resolve_test.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 19. Secret/encrypted variables

  **What to do**:
  - RED: Write test: (1) save a secret variable — stored encrypted, (2) retrieve secret — decrypted only at execution time, (3) `env show` masks secret values with `***`
  - GREEN: Add to `internal/env/`:
    - `secrets.go` — encryption/decryption using Go's `crypto/aes` with a machine-derived key (or passphrase-derived via PBKDF2)
    - Key derivation: use a machine-local key from `~/.local/share/gurl/.secret-key` (auto-generated on first use)
    - Environment variables marked as `secret:true` get encrypted before storage
    - `env create --var API_KEY=xyz --secret API_KEY` marks API_KEY as secret
    - Masked display: `env show` displays secret values as `*****`
  - REFACTOR: Ensure secrets round-trip correctly

  **Must NOT do**:
  - Use hardcoded encryption keys
  - Store secrets in plain text
  - Require user to manage key files manually (auto-generate)
  - Use complex key management (keep it simple — local machine encryption)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go crypto patterns (AES-GCM, PBKDF2)

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 16-18, 20-23)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Task 15 (env system)

  **References**:
  - `internal/env/env.go` — Environment struct (add SecretKeys field or per-var metadata)
  - `crypto/aes`, `crypto/cipher` — Go standard library AES-GCM encryption
  - `golang.org/x/crypto/pbkdf2` — Key derivation from machine secret
  - Competitor approach: Insomnia uses local encryption, Bruno uses `dotenv`, Postman uses vault

  **Acceptance Criteria**:
  - [ ] `go test ./internal/env/... -v -run TestSecret -count=1` — ALL pass
  - [ ] Secret stored encrypted in DB (raw DB read shows ciphertext)
  - [ ] Decrypted correctly at execution time
  - [ ] `env show` masks secrets

  **QA Scenarios**:
  ```
  Scenario: Secret encryption round-trip
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/env/... -v -run TestSecretEncryption -count=1
      2. Assert: exit code 0
    Expected Result: Encrypt then decrypt returns original value
    Evidence: .sisyphus/evidence/task-19-secret-roundtrip.txt

  Scenario: Secrets masked in display
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/env/... -v -run TestSecretMasking -count=1
      2. Assert: exit code 0
    Expected Result: Secret values display as "*****"
    Evidence: .sisyphus/evidence/task-19-secret-masking.txt
  ```

  **Commit**: YES
  - Message: `env: add secret variable encryption — AES-GCM with machine-local key`
  - Files: `internal/env/secrets.go`, `internal/env/secrets_test.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 20. .env file support

  **What to do**:
  - RED: Write test: (1) parse `.env` file into key-value pairs, (2) `env import --file .env --name dev` creates environment from .env, (3) handles comments, empty lines, quoted values, multiline
  - GREEN: Add to `internal/env/`:
    - `dotenv.go` — `.env` file parser (no external dependency needed — simple line-by-line parser)
    - Supports: `KEY=value`, `KEY="quoted value"`, `KEY='single quoted'`, `# comments`, empty lines, `export KEY=value`
    - Add `env import` subcommand in `env.go` CLI: `gurl env import --file .env --name dev`
    - Also: auto-detect `.env` file in current directory and offer to load
  - REFACTOR: Handle edge cases (BOM, Windows line endings, trailing whitespace)

  **Must NOT do**:
  - Add external dependency for .env parsing (Go's stdlib is sufficient for this)
  - Auto-load .env without user action (security concern)
  - Use if-else chains for line type detection — use switch

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`cli-developer`]
    - `cli-developer`: .env file parsing patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 16-19, 21-23)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Task 15 (env system)

  **References**:
  - `internal/env/env.go` — Environment struct to populate
  - `internal/cli/commands/env.go` — Add `import` subcommand
  - Standard .env format: `KEY=value`, `# comment`, `export KEY=value`
  - Bruno uses `.env` files natively alongside `.bru` files

  **Acceptance Criteria**:
  - [ ] `go test ./internal/env/... -v -run TestDotenv -count=1` — ALL pass
  - [ ] Parses standard .env format (KEY=value, comments, quotes)
  - [ ] `gurl env import --file .env --name dev` creates env with all vars

  **QA Scenarios**:
  ```
  Scenario: Parse .env file correctly
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/env/... -v -run TestDotenvParse -count=1
      2. Assert: exit code 0
    Expected Result: All standard .env patterns parsed correctly
    Evidence: .sisyphus/evidence/task-20-dotenv-parse.txt
  ```

  **Commit**: YES
  - Message: `env: add .env file import — parse dotenv format into environment variables`
  - Files: `internal/env/dotenv.go`, `internal/env/dotenv_test.go`, `internal/cli/commands/env.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 21. Auth framework (internal/auth/) + Basic auth handler

  **What to do**:
  - RED: Write tests: (1) auth framework registers handlers by type, (2) Basic auth handler sets Authorization header with base64-encoded "user:pass", (3) auth applied to client.Request before execution
  - GREEN: Create `internal/auth/` package:
    - `auth.go` — `AuthType` string enum, `AuthConfig` struct `{Type, Params map[string]string}`, `Handler` interface `{Apply(req *client.Request) error}`
    - `registry.go` — `Registry` with `Register(authType string, handler Handler)` and `Get(authType string) Handler`
    - `basic.go` — `BasicHandler` implementing Handler: reads "username" and "password" from Params, sets `Authorization: Basic <base64>`
    - Wire auth into `internal/client/`: `Request` gets optional `AuthConfig`, `Execute()` calls handler before sending
  - REFACTOR: Ensure auth is optional (nil AuthConfig = no auth applied)

  **Must NOT do**:
  - Use if-else chains for auth type dispatch — use registry map lookup
  - Hardcode auth types — use pluggable registry
  - Add all auth types here — only Basic (others are separate tasks)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go interface design, registry patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 16-20)
  - **Parallel Group**: Wave 3
  - **Blocks**: Tasks 22-29 (all auth types depend on framework)
  - **Blocked By**: Tasks 4 (client), 15 (env for auth credentials)

  **References**:
  - `internal/client/request.go` — Add AuthConfig field to Request struct
  - `internal/client/execute.go` — Call auth handler before http.Do()
  - `pkg/types/types.go:31-45` — SavedRequest (will need AuthConfig field in future schema update)
  - Competitor approach: Insomnia has per-request auth dropdown, Bruno stores auth in .bru, Postman has auth tab

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -v -count=1` — ALL pass
  - [ ] Basic auth sets correct `Authorization: Basic <base64>` header
  - [ ] Registry lookup returns correct handler for auth type
  - [ ] nil AuthConfig = request sent without auth modification

  **QA Scenarios**:
  ```
  Scenario: Basic auth applies correct header
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestBasicAuth -count=1
      2. Assert: exit code 0
    Expected Result: Authorization header contains "Basic " + base64("user:pass")
    Evidence: .sisyphus/evidence/task-21-basic-auth.txt

  Scenario: Auth registry dispatches correctly
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestAuthRegistry -count=1
      2. Assert: exit code 0
    Expected Result: Registered handler retrieved by type string
    Evidence: .sisyphus/evidence/task-21-auth-registry.txt
  ```

  **Commit**: YES (TDD commits)
  - Message: `auth: add auth framework with registry pattern + Basic auth handler`
  - Files: `internal/auth/auth.go`, `internal/auth/registry.go`, `internal/auth/basic.go`, `internal/auth/basic_test.go`, `internal/auth/registry_test.go`
  - Pre-commit: `go test ./internal/auth/... -count=1`

- [x] 22. Bearer token auth

  **What to do**:
  - RED: Write test: Bearer handler sets `Authorization: Bearer <token>` header
  - GREEN: Create `internal/auth/bearer.go`:
    - `BearerHandler` implementing Handler
    - Reads "token" from Params, sets `Authorization: Bearer <token>`
    - Register with auth registry as "bearer" type
  - REFACTOR: Handle empty token gracefully

  **Must NOT do**:
  - Implement token refresh (that's OAuth territory)
  - Hardcode token values

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 23)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Task 21 (auth framework)

  **References**:
  - `internal/auth/auth.go` — Handler interface
  - `internal/auth/basic.go` — Pattern to follow for new handler
  - `internal/auth/registry.go` — Register bearer handler

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -v -run TestBearer -count=1` — PASS
  - [ ] Sets `Authorization: Bearer <token>` header correctly

  **QA Scenarios**:
  ```
  Scenario: Bearer token auth
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestBearerAuth -count=1
      2. Assert: exit code 0
    Expected Result: Header set to "Bearer <token>"
    Evidence: .sisyphus/evidence/task-22-bearer-auth.txt
  ```

  **Commit**: YES
  - Message: `auth: add Bearer token handler — sets Authorization: Bearer <token>`
  - Files: `internal/auth/bearer.go`, `internal/auth/bearer_test.go`
  - Pre-commit: `go test ./internal/auth/... -count=1`

- [x] 23. API Key auth (header or query param)

  **What to do**:
  - RED: Write tests: (1) API key in header mode sets custom header (e.g., `X-API-Key: <key>`), (2) API key in query param mode appends `?api_key=<key>` to URL
  - GREEN: Create `internal/auth/apikey.go`:
    - `APIKeyHandler` implementing Handler
    - Params: "key" (the value), "name" (header/param name, e.g., "X-API-Key"), "in" ("header" or "query")
    - If `in=header`: set header `name: key`
    - If `in=query`: append `name=key` to URL query string
    - Register as "apikey" type
  - REFACTOR: Handle URL with existing query params (append with `&`)

  **Must NOT do**:
  - Use if-else for in=header/query — use switch

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 22)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Task 21 (auth framework)

  **References**:
  - `internal/auth/auth.go` — Handler interface
  - `internal/auth/basic.go` — Handler pattern to follow
  - OpenAPI spec: API Key security scheme supports `in: header`, `in: query`, `in: cookie`

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -v -run TestAPIKey -count=1` — ALL pass
  - [ ] Header mode sets correct custom header
  - [ ] Query mode appends to URL correctly

  **QA Scenarios**:
  ```
  Scenario: API key in header
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestAPIKeyHeader -count=1
      2. Assert: exit code 0
    Expected Result: Custom header "X-API-Key: <value>" set on request
    Evidence: .sisyphus/evidence/task-23-apikey-header.txt

  Scenario: API key in query param
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestAPIKeyQuery -count=1
      2. Assert: exit code 0
    Expected Result: URL has "?api_key=<value>" appended
    Evidence: .sisyphus/evidence/task-23-apikey-query.txt
  ```

  **Commit**: YES
  - Message: `auth: add API Key handler — supports header and query param modes`
  - Files: `internal/auth/apikey.go`, `internal/auth/apikey_test.go`
  - Pre-commit: `go test ./internal/auth/... -count=1`

- [x] 24. OAuth 2.0 (authorization code + client credentials flows)

  **What to do**:
  - RED: Write tests: (1) client credentials flow: POST to token endpoint with client_id+secret, parse JSON response, set Bearer token; (2) authorization code flow: build auth URL, exchange code for token, set Bearer; (3) token refresh when expired
  - GREEN: Create `internal/auth/oauth2.go`:
    - `OAuth2Handler` implementing Handler
    - Params: "grant_type" (authorization_code / client_credentials), "client_id", "client_secret", "token_url", "auth_url", "redirect_uri", "scope"
    - Client credentials: POST to token_url, parse `access_token` from JSON response, set Bearer header
    - Authorization code: build auth URL → open browser or show URL → listen on localhost for callback → exchange code → set Bearer
    - Token caching: cache token in memory with expiry, refresh automatically
    - Register as "oauth2" type
  - REFACTOR: Handle token refresh, error responses from OAuth providers

  **Must NOT do**:
  - Use if-else chains for grant type dispatch — use switch
  - Implement PKCE here (can be added later)
  - Store tokens in DB (in-memory cache only per session)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go OAuth2 implementation patterns, HTTP client for token exchange

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 25-33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 21 (auth framework)

  **References**:
  - `internal/auth/auth.go` — Handler interface
  - `internal/auth/registry.go` — Register as "oauth2"
  - `internal/client/` — HTTP client for token endpoint calls
  - `golang.org/x/oauth2` — Consider using Go's OAuth2 library for standard flow
  - RFC 6749 — OAuth 2.0 spec

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -v -run TestOAuth2 -count=1` — ALL pass
  - [ ] Client credentials flow gets token and sets Bearer header
  - [ ] Token refresh works when token expires
  - [ ] Auth URL correctly constructed for authorization code flow

  **QA Scenarios**:
  ```
  Scenario: Client credentials flow
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestOAuth2ClientCredentials -count=1
      2. Assert: exit code 0
    Expected Result: Token fetched from mock server, Bearer header set
    Evidence: .sisyphus/evidence/task-24-oauth2-client-creds.txt

  Scenario: Token refresh on expiry
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestOAuth2TokenRefresh -count=1
      2. Assert: exit code 0
    Expected Result: Expired token triggers refresh, new token used
    Evidence: .sisyphus/evidence/task-24-oauth2-refresh.txt
  ```

  **Commit**: YES (TDD commits)
  - Message: `auth: add OAuth 2.0 handler — client credentials + authorization code flows`
  - Files: `internal/auth/oauth2.go`, `internal/auth/oauth2_test.go`
  - Pre-commit: `go test ./internal/auth/... -count=1`

- [x] 25. OAuth 1.0

  **What to do**:
  - RED: Write test: OAuth 1.0 handler generates correct signature base string, signs with HMAC-SHA1, sets Authorization header with oauth_* params
  - GREEN: Create `internal/auth/oauth1.go`:
    - `OAuth1Handler` implementing Handler
    - Params: "consumer_key", "consumer_secret", "token", "token_secret"
    - Implement OAuth 1.0a signing: nonce, timestamp, signature base string, HMAC-SHA1 signing
    - Set `Authorization: OAuth oauth_consumer_key="...", oauth_nonce="...", ...` header
    - Register as "oauth1" type
  - REFACTOR: Ensure signature matches reference implementations

  **Must NOT do**:
  - Use external OAuth1 library unless Go standard lib is insufficient
  - Use if-else chains for parameter assembly

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go crypto (HMAC-SHA1), HTTP auth patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 24, 26-33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 21

  **References**:
  - `internal/auth/auth.go` — Handler interface
  - RFC 5849 — OAuth 1.0 Protocol
  - `crypto/hmac`, `crypto/sha1` — For HMAC-SHA1 signing

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -v -run TestOAuth1 -count=1` — PASS
  - [ ] Signature matches known test vector
  - [ ] Authorization header correctly formatted

  **QA Scenarios**:
  ```
  Scenario: OAuth 1.0 signing
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestOAuth1Signing -count=1
      2. Assert: exit code 0
    Expected Result: Correct HMAC-SHA1 signature generated
    Evidence: .sisyphus/evidence/task-25-oauth1.txt
  ```

  **Commit**: YES
  - Message: `auth: add OAuth 1.0 handler — HMAC-SHA1 signing per RFC 5849`
  - Files: `internal/auth/oauth1.go`, `internal/auth/oauth1_test.go`
  - Pre-commit: `go test ./internal/auth/... -count=1`

- [x] 26. AWS Signature v4

  **What to do**:
  - RED: Write test: AWS Sig v4 handler generates correct canonical request, string to sign, signing key, and Authorization header matching AWS test suite
  - GREEN: Create `internal/auth/awsv4.go`:
    - `AWSv4Handler` implementing Handler
    - Params: "access_key", "secret_key", "region", "service", "session_token" (optional)
    - Implement: canonical request → string to sign → signing key (HMAC chain) → signature → Authorization header
    - Set `Authorization: AWS4-HMAC-SHA256 Credential=.../.../.../s3/aws4_request, SignedHeaders=..., Signature=...`
    - Also set `x-amz-date`, `x-amz-security-token` (if session token), `x-amz-content-sha256` headers
    - Register as "awsv4" type
  - REFACTOR: Verify against AWS test suite vectors

  **Must NOT do**:
  - Use AWS SDK (too heavy — implement signing directly)
  - Skip payload hashing (required for v4)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go crypto patterns, AWS signing implementation

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 24-25, 27-33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 21

  **References**:
  - `internal/auth/auth.go` — Handler interface
  - AWS Sig v4 documentation + test suite: https://docs.aws.amazon.com/general/latest/gr/sigv4-create-canonical-request.html
  - `crypto/hmac`, `crypto/sha256` — For signing chain

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -v -run TestAWSv4 -count=1` — PASS
  - [ ] Signature matches AWS test vectors
  - [ ] All required headers set (x-amz-date, x-amz-content-sha256)

  **QA Scenarios**:
  ```
  Scenario: AWS Sig v4 test vector
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestAWSv4Signing -count=1
      2. Assert: exit code 0
    Expected Result: Generated signature matches expected test vector
    Evidence: .sisyphus/evidence/task-26-awsv4.txt
  ```

  **Commit**: YES
  - Message: `auth: add AWS Signature v4 handler — canonical request + HMAC signing chain`
  - Files: `internal/auth/awsv4.go`, `internal/auth/awsv4_test.go`
  - Pre-commit: `go test ./internal/auth/... -count=1`

- [x] 27. Digest auth

  **What to do**:
  - RED: Write test: (1) first request gets 401 with WWW-Authenticate: Digest, (2) handler parses challenge, computes response hash, (3) retry with Authorization: Digest header
  - GREEN: Create `internal/auth/digest.go`:
    - `DigestHandler` implementing Handler (needs special handling — requires 2 requests)
    - Parse WWW-Authenticate challenge: realm, nonce, qop, opaque, algorithm
    - Compute: HA1 = MD5(username:realm:password), HA2 = MD5(method:uri), response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
    - Set `Authorization: Digest username="...", realm="...", nonce="...", ...`
    - Register as "digest" type
  - REFACTOR: Handle auth-int qop, SHA-256 algorithm variant

  **Must NOT do**:
  - Skip the 401 challenge step (Digest requires server challenge first)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go crypto (MD5/SHA-256), HTTP challenge-response patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 24-26, 28-33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 21

  **References**:
  - `internal/auth/auth.go` — Handler interface (may need extension for challenge-response)
  - RFC 7616 — HTTP Digest Access Authentication
  - `crypto/md5` — For hash computation

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -v -run TestDigest -count=1` — PASS
  - [ ] Correct MD5 hash chain for challenge-response
  - [ ] Works with httptest server returning 401 Digest challenge

  **QA Scenarios**:
  ```
  Scenario: Digest auth challenge-response
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestDigestAuth -count=1
      2. Assert: exit code 0
    Expected Result: 401 challenge parsed, response hash computed, retry succeeds
    Evidence: .sisyphus/evidence/task-27-digest.txt
  ```

  **Commit**: YES
  - Message: `auth: add Digest auth handler — RFC 7616 challenge-response with MD5/SHA-256`
  - Files: `internal/auth/digest.go`, `internal/auth/digest_test.go`
  - Pre-commit: `go test ./internal/auth/... -count=1`

- [x] 28. NTLM auth

  **What to do**:
  - RED: Write test: NTLM handler performs 3-step negotiate-challenge-authenticate handshake
  - GREEN: Create `internal/auth/ntlm.go`:
    - `NTLMHandler` implementing Handler
    - Use `github.com/Azure/go-ntlmssp` or implement NTLMv2 directly
    - Step 1: Send negotiate message → get challenge from server
    - Step 2: Compute authenticate message with NTLMv2 response
    - Step 3: Send authenticate message
    - Register as "ntlm" type
  - REFACTOR: Handle domain\user format, NTLMv2 preferred over NTLMv1

  **Must NOT do**:
  - Implement NTLMv1 only (insecure — prefer NTLMv2)
  - Add dependency without checking size impact

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go NTLM patterns, binary protocol encoding

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 24-27, 29-33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 21

  **References**:
  - `internal/auth/auth.go` — Handler interface
  - `github.com/Azure/go-ntlmssp` — NTLM library option
  - MS-NLMP spec — NTLM protocol documentation

  **Acceptance Criteria**:
  - [ ] `go test ./internal/auth/... -v -run TestNTLM -count=1` — PASS
  - [ ] 3-step handshake completes against mock server

  **QA Scenarios**:
  ```
  Scenario: NTLM handshake
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/auth/... -v -run TestNTLMAuth -count=1
      2. Assert: exit code 0
    Expected Result: Negotiate→Challenge→Authenticate completes successfully
    Evidence: .sisyphus/evidence/task-28-ntlm.txt
  ```

  **Commit**: YES
  - Message: `auth: add NTLM handler — NTLMv2 three-step handshake`
  - Files: `internal/auth/ntlm.go`, `internal/auth/ntlm_test.go`
  - Pre-commit: `go test ./internal/auth/... -count=1`

- [x] 29. Auth inheritance from collections/folders

  **What to do**:
  - RED: Write test: (1) request with no auth config inherits auth from its collection, (2) request auth overrides collection auth, (3) "no auth" explicitly set on request prevents inheritance
  - GREEN: Add to auth framework:
    - `SavedRequest` gets `AuthConfig *AuthConfig` field (schema version 3 migration)
    - `Collection` gets `AuthConfig *AuthConfig` field
    - Resolution: request.AuthConfig > collection.AuthConfig > none
    - In run command: resolve auth before execution
  - REFACTOR: Handle nil at every level gracefully

  **Must NOT do**:
  - Break existing requests (nil AuthConfig = no auth, as before)
  - Use if-else chains for inheritance resolution

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go nil handling, struct embedding patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 30-33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Tasks 18 (scoping), 21 (auth framework)

  **References**:
  - `internal/auth/auth.go` — AuthConfig struct
  - `pkg/types/types.go:31-45` — SavedRequest (add AuthConfig field)
  - `pkg/types/types.go:72-77` — Collection (add AuthConfig field)
  - `internal/storage/db.go` — Schema migration to add AuthConfig fields

  **Acceptance Criteria**:
  - [ ] `go test ./... -v -run TestAuthInheritance -count=1` — ALL pass
  - [ ] Request without auth inherits from collection
  - [ ] Request with auth overrides collection
  - [ ] Existing requests without AuthConfig still work (nil = no auth)

  **QA Scenarios**:
  ```
  Scenario: Auth inheritance from collection
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./... -v -run TestAuthInheritance -count=1
      2. Assert: exit code 0
    Expected Result: Request inherits collection's auth when no request-level auth set
    Evidence: .sisyphus/evidence/task-29-auth-inheritance.txt
  ```

  **Commit**: YES
  - Message: `auth: add auth inheritance — request inherits from collection, explicit override`
  - Files: `pkg/types/types.go`, `internal/auth/resolve.go`, `internal/storage/migration.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 30. Cookie jar management

  **What to do**:
  - RED: Write tests: (1) cookies from Set-Cookie headers saved to jar, (2) subsequent requests to same domain include saved cookies, (3) cookie expiry honored, (4) cookie jar persists across sessions
  - GREEN: Create `internal/cookies/` package:
    - `jar.go` — `CookieJar` wrapping `net/http/cookiejar` with persistence
    - Persist cookies to LevelDB (key prefix `cookie:`) as JSON
    - Load cookies on startup, save after each request
    - Wire into `internal/client/`: set `http.Client.Jar`
    - Add `--cookies` flag to run command (default: enabled)
    - Add `cookies list`, `cookies clear` commands
  - REFACTOR: Handle cookie domains, paths, secure/httponly flags

  **Must NOT do**:
  - Implement custom cookie parsing (use `net/http/cookiejar`)
  - Store cookies unencrypted if they contain session tokens (use same encryption as secrets)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go HTTP cookie jar, persistence patterns

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 24-29, 31-33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 4 (client)

  **References**:
  - `internal/client/client.go` — HTTP client (set Jar field)
  - `net/http/cookiejar` — Go standard cookie jar
  - `internal/storage/db.go` — LevelDB for persistence (cookie: prefix)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/cookies/... -v -count=1` — ALL pass
  - [ ] Cookies auto-sent on subsequent requests to same domain
  - [ ] Cookies persist across CLI invocations
  - [ ] `gurl cookies list` shows stored cookies

  **QA Scenarios**:
  ```
  Scenario: Cookie round-trip
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cookies/... -v -run TestCookieJar -count=1
      2. Assert: exit code 0
    Expected Result: Set-Cookie saved, sent back on next request
    Evidence: .sisyphus/evidence/task-30-cookies.txt
  ```

  **Commit**: YES
  - Message: `cookies: add persistent cookie jar — auto-sends cookies, persists to LevelDB`
  - Files: `internal/cookies/jar.go`, `internal/cookies/jar_test.go`, `internal/cli/commands/cookies.go`
  - Pre-commit: `go test ./... -count=1`

- [x] 31. Redirect handling (follow/max config)

  **What to do**:
  - RED: Write tests: (1) redirects followed by default (max 10), (2) `--no-follow` disables redirect following, (3) `--max-redirects N` limits redirect count, (4) redirect chain tracked in response metadata
  - GREEN: Configure `internal/client/`:
    - Set `http.Client.CheckRedirect` function
    - Default: follow up to 10 redirects
    - `--no-follow` flag: return redirect response without following
    - `--max-redirects N` flag: limit redirect count
    - Track redirect chain in Response metadata (each hop URL + status)
  - REFACTOR: Add redirect info to response output

  **Must NOT do**:
  - Follow redirects infinitely (cap at configurable max, default 10)
  - Use if-else for redirect policy — use function assignment

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 24-30, 32-33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 4 (client)

  **References**:
  - `internal/client/client.go` — http.Client.CheckRedirect function
  - `net/http` — CheckRedirect signature: `func(req *Request, via []*Request) error`

  **Acceptance Criteria**:
  - [ ] `go test ./internal/client/... -v -run TestRedirect -count=1` — ALL pass
  - [ ] Default: follows up to 10 redirects
  - [ ] `--no-follow`: returns 301/302 response directly

  **QA Scenarios**:
  ```
  Scenario: Redirect following
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/client/... -v -run TestRedirectFollowing -count=1
      2. Assert: exit code 0
    Expected Result: Redirects followed, final response returned
    Evidence: .sisyphus/evidence/task-31-redirects.txt
  ```

  **Commit**: YES
  - Message: `client: add redirect handling — follow/no-follow/max-redirects config`
  - Files: `internal/client/client.go`, `internal/client/client_test.go`
  - Pre-commit: `go test ./internal/client/... -count=1`

- [x] 32. Proxy configuration

  **What to do**:
  - RED: Write tests: (1) request sent through HTTP proxy, (2) HTTPS proxy (CONNECT), (3) SOCKS5 proxy, (4) no-proxy list honored, (5) proxy from environment (HTTP_PROXY, HTTPS_PROXY)
  - GREEN: Add to `internal/client/`:
    - Add proxy config to Request or Client: `ProxyURL string`, `NoProxy []string`
    - Set `http.Transport.Proxy` based on config
    - Support: `http://proxy:8080`, `https://proxy:8080`, `socks5://proxy:1080`
    - Respect standard env vars: `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` (via `http.ProxyFromEnvironment`)
    - CLI flags: `--proxy <url>`, `--no-proxy <hosts>`
    - TOML config: `[proxy]` section for defaults
  - REFACTOR: Handle proxy auth (user:pass in URL)

  **Must NOT do**:
  - Ignore standard proxy environment variables
  - Implement custom proxy protocols (use Go's built-in support)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go HTTP transport proxy configuration

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 24-31, 33)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 4 (client)

  **References**:
  - `internal/client/client.go` — HTTP client transport configuration
  - `net/http` — `Transport.Proxy`, `ProxyFromEnvironment`
  - `golang.org/x/net/proxy` — SOCKS5 dialer support

  **Acceptance Criteria**:
  - [ ] `go test ./internal/client/... -v -run TestProxy -count=1` — ALL pass
  - [ ] HTTP proxy routes request through proxy
  - [ ] Environment proxy vars respected

  **QA Scenarios**:
  ```
  Scenario: HTTP proxy routing
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/client/... -v -run TestHTTPProxy -count=1
      2. Assert: exit code 0
    Expected Result: Request sent through proxy server
    Evidence: .sisyphus/evidence/task-32-proxy.txt
  ```

  **Commit**: YES
  - Message: `client: add proxy configuration — HTTP/HTTPS/SOCKS5, env vars, no-proxy list`
  - Files: `internal/client/client.go`, `internal/client/proxy.go`, `internal/client/proxy_test.go`
  - Pre-commit: `go test ./internal/client/... -count=1`

- [x] 33. Client certificates (mTLS) + SSL toggle

  **What to do**:
  - RED: Write tests: (1) mTLS with client cert+key authenticates to server, (2) `--insecure` skips TLS verification, (3) custom CA cert bundle, (4) TLS version pinning
  - GREEN: Add to `internal/client/`:
    - `TLSConfig` struct: `{CertFile, KeyFile, CAFile string, Insecure bool, MinTLSVersion string}`
    - Load cert/key via `tls.LoadX509KeyPair()`
    - Custom CA: load into `x509.CertPool` and set in `tls.Config.RootCAs`
    - `--insecure` / `-k`: set `tls.Config.InsecureSkipVerify = true`
    - `--cert`, `--key`, `--cacert` flags on run command
    - Configure `http.Transport.TLSClientConfig`
  - REFACTOR: Warn on `--insecure` usage

  **Must NOT do**:
  - Default to insecure (always verify TLS by default)
  - Skip cert validation silently (print warning)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go TLS configuration, x509 cert handling

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 24-32)
  - **Parallel Group**: Wave 4
  - **Blocks**: None
  - **Blocked By**: Task 4 (client)

  **References**:
  - `internal/client/client.go` — HTTP transport TLS config
  - `crypto/tls` — TLS config, LoadX509KeyPair
  - `crypto/x509` — CertPool for custom CA
  - curl flags: `-k`, `--cert`, `--key`, `--cacert`

  **Acceptance Criteria**:
  - [ ] `go test ./internal/client/... -v -run TestTLS -count=1` — ALL pass
  - [ ] mTLS authenticates with client cert
  - [ ] `--insecure` skips TLS verification with warning printed
  - [ ] Custom CA cert accepted

  **QA Scenarios**:
  ```
  Scenario: mTLS client certificate auth
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/client/... -v -run TestMTLS -count=1
      2. Assert: exit code 0
    Expected Result: Client cert presented, server accepts
    Evidence: .sisyphus/evidence/task-33-mtls.txt

  Scenario: Insecure mode skips verification
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/client/... -v -run TestInsecure -count=1
      2. Assert: exit code 0
    Expected Result: Self-signed cert accepted with warning
    Evidence: .sisyphus/evidence/task-33-insecure.txt
  ```

  **Commit**: YES
  - Message: `client: add mTLS, custom CA, insecure toggle — full TLS configuration`
  - Files: `internal/client/tls.go`, `internal/client/tls_test.go`
  - Pre-commit: `go test ./internal/client/... -count=1`

### Wave 5: Response Handling + Dynamic Values + GraphQL (Tasks 34-41)

- [ ] 34. Pretty Print JSON/XML/HTML Responses

  **What to do**:
  - **RED**: Write tests in `internal/formatter/formatter_test.go`:
    - `TestFormatJSON_PrettyPrint` — minified JSON → indented with 2-space indent
    - `TestFormatJSON_SyntaxHighlight` — output contains ANSI color codes for keys, strings, numbers, booleans, null
    - `TestFormatXML_PrettyPrint` — compact XML → indented
    - `TestFormatXML_SyntaxHighlight` — ANSI colors for tags, attributes, text
    - `TestFormatHTML_PrettyPrint` — minified HTML → indented
    - `TestAutoDetect_ContentType` — `application/json` → JSON formatter, `text/xml` → XML, `text/html` → HTML, `text/plain` → raw passthrough
    - `TestFormatJSON_InvalidInput` — returns raw input with warning, no panic
    - `TestFormatJSON_LargePayload` — 10MB JSON formatted without OOM
  - **GREEN**: Create `internal/formatter/formatter.go`:
    - `Format(body []byte, contentType string, opts FormatOptions) string` — auto-detect and dispatch
    - `FormatJSON(body []byte, opts FormatOptions) string` — use `encoding/json` Indent + custom ANSI colorizer
    - `FormatXML(body []byte, opts FormatOptions) string` — use `encoding/xml` Decoder + manual indenter
    - `FormatHTML(body []byte, opts FormatOptions) string` — lightweight tag-aware indenter (not full DOM parse)
    - `FormatOptions` struct: `Indent string`, `Color bool`, `MaxWidth int`
    - Use switch on content-type for dispatch — NO if-else chains
    - ANSI colors: keys=cyan, strings=green, numbers=yellow, booleans=magenta, null=red, XML tags=cyan, attributes=yellow
  - **REFACTOR**: Extract color theme to `internal/formatter/theme.go` for future TUI reuse

  **Must NOT do**:
  - Do NOT use external pretty-print libraries (chroma, glamour) — keep deps minimal
  - Do NOT parse HTML into full DOM — lightweight tag indentation only
  - Do NOT use if-else-if-else for content type dispatch — use switch

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single module, well-defined input/output, no external deps
  - **Skills**: [`bun-dev`]
    - `bun-dev`: Not directly applicable but no better match; Go stdlib work
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Formatter is a library, not CLI interaction

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 35, 36, 37, 38, 39)
  - **Blocks**: Task 40 (GraphQL needs formatted response display), TUI response viewer (Task 61)
  - **Blocked By**: Task 4 (HTTP client — needs response body to format)

  **References**:

  **Pattern References**:
  - `internal/core/template/engine.go` — Existing template engine shows Go text processing patterns in this codebase
  - `pkg/types/types.go:ExecutionHistory` — Has `Response string` field where formatted output gets stored

  **API/Type References**:
  - `internal/client/` (from Task 4) — `Response` struct with `Body []byte` and `ContentType string` fields
  - `encoding/json` stdlib — `json.Indent()` for JSON formatting
  - `encoding/xml` stdlib — `xml.NewDecoder()` for XML parsing

  **Test References**:
  - `internal/core/curl/parser_test.go` — Table-driven test pattern used in this codebase

  **External References**:
  - Go `encoding/json` docs: https://pkg.go.dev/encoding/json#Indent
  - ANSI escape codes reference: `\033[36m` (cyan), `\033[32m` (green), `\033[33m` (yellow), `\033[35m` (magenta), `\033[31m` (red), `\033[0m` (reset)

  **WHY Each Reference Matters**:
  - Template engine shows the codebase's string processing style
  - ExecutionHistory.Response shows where formatted output will be consumed
  - Client Response struct is the input contract for the formatter

  **Acceptance Criteria**:
  - [ ] `go test ./internal/formatter/... -v -count=1` → PASS (8+ tests)
  - [ ] `go vet ./internal/formatter/...` → clean

  **QA Scenarios**:

  ```
  Scenario: JSON pretty printing with colors
    Tool: Bash (go test + manual)
    Preconditions: Task 4 (HTTP client) complete
    Steps:
      1. Run: go test ./internal/formatter/... -v -run TestFormatJSON -count=1
      2. Assert: exit code 0, all TestFormatJSON* tests pass
      3. Run: echo '{"name":"test","count":42,"active":true,"data":null}' | go run cmd/gurl/main.go run --format json (or inline test)
      4. Assert: output contains indented JSON with ANSI color codes
    Expected Result: Pretty-printed, syntax-highlighted JSON output
    Failure Indicators: Raw/minified output, no colors, panic on edge cases
    Evidence: .sisyphus/evidence/task-34-json-pretty.txt

  Scenario: Invalid JSON returns raw input gracefully
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/formatter/... -v -run TestFormatJSON_InvalidInput -count=1
      2. Assert: exit code 0
    Expected Result: Invalid JSON returned as-is without panic
    Evidence: .sisyphus/evidence/task-34-invalid-json.txt

  Scenario: Auto-detect content type
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/formatter/... -v -run TestAutoDetect -count=1
      2. Assert: exit code 0, correct formatter selected for each content type
    Expected Result: JSON→JSON formatter, XML→XML formatter, HTML→HTML formatter, plain→passthrough
    Evidence: .sisyphus/evidence/task-34-autodetect.txt
  ```

  **Commit**: YES
  - Message: `formatter: add JSON/XML/HTML pretty printing with syntax highlighting — response readability`
  - Files: `internal/formatter/formatter.go`, `internal/formatter/formatter_test.go`, `internal/formatter/theme.go`
  - Pre-commit: `go test ./internal/formatter/... -count=1`

- [ ] 35. JSONPath and XPath Response Filtering

  **What to do**:
  - **RED**: Write tests in `internal/formatter/filter_test.go`:
    - `TestJSONPath_SimpleKey` — `$.name` extracts string value
    - `TestJSONPath_NestedPath` — `$.data.users[0].email` extracts nested value
    - `TestJSONPath_ArraySlice` — `$.items[0:3]` extracts slice
    - `TestJSONPath_Wildcard` — `$.users[*].name` extracts all names
    - `TestJSONPath_Filter` — `$.users[?(@.age > 18)]` filters array
    - `TestXPath_SimpleElement` — `//title` extracts element text
    - `TestXPath_Attribute` — `//book[@category='fiction']` filters by attribute
    - `TestJSONPath_InvalidPath` — returns error with descriptive message
    - `TestJSONPath_NoMatch` — returns empty result, no error
  - **GREEN**: Create `internal/formatter/filter.go`:
    - `FilterJSON(body []byte, path string) (string, error)` — JSONPath extraction
    - `FilterXML(body []byte, xpath string) (string, error)` — XPath extraction
    - Use `github.com/PaesslerAG/jsonpath` for JSONPath (lightweight, well-maintained)
    - Use `github.com/antchfx/xmlquery` for XPath
    - Output: filtered result pretty-printed (reuse Task 34 formatter)
  - **REFACTOR**: Add `--filter` flag support to `run` command for piping: `gurl run "my-api" --filter '$.data.users[*].name'`

  **Must NOT do**:
  - Do NOT implement custom JSONPath parser — use established library
  - Do NOT use if-else for path type detection — detect `$` prefix = JSONPath, `/` or `//` prefix = XPath

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Straightforward library integration with clear input/output
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Flag wiring is minor, not the core task

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 34, 36, 37, 38, 39)
  - **Blocks**: Assertion engine (Task 49 — needs JSONPath for response assertions)
  - **Blocked By**: Task 34 (formatter — reuses pretty-print for output)

  **References**:

  **Pattern References**:
  - `internal/formatter/formatter.go` (Task 34) — Reuse `FormatJSON` for output of filtered results
  - `internal/cli/commands/run.go` — Where `--filter` flag will be wired

  **API/Type References**:
  - `github.com/PaesslerAG/jsonpath` — `jsonpath.Get(path, data)` returns `interface{}`
  - `github.com/antchfx/xmlquery` — `xmlquery.QueryAll(doc, xpath)` returns `[]*Node`

  **External References**:
  - JSONPath spec: https://goessner.net/articles/JsonPath/
  - PaesslerAG/jsonpath: https://github.com/PaesslerAG/jsonpath
  - antchfx/xmlquery: https://github.com/antchfx/xmlquery

  **WHY Each Reference Matters**:
  - PaesslerAG/jsonpath is the most lightweight Go JSONPath lib (no heavy deps)
  - antchfx/xmlquery handles XPath 1.0 which covers all competitor features
  - run.go is where --filter flag gets wired for CLI usage

  **Acceptance Criteria**:
  - [ ] `go test ./internal/formatter/... -v -run TestJSONPath -count=1` → PASS
  - [ ] `go test ./internal/formatter/... -v -run TestXPath -count=1` → PASS
  - [ ] `go mod tidy` → clean (new deps added properly)

  **QA Scenarios**:

  ```
  Scenario: JSONPath nested extraction
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/formatter/... -v -run TestJSONPath_NestedPath -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/formatter/... -v -run TestJSONPath_ArraySlice -count=1
      4. Assert: exit code 0
    Expected Result: Nested values and array slices extracted correctly
    Evidence: .sisyphus/evidence/task-35-jsonpath.txt

  Scenario: Invalid path returns descriptive error
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/formatter/... -v -run TestJSONPath_InvalidPath -count=1
      2. Assert: exit code 0, error message contains path info
    Expected Result: Error says what went wrong, not generic "invalid path"
    Evidence: .sisyphus/evidence/task-35-invalid-path.txt

  Scenario: XPath element extraction
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/formatter/... -v -run TestXPath -count=1
      2. Assert: exit code 0
    Expected Result: XML elements and attributes filtered correctly
    Evidence: .sisyphus/evidence/task-35-xpath.txt
  ```

  **Commit**: YES
  - Message: `formatter: add JSONPath and XPath response filtering — query API responses inline`
  - Files: `internal/formatter/filter.go`, `internal/formatter/filter_test.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/formatter/... -count=1`

- [ ] 36. Response Body Diff

  **What to do**:
  - **RED**: Write tests in `internal/formatter/diff_test.go`:
    - `TestDiffJSON_FieldAdded` — new field highlighted in green
    - `TestDiffJSON_FieldRemoved` — removed field highlighted in red
    - `TestDiffJSON_FieldChanged` — changed value shows old→new
    - `TestDiffJSON_DeepNested` — nested object changes detected
    - `TestDiffJSON_ArrayReorder` — array element changes detected
    - `TestDiffText_LineDiff` — plain text unified diff format
    - `TestDiffIdentical` — returns "no differences" message
  - **GREEN**: Create `internal/formatter/diff.go`:
    - `DiffJSON(a, b []byte) (string, error)` — semantic JSON diff (key-aware, not line-based)
    - `DiffText(a, b []byte) string` — unified diff format for non-JSON
    - `DiffResponses(histA, histB ExecutionHistory) (string, error)` — compare two saved responses
    - JSON diff: normalize (sort keys, consistent formatting) then compare paths
    - Use `github.com/wI2L/jsondiff` for RFC 6902 JSON Patch format, render as colorized output
    - Text diff: use `github.com/sergi/go-diff/diffmatchpatch` for character-level diff
  - **REFACTOR**: Wire into existing `diff` command (`internal/cli/commands/diff.go`) — replace current stub with actual implementation

  **Must NOT do**:
  - Do NOT use line-based diff for JSON — must be key-aware semantic diff
  - Do NOT show full response when only a few fields changed — show only diffs

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Semantic JSON diff has subtle edge cases (arrays, nested objects, type changes)
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Diff logic is the hard part, not CLI wiring

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 34, 35, 37, 38, 39)
  - **Blocks**: None directly (diff is a standalone feature)
  - **Blocked By**: Task 9 (response storage — needs stored responses to diff)

  **References**:

  **Pattern References**:
  - `internal/cli/commands/diff.go` — Existing diff command stub to replace with real implementation
  - `internal/formatter/formatter.go` (Task 34) — Reuse ANSI color theme for diff output

  **API/Type References**:
  - `pkg/types/types.go:ExecutionHistory` — `Response string`, `RequestID string` for loading two responses
  - `internal/storage/db.go` — `GetHistory(id)` to load execution records

  **External References**:
  - wI2L/jsondiff: https://github.com/wI2L/jsondiff — RFC 6902 JSON Patch
  - sergi/go-diff: https://github.com/sergi/go-diff — Character-level text diff

  **WHY Each Reference Matters**:
  - Existing diff.go is the wiring point — read it to understand current CLI contract
  - jsondiff produces RFC 6902 patches which are semantic (path-based, not line-based)
  - ExecutionHistory is the source of stored responses to compare

  **Acceptance Criteria**:
  - [ ] `go test ./internal/formatter/... -v -run TestDiff -count=1` → PASS (7+ tests)
  - [ ] `gurl diff "req-a" "req-b"` produces colored semantic diff output

  **QA Scenarios**:

  ```
  Scenario: JSON semantic diff with field changes
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/formatter/... -v -run TestDiffJSON -count=1
      2. Assert: exit code 0, all diff tests pass
    Expected Result: Added fields green, removed red, changed shows old→new with path
    Evidence: .sisyphus/evidence/task-36-json-diff.txt

  Scenario: Identical responses show no diff
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/formatter/... -v -run TestDiffIdentical -count=1
      2. Assert: exit code 0, output contains "no differences"
    Expected Result: Clean "no differences" message, not empty output
    Evidence: .sisyphus/evidence/task-36-identical.txt
  ```

  **Commit**: YES
  - Message: `formatter: add semantic JSON diff and text diff — compare API responses meaningfully`
  - Files: `internal/formatter/diff.go`, `internal/formatter/diff_test.go`, `internal/cli/commands/diff.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/formatter/... -count=1`

- [ ] 37. Save Response to File

  **What to do**:
  - **RED**: Write tests in `internal/client/output_test.go`:
    - `TestSaveResponse_JSON` — saves JSON body to file, file content matches
    - `TestSaveResponse_Binary` — saves binary (image/pdf) without corruption
    - `TestSaveResponse_AutoFilename` — derives filename from URL path + Content-Disposition header
    - `TestSaveResponse_CustomPath` — saves to user-specified path
    - `TestSaveResponse_CreateDirs` — creates parent directories if missing
    - `TestSaveResponse_ExistingFile` — returns error (no silent overwrite) unless `--force`
    - `TestSaveResponse_Stdout` — `-o -` writes to stdout for piping
  - **GREEN**: Create `internal/client/output.go`:
    - `SaveToFile(resp *Response, path string, force bool) error`
    - `DeriveFilename(resp *Response) string` — from Content-Disposition or URL last segment
    - Wire `--output` / `-o` flag to `run` command
    - Handle `--output -` for stdout piping (useful for `gurl run "img" -o - | feh -`)
  - **REFACTOR**: Ensure binary content passes through without encoding issues (no UTF-8 assumption)

  **Must NOT do**:
  - Do NOT silently overwrite existing files — require `--force` flag
  - Do NOT assume response is text — handle binary correctly

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: File I/O with straightforward logic, no complex algorithms
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: File save is simple I/O, not CLI architecture

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 34, 35, 36, 38, 39)
  - **Blocks**: None directly
  - **Blocked By**: Task 4 (HTTP client — needs Response struct with Body bytes)

  **References**:

  **Pattern References**:
  - `internal/cli/commands/run.go` — Where `--output` flag gets added
  - `internal/cli/commands/export.go` — Shows file writing pattern in this codebase

  **API/Type References**:
  - `internal/client/` (Task 4) — `Response` struct with `Body []byte`, `Headers map[string][]string`
  - `os.MkdirAll`, `os.WriteFile` — stdlib for file operations

  **WHY Each Reference Matters**:
  - run.go is where the --output flag is wired into the CLI
  - Response.Body is the raw bytes to save (binary-safe)
  - Content-Disposition header parsing needed for auto-filename

  **Acceptance Criteria**:
  - [ ] `go test ./internal/client/... -v -run TestSaveResponse -count=1` → PASS (7 tests)
  - [ ] Binary files (images) saved without corruption

  **QA Scenarios**:

  ```
  Scenario: Save JSON response to file
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/client/... -v -run TestSaveResponse_JSON -count=1
      2. Assert: exit code 0
    Expected Result: File created with exact JSON content
    Evidence: .sisyphus/evidence/task-37-save-json.txt

  Scenario: Existing file without --force returns error
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/client/... -v -run TestSaveResponse_ExistingFile -count=1
      2. Assert: exit code 0, error message mentions "already exists" and "--force"
    Expected Result: No silent overwrite, clear error with fix instructions
    Evidence: .sisyphus/evidence/task-37-no-overwrite.txt
  ```

  **Commit**: YES
  - Message: `client: add response-to-file with auto-naming and binary support — save API responses`
  - Files: `internal/client/output.go`, `internal/client/output_test.go`, `internal/cli/commands/run.go`
  - Pre-commit: `go test ./internal/client/... -count=1`

- [ ] 38. Dynamic Template Values (UUID, Timestamp, Random)

  **What to do**:
  - **RED**: Write tests in `internal/core/template/dynamic_test.go`:
    - `TestDynamic_UUID` — `{{$uuid}}` produces valid UUID v4 format
    - `TestDynamic_Timestamp` — `{{$timestamp}}` produces Unix epoch integer
    - `TestDynamic_ISOTimestamp` — `{{$isoTimestamp}}` produces ISO 8601 string
    - `TestDynamic_RandomInt` — `{{$randomInt(1, 100)}}` produces int in range
    - `TestDynamic_RandomString` — `{{$randomString(16)}}` produces alphanumeric of length 16
    - `TestDynamic_RandomEmail` — `{{$randomEmail}}` produces valid email format
    - `TestDynamic_RandomUUID` — each call produces different value (non-deterministic check via 100 iterations)
    - `TestDynamic_UnknownFunction` — `{{$unknown}}` returns error, not empty string
  - **GREEN**: Create `internal/core/template/dynamic.go`:
    - Register dynamic value generators in template engine
    - `$uuid` → `uuid.New().String()` (use `github.com/google/uuid`)
    - `$timestamp` → `time.Now().Unix()`
    - `$isoTimestamp` → `time.Now().UTC().Format(time.RFC3339)`
    - `$randomInt(min, max)` → `crypto/rand` based random integer
    - `$randomString(len)` → alphanumeric string of given length
    - `$randomEmail` → `{randomString(8)}@example.com`
    - Use switch on function name for dispatch
  - **REFACTOR**: Integrate with existing template engine in `internal/core/template/engine.go` — add dynamic resolver as pre-pass before variable substitution

  **Must NOT do**:
  - Do NOT use `math/rand` — use `crypto/rand` for unpredictable values
  - Do NOT silently return empty string for unknown functions — return error

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple value generators, clear contracts, single module
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - None applicable

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 34, 35, 36, 37, 39)
  - **Blocks**: Pre-request scripts (Task 45 — scripts can call dynamic generators)
  - **Blocked By**: None (template engine already exists)

  **References**:

  **Pattern References**:
  - `internal/core/template/engine.go` — Existing template engine to integrate with — read `Render()` method to understand variable substitution pipeline
  - `internal/core/template/engine_test.go` — Existing test patterns for template functionality

  **API/Type References**:
  - `github.com/google/uuid` — `uuid.New().String()` for UUID v4 generation
  - `crypto/rand` stdlib — `rand.Int()` for cryptographic random integers
  - `pkg/types/types.go:Var` — Variable struct with `Name`, `Pattern`, `Example` fields

  **External References**:
  - Postman dynamic variables: https://learning.postman.com/docs/writing-scripts/script-references/variables-list/ — reference for which dynamic values competitors support

  **WHY Each Reference Matters**:
  - engine.go is the integration point — dynamic values must hook into the existing Render() pipeline
  - Postman's dynamic variable list shows what users expect (we support the most common ones)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/core/template/... -v -count=1` → PASS (all existing + 8 new tests)
  - [ ] Dynamic values resolve correctly inside saved request templates

  **QA Scenarios**:

  ```
  Scenario: UUID and timestamp generation
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/core/template/... -v -run TestDynamic -count=1
      2. Assert: exit code 0, all TestDynamic* tests pass
    Expected Result: UUID matches v4 format, timestamp is current epoch, ISO format is valid
    Evidence: .sisyphus/evidence/task-38-dynamic-values.txt

  Scenario: Unknown dynamic function returns error
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/core/template/... -v -run TestDynamic_UnknownFunction -count=1
      2. Assert: exit code 0, error mentions unknown function name
    Expected Result: Clear error, not silent empty string
    Evidence: .sisyphus/evidence/task-38-unknown-func.txt
  ```

  **Commit**: YES
  - Message: `template: add dynamic values (UUID, timestamp, random) — Postman-compatible template functions`
  - Files: `internal/core/template/dynamic.go`, `internal/core/template/dynamic_test.go`, `internal/core/template/engine.go`
  - Pre-commit: `go test ./internal/core/template/... -count=1`

  - Pre-commit: `go test ./internal/core/template/... -count=1`

- [ ] 39. Path Parameters Support

  **What to do**:
  - **RED**: Write tests in `internal/core/template/pathparam_test.go`:
    - `TestPathParam_SingleParam` — `https://api.com/users/:id` with `id=123` → `https://api.com/users/123`
    - `TestPathParam_MultipleParams` — `/users/:userId/posts/:postId` resolves both
    - `TestPathParam_ColonAndBrace` — supports both `:id` and `{id}` syntax
    - `TestPathParam_UnresolvedParam` — returns error listing unresolved params
    - `TestPathParam_URLEncoding` — special chars in param value get URL-encoded
    - `TestPathParam_EmptyValue` — empty string param value → error, not silent empty segment
  - **GREEN**: Create `internal/core/template/pathparam.go`:
    - `ResolvePathParams(url string, params map[string]string) (string, error)`
    - Parse URL for `:param` and `{param}` placeholders
    - Replace with URL-encoded values from params map
    - Return error for any unresolved parameters
    - Add `PathParams []Var` field to `SavedRequest` type
  - **REFACTOR**: Wire into template engine pipeline — path params resolve BEFORE query params

  **Must NOT do**:
  - Do NOT silently ignore unresolved path params — always error
  - Do NOT double-encode already-encoded values

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: String manipulation with clear rules, single module
  - **Skills**: []
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 34-38, 40)
  - **Blocks**: None directly
  - **Blocked By**: None (template engine already exists)

  **References**:

  **Pattern References**:
  - `internal/core/template/engine.go` — Integration point for path param resolution
  - `pkg/types/types.go:Var` — Variable struct pattern to follow for PathParams

  **API/Type References**:
  - `net/url` stdlib — `url.PathEscape()` for proper URL encoding
  - `pkg/types/types.go:SavedRequest` — Where `PathParams` field gets added

  **WHY Each Reference Matters**:
  - engine.go Render() pipeline determines WHERE path params resolve in the chain
  - Var struct shows the naming pattern used for template variables in this project

  **Acceptance Criteria**:
  - [ ] `go test ./internal/core/template/... -v -run TestPathParam -count=1` → PASS (6 tests)
  - [ ] Both `:id` and `{id}` syntax supported

  **QA Scenarios**:

  ```
  Scenario: Path param resolution with both syntaxes
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/core/template/... -v -run TestPathParam -count=1
      2. Assert: exit code 0, all TestPathParam* pass
    Expected Result: Both :id and {id} resolved, URL-encoded, unresolved → error
    Evidence: .sisyphus/evidence/task-39-pathparams.txt

  Scenario: Unresolved param produces clear error
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/core/template/... -v -run TestPathParam_UnresolvedParam -count=1
      2. Assert: exit code 0, error message lists the missing param name
    Expected Result: Error says "unresolved path parameter: :userId"
    Evidence: .sisyphus/evidence/task-39-unresolved.txt
  ```

  **Commit**: YES
  - Message: `template: add path parameter support (:id and {id} syntax) — URL param substitution`
  - Files: `internal/core/template/pathparam.go`, `internal/core/template/pathparam_test.go`, `pkg/types/types.go`
  - Pre-commit: `go test ./internal/core/template/... -count=1`

- [ ] 40. Request Timeout Configuration

  **What to do**:
  - **RED**: Write tests in `internal/client/timeout_test.go`:
    - `TestTimeout_Default` — default timeout is 30s (from config)
    - `TestTimeout_PerRequest` — per-request timeout overrides default
    - `TestTimeout_Zero` — zero timeout means no timeout (infinite)
    - `TestTimeout_Exceeded` — request exceeding timeout returns clear timeout error
    - `TestTimeout_ConnectVsTotal` — separate connect timeout and total timeout
    - `TestTimeout_FromConfig` — reads `[general] timeout = "10s"` from TOML config
  - **GREEN**: Add to `internal/client/client.go`:
    - `WithTimeout(total time.Duration)` option
    - `WithConnectTimeout(connect time.Duration)` option
    - Wire timeouts to `http.Client.Timeout` and `net.Dialer.Timeout`
    - Add `--timeout` flag to `run` command (e.g., `gurl run "api" --timeout 5s`)
    - Add `timeout` field to `SavedRequest` type for per-request defaults
    - Read default from config `[general] timeout`
  - **REFACTOR**: Ensure timeout error message is user-friendly: "Request timed out after 5s" not raw Go error

  **Must NOT do**:
  - Do NOT use hardcoded timeout — always configurable
  - Do NOT swallow timeout errors — surface them clearly

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard HTTP client configuration, well-documented Go patterns
  - **Skills**: []
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 34-39, 41)
  - **Blocks**: None directly
  - **Blocked By**: Task 4 (HTTP client — timeout is a client option)

  **References**:

  **Pattern References**:
  - `internal/client/client.go` (Task 4) — Where timeout options get added to the client
  - `internal/config/` — Config loader for reading `[general] timeout` default

  **API/Type References**:
  - `http.Client{Timeout: ...}` — Go stdlib total request timeout
  - `net.Dialer{Timeout: ...}` — Connection establishment timeout
  - `time.ParseDuration("5s")` — Parse user-provided timeout strings
  - `pkg/types/types.go:Config.General` — Where `Timeout string` config field goes

  **WHY Each Reference Matters**:
  - client.go functional options pattern (WithTimeout) integrates cleanly with existing design
  - Config General section is where timeout default lives

  **Acceptance Criteria**:
  - [ ] `go test ./internal/client/... -v -run TestTimeout -count=1` → PASS (6 tests)
  - [ ] `gurl run "api" --timeout 5s` respects the timeout

  **QA Scenarios**:

  ```
  Scenario: Timeout configuration and error message
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/client/... -v -run TestTimeout -count=1
      2. Assert: exit code 0, all timeout tests pass
    Expected Result: Timeout exceeded → "Request timed out after 5s", not raw context.DeadlineExceeded
    Evidence: .sisyphus/evidence/task-40-timeout.txt

  Scenario: Per-request timeout overrides default
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/client/... -v -run TestTimeout_PerRequest -count=1
      2. Assert: exit code 0
    Expected Result: Per-request 2s timeout used instead of default 30s
    Evidence: .sisyphus/evidence/task-40-per-request.txt
  ```

  **Commit**: YES
  - Message: `client: add configurable request timeouts (connect + total) — prevent hanging requests`
  - Files: `internal/client/client.go`, `internal/client/timeout_test.go`, `internal/cli/commands/run.go`, `pkg/types/types.go`
  - Pre-commit: `go test ./internal/client/... -count=1`

- [ ] 41. GraphQL Client

  **What to do**:
  - **RED**: Write tests in `internal/protocols/graphql/graphql_test.go`:
    - `TestGraphQL_Query` — sends query string, receives JSON response
    - `TestGraphQL_QueryWithVariables` — query + variables JSON → correct POST body
    - `TestGraphQL_Mutation` — mutation string executes correctly
    - `TestGraphQL_Introspection` — `__schema` query returns schema
    - `TestGraphQL_ErrorResponse` — GraphQL error format (errors array) parsed and displayed
    - `TestGraphQL_Headers` — auth headers passed through to GraphQL endpoint
    - `TestGraphQL_BuildRequestBody` — query + variables + operationName → correct JSON structure
    - `TestGraphQL_MultilineQuery` — multiline query with fragments handled
  - **GREEN**: Create `internal/protocols/graphql/`:
    - `graphql.go`:
      - `type Client struct` — wraps HTTP client (from Task 4)
      - `type Request struct { Query string; Variables map[string]interface{}; OperationName string }`
      - `type Response struct { Data json.RawMessage; Errors []GraphQLError }`
      - `type GraphQLError struct { Message string; Locations []Location; Path []interface{} }`
      - `Execute(ctx context.Context, endpoint string, req Request, opts ...Option) (*Response, error)`
      - Builds POST request with `Content-Type: application/json`, body = `{"query": "...", "variables": {...}, "operationName": "..."}`
      - Parse response into Data + Errors
    - `cli.go`:
      - Add `gurl graphql` subcommand: `gurl graphql "endpoint-name" --query 'query { users { name } }' --vars '{"limit": 10}'`
      - Support `--query-file` for loading query from `.graphql` file
      - Add `graphql` as request type in SavedRequest (new `Protocol` field)
  - **REFACTOR**: Wire GraphQL response through formatter (Task 34) for pretty-printed output; extract GraphQL errors into readable format

  **Must NOT do**:
  - Do NOT add graphql-specific HTTP client — reuse `internal/client` from Task 4
  - Do NOT implement subscription support yet (that's WebSocket, Task 43)
  - Do NOT use external GraphQL client libraries — it's just POST with JSON body

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Protocol implementation with error handling edge cases, CLI wiring, type design
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `hono-routing`: Server-side, not client-side GraphQL

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 34-40)
  - **Blocks**: GraphQL subscriptions (via WebSocket, Task 43)
  - **Blocked By**: Task 4 (HTTP client), Task 34 (formatter for response display)

  **References**:

  **Pattern References**:
  - `internal/client/client.go` (Task 4) — HTTP client to wrap for GraphQL requests
  - `internal/formatter/formatter.go` (Task 34) — Pretty-print GraphQL JSON responses
  - `internal/cli/commands/run.go` — Pattern for adding new subcommands

  **API/Type References**:
  - `pkg/types/types.go:SavedRequest` — Add `Protocol string` field (values: "http", "graphql", "grpc", "ws", "sse")
  - `encoding/json` — Marshal/Unmarshal GraphQL request/response bodies

  **External References**:
  - GraphQL over HTTP spec: https://graphql.github.io/graphql-over-http/draft/ — POST with application/json
  - GraphQL error format: https://spec.graphql.org/October2021/#sec-Errors — `errors` array structure

  **WHY Each Reference Matters**:
  - GraphQL over HTTP spec defines the exact POST body format and response structure
  - Error format spec ensures we parse competitor-compatible error responses
  - SavedRequest needs Protocol field to distinguish request types for storage and execution

  **Acceptance Criteria**:
  - [ ] `go test ./internal/protocols/graphql/... -v -count=1` → PASS (8 tests)
  - [ ] `gurl graphql "my-api" --query 'query { users { name } }'` executes and returns formatted response
  - [ ] GraphQL errors displayed in readable format (not raw JSON)

  **QA Scenarios**:

  ```
  Scenario: GraphQL query execution
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/graphql/... -v -run TestGraphQL_Query -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/protocols/graphql/... -v -run TestGraphQL_QueryWithVariables -count=1
      4. Assert: exit code 0
    Expected Result: Query sent as POST with correct JSON body, response parsed
    Evidence: .sisyphus/evidence/task-41-graphql-query.txt

  Scenario: GraphQL error response parsing
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/graphql/... -v -run TestGraphQL_ErrorResponse -count=1
      2. Assert: exit code 0, errors parsed into structured format
    Expected Result: GraphQL errors array parsed, message + location displayed
    Evidence: .sisyphus/evidence/task-41-graphql-errors.txt

  Scenario: Request body structure
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/graphql/... -v -run TestGraphQL_BuildRequestBody -count=1
      2. Assert: exit code 0, body matches {"query":"...","variables":{...},"operationName":"..."}
    Expected Result: Exact JSON structure per GraphQL over HTTP spec
    Evidence: .sisyphus/evidence/task-41-graphql-body.txt
  ```

  **Commit**: YES
  - Message: `protocols: add GraphQL client with query/mutation/introspection — first protocol handler`
  - Files: `internal/protocols/graphql/graphql.go`, `internal/protocols/graphql/graphql_test.go`, `internal/protocols/graphql/cli.go`, `pkg/types/types.go`, `cmd/gurl/main.go`
  - Pre-commit: `go test ./internal/protocols/graphql/... -count=1`

  - Pre-commit: `go test ./internal/protocols/graphql/... -count=1`

### Wave 6: Remaining Protocols + JavaScript Scripting (Tasks 42-48)

- [ ] 42. gRPC Client

  **What to do**:
  - **RED**: Write tests in `internal/protocols/grpc/grpc_test.go`:
    - `TestGRPC_UnaryCall` — sends unary request, receives response
    - `TestGRPC_ServerStreaming` — receives stream of messages
    - `TestGRPC_ClientStreaming` — sends stream, receives single response
    - `TestGRPC_BidirectionalStreaming` — full duplex streaming
    - `TestGRPC_Reflection` — discovers services via server reflection
    - `TestGRPC_ProtoFromFile` — loads .proto file for request/response types
    - `TestGRPC_Metadata` — sends/receives gRPC metadata (headers)
    - `TestGRPC_ErrorCodes` — gRPC status codes mapped to readable names
    - `TestGRPC_TLS` — connects with TLS credentials
  - **GREEN**: Create `internal/protocols/grpc/`:
    - `grpc.go`:
      - `type Client struct` — wraps `google.golang.org/grpc` connection
      - `Dial(target string, opts ...grpc.DialOption) error`
      - `Invoke(ctx context.Context, method string, req, resp proto.Message) error` — unary
      - `InvokeServerStream(ctx, method, req) (stream, error)` — server streaming
      - `InvokeClientStream(ctx, method) (stream, error)` — client streaming
      - `InvokeBidiStream(ctx, method) (stream, error)` — bidirectional
    - `reflection.go`:
      - `DiscoverServices(ctx, conn) ([]ServiceInfo, error)` — via gRPC server reflection
      - `BuildMessage(serviceInfo, methodName, jsonPayload) (proto.Message, error)` — dynamic message from JSON
    - `cli.go`:
      - `gurl grpc "target" --service "UserService" --method "GetUser" --data '{"id": 1}'`
      - `gurl grpc "target" --list` — list services via reflection
      - `gurl grpc "target" --proto ./user.proto --method "GetUser" --data '{"id": 1}'`
  - **REFACTOR**: Format gRPC responses as JSON for display (reuse formatter), display metadata separately

  **Must NOT do**:
  - Do NOT implement protobuf compilation — use dynamic messages or pre-compiled proto
  - Do NOT require .proto file if server supports reflection
  - Do NOT use if-else for call type dispatch — switch on call type enum

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: gRPC is complex — streaming, reflection, proto handling, TLS. Needs careful implementation.
  - **Skills**: [`golang-pro`]
    - `golang-pro`: gRPC with Go patterns, concurrent streaming, channel handling
  - **Skills Evaluated but Omitted**:
    - `golang-testing`: Covered by TDD approach in task spec

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 6 (with Tasks 43, 44, 45)
  - **Blocks**: None directly (gRPC is standalone protocol)
  - **Blocked By**: Task 4 (HTTP client patterns), Task 33 (TLS config reuse)

  **References**:

  **Pattern References**:
  - `internal/protocols/graphql/graphql.go` (Task 41) — Protocol handler pattern to follow
  - `internal/client/tls.go` (Task 33) — TLS config to reuse for gRPC TLS connections

  **API/Type References**:
  - `google.golang.org/grpc` — `grpc.Dial()`, `grpc.WithTransportCredentials()`
  - `google.golang.org/grpc/reflection/grpc_reflection_v1alpha` — Server reflection
  - `google.golang.org/protobuf/types/dynamicpb` — Dynamic protobuf messages (no .proto file needed)
  - `github.com/jhump/protoreflect` — Dynamic gRPC invocation with reflection

  **External References**:
  - gRPC Go quickstart: https://grpc.io/docs/languages/go/quickstart/
  - jhump/protoreflect: https://github.com/jhump/protoreflect — Used by grpcurl, battle-tested
  - grpcurl (reference impl): https://github.com/fullstorydev/grpcurl — CLI gRPC client to model after

  **WHY Each Reference Matters**:
  - protoreflect is how grpcurl works — proven approach for dynamic gRPC without proto files
  - GraphQL handler establishes the protocol handler pattern (Client + CLI wiring)
  - grpcurl source shows the correct way to handle reflection + dynamic invocation

  **Acceptance Criteria**:
  - [ ] `go test ./internal/protocols/grpc/... -v -count=1` → PASS (9 tests)
  - [ ] `gurl grpc "localhost:50051" --list` discovers services
  - [ ] `gurl grpc "localhost:50051" --method "GetUser" --data '{"id":1}'` returns formatted response
  - [ ] `go mod tidy` → clean (grpc deps added)

  **QA Scenarios**:

  ```
  Scenario: gRPC unary call with reflection
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/grpc/... -v -run TestGRPC_UnaryCall -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/protocols/grpc/... -v -run TestGRPC_Reflection -count=1
      4. Assert: exit code 0
    Expected Result: Unary call succeeds, reflection discovers services
    Evidence: .sisyphus/evidence/task-42-grpc-unary.txt

  Scenario: gRPC error codes readable
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/grpc/... -v -run TestGRPC_ErrorCodes -count=1
      2. Assert: exit code 0, status codes mapped to names (NOT_FOUND, PERMISSION_DENIED, etc.)
    Expected Result: gRPC status codes shown as human-readable names
    Evidence: .sisyphus/evidence/task-42-grpc-errors.txt

  Scenario: Server streaming receives all messages
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/grpc/... -v -run TestGRPC_ServerStreaming -count=1
      2. Assert: exit code 0, all stream messages collected
    Expected Result: Multiple messages received and displayed sequentially
    Evidence: .sisyphus/evidence/task-42-grpc-streaming.txt
  ```

  **Commit**: YES
  - Message: `protocols: add gRPC client with reflection, streaming, and dynamic messages — second protocol handler`
  - Files: `internal/protocols/grpc/grpc.go`, `internal/protocols/grpc/reflection.go`, `internal/protocols/grpc/grpc_test.go`, `internal/protocols/grpc/cli.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/protocols/grpc/... -count=1`

- [ ] 43. WebSocket Client

  **What to do**:
  - **RED**: Write tests in `internal/protocols/websocket/ws_test.go`:
    - `TestWS_Connect` — establishes connection to ws:// endpoint
    - `TestWS_ConnectTLS` — establishes connection to wss:// endpoint
    - `TestWS_SendText` — sends text message, receives echo
    - `TestWS_SendJSON` — sends JSON, receives JSON response
    - `TestWS_ReceiveMultiple` — receives multiple messages (streaming display)
    - `TestWS_Close` — graceful close with close frame
    - `TestWS_Reconnect` — auto-reconnect on unexpected disconnect (configurable)
    - `TestWS_Headers` — custom headers (auth, cookies) sent on handshake
    - `TestWS_Ping` — ping/pong keepalive
    - `TestWS_BinaryMessage` — binary frames handled (displayed as hex or saved to file)
  - **GREEN**: Create `internal/protocols/websocket/`:
    - `ws.go`:
      - `type Client struct` — wraps `github.com/gorilla/websocket`
      - `Connect(ctx context.Context, url string, headers http.Header) error`
      - `Send(msg []byte, msgType int) error`
      - `Receive() ([]byte, int, error)` — blocking receive
      - `Close() error` — graceful close
      - `SetReconnect(enabled bool, maxRetries int, backoff time.Duration)`
    - `interactive.go`:
      - Interactive mode: reads from stdin, sends to WS, prints received messages
      - Uses goroutines: one for reading stdin→send, one for receive→stdout
      - Ctrl+C for graceful disconnect
    - `cli.go`:
      - `gurl ws "ws://localhost:8080/socket" --header "Authorization: Bearer token"`
      - `gurl ws "ws://localhost:8080/socket" --send '{"action":"subscribe","channel":"updates"}'`
      - Interactive mode by default (like websocat), `--send` for one-shot
  - **REFACTOR**: Wire JSON messages through formatter for pretty-printed display

  **Must NOT do**:
  - Do NOT implement custom WebSocket frame parser — use gorilla/websocket
  - Do NOT block main goroutine on receive — use separate goroutines for send/receive
  - Do NOT ignore close frames — handle graceful shutdown

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Concurrent send/receive goroutines, reconnection logic, graceful shutdown — concurrency patterns
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Goroutine patterns, channel-based communication, concurrent I/O
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Interactive mode is important but goroutine management is the hard part

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 6 (with Tasks 42, 44, 45)
  - **Blocks**: GraphQL subscriptions (future, not in this plan)
  - **Blocked By**: Task 4 (HTTP client — handshake uses HTTP), Task 33 (TLS config)

  **References**:

  **Pattern References**:
  - `internal/protocols/graphql/graphql.go` (Task 41) — Protocol handler pattern
  - `internal/protocols/graphql/cli.go` (Task 41) — CLI subcommand wiring pattern

  **API/Type References**:
  - `github.com/gorilla/websocket` — `websocket.Dialer{}`, `conn.ReadMessage()`, `conn.WriteMessage()`
  - `pkg/types/types.go:SavedRequest` — `Protocol: "ws"` for saved WebSocket connections

  **External References**:
  - gorilla/websocket: https://github.com/gorilla/websocket — De facto Go WebSocket library
  - websocat (reference): https://github.com/vi/websocat — CLI WebSocket tool to model interactive mode after

  **WHY Each Reference Matters**:
  - gorilla/websocket handles frame parsing, TLS, compression — don't reinvent
  - websocat's interactive mode (stdin→ws, ws→stdout) is the UX model to follow
  - Protocol handler pattern from GraphQL ensures consistent CLI interface

  **Acceptance Criteria**:
  - [ ] `go test ./internal/protocols/websocket/... -v -count=1` → PASS (10 tests)
  - [ ] `gurl ws "ws://echo.websocket.org"` connects and echoes in interactive mode
  - [ ] Ctrl+C produces graceful close frame, not abrupt disconnect

  **QA Scenarios**:

  ```
  Scenario: WebSocket connect and text exchange
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/websocket/... -v -run TestWS_Connect -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/protocols/websocket/... -v -run TestWS_SendText -count=1
      4. Assert: exit code 0
    Expected Result: Connection established, text message sent and echo received
    Evidence: .sisyphus/evidence/task-43-ws-connect.txt

  Scenario: Graceful close with close frame
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/websocket/... -v -run TestWS_Close -count=1
      2. Assert: exit code 0, close frame sent before disconnect
    Expected Result: Clean shutdown with WebSocket close handshake
    Evidence: .sisyphus/evidence/task-43-ws-close.txt

  Scenario: Custom headers on handshake
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/websocket/... -v -run TestWS_Headers -count=1
      2. Assert: exit code 0, custom headers present in handshake request
    Expected Result: Auth and custom headers sent during WebSocket upgrade
    Evidence: .sisyphus/evidence/task-43-ws-headers.txt
  ```

  **Commit**: YES
  - Message: `protocols: add WebSocket client with interactive mode and reconnection — third protocol handler`
  - Files: `internal/protocols/websocket/ws.go`, `internal/protocols/websocket/interactive.go`, `internal/protocols/websocket/ws_test.go`, `internal/protocols/websocket/cli.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/protocols/websocket/... -count=1`

- [ ] 44. Server-Sent Events (SSE) Client

  **What to do**:
  - **RED**: Write tests in `internal/protocols/sse/sse_test.go`:
    - `TestSSE_Connect` — connects to SSE endpoint, receives events
    - `TestSSE_ParseEvent` — parses `data:`, `event:`, `id:`, `retry:` fields
    - `TestSSE_MultilineData` — multi-line `data:` fields concatenated
    - `TestSSE_EventTypes` — filters by event type (`--event "update"`)
    - `TestSSE_Reconnect` — reconnects with `Last-Event-ID` header on disconnect
    - `TestSSE_RetryField` — respects server's `retry:` directive
    - `TestSSE_Timeout` — configurable inactivity timeout
    - `TestSSE_AuthHeaders` — custom headers sent on SSE connection
  - **GREEN**: Create `internal/protocols/sse/`:
    - `sse.go`:
      - `type Client struct` — wraps HTTP client for SSE
      - `type Event struct { ID, Type, Data string; Retry int }`
      - `Connect(ctx context.Context, url string, headers http.Header) (<-chan Event, error)` — returns channel of events
      - `parseEvent(scanner *bufio.Scanner) (*Event, error)` — SSE line protocol parser
      - Reconnection logic with `Last-Event-ID` and exponential backoff
    - `cli.go`:
      - `gurl sse "https://api.example.com/events" --header "Authorization: Bearer token"`
      - `gurl sse "https://api.example.com/events" --event "update"` — filter by event type
      - Streams events to stdout, one per line, optionally JSON-formatted
  - **REFACTOR**: Wire JSON event data through formatter for pretty-printing

  **Must NOT do**:
  - Do NOT use external SSE library — SSE protocol is simple enough to parse directly (text/event-stream)
  - Do NOT ignore `retry:` field — respect server reconnection directive
  - Do NOT drop `Last-Event-ID` on reconnect — this is how SSE resumption works

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: SSE protocol parsing + reconnection logic + channel-based streaming
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Channel patterns for streaming events to consumer
  - **Skills Evaluated but Omitted**:
    - `hono-routing`: Server-side SSE, not client-side

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 6 (with Tasks 42, 43, 45)
  - **Blocks**: None directly
  - **Blocked By**: Task 4 (HTTP client — SSE uses HTTP GET with Accept: text/event-stream)

  **References**:

  **Pattern References**:
  - `internal/protocols/graphql/graphql.go` (Task 41) — Protocol handler pattern
  - `internal/client/client.go` (Task 4) — HTTP client for making the initial GET request

  **API/Type References**:
  - `bufio.Scanner` — Line-by-line reading of SSE stream
  - `http.Request` with `Accept: text/event-stream` header
  - `pkg/types/types.go:SavedRequest` — `Protocol: "sse"`

  **External References**:
  - SSE spec: https://html.spec.whatwg.org/multipage/server-sent-events.html — Event stream format
  - MDN SSE: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events

  **WHY Each Reference Matters**:
  - SSE spec defines exact line protocol: `data:`, `event:`, `id:`, `retry:`, blank line = end of event
  - Protocol is simple enough to implement directly — no library needed
  - HTTP client provides the transport, SSE adds the parsing layer

  **Acceptance Criteria**:
  - [ ] `go test ./internal/protocols/sse/... -v -count=1` → PASS (8 tests)
  - [ ] `gurl sse "endpoint"` streams events to stdout in real-time
  - [ ] Reconnection works with Last-Event-ID

  **QA Scenarios**:

  ```
  Scenario: SSE event parsing
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/sse/... -v -run TestSSE_ParseEvent -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/protocols/sse/... -v -run TestSSE_MultilineData -count=1
      4. Assert: exit code 0
    Expected Result: data/event/id/retry fields parsed correctly, multiline data concatenated
    Evidence: .sisyphus/evidence/task-44-sse-parse.txt

  Scenario: Reconnection with Last-Event-ID
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/protocols/sse/... -v -run TestSSE_Reconnect -count=1
      2. Assert: exit code 0, Last-Event-ID header sent on reconnect
    Expected Result: Client reconnects and resumes from last event ID
    Evidence: .sisyphus/evidence/task-44-sse-reconnect.txt
  ```

  **Commit**: YES
  - Message: `protocols: add SSE client with reconnection and event filtering — fourth protocol handler`
  - Files: `internal/protocols/sse/sse.go`, `internal/protocols/sse/sse_test.go`, `internal/protocols/sse/cli.go`
  - Pre-commit: `go test ./internal/protocols/sse/... -count=1`

  - Pre-commit: `go test ./internal/protocols/sse/... -count=1`

- [ ] 45. JavaScript Scripting Engine (goja Runtime)

  **What to do**:
  - **RED**: Write tests in `internal/scripting/engine_test.go`:
    - `TestJS_BasicExecution` — `var x = 1 + 2; x` returns 3
    - `TestJS_ConsoleLog` — `console.log("hello")` captured in output buffer
    - `TestJS_RequireBuiltin` — `require("crypto")` provides crypto functions
    - `TestJS_SetVariable` — `gurl.setVar("token", "abc")` sets environment variable
    - `TestJS_GetVariable` — `gurl.getVar("baseUrl")` reads environment variable
    - `TestJS_SetHeader` — `gurl.request.headers.set("X-Custom", "val")` modifies request
    - `TestJS_ReadResponse` — `gurl.response.body`, `gurl.response.status`, `gurl.response.time` accessible
    - `TestJS_Assert` — `gurl.test("status is 200", () => { gurl.expect(gurl.response.status).to.equal(200) })`
    - `TestJS_Timeout` — script exceeding 5s limit → terminated with error
    - `TestJS_SandboxNoFS` — `require("fs")` → error (no filesystem access)
    - `TestJS_SandboxNoNet` — `require("http")` → error (no network access from scripts)
  - **GREEN**: Create `internal/scripting/`:
    - `engine.go`:
      - `type Engine struct` — wraps `github.com/dop251/goja` runtime
      - `NewEngine(env *Environment, opts ...EngineOption) *Engine`
      - `Execute(script string) (*Result, error)` — run script, return result
      - `RegisterGlobals(vm *goja.Runtime)` — inject `gurl`, `console`, `require` objects
    - `globals.go`:
      - `gurl.setVar(name, value)` / `gurl.getVar(name)` — environment variable access
      - `gurl.request` — `{url, method, headers, body}` — read/modify request before send
      - `gurl.response` — `{status, body, headers, time, size}` — read response after receive
      - `gurl.test(name, fn)` / `gurl.expect(val)` — assertion API (Postman-compatible)
      - `console.log/warn/error` — captured to output buffer
    - `sandbox.go`:
      - Restrict: no `fs`, no `net`, no `os`, no `child_process`
      - Allow: `crypto` (hashing), `Buffer` (encoding), `JSON`, `Math`, `Date`
      - Execution timeout: configurable (default 5s)
  - **REFACTOR**: Ensure API naming is Postman-compatible where possible (`pm` alias → `gurl`)

  **Must NOT do**:
  - Do NOT allow filesystem access from scripts — sandbox is critical
  - Do NOT allow network access from scripts — scripts modify requests, not make their own
  - Do NOT use `math/rand` in crypto module — use `crypto/rand`
  - Do NOT implement async/await — goja is ES5.1 with limited ES6, document this limitation

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: JS runtime embedding with security sandbox, API surface design, Postman compatibility
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Go↔JS interop via goja, security sandboxing patterns
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Engine is a library, not CLI code

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 6 (with Tasks 42, 43, 44)
  - **Blocks**: Tasks 46 (pre-request), 47 (post-response), 48 (chaining), 49 (assertions)
  - **Blocked By**: Task 15 (environments — scripts read/write env vars)

  **References**:

  **Pattern References**:
  - `internal/core/template/engine.go` — Template engine pattern in this codebase (similar: engine wrapping external lib)
  - `pkg/types/types.go:SavedRequest` — Add `PreRequestScript string`, `PostResponseScript string` fields

  **API/Type References**:
  - `github.com/dop251/goja` — `goja.New()`, `vm.Set("gurl", obj)`, `vm.RunString(script)`
  - `goja.Runtime` — JavaScript runtime, supports ES5.1 + partial ES6
  - `time.AfterFunc` — For script execution timeout

  **External References**:
  - goja: https://github.com/dop251/goja — Pure Go JavaScript runtime
  - Postman scripting API: https://learning.postman.com/docs/writing-scripts/script-references/postman-sandbox-api-reference/ — API surface to be compatible with
  - Bruno scripting: https://docs.usebruno.com/scripting/introduction — Alternative API reference

  **WHY Each Reference Matters**:
  - goja is the runtime — understand its ES5.1 limitations (no async/await, limited ES6)
  - Postman's pm API is the de facto standard — users migrating from Postman expect similar API
  - Sandbox restrictions are security-critical — no FS/net access prevents malicious scripts

  **Acceptance Criteria**:
  - [ ] `go test ./internal/scripting/... -v -count=1` → PASS (11 tests)
  - [ ] `gurl.setVar`/`getVar` round-trips work
  - [ ] Scripts timeout after configured limit
  - [ ] Sandbox blocks fs/net access

  **QA Scenarios**:

  ```
  Scenario: Script execution with gurl API
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestJS_SetVariable -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/scripting/... -v -run TestJS_ReadResponse -count=1
      4. Assert: exit code 0
    Expected Result: gurl.setVar/getVar work, response object accessible
    Evidence: .sisyphus/evidence/task-45-scripting-api.txt

  Scenario: Sandbox prevents filesystem access
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestJS_SandboxNoFS -count=1
      2. Assert: exit code 0, error message about blocked module
      3. Run: go test ./internal/scripting/... -v -run TestJS_SandboxNoNet -count=1
      4. Assert: exit code 0
    Expected Result: require("fs") and require("http") both blocked with clear error
    Evidence: .sisyphus/evidence/task-45-sandbox.txt

  Scenario: Script timeout enforcement
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestJS_Timeout -count=1
      2. Assert: exit code 0, script terminated after timeout
    Expected Result: Infinite loop script terminated after 5s with timeout error
    Evidence: .sisyphus/evidence/task-45-timeout.txt
  ```

  **Commit**: YES
  - Message: `scripting: add goja JavaScript engine with sandboxed Postman-compatible API — scripting foundation`
  - Files: `internal/scripting/engine.go`, `internal/scripting/globals.go`, `internal/scripting/sandbox.go`, `internal/scripting/engine_test.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/scripting/... -count=1`

- [ ] 46. Pre-Request Scripts

  **What to do**:
  - **RED**: Write tests in `internal/scripting/prerequest_test.go`:
    - `TestPreRequest_ModifyHeader` — script adds header, header present in final request
    - `TestPreRequest_ModifyURL` — script changes URL path, final request uses modified URL
    - `TestPreRequest_SetAuthToken` — script fetches token (from env) and sets Authorization header
    - `TestPreRequest_ModifyBody` — script modifies request body JSON
    - `TestPreRequest_SkipRequest` — `gurl.skipRequest()` prevents request execution
    - `TestPreRequest_GenerateTimestamp` — script sets `X-Timestamp` to current epoch
    - `TestPreRequest_ErrorHaltsExecution` — script error → request NOT sent, error displayed
  - **GREEN**: Create `internal/scripting/prerequest.go`:
    - `RunPreRequest(engine *Engine, script string, req *client.Request) (*client.Request, error)`
    - Inject `gurl.request` object: `{url, method, headers, body, params}`
    - `gurl.request.headers.set(key, val)` — add/modify header
    - `gurl.request.headers.remove(key)` — remove header
    - `gurl.request.url = "..."` — modify URL
    - `gurl.request.body = "..."` — modify body
    - `gurl.skipRequest()` — sets flag to skip execution
    - After script runs, extract modified request back into Go `client.Request`
  - **REFACTOR**: Wire into `run` command pipeline: load pre-request script → execute → modify request → send

  **Must NOT do**:
  - Do NOT send request if pre-request script errors — halt and report
  - Do NOT allow pre-request scripts to read response (it doesn't exist yet)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Go↔JS interop with mutable request objects, pipeline integration
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `golang-pro`: Less about Go patterns, more about JS runtime integration

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (after Task 45)
  - **Blocks**: Task 48 (request chaining uses pre-request scripts)
  - **Blocked By**: Task 45 (scripting engine — core runtime)

  **References**:

  **Pattern References**:
  - `internal/scripting/engine.go` (Task 45) — Engine to use for running scripts
  - `internal/scripting/globals.go` (Task 45) — `gurl.request` object definition
  - `internal/cli/commands/run.go` — Pipeline where pre-request scripts get inserted

  **API/Type References**:
  - `internal/client/` (Task 4) — `Request` struct that gets modified by pre-request scripts
  - `pkg/types/types.go:SavedRequest.PreRequestScript` — Where script is stored

  **WHY Each Reference Matters**:
  - Engine provides the JS runtime, globals provide the gurl.request API
  - run.go pipeline: load request → **pre-request script** → build HTTP request → send → post-response script

  **Acceptance Criteria**:
  - [ ] `go test ./internal/scripting/... -v -run TestPreRequest -count=1` → PASS (7 tests)
  - [ ] Script-modified headers appear in actual HTTP request

  **QA Scenarios**:

  ```
  Scenario: Pre-request script modifies headers
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestPreRequest_ModifyHeader -count=1
      2. Assert: exit code 0, modified header in final request
    Expected Result: Header added by script present in HTTP request
    Evidence: .sisyphus/evidence/task-46-prerequest-header.txt

  Scenario: Script error halts request
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestPreRequest_ErrorHaltsExecution -count=1
      2. Assert: exit code 0, request NOT sent, error displayed
    Expected Result: Broken script → no HTTP request, clear error message
    Evidence: .sisyphus/evidence/task-46-prerequest-error.txt
  ```

  **Commit**: YES
  - Message: `scripting: add pre-request scripts — modify requests before sending via JavaScript`
  - Files: `internal/scripting/prerequest.go`, `internal/scripting/prerequest_test.go`, `internal/cli/commands/run.go`
  - Pre-commit: `go test ./internal/scripting/... -count=1`

- [ ] 47. Post-Response Scripts

  **What to do**:
  - **RED**: Write tests in `internal/scripting/postresponse_test.go`:
    - `TestPostResponse_ReadStatus` — `gurl.response.status` returns status code
    - `TestPostResponse_ReadBody` — `gurl.response.body` returns response body string
    - `TestPostResponse_ReadHeaders` — `gurl.response.headers.get("Content-Type")` works
    - `TestPostResponse_ReadTime` — `gurl.response.time` returns duration in ms
    - `TestPostResponse_SetEnvVar` — `gurl.setVar("authToken", gurl.response.body.token)` persists
    - `TestPostResponse_TestAssertion` — `gurl.test("status ok", () => { gurl.expect(gurl.response.status).to.equal(200) })` passes/fails
    - `TestPostResponse_ExtractJSONPath` — `gurl.response.json().data.users[0].id` extracts nested value
    - `TestPostResponse_ErrorDoesNotLoseResponse` — script error still displays response, error shown separately
  - **GREEN**: Create `internal/scripting/postresponse.go`:
    - `RunPostResponse(engine *Engine, script string, resp *client.Response) (*PostResponseResult, error)`
    - Inject `gurl.response` object: `{status, body, headers, time, size}`
    - `gurl.response.json()` — parse body as JSON, return JS object for dot-notation access
    - `gurl.response.text()` — return body as string
    - `gurl.test(name, fn)` — run assertion, collect pass/fail results
    - `gurl.expect(val)` — chainable assertions: `.to.equal()`, `.to.be.above()`, `.to.have.property()`, `.to.include()`
    - `PostResponseResult` struct: `{Assertions []AssertionResult, Variables map[string]string, Logs []string}`
  - **REFACTOR**: Wire into `run` command pipeline: after response received → run post-response script → display results + assertion summary

  **Must NOT do**:
  - Do NOT suppress response display on script error — show response AND error
  - Do NOT allow post-response scripts to modify the response — read-only
  - Do NOT lose assertion results on script error — collect what ran

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Chainable assertion API design, JS↔Go response bridging, result collection
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `golang-testing`: This is JS testing API, not Go testing

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (after Task 46)
  - **Blocks**: Task 48 (chaining), Task 49 (assertion engine uses same API)
  - **Blocked By**: Task 45 (scripting engine), Task 46 (pre-request scripts for pipeline)

  **References**:

  **Pattern References**:
  - `internal/scripting/prerequest.go` (Task 46) — Mirror pattern for post-response
  - `internal/scripting/globals.go` (Task 45) — `gurl.response` object definition

  **API/Type References**:
  - `internal/client/` (Task 4) — `Response` struct: `StatusCode int`, `Body []byte`, `Headers map[string][]string`, `Duration time.Duration`
  - Postman assertion API: `pm.test()`, `pm.expect()` — compatibility target

  **External References**:
  - Postman test scripts: https://learning.postman.com/docs/writing-scripts/test-scripts/ — API reference
  - Chai.js assertion style: https://www.chaijs.com/api/bdd/ — expect().to.equal() pattern

  **WHY Each Reference Matters**:
  - Postman's pm.test/pm.expect is what users migrating from Postman expect
  - Chai BDD style (.to.equal, .to.include) is the assertion API convention

  **Acceptance Criteria**:
  - [ ] `go test ./internal/scripting/... -v -run TestPostResponse -count=1` → PASS (8 tests)
  - [ ] Assertions summary displayed after response
  - [ ] Script errors don't suppress response display

  **QA Scenarios**:

  ```
  Scenario: Post-response assertions
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestPostResponse_TestAssertion -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/scripting/... -v -run TestPostResponse_ExtractJSONPath -count=1
      4. Assert: exit code 0
    Expected Result: gurl.test passes/fails correctly, JSON path extraction works
    Evidence: .sisyphus/evidence/task-47-postresponse-assert.txt

  Scenario: Script error doesn't lose response
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestPostResponse_ErrorDoesNotLoseResponse -count=1
      2. Assert: exit code 0, response body still accessible even after script error
    Expected Result: Response displayed, script error shown separately below
    Evidence: .sisyphus/evidence/task-47-error-response.txt
  ```

  **Commit**: YES
  - Message: `scripting: add post-response scripts with Postman-compatible assertion API — test responses via JS`
  - Files: `internal/scripting/postresponse.go`, `internal/scripting/postresponse_test.go`, `internal/cli/commands/run.go`
  - Pre-commit: `go test ./internal/scripting/... -count=1`

- [ ] 48. Request Chaining via Scripts

  **What to do**:
  - **RED**: Write tests in `internal/scripting/chaining_test.go`:
    - `TestChain_SetNextRequest` — `gurl.setNextRequest("login")` sets chain target
    - `TestChain_PassVariable` — post-response sets var, pre-request of next reads it
    - `TestChain_StopChain` — `gurl.setNextRequest(null)` stops execution
    - `TestChain_CircularDetection` — A→B→A detected and stopped with error after max iterations
    - `TestChain_MaxIterations` — chain exceeding 100 iterations stopped with warning
    - `TestChain_ConditionalBranch` — `if (status === 401) gurl.setNextRequest("refresh-token") else gurl.setNextRequest("get-data")`
    - `TestChain_ExecutionOrder` — chain executes in correct sequence, order logged
  - **GREEN**: Create `internal/scripting/chaining.go`:
    - `type ChainExecutor struct` — manages chain execution state
    - `gurl.setNextRequest(name string)` — set next request to execute (by name)
    - `gurl.setNextRequest(null)` — stop chain execution
    - `ExecuteChain(startRequest string, env *Environment, maxIterations int) (*ChainResult, error)`
    - Chain loop: load request → pre-script → execute → post-script → check nextRequest → repeat
    - `ChainResult` struct: `{Requests []ChainStep, TotalDuration, Variables map[string]string}`
    - Circular detection: track visited requests, error if same request seen `maxRepeat` times (default 3)
  - **REFACTOR**: Wire `--chain` flag to `run` command: `gurl run "auth-flow" --chain` enables chaining

  **Must NOT do**:
  - Do NOT allow infinite chains — enforce max iterations (default 100, configurable)
  - Do NOT lose chain state between requests — variables persist across chain steps
  - Do NOT silently ignore circular chains — detect and error

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: State machine (chain executor), circular detection, variable passing across execution boundaries
  - **Skills**: [`golang-pro`]
    - `golang-pro`: State machine patterns, iteration management
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Chain logic is the hard part, CLI wiring is minimal

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (after Tasks 46, 47)
  - **Blocks**: Collection runner (Task 50 — uses chaining for sequential runs)
  - **Blocked By**: Task 46 (pre-request scripts), Task 47 (post-response scripts)

  **References**:

  **Pattern References**:
  - `internal/scripting/prerequest.go` (Task 46) — Pre-request script execution
  - `internal/scripting/postresponse.go` (Task 47) — Post-response script execution + gurl.setNextRequest
  - `internal/cli/commands/run.go` — Where --chain flag gets wired

  **API/Type References**:
  - `internal/storage/db.go` — `GetByName(name)` to load chained requests by name
  - `internal/scripting/engine.go` (Task 45) — Engine reused across chain steps (shared state)

  **External References**:
  - Postman collection runner: https://learning.postman.com/docs/running-collections/intro-to-collection-runs/ — setNextRequest behavior
  - Newman (Postman CLI runner): https://github.com/postmanlabs/newman — CLI chaining reference

  **WHY Each Reference Matters**:
  - Postman's setNextRequest is the exact API we're implementing — same semantics
  - Newman shows how chaining works in CLI context (no GUI)
  - Storage GetByName loads the next request in the chain

  **Acceptance Criteria**:
  - [ ] `go test ./internal/scripting/... -v -run TestChain -count=1` → PASS (7 tests)
  - [ ] `gurl run "auth-flow" --chain` executes chain of requests
  - [ ] Circular chains detected and stopped

  **QA Scenarios**:

  ```
  Scenario: Request chaining with variable passing
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestChain_PassVariable -count=1
      2. Assert: exit code 0, variable set in step 1 available in step 2
    Expected Result: Post-response variable persisted across chain steps
    Evidence: .sisyphus/evidence/task-48-chain-variable.txt

  Scenario: Circular chain detection
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestChain_CircularDetection -count=1
      2. Assert: exit code 0, error mentions circular chain
    Expected Result: A→B→A chain stopped with "circular chain detected" error
    Evidence: .sisyphus/evidence/task-48-chain-circular.txt

  Scenario: Conditional branching
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/scripting/... -v -run TestChain_ConditionalBranch -count=1
      2. Assert: exit code 0, correct branch taken based on status
    Expected Result: 401→refresh-token path, 200→get-data path
    Evidence: .sisyphus/evidence/task-48-chain-conditional.txt
  ```

  **Commit**: YES
  - Message: `scripting: add request chaining via setNextRequest — Postman-compatible workflow automation`
  - Files: `internal/scripting/chaining.go`, `internal/scripting/chaining_test.go`, `internal/cli/commands/run.go`
  - Pre-commit: `go test ./internal/scripting/... -count=1`

  - Pre-commit: `go test ./internal/scripting/... -count=1`

### Wave 7: Assertions + Collection Runner + Advanced CLI (Tasks 49-56)

- [ ] 49. Assertion Engine (Declarative TOML-Based)

  **What to do**:
  - **RED**: Write tests in `internal/assertions/engine_test.go`:
    - `TestAssert_StatusCode` — `status = 200` passes on 200 response
    - `TestAssert_StatusRange` — `status = "2xx"` passes on any 200-299
    - `TestAssert_HeaderExists` — `headers.Content-Type = "exists"` passes when header present
    - `TestAssert_HeaderValue` — `headers.Content-Type = "application/json"` matches exact value
    - `TestAssert_HeaderContains` — `headers.Content-Type contains "json"` partial match
    - `TestAssert_BodyContains` — `body contains "success"` checks body text
    - `TestAssert_BodyJSONPath` — `body.$.data.id = 123` checks JSON path value
    - `TestAssert_ResponseTime` — `time < 500` response under 500ms
    - `TestAssert_BodySize` — `size < 10240` response under 10KB
    - `TestAssert_MultipleAssertions` — all assertions evaluated, summary shows pass/fail for each
    - `TestAssert_FromSavedRequest` — assertions loaded from SavedRequest.Assertions field
  - **GREEN**: Create `internal/assertions/`:
    - `engine.go`:
      - `type Assertion struct { Field, Operator, Value string }` — declarative assertion
      - `type Result struct { Assertion, Passed bool, Actual, Expected string }`
      - `Evaluate(assertions []Assertion, resp *client.Response) []Result`
      - Operators: `=`, `!=`, `<`, `>`, `<=`, `>=`, `contains`, `not_contains`, `matches` (regex), `exists`
      - Field resolvers: `status`, `headers.{name}`, `body`, `body.{jsonpath}`, `time`, `size`
      - Use switch on operator for dispatch
    - `parser.go`:
      - `ParseTOML(toml string) ([]Assertion, error)` — parse TOML assertion blocks
      - `ParseInline(args []string) ([]Assertion, error)` — parse CLI inline assertions
    - Add `Assertions []Assertion` field to `SavedRequest` type
    - Wire `--assert` flag to `run` command: `gurl run "api" --assert "status=200" --assert "time<500"`
  - **REFACTOR**: Integrate with JS scripting (Task 47) — JS assertions and TOML assertions both produce `[]Result`

  **Must NOT do**:
  - Do NOT only support JS-based assertions — TOML declarative assertions are simpler for basic checks
  - Do NOT use if-else for operator dispatch — use switch with explicit cases
  - Do NOT silently skip failed assertions — always report all results

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Flexible assertion DSL with multiple operators, field resolvers, and integration with two systems (TOML + JS)
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `tdd`: TDD approach already defined in task spec

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 7 (with Tasks 50, 53, 54, 55, 56)
  - **Blocks**: Collection runner assertions (Task 50), CI exit codes (Task 53)
  - **Blocked By**: Task 35 (JSONPath — used for body.$.path assertions), Task 47 (post-response scripts — shared result format)

  **References**:

  **Pattern References**:
  - `internal/scripting/postresponse.go` (Task 47) — JS assertion results to unify with
  - `internal/formatter/filter.go` (Task 35) — JSONPath evaluation reused for body.$.path

  **API/Type References**:
  - `internal/client/` (Task 4) — `Response` struct: StatusCode, Body, Headers, Duration
  - `pkg/types/types.go:SavedRequest` — Add `Assertions []Assertion` field
  - `internal/formatter/filter.go:FilterJSON` (Task 35) — JSONPath extraction

  **External References**:
  - Bruno assertions: https://docs.usebruno.com/testing/assertions — TOML-like declarative assertions
  - Postman tests: https://learning.postman.com/docs/writing-scripts/test-scripts/ — JS-based assertion comparison

  **WHY Each Reference Matters**:
  - Bruno's declarative assertions are the model for our TOML-based approach (simpler than JS for basic checks)
  - JSONPath filter from Task 35 provides the body.$.path evaluation engine
  - Unifying JS and TOML assertion results enables consistent reporting

  **Acceptance Criteria**:
  - [ ] `go test ./internal/assertions/... -v -count=1` → PASS (11 tests)
  - [ ] `gurl run "api" --assert "status=200" --assert "time<500"` shows assertion results
  - [ ] All operators work: =, !=, <, >, contains, matches, exists

  **QA Scenarios**:

  ```
  Scenario: Declarative assertions with multiple operators
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/assertions/... -v -run TestAssert -count=1
      2. Assert: exit code 0, all assertion tests pass
    Expected Result: Status, header, body, time, size assertions all evaluate correctly
    Evidence: .sisyphus/evidence/task-49-assertions.txt

  Scenario: Multiple assertions with mixed pass/fail
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/assertions/... -v -run TestAssert_MultipleAssertions -count=1
      2. Assert: exit code 0, summary shows individual pass/fail for each assertion
    Expected Result: Summary like "3/5 assertions passed" with per-assertion detail
    Evidence: .sisyphus/evidence/task-49-mixed-results.txt
  ```

  **Commit**: YES
  - Message: `assertions: add declarative TOML-based assertion engine — verify API responses without JS`
  - Files: `internal/assertions/engine.go`, `internal/assertions/parser.go`, `internal/assertions/engine_test.go`, `pkg/types/types.go`, `internal/cli/commands/run.go`
  - Pre-commit: `go test ./internal/assertions/... -count=1`

- [ ] 50. Collection Runner

  **What to do**:
  - **RED**: Write tests in `internal/runner/runner_test.go`:
    - `TestRunner_RunCollection` — executes all requests in collection sequentially
    - `TestRunner_RunWithOrder` — respects request order (by sort order or explicit sequence)
    - `TestRunner_ContinueOnError` — failed request doesn't stop collection (configurable)
    - `TestRunner_StopOnError` — `--bail` flag stops on first failure
    - `TestRunner_Summary` — prints summary: total, passed, failed, skipped, duration
    - `TestRunner_Variables` — collection-level variables available to all requests
    - `TestRunner_PrePostScripts` — pre/post scripts run for each request
    - `TestRunner_Assertions` — all request assertions evaluated, aggregated results
    - `TestRunner_Delay` — configurable delay between requests
    - `TestRunner_Iterations` — run collection N times with `--iterations 3`
  - **GREEN**: Create `internal/runner/`:
    - `runner.go`:
      - `type Runner struct` — collection execution engine
      - `type RunConfig struct { Collection, Environment, Bail, Iterations, Delay, DataFile string }`
      - `type RunResult struct { Total, Passed, Failed, Skipped int; Duration time.Duration; Results []RequestResult }`
      - `type RequestResult struct { Name string; Response *client.Response; Assertions []assertions.Result; Error error; Duration time.Duration }`
      - `Run(ctx context.Context, config RunConfig) (*RunResult, error)`
      - Pipeline per request: resolve env → pre-script → execute → post-script → assertions → record result
    - `cli.go`:
      - `gurl collection run "my-collection" --env "staging" --bail --iterations 3 --delay 100ms`
      - Display: progress bar during run, summary table after
  - **REFACTOR**: Extract progress display into reusable component for TUI reuse

  **Must NOT do**:
  - Do NOT run requests in parallel by default — sequential is the collection runner standard
  - Do NOT lose results on error — always produce full report
  - Do NOT hardcode iteration count — configurable via flag and config

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Orchestration engine combining environments, scripting, assertions, HTTP client — many integration points
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Context management, pipeline orchestration, result aggregation
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Runner logic is the complex part, not CLI wiring

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 7 (with Tasks 49, 53, 54, 55, 56)
  - **Blocks**: Task 51 (data-driven testing), Task 52 (reporters), Task 53 (CI exit codes)
  - **Blocked By**: Task 49 (assertions), Task 48 (chaining), Task 17 (environments wired to run)

  **References**:

  **Pattern References**:
  - `internal/cli/commands/run.go` — Single request execution pipeline to scale up
  - `internal/scripting/chaining.go` (Task 48) — Chain executor pattern (similar loop structure)
  - `internal/cli/commands/collection.go` — Existing collection commands

  **API/Type References**:
  - `internal/storage/db.go` — `GetByCollection(name)` to load all requests in a collection
  - `internal/assertions/engine.go` (Task 49) — Assertion evaluation
  - `internal/scripting/prerequest.go` (Task 46) — Pre-request script execution
  - `internal/scripting/postresponse.go` (Task 47) — Post-response script execution

  **External References**:
  - Newman (Postman CLI): https://github.com/postmanlabs/newman — Collection runner reference
  - Bruno CLI runner: https://docs.usebruno.com/bru-cli/overview — Alternative runner reference

  **WHY Each Reference Matters**:
  - Newman is the gold standard for CLI collection runners — same UX expectations
  - GetByCollection loads all requests to execute
  - Chaining executor has the same loop pattern (load → script → execute → script → next)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/runner/... -v -count=1` → PASS (10 tests)
  - [ ] `gurl collection run "my-collection"` executes all requests with summary
  - [ ] `--bail` stops on first failure
  - [ ] `--iterations 3` runs collection 3 times

  **QA Scenarios**:

  ```
  Scenario: Collection execution with summary
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/... -v -run TestRunner_RunCollection -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/runner/... -v -run TestRunner_Summary -count=1
      4. Assert: exit code 0, summary includes total/passed/failed/duration
    Expected Result: All requests executed, summary shows pass/fail counts
    Evidence: .sisyphus/evidence/task-50-runner.txt

  Scenario: Bail on first failure
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/... -v -run TestRunner_StopOnError -count=1
      2. Assert: exit code 0, execution stopped after first failed request
    Expected Result: Collection stops early, remaining requests skipped
    Evidence: .sisyphus/evidence/task-50-bail.txt
  ```

  **Commit**: YES
  - Message: `runner: add collection runner with iterations, bail, and assertion aggregation — batch API testing`
  - Files: `internal/runner/runner.go`, `internal/runner/runner_test.go`, `internal/runner/cli.go`
  - Pre-commit: `go test ./internal/runner/... -count=1`

- [ ] 51. Data-Driven Testing (CSV/JSON Datasets)

  **What to do**:
  - **RED**: Write tests in `internal/runner/datadriven_test.go`:
    - `TestDataDriven_CSV` — runs request for each CSV row, substituting variables
    - `TestDataDriven_JSON` — runs request for each JSON array element
    - `TestDataDriven_CSVHeaders` — first row used as variable names
    - `TestDataDriven_VariableSubstitution` — `{{name}}` in URL/body replaced from dataset row
    - `TestDataDriven_Iteration` — each row is a separate execution, results tracked per row
    - `TestDataDriven_EmptyFile` — returns error, not silent success
    - `TestDataDriven_MissingColumn` — error when template references nonexistent column
  - **GREEN**: Create `internal/runner/datadriven.go`:
    - `LoadCSVDataset(path string) ([]map[string]string, error)` — parse CSV, first row = headers
    - `LoadJSONDataset(path string) ([]map[string]string, error)` — parse JSON array of objects
    - `LoadDataset(path string) ([]map[string]string, error)` — auto-detect by extension
    - Integrate with runner: `--data ./users.csv` → run request once per row
    - Each row's variables merged with environment variables (row takes precedence)
    - Wire `--data` flag to both `run` and `collection run` commands
  - **REFACTOR**: Reuse template engine (Task 38 dynamic values) for variable substitution in dataset-driven requests

  **Must NOT do**:
  - Do NOT load entire large file into memory at once — stream CSV rows
  - Do NOT silently ignore missing columns — error with column name and row number

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: CSV/JSON parsing + variable substitution, well-defined I/O
  - **Skills**: []
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 7 (with Tasks 49, 50, 53, 54, 55, 56)
  - **Blocks**: None directly
  - **Blocked By**: Task 50 (collection runner — dataset feeds into runner)

  **References**:

  **Pattern References**:
  - `internal/runner/runner.go` (Task 50) — Runner pipeline where dataset rows get injected
  - `internal/core/template/engine.go` — Template variable substitution for dataset values

  **API/Type References**:
  - `encoding/csv` stdlib — `csv.NewReader()` for CSV parsing
  - `encoding/json` stdlib — `json.Decoder` for JSON array streaming
  - `internal/runner/runner.go:RunConfig` — Add `DataFile string` field

  **WHY Each Reference Matters**:
  - Runner pipeline is where each dataset row triggers a request execution
  - Template engine handles `{{variable}}` substitution from dataset values

  **Acceptance Criteria**:
  - [ ] `go test ./internal/runner/... -v -run TestDataDriven -count=1` → PASS (7 tests)
  - [ ] `gurl run "create-user" --data ./users.csv` creates users from CSV

  **QA Scenarios**:

  ```
  Scenario: CSV data-driven execution
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/... -v -run TestDataDriven_CSV -count=1
      2. Assert: exit code 0, each row executed as separate request
    Expected Result: N rows → N requests, each with row-specific variables
    Evidence: .sisyphus/evidence/task-51-csv-driven.txt

  Scenario: Missing column error
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/... -v -run TestDataDriven_MissingColumn -count=1
      2. Assert: exit code 0, error mentions column name and row number
    Expected Result: "column 'userId' not found in row 3" style error
    Evidence: .sisyphus/evidence/task-51-missing-column.txt
  ```

  **Commit**: YES
  - Message: `runner: add CSV/JSON data-driven testing — parameterize requests from datasets`
  - Files: `internal/runner/datadriven.go`, `internal/runner/datadriven_test.go`, `internal/cli/commands/run.go`
  - Pre-commit: `go test ./internal/runner/... -count=1`

- [ ] 52. Test Reporters (JUnit XML, JSON, HTML)

  **What to do**:
  - **RED**: Write tests in `internal/runner/reporters_test.go`:
    - `TestReporter_JUnit` — produces valid JUnit XML with testsuites/testcases
    - `TestReporter_JSON` — produces structured JSON report with all fields
    - `TestReporter_HTML` — produces self-contained HTML report (embedded CSS, no external deps)
    - `TestReporter_JUnit_FailedTest` — failed assertion → `<failure>` element with message
    - `TestReporter_JUnit_Skipped` — skipped request → `<skipped>` element
    - `TestReporter_Console` — default console output with colored pass/fail markers
    - `TestReporter_Multiple` — multiple reporters active simultaneously
  - **GREEN**: Create `internal/runner/reporters/`:
    - `reporter.go`: `type Reporter interface { Report(result *RunResult) ([]byte, error) }`
    - `junit.go`: JUnit XML format — compatible with CI tools (Jenkins, GitHub Actions)
    - `json.go`: Machine-readable JSON report
    - `html.go`: Self-contained HTML report with:
      - Summary stats (total/passed/failed/duration)
      - Per-request expandable sections with response details
      - Assertion results with pass/fail styling
      - Embedded CSS (no external dependencies — single file)
    - `console.go`: Terminal output with ANSI colors (default reporter)
    - Wire `--reporter` flag: `gurl collection run "api" --reporter junit --reporter-output ./report.xml`
    - Support multiple: `--reporter junit --reporter json` outputs both
  - **REFACTOR**: Extract ANSI color usage to use formatter theme (Task 34)

  **Must NOT do**:
  - Do NOT use external template libraries for HTML — use Go's `html/template`
  - Do NOT require external CSS/JS for HTML report — everything embedded
  - Do NOT break JUnit XML compatibility — must work with GitHub Actions test summary

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Multiple output formats with compatibility requirements, HTML generation
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Reporters are output formatters, not CLI interaction

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 7 (with Tasks 49, 50, 51, 53, 54, 55, 56)
  - **Blocks**: None directly
  - **Blocked By**: Task 50 (collection runner — reporters consume RunResult)

  **References**:

  **Pattern References**:
  - `internal/formatter/formatter.go` (Task 34) — ANSI theme for console reporter
  - `internal/runner/runner.go` (Task 50) — `RunResult` struct consumed by reporters

  **API/Type References**:
  - `encoding/xml` stdlib — JUnit XML generation
  - `html/template` stdlib — HTML report generation
  - `internal/runner/runner.go:RunResult` — Input for all reporters

  **External References**:
  - JUnit XML schema: https://llg.cubic.org/docs/junit/ — XML structure reference
  - Newman HTML reporter: https://github.com/postmanlabs/newman-reporter-html — HTML report reference

  **WHY Each Reference Matters**:
  - JUnit XML must match the schema that GitHub Actions and Jenkins expect
  - Newman's HTML reporter shows the UX standard for API test reports

  **Acceptance Criteria**:
  - [ ] `go test ./internal/runner/reporters/... -v -count=1` → PASS (7 tests)
  - [ ] JUnit XML validates against schema
  - [ ] HTML report opens in browser, self-contained

  **QA Scenarios**:

  ```
  Scenario: JUnit XML report generation
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/reporters/... -v -run TestReporter_JUnit -count=1
      2. Assert: exit code 0, output is valid XML with <testsuites> root
    Expected Result: JUnit-compatible XML with testcases, failures, timing
    Evidence: .sisyphus/evidence/task-52-junit.txt

  Scenario: Self-contained HTML report
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/reporters/... -v -run TestReporter_HTML -count=1
      2. Assert: exit code 0, HTML contains <style> block (no external CSS), all sections present
    Expected Result: Single HTML file with embedded styles, pass/fail visualization
    Evidence: .sisyphus/evidence/task-52-html.txt
  ```

  **Commit**: YES
  - Message: `runner: add JUnit XML, JSON, and HTML test reporters — CI-compatible test output`
  - Files: `internal/runner/reporters/reporter.go`, `internal/runner/reporters/junit.go`, `internal/runner/reporters/json.go`, `internal/runner/reporters/html.go`, `internal/runner/reporters/console.go`, `internal/runner/reporters/reporters_test.go`
  - Pre-commit: `go test ./internal/runner/reporters/... -count=1`

  - Pre-commit: `go test ./internal/runner/reporters/... -count=1`

- [ ] 53. CI-Friendly Exit Codes

  **What to do**:
  - **RED**: Write tests in `internal/runner/exitcode_test.go`:
    - `TestExitCode_AllPass` — 0 when all assertions pass
    - `TestExitCode_SomeFail` — 1 when any assertion fails
    - `TestExitCode_RuntimeError` — 2 when runner encounters runtime error (connection refused, timeout)
    - `TestExitCode_NoRequests` — 3 when collection is empty
    - `TestExitCode_ScriptError` — 4 when pre/post script has JS error
    - `TestExitCode_Mapping` — all codes documented in `--help` output
  - **GREEN**: Create `internal/runner/exitcode.go`:
    - `type ExitCode int` with named constants
    - `DetermineExitCode(result *RunResult) ExitCode` — maps result to exit code
    - Wire exit codes into `collection run` and `run --assert` commands via `os.Exit()`
    - Add `--ci` flag for strict mode: any warning becomes failure
    - Exit codes: 0=success, 1=assertion_failure, 2=runtime_error, 3=empty_collection, 4=script_error
  - **REFACTOR**: Ensure `run` command (single request) also returns proper exit codes when assertions used

  **Must NOT do**:
  - Do NOT use exit code 1 for everything — differentiate failure types for CI debugging
  - Do NOT call os.Exit() deep in library code — return ExitCode, let CLI layer call os.Exit()

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small module, clear mapping, minimal logic
  - **Skills**: []
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 7 (with Tasks 49-52, 54, 55, 56)
  - **Blocks**: None directly
  - **Blocked By**: Task 50 (runner — exit codes consume RunResult)

  **References**:

  **Pattern References**:
  - `cmd/gurl/main.go` — Top-level CLI entry where os.Exit is called
  - `internal/runner/runner.go` (Task 50) — `RunResult` determines exit code

  **API/Type References**:
  - `internal/runner/runner.go:RunResult` — Pass/Fail/Error counts drive exit code
  - `os.Exit(code)` — Only called at CLI layer, not in library

  **External References**:
  - Newman exit codes: https://github.com/postmanlabs/newman#exit-status — Industry standard for API testing exit codes

  **WHY Each Reference Matters**:
  - Newman's exit code convention is what CI engineers expect
  - os.Exit must only be called at CLI layer, not inside runner library

  **Acceptance Criteria**:
  - [ ] `go test ./internal/runner/... -v -run TestExitCode -count=1` → PASS (6 tests)
  - [ ] `gurl collection run "api" && echo "pass" || echo "fail"` works correctly in CI

  **QA Scenarios**:

  ```
  Scenario: Exit codes differentiate failure types
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/... -v -run TestExitCode -count=1
      2. Assert: exit code 0, all exit code mappings correct
    Expected Result: 0=success, 1=assertion fail, 2=runtime error, 3=empty, 4=script error
    Evidence: .sisyphus/evidence/task-53-exitcodes.txt
  ```

  **Commit**: YES
  - Message: `runner: add CI-friendly exit codes — differentiate assertion failures from runtime errors`
  - Files: `internal/runner/exitcode.go`, `internal/runner/exitcode_test.go`, `cmd/gurl/main.go`
  - Pre-commit: `go test ./internal/runner/... -count=1`

- [ ] 54. Edit Command (Full Implementation)

  **What to do**:
  - **RED**: Write tests in `internal/cli/commands/edit_test.go`:
    - `TestEdit_ChangeMethod` — `gurl edit "api" --method POST` updates method
    - `TestEdit_AddHeader` — `gurl edit "api" --header "Authorization: Bearer token"` adds header
    - `TestEdit_RemoveHeader` — `gurl edit "api" --remove-header "Authorization"` removes header
    - `TestEdit_ChangeURL` — `gurl edit "api" --url "https://new-api.com"` updates URL
    - `TestEdit_ChangeBody` — `gurl edit "api" --body '{"new":"data"}'` updates body
    - `TestEdit_SetCollection` — `gurl edit "api" --collection "v2"` moves to collection
    - `TestEdit_AddTag` — `gurl edit "api" --tag "critical"` adds tag
    - `TestEdit_SetPreScript` — `gurl edit "api" --pre-script ./setup.js` sets pre-request script
    - `TestEdit_SetPostScript` — `gurl edit "api" --post-script ./validate.js` sets post-response script
    - `TestEdit_SetAssertions` — `gurl edit "api" --assert "status=200"` adds assertion
    - `TestEdit_Interactive` — opens $EDITOR with TOML representation when no flags (future, stub test)
  - **GREEN**: Rewrite `internal/cli/commands/edit.go`:
    - Replace current stub with full implementation
    - Each flag maps to a field update on SavedRequest
    - Load request → apply changes → validate → save back
    - Support `--pre-script-file` and `--post-script-file` for loading scripts from files
    - Support `--assert` for inline assertion addition
    - Validate: URL format, method is valid HTTP method, JSON body parses if content-type is JSON
  - **REFACTOR**: Extract validation into `internal/validation/` for reuse by save command

  **Must NOT do**:
  - Do NOT implement interactive editor mode yet — that's TUI territory (Wave 8)
  - Do NOT allow editing nonexistent requests — error with "request not found" and suggestions

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: CLI flag handling and storage update, straightforward CRUD
  - **Skills**: [`cli-developer`]
    - `cli-developer`: Multiple CLI flags, validation, urfave/cli patterns
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 7 (with Tasks 49-53, 55, 56)
  - **Blocks**: TUI request builder (Task 60 — edit via TUI)
  - **Blocked By**: Task 14 (fix save command — edit reads/writes same storage)

  **References**:

  **Pattern References**:
  - `internal/cli/commands/edit.go` — Existing stub to replace
  - `internal/cli/commands/save.go` (Task 14) — Save command pattern for writing back to storage
  - `internal/cli/commands/rename.go` — Shows pattern for loading, modifying, saving a request

  **API/Type References**:
  - `internal/storage/db.go` — `GetByName(name)`, `Update(request)` methods
  - `pkg/types/types.go:SavedRequest` — All fields that can be edited

  **WHY Each Reference Matters**:
  - edit.go is the file being rewritten — read current stub for CLI contract
  - rename.go shows the load→modify→save pattern already established in this codebase
  - SavedRequest defines every editable field

  **Acceptance Criteria**:
  - [ ] `go test ./internal/cli/commands/... -v -run TestEdit -count=1` → PASS (11 tests)
  - [ ] `gurl edit "api" --method POST --header "X-New: val"` updates request correctly

  **QA Scenarios**:

  ```
  Scenario: Edit request fields via CLI flags
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/cli/commands/... -v -run TestEdit_ChangeMethod -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/cli/commands/... -v -run TestEdit_AddHeader -count=1
      4. Assert: exit code 0
    Expected Result: Method changed, header added, persisted to storage
    Evidence: .sisyphus/evidence/task-54-edit.txt

  Scenario: Edit nonexistent request returns error
    Tool: Bash (go test + CLI)
    Steps:
      1. Run: gurl edit "nonexistent" --method POST 2>&1
      2. Assert: output contains "not found"
    Expected Result: Clear "request 'nonexistent' not found" error
    Evidence: .sisyphus/evidence/task-54-edit-notfound.txt
  ```

  **Commit**: YES
  - Message: `cli: implement full edit command — modify method, headers, body, scripts, assertions inline`
  - Files: `internal/cli/commands/edit.go`, `internal/cli/commands/edit_test.go`
  - Pre-commit: `go test ./internal/cli/commands/... -count=1`

- [ ] 55. Nested Folders (Hierarchical Organization)

  **What to do**:
  - **RED**: Write tests in `internal/storage/folders_test.go`:
    - `TestFolder_Create` — create folder `/api/v2/users`
    - `TestFolder_Nested` — folder inside folder: `/api/v2/users/admin`
    - `TestFolder_MoveRequest` — move request into folder
    - `TestFolder_ListFolder` — list requests in specific folder
    - `TestFolder_ListRecursive` — list all requests in folder and subfolders
    - `TestFolder_DeleteFolder` — delete folder moves requests to parent (or requires empty)
    - `TestFolder_FolderPath` — requests display with full path: `api/v2/users/get-user`
    - `TestFolder_RootRequests` — requests without folder appear at root level
  - **GREEN**: Extend storage system:
    - Add `Folder string` field to `SavedRequest` (path-like: "api/v2/users")
    - `CreateFolder(path string) error`
    - `ListFolder(path string) ([]SavedRequest, error)` — requests in this folder
    - `ListFolderRecursive(path string) ([]SavedRequest, error)` — requests in folder + subfolders
    - `MoveToFolder(requestName, folderPath string) error`
    - Update `list` command to show folder tree structure
    - Wire `--folder` flag: `gurl save "get-user" https://api.com/users --folder "api/v2/users"`
  - **REFACTOR**: Update list command display to show tree-like structure with indentation

  **Must NOT do**:
  - Do NOT create actual filesystem directories — folders are virtual (stored in DB)
  - Do NOT allow deleting non-empty folders without confirmation flag
  - Do NOT use recursive function calls for folder traversal — use iterative approach with stack

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Hierarchical data model in flat KV store, tree display, path parsing
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Tree display is minor compared to storage logic

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 7 (with Tasks 49-54, 56)
  - **Blocks**: TUI request list panel (Task 59 — shows folder tree)
  - **Blocked By**: Task 8 (DB schema versioning — new field needs migration)

  **References**:

  **Pattern References**:
  - `internal/storage/db.go` — Storage system to extend with folder support
  - `internal/cli/commands/list.go` — List command to update with folder display
  - `internal/cli/commands/collection.go` — Collections are a related concept (folders within collections)

  **API/Type References**:
  - `pkg/types/types.go:SavedRequest` — Add `Folder string` field
  - `internal/storage/db.go:DB` interface — Add folder methods

  **External References**:
  - Insomnia folder structure: https://docs.insomnia.rest/insomnia/request-collection — Nested folders reference
  - Bruno folder-based storage: https://docs.usebruno.com/get-started/overview — Files mirror folder structure

  **WHY Each Reference Matters**:
  - Insomnia's folder model is the UX target — nested folders within collections
  - Storage DB interface needs new methods for folder CRUD
  - List command display changes to show tree indentation

  **Acceptance Criteria**:
  - [ ] `go test ./internal/storage/... -v -run TestFolder -count=1` → PASS (8 tests)
  - [ ] `gurl list` shows folder tree structure
  - [ ] `gurl save "api" url --folder "v2/users"` saves to folder

  **QA Scenarios**:

  ```
  Scenario: Create and list nested folders
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/storage/... -v -run TestFolder_Nested -count=1
      2. Assert: exit code 0
      3. Run: go test ./internal/storage/... -v -run TestFolder_ListRecursive -count=1
      4. Assert: exit code 0
    Expected Result: Nested folders created, recursive listing shows full tree
    Evidence: .sisyphus/evidence/task-55-folders.txt

  Scenario: Folder path displayed with request
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/storage/... -v -run TestFolder_FolderPath -count=1
      2. Assert: exit code 0, requests show full path like "api/v2/users/get-user"
    Expected Result: Path-like display for organized browsing
    Evidence: .sisyphus/evidence/task-55-folder-path.txt
  ```

  **Commit**: YES
  - Message: `storage: add nested folder hierarchy — organize requests like Insomnia/Bruno`
  - Files: `internal/storage/db.go`, `internal/storage/folders_test.go`, `pkg/types/types.go`, `internal/cli/commands/list.go`, `internal/cli/commands/save.go`
  - Pre-commit: `go test ./internal/storage/... -count=1`

- [ ] 56. Request Sequencing (Execution Order)

  **What to do**:
  - **RED**: Write tests in `internal/runner/sequence_test.go`:
    - `TestSequence_Explicit` — `gurl sequence set "login" 1 && gurl sequence set "get-data" 2` defines order
    - `TestSequence_InCollection` — requests in collection run in sequence order
    - `TestSequence_Unordered` — unsequenced requests run in name-alphabetical order
    - `TestSequence_Reorder` — changing sequence number reorders execution
    - `TestSequence_Gaps` — sequence numbers with gaps (1, 5, 10) still work correctly
    - `TestSequence_Display` — `gurl sequence list "collection"` shows ordered list
  - **GREEN**: Extend SavedRequest and runner:
    - Add `SortOrder int` field to `SavedRequest` type
    - `gurl sequence set "request-name" 1` — set execution order
    - `gurl sequence list "collection"` — show requests in order
    - Runner (Task 50) sorts by `SortOrder` before execution (0 = unordered, sorted alphabetically)
  - **REFACTOR**: Integrate with collection runner — sequence is the default ordering for collection runs

  **Must NOT do**:
  - Do NOT require sequential numbering — gaps are fine (1, 5, 10)
  - Do NOT break existing behavior — unsequenced requests default to alphabetical

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple numeric ordering, minimal new logic
  - **Skills**: []
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 7 (with Tasks 49-55)
  - **Blocks**: None directly
  - **Blocked By**: Task 50 (collection runner — sequence feeds into runner ordering)

  **References**:

  **Pattern References**:
  - `internal/runner/runner.go` (Task 50) — Sort requests by SortOrder before execution
  - `internal/cli/commands/collection.go` — Related collection commands

  **API/Type References**:
  - `pkg/types/types.go:SavedRequest` — Add `SortOrder int` field
  - `sort.Slice` stdlib — Sort requests by SortOrder

  **WHY Each Reference Matters**:
  - Runner needs to sort by SortOrder before executing collection
  - SortOrder field on SavedRequest persists the ordering

  **Acceptance Criteria**:
  - [ ] `go test ./internal/runner/... -v -run TestSequence -count=1` → PASS (6 tests)
  - [ ] `gurl sequence set "login" 1 && gurl sequence set "data" 2` orders correctly

  **QA Scenarios**:

  ```
  Scenario: Request sequencing in collection
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/... -v -run TestSequence_InCollection -count=1
      2. Assert: exit code 0, requests executed in sequence order
    Expected Result: Requests run in SortOrder: 1→2→3, not alphabetical
    Evidence: .sisyphus/evidence/task-56-sequence.txt

  Scenario: Unsequenced requests default to alphabetical
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/runner/... -v -run TestSequence_Unordered -count=1
      2. Assert: exit code 0, alphabetical ordering for SortOrder=0
    Expected Result: Backward-compatible alphabetical order for unsequenced
    Evidence: .sisyphus/evidence/task-56-unordered.txt
  ```

  **Commit**: YES
  - Message: `runner: add request sequencing — control execution order in collections`
  - Files: `internal/runner/sequence.go`, `internal/runner/sequence_test.go`, `pkg/types/types.go`, `internal/runner/runner.go`
  - Pre-commit: `go test ./internal/runner/... -count=1`

  - Pre-commit: `go test ./internal/runner/... -count=1`

### Wave 8: Terminal UI — bubbletea (Tasks 57-62)

- [ ] 57. TUI Foundation (bubbletea App Shell)

  **What to do**:
  - **RED**: Write tests in `internal/tui/app_test.go`:
    - `TestTUI_Init` — app initializes without error, shows welcome screen
    - `TestTUI_QuitOnQ` — pressing 'q' quits the app cleanly
    - `TestTUI_QuitOnCtrlC` — Ctrl+C quits the app
    - `TestTUI_Layout` — app has 3 panels: sidebar (request list), main (request/response), statusbar
    - `TestTUI_Resize` — terminal resize recalculates panel dimensions
    - `TestTUI_FocusSwitch` — Tab key switches focus between panels
    - `TestTUI_StatusBar` — status bar shows current env, request count, version
  - **GREEN**: Create `internal/tui/`:
    - `app.go`:
      - `type App struct` — root bubbletea model
      - `func NewApp(db storage.DB, config *Config) *App`
      - `Init() tea.Cmd` — load initial data (request list, config)
      - `Update(msg tea.Msg) (tea.Model, tea.Cmd)` — central message handler with switch
      - `View() string` — render 3-panel layout using lipgloss
    - `layout.go`:
      - `type Layout struct { SidebarWidth, MainWidth, StatusHeight int }`
      - `CalculateLayout(width, height int) Layout` — responsive panel sizing
      - Sidebar: 25% width (min 30 chars), Main: 75%, Status: 1 line
    - `styles.go`:
      - lipgloss styles for borders, selected items, headers, status bar
      - Color theme matching formatter (Task 34) ANSI theme
    - `statusbar.go`:
      - `type StatusBar struct` — bubbletea sub-model
      - Shows: current environment, request count, gurl version, last action message
    - Add `gurl tui` command to launch TUI
    - Add `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles` to deps
  - **REFACTOR**: Extract common key bindings into `internal/tui/keys.go` for consistent keyboard shortcuts across panels

  **Must NOT do**:
  - Do NOT implement raw terminal I/O — use bubbletea exclusively (from AGENT.md)
  - Do NOT hardcode panel sizes — responsive layout based on terminal dimensions
  - Do NOT use if-else for key handling — use switch on key.String()

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: TUI layout design, visual styling with lipgloss, panel-based UI
  - **Skills**: [`building-tui-apps`, `cli-developer`]
    - `building-tui-apps`: bubbletea architecture, panel layouts, keyboard handling, resize
    - `cli-developer`: CLI integration, terminal I/O patterns
  - **Skills Evaluated but Omitted**:
    - `opentui-core-concepts`: OpenTUI is a different framework, not bubbletea

  **Parallelization**:
  - **Can Run In Parallel**: NO (foundation for all TUI tasks)
  - **Parallel Group**: Wave 8 leader (Tasks 58-62 depend on this)
  - **Blocks**: ALL other TUI tasks (58-62)
  - **Blocked By**: None directly (uses storage and config which exist by Wave 8)

  **References**:

  **Pattern References**:
  - `internal/tui/` — Currently empty directory, this creates the foundation
  - `internal/formatter/theme.go` (Task 34) — Color theme to reuse for TUI styling
  - `cmd/gurl/main.go` — Where `gurl tui` subcommand gets registered

  **API/Type References**:
  - `github.com/charmbracelet/bubbletea` — `tea.Model` interface: Init, Update, View
  - `github.com/charmbracelet/lipgloss` — Terminal styling: `lipgloss.NewStyle().Border().Padding()`
  - `github.com/charmbracelet/bubbles` — Pre-built components: viewport, textinput, list, spinner
  - `internal/storage/db.go` — DB interface for loading requests
  - `pkg/types/types.go:Config.UI` — UI configuration section

  **External References**:
  - bubbletea tutorial: https://github.com/charmbracelet/bubbletea/tree/master/tutorials
  - lipgloss: https://github.com/charmbracelet/lipgloss — Terminal layout and styling
  - lazygit TUI: https://github.com/jesseduffield/lazygit — 3-panel TUI reference (sidebar + main + status)

  **WHY Each Reference Matters**:
  - bubbletea is THE framework per AGENT.md — no other option
  - lazygit shows the 3-panel TUI pattern (sidebar+main+statusbar) that works well for API clients
  - lipgloss handles borders, padding, colors — essential for panel layout
  - bubbles provides pre-built list, viewport, textinput components

  **Acceptance Criteria**:
  - [ ] `go test ./internal/tui/... -v -count=1` → PASS (7 tests)
  - [ ] `gurl tui` launches TUI with 3 panels
  - [ ] q/Ctrl+C quits cleanly, Tab switches focus

  **QA Scenarios**:

  ```
  Scenario: TUI launches and displays layout
    Tool: interactive_bash (tmux)
    Preconditions: Requests exist in DB
    Steps:
      1. tmux new-session -d -s tui-test
      2. tmux send-keys -t tui-test "gurl tui" Enter
      3. Wait 2s
      4. tmux capture-pane -t tui-test -p > .sisyphus/evidence/task-57-tui-launch.txt
      5. Assert: output contains sidebar border, main panel, status bar
      6. tmux send-keys -t tui-test "q"
      7. Assert: process exits cleanly
    Expected Result: 3-panel layout visible, clean quit on 'q'
    Failure Indicators: Blank screen, panic, no borders/styling
    Evidence: .sisyphus/evidence/task-57-tui-launch.txt

  Scenario: Terminal resize recalculates layout
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/tui/... -v -run TestTUI_Resize -count=1
      2. Assert: exit code 0, panel widths adjust proportionally
    Expected Result: Sidebar stays 25%, main panel fills remainder
    Evidence: .sisyphus/evidence/task-57-tui-resize.txt
  ```

  **Commit**: YES
  - Message: `tui: add bubbletea app shell with 3-panel layout — TUI foundation`
  - Files: `internal/tui/app.go`, `internal/tui/layout.go`, `internal/tui/styles.go`, `internal/tui/statusbar.go`, `internal/tui/keys.go`, `internal/tui/app_test.go`, `cmd/gurl/main.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/tui/... -count=1`

- [ ] 58. TUI: Request List Panel (Sidebar)

  **What to do**:
  - **RED**: Write tests in `internal/tui/requestlist_test.go`:
    - `TestRequestList_Load` — loads all requests from DB, displays in sidebar
    - `TestRequestList_Navigate` — j/k or ↑/↓ navigates list
    - `TestRequestList_Select` — Enter on request loads it in main panel
    - `TestRequestList_Filter` — `/` opens filter, typing filters by name
    - `TestRequestList_FolderTree` — shows folder hierarchy with expand/collapse
    - `TestRequestList_MethodColor` — GET=green, POST=blue, PUT=yellow, DELETE=red
    - `TestRequestList_CollectionGroup` — requests grouped by collection
    - `TestRequestList_Empty` — empty state shows "No requests. Save one first."
  - **GREEN**: Create `internal/tui/requestlist.go`:
    - `type RequestList struct` — bubbletea sub-model wrapping `bubbles/list`
    - Displays: method badge (colored) + request name + folder path
    - Folder support: expandable/collapsible tree nodes
    - Filter mode: '/' activates fuzzy search, Esc clears
    - Collection grouping: headers with collection name, requests underneath
    - Keyboard: j/k navigate, Enter select, / filter, Tab switch focus
    - Method color mapping: switch on method → lipgloss color
  - **REFACTOR**: Extract list item rendering into `internal/tui/listitem.go` for reuse

  **Must NOT do**:
  - Do NOT build custom list component — use `bubbles/list` as base
  - Do NOT load all request bodies upfront — lazy load on selection
  - Do NOT use if-else for method color — use switch or map

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: List UI with colors, tree structure, filter UX
  - **Skills**: [`building-tui-apps`]
    - `building-tui-apps`: bubbletea list component, keyboard navigation, filtering
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Not CLI, purely TUI component

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Task 57)
  - **Parallel Group**: Wave 8 post-foundation (with Tasks 59, 60, 61, 62)
  - **Blocks**: None directly (other panels communicate via messages)
  - **Blocked By**: Task 57 (TUI foundation), Task 55 (nested folders for tree display)

  **References**:

  **Pattern References**:
  - `internal/tui/app.go` (Task 57) — Parent app model that hosts this panel
  - `internal/tui/styles.go` (Task 57) — Shared lipgloss styles

  **API/Type References**:
  - `github.com/charmbracelet/bubbles/list` — Pre-built list component
  - `internal/storage/db.go` — `GetAll()`, `GetByCollection()` to load requests
  - `pkg/types/types.go:SavedRequest` — Data displayed in list

  **External References**:
  - bubbles list: https://github.com/charmbracelet/bubbles/tree/master/list — List component docs

  **WHY Each Reference Matters**:
  - bubbles/list provides filtering, pagination, keyboard nav out of the box
  - Storage GetAll() loads the request data to display

  **Acceptance Criteria**:
  - [ ] `go test ./internal/tui/... -v -run TestRequestList -count=1` → PASS (8 tests)
  - [ ] Requests display with colored method badges
  - [ ] Folder tree expandable/collapsible

  **QA Scenarios**:

  ```
  Scenario: Request list with navigation and filter
    Tool: interactive_bash (tmux)
    Preconditions: Multiple requests saved in DB
    Steps:
      1. tmux new-session -d -s tui-list
      2. tmux send-keys -t tui-list "gurl tui" Enter
      3. Wait 2s
      4. tmux send-keys -t tui-list "j" — navigate down
      5. tmux send-keys -t tui-list "j"
      6. tmux send-keys -t tui-list "/" — open filter
      7. tmux send-keys -t tui-list "user" — type filter text
      8. tmux capture-pane -t tui-list -p > .sisyphus/evidence/task-58-list.txt
      9. Assert: filtered list shows only requests with "user" in name
      10. tmux send-keys -t tui-list "q"
    Expected Result: List navigable, filter works, method colors visible
    Evidence: .sisyphus/evidence/task-58-list.txt

  Scenario: Empty state display
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/tui/... -v -run TestRequestList_Empty -count=1
      2. Assert: exit code 0, message "No requests" displayed
    Expected Result: Helpful empty state instead of blank panel
    Evidence: .sisyphus/evidence/task-58-empty.txt
  ```

  **Commit**: YES
  - Message: `tui: add request list panel with folder tree, filter, and method colors — sidebar navigation`
  - Files: `internal/tui/requestlist.go`, `internal/tui/listitem.go`, `internal/tui/requestlist_test.go`
  - Pre-commit: `go test ./internal/tui/... -count=1`

- [ ] 59. TUI: Request Builder Panel

  **What to do**:
  - **RED**: Write tests in `internal/tui/requestbuilder_test.go`:
    - `TestBuilder_DisplayRequest` — shows URL, method, headers, body of selected request
    - `TestBuilder_EditURL` — Enter on URL field enables editing, Esc saves
    - `TestBuilder_EditMethod` — cycle method: GET→POST→PUT→PATCH→DELETE→GET
    - `TestBuilder_AddHeader` — 'a' in headers section adds new header row
    - `TestBuilder_RemoveHeader` — 'd' on header removes it (with confirmation)
    - `TestBuilder_EditBody` — Tab to body section, Enter to edit, syntax highlight JSON
    - `TestBuilder_SendRequest` — Ctrl+Enter sends request, shows loading spinner
    - `TestBuilder_SaveChanges` — Ctrl+S saves edited request back to DB
    - `TestBuilder_NewRequest` — 'n' creates new blank request form
  - **GREEN**: Create `internal/tui/requestbuilder.go`:
    - `type RequestBuilder struct` — bubbletea sub-model for request editing
    - Sections: URL bar (method dropdown + URL input), Headers (key-value table), Body (textarea with syntax highlight)
    - URL bar: method selector (bubbles/list or custom spinner) + text input for URL
    - Headers: table with add/remove, tab between key and value fields
    - Body: textarea with JSON/XML syntax highlighting (reuse formatter theme)
    - Keyboard: Ctrl+Enter=send, Ctrl+S=save, Tab=next section, Shift+Tab=prev section
    - Use `bubbles/textinput` for URL and header values, `bubbles/textarea` for body
  - **REFACTOR**: Extract form field components into `internal/tui/formfield.go` for reuse across panels

  **Must NOT do**:
  - Do NOT implement raw text editing — use bubbles/textarea and bubbles/textinput
  - Do NOT block UI while request is executing — show spinner, process async
  - Do NOT lose unsaved changes on navigation — warn user

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Complex form UI with multiple input types, tabs, syntax highlighting
  - **Skills**: [`building-tui-apps`]
    - `building-tui-apps`: bubbletea form components, focus management, async operations
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Task 57)
  - **Parallel Group**: Wave 8 post-foundation (with Tasks 58, 60, 61, 62)
  - **Blocks**: None directly
  - **Blocked By**: Task 57 (TUI foundation), Task 54 (edit command — builder reuses edit logic)

  **References**:

  **Pattern References**:
  - `internal/tui/app.go` (Task 57) — Parent app model, message passing to builder
  - `internal/tui/styles.go` (Task 57) — Shared styles for form fields
  - `internal/cli/commands/edit.go` (Task 54) — Edit logic to reuse for save/update

  **API/Type References**:
  - `github.com/charmbracelet/bubbles/textinput` — Text input for URL, header values
  - `github.com/charmbracelet/bubbles/textarea` — Multiline editor for body
  - `github.com/charmbracelet/bubbles/spinner` — Loading indicator during request
  - `internal/client/` (Task 4) — HTTP client for sending requests

  **External References**:
  - bubbles textinput: https://github.com/charmbracelet/bubbles/tree/master/textinput
  - bubbles textarea: https://github.com/charmbracelet/bubbles/tree/master/textarea

  **WHY Each Reference Matters**:
  - bubbles components provide cursor handling, scrolling, focus — don't build from scratch
  - HTTP client is called when user presses Ctrl+Enter to send
  - Edit command logic handles validation and persistence

  **Acceptance Criteria**:
  - [ ] `go test ./internal/tui/... -v -run TestBuilder -count=1` → PASS (9 tests)
  - [ ] Ctrl+Enter sends request with loading spinner
  - [ ] URL, method, headers, body all editable

  **QA Scenarios**:

  ```
  Scenario: Request builder with editable fields
    Tool: interactive_bash (tmux)
    Steps:
      1. tmux new-session -d -s tui-builder
      2. tmux send-keys -t tui-builder "gurl tui" Enter
      3. Wait 2s
      4. tmux send-keys -t tui-builder Enter — select first request
      5. Wait 1s
      6. tmux capture-pane -t tui-builder -p > .sisyphus/evidence/task-59-builder.txt
      7. Assert: URL, method, headers, body sections visible
      8. tmux send-keys -t tui-builder "q"
    Expected Result: Request details displayed with editable fields
    Evidence: .sisyphus/evidence/task-59-builder.txt

  Scenario: Send request from TUI
    Tool: interactive_bash (tmux)
    Steps:
      1. Launch TUI, select request
      2. tmux send-keys -t tui-builder C-Enter — Ctrl+Enter to send
      3. Wait 3s for response
      4. tmux capture-pane -t tui-builder -p > .sisyphus/evidence/task-59-send.txt
      5. Assert: spinner shown during request, response appears after
    Expected Result: Loading spinner → response displayed in response panel
    Evidence: .sisyphus/evidence/task-59-send.txt
  ```

  **Commit**: YES
  - Message: `tui: add request builder panel — edit URL, method, headers, body and send requests`
  - Files: `internal/tui/requestbuilder.go`, `internal/tui/formfield.go`, `internal/tui/requestbuilder_test.go`
  - Pre-commit: `go test ./internal/tui/... -count=1`

  - Pre-commit: `go test ./internal/tui/... -count=1`

- [ ] 60. TUI: Response Viewer Panel

  **What to do**:
  - **RED**: Write tests in `internal/tui/responseviewer_test.go`:
    - `TestViewer_DisplayResponse` — shows status code, headers, body
    - `TestViewer_StatusColor` — 2xx=green, 3xx=yellow, 4xx=orange, 5xx=red
    - `TestViewer_PrettyPrintJSON` — JSON body syntax-highlighted in viewer
    - `TestViewer_ScrollBody` — long body scrollable with j/k or ↑/↓
    - `TestViewer_TabSections` — tabs for Body, Headers, Cookies, Timing
    - `TestViewer_CopyToClipboard` — 'y' copies response body to clipboard
    - `TestViewer_SaveToFile` — 's' saves response body to file (prompts filename)
    - `TestViewer_ResponseMeta` — shows: status code, duration, size, content-type
  - **GREEN**: Create `internal/tui/responseviewer.go`:
    - `type ResponseViewer struct` — bubbletea sub-model
    - Tab bar: Body | Headers | Cookies | Timing (use custom tab component)
    - Body tab: syntax-highlighted response (reuse formatter from Task 34)
    - Headers tab: key-value table of response headers
    - Cookies tab: parsed Set-Cookie headers
    - Timing tab: DNS, connect, TLS, TTFB, total time breakdown
    - Status badge: colored based on status code range (switch on range)
    - Scrollable viewport for long responses (use bubbles/viewport)
    - 'y' to copy body to system clipboard (via `github.com/atotto/clipboard`)
    - 's' to save body to file (prompt with textinput for filename)
  - **REFACTOR**: Extract tab component into `internal/tui/tabs.go` for reuse

  **Must NOT do**:
  - Do NOT render entire response at once for very large bodies — use viewport with lazy rendering
  - Do NOT lose response when switching tabs — cache response in model
  - Do NOT use if-else for status color — use switch on status code range

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Rich response display with tabs, syntax highlighting, scroll, clipboard
  - **Skills**: [`building-tui-apps`]
    - `building-tui-apps`: bubbletea viewport, tab patterns, clipboard integration
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Task 57)
  - **Parallel Group**: Wave 8 post-foundation (with Tasks 58, 59, 61, 62)
  - **Blocks**: None directly
  - **Blocked By**: Task 57 (TUI foundation), Task 34 (formatter for syntax highlighting)

  **References**:

  **Pattern References**:
  - `internal/tui/app.go` (Task 57) — Parent app, receives response messages
  - `internal/formatter/formatter.go` (Task 34) — Syntax highlighting for body display
  - `internal/formatter/theme.go` (Task 34) — Color theme for consistent styling

  **API/Type References**:
  - `github.com/charmbracelet/bubbles/viewport` — Scrollable content area
  - `github.com/atotto/clipboard` — System clipboard access
  - `internal/client/` (Task 4) — `Response` struct: StatusCode, Body, Headers, Duration, Timing

  **External References**:
  - bubbles viewport: https://github.com/charmbracelet/bubbles/tree/master/viewport
  - Insomnia response panel: reference for tab layout (Preview, Timeline, Header, Cookie)

  **WHY Each Reference Matters**:
  - Viewport handles scrolling for large responses (essential for real-world API responses)
  - Formatter provides syntax highlighting already implemented in Task 34
  - Response struct has all the data fields to display

  **Acceptance Criteria**:
  - [ ] `go test ./internal/tui/... -v -run TestViewer -count=1` → PASS (8 tests)
  - [ ] Response displayed with syntax-highlighted body, scrollable
  - [ ] Tab switching between Body/Headers/Cookies/Timing

  **QA Scenarios**:

  ```
  Scenario: Response viewer with tabs
    Tool: interactive_bash (tmux)
    Steps:
      1. Launch TUI, select request, send with Ctrl+Enter
      2. Wait for response
      3. tmux capture-pane -t tui-resp -p > .sisyphus/evidence/task-60-body.txt
      4. Assert: status code badge visible, body syntax-highlighted
      5. Press Tab to switch to Headers tab
      6. tmux capture-pane -t tui-resp -p > .sisyphus/evidence/task-60-headers.txt
      7. Assert: response headers displayed as key-value table
    Expected Result: Tabbed response viewer with colored status and highlighted body
    Evidence: .sisyphus/evidence/task-60-body.txt, .sisyphus/evidence/task-60-headers.txt

  Scenario: Scroll long response body
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/tui/... -v -run TestViewer_ScrollBody -count=1
      2. Assert: exit code 0, viewport scrolls with j/k
    Expected Result: Large response scrollable, viewport updates position indicator
    Evidence: .sisyphus/evidence/task-60-scroll.txt
  ```

  **Commit**: YES
  - Message: `tui: add response viewer with tabs, syntax highlighting, and clipboard — view API responses`
  - Files: `internal/tui/responseviewer.go`, `internal/tui/tabs.go`, `internal/tui/responseviewer_test.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/tui/... -count=1`

- [ ] 61. TUI: Environment Switcher

  **What to do**:
  - **RED**: Write tests in `internal/tui/envswitcher_test.go`:
    - `TestEnvSwitcher_List` — shows all environments in dropdown/popup
    - `TestEnvSwitcher_Select` — selecting environment updates status bar and active env
    - `TestEnvSwitcher_Variables` — switching env updates variable resolution in request builder
    - `TestEnvSwitcher_Shortcut` — Ctrl+E opens environment switcher
    - `TestEnvSwitcher_CurrentHighlighted` — current env visually highlighted in list
    - `TestEnvSwitcher_NoEnvs` — shows "No environments configured" with hint to create one
  - **GREEN**: Create `internal/tui/envswitcher.go`:
    - `type EnvSwitcher struct` — bubbletea sub-model (overlay/popup)
    - Popup list of environments from storage (Task 15)
    - Ctrl+E opens popup, j/k navigate, Enter selects, Esc closes
    - Selection updates: status bar environment display, active environment in app state
    - Variable preview: show variable count for each environment
  - **REFACTOR**: Extract popup component into `internal/tui/popup.go` for reuse (confirmations, prompts)

  **Must NOT do**:
  - Do NOT block entire app for env selection — popup overlay pattern
  - Do NOT lose request state when switching environments — only variables change

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Popup overlay UI pattern, state management across panels
  - **Skills**: [`building-tui-apps`]
    - `building-tui-apps`: bubbletea overlay/popup patterns, state propagation
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Task 57)
  - **Parallel Group**: Wave 8 post-foundation (with Tasks 58, 59, 60, 62)
  - **Blocks**: None directly
  - **Blocked By**: Task 57 (TUI foundation), Task 15 (environment system)

  **References**:

  **Pattern References**:
  - `internal/tui/app.go` (Task 57) — App state where active environment is stored
  - `internal/tui/statusbar.go` (Task 57) — Status bar displays current environment
  - `internal/tui/styles.go` (Task 57) — Popup styling

  **API/Type References**:
  - `internal/storage/db.go` — Load environments (Task 15 additions)
  - `internal/tui/app.go:App` — `ActiveEnvironment string` state field

  **WHY Each Reference Matters**:
  - Environment list comes from storage, status bar shows current selection
  - App state propagates environment choice to request builder for variable resolution

  **Acceptance Criteria**:
  - [ ] `go test ./internal/tui/... -v -run TestEnvSwitcher -count=1` → PASS (6 tests)
  - [ ] Ctrl+E opens popup, selection updates status bar

  **QA Scenarios**:

  ```
  Scenario: Environment switcher popup
    Tool: interactive_bash (tmux)
    Steps:
      1. Launch TUI
      2. tmux send-keys -t tui-env C-e — Ctrl+E to open switcher
      3. Wait 1s
      4. tmux capture-pane -t tui-env -p > .sisyphus/evidence/task-61-envswitcher.txt
      5. Assert: popup showing list of environments
      6. tmux send-keys -t tui-env Enter — select first env
      7. tmux capture-pane -t tui-env -p > .sisyphus/evidence/task-61-selected.txt
      8. Assert: status bar shows selected environment name
    Expected Result: Popup with environment list, selection updates status bar
    Evidence: .sisyphus/evidence/task-61-envswitcher.txt
  ```

  **Commit**: YES
  - Message: `tui: add environment switcher popup — switch environments with Ctrl+E`
  - Files: `internal/tui/envswitcher.go`, `internal/tui/popup.go`, `internal/tui/envswitcher_test.go`
  - Pre-commit: `go test ./internal/tui/... -count=1`

- [ ] 62. TUI: Keyboard Shortcuts and Help

  **What to do**:
  - **RED**: Write tests in `internal/tui/help_test.go`:
    - `TestHelp_Toggle` — '?' toggles help overlay
    - `TestHelp_ShowAllShortcuts` — help shows all shortcuts grouped by context
    - `TestHelp_ContextSensitive` — help shows different shortcuts based on focused panel
    - `TestHelp_ShortcutBar` — bottom bar shows most common shortcuts (like vim status line)
    - `TestHelp_Close` — Esc or '?' closes help
  - **GREEN**: Create `internal/tui/help.go`:
    - `type HelpPanel struct` — bubbletea sub-model (overlay)
    - Global shortcuts:
      - `?` — toggle help
      - `q` — quit
      - `Tab` — switch panel focus
      - `Ctrl+E` — environment switcher
      - `Ctrl+Enter` — send request
      - `Ctrl+S` — save request
      - `n` — new request
      - `/` — filter requests
    - Context shortcuts: different based on focused panel
    - Bottom shortcut bar: persistent hint line showing 3-4 most relevant shortcuts for current context
  - **REFACTOR**: Wire all keyboard shortcuts through `internal/tui/keys.go` (Task 57) centralized key binding registry

  **Must NOT do**:
  - Do NOT hardcode shortcut strings in help — generate from keys.go registry
  - Do NOT show help in a new screen — use overlay that preserves context

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Help overlay is display-only, shortcuts already defined in keys.go
  - **Skills**: [`building-tui-apps`]
    - `building-tui-apps`: bubbletea help patterns, key binding display
  - **Skills Evaluated but Omitted**: None

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Task 57)
  - **Parallel Group**: Wave 8 post-foundation (with Tasks 58, 59, 60, 61)
  - **Blocks**: None directly
  - **Blocked By**: Task 57 (TUI foundation — keys.go registry)

  **References**:

  **Pattern References**:
  - `internal/tui/keys.go` (Task 57) — Centralized key binding definitions
  - `internal/tui/popup.go` (Task 61) — Popup overlay component to reuse
  - `internal/tui/styles.go` (Task 57) — Help panel styling

  **API/Type References**:
  - `github.com/charmbracelet/bubbles/help` — Pre-built help component
  - `github.com/charmbracelet/bubbles/key` — Key binding definitions

  **External References**:
  - bubbles help: https://github.com/charmbracelet/bubbles/tree/master/help — Help component
  - lazygit help panel: reference for context-sensitive help

  **WHY Each Reference Matters**:
  - bubbles/help provides the help rendering framework
  - bubbles/key provides key binding registration that auto-generates help text
  - Context-sensitive help requires knowing which panel is focused

  **Acceptance Criteria**:
  - [ ] `go test ./internal/tui/... -v -run TestHelp -count=1` → PASS (5 tests)
  - [ ] '?' shows help overlay, Esc closes it
  - [ ] Bottom bar shows context-sensitive shortcuts

  **QA Scenarios**:

  ```
  Scenario: Help overlay with shortcuts
    Tool: interactive_bash (tmux)
    Steps:
      1. Launch TUI
      2. tmux send-keys -t tui-help "?" — open help
      3. Wait 1s
      4. tmux capture-pane -t tui-help -p > .sisyphus/evidence/task-62-help.txt
      5. Assert: help overlay visible with grouped shortcuts
      6. tmux send-keys -t tui-help Escape — close help
      7. Assert: help closed, TUI restored
    Expected Result: Help overlay with all shortcuts, context-grouped
    Evidence: .sisyphus/evidence/task-62-help.txt

  Scenario: Bottom shortcut bar changes with context
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/tui/... -v -run TestHelp_ShortcutBar -count=1
      2. Assert: exit code 0, bar content changes based on focused panel
    Expected Result: Request list focus → "Enter:select /: filter", Builder focus → "C-Enter:send C-S:save"
    Evidence: .sisyphus/evidence/task-62-shortcutbar.txt
  ```

  **Commit**: YES
  - Message: `tui: add keyboard shortcuts help and context-sensitive shortcut bar — discoverable TUI`
  - Files: `internal/tui/help.go`, `internal/tui/help_test.go`, `internal/tui/keys.go`
  - Pre-commit: `go test ./internal/tui/... -count=1`

- [ ] 63. Plugin System Architecture

  **What to do**:
  - **RED**: Write tests in `internal/plugins/loader_test.go`:
    - `TestLoader_DiscoverPlugins` — finds plugin directories in `~/.config/gurl/plugins/`
    - `TestLoader_LoadPlugin` — loads a Go plugin (.so) and extracts interfaces
    - `TestLoader_MiddlewarePlugin` — loaded plugin implements MiddlewarePlugin, BeforeRequest/AfterResponse called
    - `TestLoader_OutputPlugin` — loaded plugin implements OutputPlugin, Format/Render called
    - `TestLoader_CommandPlugin` — loaded plugin registers new CLI subcommand
    - `TestLoader_InvalidPlugin` — graceful error when .so doesn't implement any known interface
    - `TestLoader_DisabledPlugin` — plugin in dir but not in config's `enabled` list → skipped
    - `TestLoader_PluginPanic` — panic inside plugin → recovered, logged, request continues
    - `TestRegistry_Register` — register built-in plugins via code (not .so)
    - `TestRegistry_Middleware_Chain` — multiple middleware plugins chain correctly (order preserved)
  - **GREEN**: Create `internal/plugins/`:
    - `interfaces.go`: Plugin interfaces exactly matching PRD:
      ```go
      type MiddlewarePlugin interface {
          Name() string
          BeforeRequest(ctx *RequestContext) *RequestContext
          AfterResponse(ctx *ResponseContext) *ResponseContext
      }
      type OutputPlugin interface {
          Name() string
          Format() string
          Render(ctx *ResponseContext) string
      }
      type CommandPlugin interface {
          Name() string
          Command() string
          Description() string
          Run(args []string) error
      }
      type RequestContext struct {
          Request *client.Request
          Env     map[string]string
      }
      type ResponseContext struct {
          Request  *client.Request
          Response *client.Response
          Env      map[string]string
      }
      ```
    - `registry.go`: `type Registry struct` — central plugin registry:
      - `Register(plugin interface{})` — type-switches to categorize into middleware/output/command
      - `Middleware() []MiddlewarePlugin` — ordered list
      - `Outputs() []OutputPlugin` — all output plugins
      - `Commands() []CommandPlugin` — all command plugins
      - `ApplyBeforeRequest(ctx *RequestContext) *RequestContext` — chains all middleware
      - `ApplyAfterResponse(ctx *ResponseContext) *ResponseContext` — chains all middleware
    - `loader.go`: `type Loader struct` — file-system plugin discovery + loading:
      - `Discover(pluginDir string) ([]string, error)` — find .so files in subdirs
      - `Load(path string) (interface{}, error)` — `plugin.Open()` + `plugin.Lookup("Plugin")` symbol
      - `LoadAll(config Config) (*Registry, error)` — discover, filter by enabled list, load, register
      - Wrap each plugin call in recover() for panic safety
    - Wire into `cmd/gurl/main.go`:
      - Load plugins during app startup
      - Register CommandPlugin entries as new urfave/cli commands
      - Inject middleware chain into request execution pipeline (Task 4 client)
  - **REFACTOR**: Extract `RequestContext`/`ResponseContext` to `pkg/types/` if used outside plugins package

  **Must NOT do**:
  - Do NOT use `reflect` for interface detection — use type assertions with `switch`
  - Do NOT panic on plugin errors — recover and log, continue without the plugin
  - Do NOT require plugins for core functionality — all features work without plugins
  - Do NOT load plugins not in the `enabled` config list — security boundary

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Go plugin system requires careful interface design, panic recovery, dynamic loading, and integration with existing CLI framework
  - **Skills**: [`cli-developer`]
    - `cli-developer`: Plugin system integrates with CLI command registration and middleware chain
  - **Skills Evaluated but Omitted**:
    - `golang-pro`: While Go-specific, plugin system is more about architecture than concurrency

  **Parallelization**:
  - **Can Run In Parallel**: NO (foundation for Tasks 64, 65)
  - **Parallel Group**: Wave 9 first (sequential — 64, 65 depend on this)
  - **Blocks**: Task 64 (template function plugins), Task 65 (auth plugins)
  - **Blocked By**: Task 4 (HTTP client — RequestContext/ResponseContext reference client types), Task 15 (environments — Env map in contexts)

  **References**:

  **Pattern References**:
  - `internal/auth/auth.go` (Task 21) — Handler interface pattern: Name() + Apply() — similar to plugin interface pattern
  - `cmd/gurl/main.go` — CLI command registration pattern (where CommandPlugin entries get registered)
  - `internal/client/client.go` (Task 4) — Request/Response types used in plugin contexts

  **API/Type References**:
  - `plugin` stdlib — `plugin.Open()`, `plugin.Lookup()` for loading .so files
  - `pkg/types/types.go:Config.Plugins.Enabled` — Config `[]string` listing enabled plugins
  - `PRD.md:703-730` — Plugin interfaces as specified in the PRD (MUST match exactly)

  **External References**:
  - Go plugin package: https://pkg.go.dev/plugin — official docs for Go plugin system
  - HashiCorp go-plugin: https://github.com/hashicorp/go-plugin — reference for plugin architecture (gRPC-based, more complex — we use simpler Go plugin)

  **WHY Each Reference Matters**:
  - PRD defines the exact interfaces — this task implements them verbatim
  - Auth handler pattern shows how to do Name() + Apply() interface that plugins follow
  - Go `plugin` stdlib is the loading mechanism for .so files
  - Config.Plugins.Enabled is the security boundary for which plugins activate

  **Acceptance Criteria**:
  - [ ] `go test ./internal/plugins/... -v -count=1` → PASS (10 tests)
  - [ ] Plugin interfaces match PRD exactly (MiddlewarePlugin, OutputPlugin, CommandPlugin)
  - [ ] Registry chains middleware in order, recovers from panics
  - [ ] Disabled plugins not loaded even if present on disk

  **QA Scenarios**:

  ```
  Scenario: Plugin registry with middleware chain
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/plugins/... -v -run TestRegistry -count=1
      2. Assert: exit code 0, middleware chain applies in registration order
    Expected Result: BeforeRequest A → BeforeRequest B → execute → AfterResponse B → AfterResponse A
    Evidence: .sisyphus/evidence/task-63-registry.txt

  Scenario: Invalid plugin recovery
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/plugins/... -v -run TestLoader_InvalidPlugin -count=1
      2. Assert: exit code 0, error logged, no panic
    Expected Result: Loader returns descriptive error, does not crash process
    Evidence: .sisyphus/evidence/task-63-invalid-plugin.txt

  Scenario: Plugin panic recovery
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/plugins/... -v -run TestLoader_PluginPanic -count=1
      2. Assert: exit code 0, panic recovered, request continues without plugin
    Expected Result: Plugin panic caught, logged with plugin name, request proceeds
    Evidence: .sisyphus/evidence/task-63-panic-recovery.txt
  ```

  **Commit**: YES
  - Message: `plugins: add plugin system architecture with middleware/output/command interfaces — extensible core`
  - Files: `internal/plugins/interfaces.go`, `internal/plugins/registry.go`, `internal/plugins/loader.go`, `internal/plugins/loader_test.go`
  - Pre-commit: `go test ./internal/plugins/... -count=1`

- [ ] 64. Template Function Plugins (Output Plugins)

  **What to do**:
  - **RED**: Write tests in `internal/plugins/builtins/output_test.go`:
    - `TestOutputPlugin_Slack` — renders response as Slack markdown block (```json``` with status)
    - `TestOutputPlugin_Markdown` — renders as GitHub-flavored markdown table (headers) + fenced code block (body)
    - `TestOutputPlugin_CSV` — renders response headers + body fields as CSV rows
    - `TestOutputPlugin_Minimal` — renders only status code + first 80 chars of body
    - `TestOutputPlugin_Registration` — all built-in output plugins register in Registry
    - `TestOutputPlugin_Wire` — `gurl run "test" --output slack` triggers Slack plugin
  - **GREEN**: Create `internal/plugins/builtins/`:
    - `slack.go`: `type SlackOutput struct` implementing `OutputPlugin`:
      - `Name()` → "slack", `Format()` → "slack"
      - `Render()` → Slack-formatted: status emoji (✅/❌) + URL + ```response body```
    - `markdown.go`: `type MarkdownOutput struct` implementing `OutputPlugin`:
      - `Name()` → "markdown", `Format()` → "markdown"
      - `Render()` → `# GET https://... (200 OK)\n\n| Header | Value |\n...\n\n```json\n{body}\n```
    - `csv.go`: `type CSVOutput struct` implementing `OutputPlugin`:
      - `Name()` → "csv", `Format()` → "csv"
      - `Render()` → status,url,duration,content-type\n200,https://...,145ms,application/json
    - `minimal.go`: `type MinimalOutput struct` implementing `OutputPlugin`:
      - `Name()` → "minimal", `Format()` → "minimal"
      - `Render()` → `200 OK (145ms)` — one-line output
    - `register.go`: `RegisterBuiltins(registry *plugins.Registry)` — registers all built-in output plugins
    - Wire `--output` flag to `run` command: select output plugin by format name
    - Fallback: if `--output X` not found in registry, error with available formats
  - **REFACTOR**: Move existing formatter (Task 34) to also register as the "pretty" output plugin, unifying the output pipeline

  **Must NOT do**:
  - Do NOT import external Slack/Markdown libraries — generate text directly
  - Do NOT break existing `--format` flag — `--output` is for plugin outputs, `--format` for built-in (json/yaml/table)
  - Do NOT hardcode output plugin list — use Registry discovery

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Multiple output plugins each with specific formatting rules, integration with CLI flags
  - **Skills**: [`cli-developer`]
    - `cli-developer`: CLI flag integration, output formatting patterns
  - **Skills Evaluated but Omitted**:
    - `building-tui-apps`: Output plugins are CLI output, not TUI

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 65 after Task 63)
  - **Parallel Group**: Wave 9 post-foundation (with Task 65, 66)
  - **Blocks**: None directly
  - **Blocked By**: Task 63 (plugin system — OutputPlugin interface and Registry)

  **References**:

  **Pattern References**:
  - `internal/plugins/interfaces.go` (Task 63) — OutputPlugin interface to implement
  - `internal/plugins/registry.go` (Task 63) — Registry.Register() for built-in plugins
  - `internal/formatter/formatter.go` (Task 34) — Existing formatter to wrap as "pretty" output plugin

  **API/Type References**:
  - `internal/plugins/interfaces.go:OutputPlugin` — `Name()`, `Format()`, `Render(*ResponseContext)`
  - `internal/plugins/interfaces.go:ResponseContext` — Request, Response, Env available in Render
  - `internal/client/client.go:Response` — StatusCode, Body, Headers, Duration fields

  **External References**:
  - Slack message formatting: https://api.slack.com/reference/surfaces/formatting — Slack markdown spec
  - GitHub Flavored Markdown tables: https://github.github.com/gfm/#tables-extension-

  **WHY Each Reference Matters**:
  - OutputPlugin interface is the contract from Task 63
  - Existing formatter becomes the "pretty" plugin, unifying all output through the plugin pipeline
  - Slack/GFM specs ensure output is valid in those contexts

  **Acceptance Criteria**:
  - [ ] `go test ./internal/plugins/builtins/... -v -count=1` → PASS (6 tests)
  - [ ] `gurl run "test" --output slack` produces Slack-formatted output
  - [ ] `gurl run "test" --output markdown` produces GFM table + code block
  - [ ] `gurl run "test" --output csv` produces CSV row

  **QA Scenarios**:

  ```
  Scenario: Slack output plugin
    Tool: Bash
    Steps:
      1. Run: go test ./internal/plugins/builtins/... -v -run TestOutputPlugin_Slack -count=1
      2. Assert: exit code 0, output contains emoji + fenced code block
    Expected Result: "✅ GET https://example.com (200 OK)\n```json\n{...}\n```"
    Evidence: .sisyphus/evidence/task-64-slack.txt

  Scenario: CLI integration with --output flag
    Tool: Bash
    Steps:
      1. Save a test request: gurl save "plugin-test" https://httpbin.org/get
      2. Run: gurl run "plugin-test" --output minimal
      3. Assert: output is single line with status code and duration
    Expected Result: "200 OK (Nms)" — one-line minimal output
    Evidence: .sisyphus/evidence/task-64-minimal-cli.txt

  Scenario: Unknown output format error
    Tool: Bash
    Steps:
      1. Run: gurl run "plugin-test" --output nonexistent 2>&1
      2. Assert: exit code 1, error message lists available formats
    Expected Result: "unknown output format 'nonexistent', available: pretty, slack, markdown, csv, minimal"
    Evidence: .sisyphus/evidence/task-64-unknown-format.txt
  ```

  **Commit**: YES
  - Message: `plugins: add built-in output plugins (slack, markdown, csv, minimal) — extensible output formatting`
  - Files: `internal/plugins/builtins/slack.go`, `internal/plugins/builtins/markdown.go`, `internal/plugins/builtins/csv.go`, `internal/plugins/builtins/minimal.go`, `internal/plugins/builtins/register.go`, `internal/plugins/builtins/output_test.go`
  - Pre-commit: `go test ./internal/plugins/builtins/... -count=1`

- [ ] 65. Auth Plugins (Middleware Plugins)

  **What to do**:
  - **RED**: Write tests in `internal/plugins/builtins/auth_middleware_test.go`:
    - `TestAuthMiddleware_Logging` — logs request URL, method, headers (redacted auth) before send
    - `TestAuthMiddleware_Timing` — adds X-Gurl-Duration header to ResponseContext after response
    - `TestAuthMiddleware_RetryOn401` — if response is 401 and auth is configured, retry once with refreshed token
    - `TestAuthMiddleware_UserAgent` — sets User-Agent header to `gurl/<version>` if not already set
    - `TestAuthMiddleware_Chain` — timing → user-agent → logging → execute → logging (verify order)
    - `TestAuthMiddleware_CustomHeader` — middleware that injects custom header from env (e.g., X-Request-ID from env var)
  - **GREEN**: Create built-in middleware plugins in `internal/plugins/builtins/`:
    - `timing.go`: `type TimingMiddleware struct` implementing `MiddlewarePlugin`:
      - `BeforeRequest()` — records start time in context
      - `AfterResponse()` — adds `X-Gurl-Duration` to response context, logs duration
    - `useragent.go`: `type UserAgentMiddleware struct` implementing `MiddlewarePlugin`:
      - `BeforeRequest()` — sets User-Agent to `gurl/<version>` if not present
      - `AfterResponse()` — pass-through
    - `logging.go`: `type LoggingMiddleware struct` implementing `MiddlewarePlugin`:
      - `BeforeRequest()` — logs: `→ METHOD URL` with redacted auth headers
      - `AfterResponse()` — logs: `← STATUS_CODE (duration) SIZE`
    - `retry401.go`: `type Retry401Middleware struct` implementing `MiddlewarePlugin`:
      - `AfterResponse()` — if 401 and auth handler exists, trigger one re-auth + retry
      - Needs reference to auth registry (Task 21) for token refresh
    - Update `register.go`: register all middleware in correct order (timing first, logging last)
  - **REFACTOR**: Ensure middleware order is documented in help/config — timing → useragent → custom → retry → logging

  **Must NOT do**:
  - Do NOT log sensitive headers (Authorization, Cookie) — redact to "Bearer [REDACTED]" pattern
  - Do NOT retry infinitely on 401 — exactly ONE retry, then surface the 401
  - Do NOT make middleware mandatory — all can be disabled via config
  - Do NOT use if-else chain for header redaction — use map of sensitive header names

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Middleware chain with ordering, 401 retry logic, sensitive data redaction
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `cli-developer`: Middleware is internal pipeline, not CLI interaction
    - `golang-pro`: Not concurrency-focused

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 64 after Task 63)
  - **Parallel Group**: Wave 9 post-foundation (with Task 64, 66)
  - **Blocks**: None directly
  - **Blocked By**: Task 63 (plugin system — MiddlewarePlugin interface and Registry), Task 21 (auth framework — for retry401 middleware)

  **References**:

  **Pattern References**:
  - `internal/plugins/interfaces.go` (Task 63) — MiddlewarePlugin interface to implement
  - `internal/plugins/registry.go` (Task 63) — `ApplyBeforeRequest`/`ApplyAfterResponse` chain
  - `internal/auth/auth.go` (Task 21) — Auth handler registry for retry401 token refresh

  **API/Type References**:
  - `internal/plugins/interfaces.go:MiddlewarePlugin` — `BeforeRequest(*RequestContext)`, `AfterResponse(*ResponseContext)`
  - `internal/plugins/interfaces.go:RequestContext` — Mutable request + env
  - `internal/auth/auth.go:Registry` (Task 21) — `GetHandler(authType string)` for retry

  **WHY Each Reference Matters**:
  - MiddlewarePlugin interface is the contract — timing/logging/retry all implement it
  - Auth registry is needed by retry401 to refresh credentials on 401 responses
  - RequestContext.Request is mutated by middleware (add headers, modify URL)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/plugins/builtins/... -v -run TestAuthMiddleware -count=1` → PASS (6 tests)
  - [ ] User-Agent set on all requests unless overridden
  - [ ] Timing information available after request
  - [ ] Authorization header redacted in logs

  **QA Scenarios**:

  ```
  Scenario: Middleware chain order
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/plugins/builtins/... -v -run TestAuthMiddleware_Chain -count=1
      2. Assert: exit code 0, middleware called in correct order
    Expected Result: timing.Before → ua.Before → logging.Before → [exec] → logging.After → ua.After → timing.After
    Evidence: .sisyphus/evidence/task-65-chain.txt

  Scenario: Sensitive header redaction
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/plugins/builtins/... -v -run TestAuthMiddleware_Logging -count=1
      2. Assert: exit code 0, "Authorization" value replaced with "[REDACTED]"
    Expected Result: Log output shows "Authorization: Bearer [REDACTED]", not the actual token
    Evidence: .sisyphus/evidence/task-65-redaction.txt

  Scenario: 401 retry with re-auth
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/plugins/builtins/... -v -run TestAuthMiddleware_RetryOn401 -count=1
      2. Assert: exit code 0, first call returns 401, retry returns 200
    Expected Result: Middleware detects 401, re-authenticates, retries once, returns new response
    Evidence: .sisyphus/evidence/task-65-retry401.txt
  ```

  **Commit**: YES
  - Message: `plugins: add built-in middleware plugins (timing, user-agent, logging, retry401) — request pipeline`
  - Files: `internal/plugins/builtins/timing.go`, `internal/plugins/builtins/useragent.go`, `internal/plugins/builtins/logging.go`, `internal/plugins/builtins/retry401.go`, `internal/plugins/builtins/auth_middleware_test.go`
  - Pre-commit: `go test ./internal/plugins/builtins/... -count=1`

- [ ] 66. Multi-Language Code Generation

  **What to do**:
  - **RED**: Write tests in `internal/codegen/codegen_test.go`:
    - `TestCodeGen_Go` — generates Go net/http code from SavedRequest
    - `TestCodeGen_Python` — generates Python requests library code
    - `TestCodeGen_JavaScript` — generates Node.js fetch/axios code
    - `TestCodeGen_Curl` — generates curl command (round-trip: SavedRequest → curl string)
    - `TestCodeGen_Go_WithAuth` — Go code includes auth header/setup
    - `TestCodeGen_Go_WithBody` — Go code includes json.Marshal for body
    - `TestCodeGen_Python_WithHeaders` — Python code includes headers dict
    - `TestCodeGen_JavaScript_WithFormData` — JS code includes FormData for multipart
    - `TestCodeGen_UnknownLanguage` — returns error with available language list
    - `TestCodeGen_AllLanguages` — `ListLanguages()` returns all supported languages
  - **GREEN**: Create `internal/codegen/`:
    - `codegen.go`: Core interface and dispatcher:
      ```go
      type Generator interface {
          Language() string
          Generate(req *types.SavedRequest, opts *GenOptions) (string, error)
      }
      type GenOptions struct {
          IncludeComments bool
          IncludeImports  bool
          AuthConfig      *auth.Config // optional auth to bake in
      }
      func Generate(lang string, req *types.SavedRequest, opts *GenOptions) (string, error)
      func ListLanguages() []string
      ```
    - `go_gen.go`: Go code generator:
      - Generates `net/http` code with proper imports
      - Handles: GET/POST/PUT/DELETE/PATCH, headers, JSON body, auth headers
      - Includes `defer resp.Body.Close()` and `io.ReadAll` pattern
      - Uses `text/template` for code template
    - `python_gen.go`: Python code generator:
      - Generates `requests` library code
      - Handles: method, headers dict, json body, auth tuple
      - Uses f-strings for variable interpolation
    - `javascript_gen.go`: JavaScript code generator:
      - Generates `fetch()` or `axios` code (default: fetch)
      - Handles: method, headers object, JSON.stringify body, FormData for multipart
      - async/await pattern
    - `curl_gen.go`: Curl command generator:
      - Generates valid curl command from SavedRequest (reverse of parser)
      - Handles: method, headers (-H), body (-d), auth (--user)
      - Proper shell escaping for values
    - Wire CLI: `gurl codegen "request-name" --lang go` → outputs code to stdout
    - Add `--lang` flag with completion: go, python, javascript, curl
    - Add `--clipboard` flag: copy generated code to system clipboard
  - **REFACTOR**: Use `text/template` for all generators — separate code templates from logic

  **Must NOT do**:
  - Do NOT hardcode code strings with string concatenation — use text/template
  - Do NOT generate code with hardcoded secrets — use placeholder `<your-token-here>` for auth
  - Do NOT import code generation libraries — templates are simple enough for text/template
  - Do NOT support every language — start with Go, Python, JS, curl. Plugin system allows extension

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: 4 language generators, each with template logic, auth integration, shell escaping
  - **Skills**: [`cli-developer`]
    - `cli-developer`: CLI subcommand registration, clipboard integration
  - **Skills Evaluated but Omitted**:
    - `golang-pro`: Not Go-concurrency specific — this is template rendering

  **Parallelization**:
  - **Can Run In Parallel**: YES (independent of Tasks 64, 65)
  - **Parallel Group**: Wave 9 post-foundation (with Tasks 64, 65)
  - **Blocks**: None directly
  - **Blocked By**: Task 4 (HTTP client — code generators reference client.Request fields for accurate generation)

  **References**:

  **Pattern References**:
  - `internal/core/curl/parser.go` (Task 3) — Curl parser: codegen is the REVERSE operation (SavedRequest → curl)
  - `internal/cli/commands/paste.go` — Existing "copy as curl" command: simplistic version of curl_gen
  - `internal/cli/commands/export.go` — CLI subcommand pattern for `gurl codegen`

  **API/Type References**:
  - `pkg/types/types.go:SavedRequest` — Input to all generators: URL, Method, Headers, Body, Variables
  - `text/template` stdlib — Template engine for code generation
  - `github.com/atotto/clipboard` — System clipboard for `--clipboard` flag (same as TUI Task 60)
  - `internal/auth/auth.go:Config` (Task 21) — Auth configuration to generate auth code

  **External References**:
  - Insomnia code gen: generates for 15+ languages — reference for output style
  - Postman code gen: https://github.com/postmanlabs/postman-code-generators — reference implementation
  - httpbin: https://httpbin.org — test target for generated code verification

  **WHY Each Reference Matters**:
  - SavedRequest is the data model all generators consume
  - paste.go has a primitive curl generation that curl_gen replaces/extends
  - Postman's code gen repo shows the scope of per-language templates
  - text/template separates code templates from generation logic (key for maintainability)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/codegen/... -v -count=1` → PASS (10 tests)
  - [ ] `gurl codegen "request" --lang go` outputs compilable Go code
  - [ ] `gurl codegen "request" --lang python` outputs runnable Python code
  - [ ] `gurl codegen "request" --lang curl` outputs valid curl command
  - [ ] Generated curl command round-trips: parse → save → codegen curl → matches original

  **QA Scenarios**:

  ```
  Scenario: Go code generation with auth
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/codegen/... -v -run TestCodeGen_Go_WithAuth -count=1
      2. Assert: exit code 0, generated code includes req.Header.Set("Authorization", ...)
    Expected Result: Valid Go code with net/http, proper imports, auth header placeholder
    Evidence: .sisyphus/evidence/task-66-go-auth.txt

  Scenario: Python code generation
    Tool: Bash (go test)
    Steps:
      1. Run: go test ./internal/codegen/... -v -run TestCodeGen_Python -count=1
      2. Assert: exit code 0, generates `import requests` + `requests.get(url, headers=...)`
    Expected Result: Runnable Python code with requests library
    Evidence: .sisyphus/evidence/task-66-python.txt

  Scenario: Curl round-trip
    Tool: Bash
    Steps:
      1. Save: gurl save "roundtrip" --curl "curl -X POST -H 'Content-Type: application/json' -d '{\"key\":\"val\"}' https://httpbin.org/post"
      2. Generate: gurl codegen "roundtrip" --lang curl
      3. Assert: output contains -X POST, -H 'Content-Type: application/json', -d with body, URL at end
    Expected Result: Generated curl command preserves method, headers, body, URL
    Evidence: .sisyphus/evidence/task-66-curl-roundtrip.txt

  Scenario: Unknown language error
    Tool: Bash
    Steps:
      1. Run: gurl codegen "roundtrip" --lang ruby 2>&1
      2. Assert: exit code 1, error lists available languages
    Expected Result: "unsupported language 'ruby', available: go, python, javascript, curl"
    Evidence: .sisyphus/evidence/task-66-unknown-lang.txt
  ```

  **Commit**: YES
  - Message: `codegen: add multi-language code generation (Go, Python, JS, curl) — export requests as code`
  - Files: `internal/codegen/codegen.go`, `internal/codegen/go_gen.go`, `internal/codegen/python_gen.go`, `internal/codegen/javascript_gen.go`, `internal/codegen/curl_gen.go`, `internal/codegen/codegen_test.go`, `internal/cli/commands/codegen.go`
  - Pre-commit: `go test ./internal/codegen/... -count=1`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go test ./... -count=1`. Review all changed files for: `interface{}` when concrete type works, empty error handling, `fmt.Println` in prod code, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp). Verify NO if-else-if-else chains exist.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high` 
  Start from clean state. Test: `gurl save` with complex curl commands, `gurl run` with auth, `gurl env` workflows, `gurl diff` with response bodies, all import formats still work, collection runner with assertions. Save evidence to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff. Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

Each commit is ONE of: (a) a failing test, (b) code making a test pass, (c) a refactoring with no behavior change.
Format: `<scope>: <verb> <what> — <why>`

- **T1**: `rename: scurl→gurl across all paths, configs, env vars, docs`
- **T3**: `parser: rewrite curl parser using shell tokenization — regex can't handle quoting`
- **T4**: `client: add internal/client with net/http executor — replace curl shelling`
- **T10**: `run: migrate to internal/client — unify execution path`
- **T15**: `env: add environment system with variable scoping`
- **T21**: `auth: add auth framework + Basic auth handler`
- **T41**: `graphql: add GraphQL client with introspection`
- **T45**: `scripting: add JavaScript runtime via goja`
- **T49**: `assertions: add assertion engine with declarative syntax`
- **T50**: `runner: add collection runner with CI exit codes`
- **T57**: `tui: add bubbletea foundation + main layout`
- **T63**: `plugins: add plugin system architecture`

---

## Success Criteria

### Verification Commands
```bash
go test ./... -count=1              # Expected: ALL PASS, 0 failures
go vet ./...                        # Expected: no issues
go build ./cmd/gurl                 # Expected: clean build
gurl save test --curl "curl -X POST -H 'Auth: Bearer tok' -d '{\"a\":1}' https://httpbin.org/post"  # Expected: saves correctly
gurl run test --env dev             # Expected: uses dev env, captures response body
gurl history test                   # Expected: shows response body + metadata
gurl diff test                      # Expected: shows body diff between runs
gurl env list                       # Expected: shows all environments
gurl run test --auth basic --user u:p  # Expected: authenticates
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass (go test ./...)
- [ ] All 4 import formats still work
- [ ] TUI launches and shows request list
- [ ] Collection runner produces JUnit XML report
