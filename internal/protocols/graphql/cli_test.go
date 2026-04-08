package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGraphQLCommand_ReturnsCorrectCommand(t *testing.T) {
	cmd := GraphQLCommand(nil)

	if cmd.Name != "graphql" {
		t.Errorf("expected name 'graphql', got '%s'", cmd.Name)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "gql" {
		t.Errorf("expected alias 'gql', got %v", cmd.Aliases)
	}

	if cmd.Usage != "Execute a GraphQL query" {
		t.Errorf("expected usage 'Execute a GraphQL query', got '%s'", cmd.Usage)
	}

	// Check flags exist
	flagNames := make(map[string]bool)
	for _, flag := range cmd.Flags {
		flagNames[flag.Names()[0]] = true
	}

	expectedFlags := []string{"query", "query-file", "vars", "operation-name", "format", "color"}
	for _, name := range expectedFlags {
		if !flagNames[name] {
			t.Errorf("expected flag '%s' to exist", name)
		}
	}
}

func TestGraphQLCommand_MissingEndpoint(t *testing.T) {
	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{"graphql"})
	if err == nil {
		t.Fatal("expected error for missing endpoint, got nil")
	}

	if err.Error() != "endpoint is required" {
		t.Errorf("expected 'endpoint is required', got '%s'", err.Error())
	}
}

func TestGraphQLCommand_MissingQuery(t *testing.T) {
	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{"graphql", "https://example.com/graphql"})
	if err == nil {
		t.Fatal("expected error for missing query, got nil")
	}

	if err.Error() != "either --query or --query-file is required" {
		t.Errorf("expected 'either --query or --query-file is required', got '%s'", err.Error())
	}
}

func TestGraphQLCommand_ValidQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if reqBody["query"] == nil {
			t.Error("expected 'query' field in request body")
		}

		query, ok := reqBody["query"].(string)
		if !ok || query != "{ users { name } }" {
			t.Errorf("expected query '{ users { name } }', got %v", reqBody["query"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"users": [{"name": "Alice"}, {"name": "Bob"}]}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{"graphql", server.URL, "--query", "{ users { name } }"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLCommand_QueryFile(t *testing.T) {
	// Create a temp file with query content
	tmpDir := t.TempDir()
	queryFile := filepath.Join(tmpDir, "test.graphql")
	err := os.WriteFile(queryFile, []byte("{ hello }"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp query file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		query, ok := reqBody["query"].(string)
		if !ok || query != "{ hello }" {
			t.Errorf("expected query '{ hello }', got %v", reqBody["query"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"hello": "world"}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	err = cmd.Run(context.Background(), []string{"graphql", server.URL, "--query-file", queryFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLCommand_QueryFileNotFound(t *testing.T) {
	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{"graphql", "https://example.com/graphql", "--query-file", "/nonexistent/path/query.graphql"})
	if err == nil {
		t.Fatal("expected error for nonexistent query file, got nil")
	}

	expectedPrefix := "failed to read query file:"
	if len(err.Error()) < len(expectedPrefix) || err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected error starting with '%s', got '%s'", expectedPrefix, err.Error())
	}
}

func TestGraphQLCommand_VarsFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if reqBody.Variables == nil {
			t.Fatal("expected variables to be set")
		}
		if reqBody.Variables["limit"] != float64(10) {
			t.Errorf("expected limit=10, got %v", reqBody.Variables["limit"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"users": []}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{
		"graphql", server.URL,
		"--query", "query($limit: Int) { users(limit: $limit) { name } }",
		"--vars", `{"limit": 10}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLCommand_InvalidVarsJSON(t *testing.T) {
	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{
		"graphql", "https://example.com/graphql",
		"--query", "{ hello }",
		"--vars", `{invalid json}`,
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON in vars, got nil")
	}

	expectedPrefix := "failed to parse variables JSON:"
	if len(err.Error()) < len(expectedPrefix) || err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected error starting with '%s', got '%s'", expectedPrefix, err.Error())
	}
}

func TestGraphQLCommand_OperationName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody["operationName"] != "GetUsers" {
			t.Errorf("expected operationName='GetUsers', got %v", reqBody["operationName"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"users": []}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{
		"graphql", server.URL,
		"--query", "query GetUsers { users { name } }",
		"--operation-name", "GetUsers",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLCommand_GraphQLErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return error response with data
		resp := `{"errors": [{"message": "Field 'nonexistent' not found", "locations": [{"line": 1, "column": 3}]}], "data": {"nonexistent": null}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	// Action should not return error even with GraphQL errors (it prints them to stderr)
	err := cmd.Run(context.Background(), []string{"graphql", server.URL, "--query", "{ nonexistent }"})
	if err != nil {
		t.Fatalf("expected no error from Action, got: %v", err)
	}
}

func TestGraphQLCommand_ErrorResponseNoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return error response without data
		resp := `{"errors": [{"message": "Field 'nonexistent' not found", "locations": [{"line": 1, "column": 3}]}], "data": null}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	// When data is null but errors exist, Action returns nil (errors printed to stderr)
	err := cmd.Run(context.Background(), []string{"graphql", server.URL, "--query", "{ nonexistent }"})
	if err != nil {
		t.Fatalf("expected no error from Action, got: %v", err)
	}
}

func TestGraphQLCommand_RequestFailure(t *testing.T) {
	// Server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	server.Close()

	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{"graphql", server.URL, "--query", "{ hello }"})
	if err == nil {
		t.Fatal("expected error for failed request, got nil")
	}
}

func TestGraphQLCommand_FormatFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"hello": "world"}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{
		"graphql", server.URL,
		"--query", "{ hello }",
		"--format", "json",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLCommand_ColorFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"hello": "world"}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	err := cmd.Run(context.Background(), []string{
		"graphql", server.URL,
		"--query", "{ hello }",
		"--color",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLCommand_FlagAliases(t *testing.T) {
	cmd := GraphQLCommand(nil)

	// Check flag aliases
	flagAliases := map[string][]string{
		"query":          {"q"},
		"query-file":     {"f"},
		"vars":           {"v"},
		"operation-name": {"op"},
		"format":         {"fmt"},
		"color":          {"c"},
	}

	for flagName, expectedAliases := range flagAliases {
		found := false
		for _, flag := range cmd.Flags {
			if flag.Names()[0] == flagName {
				found = true
				aliases := flag.Names()[1:]
				if len(aliases) != len(expectedAliases) {
					t.Errorf("flag %s: expected aliases %v, got %v", flagName, expectedAliases, aliases)
				}
				for i, alias := range aliases {
					if alias != expectedAliases[i] {
						t.Errorf("flag %s: expected alias '%s' at position %d, got '%s'", flagName, expectedAliases[i], i, alias)
					}
				}
				break
			}
		}
		if !found {
			t.Errorf("flag '%s' not found", flagName)
		}
	}
}

func TestGraphQLCommand_QueryAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		query, ok := reqBody["query"].(string)
		if !ok || query != "{ hello }" {
			t.Errorf("expected query '{ hello }', got %v", reqBody["query"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"hello": "world"}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cmd := GraphQLCommand(nil)

	// Use -q alias for --query
	err := cmd.Run(context.Background(), []string{"graphql", server.URL, "-q", "{ hello }"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
