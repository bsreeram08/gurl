package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCommand(t *testing.T) {
	tests := []struct {
		name    string
		setupFn func(t *testing.T) (string, func())
		args    []string
		stdin   string
		wantErr bool
		checkFn func(*testing.T, *mockDB)
	}{
		{
			name: "reads curl from stdin and parses correctly",
			setupFn: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			args:  []string{"--name", "test-stdin"},
			stdin: "curl -X POST https://example.com -H 'Content-Type: application/json' -d '{\"key\":\"value\"}'",
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("test-stdin")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.URL != "https://example.com" {
					t.Errorf("expected URL 'https://example.com', got '%s'", req.URL)
				}
				if req.Method != "POST" {
					t.Errorf("expected method 'POST', got '%s'", req.Method)
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
			name: "reads curl from --file flag",
			setupFn: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "curl.txt")
				err := os.WriteFile(tmpFile, []byte("curl -X GET https://api.example.com"), 0644)
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				return tmpFile, func() {}
			},
			args:  []string{"--file", "{{TEST_FILE}}", "--name", "test-file"},
			stdin: "curl -X GET https://api.example.com",
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("test-file")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.URL != "https://api.example.com" {
					t.Errorf("expected URL 'https://api.example.com', got '%s'", req.URL)
				}
				if req.Method != "GET" {
					t.Errorf("expected method 'GET', got '%s'", req.Method)
				}
			},
		},
		{
			name: "auto-generates name from URL when --name not provided",
			setupFn: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			args:  []string{},
			stdin: "curl https://auto-name.example.com/api/users",
			checkFn: func(t *testing.T, db *mockDB) {
				// Should have saved with an auto-generated name based on URL
				reqs, err := db.ListRequests(nil)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(reqs) == 0 {
					t.Fatal("expected at least one request to be saved")
				}
				// The name should contain parts of the URL
				found := false
				for _, req := range reqs {
					if req.URL == "https://auto-name.example.com/api/users" {
						found = true
						if req.Name == "" {
							t.Error("expected non-empty auto-generated name")
						}
						break
					}
				}
				if !found {
					t.Error("expected request with URL 'https://auto-name.example.com/api/users' to be saved")
				}
			},
		},
		{
			name: "applies --collection flag",
			setupFn: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			args:  []string{"--name", "test-col", "--collection", "mycollection"},
			stdin: "curl https://collection.example.com",
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("test-col")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.Collection != "mycollection" {
					t.Errorf("expected collection 'mycollection', got '%s'", req.Collection)
				}
			},
		},
		{
			name: "handles empty stdin with error",
			setupFn: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			args:    []string{},
			stdin:   "",
			wantErr: true,
		},
		{
			name: "handles invalid curl with error",
			setupFn: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			args:    []string{"--name", "bad-curl"},
			stdin:   "not a curl command",
			wantErr: true,
		},
		{
			name: "handles --file with non-existent file",
			setupFn: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			args:    []string{"--file", "/nonexistent/file.txt", "--name", "missing"},
			wantErr: true,
		},
		{
			name: "saves basic GET request from stdin",
			setupFn: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
			args:  []string{"--name", "simple-get"},
			stdin: "curl https://simple.example.com",
			checkFn: func(t *testing.T, db *mockDB) {
				req, err := db.GetRequestByName("simple-get")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req == nil {
					t.Fatal("expected request to be saved")
				}
				if req.Method != "GET" {
					t.Errorf("expected method 'GET', got '%s'", req.Method)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			cmd := DetectCommand(db)

			// Build args slice: first element is command name (ignored by action)
			fullArgs := append([]string{"detect"}, tt.args...)

			// Handle --file flag with temp file path substitution
			for i, arg := range fullArgs {
				if arg == "--file" || arg == "-f" {
					if i+1 < len(fullArgs) {
						filePath := fullArgs[i+1]
						if filePath == "{{TEST_FILE}}" {
							// Create temp file for this test
							tmpDir := t.TempDir()
							tmpFile := filepath.Join(tmpDir, "curl.txt")
							curlContent := tt.stdin
							if curlContent == "" {
								curlContent = "curl https://default.example.com"
							}
							err := os.WriteFile(tmpFile, []byte(curlContent), 0644)
							if err != nil {
								t.Fatalf("failed to create temp file: %v", err)
							}
							fullArgs[i+1] = tmpFile
						}
					}
				}
			}

			// Set up stdin if needed
			if tt.stdin != "" {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				_, err := w.WriteString(tt.stdin)
				if err != nil {
					t.Fatalf("failed to write to pipe: %v", err)
				}
				w.Close()
				defer func() {
					os.Stdin = oldStdin
				}()
			}

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

// TestParseCurlHelper tests the curl.ParseCurl function indirectly through detect
func TestDetectParsesHeadersCorrectly(t *testing.T) {
	db := newMockDB()
	cmd := DetectCommand(db)

	// Test with multiple headers
	r, w, _ := os.Pipe()
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	_, _ = w.WriteString("curl -X POST https://headers.example.com -H 'Authorization: Bearer token123' -H 'X-Custom-Header: value'")
	w.Close()

	err := cmd.Run(context.Background(), []string{"detect", "--name", "multi-header"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := db.GetRequestByName("multi-header")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(req.Headers))
	}
}
