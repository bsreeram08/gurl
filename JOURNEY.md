# Gurl Journey

> The story of building Gurl, told through human decisions.

**GURL** = **G**url's **U**niversal **R**equest **L**ibrary

*(Like GNU = "GNU's Not Unix")*

---

## The Beginning

**Date:** 2026-04-07

### Why Gurl?

**Problem:** The user's curl workflow was chaotic:
- 47 variations of the same endpoint in shell history
- Ctrl+R fatigue searching for "that one request"
- Copy-paste from Cursor to terminal constantly
- No easy way to share API requests with AI agents

**Solution (in progress):** A CLI tool that saves, organizes, and replays curl requests.

---

## Decisions Log

| Date | Decision | Rationale | Alternative Considered |
|------|----------|-----------|----------------------|
| 2026-04-07 | Name: Gurl | Memorable, short, Google-proof | gurl, rcurl, hurl, acurl |
| 2026-04-07 | GURL acronym | "Gurl's Universal Request Library" - like GNU | Various alternatives |
| 2026-04-07 | Go over Zig | Faster development, better CLI ecosystem (cobra, bubbletea) | Zig for performance |
| 2026-04-07 | TOML config | Native Go library, human-readable | YAML, JSON |
| 2026-04-07 | LMDB storage | Fast, embedded, single-file DB | SQLite, badger |
| 2026-04-07 | urfave/cli | Standard Go CLI, good docs | cobra, kingpin |
| 2026-04-07 | bubbletea TUI | Elm architecture, declarative | tview, gocui |
| 2026-04-07 | Plugin system | Extensible by design | Hardcoded features |
| 2026-04-07 | Agent API | AI agents can use Gurl too | CLI only |
| 2026-04-08 | goja for JS scripting | Embedded JS runtime in Go — real JS, not shell strings | Lua, Rhai, no scripting |
| 2026-04-08 | AES-256-GCM for secrets | Authenticated encryption, never in logs or codegen output | .netrc, env files, plaintext |
| 2026-04-08 | GraphQL as first-class | Devs use GQL heavily; "import as raw curl" was a cop-out | Staying HTTP-only |
| 2026-04-08 | Multi-protocol (gRPC, WS, SSE) | Hurl and Insomnia have these; gaps block serious API devs | HTTP-only scope |
| 2026-04-09 | Reposition as API workbench | "curl wrapper" undersells the tool; Postman is the real comparison | Keep curl framing |
| 2026-04-09 | Homebrew tap (personal) | Fast path to `brew install` without waiting for main tap approval | Main tap only |

---

## Rejected Ideas

| Idea | Why Rejected |
|------|-------------|
| Web dashboard | Outside terminal focus |
| Team sync | Adds complexity, local-first first |
| Request chaining | Handled by `sequence` command instead |

---

## Technical Philosophy

### Deterministic Programming
"No if-else-if-else chains. Ever."

Why:
- Easier to reason about
- Forces explicit case handling
- No hidden fallthrough bugs

### Local-First
- No cloud dependency
- Data stays on machine
- Export/import for sharing

### Plugin Everything
- Every feature can be extended
- Drop-in plugins in ~/.config/gurl/plugins/
- Middleware, output, commands

---

## Milestones

### v0.1.0 - Core Foundation ✅
- [x] Save curl requests by name
- [x] Run by name with variables
- [x] List and search
- [x] Persistent storage

### v0.2.0 - Import System ✅
- [x] OpenAPI/Swagger import
- [x] Insomnia import
- [x] Bruno import
- [x] Postman import
- [x] HAR import

### v0.3.0 - History & Timeline ✅
- [x] Execution history per request
- [x] Global timeline view
- [x] Diff responses between runs

### v0.4.0 - Auth & Security ✅
- [x] AES-256-GCM encrypted secrets
- [x] OAuth 1/2, AWS SigV4, Digest, NTLM
- [x] Cookie handling, redirects, proxy, mTLS

### v0.5.0 - Protocols, Scripting, Assertions ✅
- [x] GraphQL
- [x] gRPC
- [x] WebSocket
- [x] SSE
- [x] JavaScript pre/post-request hooks (goja)
- [x] Response assertions (status, headers, JSONPath, XPath)
- [x] Collection runner with CSV/JSON data-driven input

### v0.6.0 - TUI & Output ✅
- [x] Full bubbletea TUI with arrow navigation
- [x] Interactive picker for `gurl run`
- [x] Syntax highlighting, JSON pretty-print
- [x] Code generation (curl, Go, Python, JavaScript)
- [x] Plugin system (middleware + output formatters)

### v0.1.15–v0.1.17 - Hardening ✅
- [x] Security hardening (21 issues resolved)
- [x] Automated Homebrew formula publishing
- [x] Self-update command
- [x] Comprehensive test coverage

### v0.1.18 - Security Hardening II ✅
- [x] AWSv4: inject host header from URL when missing
- [x] OAuth2: remove HTTP status code from error message (info disclosure)
- [x] Sandbox: block eval() and Function() constructor (sandbox escape prevention)
- [x] Sandbox: make crypto.digest() and Buffer.from() throw uncatchable JS errors
- [x] Template: single-pass regexp substitution (deterministic, prevents injection)
- [x] Update: real SHA256 checksum verification with subtle.ConstantTimeCompare
- [x] Path traversal: symlink-aware validation for import/export/run commands

### Next
- [ ] OpenAPI request validation (not just import)
- [ ] Git-friendly collection format (human-readable, diffs well)
- [ ] Homebrew main tap (`brew install gurl` without tap)
- [ ] AI-assisted test generation

---

## Lessons Learned

1. **Spec first, code second** — The PRD saved hours of rework
2. **Grill yourself** — The "grill-me" session revealed hidden requirements
3. **Agents work better with plans** — Subagents with clear deliverables
4. **"Rejected" is a snapshot, not a verdict** — GraphQL, WebSocket, environments were all rejected at day 1 and all shipped by week 1
5. **Positioning matters early** — "curl wrapper" attracted the wrong mental model; "API workbench" is the right frame
6. **Security is not a phase** — 21 issues caught in one audit pass; bake it in from the start

---

## Future Ideas

- [ ] OpenAPI validation (validate requests against spec before sending)
- [ ] Git-native collection format (`.gurl` files that diff cleanly)
- [ ] Mock server (serve saved responses locally)
- [ ] Team sync via Git (official story: store collections in Git, sync via PRs)
- [ ] AI-assisted assertion generation from response bodies
- [ ] VS Code extension
- [ ] Homebrew main tap

---

*Last updated: 2026-04-11*
