## What and why

<!-- What changes, and why. Link the issue if there is one. -->

## Checklist

- [ ] Tests cover the change; `go test ./...` passes (`-race` on touched packages)
- [ ] `go vet ./...` is clean; changed files are `gofmt`-formatted
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`
- [ ] User-facing change: README / docs updated, and `gurl docs` still describes it
- [ ] No secret values in code, fixtures, logs or output
- [ ] Commit messages use conventional prefixes and have no `Co-Authored-By` trailers
