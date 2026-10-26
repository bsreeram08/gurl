package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/internal/share"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// shareBundle is the decrypted payload of a share bundle: the native export
// fields plus a kind marker and whether secret values are included.
type shareBundle struct {
	Kind        string                `json:"kind"`
	Version     string                `json:"version"`
	ExportedAt  string                `json:"exported_at"`
	WithSecrets bool                  `json:"with_secrets"`
	From        string                `json:"from,omitempty"`
	Collection  *types.Collection     `json:"collection,omitempty"`
	Requests    []*types.SavedRequest `json:"requests"`
}

// signedEnvelope wraps a bundle with an SSHSIG signature over its exact bytes.
type signedEnvelope struct {
	Kind      string `json:"kind"`
	Bundle    []byte `json:"bundle"`
	Signature string `json:"signature"`
}

const signedKind = share.Kind + "-signed"

var shareStdinIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ShareCommand creates the share command
func ShareCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:      "share",
		Usage:     "Encrypt requests to a recipient's public key (github:<user>, SSH or age key)",
		ArgsUsage: "[request]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Share one request by name"},
			&cli.StringFlag{Name: "collection", Aliases: []string{"c"}, Usage: "Share an entire collection with its variables"},
			&cli.BoolFlag{Name: "all", Aliases: []string{"a"}, Usage: "Share all requests"},
			&cli.StringSliceFlag{Name: "to", Usage: "Recipient: github:<user>, a public key file, or a literal ssh-ed25519/ssh-rsa/age1 key (repeatable)"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output file (default: stdout)"},
			&cli.BoolFlag{Name: "with-secrets", Usage: "Include secret collection variable values"},
			&cli.BoolFlag{Name: "binary", Usage: "Write raw age output instead of ASCII armor"},
			&cli.StringFlag{Name: "sign", Usage: "SSH private key to sign with (default: $GURL_SHARE_IDENTITY, ~/.ssh/id_ed25519, ~/.ssh/id_rsa)"},
			&cli.BoolFlag{Name: "no-sign", Usage: "Send the bundle unsigned"},
			&cli.StringFlag{Name: "from", Usage: "Who you are, e.g. github:<you>; the receiver checks the signature against those keys (or GURL_SHARE_FROM)"},
			&cli.BoolFlag{Name: "update-keys", Usage: "Accept and re-pin a recipient's changed GitHub keys"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			bundle, err := buildShareBundle(db, c)
			if err != nil {
				return err
			}
			bundle.From = c.String("from")
			if bundle.From == "" {
				bundle.From = os.Getenv("GURL_SHARE_FROM")
			}
			pins, err := loadPins(c)
			if err != nil {
				return err
			}
			recipients, warnings, err := share.ResolveRecipients(ctx, c.StringSlice("to"), pins)
			for _, w := range warnings {
				fmt.Fprintln(os.Stderr, "⚠", w)
			}
			if err != nil {
				return err
			}
			for _, r := range recipients {
				fmt.Fprintf(os.Stderr, "→ %s %s\n", r.Source, r.Fingerprint)
			}
			for _, w := range literalSecretWarnings(bundle.Requests) {
				fmt.Fprintln(os.Stderr, "⚠", w)
			}

			payload, err := json.Marshal(bundle)
			if err != nil {
				return fmt.Errorf("failed to marshal share bundle: %w", err)
			}
			if payload, err = signPayload(c, payload); err != nil {
				return err
			}
			var out bytes.Buffer
			if err := share.Seal(&out, payload, recipients, !c.Bool("binary")); err != nil {
				return fmt.Errorf("failed to encrypt bundle: %w", err)
			}

			path := c.String("output")
			if path == "" {
				if _, err := os.Stdout.Write(out.Bytes()); err != nil {
					return err
				}
				return savePins(pins)
			}
			if err := validateOutputPath(path); err != nil {
				return fmt.Errorf("invalid output path: %w", err)
			}
			if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			if err := savePins(pins); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "✓ Shared %d request(s) to %s\n", len(bundle.Requests), path)
			return nil
		},
	}
}

func buildShareBundle(db storage.DB, c *cli.Command) (*shareBundle, error) {
	bundle := &shareBundle{
		Kind:        share.Kind,
		Version:     "1.0",
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		WithSecrets: c.Bool("with-secrets"),
	}
	name := c.String("name")
	if name == "" {
		name = c.Args().First()
	}
	switch {
	case name != "":
		req, err := db.GetRequestByName(name)
		if err != nil {
			return nil, fmt.Errorf("request not found: %w", err)
		}
		bundle.Requests = []*types.SavedRequest{req}
	case c.String("collection") != "":
		collName := c.String("collection")
		requests, err := db.ListRequests(&storage.ListOptions{Collection: collName})
		if err != nil {
			return nil, fmt.Errorf("failed to list collection: %w", err)
		}
		collection, err := db.GetCollectionByName(collName)
		if err != nil && !storage.IsCollectionNotFound(err) {
			return nil, fmt.Errorf("failed to load collection %q: %w", collName, err)
		}
		if collection == nil && len(requests) == 0 {
			return nil, fmt.Errorf("collection %q not found or empty", collName)
		}
		if collection == nil {
			collection = types.NewCollection(collName)
		}
		if bundle.Collection, err = collectionForShare(collection, bundle.WithSecrets); err != nil {
			return nil, err
		}
		bundle.Requests = requests
	case c.Bool("all"):
		requests, err := db.ListRequests(&storage.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list requests: %w", err)
		}
		bundle.Requests = requests
	default:
		return nil, fmt.Errorf("specify a request name, --collection, or --all")
	}
	if bundle.WithSecrets && bundle.Collection == nil {
		return nil, fmt.Errorf("--with-secrets requires --collection")
	}
	return bundle, nil
}

// collectionForShare copies the collection for the bundle. Secret values are
// blanked unless withSecrets; the age envelope protects them when included.
func collectionForShare(source *types.Collection, withSecrets bool) (*types.Collection, error) {
	coll := *source
	coll.Encryption = nil
	coll.Variables = make(map[string]string, len(source.Variables))
	coll.SecretKeys = make(map[string]bool, len(source.SecretKeys))
	for k, v := range source.Variables {
		if source.IsSecret(k) {
			if !withSecrets {
				v = ""
			} else if storage.IsCollectionEncryptedValue(v) {
				return nil, fmt.Errorf("collection %q has locked secrets; run collection unlock first", source.Name)
			}
		}
		coll.Variables[k] = v
	}
	for k, v := range source.SecretKeys {
		coll.SecretKeys[k] = v
	}
	return &coll, nil
}

func literalSecretWarnings(requests []*types.SavedRequest) []string {
	var out []string
	for _, req := range requests {
		for _, h := range req.Headers {
			if redact.SensitiveName(h.Key) && redact.Literal(h.Value) {
				out = append(out, fmt.Sprintf("%s: header %s holds a literal value and will be shared as-is", req.Name, h.Key))
			}
		}
	}
	return out
}

// ReceiveCommand creates the receive command
func ReceiveCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:      "receive",
		Usage:     "Decrypt a share bundle with your private key and import it",
		ArgsUsage: "<path|->",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "identity", Aliases: []string{"i"}, Usage: "Private key to decrypt with (repeatable; default: $GURL_SHARE_IDENTITY, ~/.ssh/id_ed25519, ~/.ssh/id_rsa)"},
			&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "Overwrite existing requests with the same name"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Show the bundle's contents without importing"},
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Import without asking for confirmation"},
			&cli.BoolFlag{Name: "require-signed", Usage: "Refuse bundles that are unsigned or whose signer can't be checked against GitHub"},
			&cli.BoolFlag{Name: "update-keys", Usage: "Accept and re-pin the sender's changed GitHub keys"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			path := c.Args().First()
			if c.Args().Len() > 1 || (path == "-" && argsAfterDash(os.Args)) {
				return fmt.Errorf("unexpected arguments after %q (put flags before the path)", path)
			}
			if path == "" {
				if shareStdinIsTerminal() {
					return fmt.Errorf("bundle path is required (use - for stdin)")
				}
				path = "-"
			}
			var in io.Reader = os.Stdin
			if path != "-" {
				f, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open bundle: %w", err)
				}
				defer f.Close()
				in = f
			}

			identities, err := receiveIdentities(c)
			if err != nil {
				return err
			}
			plain, err := share.Open(in, identities, keyPassphrase(c, path != "-"))
			if err != nil {
				return err
			}
			pins, err := loadPins(c)
			if err != nil {
				return err
			}
			bundle, signer, err := openBundle(ctx, plain, pins)
			if err != nil {
				return err
			}

			sanitizeReceived(bundle)
			printBundlePreview(bundle)
			fmt.Println(signer.line)
			if c.Bool("require-signed") && !signer.verified {
				return fmt.Errorf("bundle sender is not verified (--require-signed)")
			}
			if err := savePins(pins); err != nil {
				return err
			}
			if c.Bool("dry-run") {
				return nil
			}
			if !c.Bool("yes") {
				if path == "-" || !canPrompt(c, shareStdinIsTerminal) {
					return fmt.Errorf("pass --yes to import non-interactively")
				}
				fmt.Print("Import? [y/N] ")
				answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			if bundle.Collection != nil {
				imported, skipped, err := importCollectionRecords(db, bundle.Collection, bundle.Requests, c.Bool("force"))
				if err != nil {
					return err
				}
				fmt.Printf("✓ Imported collection '%s' (%d requests, %d skipped)\n", printable(bundle.Collection.Name), imported, skipped)
				return nil
			}
			imported, skipped := 0, 0
			for _, req := range bundle.Requests {
				if existing, err := db.GetRequestByName(req.Name); err == nil && existing != nil {
					if !c.Bool("force") {
						fmt.Printf("⚠ Skipped '%s' (already exists, use --force to overwrite)\n", printable(req.Name))
						skipped++
						continue
					}
					req.ID = existing.ID
					if err := db.DeleteRequest(existing.ID); err != nil {
						return fmt.Errorf("failed to overwrite request %q: %w", req.Name, err)
					}
				}
				if err := db.SaveRequest(req); err != nil {
					return fmt.Errorf("failed to import request %q: %w", req.Name, err)
				}
				imported++
			}
			fmt.Printf("✓ Imported %d request(s) (%d skipped)\n", imported, skipped)
			return nil
		},
	}
}

// argsAfterDash reports whether anything follows a lone "-" on the command
// line. The flag parser silently drops flags after "-" (`receive - --yes`
// would ignore --yes), and they never reach c.Args().
// ponytail: scans os.Args; a parser option would be cleaner if urfave grows one.
func argsAfterDash(args []string) bool {
	for i, a := range args {
		if a == "-" {
			return i < len(args)-1
		}
	}
	return false
}

func receiveIdentities(c *cli.Command) ([]string, error) {
	explicit := c.StringSlice("identity")
	if env := os.Getenv("GURL_SHARE_IDENTITY"); env != "" {
		explicit = append(explicit, env)
	}
	for _, p := range explicit {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("identity %s: %w", p, err)
		}
	}
	return append(explicit, share.DefaultIdentities()...), nil
}

// sanitizeReceived drops fields a sender must not control. IDs come from the
// sender's database and could collide with (and overwrite) unrelated local
// records; a loose request's collection could attach it to a local collection
// and its secrets. Requests then match local records by name only.
func sanitizeReceived(b *shareBundle) {
	if b.Collection != nil {
		b.Collection.ID = ""
	}
	for _, req := range b.Requests {
		req.ID = ""
		if b.Collection == nil {
			req.Collection = ""
		}
	}
}

// printable replaces control and format characters in sender-controlled
// text, so a bundle can't use terminal escapes, bidi overrides or zero-width
// characters to rewrite or hide the preview (a fake "✓ Signed by" line, a
// cleared script warning, a reordered URL).
func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return '�'
		}
		return r
	}, s)
}

func printBundlePreview(b *shareBundle) {
	if b.Collection != nil {
		fmt.Printf("Collection: %s (%d variables)\n", printable(b.Collection.Name), len(b.Collection.Variables))
	}
	secrets := "not included"
	if b.WithSecrets {
		secrets = "included"
	}
	fmt.Printf("Shared at: %s · secrets %s\n", printable(b.ExportedAt), secrets)
	scripted := 0
	for _, req := range b.Requests {
		var extras []string
		if req.PreScript != "" {
			extras = append(extras, "pre-script")
		}
		if req.PostScript != "" {
			extras = append(extras, "post-script")
		}
		if req.AuthConfig != nil && req.AuthConfig.Type != "" {
			extras = append(extras, "auth: "+printable(req.AuthConfig.Type))
		}
		note := ""
		if len(extras) > 0 {
			note = "  [" + strings.Join(extras, ", ") + "]"
		}
		if req.PreScript != "" || req.PostScript != "" {
			scripted++
		}
		fmt.Printf("  %-7s %-24s %s%s\n", printable(req.Method), printable(req.Name), printable(req.URL), note)
	}
	if scripted > 0 {
		fmt.Printf("⚠ %d request(s) carry scripts; they run on your machine when you run the request. Check them with 'gurl show' first.\n", scripted)
	}
}

func loadPins(c *cli.Command) (*share.KnownKeys, error) {
	pins, err := share.LoadKnownKeys(share.DefaultKnownKeysPath())
	if err != nil {
		return nil, err
	}
	pins.Update = c.Bool("update-keys")
	return pins, nil
}

func savePins(pins *share.KnownKeys) error {
	for _, n := range pins.Notes {
		fmt.Fprintln(os.Stderr, "•", n)
	}
	return pins.Save()
}

// keyPassphrase prompts for a private key's passphrase when a terminal is
// available.
func keyPassphrase(c *cli.Command, stdinFree bool) func(string) ([]byte, error) {
	return func(keyPath string) ([]byte, error) {
		if !stdinFree || !canPrompt(c, shareStdinIsTerminal) {
			return nil, fmt.Errorf("key %s is passphrase-protected and no terminal is available", keyPath)
		}
		p, err := readCollectionPassphrase(fmt.Sprintf("Passphrase for %s: ", keyPath))
		return []byte(p), err
	}
}

// signPayload wraps payload in a signed envelope. An explicit --sign key must
// work; with the default keys, a missing or unusable key sends unsigned.
func signPayload(c *cli.Command, payload []byte) ([]byte, error) {
	if c.Bool("no-sign") {
		return payload, nil
	}
	explicit := c.String("sign")
	if explicit == "" {
		explicit = os.Getenv("GURL_SHARE_IDENTITY")
	}
	paths := share.DefaultIdentities()
	if explicit != "" {
		paths = []string{explicit}
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil && explicit == "" {
			continue
		}
		signer, err := share.LoadSigner(p, keyPassphrase(c, true))
		if err != nil {
			if explicit != "" {
				return nil, fmt.Errorf("signing key %s: %w", p, err)
			}
			fmt.Fprintf(os.Stderr, "⚠ not signing: %s: %v (pass --sign <key> or --no-sign)\n", p, err)
			return payload, nil
		}
		sig, err := share.Sign(signer, payload)
		if err != nil {
			return nil, fmt.Errorf("signing bundle: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✎ signed with %s\n", ssh.FingerprintSHA256(signer.PublicKey()))
		return json.Marshal(signedEnvelope{Kind: signedKind, Bundle: payload, Signature: sig})
	}
	fmt.Fprintln(os.Stderr, "⚠ not signing: no SSH key found (pass --sign <key> or --no-sign)")
	return payload, nil
}

type signerStatus struct {
	line     string
	verified bool
}

// openBundle unwraps a decrypted payload, verifying its signature if signed.
// A signature that fails, or a key that isn't one of the claimed sender's,
// is an error; an unsigned bundle is reported, not refused.
func openBundle(ctx context.Context, plain []byte, pins *share.KnownKeys) (*shareBundle, signerStatus, error) {
	notBundle := fmt.Errorf("not a gurl share bundle")
	var env signedEnvelope
	if err := json.Unmarshal(plain, &env); err != nil {
		return nil, signerStatus{}, notBundle
	}
	var pub ssh.PublicKey
	if env.Kind == signedKind {
		var err error
		if pub, err = share.Verify(env.Signature, env.Bundle); err != nil {
			return nil, signerStatus{}, fmt.Errorf("bundle signature is invalid: %w", err)
		}
		plain = env.Bundle
	}

	var bundle shareBundle
	if err := json.Unmarshal(plain, &bundle); err != nil || bundle.Kind != share.Kind {
		return nil, signerStatus{}, notBundle
	}
	if bundle.Version != "1.0" {
		return nil, signerStatus{}, storage.ErrUnsupportedCollectionExportVersion(printable(bundle.Version))
	}
	bundle.From = printable(bundle.From)
	if pub == nil {
		return &bundle, signerStatus{line: "⚠ Unsigned: the sender can't be verified"}, nil
	}

	fp := ssh.FingerprintSHA256(pub)
	if !strings.HasPrefix(bundle.From, "github:") {
		line := "Signed by key " + fp
		if bundle.From != "" {
			line += fmt.Sprintf(", claims %q (unchecked)", bundle.From)
		}
		return &bundle, signerStatus{line: line}, nil
	}
	_, fps, err := share.GitHubKeys(ctx, bundle.From, pins)
	switch {
	case err != nil:
		return &bundle, signerStatus{line: fmt.Sprintf("Signed by key %s, claims %s (couldn't check: %v)", fp, bundle.From, err)}, nil
	case !slices.Contains(fps, fp):
		return nil, signerStatus{}, fmt.Errorf("bundle claims to be from %s but is signed by %s, which is not one of their keys", bundle.From, fp)
	}
	return &bundle, signerStatus{line: fmt.Sprintf("✓ Signed by %s (%s)", bundle.From, fp), verified: true}, nil
}
