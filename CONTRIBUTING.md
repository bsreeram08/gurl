# Contributing to gurl

Thanks for helping. This guide covers building, testing, how the code fits together, and what a pull request needs.

## Prerequisites

- Go 1.25 or newer (`go.mod` declares `go 1.25.0`, toolchain `go1.26.3`; CI builds with 1.26.3).
- Git. Clone with `--recurse-submodules` if you will work on the docs site: its Hugo theme is a submodule.
- [Hugo](https://gohugo.io/) (extended), only to preview the docs site.

## Build, test, vet

```bash
git clone https://github.com/bsreeram08/gurl.git && cd gurl
go build -o gurl ./cmd/gurl         # dev build; `make build` stamps the version from git
./gurl --help

go test ./...                        # what CI runs (also: make test)
go test -race ./internal/runner/...  # use -race on every package you touched (make test-race for all)
go vet ./...
gofmt -w <files you changed>         # format what you touch
```

To try a change without touching your real data, point gurl at a scratch database: `GURL_DB_PATH=/tmp/gurl-dev.db GURL_COOKIE_DB_PATH=/tmp/gurl-dev-cookies.db ./gurl ...`.

## Project layout

```text
cmd/gurl/               entry point: wires storage, env store and plugins into the command tree (newApp)
internal/cli/commands/  one file per command (run.go, env.go, diag.go, docs.go, agent.go, ...)
internal/resolve/       request resolution: saved request + scopes -> exact request, or resolution problems
internal/redact/        masking rules for everything gurl prints or stores
internal/runner/        execution: runs, chains, collection runs, data rows, execution records, exit codes
internal/storage/       storage.DB interface; LevelDB-backed store, project (.gurl/) file store, history
internal/storage/memstore/  in-memory storage.DB for tests
internal/env/           environments: Store interface, LevelDB and file stores, secrets, aliases, persist
internal/secrets/       AES-256-GCM encryption helpers
internal/protocols/     graphql, grpc, websocket, sse commands
internal/...            auth, client, codegen, importers, extract, assertions, scripting, plugins, ...
pkg/types/              shared types (SavedRequest, ExecutionHistory, ...)
docs/                   Hugo documentation site (published by the Docs workflow)
scripts/                install and release scripts
```

## Architecture notes

[CONTEXT.md](CONTEXT.md) is the glossary. Use its terms in code, comments, commit messages and docs.

- **Request resolution** happens in one place: `internal/resolve`. Commands don't substitute placeholders themselves. Resolution either produces the exact request to send or a list of **resolution problems** (missing variable, leftover placeholder, bad type, unknown dynamic function), and any problem stops the send.
- **Scopes** override each other, lowest first: environment < collection < data row < CLI < runtime. Placeholders are filled in a single pass, so a value that still contains a placeholder is a problem, not something to expand again.
- **Variable origin** is the scope a variable's final value came from. `--persist` writes back to the origin, so keep the origin attached as values move through the runner.
- **Execution records** (`types.ExecutionHistory`) are written for every run, including failures before a response arrived, and are what `history`, `diag` and `diff` read. Mask secret values before you save a record (see `internal/redact`).
- **Storage** goes through interfaces: `storage.DB` for requests, collections and history, and `env.Store` for environments. Commands take these as constructor arguments, so they work the same against LevelDB, `.gurl/` project files or in-memory fakes.

## Testing conventions

- Test through public entry points: the command constructors (`commands.RunCommand(db, envStore)`, ...) or `newApp` in `cmd/gurl`.
- Use the in-memory fakes, not the real database: `memstore.New()` for `storage.DB` and `env.NewMemoryStore()` for `env.Store`.
- Capture output with the existing helpers instead of writing new ones, for example `captureStdout` / `captureStreams` in `internal/cli/commands` and `runWithArgs` in `cmd/gurl`.
- Use `httptest.NewServer` for HTTP. Tests must not reach the network.
- Fixtures contain no real secrets: use obvious fakes like `sk-test-123`. When you test masking, assert that the fake value does **not** appear in the output.
- `gurl docs --json` must stay under 32KB, and every visible command must appear in it (`cmd/gurl/agent_test.go`).

## Security rules

- Never log, print, store or export a secret value. Show secret and credential-looking values through `internal/redact` (`redact.Mask` is `*****`).
- Execution records, dry runs, `show`, `list --json`, `codegen` and `paste` mask secrets by default. `--reveal` is the only way out, and agent mode refuses it.
- Agent mode (`--agent`, `GURL_AGENT=1`, `agentgurl`) also masks secrets in response bodies, headers and errors. A new output path has to respect `jsonOutput`, `noInput` / `canPrompt` and `rejectReveal` in `internal/cli/commands/agent.go`.
- Resolution problem messages name the variable, never its value.
- To report a vulnerability, see [SECURITY.md](SECURITY.md). Don't open a public issue.

## Commit style

Conventional prefixes with an optional scope, as in the git log: `feat(collection): ...`, `fix(redact): ...`, `docs: ...`, `test: ...`, `refactor: ...`, `ci: ...`, `build: ...`, `chore: ...`.

- Write the subject in the imperative and keep it short. Use the body to explain why.
- **Don't add `Co-Authored-By` trailers**, including AI-assistant attribution.
- Keep each commit to one logical change.

## Pull request checklist

- [ ] Tests cover the change, and `go test ./...` passes (`-race` on touched packages).
- [ ] `go vet ./...` is clean, and every file you changed is `gofmt`-formatted.
- [ ] `CHANGELOG.md`: add an entry under `## [Unreleased]` at the top (create the heading if it is missing), using Added / Changed / Fixed.
- [ ] User-facing change: update `README.md` and/or `docs/`, and check that `gurl docs` still describes it (flag usage strings are the docs).
- [ ] No secret values in code, fixtures, logs or output.

## Release process

Maintainers only:

1. Move the `Unreleased` entries in `CHANGELOG.md` under `## [vX.Y.Z] - YYYY-MM-DD` and merge to `master`.
2. Tag `master` and push the tag: `scripts/release.sh vX.Y.Z` (or `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`).
3. The **Build** workflow then tests and builds darwin/linux amd64/arm64 binaries, publishes the GitHub release with `SHA256SUMS`, pushes the multi-arch image to `ghcr.io/bsreeram08/gurl` (`vX.Y.Z`, and `latest` for non-prerelease tags), and commits the updated Homebrew formula (`Formula/gurl.rb`) to `master`.
