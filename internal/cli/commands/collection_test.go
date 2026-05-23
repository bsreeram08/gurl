package commands

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/project"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

func TestCollectionListCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "lists collections",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:         "req-1",
					Name:       "request1",
					URL:        "https://example.com",
					Collection: "api",
					UpdatedAt:  1700000000,
				}
				db.names["request1"] = "req-1"
				db.requests["req-2"] = &types.SavedRequest{
					ID:         "req-2",
					Name:       "request2",
					URL:        "https://example.com",
					Collection: "web",
					UpdatedAt:  1700000001,
				}
				db.names["request2"] = "req-2"
			},
			args:    []string{"list"},
			wantErr: false,
		},
		{
			name:    "shows empty message when no collections",
			setup:   func(db *mockDB) {},
			args:    []string{"list"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := CollectionCommand(db, &env.EnvStorage{})
			fullArgs := append([]string{"collection"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCollectionShowCommand(t *testing.T) {
	db := newMockDB()
	db.requests["req-1"] = &types.SavedRequest{
		ID:         "req-1",
		Name:       "get-user",
		URL:        "https://example.com/users/1",
		Method:     "GET",
		Collection: "api",
		UpdatedAt:  1700000000,
	}
	db.names["get-user"] = "req-1"

	cmd := CollectionCommand(db, &env.EnvStorage{})
	output := captureStdout(t, func() {
		err := cmd.Run(context.Background(), []string{"collection", "show", "api"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Collection: api") {
		t.Errorf("expected collection name in output, got %q", output)
	}
	if !strings.Contains(output, "get-user") || !strings.Contains(output, "https://example.com/users/1") {
		t.Errorf("expected request details in output, got %q", output)
	}
}

func TestCollectionRunCommandIncludesAssertBailFlag(t *testing.T) {
	db := newMockDB()
	cmd := CollectionCommand(db, &env.EnvStorage{})

	var runCmd *cli.Command
	for _, subcommand := range cmd.Commands {
		if subcommand.Name == "run" {
			runCmd = subcommand
			break
		}
	}
	if runCmd == nil {
		t.Fatal("expected collection run subcommand")
	}

	if !commandHasFlag(runCmd, "assert-bail") {
		t.Fatalf("expected collection run command to expose --assert-bail flag")
	}
}

func TestCollectionMigrateCommandExportsDBCollectionToFiles(t *testing.T) {
	base := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "gurl.db"))
	if err := base.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer base.Close()
	if err := base.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "legacy request",
		URL:        "https://example.com",
		Method:     "GET",
		Collection: "legacy",
	}); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	proj, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	fileStore := storage.NewFileStore(proj)
	db := storage.NewProjectDB(base, fileStore)
	cmd := CollectionCommand(db, &env.EnvStorage{})

	if err := cmd.Run(context.Background(), []string{"collection", "migrate", "legacy"}); err != nil {
		t.Fatalf("migrate command failed: %v", err)
	}

	req, err := fileStore.GetRequest("req-1")
	if err != nil {
		t.Fatalf("expected migrated request file: %v", err)
	}
	if req.URL != "https://example.com" || req.Collection != "legacy" {
		t.Fatalf("unexpected migrated request: %+v", req)
	}
}

func TestCollectionExportImportRoundTripEncryptsSecrets(t *testing.T) {
	sourceBase := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "source.db"))
	if err := sourceBase.Open(); err != nil {
		t.Fatalf("failed to open source db: %v", err)
	}
	defer sourceBase.Close()
	sourceProject, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("source Init failed: %v", err)
	}
	sourceDB := storage.NewProjectDB(sourceBase, storage.NewFileStore(sourceProject))
	collection := types.NewCollection("payments")
	collection.SetSecretVariable("API_KEY", "secret-token")
	if err := sourceDB.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	if err := sourceDB.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "list payments",
		URL:        "https://example.com/payments",
		Method:     "GET",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	exportPath := filepath.Join(t.TempDir(), "payments.gurl")
	exportCmd := CollectionCommand(sourceDB, &env.EnvStorage{})
	if err := exportCmd.Run(context.Background(), []string{"collection", "export", "payments", "--passphrase", "team-pass", "--output", exportPath}); err != nil {
		t.Fatalf("collection export failed: %v", err)
	}
	exported, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read export: %v", err)
	}
	if strings.Contains(string(exported), "secret-token") {
		t.Fatal("export should not contain plaintext secret")
	}

	targetBase := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "target.db"))
	if err := targetBase.Open(); err != nil {
		t.Fatalf("failed to open target db: %v", err)
	}
	defer targetBase.Close()
	targetProject, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("target Init failed: %v", err)
	}
	targetStore := storage.NewFileStore(targetProject)
	targetDB := storage.NewProjectDB(targetBase, targetStore)
	importCmd := CollectionCommand(targetDB, &env.EnvStorage{})
	if err := importCmd.Run(context.Background(), []string{"collection", "import", exportPath, "--passphrase", "team-pass"}); err != nil {
		t.Fatalf("collection import failed: %v", err)
	}

	imported, err := targetDB.GetCollectionByName("payments")
	if err != nil {
		t.Fatalf("GetCollectionByName failed: %v", err)
	}
	if imported.Variables["API_KEY"] != "secret-token" {
		t.Fatalf("expected decrypted imported secret, got %q", imported.Variables["API_KEY"])
	}
	collectionPath, err := targetStore.CollectionPath("payments")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	rawCollection, err := os.ReadFile(filepath.Join(collectionPath, "collection.json"))
	if err != nil {
		t.Fatalf("failed to read imported collection: %v", err)
	}
	if strings.Contains(string(rawCollection), "secret-token") {
		t.Fatal("imported collection should be re-encrypted locally")
	}
}

func TestCollectionImportForceReusesExistingRequestID(t *testing.T) {
	sourceBase := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "source.db"))
	if err := sourceBase.Open(); err != nil {
		t.Fatalf("failed to open source db: %v", err)
	}
	defer sourceBase.Close()
	sourceProject, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("source Init failed: %v", err)
	}
	sourceDB := storage.NewProjectDB(sourceBase, storage.NewFileStore(sourceProject))
	if err := sourceDB.SaveCollection(types.NewCollection("payments")); err != nil {
		t.Fatalf("source SaveCollection failed: %v", err)
	}
	if err := sourceDB.SaveRequest(&types.SavedRequest{
		ID:         "exported-id",
		Name:       "list payments",
		URL:        "https://new.example.com/payments",
		Method:     "GET",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("source SaveRequest failed: %v", err)
	}

	exportPath := filepath.Join(t.TempDir(), "payments.gurl")
	exportCmd := CollectionCommand(sourceDB, &env.EnvStorage{})
	if err := exportCmd.Run(context.Background(), []string{"collection", "export", "payments", "--output", exportPath}); err != nil {
		t.Fatalf("collection export failed: %v", err)
	}

	targetBase := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "target.db"))
	if err := targetBase.Open(); err != nil {
		t.Fatalf("failed to open target db: %v", err)
	}
	defer targetBase.Close()
	targetProject, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("target Init failed: %v", err)
	}
	targetStore := storage.NewFileStore(targetProject)
	targetDB := storage.NewProjectDB(targetBase, targetStore)
	if err := targetDB.SaveCollection(types.NewCollection("payments")); err != nil {
		t.Fatalf("target SaveCollection failed: %v", err)
	}
	if err := targetDB.SaveRequest(&types.SavedRequest{
		ID:         "existing-id",
		Name:       "list payments",
		URL:        "https://old.example.com/payments",
		Method:     "GET",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("target SaveRequest failed: %v", err)
	}

	importCmd := CollectionCommand(targetDB, &env.EnvStorage{})
	if err := importCmd.Run(context.Background(), []string{"collection", "import", exportPath, "--force"}); err != nil {
		t.Fatalf("collection import --force failed: %v", err)
	}

	loaded, err := targetDB.GetRequestByName("list payments")
	if err != nil {
		t.Fatalf("GetRequestByName failed: %v", err)
	}
	if loaded.ID != "existing-id" {
		t.Fatalf("expected forced import to reuse existing ID, got %q", loaded.ID)
	}
	if loaded.URL != "https://new.example.com/payments" {
		t.Fatalf("expected request to be overwritten, got %s", loaded.URL)
	}
	requests, err := targetDB.ListRequests(&storage.ListOptions{Collection: "payments"})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected one request after forced overwrite, got %+v", requests)
	}
	if _, err := targetDB.GetRequest("exported-id"); err == nil {
		t.Fatal("expected exported ID not to create a second request")
	}
}

func TestCollectionImportForceReplacesLockedCollection(t *testing.T) {
	sourceBase := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "source.db"))
	if err := sourceBase.Open(); err != nil {
		t.Fatalf("failed to open source db: %v", err)
	}
	defer sourceBase.Close()
	sourceProject, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("source Init failed: %v", err)
	}
	sourceDB := storage.NewProjectDB(sourceBase, storage.NewFileStore(sourceProject))
	sourceCollection := types.NewCollection("payments")
	sourceCollection.SetSecretVariable("API_KEY", "secret-token")
	if err := sourceDB.SaveCollection(sourceCollection); err != nil {
		t.Fatalf("source SaveCollection failed: %v", err)
	}
	if err := sourceDB.SaveRequest(&types.SavedRequest{
		ID:         "exported-id",
		Name:       "list payments",
		URL:        "https://new.example.com/payments",
		Method:     "GET",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("source SaveRequest failed: %v", err)
	}

	exportPath := filepath.Join(t.TempDir(), "payments.gurl")
	exportCmd := CollectionCommand(sourceDB, &env.EnvStorage{})
	if err := exportCmd.Run(context.Background(), []string{"collection", "export", "payments", "--output", exportPath, "--passphrase", "team-pass"}); err != nil {
		t.Fatalf("collection export failed: %v", err)
	}

	targetBase := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "target.db"))
	if err := targetBase.Open(); err != nil {
		t.Fatalf("failed to open target db: %v", err)
	}
	defer targetBase.Close()
	targetProject, err := project.Init(t.TempDir())
	if err != nil {
		t.Fatalf("target Init failed: %v", err)
	}
	targetStore := storage.NewFileStore(targetProject)
	targetDB := storage.NewProjectDB(targetBase, targetStore)
	targetCollection := types.NewCollection("payments")
	targetCollection.SetSecretVariable("API_KEY", "old-token")
	if err := targetDB.SaveCollection(targetCollection); err != nil {
		t.Fatalf("target SaveCollection failed: %v", err)
	}
	if err := targetDB.SaveRequest(&types.SavedRequest{
		ID:         "existing-id",
		Name:       "list payments",
		URL:        "https://old.example.com/payments",
		Method:     "GET",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("target SaveRequest failed: %v", err)
	}
	collectionPath, err := targetStore.CollectionPath("payments")
	if err != nil {
		t.Fatalf("CollectionPath failed: %v", err)
	}
	if err := os.Remove(filepath.Join(collectionPath, "collection.key")); err != nil {
		t.Fatalf("failed to remove collection key: %v", err)
	}
	if _, err := targetDB.GetCollectionByName("payments"); !storage.IsCollectionLocked(err) {
		t.Fatalf("expected locked target collection, got %v", err)
	}

	importCmd := CollectionCommand(targetDB, &env.EnvStorage{})
	err = importCmd.Run(context.Background(), []string{"collection", "import", exportPath, "--passphrase", "team-pass"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected locked collection to be treated as existing without --force, got %v", err)
	}
	if err := importCmd.Run(context.Background(), []string{"collection", "import", exportPath, "--force", "--passphrase", "team-pass"}); err != nil {
		t.Fatalf("collection import --force failed: %v", err)
	}

	loadedCollection, err := targetDB.GetCollectionByName("payments")
	if err != nil {
		t.Fatalf("GetCollectionByName failed: %v", err)
	}
	if loadedCollection.ID != targetCollection.ID {
		t.Fatalf("expected locked collection ID to be reused, got %q", loadedCollection.ID)
	}
	if loadedCollection.Variables["API_KEY"] != "secret-token" {
		t.Fatalf("expected imported secret, got %q", loadedCollection.Variables["API_KEY"])
	}
	loadedRequest, err := targetDB.GetRequestByName("list payments")
	if err != nil {
		t.Fatalf("GetRequestByName failed: %v", err)
	}
	if loadedRequest.ID != "existing-id" {
		t.Fatalf("expected existing request ID to be reused, got %q", loadedRequest.ID)
	}
	if loadedRequest.URL != "https://new.example.com/payments" {
		t.Fatalf("expected request to be overwritten, got %s", loadedRequest.URL)
	}
	rawData, err := os.ReadFile(filepath.Join(collectionPath, "collection.json"))
	if err != nil {
		t.Fatalf("failed to read stored collection: %v", err)
	}
	if strings.Contains(string(rawData), "secret-token") {
		t.Fatal("imported secret should be re-encrypted locally")
	}
}

func TestCollectionPassphraseUsesEnvFallback(t *testing.T) {
	t.Setenv("GURL_IMPORT_PASSPHRASE", "env-pass")

	var got string
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "passphrase"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			got = collectionPassphrase(c)
			return nil
		},
	}
	if err := cmd.Run(context.Background(), []string{"test"}); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if got != "env-pass" {
		t.Fatalf("expected env passphrase fallback, got %q", got)
	}
}

func commandHasFlag(cmd *cli.Command, name string) bool {
	for _, flag := range cmd.Flags {
		named, ok := flag.(interface{ Names() []string })
		if !ok {
			continue
		}
		for _, flagName := range named.Names() {
			if flagName == name {
				return true
			}
		}
	}
	return false
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	return buf.String()
}

func TestCollectionAddCommand(t *testing.T) {
	db := newMockDB()
	cmd := CollectionCommand(db, &env.EnvStorage{})

	err := cmd.Run(context.Background(), []string{"collection", "add", "newcollection"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCollectionCreateAndShowVariables(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	cmd := CollectionCommand(db, env.NewEnvStorage(db))
	if err := cmd.Run(context.Background(), []string{
		"collection", "create", "payments",
		"--var", "BASE_URL=https://api.example.com",
		"--secret", "API_KEY=secret-value",
	}); err != nil {
		t.Fatalf("collection create failed: %v", err)
	}

	collection, err := db.GetCollectionByName("payments")
	if err != nil {
		t.Fatalf("failed to load collection: %v", err)
	}
	if collection.Variables["BASE_URL"] != "https://api.example.com" || collection.Variables["API_KEY"] != "secret-value" || !collection.IsSecret("API_KEY") {
		t.Fatalf("collection variables mismatch: %+v", collection)
	}

	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"collection", "show", "payments"}); err != nil {
			t.Fatalf("collection show failed: %v", err)
		}
	})
	if !strings.Contains(output, "BASE_URL = https://api.example.com") || !strings.Contains(output, "API_KEY = *****") {
		t.Fatalf("expected variables with masked secret, got:\n%s", output)
	}
	if strings.Contains(output, "secret-value") {
		t.Fatalf("collection show leaked secret value:\n%s", output)
	}
}

func TestCollectionRemoveCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "removes collection",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:         "req-1",
					Name:       "request1",
					URL:        "https://example.com",
					Collection: "api",
				}
				db.names["request1"] = "req-1"
			},
			args:    []string{"remove", "api"},
			wantErr: false,
		},
		{
			name:    "fails when collection name is missing",
			setup:   func(db *mockDB) {},
			args:    []string{"remove"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := CollectionCommand(db, &env.EnvStorage{})
			fullArgs := append([]string{"collection"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCollectionDeleteForceCascadesRequestsAndCollection(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	collection := types.NewCollection("payments")
	collection.SetSecretVariable("API_KEY", "secret")
	if err := db.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	for _, req := range []*types.SavedRequest{
		{ID: "req-1", Name: "list payments", URL: "https://example.com/payments", Collection: "payments"},
		{ID: "req-2", Name: "get payment", URL: "https://example.com/payments/1", Collection: "payments"},
	} {
		if err := db.SaveRequest(req); err != nil {
			t.Fatalf("SaveRequest failed: %v", err)
		}
	}

	cmd := CollectionCommand(db, &env.EnvStorage{})
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"collection", "delete", "payments", "--force"}); err != nil {
			t.Fatalf("collection delete failed: %v", err)
		}
	})

	if !strings.Contains(output, "Deleted collection 'payments' (2 requests deleted)") {
		t.Fatalf("expected delete summary, got %q", output)
	}
	if _, err := db.GetCollectionByName("payments"); err == nil {
		t.Fatal("expected collection record to be deleted")
	}
	requests, err := db.ListRequests(&storage.ListOptions{Collection: "payments"})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("expected collection requests to be deleted, got %+v", requests)
	}
	if _, err := db.GetRequestByName("list payments"); err == nil {
		t.Fatal("expected request to be deleted")
	}
}

func TestCollectionDeleteRequiresConfirmationWhenInteractive(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	collection := types.NewCollection("payments")
	collection.SetSecretVariable("API_KEY", "secret")
	if err := db.SaveCollection(collection); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "list payments",
		URL:        "https://example.com/payments",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	oldInteractive := collectionDeleteIsInteractive
	collectionDeleteIsInteractive = func() bool { return true }
	defer func() { collectionDeleteIsInteractive = oldInteractive }()
	oldStdin := os.Stdin
	stdin, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	if _, err := writer.WriteString("y\n"); err != nil {
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

	cmd := CollectionCommand(db, &env.EnvStorage{})
	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"collection", "delete", "payments"}); err != nil {
			t.Fatalf("collection delete failed: %v", err)
		}
	})

	if !strings.Contains(output, "Collection \"payments\" has 1 request and 1 secret. Delete all? [y/N]") {
		t.Fatalf("expected confirmation prompt, got %q", output)
	}
	if _, err := db.GetRequestByName("list payments"); err == nil {
		t.Fatal("expected confirmed delete to remove request")
	}
}

func TestCollectionDeleteErrorsNonInteractiveWithoutForce(t *testing.T) {
	db := storage.NewLMDBWithPath(filepath.Join(t.TempDir(), "collections.db"))
	if err := db.Open(); err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	if err := db.SaveCollection(types.NewCollection("payments")); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}
	if err := db.SaveRequest(&types.SavedRequest{
		ID:         "req-1",
		Name:       "list payments",
		URL:        "https://example.com/payments",
		Collection: "payments",
	}); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	cmd := CollectionCommand(db, &env.EnvStorage{})
	err := cmd.Run(context.Background(), []string{"collection", "delete", "payments"})
	if err == nil || !strings.Contains(err.Error(), "use --force") {
		t.Fatalf("expected non-interactive confirmation error, got %v", err)
	}
	if _, err := db.GetRequestByName("list payments"); err != nil {
		t.Fatalf("request should remain after refused delete: %v", err)
	}
}

func TestCollectionRenameCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockDB)
		args    []string
		wantErr bool
	}{
		{
			name: "renames collection",
			setup: func(db *mockDB) {
				db.requests["req-1"] = &types.SavedRequest{
					ID:         "req-1",
					Name:       "request1",
					URL:        "https://example.com",
					Collection: "oldname",
				}
				db.names["request1"] = "req-1"
			},
			args:    []string{"rename", "oldname", "newname"},
			wantErr: false,
		},
		{
			name:    "fails when old name is missing",
			setup:   func(db *mockDB) {},
			args:    []string{"rename"},
			wantErr: true,
		},
		{
			name:    "fails when new name is missing",
			setup:   func(db *mockDB) {},
			args:    []string{"rename", "oldname"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockDB()
			if tt.setup != nil {
				tt.setup(db)
			}

			cmd := CollectionCommand(db, &env.EnvStorage{})
			fullArgs := append([]string{"collection"}, tt.args...)

			err := cmd.Run(context.Background(), fullArgs)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
