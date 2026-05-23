package runner

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

func TestCollectionRunCommandPersistWritesDirtyVarsAndMasksSecrets(t *testing.T) {
	db := newMockDB()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord_123"}}`))
	}))
	defer server.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "create-order",
		Name:       "create order",
		Method:     "GET",
		URL:        "{{baseUrl}}/orders",
		Collection: "orders",
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
		PostScript: `gurl.setVar("sessionToken", "secret-456")`,
	})

	envStorage := newTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	beta.SetSecretVariable("sessionToken", "old-secret")
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := CollectionRunCommand(db, envStorage)
	output := captureStdoutFile(t, func() {
		if err := cmd.Run(context.Background(), []string{"run", "orders", "--env", "beta", "--persist"}); err != nil {
			t.Fatalf("collection run failed: %v", err)
		}
	})

	reloaded, err := envStorage.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("failed to reload env: %v", err)
	}
	if reloaded.Variables["orderId"] != "ord_123" {
		t.Fatalf("expected orderId to persist, got %+v", reloaded.Variables)
	}
	if reloaded.Variables["sessionToken"] != "secret-456" {
		t.Fatalf("expected sessionToken to persist, got %+v", reloaded.Variables)
	}
	if !reloaded.IsSecret("sessionToken") || reloaded.IsSecret("orderId") {
		t.Fatalf("expected existing secret metadata only, got %+v", reloaded.SecretKeys)
	}
	if !strings.Contains(output, "Persisted 2 variables to environment \"beta\"") || !strings.Contains(output, "orderId = ord_123") || !strings.Contains(output, "sessionToken = *****") {
		t.Fatalf("expected masked persist summary, got output:\n%s", output)
	}
	if strings.Contains(output, "secret-456") {
		t.Fatalf("persist summary leaked secret value:\n%s", output)
	}
}

func TestCollectionRunCommandPersistDryRunFailsFast(t *testing.T) {
	db := newMockDB()
	envStorage := newTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", "http://127.0.0.1")
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := CollectionRunCommand(db, envStorage)
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	err := cmd.Run(context.Background(), []string{"run", "orders", "--env", "beta", "--persist", "--dry-run"})
	if err == nil {
		t.Fatal("expected persist/dry-run incompatibility error")
	}
	if !strings.Contains(err.Error(), "--persist and --dry-run cannot be used together") {
		t.Fatalf("expected clear incompatible flags error, got %v", err)
	}
	reloaded, err := envStorage.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("failed to reload env: %v", err)
	}
	if _, ok := reloaded.Variables["orderId"]; ok {
		t.Fatalf("expected no env mutation on incompatible flags, got %+v", reloaded.Variables)
	}
}

func TestCollectionRunCommandDryRunPrintsDiagnostics(t *testing.T) {
	db := newMockDB()
	serverCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalls++
		_, _ = w.Write([]byte(`{"should":"not happen"}`))
	}))
	defer server.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "create-order",
		Name:       "create order",
		Method:     "POST",
		URL:        "{{baseUrl}}/orders/{{tenant}}",
		Collection: "orders",
		SortOrder:  1,
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
		},
	})
	db.SaveRequest(&types.SavedRequest{
		ID:         "pay-order",
		Name:       "pay order",
		Method:     "GET",
		URL:        "{{baseUrl}}/payments/{{orderId}}/{{missingVar}}",
		Collection: "orders",
		SortOrder:  2,
	})

	envStorage := newTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("baseUrl", server.URL)
	beta.SetVariable("tenant", "acme")
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	cmd := CollectionRunCommand(db, envStorage)
	output := captureStdoutFile(t, func() {
		if err := cmd.Run(context.Background(), []string{"run", "orders", "--env", "beta", "--dry-run"}); err != nil {
			t.Fatalf("collection dry-run failed: %v", err)
		}
	})

	if serverCalls != 0 {
		t.Fatalf("expected dry-run to make zero HTTP requests, got %d", serverCalls)
	}
	for _, want := range []string{
		`Dry run: collection "orders"`,
		`Requests: 2`,
		`Environment: beta`,
		`1. create order`,
		`POST ` + server.URL + `/orders/acme`,
		`orderId ← jsonpath:$.data.orderId`,
		`2. pay order`,
		`GET ` + server.URL + `/payments/{{orderId}}/{{missingVar}}`,
		`orderId from step 1 extraction`,
		`warning: unresolved {{missingVar}}`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected dry-run output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestCollectionRunUsesCollectionVarsBetweenEnvAndCLI(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	envStorage := newTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("ROOT", server.URL)
	beta.SetVariable("BASE_URL", "https://wrong.example.com")
	beta.SetVariable("VERSION", "env-version")
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	collection := types.NewCollection("payments")
	collection.SetVariable("BASE_URL", "{{ROOT}}/collection")
	collection.SetVariable("VERSION", "collection-version")
	if err := db.SaveCollection(collection); err != nil {
		t.Fatalf("failed to save collection: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "list-payments",
		Name:       "list payments",
		Method:     "GET",
		URL:        "{{BASE_URL}}/{{VERSION}}",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	results, err := NewRunner(db, envStorage).Run(context.Background(), RunConfig{
		CollectionName: "payments",
		Environment:    "beta",
		Vars:           map[string]string{"VERSION": "cli-version"},
	})
	if err != nil {
		t.Fatalf("collection run failed: %v", err)
	}
	if len(results) != 1 || results[0].Passed != 1 {
		t.Fatalf("expected request to pass, got %+v", results)
	}
	if gotPath != "/collection/cli-version" {
		t.Fatalf("expected env-expanded collection BASE_URL and CLI VERSION, got path %q", gotPath)
	}
}

func TestCollectionRunCommandPersistRoutesDirtyVarsByOrigin(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"orderId":"ord-new","envToken":"env-new","newVar":"new-value"}}`))
	}))
	defer server.Close()

	envStorage := newTestEnvStorage(t)
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("envToken", "env-old")
	if err := envStorage.SaveEnv(beta); err != nil {
		t.Fatalf("failed to save env: %v", err)
	}

	collection := types.NewCollection("orders")
	collection.SetVariable("orderId", "ord-old")
	if err := db.SaveCollection(collection); err != nil {
		t.Fatalf("failed to save collection: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "create-order",
		Name:       "create order",
		Method:     "GET",
		URL:        server.URL,
		Collection: "orders",
		Extracts: []types.Extract{
			{Name: "orderId", Source: "jsonpath:$.data.orderId"},
			{Name: "envToken", Source: "jsonpath:$.data.envToken"},
			{Name: "newVar", Source: "jsonpath:$.data.newVar"},
		},
	}); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	cmd := CollectionRunCommand(db, envStorage)
	output := captureStdoutFile(t, func() {
		if err := cmd.Run(context.Background(), []string{"run", "orders", "--env", "beta", "--persist"}); err != nil {
			t.Fatalf("collection run failed: %v", err)
		}
	})

	reloadedEnv, err := envStorage.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("failed to reload env: %v", err)
	}
	if reloadedEnv.Variables["envToken"] != "env-new" {
		t.Fatalf("expected env-origin var persisted to env, got %+v", reloadedEnv.Variables)
	}
	if _, ok := reloadedEnv.Variables["orderId"]; ok {
		t.Fatalf("collection-origin var leaked into env: %+v", reloadedEnv.Variables)
	}

	reloadedCollection, err := db.GetCollectionByName("orders")
	if err != nil {
		t.Fatalf("failed to reload collection: %v", err)
	}
	if reloadedCollection.Variables["orderId"] != "ord-new" || reloadedCollection.Variables["newVar"] != "new-value" {
		t.Fatalf("expected collection vars to be persisted, got %+v", reloadedCollection.Variables)
	}
	if strings.Contains(output, "Persisted 1 variable to environment \"beta\"") == false ||
		strings.Contains(output, "Persisted 2 variables to collection \"orders\"") == false {
		t.Fatalf("expected persist summaries for env and collection, got:\n%s", output)
	}
}

func newTestEnvStorage(t *testing.T) *env.EnvStorage {
	t.Helper()
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "env.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open env db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return env.NewEnvStorage(db)
}

func captureStdoutFile(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	return string(data)
}
