package commands

// docSection is one static section of `apigent docs`. Everything about
// commands and flags is generated from the command tree; only concepts that
// the tree cannot describe live here. Keep each section short and true to
// the code it names.
type docSection struct {
	Key   string
	Title string
	Body  string
}

var docSections = []docSection{
	{"agent_mode", "Agent mode", `--agent (any position), APIGENT_AGENT=1, or a symlink named agentapigent (ln -s "$(command -v apigent)" ~/.local/bin/agentapigent). GURL_AGENT and agentgurl still work.
= --json wherever supported + --no-input + --reveal/paste/clipboard refused + secrets masked in bodies, headers and errors too. Errors: {"ok":false,"error":"..."} on stdout (and stderr).
--no-input (APIGENT_NO_INPUT=1; implied by --json): never prompt.`},
	{"placeholders", "Placeholders", `{{name}} in URL, headers, body and auth params, filled at send time.
Typed: {{n:num}} {{b:bool}} {{s:raw}}; quoted "{{n:num}}" in a JSON body drops the quotes.
Dynamic: {{$uuid}} {{$timestamp}} {{$isoTimestamp}} {{$randomInt(1,100)}} {{$randomString(8)}} {{$randomEmail}}.
Env alias: env value {{OTHER}} = key OTHER of the same env. No re-expansion.`},
	{"scopes", "Scope precedence", `Lowest first: environment < collection < data row (--data) < CLI (--var) < runtime (extracted or script-set earlier in the run). Collection values may reference env keys.`},
	{"extract", "Extract", `--extract VAR=METHOD:EXPR (repeatable): jsonpath:$.items[0].id | jq:$.id (alias) | header:X-Request-Id | regex:id=(\d+) (first group). Test: apigent extract VAR=jsonpath:$.id --file resp.json`},
	{"persist", "--persist", `Writes extracted/script variables to their origin: env values to the env (--env or active), others to the request's collection, else the env. Locked collection = error. Without it they are dropped.`},
	{"run_json", "run --json", `stdout = one object per execution; NDJSON for --chain steps and --data rows (persisted on the last line). Empty fields omitted:
{"id","short_id","request","collection","environment","row","ok","skipped","skip_reason","method","url","status","status_text","duration_ms","size_bytes","response_headers":{},"body":<JSON or string>,"error","failure_phase","problems":[{"kind","variable","location","message"}],"assertions":[{"name","passed","message"}],"extracted":{},"persisted":{"environment","variables","collections":[{"name","variables"}]}}
ok = no error, no failed assertion, 2xx. Masked like execution records; body as received (masked in agent mode).
kinds: missing leftover_placeholder bad_type unknown_dynamic auth. phases: request_build http extraction assertion status pre_request_script post_response_script run_if next_request.
--dry-run --json: {"dry_run":true,"ok","method","url","headers":[{"key","value"}],"body","extracts","problems"}; nothing sent or recorded.`},
	{"exit_codes", "Exit codes", `0 ok (non-2xx too, unless --fail). 1 error: not found, problems, transport, script, extraction, assertion, non-2xx with --fail. 2 usage: missing argument with --no-input, conflicting flags, --reveal in agent mode. exec: the command's code.`},
	{"exec", "apigent exec", `apigent exec --env NAME [--var K=V] -- <cmd> runs cmd with the env's variables in its environment; apigent never prints them.`},
	{"diag", "apigent diag", `Every run is recorded (failures too, not dry runs). apigent diag <id>: request as sent, variable origins, response, assertions, changes since last success.`},
	{"mcp", "apigent mcp", `MCP server on stdio; one tool per saved request (--collection narrows). Env fixed at start (--env, else active). Tool args = placeholders the env and collection leave undefined; any other argument is refused. A call = run --agent; result = the run --json object, isError = !ok.`},
	{"legacy", "Formerly gurl", `gurl, agentgurl, GURL_* variables, ~/.local/share/gurl, ~/.config/gurl and .gurl/ projects still work when the apigent ones are absent. apigent migrate [--dry-run] moves them.`},
}
