---
title: "Security"
description: "How gurl encrypts secrets at rest and keeps them out of output, records and AI context windows."
weight: 5
---

gurl is built on one principle: credentials stay on your machine and out of anything gurl prints or stores.

| Page | Covers |
|------|--------|
| [Credential storage]({{< relref "/security/credential-storage" >}}) | AES-256-GCM encryption of environment and collection secrets, machine and collection keys, sharing secrets with a passphrase |
| [Secret masking]({{< relref "/security/header-redaction" >}}) | What counts as a secret, where it is masked, `--reveal`, agent mode, execution records |

## Core properties

**Secrets are encrypted at rest.** Environment and collection variables marked secret are encrypted with AES-256-GCM. The machine key is generated with `crypto/rand` on first use and stored with owner-only permissions; encrypted values are useless on another machine.

**Secrets are entered without echo.** `gurl env set prod API_KEY=- --secret API_KEY` reads the value from stdin, prompting without echo on a terminal.

**Auth is applied inside gurl.** Saved auth configs are filled from placeholders and applied to the outgoing request; the calling process never builds the credential string.

**Output is masked.** `show`, `list --json`, `codegen`, `paste`, `env show`, dry runs and `run --json` replace secrets with `*****`, including their URL- and JSON-escaped forms. [Agent mode]({{< relref "/ai/agent-mode" >}}) also masks response bodies, headers, errors and `gurl exec` child output.

**Execution records never store secrets.** Every run is recorded for `gurl diag`, with secrets masked before saving.

| Layer | What it handles | Where secrets live |
|-------|-----------------|--------------------|
| Environment and collection secrets | Encrypted storage and retrieval | AES-256-GCM at rest |
| Auth handlers | Credential injection at send time | In memory only |
| Output and execution records | Masked display and auditing | Never: replaced with `*****` |

## Code generation safety

All code generators (curl, Python, JavaScript, Go) escape values for the target language, preventing shell or code injection. `codegen -l curl` quotes each header as one argument (`-H 'Content-Type: application/json'`), masks literal secrets unless `--reveal`, and always replaces `Authorization` with a placeholder.

## Path protection

Export and import commands validate file paths to prevent directory traversal; paths containing `..` are rejected. Temporary files used during self-update are created with `os.CreateTemp`.

## Scripting sandbox

The JavaScript runtime restricts secret variable access by default: scripts cannot read secrets unless explicitly granted access. URL changes made by scripts are limited to `http` and `https`.
