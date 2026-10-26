package share

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestSignVerify(t *testing.T) {
	dir := t.TempDir()
	keyPath, pubLine := newKey(t, dir, "bob")
	signer, err := LoadSigner(keyPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte(`{"kind":"gurl-share"}`)
	sig, err := Sign(signer, msg)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := Verify(sig, msg)
	if err != nil {
		t.Fatal(err)
	}
	if string(ssh.MarshalAuthorizedKey(pub)) != pubLine+"\n" {
		t.Fatalf("verified key %q, want %q", ssh.MarshalAuthorizedKey(pub), pubLine)
	}
	if _, err := Verify(sig, append(msg, ' ')); err == nil {
		t.Fatal("tampered message verified")
	}

	// The signature is plain SSHSIG: ssh-keygen accepts it too.
	if _, err := exec.LookPath("ssh-keygen"); err == nil {
		allowed := filepath.Join(dir, "allowed")
		os.WriteFile(allowed, []byte("bob@x "+pubLine+"\n"), 0600)
		sigFile := filepath.Join(dir, "msg.sig")
		os.WriteFile(sigFile, []byte(sig), 0600)
		cmd := exec.Command("ssh-keygen", "-Y", "verify", "-f", allowed, "-I", "bob@x", "-n", "gurl-share", "-s", sigFile)
		cmd.Stdin = strings.NewReader(string(msg))
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("ssh-keygen -Y verify: %v\n%s", err, out)
		}
	}
}

func TestSignVerifyRSA(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(key, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "id_rsa")
	os.WriteFile(path, pem.EncodeToMemory(block), 0600)
	signer, err := LoadSigner(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := Sign(signer, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(sig, []byte("x")); err != nil {
		t.Fatal(err)
	}
}

func TestKnownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gurl", "known_keys.json")
	k, _ := LoadKnownKeys(path)
	if err := k.Check("github:alice", []string{"SHA256:b", "SHA256:a"}); err != nil {
		t.Fatal(err)
	}
	if err := k.Save(); err != nil {
		t.Fatal(err)
	}

	k, _ = LoadKnownKeys(path)
	if err := k.Check("github:alice", []string{"SHA256:a", "SHA256:b"}); err != nil {
		t.Fatalf("same keys, different order: %v", err)
	}
	err := k.Check("github:alice", []string{"SHA256:a", "SHA256:evil"})
	if err == nil || !strings.Contains(err.Error(), "added: SHA256:evil; removed: SHA256:b") {
		t.Fatalf("err = %v", err)
	}
	k.Update = true
	if err := k.Check("github:alice", []string{"SHA256:a", "SHA256:evil"}); err != nil {
		t.Fatal(err)
	}
	var nilPins *KnownKeys
	if err := nilPins.Check("github:x", nil); err != nil {
		t.Fatal(err)
	}
}
