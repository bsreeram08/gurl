# AGENTS.md

This guide is for AI coding agents working **on** the valetkey repository (formerly gurl). If you want to **use** valetkey, run `valetkey docs` instead.

valetkey is a Go CLI: a terminal library of named HTTP requests, run against environments and collections, that AI agents can drive (`--agent`, `valetkey mcp`) without seeing secrets. Module: `github.com/bsreeram08/valetkey`. CLI framework: `urfave/cli/v3`. Storage: goleveldb plus optional `.valetkey/` (or legacy `.gurl/`) project files.

Read [CONTEXT.md](CONTEXT.md) first. It defines the terms used everywhere: saved request, placeholder, scope, variable origin, request resolution, resolution problem, alias, dry run, persist. Use those words, not "template" or "substitution".

## Commands

```bash
go build -o valetkey ./cmd/valetkey
go test ./...
go test -race ./internal/<pkg>/...      # on every package you touched
go vet ./...
VALETKEY_DB_PATH=/tmp/v.db VALETKEY_COOKIE_DB_PATH=/tmp/vc.db ./valetkey ...   # manual runs without touching real data
```

## Layout

| Path | Owns |
|------|------|
| `cmd/valetkey/main.go` | `newApp`: the command tree; agent-mode detection (`agentvaletkey`, `agentgurl`) |
| `internal/brand/` | The product name and every legacy (`gurl`) fallback: env vars, data/config/plugin dirs, project dir, binary names |
| `internal/cli/commands/` | One file per command; `agent.go` (JSON / no-input / reveal rules); `docs.go`, `docs_sections.go` (`valetkey docs`); `mcp.go` (MCP server) |
| `internal/resolve/` | Request resolution, the only place placeholders are filled |
| `internal/redact/` | Masking rules for everything printed or stored |
| `internal/runner/` | Runs, chains, collection and data-driven runs, execution records, exit codes |
| `internal/storage/` (+ `memstore/`) | `storage.DB` interface, LevelDB and project file stores, in-memory fake |
| `internal/env/` | `env.Store` interface, environments, secrets, aliases, persist; `NewMemoryStore()` fake |
| `docs/` | Hugo site. Don't edit it as a side effect of code changes unless asked |

## Invariants

1. **Secrets are never logged, printed, stored in plain text or exported.** Mask through `internal/redact`. Execution records are masked before saving. Only `--reveal` shows secrets, and agent mode refuses it.
2. **One resolution module.** All placeholder filling, scope merging and problem detection go through `internal/resolve`. Don't add string replacement of `{{...}}` anywhere else. Fill in a single pass; a leftover placeholder is a resolution problem.
3. **Storage through interfaces.** Commands receive `storage.DB` and `env.Store` and never open the database themselves. Tests use `memstore.New()` and `env.NewMemoryStore()`.
4. **Agent mode holds for every output path.** Use `jsonOutput(c)`, `canPrompt(c, ...)` / `noInput(c)` and `rejectReveal(c)` from `agent.go`. In JSON mode stdout carries only JSON, and errors are `{"ok":false,"error":"..."}`.
5. **Exit codes:** 0 success, 1 error, 2 usage (`cli.Exit(msg, 2)`, as in `agent.go`). `collection run` maps results through `runner.DetermineExitCode` (3 means an empty collection). Changing a code breaks scripts, so treat it as a breaking change.
6. **Names go through `internal/brand`.** Don't hard-code `valetkey`, `gurl`, `VALETKEY_`/`GURL_` or data paths; use `brand.Getenv`, `brand.DataDir` and friends so legacy installs keep working. These identifiers are frozen and must not change: `gurlenc:v1:` (secret prefix), `gurl-share` (bundle kind and SSH signature namespace), the `gurl` keychain service, and the `.gurl` export format.
7. **MCP tools can't override defined values.** A tool call may only set fillable placeholders; it runs through `run --agent` in a child process. Keep it that way.
8. **No real secrets in fixtures.** Use fakes like `sk-test-123` and assert that they don't appear in output.

## Adding a command

1. Create `internal/cli/commands/<name>.go` with a `func <Name>Command(db storage.DB, ...) *cli.Command` constructor. Give it `Usage`, `ArgsUsage` if it takes arguments, and a clear `Usage` on every flag.
2. Register it in `newApp` in `cmd/valetkey/main.go`.
3. If it prints data, add `--json` (`-j`) and honor `jsonOutput(c)`. If it can prompt, gate the prompt with `canPrompt`. If it can show secrets, call `rejectReveal`.
4. `valetkey docs` is generated from the command tree, so a visible command and its flags appear automatically. Only add to `docs_sections.go` for a concept the flags can't describe, and keep it short.
5. Keep `valetkey docs --json` under the **32KB budget**. `TestDocsJSONListsEveryCommandCompactly` in `cmd/valetkey/agent_test.go` enforces it and checks that every visible command is listed. Hidden commands (`Hidden: true`) are left out.
6. Add tests, an entry under `## [Unreleased]` in `CHANGELOG.md`, and README updates if users will notice the change.

## Commits

Conventional prefixes (`feat(scope):`, `fix:`, `docs:`, `test:`, `refactor:`, `ci:`, `chore:`). **No `Co-Authored-By` trailers.** See [CONTRIBUTING.md](CONTRIBUTING.md) for the PR checklist and release process.

## Agent skills

### Issue tracker

Issues live in GitHub Issues for bsreeram08/valetkey, managed with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Default labels: needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
