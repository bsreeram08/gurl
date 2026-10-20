package runner

import (
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/internal/storage/memstore"
	"github.com/sreeram/gurl/pkg/types"
)

func TestPersistRefusesPlaceholderValues(t *testing.T) {
	_, err := PersistDirtyVariables(nil, nil, "beta", "", map[string]string{"orderId": "{{API_KEY}}"}, nil)
	if err == nil || !strings.Contains(err.Error(), "orderId") {
		t.Fatalf("expected refusal, got %v", err)
	}
}

func TestPersistDirtyVariablesRoutesByOrigin(t *testing.T) {
	envs := env.NewMemoryStore()
	beta := env.NewEnvironment("beta", "")
	beta.SetVariable("token", "old")
	if err := envs.SaveEnv(beta); err != nil {
		t.Fatalf("SaveEnv: %v", err)
	}
	db := memstore.New()
	orders := types.NewCollection("orders")
	orders.SetSecretVariable("apiKey", "old-key")
	if err := db.SaveCollection(orders); err != nil {
		t.Fatalf("SaveCollection: %v", err)
	}

	vars := map[string]string{
		"token":     "new-token", // environment origin
		"orderId":   "ord_1",     // collection origin
		"apiKey":    "new-key",   // collection origin, existing secret
		"fromCLI":   "cli",       // CLI origin in a collection-backed run
		"paymentId": "pay_1",     // retargeted to another collection
	}
	origins := map[string]resolve.Origin{
		"token":     resolve.OriginEnvironment,
		"orderId":   resolve.OriginCollection,
		"apiKey":    resolve.OriginCollection,
		"fromCLI":   resolve.OriginCLI,
		"paymentId": resolve.OriginCollection,
	}
	targets := map[string]string{"paymentId": "payments"}

	summary, err := PersistDirtyVariablesWithCollectionTargets(envs, db, "beta", "orders", vars, origins, targets)
	if err != nil {
		t.Fatalf("PersistDirtyVariablesWithCollectionTargets: %v", err)
	}

	gotEnv, err := envs.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("GetEnvByName: %v", err)
	}
	if gotEnv.Variables["token"] != "new-token" {
		t.Errorf("env token = %q, want new-token", gotEnv.Variables["token"])
	}
	for _, key := range []string{"orderId", "apiKey", "fromCLI", "paymentId"} {
		if _, ok := gotEnv.Variables[key]; ok {
			t.Errorf("collection-bound %q leaked into env: %+v", key, gotEnv.Variables)
		}
	}

	gotOrders, err := db.GetCollectionByName("orders")
	if err != nil {
		t.Fatalf("GetCollectionByName(orders): %v", err)
	}
	if gotOrders.Variables["orderId"] != "ord_1" || gotOrders.Variables["fromCLI"] != "cli" || gotOrders.Variables["apiKey"] != "new-key" {
		t.Errorf("orders vars = %+v", gotOrders.Variables)
	}
	if !gotOrders.IsSecret("apiKey") || gotOrders.IsSecret("orderId") {
		t.Errorf("orders secret keys = %+v, want only apiKey", gotOrders.SecretKeys)
	}

	// A collection that does not exist yet is created.
	gotPayments, err := db.GetCollectionByName("payments")
	if err != nil {
		t.Fatalf("GetCollectionByName(payments): %v", err)
	}
	if gotPayments.Variables["paymentId"] != "pay_1" || len(gotPayments.Variables) != 1 {
		t.Errorf("payments vars = %+v, want only paymentId", gotPayments.Variables)
	}

	if summary.EnvironmentVars["token"] != "new-token" || len(summary.EnvironmentVars) != 1 {
		t.Errorf("summary env vars = %+v", summary.EnvironmentVars)
	}
	if len(summary.Collections) != 2 || summary.Collections[0].Name != "orders" || summary.Collections[1].Name != "payments" {
		t.Errorf("summary collections = %+v", summary.Collections)
	}
}

func TestPersistDirtyVariablesWithoutCollectionGoesToEnv(t *testing.T) {
	envs := env.NewMemoryStore()
	if err := envs.SaveEnv(env.NewEnvironment("beta", "")); err != nil {
		t.Fatalf("SaveEnv: %v", err)
	}
	db := memstore.New()

	_, err := PersistDirtyVariables(envs, db, "beta", "", map[string]string{"orderId": "ord_1"},
		map[string]resolve.Origin{"orderId": resolve.OriginCollection})
	if err != nil {
		t.Fatalf("PersistDirtyVariables: %v", err)
	}
	got, err := envs.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("GetEnvByName: %v", err)
	}
	if got.Variables["orderId"] != "ord_1" {
		t.Errorf("env vars = %+v, want orderId persisted", got.Variables)
	}
	if collections, _ := db.ListCollections(); len(collections) != 0 {
		t.Errorf("expected no collections, got %+v", collections)
	}
}

func TestPersistDirtyVariablesReturnsLockedCollectionError(t *testing.T) {
	db := memstore.New()
	orders := types.NewCollection("orders")
	orders.SetSecretVariable("apiKey", "keep-me")
	if err := db.SaveCollection(orders); err != nil {
		t.Fatalf("SaveCollection: %v", err)
	}
	db.Lock("orders", "pass")

	_, err := PersistDirtyVariables(nil, db, "", "orders", map[string]string{"orderId": "ord_1"},
		map[string]resolve.Origin{"orderId": resolve.OriginCollection})
	if !storage.IsCollectionLocked(err) {
		t.Fatalf("expected locked collection error, got %v", err)
	}

	// The locked collection was not replaced by a fresh one.
	raw, err := db.GetRawCollectionByName("orders")
	if err != nil {
		t.Fatalf("GetRawCollectionByName: %v", err)
	}
	if raw.ID != orders.ID || raw.Variables["apiKey"] != "keep-me" {
		t.Fatalf("locked collection was overwritten: %+v", raw)
	}
	if _, ok := raw.Variables["orderId"]; ok {
		t.Fatalf("orderId persisted into locked collection: %+v", raw.Variables)
	}
}
