// Package resolve implements request resolution: turning a saved request plus
// its scopes into the exact request that will be sent, or into the list of
// resolution problems that stop it from being sent.
//
// Scopes override each other lowest first: environment < collection < data
// row < CLI < runtime (values extracted from responses or set by scripts
// earlier in the same run). The scope a variable's final value came from is
// its variable origin, which decides where --persist writes it back.
//
// Callers pass the environment scope with aliases already resolved, i.e. the
// result of env.(*Environment).ResolvedVariables(). Collection values that
// reference environment keys ({{name}}) are resolved against that
// environment scope here, before the scopes are merged.
//
// Placeholders are filled in a single pass: a value that itself contains a
// placeholder is a problem, never re-expanded. resolve never masks secrets;
// it marks them (Var.Secret) and keeps values out of problem messages.
package resolve

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sreeram/gurl/internal/auth"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/core/template"
	"github.com/sreeram/gurl/pkg/types"
)

// Origin is the scope a variable's final value came from.
type Origin string

const (
	OriginEnvironment Origin = "environment"
	OriginCollection  Origin = "collection"
	OriginCLI         Origin = "cli"
	OriginData        Origin = "data"
)

// Scopes holds every source of variable values for one resolution.
type Scopes struct {
	// Env must already have aliases resolved (see package doc).
	Env        map[string]string
	Collection map[string]string
	Data       map[string]string
	CLI        map[string]string

	// EnvSecrets and CollectionSecrets name the keys marked secret in the
	// environment and the collection.
	EnvSecrets        map[string]bool
	CollectionSecrets map[string]bool

	// Runtime holds values extracted from responses or set by scripts
	// earlier in a run. It is the top scope; each value keeps its own
	// origin so --persist writes it back where it came from.
	Runtime map[string]Var
}

// Var is one merged variable.
type Var struct {
	Value  string
	Origin Origin
	Secret bool
}

// Options tune a resolution.
type Options struct {
	// Auth applies the request's auth config; nil means auth.BuiltinRegistry().
	Auth *auth.Registry
	// Preview is for dry runs: {{$fn}} dynamic functions are left in place
	// and not reported, and auth is checked but not applied (applying may
	// contact a token endpoint).
	Preview bool
	// Interactive is for callers that ask the user for missing values (shell,
	// TUI): blank values count as missing, and :name / {name} path
	// parameters in the URL (plus the request's PathParams) are filled from
	// the scopes.
	Interactive bool
}

// Resolved is the outcome of a resolution.
type Resolved struct {
	// Request is always filled in as far as possible; placeholders that could
	// not be resolved are left as written. It must not be sent when Resolve
	// reported problems.
	Request client.Request
	// Vars is the merged variable view the request was resolved against.
	Vars map[string]Var
}

// ProblemKind classifies a resolution problem.
type ProblemKind string

const (
	ProblemMissing             ProblemKind = "missing"
	ProblemLeftoverPlaceholder ProblemKind = "leftover_placeholder"
	ProblemBadType             ProblemKind = "bad_type"
	ProblemUnknownDynamic      ProblemKind = "unknown_dynamic"
	ProblemAuth                ProblemKind = "auth"
)

// Problem is one reason a saved request cannot be resolved. Message never
// contains a variable's value.
type Problem struct {
	Kind ProblemKind
	// Variable is the variable (or "$fn" dynamic function) at fault; empty
	// for auth problems.
	Variable string
	// Location is where the placeholder sits: "url", "header <key>",
	// "body" or "auth".
	Location string
	Message  string
}

// Problems lists every resolution problem in request order: URL, headers,
// body, auth.
type Problems []Problem

// Error starts with "missing required variables: a, b" when variables are
// missing, followed by the other problems as "<location>: <message>",
// separated by "; ".
func (p Problems) Error() string {
	var parts []string
	if missing := p.Missing(); len(missing) > 0 {
		parts = append(parts, "missing required variables: "+strings.Join(missing, ", "))
	}
	for _, problem := range p.Others() {
		parts = append(parts, problem.Location+": "+problem.Message)
	}
	return strings.Join(parts, "; ")
}

// Err returns p as an error, or nil when there are no problems.
func (p Problems) Err() error {
	if len(p) == 0 {
		return nil
	}
	return p
}

// Missing returns the names of missing variables in order, without duplicates.
func (p Problems) Missing() []string {
	var names []string
	seen := make(map[string]bool)
	for _, problem := range p {
		if problem.Kind == ProblemMissing && !seen[problem.Variable] {
			seen[problem.Variable] = true
			names = append(names, problem.Variable)
		}
	}
	return names
}

// Others returns the problems that are not missing variables.
func (p Problems) Others() Problems {
	var others Problems
	for _, problem := range p {
		if problem.Kind != ProblemMissing {
			others = append(others, problem)
		}
	}
	return others
}

// collectionRefPattern matches a {{name}} reference to an environment key
// inside a collection value.
var collectionRefPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// Vars merges the scopes into one variable view, with origins and secret
// flags. A variable is secret when its winning origin marks it secret.
func Vars(s Scopes) map[string]Var {
	vars := make(map[string]Var, len(s.Env)+len(s.Collection)+len(s.Data)+len(s.CLI)+len(s.Runtime))
	for key, value := range s.Env {
		vars[key] = Var{Value: value, Origin: OriginEnvironment, Secret: s.EnvSecrets[key]}
	}
	for key, value := range s.Collection {
		vars[key] = Var{Value: resolveEnvRefs(value, s.Env), Origin: OriginCollection, Secret: s.CollectionSecrets[key]}
	}
	for key, value := range s.Data {
		vars[key] = Var{Value: value, Origin: OriginData}
	}
	for key, value := range s.CLI {
		vars[key] = Var{Value: value, Origin: OriginCLI}
	}
	for key, v := range s.Runtime {
		v.Secret = v.Secret || s.secret(v.Origin, key)
		vars[key] = v
	}
	return vars
}

func (s Scopes) secret(origin Origin, key string) bool {
	switch origin {
	case OriginEnvironment:
		return s.EnvSecrets[key]
	case OriginCollection:
		return s.CollectionSecrets[key]
	}
	return false
}

// resolveEnvRefs replaces {{name}} references to known environment keys in a
// collection value; unknown references are left for request resolution to
// report.
func resolveEnvRefs(value string, env map[string]string) string {
	if len(env) == 0 || !strings.Contains(value, "{{") {
		return value
	}
	return collectionRefPattern.ReplaceAllStringFunc(value, func(match string) string {
		if v, ok := env[match[2:len(match)-2]]; ok {
			return v
		}
		return match
	})
}

// Resolve turns req into the request to send, resolving placeholders in the
// URL, header keys and values, body and auth params, then applying auth. All
// problems are collected, not just the first.
func Resolve(req *types.SavedRequest, s Scopes, o Options) (Resolved, Problems) {
	vars := Vars(s)
	f := &filler{values: make(map[string]string, len(vars)), dynamic: template.DynamicResolve}
	if o.Preview {
		f.dynamic = template.DynamicKeep
	}
	for key, v := range vars {
		if o.Interactive && strings.TrimSpace(v.Value) == "" {
			continue
		}
		f.values[key] = v.Value
	}

	url := req.URL
	if o.Interactive {
		url = f.pathParams(req)
	}
	out := client.Request{
		Method:  req.Method,
		URL:     f.fill("url", url),
		Headers: make([]client.Header, 0, len(req.Headers)),
	}
	if out.Method == "" {
		out.Method = "GET"
	}
	for _, h := range req.Headers {
		location := "header " + h.Key
		out.Headers = append(out.Headers, client.Header{Key: f.fill(location, h.Key), Value: f.fill(location, h.Value)})
	}
	if req.Body != "" {
		out.Body = f.fill("body", req.Body)
	}
	f.applyAuth(req.AuthConfig, &out, o)

	return Resolved{Request: out, Vars: vars}, f.problems
}

type filler struct {
	values   map[string]string
	dynamic  template.Dynamic
	problems Problems
	seen     map[Problem]bool
}

func (f *filler) add(p Problem) {
	if f.seen == nil {
		f.seen = make(map[Problem]bool)
	}
	if !f.seen[p] {
		f.seen[p] = true
		f.problems = append(f.problems, p)
	}
}

func (f *filler) fill(location, text string) string {
	out, issues := template.Fill(text, f.values, f.dynamic)
	for _, issue := range issues {
		p := Problem{Variable: issue.Name, Location: location, Message: issue.Message}
		switch issue.Kind {
		case template.IssueMissing:
			p.Kind = ProblemMissing
			p.Message = fmt.Sprintf("missing required variable %q", issue.Name)
		case template.IssueLeftover:
			p.Kind = ProblemLeftoverPlaceholder
		case template.IssueBadType:
			p.Kind = ProblemBadType
		case template.IssueDynamic:
			p.Kind = ProblemUnknownDynamic
		}
		f.add(p)
	}
	return out
}

// pathParams fills the :name / {name} path parameters of req.URL. Every
// parameter in the URL and every entry of req.PathParams must have a
// non-blank value.
func (f *filler) pathParams(req *types.SavedRequest) string {
	names := template.PathParamNames(req.URL)
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		seen[name] = true
	}
	for _, p := range req.PathParams {
		if p.Name != "" && !seen[p.Name] {
			seen[p.Name] = true
			names = append(names, p.Name)
		}
	}
	params := make(map[string]string, len(names))
	for _, name := range names {
		if v, ok := f.values[name]; ok {
			params[name] = v
			continue
		}
		f.add(Problem{Kind: ProblemMissing, Variable: name, Location: "url", Message: fmt.Sprintf("missing required variable %q", name)})
	}
	return template.FillPathParams(req.URL, params)
}

// applyAuth resolves the auth params and applies the auth config to out. Auth
// is only applied when everything else resolved and this is not a preview.
func (f *filler) applyAuth(cfg *types.AuthConfig, out *client.Request, o Options) {
	if cfg == nil || cfg.Type == "" || cfg.Type == "none" {
		return
	}
	names := make([]string, 0, len(cfg.Params))
	for name := range cfg.Params {
		names = append(names, name)
	}
	sort.Strings(names)
	params := make(map[string]string, len(names))
	for _, name := range names {
		params[name] = f.fill("auth", cfg.Params[name])
	}

	registry := o.Auth
	if registry == nil {
		registry = auth.BuiltinRegistry()
	}
	if registry.Get(cfg.Type) == nil {
		f.add(Problem{Kind: ProblemAuth, Location: "auth", Message: fmt.Sprintf("unknown auth type %q", cfg.Type)})
		return
	}
	if o.Preview || len(f.problems) > 0 {
		return
	}
	if err := registry.Apply(cfg.Type, out, params); err != nil {
		f.add(Problem{Kind: ProblemAuth, Location: "auth", Message: err.Error()})
	}
}
