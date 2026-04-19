package commands

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

type shellEnvStore interface {
	GetEnvByName(name string) (*env.Environment, error)
	GetActiveEnv() (string, error)
	ListEnvs() ([]*env.Environment, error)
	SetActiveEnv(name string) error
}

type shellSession struct {
	db            storage.DB
	envStorage    shellEnvStore
	reader        *bufio.Reader
	out           io.Writer
	inFile        *os.File
	outFile       *os.File
	terminal      *term.Terminal
	terminalState *term.State
	history       *shellHistory
	current       *types.SavedRequest
	persisted     bool
	dirty         bool
	activeEnvName string
	outputFormat  string
}

type shellHistory struct {
	entries []string
	max     int
	enabled bool
}

type shellReadWriter struct {
	in  io.Reader
	out io.Writer
}

func (rw *shellReadWriter) Read(p []byte) (int, error)  { return rw.in.Read(p) }
func (rw *shellReadWriter) Write(p []byte) (int, error) { return rw.out.Write(p) }

func (h *shellHistory) Add(entry string) {
	if h == nil || !h.enabled {
		return
	}

	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}
	if len(h.entries) > 0 && h.entries[0] == entry {
		return
	}

	h.entries = append([]string{entry}, h.entries...)
	if h.max > 0 && len(h.entries) > h.max {
		h.entries = h.entries[:h.max]
	}
}

func (h *shellHistory) Len() int {
	if h == nil {
		return 0
	}
	return len(h.entries)
}

func (h *shellHistory) At(idx int) string {
	return h.entries[idx]
}

type shellCommandHelp struct {
	Name     string
	Usage    string
	Summary  string
	Examples []string
}

var shellHelpTopics = []shellCommandHelp{
	{Name: "help", Usage: "help [command]", Summary: "Show shell help or detailed help for one command", Examples: []string{"help", "help send"}},
	{Name: "list", Usage: "list [pattern]", Summary: "List saved requests, optionally filtered by text", Examples: []string{"list", "list billing"}},
	{Name: "open", Usage: "open [name]", Summary: "Open an existing request; omitting the name shows a numbered selector", Examples: []string{"open beta fiserv", "open"}},
	{Name: "new", Usage: "new [name]", Summary: "Start a new draft request", Examples: []string{"new health check"}},
	{Name: "show", Usage: "show", Summary: "Display the current draft, headers, auth, and body", Examples: []string{"show"}},
	{Name: "set", Usage: "set <field> <value>", Summary: "Update request fields like method, url, body, collection, folder, or timeout", Examples: []string{"set method POST", "set url https://api.example.com/users", "set timeout 5s"}},
	{Name: "header", Usage: "header add <key> <value> | header remove <key> | header clear", Summary: "Manage request headers", Examples: []string{"header add Authorization Bearer-token", "header remove Authorization"}},
	{Name: "query", Usage: "query add <key> <value> | query remove <key> | query clear", Summary: "Manage URL query parameters", Examples: []string{"query add page 1", "query remove page"}},
	{Name: "auth", Usage: "auth none | auth basic <user> <pass> | auth bearer <token> | auth apikey [header] <value>", Summary: "Configure request authentication", Examples: []string{"auth bearer abc123", "auth apikey X-API-Key secret"}},
	{Name: "env", Usage: "env list | env use <name> | env show", Summary: "Inspect or switch environments for variable substitution", Examples: []string{"env list", "env use staging"}},
	{Name: "save", Usage: "save [name]", Summary: "Persist the current draft", Examples: []string{"save", "save beta fiserv"}},
	{Name: "send", Usage: "send [key=value ...]", Summary: "Execute the current draft; missing template vars are prompted interactively", Examples: []string{"send", "send amount=10 currency=USD"}},
	{Name: "close", Usage: "close", Summary: "Close the current draft", Examples: []string{"close"}},
	{Name: "quit", Usage: "quit", Summary: "Exit the shell", Examples: []string{"quit"}},
}

func ShellCommand(db storage.DB, envStorage *env.EnvStorage) *cli.Command {
	return &cli.Command{
		Name:    "shell",
		Aliases: []string{"sh", "repl"},
		Usage:   "Launch the typed interactive shell",
		Description: `gurl shell launches a line-oriented REPL for browsing, editing, and
executing requests without fullscreen keybindings or terminal key conflicts.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Response output format (auto|json|table)",
				Value:   "auto",
			},
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Active environment to use for this shell session",
			},
			&cli.StringFlag{
				Name:    "request",
				Aliases: []string{"r"},
				Usage:   "Open a request immediately by name",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			session := newShellSession(db, envStorage, os.Stdin, os.Stdout, c.String("format"))

			if envName := strings.TrimSpace(c.String("env")); envName != "" {
				if err := session.useEnv(envName); err != nil {
					return err
				}
			}

			if requestName := strings.TrimSpace(c.String("request")); requestName != "" {
				if err := session.openRequest(requestName); err != nil {
					return err
				}
			}

			return session.Run(ctx)
		},
	}
}

func newShellSession(db storage.DB, envStorage shellEnvStore, in io.Reader, out io.Writer, outputFormat string) *shellSession {
	session := &shellSession{
		db:           db,
		envStorage:   envStorage,
		reader:       bufio.NewReader(in),
		out:          out,
		outputFormat: strings.TrimSpace(outputFormat),
		history:      &shellHistory{max: 100},
	}

	if file, ok := in.(*os.File); ok {
		session.inFile = file
	}
	if file, ok := out.(*os.File); ok {
		session.outFile = file
	}

	if session.outputFormat == "" {
		session.outputFormat = "auto"
	}

	if envStorage != nil {
		if active, err := envStorage.GetActiveEnv(); err == nil {
			session.activeEnvName = active
		}
	}

	session.initTerminal()

	return session
}

func (s *shellSession) Run(ctx context.Context) error {
	defer s.closeTerminal()

	fmt.Fprintln(s.out, "gurl shell")
	fmt.Fprintln(s.out, "Type `help` for commands. This shell only uses typed commands.")

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		line, err := s.readLine(s.prompt(), true)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		line = strings.TrimSpace(line)
		if line != "" {
			shouldExit, execErr := s.executeLine(line)
			if execErr != nil {
				fmt.Fprintf(s.out, "error: %v\n", execErr)
			}
			if shouldExit {
				return nil
			}
		}

		if errors.Is(err, io.EOF) {
			fmt.Fprintln(s.out)
			return nil
		}
	}
}

func (s *shellSession) initTerminal() {
	if s.inFile == nil || s.outFile == nil {
		return
	}
	if !term.IsTerminal(int(s.inFile.Fd())) || !term.IsTerminal(int(s.outFile.Fd())) {
		return
	}

	state, err := term.MakeRaw(int(s.inFile.Fd()))
	if err != nil {
		return
	}

	s.terminalState = state
	s.terminal = term.NewTerminal(&shellReadWriter{in: s.inFile, out: s.outFile}, "")
	s.terminal.History = s.history
	s.out = s.terminal
}

func (s *shellSession) closeTerminal() {
	if s.terminalState == nil || s.inFile == nil {
		return
	}
	_ = term.Restore(int(s.inFile.Fd()), s.terminalState)
	s.terminalState = nil
}

func (s *shellSession) readLine(prompt string, recordHistory bool) (string, error) {
	if s.terminal != nil {
		s.history.enabled = recordHistory
		s.terminal.SetPrompt(prompt)
		return s.terminal.ReadLine()
	}

	if prompt != "" {
		fmt.Fprint(s.out, prompt)
	}

	line, err := s.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	return strings.TrimRight(line, "\r\n"), err
}

func (s *shellSession) executeLine(line string) (bool, error) {
	command, rest := splitCommand(line)
	if command == "" {
		return false, nil
	}

	force := strings.HasSuffix(command, "!")
	command = strings.TrimSuffix(command, "!")

	switch strings.ToLower(command) {
	case "help", "?":
		return false, s.printHelp(rest)
	case "exit", "quit":
		if err := s.ensureDiscardAllowed(force, "quit!"); err != nil {
			return false, err
		}
		return true, nil
	case "list", "ls":
		return false, s.listRequests(rest)
	case "open":
		if err := s.ensureDiscardAllowed(force, "open! <name>"); err != nil {
			return false, err
		}
		return false, s.handleOpen(rest)
	case "new":
		if err := s.ensureDiscardAllowed(force, "new! <name>"); err != nil {
			return false, err
		}
		return false, s.handleNew(rest)
	case "close":
		if err := s.ensureDiscardAllowed(force, "close!"); err != nil {
			return false, err
		}
		s.current = nil
		s.persisted = false
		s.dirty = false
		fmt.Fprintln(s.out, "Closed current draft.")
		return false, nil
	case "show", "status":
		return false, s.showCurrent()
	case "set":
		return false, s.handleSet(rest)
	case "header":
		return false, s.handleHeader(rest)
	case "query":
		return false, s.handleQuery(rest)
	case "auth":
		return false, s.handleAuth(rest)
	case "env":
		return false, s.handleEnv(rest)
	case "save":
		return false, s.handleSave(rest)
	case "send", "run":
		return false, s.handleSend(rest)
	default:
		return false, errors.New(s.unknownCommandMessage(command))
	}
}

func (s *shellSession) prompt() string {
	if s.current == nil {
		return "gurl> "
	}

	label := strings.TrimSpace(s.current.Name)
	if label == "" {
		label = "draft"
	}
	if s.dirty {
		label += "*"
	}

	return fmt.Sprintf("gurl[%s]> ", label)
}

func (s *shellSession) printHelp(topic string) error {
	topic = canonicalShellCommand(topic)
	if topic != "" {
		for _, item := range shellHelpTopics {
			if item.Name != topic {
				continue
			}

			fmt.Fprintf(s.out, "%s\n  %s\n", item.Name, item.Summary)
			fmt.Fprintf(s.out, "usage: %s\n", item.Usage)
			if len(item.Examples) > 0 {
				fmt.Fprintln(s.out, "examples:")
				for _, example := range item.Examples {
					fmt.Fprintf(s.out, "  %s\n", example)
				}
			}
			return nil
		}

		return errors.New(s.unknownHelpTopicMessage(topic))
	}

	lines := []string{
		"",
		"Core commands:",
		"  list [pattern]                List saved requests",
		"  open [name]                   Open a saved request",
		"  new [name]                    Start a new draft",
		"  show                          Show the current draft",
		"  save [name]                   Save the current draft",
		"  send [key=value ...]          Send the current draft",
		"",
		"Editing commands:",
		"  set method <METHOD>",
		"  set url <URL>",
		"  set body <TEXT>",
		"  set name <NAME>",
		"  set collection <NAME>",
		"  set folder <PATH>",
		"  set timeout <DURATION>",
		"  header add <KEY> <VALUE>",
		"  header remove <KEY>",
		"  query add <KEY> <VALUE>",
		"  query remove <KEY>",
		"  auth none",
		"  auth basic <USER> <PASS>",
		"  auth bearer <TOKEN>",
		"  auth apikey [HEADER] <VALUE>",
		"",
		"Environment commands:",
		"  env list                      List environments",
		"  env use <NAME>                Activate an environment",
		"  env show                      Show the active environment",
		"",
		"Session commands:",
		"  close                         Close the current draft",
		"  quit                          Exit the shell",
		"",
		"Repeat open/new/close/quit with ! to discard unsaved changes.",
		"Type `help <command>` for detailed usage and examples.",
		"",
	}

	for _, line := range lines {
		fmt.Fprintln(s.out, line)
	}

	return nil
}

func (s *shellSession) listRequests(pattern string) error {
	requests, err := s.db.ListRequests(nil)
	if err != nil {
		return fmt.Errorf("failed to list requests: %w", err)
	}

	sort.Slice(requests, func(i, j int) bool {
		return strings.ToLower(requests[i].Name) < strings.ToLower(requests[j].Name)
	})

	pattern = strings.ToLower(strings.TrimSpace(pattern))
	matched := 0
	for _, req := range requests {
		name := req.Name
		if name == "" {
			name = req.URL
		}
		if pattern != "" && !strings.Contains(strings.ToLower(name), pattern) && !strings.Contains(strings.ToLower(req.URL), pattern) {
			continue
		}

		matched++
		line := fmt.Sprintf("%2d. %-7s %s", matched, strings.ToUpper(req.Method), name)
		if req.Collection != "" {
			line += fmt.Sprintf("  [%s]", req.Collection)
		}
		fmt.Fprintln(s.out, line)
	}

	if matched == 0 {
		fmt.Fprintln(s.out, "No matching requests.")
	}

	return nil
}

func (s *shellSession) handleOpen(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		requests, err := s.db.ListRequests(nil)
		if err != nil {
			return fmt.Errorf("failed to list requests: %w", err)
		}
		chosen, err := s.selectRequest(requests, "Open request by number or exact name: ")
		if err != nil {
			return err
		}
		if chosen == "" {
			return nil
		}
		name = chosen
	}

	return s.openRequest(name)
}

func (s *shellSession) openRequest(name string) error {
	req, err := s.db.GetRequestByName(name)
	if err != nil {
		return fmt.Errorf("request not found: %s", name)
	}

	s.current = cloneRequest(req)
	s.persisted = true
	s.dirty = false
	fmt.Fprintf(s.out, "Opened %q.\n", s.current.Name)
	return nil
}

func (s *shellSession) handleNew(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "untitled"
	}

	s.current = types.NewSavedRequest(name, "", "GET")
	s.persisted = false
	s.dirty = true
	fmt.Fprintf(s.out, "Started new draft %q.\n", s.current.Name)
	return nil
}

func (s *shellSession) showCurrent() error {
	if s.current == nil {
		return fmt.Errorf("no request is open")
	}

	req := s.current
	fmt.Fprintf(s.out, "Name:       %s\n", req.Name)
	fmt.Fprintf(s.out, "Method:     %s\n", blankOr(req.Method, "GET"))
	fmt.Fprintf(s.out, "URL:        %s\n", req.URL)
	if req.Collection != "" {
		fmt.Fprintf(s.out, "Collection: %s\n", req.Collection)
	}
	if req.Folder != "" {
		fmt.Fprintf(s.out, "Folder:     %s\n", req.Folder)
	}
	if req.Timeout != "" {
		fmt.Fprintf(s.out, "Timeout:    %s\n", req.Timeout)
	}
	if s.activeEnvName != "" {
		fmt.Fprintf(s.out, "Env:        %s\n", s.activeEnvName)
	}
	fmt.Fprintf(s.out, "Saved:      %t\n", s.persisted)
	fmt.Fprintf(s.out, "Dirty:      %t\n", s.dirty)

	if len(req.Headers) > 0 {
		fmt.Fprintln(s.out, "\nHeaders:")
		for _, h := range req.Headers {
			fmt.Fprintf(s.out, "  %s: %s\n", h.Key, h.Value)
		}
	}

	if req.Body != "" {
		fmt.Fprintln(s.out, "\nBody:")
		fmt.Fprintln(s.out, req.Body)
	}

	if req.AuthConfig != nil {
		fmt.Fprintln(s.out, "\nAuth:")
		switch req.AuthConfig.Type {
		case "basic":
			fmt.Fprintf(s.out, "  basic username=%s password=%s\n", req.AuthConfig.Params["username"], maskValue(req.AuthConfig.Params["password"]))
		case "bearer":
			fmt.Fprintf(s.out, "  bearer token=%s\n", maskValue(req.AuthConfig.Params["token"]))
		case "apikey":
			header := req.AuthConfig.Params["header"]
			if header == "" {
				header = "X-API-Key"
			}
			fmt.Fprintf(s.out, "  apikey header=%s value=%s\n", header, maskValue(req.AuthConfig.Params["value"]))
		default:
			fmt.Fprintf(s.out, "  %s\n", req.AuthConfig.Type)
		}
	}

	return nil
}

func (s *shellSession) handleSet(rest string) error {
	req, err := s.requireCurrent()
	if err != nil {
		return err
	}

	field, value := splitCommand(rest)
	switch strings.ToLower(field) {
	case "method":
		method := strings.ToUpper(strings.TrimSpace(value))
		if !validMethods[method] {
			return fmt.Errorf("invalid HTTP method: %s", method)
		}
		req.Method = method
	case "url":
		req.URL = strings.TrimSpace(value)
	case "body":
		req.Body = value
	case "name":
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("name cannot be empty")
		}
		req.Name = strings.TrimSpace(value)
	case "collection":
		req.Collection = strings.TrimSpace(value)
	case "folder":
		req.Folder = strings.TrimSpace(value)
	case "timeout":
		value = strings.TrimSpace(value)
		if value != "" {
			if _, err := time.ParseDuration(value); err != nil {
				return fmt.Errorf("invalid timeout format %q: %w", value, err)
			}
		}
		req.Timeout = value
	default:
		return fmt.Errorf("unknown set field %q", field)
	}

	s.markDirty()
	return nil
}

func (s *shellSession) handleHeader(rest string) error {
	req, err := s.requireCurrent()
	if err != nil {
		return err
	}

	action, value := splitCommand(rest)
	switch strings.ToLower(action) {
	case "add":
		key, headerValue := splitCommand(value)
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(headerValue) == "" {
			return fmt.Errorf("usage: header add <KEY> <VALUE>")
		}
		updated := false
		for i, h := range req.Headers {
			if strings.EqualFold(h.Key, key) {
				req.Headers[i].Value = headerValue
				updated = true
				break
			}
		}
		if !updated {
			req.Headers = append(req.Headers, types.Header{Key: key, Value: headerValue})
		}
	case "remove":
		key := strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("usage: header remove <KEY>")
		}
		filtered := req.Headers[:0]
		for _, h := range req.Headers {
			if !strings.EqualFold(h.Key, key) {
				filtered = append(filtered, h)
			}
		}
		req.Headers = filtered
	case "clear":
		req.Headers = nil
	default:
		return fmt.Errorf("unknown header action %q", action)
	}

	s.markDirty()
	return nil
}

func (s *shellSession) handleQuery(rest string) error {
	req, err := s.requireCurrent()
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.URL) == "" {
		return fmt.Errorf("set the URL before editing query params")
	}

	action, value := splitCommand(rest)
	parsed, err := url.Parse(req.URL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", req.URL, err)
	}

	queryValues := parsed.Query()
	switch strings.ToLower(action) {
	case "add":
		key, queryValue := splitCommand(value)
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("usage: query add <KEY> <VALUE>")
		}
		queryValues.Set(key, queryValue)
	case "remove":
		key := strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("usage: query remove <KEY>")
		}
		queryValues.Del(key)
	case "clear":
		queryValues = url.Values{}
	default:
		return fmt.Errorf("unknown query action %q", action)
	}

	parsed.RawQuery = queryValues.Encode()
	req.URL = parsed.String()
	s.markDirty()
	return nil
}

func (s *shellSession) handleAuth(rest string) error {
	req, err := s.requireCurrent()
	if err != nil {
		return err
	}

	authType, value := splitCommand(rest)
	switch strings.ToLower(authType) {
	case "none":
		req.AuthConfig = nil
	case "basic":
		username, password := splitCommand(value)
		if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
			return fmt.Errorf("usage: auth basic <USER> <PASS>")
		}
		req.AuthConfig = &types.AuthConfig{
			Type: "basic",
			Params: map[string]string{
				"username": username,
				"password": password,
			},
		}
	case "bearer":
		token := strings.TrimSpace(value)
		if token == "" {
			return fmt.Errorf("usage: auth bearer <TOKEN>")
		}
		req.AuthConfig = &types.AuthConfig{
			Type: "bearer",
			Params: map[string]string{
				"token": token,
			},
		}
	case "apikey":
		first, remainder := splitCommand(value)
		first = strings.TrimSpace(first)
		remainder = strings.TrimSpace(remainder)
		if first == "" {
			return fmt.Errorf("usage: auth apikey [HEADER] <VALUE>")
		}

		header := "X-API-Key"
		apiValue := first
		if remainder != "" {
			header = first
			apiValue = remainder
		}

		req.AuthConfig = &types.AuthConfig{
			Type: "apikey",
			Params: map[string]string{
				"header": header,
				"value":  apiValue,
			},
		}
	default:
		return fmt.Errorf("unknown auth type %q", authType)
	}

	s.markDirty()
	return nil
}

func (s *shellSession) handleEnv(rest string) error {
	if s.envStorage == nil {
		return fmt.Errorf("environment storage is unavailable")
	}

	action, value := splitCommand(rest)
	switch strings.ToLower(action) {
	case "list":
		envs, err := s.envStorage.ListEnvs()
		if err != nil {
			return fmt.Errorf("failed to list environments: %w", err)
		}
		sort.Slice(envs, func(i, j int) bool {
			return strings.ToLower(envs[i].Name) < strings.ToLower(envs[j].Name)
		})
		if len(envs) == 0 {
			fmt.Fprintln(s.out, "No environments found.")
			return nil
		}
		for _, item := range envs {
			prefix := " "
			if item.Name == s.activeEnvName {
				prefix = "*"
			}
			fmt.Fprintf(s.out, "%s %s\n", prefix, item.Name)
		}
		return nil
	case "use":
		return s.useEnv(strings.TrimSpace(value))
	case "show":
		if s.activeEnvName == "" {
			fmt.Fprintln(s.out, "No active environment.")
			return nil
		}
		envObj, err := s.envStorage.GetEnvByName(s.activeEnvName)
		if err != nil {
			return fmt.Errorf("failed to load environment %q: %w", s.activeEnvName, err)
		}
		fmt.Fprintf(s.out, "Active environment: %s (%d vars)\n", envObj.Name, len(envObj.Variables))
		return nil
	default:
		return fmt.Errorf("usage: env list | env use <NAME> | env show")
	}
}

func (s *shellSession) useEnv(name string) error {
	if s.envStorage == nil {
		return fmt.Errorf("environment storage is unavailable")
	}
	if name == "" {
		return fmt.Errorf("usage: env use <NAME>")
	}

	if _, err := s.envStorage.GetEnvByName(name); err != nil {
		return fmt.Errorf("environment not found: %s", name)
	}
	if err := s.envStorage.SetActiveEnv(name); err != nil {
		return fmt.Errorf("failed to activate environment: %w", err)
	}

	s.activeEnvName = name
	fmt.Fprintf(s.out, "Active environment set to %q.\n", name)
	return nil
}

func (s *shellSession) handleSave(name string) error {
	req, err := s.requireCurrent()
	if err != nil {
		return err
	}

	if trimmed := strings.TrimSpace(name); trimmed != "" {
		req.Name = trimmed
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("request name cannot be empty")
	}

	now := time.Now().Unix()
	if req.CreatedAt == 0 {
		req.CreatedAt = now
	}
	req.UpdatedAt = now

	if s.persisted {
		if err := s.db.UpdateRequest(req); err != nil {
			return fmt.Errorf("failed to update request: %w", err)
		}
	} else {
		if err := s.db.SaveRequest(req); err != nil {
			return fmt.Errorf("failed to save request: %w", err)
		}
		s.persisted = true
	}

	s.dirty = false
	fmt.Fprintf(s.out, "Saved %q.\n", req.Name)
	return nil
}

func (s *shellSession) handleSend(rest string) error {
	req, err := s.requireCurrent()
	if err != nil {
		return err
	}

	overrides, err := parseAssignments(strings.Fields(strings.TrimSpace(rest)))
	if err != nil {
		return err
	}

	vars := collectRequestDefaults(req)
	for k, v := range s.collectEnvVars() {
		vars[k] = v
	}
	for k, v := range overrides {
		vars[k] = v
	}

	missing := missingTemplateVars(req, vars)
	if len(missing) > 0 {
		if err := s.promptForMissingVars(vars, missing); err != nil {
			return err
		}
	}

	clientReq, err := buildClientRequestFromSavedRequest(req, vars)
	if err != nil {
		return err
	}

	resp, err := client.Execute(clientReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	if s.persisted && req.ID != "" {
		history := types.NewExecutionHistory(
			req.ID,
			string(resp.Body),
			resp.StatusCode,
			resp.Duration.Milliseconds(),
			resp.Size,
		)
		if err := s.db.SaveHistory(history); err != nil {
			return fmt.Errorf("failed to save history: %w", err)
		}
	}

	format := s.outputFormat
	if format == "" || format == "auto" {
		if strings.TrimSpace(req.OutputFormat) != "" {
			format = req.OutputFormat
		} else {
			format = "auto"
		}
	}

	fmt.Fprintf(
		s.out,
		"%s %s\nStatus: %d %s | Duration: %s | Size: %s\n\n",
		clientReq.Method,
		clientReq.URL,
		resp.StatusCode,
		http.StatusText(resp.StatusCode),
		resp.Duration,
		formatBytes(resp.Size),
	)
	if err := printResponse(s.out, clientReq.Method, clientReq.URL, resp, format); err != nil {
		return err
	}
	fmt.Fprintln(s.out)

	return nil
}

func (s *shellSession) promptForMissingVars(vars map[string]string, missing []string) error {
	for _, name := range missing {
		if strings.TrimSpace(vars[name]) != "" {
			continue
		}

		line, err := s.readLine(name+": ", false)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to read %s: %w", name, err)
		}

		value := strings.TrimSpace(line)
		if value == "" {
			return fmt.Errorf("missing template variables: %s", strings.Join(missingTemplateKeys(vars, missing), ", "))
		}
		vars[name] = value

		if errors.Is(err, io.EOF) {
			break
		}
	}

	if unresolved := missingTemplateKeys(vars, missing); len(unresolved) > 0 {
		return fmt.Errorf("missing template variables: %s", strings.Join(unresolved, ", "))
	}

	return nil
}

func (s *shellSession) requireCurrent() (*types.SavedRequest, error) {
	if s.current == nil {
		return nil, fmt.Errorf("no request is open")
	}
	return s.current, nil
}

func (s *shellSession) markDirty() {
	if s.current == nil {
		return
	}
	s.current.UpdatedAt = time.Now().Unix()
	s.dirty = true
}

func (s *shellSession) ensureDiscardAllowed(force bool, hint string) error {
	if !s.dirty || force {
		return nil
	}
	return fmt.Errorf("unsaved changes would be lost; run `save` or repeat with `%s`", hint)
}

func (s *shellSession) unknownCommandMessage(command string) string {
	message := fmt.Sprintf("unknown command %q", command)
	if suggestions := suggestShellCommands(command); len(suggestions) > 0 {
		message += ". Did you mean: " + strings.Join(suggestions, ", ") + "?"
	}
	return message + " Type `help` for commands."
}

func (s *shellSession) unknownHelpTopicMessage(topic string) string {
	message := fmt.Sprintf("no help for %q", topic)
	if suggestions := suggestShellCommands(topic); len(suggestions) > 0 {
		message += ". Try: help " + suggestions[0]
	}
	return message + "."
}

func (s *shellSession) collectEnvVars() map[string]string {
	vars := make(map[string]string)
	if s.envStorage == nil || s.activeEnvName == "" {
		return vars
	}

	envObj, err := s.envStorage.GetEnvByName(s.activeEnvName)
	if err != nil || envObj == nil {
		return vars
	}

	for k, v := range envObj.Variables {
		vars[k] = v
	}
	return vars
}

func (s *shellSession) selectRequest(requests []*types.SavedRequest, prompt string) (string, error) {
	if len(requests) == 0 {
		fmt.Fprintln(s.out, "No saved requests found.")
		return "", nil
	}

	sorted := make([]*types.SavedRequest, len(requests))
	copy(sorted, requests)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	for i, req := range sorted {
		name := req.Name
		if name == "" {
			name = req.URL
		}
		fmt.Fprintf(s.out, "%2d. %-7s %s\n", i+1, strings.ToUpper(req.Method), name)
	}

	selection, err := s.readLine(prompt, false)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	selection = strings.TrimSpace(selection)
	if selection == "" {
		return "", nil
	}

	if idx, convErr := strconv.Atoi(selection); convErr == nil {
		if idx < 1 || idx > len(sorted) {
			return "", fmt.Errorf("selection out of range: %d", idx)
		}
		return sorted[idx-1].Name, nil
	}

	for _, req := range sorted {
		if req.Name == selection {
			return req.Name, nil
		}
	}

	return "", fmt.Errorf("request not found: %s", selection)
}

func promptSelectRequest(reader *bufio.Reader, out io.Writer, requests []*types.SavedRequest, prompt string) (string, error) {
	if len(requests) == 0 {
		fmt.Fprintln(out, "No saved requests found.")
		return "", nil
	}

	sorted := make([]*types.SavedRequest, len(requests))
	copy(sorted, requests)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	for i, req := range sorted {
		name := req.Name
		if name == "" {
			name = req.URL
		}
		fmt.Fprintf(out, "%2d. %-7s %s\n", i+1, strings.ToUpper(req.Method), name)
	}

	fmt.Fprint(out, prompt)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	selection := strings.TrimSpace(line)
	if selection == "" {
		return "", nil
	}

	if idx, convErr := strconv.Atoi(selection); convErr == nil {
		if idx < 1 || idx > len(sorted) {
			return "", fmt.Errorf("selection out of range: %d", idx)
		}
		return sorted[idx-1].Name, nil
	}

	for _, req := range sorted {
		if req.Name == selection {
			return req.Name, nil
		}
	}

	return "", fmt.Errorf("request not found: %s", selection)
}

func splitCommand(input string) (string, string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", ""
	}

	idx := strings.IndexAny(input, " \t")
	if idx == -1 {
		return input, ""
	}

	return input[:idx], strings.TrimSpace(input[idx+1:])
}

func parseAssignments(values []string) (map[string]string, error) {
	assignments := make(map[string]string)
	for _, value := range values {
		key, remainder, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid variable override %q (expected key=value)", value)
		}
		assignments[strings.TrimSpace(key)] = remainder
	}
	return assignments, nil
}

func canonicalShellCommand(command string) string {
	command = strings.TrimSpace(strings.ToLower(command))
	command = strings.TrimSuffix(command, "!")

	switch command {
	case "?", "help":
		return "help"
	case "ls":
		return "list"
	case "status":
		return "show"
	case "run":
		return "send"
	case "exit":
		return "quit"
	default:
		return command
	}
}

func suggestShellCommands(input string) []string {
	input = canonicalShellCommand(input)
	if input == "" {
		return nil
	}

	type candidate struct {
		name  string
		score int
	}

	suggestions := make([]candidate, 0, len(shellHelpTopics))
	for _, item := range shellHelpTopics {
		score := shellSuggestionScore(input, item.Name)
		if score <= 3 {
			suggestions = append(suggestions, candidate{name: item.Name, score: score})
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].score == suggestions[j].score {
			return suggestions[i].name < suggestions[j].name
		}
		return suggestions[i].score < suggestions[j].score
	})

	limit := min(3, len(suggestions))
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, suggestions[i].name)
	}
	return result
}

func shellSuggestionScore(input, candidate string) int {
	switch {
	case input == candidate:
		return 0
	case strings.HasPrefix(candidate, input), strings.HasPrefix(input, candidate):
		return 1
	case strings.Contains(candidate, input), strings.Contains(input, candidate):
		return 2
	default:
		return levenshteinDistance(input, candidate)
	}
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}

	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)

	for j := range previous {
		previous[j] = j
	}

	for i := 1; i <= len(a); i++ {
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			current[j] = min(
				min(current[j-1]+1, previous[j]+1),
				previous[j-1]+cost,
			)
		}
		copy(previous, current)
	}

	return previous[len(b)]
}

func missingTemplateKeys(vars map[string]string, names []string) []string {
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(vars[name]) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func cloneRequest(req *types.SavedRequest) *types.SavedRequest {
	if req == nil {
		return nil
	}

	copyReq := *req
	copyReq.Headers = append([]types.Header{}, req.Headers...)
	copyReq.Variables = append([]types.Var{}, req.Variables...)
	copyReq.PathParams = append([]types.Var{}, req.PathParams...)
	copyReq.Tags = append([]string{}, req.Tags...)
	copyReq.Assertions = append([]types.Assertion{}, req.Assertions...)
	if req.AuthConfig != nil {
		params := make(map[string]string, len(req.AuthConfig.Params))
		for k, v := range req.AuthConfig.Params {
			params[k] = v
		}
		copyReq.AuthConfig = &types.AuthConfig{
			Type:   req.AuthConfig.Type,
			Params: params,
		}
	}

	return &copyReq
}

func collectRequestDefaults(req *types.SavedRequest) map[string]string {
	vars := make(map[string]string)
	if req == nil {
		return vars
	}

	for _, item := range req.Variables {
		if item.Name == "" {
			continue
		}
		vars[item.Name] = item.Example
	}
	for _, item := range req.PathParams {
		if item.Name == "" {
			continue
		}
		if item.Example != "" {
			vars[item.Name] = item.Example
		}
	}

	return vars
}

func missingTemplateVars(req *types.SavedRequest, provided map[string]string) []string {
	if req == nil {
		return nil
	}

	seen := make(map[string]bool)
	ordered := make([]string, 0)

	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		ordered = append(ordered, name)
	}

	appendNames := func(values []string) {
		for _, name := range values {
			add(name)
		}
	}

	appendNames(template.ExtractVarNames(req.URL))
	appendNames(template.ExtractVarNames(req.Body))
	for _, header := range req.Headers {
		appendNames(template.ExtractVarNames(header.Key))
		appendNames(template.ExtractVarNames(header.Value))
	}

	if req.AuthConfig != nil {
		switch req.AuthConfig.Type {
		case "basic":
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["username"]))
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["password"]))
		case "bearer":
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["token"]))
		case "apikey":
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["header"]))
			appendNames(template.ExtractVarNames(req.AuthConfig.Params["value"]))
		}
	}

	for _, item := range req.PathParams {
		add(item.Name)
	}

	if len(ordered) == 0 {
		for _, item := range template.GetVariablesFromRequest(req) {
			add(item.Name)
		}
	}

	missing := make([]string, 0, len(ordered))
	for _, name := range ordered {
		if value, ok := provided[name]; !ok || strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func buildClientRequestFromSavedRequest(req *types.SavedRequest, provided map[string]string) (client.Request, error) {
	if req == nil {
		return client.Request{}, fmt.Errorf("no request is open")
	}

	execReq := cloneRequest(req)
	for i, item := range execReq.PathParams {
		if value, ok := provided[item.Name]; ok {
			execReq.PathParams[i].Example = value
		}
	}

	if err := template.ResolvePathParamsInRequest(execReq); err != nil {
		return client.Request{}, fmt.Errorf("failed to resolve path params: %w", err)
	}

	resolvedURL, err := template.Substitute(execReq.URL, provided)
	if err != nil {
		return client.Request{}, fmt.Errorf("failed to substitute URL: %w", err)
	}

	resolvedBody := execReq.Body
	if resolvedBody != "" {
		resolvedBody, err = template.Substitute(resolvedBody, provided)
		if err != nil {
			return client.Request{}, fmt.Errorf("failed to substitute body: %w", err)
		}
	}

	headers := make([]client.Header, 0, len(execReq.Headers)+1)
	for _, header := range execReq.Headers {
		key, err := template.Substitute(header.Key, provided)
		if err != nil {
			return client.Request{}, fmt.Errorf("failed to substitute header key: %w", err)
		}
		value, err := template.Substitute(header.Value, provided)
		if err != nil {
			return client.Request{}, fmt.Errorf("failed to substitute header value: %w", err)
		}
		headers = append(headers, client.Header{Key: key, Value: value})
	}

	if execReq.AuthConfig != nil {
		switch execReq.AuthConfig.Type {
		case "basic":
			username, err := template.Substitute(execReq.AuthConfig.Params["username"], provided)
			if err != nil {
				return client.Request{}, fmt.Errorf("failed to substitute basic auth username: %w", err)
			}
			password, err := template.Substitute(execReq.AuthConfig.Params["password"], provided)
			if err != nil {
				return client.Request{}, fmt.Errorf("failed to substitute basic auth password: %w", err)
			}
			headers = append(headers, client.Header{Key: "Authorization", Value: "Basic " + basicAuth(username, password)})
		case "bearer":
			token, err := template.Substitute(execReq.AuthConfig.Params["token"], provided)
			if err != nil {
				return client.Request{}, fmt.Errorf("failed to substitute bearer token: %w", err)
			}
			headers = append(headers, client.Header{Key: "Authorization", Value: "Bearer " + token})
		case "apikey":
			headerKey := execReq.AuthConfig.Params["header"]
			if headerKey == "" {
				headerKey = "X-API-Key"
			}
			headerKey, err = template.Substitute(headerKey, provided)
			if err != nil {
				return client.Request{}, fmt.Errorf("failed to substitute API key header: %w", err)
			}
			headerValue, err := template.Substitute(execReq.AuthConfig.Params["value"], provided)
			if err != nil {
				return client.Request{}, fmt.Errorf("failed to substitute API key value: %w", err)
			}
			headers = append(headers, client.Header{Key: headerKey, Value: headerValue})
		}
	}

	method := strings.ToUpper(strings.TrimSpace(execReq.Method))
	if method == "" {
		method = "GET"
	}

	clientReq := client.Request{
		Method:  method,
		URL:     resolvedURL,
		Headers: headers,
		Body:    resolvedBody,
	}

	if execReq.Timeout != "" {
		timeout, err := time.ParseDuration(execReq.Timeout)
		if err != nil {
			return client.Request{}, fmt.Errorf("invalid timeout %q: %w", execReq.Timeout, err)
		}
		clientReq.Timeout = timeout
	}

	return clientReq, nil
}

func maskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func blankOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
