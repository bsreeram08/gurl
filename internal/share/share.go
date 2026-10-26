// Package share encrypts gurl export bundles to a recipient's public key and
// decrypts them with a local private key. Recipients are SSH keys (ed25519 or
// RSA), age X25519 keys, or github:<user>, which resolves to the keys GitHub
// publishes at https://github.com/<user>.keys. All crypto is filippo.io/age.
package share

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"filippo.io/age/armor"
	"golang.org/x/crypto/ssh"
)

// Kind marks a decrypted payload as a gurl share bundle.
const Kind = "gurl-share"

// GitHubKeysURL is the base URL for github:<user> lookups. Tests override it.
var GitHubKeysURL = "https://github.com"

const maxKeysResponse = 64 << 10

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Recipient is one resolved public key.
type Recipient struct {
	Source      string // the --to spec it came from
	Fingerprint string
	age.Recipient
}

// ResolveRecipients turns --to specs into age recipients. Keys age can't use
// (ecdsa, sk-*) are skipped and reported in warnings; a spec that yields no
// usable key is an error. github:<user> keys are checked against pins, which
// may be nil.
func ResolveRecipients(ctx context.Context, specs []string, pins *KnownKeys) ([]Recipient, []string, error) {
	if len(specs) == 0 {
		return nil, nil, errors.New("at least one --to recipient is required")
	}
	var out []Recipient
	var warnings []string
	for _, spec := range specs {
		lines, err := keyLines(ctx, spec, pins)
		if err != nil {
			return nil, warnings, err
		}
		found := 0
		for _, line := range lines {
			r, fp, err := parseRecipient(line)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s: skipped key: %v", spec, err))
				continue
			}
			out = append(out, Recipient{Source: spec, Fingerprint: fp, Recipient: r})
			found++
		}
		if found == 0 {
			return nil, warnings, fmt.Errorf("%s: no usable ed25519, RSA or age key found", spec)
		}
	}
	return out, warnings, nil
}

func keyLines(ctx context.Context, spec string, pins *KnownKeys) ([]string, error) {
	switch {
	case strings.HasPrefix(spec, "github:"):
		lines, _, err := GitHubKeys(ctx, spec, pins)
		return lines, err
	case strings.HasPrefix(spec, "ssh-"), strings.HasPrefix(spec, "age1"):
		return []string{spec}, nil
	default:
		data, err := os.ReadFile(spec)
		if err != nil {
			return nil, fmt.Errorf("recipient %q is not a key, github:<user> or readable file: %w", spec, err)
		}
		return splitLines(string(data)), nil
	}
}

// GitHubKeys fetches the public keys GitHub publishes for spec
// ("github:<user>") and checks their fingerprints against pins.
func GitHubKeys(ctx context.Context, spec string, pins *KnownKeys) ([]string, []string, error) {
	user := strings.TrimPrefix(spec, "github:")
	if user == "" || strings.ContainsAny(user, "/?#") {
		return nil, nil, fmt.Errorf("invalid GitHub user in %q", spec)
	}
	lines, err := fetchGitHubKeys(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	var fps []string
	for _, line := range lines {
		if pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line)); err == nil {
			fps = append(fps, ssh.FingerprintSHA256(pub))
		}
	}
	if err := pins.Check(spec, fps); err != nil {
		return nil, nil, err
	}
	return lines, fps, nil
}

func fetchGitHubKeys(ctx context.Context, user string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GitHubKeysURL+"/"+user+".keys", nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching keys for github:%s: %w", user, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching keys for github:%s: HTTP %d", user, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxKeysResponse))
	if err != nil {
		return nil, fmt.Errorf("fetching keys for github:%s: %w", user, err)
	}
	lines := splitLines(string(body))
	if len(lines) == 0 {
		return nil, fmt.Errorf("github:%s has no public SSH keys", user)
	}
	return lines, nil
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}

func parseRecipient(line string) (age.Recipient, string, error) {
	if strings.HasPrefix(line, "age1") {
		r, err := age.ParseX25519Recipient(line)
		return r, line, err
	}
	r, err := agessh.ParseRecipient(line)
	if err != nil {
		return nil, "", err
	}
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return nil, "", err
	}
	return r, ssh.FingerprintSHA256(pub), nil
}

// Seal encrypts payload to the recipients. Armored output is PEM-style text
// that survives copy and paste.
func Seal(w io.Writer, payload []byte, recipients []Recipient, armored bool) error {
	rs := make([]age.Recipient, len(recipients))
	for i, r := range recipients {
		rs[i] = r.Recipient
	}
	dst := w
	var aw io.WriteCloser
	if armored {
		aw = armor.NewWriter(w)
		dst = aw
	}
	enc, err := age.Encrypt(dst, rs...)
	if err != nil {
		return err
	}
	if _, err := enc.Write(payload); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	if aw != nil {
		return aw.Close()
	}
	return nil
}

// DefaultIdentities lists the private keys tried when no --identity is given.
func DefaultIdentities() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".ssh", "id_ed25519"), filepath.Join(home, ".ssh", "id_rsa")}
}

// Open decrypts an armored or binary bundle with the first identity that
// matches. Missing default key files are skipped; passphrase is called for
// passphrase-protected keys and may be nil.
func Open(r io.Reader, identityPaths []string, passphrase func(path string) ([]byte, error)) ([]byte, error) {
	var ids []age.Identity
	var tried []string
	for _, path := range identityPaths {
		id, err := loadIdentity(path, passphrase)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("identity %s: %w", path, err)
		}
		ids = append(ids, id)
		tried = append(tried, path)
	}
	if len(ids) == 0 {
		return nil, errors.New("no private key found: pass --identity <path> or set GURL_SHARE_IDENTITY")
	}

	br := bufio.NewReader(r)
	src := io.Reader(br)
	if head, _ := br.Peek(len(armor.Header)); string(head) == armor.Header {
		src = armor.NewReader(br)
	}
	dec, err := age.Decrypt(src, ids...)
	if err != nil {
		var noMatch *age.NoIdentityMatchError
		if errors.As(err, &noMatch) {
			return nil, fmt.Errorf("bundle is not encrypted to any of your keys (tried %s)", strings.Join(tried, ", "))
		}
		return nil, fmt.Errorf("decrypting bundle: %w", err)
	}
	return io.ReadAll(dec)
}

func loadIdentity(path string, passphrase func(string) ([]byte, error)) (age.Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(data, []byte("AGE-SECRET-KEY-")) {
		ids, err := age.ParseIdentities(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return ids[0], nil
	}
	id, err := agessh.ParseIdentity(data)
	var missing *ssh.PassphraseMissingError
	if !errors.As(err, &missing) {
		return id, err
	}
	if passphrase == nil || missing.PublicKey == nil {
		return nil, errors.New("key is passphrase-protected")
	}
	return agessh.NewEncryptedSSHIdentity(missing.PublicKey, data, func() ([]byte, error) { return passphrase(path) })
}
