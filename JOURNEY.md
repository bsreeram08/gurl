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
| 2026-04-07 | Name: Gurl | Memorable, short, Google-proof | scurl, rcurl, hurl, acurl |
| 2026-04-07 | GURL acronym | "Gurl's Universal Request Library" - like GNU | Various alternatives |
| 2026-04-07 | Go over Zig | Faster development, better CLI ecosystem (cobra, bubbletea) | Zig for performance |
| 2026-04-07 | TOML config | Native Go library, human-readable | YAML, JSON |
| 2026-04-07 | LMDB storage | Fast, embedded, single-file DB | SQLite, badger |
| 2026-04-07 | urfave/cli | Standard Go CLI, good docs | cobra, kingpin |
| 2026-04-07 | bubbletea TUI | Elm architecture, declarative | tview, gocui |
| 2026-04-07 | Plugin system | Extensible by design | Hardcoded features |
| 2026-04-07 | Agent API | AI agents can use Gurl too | CLI only |

---

## Rejected Ideas

| Idea | Why Rejected |
|------|-------------|
| Web dashboard | Outside terminal focus |
| Team sync | Adds complexity, prefers local-first |
| GraphQL support | Can import as raw curl |
| Request chaining | Future phase, not MVP |

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

### v0.1.0 - Core Foundation
- [x] Save curl requests by name
- [x] Run by name with variables
- [x] List and search
- [ ] LMDB persistence (currently in-memory)

### v0.2.0 - Import System
- [ ] OpenAPI/Swagger import
- [ ] Insomnia import
- [ ] Bruno import

### v0.3.0 - History & Timeline
- [ ] Execution history
- [ ] Timeline view
- [ ] Diff responses

---

## Lessons Learned

1. **Spec first, code second** - The PRD saved hours of rework
2. **Grill yourself** - The "grill-me" session revealed hidden requirements
3. **Agents work better with plans** - Subagents with clear deliverables

---

## Future Ideas (Not Implemented)

- [ ] Request templates with multiple variables
- [ ] Environment variables (dev/staging/prod)
- [ ] Cloud sync with E2E encryption
- [ ] VS Code extension
- [ ] HTTP/2 and WebSocket support

---

*Last updated: 2026-04-07*
