package env

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Environment struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Variables  map[string]string `json:"variables"`
	SecretKeys map[string]bool   `json:"secret_keys"`
	ParentID   string            `json:"parent_id"`
	CreatedAt  int64             `json:"created_at"`
	UpdatedAt  int64             `json:"updated_at"`
}

func NewEnvironment(name string, parentID string) *Environment {
	now := time.Now().Unix()
	return &Environment{
		ID:         uuid.New().String(),
		Name:       name,
		Variables:  make(map[string]string),
		SecretKeys: make(map[string]bool),
		ParentID:   parentID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (e *Environment) SetVariable(key, value string) {
	e.Variables[key] = value
	e.UpdatedAt = time.Now().Unix()
}

func (e *Environment) GetVariable(key string) (string, bool) {
	v, ok := e.Variables[key]
	return v, ok
}

func (e *Environment) DeleteVariable(key string) {
	delete(e.Variables, key)
	delete(e.SecretKeys, key)
	e.UpdatedAt = time.Now().Unix()
}

func (e *Environment) SetSecretVariable(key, value string) {
	e.Variables[key] = value
	e.SecretKeys[key] = true
	e.UpdatedAt = time.Now().Unix()
}

func (e *Environment) IsSecret(key string) bool {
	return e.SecretKeys[key]
}

// aliasPattern matches a {{name}} reference inside an environment value.
var aliasPattern = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

// maxAliasDepth bounds how many levels of {{name}} references between
// variables of the same environment are followed.
const maxAliasDepth = 8

// ResolvedVariables returns a copy of the environment's variables in which
// {{name}} references to other keys of the same environment are resolved
// (e.g. API_KEY="{{SA_KEY}}"). Environment config is trusted, so aliasing is
// resolved here rather than at request substitution time. References to
// unknown keys, references that form a cycle, and chains deeper than
// maxAliasDepth are left literal so request substitution reports them.
func (e *Environment) ResolvedVariables() map[string]string {
	resolved := make(map[string]string, len(e.Variables))
	if len(e.Variables) == 0 {
		return resolved
	}

	// Resolution is computed without a depth cut-off (cycles are cut via the
	// call stack, so recursion is bounded by the number of keys) along with
	// each key's alias-chain height; keys whose chain is cyclic or longer
	// than maxAliasDepth keep their raw value. This keeps results independent
	// of map iteration order.
	type result struct {
		value  string
		height int
	}
	memo := make(map[string]result, len(e.Variables))
	var resolve func(key string, stack []string) (result, bool)
	resolve = func(key string, stack []string) (result, bool) {
		if r, ok := memo[key]; ok {
			return r, true
		}
		raw := e.Variables[key]
		if !strings.Contains(raw, "{{") {
			memo[key] = result{value: raw}
			return memo[key], true
		}
		stack = append(stack, key)
		complete := true
		height := 0
		out := aliasPattern.ReplaceAllStringFunc(raw, func(match string) string {
			ref := match[2 : len(match)-2]
			if _, ok := e.Variables[ref]; !ok {
				return match
			}
			for _, active := range stack {
				if active == ref {
					complete = false
					return match
				}
			}
			r, ok := resolve(ref, stack)
			if !ok {
				complete = false
			}
			if r.height+1 > height {
				height = r.height + 1
			}
			return r.value
		})
		r := result{value: out, height: height}
		if complete {
			memo[key] = r
		}
		return r, complete
	}

	for key, raw := range e.Variables {
		if r, ok := resolve(key, nil); ok && r.height <= maxAliasDepth {
			resolved[key] = r.value
		} else {
			resolved[key] = raw // part of a cycle or too deep: keep literal
		}
	}
	return resolved
}
