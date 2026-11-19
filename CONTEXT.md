# apigent

A terminal library of named HTTP requests that are run against environments and collections, with variables filled in at send time. People and AI agents run them by name; agents never see the secrets. Formerly gurl (see `docs/adr/0001-rename-to-apigent.md`).

## Language

### Requests and variables

**Saved request**:
A named, stored HTTP request whose URL, headers, body and auth may contain placeholders.
_Avoid_: request template, entry

**Placeholder**:
A `{{name}}` marker in a saved request, optionally typed (`{{name:num}}`) or dynamic (`{{$uuid}}`), replaced by a value at send time.
_Avoid_: template variable, token

**Scope**:
One source of variable values: environment, collection, data row, or CLI. Scopes override each other in that order, lowest first.
_Avoid_: layer, context

**Variable origin**:
The scope a variable's final value came from; it decides where `--persist` writes the value back.

**Request resolution**:
Turning a saved request plus its scopes into the exact request that will be sent, or into the list of problems that stop it from being sent.
_Avoid_: substitution, templating, rendering

**Resolution problem**:
A reason a saved request cannot be resolved: a missing variable, a leftover placeholder inside a value, a badly typed value, or an unknown dynamic function.

**Alias**:
An environment value that refers to another key of the same environment (`API_KEY={{SA_KEY}}`).

### Running

**Dry run**:
Resolving a saved request and showing the result without sending it.

**Persist**:
Writing variables extracted from a response, or set by a script, back to their variable origin.

### Agents

**Agent mode**:
The mode an AI agent runs in: JSON output, no prompts, `--reveal` refused, and secrets masked in response bodies, headers and errors too.

**Tool**:
A saved request served to an MCP client by `apigent mcp`. Its arguments are its fillable placeholders.
_Avoid_: endpoint, action

**Fillable placeholder**:
A placeholder that neither the pinned environment nor the request's collection defines: the only values a tool call may supply.

**Legacy location**:
A pre-rename path or name (`gurl`, `GURL_*`, `~/.local/share/gurl`, `.gurl/`) that is still read when the apigent one is absent. Resolved only in `internal/brand`.
