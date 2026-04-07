package env

import (
	"regexp"
)

type Resolver struct {
	storage *EnvStorage
}

func NewResolver(storage *EnvStorage) *Resolver {
	return &Resolver{storage: storage}
}

var variablePattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

func (r *Resolver) ResolveVariables(text string, envID string) (string, error) {
	if envID == "" {
		return text, nil
	}

	env, err := r.storage.GetEnv(envID)
	if err != nil {
		return text, nil
	}

	vars := make(map[string]string)
	currentEnv := env
	for currentEnv != nil {
		for k, v := range currentEnv.Variables {
			vars[k] = v
		}
		if currentEnv.ParentID == "" {
			break
		}
		currentEnv, _ = r.storage.GetEnv(currentEnv.ParentID)
	}

	result := variablePattern.ReplaceAllStringFunc(text, func(match string) string {
		varName := match[2 : len(match)-2]
		if val, ok := vars[varName]; ok {
			return val
		}
		return match
	})

	return result, nil
}

func Resolve(text string, variables map[string]string) string {
	if len(variables) == 0 {
		return text
	}

	result := variablePattern.ReplaceAllStringFunc(text, func(match string) string {
		varName := match[2 : len(match)-2]
		if val, ok := variables[varName]; ok {
			return val
		}
		return match
	})

	return result
}
