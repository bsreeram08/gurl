package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

func TestSaveCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		checkFn func(*testing.T, *mockDB)
	}{
		{
			name:    "saves basic request with name and URL",
			args:    []string{"test", "https://example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("test")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.URL != "https://example.com" {
					t.Errorf("expected URL 'https://example.com', got '%s'", req.URL)
				}
				if req.Method != "GET" {
					t.Errorf("expected method 'GET', got '%s'", req.Method)
				}
			},
		},
		{
			name:    "saves with custom format flag",
			args:    []string{"json_req", "https://api.example.com", "-f", "json"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("json_req")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req.OutputFormat != "json" {
					t.Errorf("expected format 'json', got '%s'", req.OutputFormat)
				}
			},
		},
		{
			name:    "fails when name argument is missing",
			args:    []string{"https://example.com"},
			wantErr: true,
		},
		{
			name:    "fails when URL argument is missing",
			args:    []string{"testname"},
			wantErr: true,
		},
		{
			name:    "saves multiple requests",
			args:    []string{"multi1", "https://first.example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				// Save another request
				db.names["multi2"] = "id2"
				db.requests["id2"] = &types.SavedRequest{
					ID:   "id2",
					Name: "multi2",
					URL:  "https://second.example.com",
				}

				req1, _ := db.GetRequestByName("multi1")
				req2, _ := db.GetRequestByName("multi2")
				if req1 == nil || req2 == nil {
					t.Fatal("expected both requests to exist")
				}
			},
		},
		{
			name:    "saves with description",
			args:    []string{"with_desc", "https://desc.example.com", "-d", "My description"},
			wantErr: false,
		},
		{
			name:    "saves with multiple tags",
			args:    []string{"multi_tag", "https://tag.example.com", "--tag", "api", "--tag", "auth"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("multi_tag")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if len(req.Tags) != 2 {
					t.Errorf("expected 2 tags, got %d", len(req.Tags))
				}
				if req.Tags[0] != "api" || req.Tags[1] != "auth" {
					t.Errorf("expected tags [api auth], got %v", req.Tags)
				}
			},
		},
		{
			name:    "saves with single tag",
			args:    []string{"single_tag", "https://single.example.com", "--tag", "important"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("single_tag")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if len(req.Tags) != 1 {
					t.Errorf("expected 1 tag, got %d", len(req.Tags))
				}
				if req.Tags[0] != "important" {
					t.Errorf("expected tag [important], got %v", req.Tags)
				}
			},
		},
		{
			name:    "saves with --curl flag and full curl command",
			args:    []string{"curl_test", "--curl", `curl -X POST -H "Content-Type: application/json" -d '{"key":"value"}' https://example.com`},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("curl_test")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.Method != "POST" {
					t.Errorf("expected method 'POST', got '%s'", req.Method)
				}
				if req.URL != "https://example.com" {
					t.Errorf("expected URL 'https://example.com', got '%s'", req.URL)
				}
				if len(req.Headers) != 1 {
					t.Errorf("expected 1 header, got %d", len(req.Headers))
				}
				if req.Body != `{"key":"value"}` {
					t.Errorf("expected body '{\"key\":\"value\"}', got '%s'", req.Body)
				}
			},
		},
		{
			name:    "saves with -X -H -d individual flags",
			args:    []string{"flags_test", "-X", "PUT", "-H", "Authorization: Bearer token123", "-d", "name=test", "https://api.example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("flags_test")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.Method != "PUT" {
					t.Errorf("expected method 'PUT', got '%s'", req.Method)
				}
				if req.URL != "https://api.example.com" {
					t.Errorf("expected URL 'https://api.example.com', got '%s'", req.URL)
				}
				if len(req.Headers) != 1 {
					t.Errorf("expected 1 header, got %d", len(req.Headers))
				}
				if req.Headers[0].Key != "Authorization" || req.Headers[0].Value != "Bearer token123" {
					t.Errorf("expected header 'Authorization: Bearer token123', got '%s: %s'", req.Headers[0].Key, req.Headers[0].Value)
				}
				if req.Body != "name=test" {
					t.Errorf("expected body 'name=test', got '%s'", req.Body)
				}
			},
		},
		{
			name:    "saves direct URL using --name flag",
			args:    []string{"--name", "named_request", "https://api.example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("named_request")
				if err != nil {
					t.Fatalf("expected request saved under --name value: %v", err)
				}
				if req.URL != "https://api.example.com" {
					t.Errorf("expected URL from positional argument, got %q", req.URL)
				}
			},
		},
		{
			name:    "saves flag-based request using --name flag",
			args:    []string{"--name", "create_order", "-X", "POST", "-H", "Content-Type: application/json", "-d", `{"ok":true}`, "https://api.example.com/orders"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("create_order")
				if err != nil {
					t.Fatalf("expected request saved under --name value: %v", err)
				}
				if req.URL != "https://api.example.com/orders" {
					t.Errorf("expected URL from positional argument, got %q", req.URL)
				}
				if req.Method != "POST" {
					t.Errorf("expected POST method, got %q", req.Method)
				}
			},
		},
		{
			name:    "rejects extra positional args when --name is set",
			args:    []string{"--name", "named_request", "https://api.example.com", "extra"},
			wantErr: true,
		},
		{
			name:    "saves with multiple -H flags",
			args:    []string{"multi_header", "-X", "POST", "-H", "Content-Type: application/json", "-H", "Accept: text/plain", "https://multi-header.example.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("multi_header")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if len(req.Headers) != 2 {
					t.Errorf("expected 2 headers, got %d", len(req.Headers))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			cmd := SaveCommand(db)

			// Build args slice: first element is command name (ignored by action)
			fullArgs := append([]string{"save"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, db)
			}
		})
	}
}

func TestSaveCommandConfirmationUsesSavedNameAndURL(t *testing.T) {
	db := newMockDB()
	cmd := SaveCommand(db)

	output := captureStdout(t, func() {
		err := cmd.Run(context.Background(), []string{"save", "--name", "foo", "https://example.com"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	want := "✓ Saved request 'foo' (https://example.com)"
	if !strings.Contains(output, want) {
		t.Errorf("expected confirmation %q, got %q", want, output)
	}
}

func TestSaveCommandErrorsForMissingCollectionNonInteractive(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	oldInteractive := saveCollectionIsInteractive
	saveCollectionIsInteractive = func() bool { return false }
	defer func() { saveCollectionIsInteractive = oldInteractive }()

	cmd := SaveCommand(db)
	err := cmd.Run(context.Background(), []string{"save", "list payments", "https://example.com/payments", "--collection", "payments"})
	if err == nil || !strings.Contains(err.Error(), "collection \"payments\" does not exist") {
		t.Fatalf("expected missing collection error, got %v", err)
	}
	if _, err := db.GetRequestByName("list payments"); err == nil {
		t.Fatal("request should not be saved when collection is missing")
	}
}

func TestSaveCommandReturnsCollectionLookupErrors(t *testing.T) {
	db := &failingCollectionLookupDB{
		mockDB:    newMockDB(),
		lookupErr: errors.New("collection index unavailable"),
	}

	oldInteractive := saveCollectionIsInteractive
	saveCollectionIsInteractive = func() bool { return false }
	defer func() { saveCollectionIsInteractive = oldInteractive }()

	cmd := SaveCommand(db)
	err := cmd.Run(context.Background(), []string{"save", "list payments", "https://example.com/payments", "--collection", "payments"})
	if err == nil || !strings.Contains(err.Error(), "collection index unavailable") {
		t.Fatalf("expected lookup error, got %v", err)
	}
	if _, err := db.GetRequestByName("list payments"); err == nil {
		t.Fatal("request should not be saved when collection lookup fails")
	}
}

func TestSaveCommandSavesIntoExistingCollection(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	if err := db.SaveCollection(types.NewCollection("payments")); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	cmd := SaveCommand(db)
	if err := cmd.Run(context.Background(), []string{"save", "list payments", "https://example.com/payments", "--collection", "payments"}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	req, err := db.GetRequestByName("list payments")
	if err != nil {
		t.Fatalf("GetRequestByName failed: %v", err)
	}
	if req.Collection != "payments" {
		t.Fatalf("expected request collection payments, got %q", req.Collection)
	}
}

func TestSaveCommandCanCreateMissingCollectionInteractively(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	oldInteractive := saveCollectionIsInteractive
	saveCollectionIsInteractive = func() bool { return true }
	defer func() { saveCollectionIsInteractive = oldInteractive }()

	oldStdin := os.Stdin
	stdin, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	if _, err := writer.WriteString("yes\n"); err != nil {
		t.Fatalf("failed to write confirmation: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close stdin writer: %v", err)
	}
	os.Stdin = stdin
	defer func() {
		os.Stdin = oldStdin
		stdin.Close()
	}()

	cmd := SaveCommand(db)
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"save", "list payments", "https://example.com/payments", "--collection", "payments"}); err != nil {
			t.Fatalf("save failed: %v", err)
		}
	})

	if !strings.Contains(output, "Collection \"payments\" does not exist. Create it? [y/N]") {
		t.Fatalf("expected create prompt, got %q", output)
	}
	if _, err := db.GetCollectionByName("payments"); err != nil {
		t.Fatalf("expected collection to be created: %v", err)
	}
	req, err := db.GetRequestByName("list payments")
	if err != nil {
		t.Fatalf("expected request to be saved: %v", err)
	}
	if req.Collection != "payments" {
		t.Fatalf("expected request collection payments, got %q", req.Collection)
	}
}

func TestSaveCommandPersistsExtractsScriptsAndJQAlias(t *testing.T) {
	db := newMockDB()
	cmd := SaveCommand(db)
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}

	err := cmd.Run(context.Background(), []string{
		"save",
		"login",
		"https://api.example.com/login",
		"--extract", "token=jsonpath:$.token",
		"--extract", "requestId=jq:$.request.id",
		"--pre-script", "gurl.setVar('tenant', 'acme')",
		"--post-script", "gurl.setVar('seen', 'yes')",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := db.GetRequestByName("login")
	if err != nil {
		t.Fatalf("load saved request: %v", err)
	}
	if req.PreScript != "gurl.setVar('tenant', 'acme')" {
		t.Fatalf("pre script not persisted: %q", req.PreScript)
	}
	if req.PostScript != "gurl.setVar('seen', 'yes')" {
		t.Fatalf("post script not persisted: %q", req.PostScript)
	}
	if len(req.Extracts) != 2 {
		t.Fatalf("expected two extracts, got %#v", req.Extracts)
	}
	if req.Extracts[0].Name != "token" || req.Extracts[0].Source != "jsonpath:$.token" {
		t.Fatalf("first extract mismatch: %#v", req.Extracts[0])
	}
	if req.Extracts[1].Name != "requestId" || req.Extracts[1].Source != "jsonpath:$.request.id" {
		t.Fatalf("jq alias should store as jsonpath, got %#v", req.Extracts[1])
	}
}

type failingCollectionLookupDB struct {
	*mockDB
	lookupErr error
}

func (db *failingCollectionLookupDB) SaveCollection(collection *types.Collection) error {
	return nil
}

func (db *failingCollectionLookupDB) GetCollection(id string) (*types.Collection, error) {
	return nil, storage.ErrCollectionNotFound
}

func (db *failingCollectionLookupDB) GetCollectionByName(name string) (*types.Collection, error) {
	return nil, db.lookupErr
}

func (db *failingCollectionLookupDB) ListCollections() ([]*types.Collection, error) {
	return nil, nil
}

func (db *failingCollectionLookupDB) DeleteCollection(id string) error {
	return nil
}

func (db *failingCollectionLookupDB) UpdateCollection(collection *types.Collection) error {
	return nil
}

func TestSaveCommandPersistsAuthConfigInAllSaveModes(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantName   string
		wantType   string
		wantParams map[string]string
	}{
		{
			name:     "normal URL mode",
			args:     []string{"oauth", "https://api.example.com", "--auth", "oauth2", "--auth-param", "client_id=abc", "--auth-param", "token_url=https://auth.example.com/token", "--auth-param", "flow=client_credentials"},
			wantName: "oauth",
			wantType: "oauth2",
			wantParams: map[string]string{
				"client_id": "abc",
				"token_url": "https://auth.example.com/token",
				"flow":      "client_credentials",
			},
		},
		{
			name:     "individual flags mode",
			args:     []string{"api_key", "-X", "POST", "--auth", "apikey", "--auth-param", "header=X-Api-Key", "--auth-param", "value=secret", "https://api.example.com"},
			wantName: "api_key",
			wantType: "apikey",
			wantParams: map[string]string{
				"header": "X-Api-Key",
				"value":  "secret",
			},
		},
		{
			name:     "curl flag mode",
			args:     []string{"curl_auth", "--curl", "curl https://api.example.com", "--auth", "bearer", "--auth-param", "token={{token}}"},
			wantName: "curl_auth",
			wantType: "bearer",
			wantParams: map[string]string{
				"token": "{{token}}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			cmd := SaveCommand(db)

			err := cmd.Run(context.Background(), append([]string{"save"}, tt.args...))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			req, err := db.GetRequestByName(tt.wantName)
			if err != nil {
				t.Fatalf("load saved request: %v", err)
			}
			assertAuthConfig(t, req.AuthConfig, tt.wantType, tt.wantParams)
		})
	}
}

func TestSaveCommandRejectsUnknownAuthType(t *testing.T) {
	db := newMockDB()
	cmd := SaveCommand(db)

	err := cmd.Run(context.Background(), []string{"save", "bad_auth", "https://api.example.com", "--auth", "madeup"})
	if err == nil {
		t.Fatal("expected unknown auth type error")
	}
	if !strings.Contains(err.Error(), "unknown auth type") {
		t.Fatalf("expected unknown auth type error, got %v", err)
	}
}

func TestSaveCommandRejectsMalformedAuthParamWithExitCode2(t *testing.T) {
	db := newMockDB()
	cmd := SaveCommand(db)
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}

	err := cmd.Run(context.Background(), []string{
		"save",
		"bad",
		"https://api.example.com",
		"--auth", "bearer",
		"--auth-param", "token",
	})
	if err == nil {
		t.Fatal("expected malformed auth param error")
	}
	if !strings.Contains(err.Error(), "auth-param must be KEY=VALUE") {
		t.Fatalf("expected clear auth-param format error, got %v", err)
	}
	exitCoder, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("expected cli.ExitCoder error, got %T: %v", err, err)
	}
	if exitCoder.ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCoder.ExitCode())
	}
}

func TestSaveCommandRejectsMalformedExtractWithExitCode2(t *testing.T) {
	db := newMockDB()
	cmd := SaveCommand(db)
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}

	err := cmd.Run(context.Background(), []string{
		"save",
		"bad",
		"https://api.example.com",
		"--extract", "missingSeparator",
	})
	if err == nil {
		t.Fatal("expected malformed extract error")
	}
	if !strings.Contains(err.Error(), "extract must be VAR_NAME=METHOD:EXPRESSION") {
		t.Fatalf("expected clear extract format error, got %v", err)
	}
	exitCoder, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("expected cli.ExitCoder error, got %T: %v", err, err)
	}
	if exitCoder.ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCoder.ExitCode())
	}
}

func assertAuthConfig(t *testing.T, got *types.AuthConfig, wantType string, wantParams map[string]string) {
	t.Helper()
	if got == nil {
		t.Fatal("expected auth config to be persisted")
	}
	if got.Type != wantType {
		t.Fatalf("expected auth type %q, got %q", wantType, got.Type)
	}
	if len(got.Params) != len(wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, got.Params)
	}
	for key, want := range wantParams {
		if got.Params[key] != want {
			t.Fatalf("expected param %s=%q, got %q", key, want, got.Params[key])
		}
	}
}
