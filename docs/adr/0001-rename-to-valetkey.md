# 1. Rename gurl to valetkey and position it as an API tool for AI agents

Date: 2026-11-19
Status: Accepted

## Context

gurl started as a curl saver: a terminal library of named HTTP requests. Over time it grew the parts an AI agent needs to call APIs safely: agent mode (JSON, no prompts, `--reveal` refused, secrets masked in response bodies), `exec` (secrets passed to a subprocess without being printed), execution records with `diag`, and a docs reference generated from the binary.

A market scan (November 2026) found two groups of tools, with a gap between them:

- **API clients** (Postman, Insomnia 12, Bruno, Hoppscotch) add AI as an assistant inside the app, or offer an MCP server over their own data. None of them treats the agent as an untrusted caller that must never hold the keys.
- **Agent and MCP gateways and credential brokers** (Docker MCP Gateway, ToolHive, the Linux Foundation's agentgateway, Kong AI Gateway, Cloudflare MCP portals, Portkey, Composio, Arcade, Infisical Agent Vault) inject credentials, but they proxy whole MCP servers or hosts, or are hosted catalogs. They usually need containers, Kubernetes or a SaaS account, and none has a library of saved requests, environments, run records or diagnosis.

No tool combined a git-friendly library of saved requests with an agent contract and secrets that never reach the agent. gurl already did all three, but its name worked against it:

- At least five other curl-like "gurl" projects exist on GitHub, including a Postman-style TUI.
- gurl.dev, gurl.io and gurl.sh are all registered.
- The name reads as "girl"/"curl" and ties the project to curl replacements.

## Decision

1. **Name: valetkey.** A valet key lets someone drive your car without opening the trunk or glovebox, which is what an agent gets here. "Valet Key" is also a known cloud design pattern for scoped, limited access.
   - We checked candidates against domains, USPTO marks (classes 9 and 42) and GitHub. For valetkey, valetkey.sh was available, no live USPTO mark was found, and there were only 11 small repos. That search is not a legal clearance.
   - Rejected: reqd (shorter, but the meaning is unclear) and ushr (a live USHER mark for access-gateway software).
   - Several dictionary words had their main domains taken: wicket (which also has live software marks), errand, tollgate, postern, lanyard and tessera.
   - Avoid "agent gateway", "agent vault" and "harness": other products already use them.
2. **Positioning:** "Saved API requests your AI agents can run without ever seeing the keys." Insomnia/Postman for agents, local-first. The saved request is the unit of permission.
3. **Staged roadmap.**
   - Now: repositioning, the rename with a compatibility layer, and `valetkey mcp`.
   - Later: a local gateway or proxy mode, with per-environment host allowlists, approval prompts for chosen requests, rate limits, and an audit log built from execution records.
4. **Compatibility.** The compatibility rules live in `internal/brand`, the only place that knows both names.
   - Every lookup prefers the new name and falls back to the old one: environment variables, data/config/plugin directories, the project directory, rc files, binary names and the script global.
   - `valetkey migrate` renames the old locations atomically and never overwrites an existing destination.
   - Release assets and the container image are published under both names during the transition.
   - Homebrew moves users with `formula_renames.json`.
5. **Frozen identifiers.** Changing these would break data or bundles that already exist, so they keep their gurl names: the `gurlenc:v1:` secret prefix, the `gurl-share` bundle kind and SSH signature namespace, the `gurl` keychain service, and the `.gurl` export format.
6. **MCP safety rule.** A tool call may only fill placeholders that the pinned environment and the collection leave undefined. Otherwise an agent could override `BASE_URL` and send the request's secrets to a host it controls. Calls run through `run --agent` in a child process, so they inherit agent-mode masking, records and exit codes without a second code path.

## Consequences

- The Go module path changes to `github.com/bsreeram08/valetkey`. Nothing imports it from outside, so only this repository is affected.
- The maintainer must rename the GitHub repository to `bsreeram08/valetkey` **before the first valetkey release**. The new self-updater, installer, Homebrew job and image tags all point there. GitHub redirects the old repo URLs, but not the Pages docs site, so `bsreeram08.github.io/gurl` links will break.
- After `valetkey migrate`, releases from before the rename no longer see the data. `valetkey` keeps reading the old locations until then, so migration is optional.
- Docker users who mount `/home/gurl/...` must switch to `/home/valetkey/...`.
- The frozen identifiers will keep the old name visible in a few places, by design.
