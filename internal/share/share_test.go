package share

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"golang.org/x/crypto/ssh"
)

// newKey writes an ed25519 private key to dir and returns its path and
// authorized_keys line.
func newKey(t *testing.T, dir, name string) (string, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return path, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
}

func TestSealOpenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	keyPath, pubLine := newKey(t, dir, "alice")
	recipients, _, err := ResolveRecipients(context.Background(), []string{pubLine}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(recipients[0].Fingerprint, "SHA256:") {
		t.Fatalf("fingerprint = %q", recipients[0].Fingerprint)
	}
	for _, armored := range []bool{true, false} {
		var buf bytes.Buffer
		if err := Seal(&buf, []byte(`{"kind":"gurl-share"}`), recipients, armored); err != nil {
			t.Fatal(err)
		}
		if armored != strings.HasPrefix(buf.String(), "-----BEGIN AGE ENCRYPTED FILE-----") {
			t.Fatalf("armored=%v but output starts %q", armored, buf.String()[:20])
		}
		got, err := Open(&buf, []string{filepath.Join(dir, "missing"), keyPath}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != `{"kind":"gurl-share"}` {
			t.Fatalf("got %q", got)
		}
	}
}

func TestOpenWrongKey(t *testing.T) {
	dir := t.TempDir()
	_, alicePub := newKey(t, dir, "alice")
	bobKey, _ := newKey(t, dir, "bob")
	recipients, _, _ := ResolveRecipients(context.Background(), []string{alicePub}, nil)
	var buf bytes.Buffer
	if err := Seal(&buf, []byte("x"), recipients, true); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(&buf, []string{bobKey}, nil); err == nil || !strings.Contains(err.Error(), "not encrypted to any of your keys") {
		t.Fatalf("err = %v", err)
	}
}

func TestGitHubRecipientSkipsUnsupportedKeys(t *testing.T) {
	_, pubLine := newKey(t, t.TempDir(), "alice")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/alice.keys" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=\n%s\n", pubLine)
	}))
	defer srv.Close()
	old := GitHubKeysURL
	GitHubKeysURL = srv.URL
	defer func() { GitHubKeysURL = old }()

	recipients, warnings, err := ResolveRecipients(context.Background(), []string{"github:alice"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(recipients) != 1 || len(warnings) != 1 {
		t.Fatalf("recipients=%d warnings=%v", len(recipients), warnings)
	}
	if _, _, err := ResolveRecipients(context.Background(), []string{"github:nobody"}, nil); err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestAgeIdentityFileWithComments(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "key.txt")
	os.WriteFile(path, []byte("# created: 2026-10-26\n# public key: "+id.Recipient().String()+"\n"+id.String()+"\n"), 0600)
	recipients, _, err := ResolveRecipients(context.Background(), []string{id.Recipient().String()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Seal(&buf, []byte("hi"), recipients, true); err != nil {
		t.Fatal(err)
	}
	if got, err := Open(&buf, []string{path}, nil); err != nil || string(got) != "hi" {
		t.Fatalf("got %q, err %v", got, err)
	}
}
