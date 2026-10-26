package commands

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/share"
	"github.com/sreeram/gurl/pkg/types"
	"golang.org/x/crypto/ssh"
)

func TestCollectionForShareStripsSecrets(t *testing.T) {
	coll := types.NewCollection("payments")
	coll.Variables["base"] = "https://api.example.com"
	coll.SetSecretVariable("token", "s3cret")

	stripped, err := collectionForShare(coll, false)
	if err != nil {
		t.Fatal(err)
	}
	if stripped.Variables["token"] != "" || !stripped.IsSecret("token") || stripped.Variables["base"] != "https://api.example.com" {
		t.Fatalf("stripped = %+v", stripped)
	}
	if coll.Variables["token"] != "s3cret" {
		t.Fatal("source collection was modified")
	}

	full, err := collectionForShare(coll, true)
	if err != nil {
		t.Fatal(err)
	}
	if full.Variables["token"] != "s3cret" {
		t.Fatalf("with secrets = %+v", full)
	}
}

func TestOpenBundleSignatures(t *testing.T) {
	dir := t.TempDir()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(priv, "")
	keyPath := filepath.Join(dir, "bob")
	os.WriteFile(keyPath, pem.EncodeToMemory(block), 0600)
	sshPub, _ := ssh.NewPublicKey(pub)
	signer, err := share.LoadSigner(keyPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bob.keys" {
			w.Write(ssh.MarshalAuthorizedKey(sshPub))
			return
		}
		w.Write([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGb3m9q6ZtF5Jx1dW9oqNbXnPq2QqP3Wl1oJ4r6s3n5L\n"))
	}))
	defer srv.Close()
	old := share.GitHubKeysURL
	share.GitHubKeysURL = srv.URL
	defer func() { share.GitHubKeysURL = old }()

	seal := func(from string) []byte {
		payload, _ := json.Marshal(shareBundle{Kind: share.Kind, Version: "1.0", From: from})
		sig, err := share.Sign(signer, payload)
		if err != nil {
			t.Fatal(err)
		}
		out, _ := json.Marshal(signedEnvelope{Kind: signedKind, Bundle: payload, Signature: sig})
		return out
	}
	ctx := context.Background()

	if _, st, err := openBundle(ctx, seal("github:bob"), nil); err != nil || !st.verified {
		t.Fatalf("github:bob: verified=%v err=%v", st.verified, err)
	}
	if _, _, err := openBundle(ctx, seal("github:mallory"), nil); err == nil || !strings.Contains(err.Error(), "not one of their keys") {
		t.Fatalf("forged claim: err = %v", err)
	}
	unsigned, _ := json.Marshal(shareBundle{Kind: share.Kind, Version: "1.0"})
	if _, st, err := openBundle(ctx, unsigned, nil); err != nil || st.verified {
		t.Fatalf("unsigned: verified=%v err=%v", st.verified, err)
	}
	tampered := seal("github:bob")
	tampered = bytes.Replace(tampered, []byte(`"bundle":"`), []byte(`"bundle":"e30`), 1)
	if _, _, err := openBundle(ctx, tampered, nil); err == nil {
		t.Fatal("tampered bundle accepted")
	}
}

func TestArgsAfterDash(t *testing.T) {
	cases := map[string]bool{
		"gurl receive - --yes": true,
		"gurl receive --yes -": false,
		"gurl receive f.age":   false,
	}
	for line, want := range cases {
		if got := argsAfterDash(strings.Fields(line)); got != want {
			t.Errorf("%q: got %v, want %v", line, got, want)
		}
	}
}

func TestSanitizeReceived(t *testing.T) {
	loose := &shareBundle{Requests: []*types.SavedRequest{{ID: "victim-id", Name: "x", Collection: "payments"}}}
	sanitizeReceived(loose)
	if r := loose.Requests[0]; r.ID != "" || r.Collection != "" {
		t.Fatalf("loose request kept sender fields: %+v", r)
	}

	coll := types.NewCollection("shared")
	coll.ID = "victim-coll"
	withColl := &shareBundle{Collection: coll, Requests: []*types.SavedRequest{{ID: "victim-id", Name: "x", Collection: "shared"}}}
	sanitizeReceived(withColl)
	if withColl.Collection.ID != "" || withColl.Requests[0].ID != "" || withColl.Requests[0].Collection != "shared" {
		t.Fatalf("collection bundle: %+v %+v", withColl.Collection, withColl.Requests[0])
	}
}

func TestPrintableStripsTerminalEscapes(t *testing.T) {
	got := printable("get-user\x1b[2K\r\x1b[32m✓ Signed by github:bob\u009b0m\x9b")
	if strings.ContainsRune(got, '\x1b') || strings.ContainsRune(got, '\r') || strings.ContainsRune(got, '\u009b') || strings.Contains(got, "\x9b") {
		t.Fatalf("control characters kept: %q", got)
	}
	if printable("https://api.example.com/ü") != "https://api.example.com/ü" {
		t.Fatal("printable text changed")
	}
}

func TestOpenBundleSanitizesFrom(t *testing.T) {
	plain, _ := json.Marshal(shareBundle{Kind: share.Kind, Version: "1.0", From: "bob\x1b[2K"})
	b, _, err := openBundle(context.Background(), plain, nil)
	if err != nil || strings.Contains(b.From, "\x1b") {
		t.Fatalf("from = %q, err = %v", b.From, err)
	}
}
