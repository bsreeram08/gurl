package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

func TestEditCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
		checkFn func(*testing.T, *mockDB)
	}{
		{
			name: "change method",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--method", "POST"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.Method != "POST" {
					t.Errorf("expected method POST, got %s", req.Method)
				}
			},
		},
		{
			name: "add header",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--header", "Authorization: Bearer token"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if len(req.Headers) != 1 {
					t.Errorf("expected 1 header, got %d", len(req.Headers))
				}
				if req.Headers[0].Key != "Authorization" || req.Headers[0].Value != "Bearer token" {
					t.Errorf("expected header Authorization: Bearer token, got %s: %s",
						req.Headers[0].Key, req.Headers[0].Value)
				}
			},
		},
		{
			name: "remove header",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
					Headers: []types.Header{
						{Key: "Authorization", Value: "Bearer token"},
						{Key: "Content-Type", Value: "application/json"},
					},
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--remove-header", "Authorization"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if len(req.Headers) != 1 {
					t.Errorf("expected 1 header remaining, got %d", len(req.Headers))
				}
				if req.Headers[0].Key != "Content-Type" {
					t.Errorf("expected remaining header Content-Type, got %s", req.Headers[0].Key)
				}
			},
		},
		{
			name: "change URL",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://old-api.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--url", "https://new-api.com"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.URL != "https://new-api.com" {
					t.Errorf("expected URL https://new-api.com, got %s", req.URL)
				}
			},
		},
		{
			name: "change body",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--body", `{"new":"data"}`},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.Body != `{"new":"data"}` {
					t.Errorf("expected body {\"new\":\"data\"}, got %s", req.Body)
				}
			},
		},
		{
			name: "set collection",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:         "id1",
					Name:       "api",
					URL:        "https://example.com",
					Method:     "GET",
					Collection: "v1",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--collection", "v2"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.Collection != "v2" {
					t.Errorf("expected collection v2, got %s", req.Collection)
				}
			},
		},
		{
			name: "add tag",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
					Tags:   []string{"existing"},
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--tag", "critical"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if len(req.Tags) != 2 {
					t.Errorf("expected 2 tags, got %d", len(req.Tags))
				}
				found := false
				for _, tag := range req.Tags {
					if tag == "critical" {
						found = true
					}
				}
				if !found {
					t.Error("expected tag 'critical' to be added")
				}
			},
		},
		{
			name: "add assertion",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--assert", "status=200"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if len(req.Assertions) != 1 {
					t.Errorf("expected 1 assertion, got %d", len(req.Assertions))
				}
				if req.Assertions[0].Field != "status" || req.Assertions[0].Op != "=" || req.Assertions[0].Value != "200" {
					t.Errorf("expected assertion status=200, got %s%s%s",
						req.Assertions[0].Field, req.Assertions[0].Op, req.Assertions[0].Value)
				}
			},
		},
		{
			name:    "fails for non-existent request",
			setup:   func(db *mockDB) {},
			args:    []string{"nonexistent", "--method", "POST"},
			wantErr: true,
		},
		{
			name: "fails for invalid HTTP method",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--method", "INVALID"},
			wantErr: true,
		},
		{
			name: "fails without request name argument",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://example.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "multiple flags in one command",
			setup: func(db *mockDB) {
				db.requests["id1"] = &types.SavedRequest{
					ID:     "id1",
					Name:   "api",
					URL:    "https://old-api.com",
					Method: "GET",
				}
				db.names["api"] = "id1"
			},
			args:    []string{"api", "--method", "POST", "--url", "https://new-api.com", "--header", "X-Custom: value"},
			wantErr: false,
			checkFn: func(t *testing.T, db *mockDB) {
				req, _ := db.GetRequestByName("api")
				if req.Method != "POST" {
					t.Errorf("expected method POST, got %s", req.Method)
				}
				if req.URL != "https://new-api.com" {
					t.Errorf("expected URL https://new-api.com, got %s", req.URL)
				}
				if len(req.Headers) != 1 || req.Headers[0].Key != "X-Custom" {
					t.Errorf("expected header X-Custom: value")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := EditCommand(db)

			fullArgs := append([]string{"edit"}, tt.args...)

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

func TestEditCommandPersistsScriptsRunIfAndMergesExtractsByName(t *testing.T) {
	db := newMockDB()
	db.requests["id1"] = &types.SavedRequest{
		ID:           "id1",
		Name:         "api",
		URL:          "https://example.com",
		Method:       "GET",
		OutputFormat: "auto",
		Extracts: []types.Extract{
			{Name: "token", Source: "jsonpath:$.oldToken"},
			{Name: "requestId", Source: "header:X-Request-Id"},
		},
	}
	db.names["api"] = "id1"

	cmd := EditCommand(db)
	err := cmd.Run(context.Background(), []string{
		"edit",
		"api",
		"--extract", "token=jsonpath:$.token",
		"--extract", "orderId=jq:$.order.id",
		"--pre-script", "gurl.setVar('tenant', 'acme')",
		"--post-script", "gurl.setVar('seen', 'yes')",
		"--run-if", "tenant != ''",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := db.GetRequestByName("api")
	if err != nil {
		t.Fatalf("load edited request: %v", err)
	}
	if req.PreScript != "gurl.setVar('tenant', 'acme')" {
		t.Fatalf("pre script not persisted: %q", req.PreScript)
	}
	if req.PostScript != "gurl.setVar('seen', 'yes')" {
		t.Fatalf("post script not persisted: %q", req.PostScript)
	}
	if req.RunIf != "tenant != ''" {
		t.Fatalf("run_if not persisted: %q", req.RunIf)
	}
	want := []types.Extract{
		{Name: "token", Source: "jsonpath:$.token"},
		{Name: "requestId", Source: "header:X-Request-Id"},
		{Name: "orderId", Source: "jsonpath:$.order.id"},
	}
	if len(req.Extracts) != len(want) {
		t.Fatalf("expected %d extracts, got %#v", len(want), req.Extracts)
	}
	for i := range want {
		if req.Extracts[i] != want[i] {
			t.Fatalf("extract[%d] mismatch: got %#v want %#v", i, req.Extracts[i], want[i])
		}
	}
}

func TestEditCommandRemoveExtractByName(t *testing.T) {
	db := newMockDB()
	db.requests["id1"] = &types.SavedRequest{
		ID:     "id1",
		Name:   "api",
		URL:    "https://example.com",
		Method: "GET",
		Extracts: []types.Extract{
			{Name: "token", Source: "jsonpath:$.token"},
			{Name: "requestId", Source: "header:X-Request-Id"},
		},
	}
	db.names["api"] = "id1"

	cmd := EditCommand(db)
	err := cmd.Run(context.Background(), []string{"edit", "api", "--remove-extract", "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := db.GetRequestByName("api")
	if err != nil {
		t.Fatalf("load edited request: %v", err)
	}
	if len(req.Extracts) != 1 || req.Extracts[0].Name != "requestId" {
		t.Fatalf("expected only requestId extract to remain, got %#v", req.Extracts)
	}
}

func TestEditCommandRemoveMissingExtractExitsZeroWithMessage(t *testing.T) {
	db := newMockDB()
	db.requests["id1"] = &types.SavedRequest{
		ID:     "id1",
		Name:   "api",
		URL:    "https://example.com",
		Method: "GET",
		Extracts: []types.Extract{
			{Name: "token", Source: "jsonpath:$.token"},
		},
	}
	db.names["api"] = "id1"

	cmd := EditCommand(db)
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"edit", "api", "--remove-extract", "missing"}); err != nil {
			t.Fatalf("missing extract removal should exit 0, got %v", err)
		}
	})
	if !strings.Contains(output, "extract missing not found") {
		t.Fatalf("expected not found message, got %q", output)
	}

	req, err := db.GetRequestByName("api")
	if err != nil {
		t.Fatalf("load edited request: %v", err)
	}
	if len(req.Extracts) != 1 || req.Extracts[0].Name != "token" {
		t.Fatalf("missing removal should not change extracts, got %#v", req.Extracts)
	}
}

func TestEditCommandRejectsMalformedExtractWithExitCode2(t *testing.T) {
	db := newMockDB()
	db.requests["id1"] = &types.SavedRequest{
		ID:     "id1",
		Name:   "api",
		URL:    "https://example.com",
		Method: "GET",
	}
	db.names["api"] = "id1"

	cmd := EditCommand(db)
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	err := cmd.Run(context.Background(), []string{"edit", "api", "--extract", "token=xml:$.token"})
	if err == nil {
		t.Fatal("expected malformed extract error")
	}
	if !strings.Contains(err.Error(), "extract method must be one of jsonpath, header, regex, jq") {
		t.Fatalf("expected clear extract method error, got %v", err)
	}
	exitCoder, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("expected cli.ExitCoder error, got %T: %v", err, err)
	}
	if exitCoder.ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCoder.ExitCode())
	}
}
