# Gurl Project Plan

## Overview
Rebuild the terminal-curl project as **Gurl** - a smart curl saver and API companion with import support for OpenAPI, Insomnia, and other formats.

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-07 | Rename to "Gurl" | Memorable, short, Google-proof |
| 2026-04-07 | GitHub Actions | Standard, well-documented, free tier sufficient |
| 2026-04-07 | TOML config | Human-readable, no external deps |
| 2026-04-07 | LMDB storage | Fast, embedded, single-file |

---

## TODO: Create Comprehensive README

**Tasks:**
- [ ] Project title and one-liner
- [ ] Feature list (from PRD)
- [ ] Quick start / installation
- [ ] Usage examples for all commands
- [ ] Configuration guide (TOML)
- [ ] Plugin system documentation
- [ ] Contributing guidelines
- [ ] License
- [ ] Badges (build, Go version)

---

## TODO: Setup Multi-Platform CI/CD

**Tasks:**
- [ ] `.github/workflows/build.yml`
- [ ] Matrix: {os: [linux, macos, windows], arch: [amd64, arm64]}
- [ ] Go 1.21+ for all builds
- [ ] Generate SHA256 checksums
- [ ] Create GitHub Release on tag
- [ ] Upload binaries as release assets

**Build Targets:**
```
gurl-linux-amd64
gurl-linux-arm64
gurl-darwin-amd64
gurl-darwin-arm64
gurl-windows-amd64.exe
```

---

## TODO: Create Release Script

**Tasks:**
- [ ] `scripts/release.sh`
- [ ] Semantic versioning (major.minor.patch)
- [ ] Changelog generation
- [ ] Build all platforms
- [ ] Create git tag
- [ ] Push to remote
- [ ] GitHub release via gh CLI

---

## TODO: Add Import Support

**Importers to implement:**

1. **OpenAPI/Swagger (3.0+)**
   - Parse YAML/JSON spec
   - Extract endpoints, methods, parameters
   - Convert to Gurl requests
   - Group by tag or path

2. **Insomnia (v10+)**
   - Parse `.json` export
   - Extract requests, environments
   - Handle folders/collections

3. **Bruno**
   - Parse `.bru` files in directories
   - Extract request definitions
   - Handle environments

4. **Postman (v2.1)**
   - Parse `collection.json`
   - Extract requests, variables

5. **HAR files**
   - Parse HTTP Archive format
   - Extract requests from log

**Import Command:**
```bash
gurl import openapi ./openapi.yaml --collection orders
gurl import insomnia ./insomnia.json
gurl import bruno ./requests/
```

---

## TODO: Create Journey.md

**Purpose:** Document the human decisions, discussions, and evolution of the project.

**Sections:**
- [ ] Project inception (why Gurl?)
- [ ] Key design decisions
- [ ] Rejected alternatives
- [ ] Lessons learned
- [ ] Future roadmap ideas

---

## TODO: Deterministic Code Rules

**Rule:** Never use if-else-if-else chains. Use match/switch with explicit cases or early returns.

**Pattern:**
```go
// BAD
if x == "a" {
    doA()
} else if x == "b" {
    doB()
} else {
    doDefault()
}

// GOOD - Option 1: Switch
switch x {
case "a":
    doA()
case "b":
    doB()
default:
    doDefault()
}

// GOOD - Option 2: Early return
if x == "a" {
    doA()
    return
}
if x == "b" {
    doB()
    return
}
doDefault()

// GOOD - Option 3: Match with Map
actions := map[string]func(){
    "a": doA,
    "b": doB,
}
if fn, ok := actions[x]; ok {
    fn()
} else {
    doDefault()
}
```

---

## TODO: Code Quality

**Linter Configuration:**
- golangci-lint with strict settings
- no else in if blocks
- error handling checks
- cyclomatic complexity limits

---

## Completed

- [x] Rename project to Gurl
- [x] README.md
- [x] GitHub Actions CI/CD (build.yml)
- [x] Install script (scripts/install.sh)
- [x] Import system (OpenAPI, Insomnia, Bruno, Postman, HAR)
- [x] Core CRUD: save (basic), run, list, delete, rename
- [x] Storage: LevelDB with indices
- [x] History tracking + timeline
- [x] Collections + tags
- [x] Template engine + variable detection
- [x] Export (JSON) + paste (clipboard)
- [x] Update command (self-update from GitHub releases)
- [x] Shell completions (bash, zsh, fish)
- [x] Journey.md

## In Progress

### Fix: Build workflow binary naming (v0.1.10)
- [x] Fix `ubuntu-latest` → `linux` in binary names (build.yml)
- [ ] Tag and release v0.1.10 to publish correctly-named binaries

### Enhance: Save command (full curl support)
The save command currently only accepts `<name> <url>` and stores as GET with no headers/body.
The data model (`SavedRequest`) already supports headers, body, method, and curl_cmd — they're just not wired into the save command.

**Implementation tasks:**
- [ ] Add `-X/--method` flag for HTTP method
- [ ] Add `-H/--header` flag (repeatable) for headers
- [ ] Add `-d/--data` flag for request body
- [ ] Add `--curl` flag for raw curl string parsing (uses existing `internal/core/curl/parser.go`)
- [ ] Add stdin pipe detection for curl strings
- [ ] Auto-detect POST when body present and no method specified
- [ ] Store original curl command in `CurlCmd` field
- [ ] Fix `--tag` flag to support multiple values (currently single StringFlag)
- [ ] Wire in variable detection from `internal/core/curl/detector.go`

**Files to modify:**
- `internal/cli/commands/save.go` — main enhancement target
- `internal/core/curl/parser.go` — may need fixes for multiline/complex bodies

---

## TODO: Remaining Feature Gaps

### Stub commands that need implementation
- [ ] `detect` command — interactive curl parsing flow (Phase 2, TUI)
- [ ] `edit` command — modify saved requests (TUI form)
- [ ] `diff` command — needs response body storage in history

### Config integration
- [ ] Wire `internal/config/` loader into commands
- [ ] Respect config settings (history_depth, auto_template, output format, etc.)

### Testing
- [ ] Integration tests for save → run round-trip
- [ ] Parser tests for complex curl strings
- [ ] Storage layer tests beyond basic mocks

---

## Implementation Order (Next)

1. **Enhance save command** (headers, body, method, curl parsing)
2. **Fix build workflow** (binary naming) + release v0.1.10
3. **Wire config** into commands
4. **Implement detect** command
5. **Implement edit** command
6. **Add response storage** for diff
7. **Expand tests**

---

## Agent Assignments

| Agent | Tasks |
|-------|-------|
| Agent-1 | README + GitHub Actions + Release script |
| Agent-2 | Import system architecture + OpenAPI |
| Agent-3 | Insomnia + Bruno importers |
| Agent-4 | Code review + Journey.md |
