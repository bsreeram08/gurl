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
