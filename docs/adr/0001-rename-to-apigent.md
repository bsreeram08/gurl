# 1. Rename gurl to apigent and position it as an API tool for AI agents

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

1. **Name: apigent** (API + agent). It says what the tool is: APIs, run by agents.
   - The first choice, valetkey, was dropped before release. It is one letter from VALKEY, a registered class 9 software mark (the Linux Foundation's Redis fork) in the same developer-tools space, and "valet" is already Laravel Valet.
   - We checked candidates against USPTO marks (classes 9 and 42) and GitHub. apigent had no live software mark (the nearest is AIGENT, customer-service software) and only a dozen small repos. That search is not a legal clearance.
   - Rejected: apiary (Oracle's APIARY mark for API design software), agentapi (Coder's AgentAPI), apiai (Google's former API.AI), steward and writ (live software marks), courier, porter, envoy and vouch (existing API or proxy tools), reqd (meaning unclear) and ushr (a live USHER mark for access-gateway software).
   - Domains don't constrain the name: the docs live on GitHub Pages or a subdomain we own.
   - Avoid "agent gateway", "agent vault" and "harness": other products already use them.
2. **Positioning:** "Saved API requests your AI agents can run without ever seeing the keys." Insomnia/Postman for agents, local-first. The saved request is the unit of permission.
3. **Staged roadmap.**
   - Now: repositioning, the rename with a compatibility layer, and `apigent mcp`.
   - Later: a local gateway or proxy mode, with per-environment host allowlists, approval prompts for chosen requests, rate limits, and an audit log built from execution records.
4. **Compatibility.** The compatibility rules live in `internal/brand`, the only place that knows both names.
   - Every lookup prefers the new name and falls back to the old one: environment variables, data/config/plugin directories, the project directory, rc files, binary names and the script global.
   - `apigent migrate` renames the old locations atomically and never overwrites an existing destination.
   - Release assets and the container image are published under both names during the transition.
   - Homebrew moves users with `formula_renames.json`.
5. **Frozen identifiers.** Changing these would break data or bundles that already exist, so they keep their gurl names: the `gurlenc:v1:` secret prefix, the `gurl-share` bundle kind and SSH signature namespace, the `gurl` keychain service, and the `.gurl` export format.
6. **MCP safety rule.** A tool call may only fill placeholders that the pinned environment and the collection leave undefined. Otherwise an agent could override `BASE_URL` and send the request's secrets to a host it controls. Calls run through `run --agent` in a child process, so they inherit agent-mode masking, records and exit codes without a second code path.

## Consequences

- The Go module path changes to `github.com/bsreeram08/apigent`. Nothing imports it from outside, so only this repository is affected.
- The maintainer must rename the GitHub repository to `bsreeram08/apigent` **before the first apigent release**. The new self-updater, installer, Homebrew job and image tags all point there. GitHub redirects the old repo URLs, but not the Pages docs site, so `bsreeram08.github.io/gurl` links will break.
- After `apigent migrate`, releases from before the rename no longer see the data. `apigent` keeps reading the old locations until then, so migration is optional.
- Docker users who mount `/home/gurl/...` must switch to `/home/apigent/...`.
- The frozen identifiers will keep the old name visible in a few places, by design.
