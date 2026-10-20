# Security policy

## Reporting a vulnerability

Report it privately through GitHub: open the repository's **Security** tab and choose **Report a vulnerability** (<https://github.com/bsreeram08/gurl/security/advisories/new>).

Don't open a public issue, pull request or discussion about the vulnerability. If private reporting isn't available to you, open an issue that only asks for a private contact, with no details.

Include:

- the gurl version (`gurl --version`) and how you installed it (Homebrew, script, binary, Docker, source);
- the steps to reproduce, with fake credentials only;
- what you expected and what happened, for example which output or file contained a secret.

You should get an acknowledgement within a few days. Fixes are released as a new patch version and listed in [CHANGELOG.md](CHANGELOG.md).

## Supported versions

Only the latest release gets security fixes. Upgrade with `gurl update`, `brew upgrade gurl` or `ghcr.io/bsreeram08/gurl:latest`.

## Scope

In scope:

- a secret value showing up where gurl promises to mask it: terminal output, `--json` output, agent-mode output, execution records (`history`, `diag`, `diff`), `show`, `list`, `codegen`, `paste`, exports or error messages;
- secrets stored unencrypted, or collection or project files that leak key material into git;
- bypasses of agent mode (`--agent`, `GURL_AGENT=1`, `agentgurl`), such as getting `--reveal` or the clipboard to work;
- escapes from the JavaScript script sandbox, path traversal in import, export or output files, and flaws in `gurl update` checksum verification.

Out of scope:

- a user deliberately revealing their own secrets (`--reveal`, `env show --reveal`, `paste --reveal`);
- response bodies stored as received, outside agent mode (this is documented behavior);
- vulnerabilities in the APIs you call with gurl.

## Secret-handling guarantees

- Environment and collection secret values are encrypted at rest with AES-256-GCM. Local key files such as `.gurl/**/collection.key` are gitignored, and shared exports are encrypted with a passphrase.
- Secret variables and credential-looking values (Authorization, Cookie, `*key*`, `*token*`, `*secret*`, ...) are masked as `*****` wherever gurl shows or exports a request, and in the request part of execution records before they are saved. Only `--reveal` shows them. Outside agent mode, response bodies are shown and stored as the server sent them.
- Agent mode refuses `--reveal`, `paste` and `codegen --clipboard`, and also masks secret values in response bodies, response headers and errors.
- `gurl exec` passes variables to a child process without printing them.
- Resolution problems name the variable, never its value.
