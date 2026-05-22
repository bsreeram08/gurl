package env

import "fmt"

type PersistStore interface {
	GetEnvByName(name string) (*Environment, error)
	SaveEnv(env *Environment) error
}

type ActiveEnvStore interface {
	GetActiveEnv() (string, error)
}

func ResolvePersistEnvironmentName(store ActiveEnvStore, explicitName string) (string, error) {
	if explicitName != "" {
		return explicitName, nil
	}
	if store == nil {
		return "", fmt.Errorf("--persist requires --env or an active environment")
	}
	activeName, err := store.GetActiveEnv()
	if err != nil {
		return "", fmt.Errorf("failed to resolve active environment for --persist: %w", err)
	}
	if activeName == "" {
		return "", fmt.Errorf("--persist requires --env or an active environment")
	}
	return activeName, nil
}

func PersistVariables(store PersistStore, envName string, vars map[string]string) (map[string]string, error) {
	if envName == "" {
		return nil, fmt.Errorf("--persist requires --env or an active environment")
	}
	if store == nil {
		return nil, fmt.Errorf("environment storage is not configured")
	}

	current, err := store.GetEnvByName(envName)
	if err != nil {
		return nil, fmt.Errorf("failed to load environment %q for --persist: %w", envName, err)
	}
	if current == nil {
		return nil, fmt.Errorf("environment %q not found", envName)
	}

	persisted := copyEnvStringMap(vars)
	if len(persisted) == 0 {
		return persisted, nil
	}

	clone := CloneEnvironment(current)
	if clone.Variables == nil {
		clone.Variables = make(map[string]string)
	}
	if clone.SecretKeys == nil {
		clone.SecretKeys = make(map[string]bool)
	}
	for key, value := range persisted {
		clone.SetVariable(key, value)
	}

	if err := store.SaveEnv(clone); err != nil {
		return nil, fmt.Errorf("failed to persist variables to environment %q: %w", envName, err)
	}
	return persisted, nil
}

func CloneEnvironment(source *Environment) *Environment {
	if source == nil {
		return nil
	}
	clone := *source
	clone.Variables = copyEnvStringMap(source.Variables)
	clone.SecretKeys = copyEnvBoolMap(source.SecretKeys)
	return &clone
}

func MaskedValue(environment *Environment, key string, value string) string {
	if environment != nil && environment.IsSecret(key) {
		return MaskSecret(value)
	}
	return value
}

func copyEnvStringMap(source map[string]string) map[string]string {
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func copyEnvBoolMap(source map[string]bool) map[string]bool {
	copy := make(map[string]bool, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
