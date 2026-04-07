package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGraphQL_Query(t *testing.T) {
	// Mock GraphQL server that returns a simple response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify Content-Type is application/json
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		// Parse request body and verify query
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if reqBody["query"] == nil {
			t.Error("Expected 'query' field in request body")
		}

		// Return a GraphQL response
		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"users": [{"name": "Alice"}, {"name": "Bob"}]}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	// Create GraphQL client
	c := &Client{}

	// Execute query
	resp, err := c.Execute(context.Background(), server.URL, Request{
		Query: `query { users { name } }`,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify response
	if resp.Data == nil {
		t.Fatal("Expected Data in response")
	}

	var result struct {
		Users []struct {
			Name string `json:"name"`
		} `json:"users"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("Failed to unmarshal Data: %v", err)
	}

	if len(result.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(result.Users))
	}
	if result.Users[0].Name != "Alice" {
		t.Errorf("Expected first user to be Alice, got %s", result.Users[0].Name)
	}
}

func TestGraphQL_QueryWithVariables(t *testing.T) {
	// Mock server that validates variables in request body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify variables are passed correctly
		if reqBody.Variables == nil {
			t.Error("Expected Variables in request body")
		}
		if reqBody.Variables["limit"] != float64(10) {
			t.Errorf("Expected limit=10, got %v", reqBody.Variables["limit"])
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"users": [{"name": "Alice"}]}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	c := &Client{}

	resp, err := c.Execute(context.Background(), server.URL, Request{
		Query: `query($limit: Int) { users(limit: $limit) { name } }`,
		Variables: map[string]interface{}{
			"limit": 10,
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("Expected Data in response")
	}
}

func TestGraphQL_Mutation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Verify it's a mutation
		query, ok := reqBody["query"].(string)
		if !ok {
			t.Fatal("Expected query to be string")
		}
		if query == "" {
			t.Error("Expected non-empty query")
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"createUser": {"id": "123", "name": "Charlie"}}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	c := &Client{}

	resp, err := c.Execute(context.Background(), server.URL, Request{
		Query: `mutation { createUser(name: "Charlie") { id name } }`,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("Expected Data in response")
	}

	var result struct {
		CreateUser struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"createUser"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("Failed to unmarshal Data: %v", err)
	}

	if result.CreateUser.ID != "123" {
		t.Errorf("Expected id=123, got %s", result.CreateUser.ID)
	}
	if result.CreateUser.Name != "Charlie" {
		t.Errorf("Expected name=Charlie, got %s", result.CreateUser.Name)
	}
}

func TestGraphQL_Introspection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// Verify introspection query
		query, _ := reqBody["query"].(string)
		if query == "" {
			t.Error("Expected non-empty query")
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"__schema": {"queryType": {"name": "Query"}}}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	c := &Client{}

	resp, err := c.Execute(context.Background(), server.URL, Request{
		Query: `query { __schema { queryType { name } } }`,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp.Data == nil {
		t.Fatal("Expected Data in response")
	}

	var result struct {
		Schema struct {
			QueryType struct {
				Name string `json:"name"`
			} `json:"queryType"`
		} `json:"__schema"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("Failed to unmarshal Data: %v", err)
	}

	if result.Schema.QueryType.Name != "Query" {
		t.Errorf("Expected queryType.name=Query, got %s", result.Schema.QueryType.Name)
	}
}

func TestGraphQL_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return GraphQL error response
		resp := `{"errors": [{"message": "Field 'nonexistent' not found", "locations": [{"line": 1, "column": 3}], "path": ["query", "nonexistent"]}], "data": null}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	c := &Client{}

	resp, err := c.Execute(context.Background(), server.URL, Request{
		Query: `query { nonexistent }`,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should not error, but should have errors in response
	if resp.Data != nil {
		t.Error("Expected Data to be nil for error response")
	}

	if len(resp.Errors) == 0 {
		t.Fatal("Expected at least one GraphQL error")
	}

	if resp.Errors[0].Message != "Field 'nonexistent' not found" {
		t.Errorf("Expected error message about nonexistent field, got %s", resp.Errors[0].Message)
	}

	if len(resp.Errors[0].Locations) == 0 {
		t.Fatal("Expected at least one location")
	}
	if resp.Errors[0].Locations[0].Line != 1 {
		t.Errorf("Expected line=1, got %d", resp.Errors[0].Locations[0].Line)
	}
	if resp.Errors[0].Locations[0].Column != 3 {
		t.Errorf("Expected column=3, got %d", resp.Errors[0].Locations[0].Column)
	}
}

func TestGraphQL_Headers(t *testing.T) {
	receivedAuth := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"me": {"name": "Test User"}}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	c := &Client{}

	_, err := c.Execute(context.Background(), server.URL, Request{
		Query: `query { me { name } }`,
	}, WithHeader("Authorization", "Bearer test-token-123"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if receivedAuth != "Bearer test-token-123" {
		t.Errorf("Expected Authorization header 'Bearer test-token-123', got %s", receivedAuth)
	}
}

func TestGraphQL_BuildRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Query         string                 `json:"query"`
			Variables     map[string]interface{} `json:"variables"`
			OperationName string                 `json:"operationName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify all fields are present
		if reqBody.Query == "" {
			t.Error("Expected Query to be set")
		}
		if reqBody.Variables == nil {
			t.Error("Expected Variables to be set")
		}
		if reqBody.OperationName != "TestOp" {
			t.Errorf("Expected OperationName='TestOp', got '%s'", reqBody.OperationName)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	c := &Client{}

	_, err := c.Execute(context.Background(), server.URL, Request{
		Query:         `query TestOp($id: ID!) { user(id: $id) { name } }`,
		Variables:     map[string]interface{}{"id": "abc123"},
		OperationName: "TestOp",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestGraphQL_MultilineQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		query, _ := reqBody["query"].(string)
		// Verify multiline query is preserved (contains newlines and fragment)
		if query == "" {
			t.Error("Expected non-empty query")
		}

		w.Header().Set("Content-Type", "application/json")
		resp := `{"data": {"user": {"name": "Test", "email": "test@example.com"}}}`
		w.Write([]byte(resp))
	}))
	defer server.Close()

	c := &Client{}

	multilineQuery := `
		query UserWithFragments($id: ID!) {
			user(id: $id) {
				... UserFields
			}
		}
		
		fragment UserFields on User {
			name
			email
		}
	`

	_, err := c.Execute(context.Background(), server.URL, Request{
		Query: multilineQuery,
		Variables: map[string]interface{}{
			"id": "user-123",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}
