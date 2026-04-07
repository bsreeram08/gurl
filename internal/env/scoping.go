package env

type Scoper struct {
	storage *EnvStorage
}

func NewScoper(storage *EnvStorage) *Scoper {
	return &Scoper{storage: storage}
}

func (s *Scoper) GetScopedVariables(envID string) map[string]string {
	result := make(map[string]string)

	if envID == "" {
		return result
	}

	var chain []*Environment
	visited := make(map[string]bool)

	currentEnv, err := s.storage.GetEnv(envID)
	if err != nil {
		return result
	}

	for currentEnv != nil {
		if visited[currentEnv.ID] {
			break
		}
		visited[currentEnv.ID] = true

		chain = append(chain, currentEnv)

		if currentEnv.ParentID == "" {
			break
		}

		currentEnv, _ = s.storage.GetEnv(currentEnv.ParentID)
	}

	for i := len(chain) - 1; i >= 0; i-- {
		env := chain[i]
		for k, v := range env.Variables {
			result[k] = v
		}
	}

	return result
}

func (s *Scoper) GetVariableChain(envID string, variableName string) []string {
	var chain []string

	if envID == "" {
		return chain
	}

	visited := make(map[string]bool)

	currentEnv, err := s.storage.GetEnv(envID)
	if err != nil {
		return chain
	}

	for currentEnv != nil {
		if visited[currentEnv.ID] {
			break
		}
		visited[currentEnv.ID] = true

		if val, ok := currentEnv.Variables[variableName]; ok {
			chain = append(chain, currentEnv.Name+":"+val)
		}

		if currentEnv.ParentID == "" {
			break
		}

		currentEnv, _ = s.storage.GetEnv(currentEnv.ParentID)
	}

	return chain
}
