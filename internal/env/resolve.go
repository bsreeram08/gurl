package env

func ResolveVariables(cliVars, envVars, globalVars map[string]string) map[string]string {
	result := make(map[string]string)

	scopes := []map[string]string{globalVars, envVars, cliVars}

	for _, scope := range scopes {
		for k, v := range scope {
			result[k] = v
		}
	}

	return result
}
