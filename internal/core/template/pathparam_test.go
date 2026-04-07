package template

import (
	"testing"
)

func TestPathParam_SingleParam(t *testing.T) {
	url := "https://api.com/users/:id"
	params := map[string]string{"id": "123"}
	result, err := ResolvePathParams(url, params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "https://api.com/users/123"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPathParam_MultipleParams(t *testing.T) {
	url := "/users/:userId/posts/:postId"
	params := map[string]string{"userId": "42", "postId": "99"}
	result, err := ResolvePathParams(url, params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "/users/42/posts/99"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPathParam_ColonAndBrace(t *testing.T) {
	url := "/users/:userId/item/{itemId}"
	params := map[string]string{"userId": "abc", "itemId": "xyz"}
	result, err := ResolvePathParams(url, params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "/users/abc/item/xyz"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPathParam_UnresolvedParam(t *testing.T) {
	url := "/users/:userId/posts/:postId"
	params := map[string]string{"userId": "42"} // missing postId
	_, err := ResolvePathParams(url, params)
	if err == nil {
		t.Fatal("expected error for unresolved param, got nil")
	}
}

func TestPathParam_URLEncoding(t *testing.T) {
	url := "/search/:query"
	params := map[string]string{"query": "hello world & foo=bar"}
	result, err := ResolvePathParams(url, params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "/search/hello%20world%20&%20foo=bar"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPathParam_EmptyValue(t *testing.T) {
	url := "/users/:id"
	params := map[string]string{"id": ""}
	_, err := ResolvePathParams(url, params)
	if err == nil {
		t.Fatal("expected error for empty param value, got nil")
	}
}

func TestPathParam_BothSyntaxes_SameParam(t *testing.T) {
	// Same param name should resolve both syntaxes
	url := "/users/{id}/:id" // this is actually different params, not same
	params := map[string]string{"id": "42"}
	result, err := ResolvePathParams(url, params)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Both :id and {id} get replaced since they both reference "id"
	expected := "/users/42/42"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
