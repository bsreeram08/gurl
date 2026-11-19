package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/bsreeram08/valetkey/internal/brand"
	"github.com/bsreeram08/valetkey/internal/env"
	"github.com/bsreeram08/valetkey/internal/redact"
	"github.com/bsreeram08/valetkey/internal/runner"
	"github.com/bsreeram08/valetkey/internal/storage"
	"github.com/bsreeram08/valetkey/pkg/types"
	"github.com/urfave/cli/v3"
)

// The mcp command serves saved requests as Model Context Protocol tools
// over stdio. Each saved request is one tool. A call runs the request
// through `valetkey --agent run` in a child process, so it gets exactly the
// agent-mode guarantees of the CLI: one resolution path, masked JSON output,
// an execution record, and no way to reveal secrets.
//
// The environment is fixed when the server starts. A tool's arguments are
// the placeholders that the environment and collection leave undefined;
// an agent cannot override a value they define (such as BASE_URL), so it
// cannot redirect a request, and the secrets attached to it, elsewhere.

// mcpProtocolVersions are the MCP revisions this server speaks, newest first.
var mcpProtocolVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// MCPCommand serves saved requests as MCP tools on stdin/stdout.
func MCPCommand(db storage.DB, envStorage env.Store) *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "Serve saved requests as MCP tools over stdio; agents call them without seeing secrets",
		Description: `Starts a Model Context Protocol server on stdin/stdout. Every saved request
(or every request in --collection) becomes a tool. Calls run the request in
agent mode against the environment fixed at start (--env, else the active
one) and return the masked JSON result. Tool arguments can only fill
placeholders the environment and collection leave undefined.

Example client config: {"command": "valetkey", "args": ["mcp", "--env", "dev"]}`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "env", Aliases: []string{"e"}, Usage: "Environment every call runs against (default: the active one)"},
			&cli.StringFlag{Name: "collection", Aliases: []string{"c"}, Usage: "Expose only the requests in this collection"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			envName := c.String("env")
			if envName != "" {
				if e, err := envStorage.GetEnvByName(envName); err != nil || e == nil {
					return cli.Exit(fmt.Sprintf("environment '%s' not found", envName), 2)
				}
			} else if active, err := envStorage.GetActiveEnv(); err == nil {
				envName = active
			}
			self, err := os.Executable()
			if err != nil {
				return fmt.Errorf("cannot locate the %s binary to run requests: %w", brand.Name, err)
			}
			var global []string
			if dir := c.String("project-dir"); dir != "" {
				global = append(global, "--project-dir", dir)
			}
			srv := &mcpServer{
				db:         db,
				envStorage: envStorage,
				envName:    envName,
				collection: c.String("collection"),
				version:    c.Root().Version,
				runRequest: childRunner(self, global),
			}
			return srv.serve(ctx, os.Stdin, os.Stdout)
		},
	}
}

// childRunner runs `<self> --agent [global...] run <args...>` and returns its
// stdout: one JSON object in agent mode. Stderr is dropped; in agent mode
// every error is also reported on stdout.
func childRunner(self string, global []string) func(ctx context.Context, args []string) ([]byte, error) {
	return func(ctx context.Context, args []string) ([]byte, error) {
		argv := append(append([]string{"--agent"}, global...), append([]string{"run"}, args...)...)
		cmd := exec.CommandContext(ctx, self, argv...)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stdin = nil
		err := cmd.Run()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// A failed request still prints its JSON result.
			err = nil
		}
		return stdout.Bytes(), err
	}
}

type mcpServer struct {
	db         storage.DB
	envStorage env.Store
	envName    string
	collection string
	version    string
	// runRequest runs the CLI run command with args; replaced in tests.
	runRequest func(ctx context.Context, args []string) ([]byte, error)
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	rpcParseError     = -32700
	rpcInvalidRequest = -32600
	rpcMethodNotFound = -32601
	rpcInvalidParams  = -32602
)

// serve handles newline-delimited JSON-RPC messages until in is closed.
// Requests are handled one at a time, in order.
func (s *mcpServer) serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if resp := s.handle(ctx, line); resp != nil {
			if err := writeJSONLine(out, resp); err != nil {
				return err
			}
		}
		if ctx.Err() != nil {
			return nil
		}
	}
	return scanner.Err()
}

// handle returns the response to one message, or nil for a notification.
func (s *mcpServer) handle(ctx context.Context, line []byte) *rpcResponse {
	var msg rpcMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return &rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: rpcParseError, Message: "parse error"}}
	}
	if len(msg.ID) == 0 || string(msg.ID) == "null" {
		return nil // notification (e.g. notifications/initialized): no reply
	}
	resp := &rpcResponse{JSONRPC: "2.0", ID: msg.ID}
	if msg.JSONRPC != "2.0" || msg.Method == "" {
		resp.Error = &rpcError{Code: rpcInvalidRequest, Message: "invalid request"}
		return resp
	}
	var err *rpcError
	switch msg.Method {
	case "initialize":
		resp.Result = s.initialize(msg.Params)
	case "ping":
		resp.Result = struct{}{}
	case "tools/list":
		resp.Result, err = s.listTools(ctx)
	case "tools/call":
		resp.Result, err = s.callTool(ctx, msg.Params)
	default:
		err = &rpcError{Code: rpcMethodNotFound, Message: "method not found: " + msg.Method}
	}
	resp.Error = err
	if err != nil {
		resp.Result = nil
	}
	return resp
}

func (s *mcpServer) initialize(params json.RawMessage) any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &p)
	version := mcpProtocolVersions[0]
	for _, v := range mcpProtocolVersions {
		if v == p.ProtocolVersion {
			version = v
		}
	}
	instructions := "Each tool runs one saved HTTP request and returns the masked JSON result. " +
		"Secrets stay inside " + brand.Name + "; tool arguments only fill placeholders the environment leaves undefined."
	if s.envName != "" {
		instructions += " Environment: " + s.envName + "."
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": brand.Name, "version": s.version},
		"instructions":    instructions,
	}
}

type mcpTool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// toolEntry is a saved request exposed as a tool.
type toolEntry struct {
	tool mcpTool
	req  *types.SavedRequest
}

// tools lists the exposed requests, sorted by name, with unique tool names.
func (s *mcpServer) tools(ctx context.Context) ([]toolEntry, error) {
	reqs, err := s.db.ListRequests(&storage.ListOptions{Collection: s.collection})
	if err != nil {
		return nil, fmt.Errorf("listing saved requests: %w", err)
	}
	sort.Slice(reqs, func(i, j int) bool { return reqs[i].Name < reqs[j].Name })
	used := map[string]bool{}
	entries := make([]toolEntry, 0, len(reqs))
	for _, req := range reqs {
		fillable, err := s.fillable(ctx, req)
		if err != nil {
			return nil, err
		}
		props := map[string]any{}
		for _, name := range fillable {
			props[name] = map[string]any{"type": "string", "description": "Value for the {{" + name + "}} placeholder"}
		}
		schema := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(fillable) > 0 {
			schema["required"] = fillable
		}
		shown := redact.SavedRequest(req)
		desc := fmt.Sprintf("%s %s (saved request %q", shown.Method, shown.URL, req.Name)
		if req.Collection != "" {
			desc += ", collection " + req.Collection
		}
		desc += "). Returns the masked JSON result: status, headers, body, assertions, extracted variables."
		method := strings.ToUpper(req.Method)
		entries = append(entries, toolEntry{req: req, tool: mcpTool{
			Name:        uniqueToolName(req.Name, used),
			Title:       req.Name,
			Description: desc,
			InputSchema: schema,
			Annotations: map[string]any{
				"readOnlyHint":  method == "GET" || method == "HEAD" || method == "OPTIONS",
				"openWorldHint": true,
			},
		}})
	}
	return entries, nil
}

// fillable lists the placeholders of req that the pinned environment and
// the request's collection leave undefined: the only values a tool call
// may supply. It plans the request without sending it.
func (s *mcpServer) fillable(ctx context.Context, req *types.SavedRequest) ([]string, error) {
	var e *env.Environment
	if s.envName != "" {
		found, err := s.envStorage.GetEnvByName(s.envName)
		if err != nil || found == nil {
			return nil, fmt.Errorf("environment '%s' not found", s.envName)
		}
		e = found
	}
	scopes := runner.ScopesFor(s.db, e, req.Collection, map[string]string{})
	plan := runner.NewRunner(s.db, s.envStorage).PlanSavedRequest(ctx, req, scopes)
	return plan.Result.Problems.Missing(), nil
}

func (s *mcpServer) listTools(ctx context.Context) (any, *rpcError) {
	entries, err := s.tools(ctx)
	if err != nil {
		return nil, &rpcError{Code: rpcInvalidParams, Message: err.Error()}
	}
	tools := make([]mcpTool, len(entries))
	for i, e := range entries {
		tools[i] = e.tool
	}
	return map[string]any{"tools": tools}, nil
}

func (s *mcpServer) callTool(ctx context.Context, params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Name == "" {
		return nil, &rpcError{Code: rpcInvalidParams, Message: "tools/call needs a tool name"}
	}
	entries, err := s.tools(ctx)
	if err != nil {
		return nil, &rpcError{Code: rpcInvalidParams, Message: err.Error()}
	}
	var entry *toolEntry
	for i := range entries {
		if entries[i].tool.Name == p.Name {
			entry = &entries[i]
		}
	}
	if entry == nil {
		return nil, &rpcError{Code: rpcInvalidParams, Message: "unknown tool: " + p.Name}
	}

	allowed := map[string]bool{}
	if props, ok := entry.tool.InputSchema["properties"].(map[string]any); ok {
		for name := range props {
			allowed[name] = true
		}
	}
	args := []string{entry.req.Name}
	if s.envName != "" {
		args = append(args, "--env", s.envName)
	}
	names := make([]string, 0, len(p.Arguments))
	for name := range p.Arguments {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !allowed[name] {
			return toolError(fmt.Sprintf("argument %q is not allowed: only placeholders the environment leaves undefined can be set (allowed: %s)", name, joinOrNone(entry.tool.InputSchema))), nil
		}
		value, ok := p.Arguments[name].(string)
		if !ok {
			return toolError(fmt.Sprintf("argument %q must be a string", name)), nil
		}
		args = append(args, "--var", name+"="+value)
	}

	out, err := s.runRequest(ctx, args)
	if err != nil {
		return toolError("running the request failed: " + err.Error()), nil
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return toolError("the request produced no result"), nil
	}
	var status struct {
		OK bool `json:"ok"`
	}
	_ = json.Unmarshal([]byte(text), &status)
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": !status.OK,
	}, nil
}

func toolError(msg string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": msg}},
		"isError": true,
	}
}

func joinOrNone(schema map[string]any) string {
	req, _ := schema["required"].([]string)
	if len(req) == 0 {
		return "none"
	}
	return strings.Join(req, ", ")
}

var toolNameInvalid = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// uniqueToolName turns a request name into a tool name that MCP clients
// accept (letters, digits, _ and -, at most 64 characters), unique in used.
func uniqueToolName(name string, used map[string]bool) string {
	base := strings.Trim(toolNameInvalid.ReplaceAllString(name, "_"), "_")
	if base == "" {
		base = "request"
	}
	if len(base) > 60 {
		base = base[:60]
	}
	candidate := base
	for i := 2; used[candidate]; i++ {
		candidate = fmt.Sprintf("%s_%d", base, i)
	}
	used[candidate] = true
	return candidate
}
