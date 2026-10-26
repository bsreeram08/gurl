package share

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh"
)

// Signatures use OpenSSH's SSHSIG format (PROTOCOL.sshsig), so a bundle's
// signature can also be checked with `ssh-keygen -Y verify -n gurl-share`.
const (
	sigNamespace = "gurl-share"
	sigMagic     = "SSHSIG"
	sigHash      = "sha512"
	sigPEMType   = "SSH SIGNATURE"
)

// LoadSigner reads an SSH private key for signing. passphrase is called for
// passphrase-protected keys and may be nil.
func LoadSigner(path string, passphrase func(path string) ([]byte, error)) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	var missing *ssh.PassphraseMissingError
	if !errors.As(err, &missing) {
		return signer, err
	}
	if passphrase == nil {
		return nil, errors.New("key is passphrase-protected")
	}
	pass, err := passphrase(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKeyWithPassphrase(data, pass)
}

// Sign returns an armored SSHSIG signature of msg.
func Sign(signer ssh.Signer, msg []byte) (string, error) {
	data := signedData(msg)
	var sig *ssh.Signature
	var err error
	if as, ok := signer.(ssh.AlgorithmSigner); ok && signer.PublicKey().Type() == ssh.KeyAlgoRSA {
		sig, err = as.SignWithAlgorithm(rand.Reader, data, ssh.KeyAlgoRSASHA512)
	} else {
		sig, err = signer.Sign(rand.Reader, data)
	}
	if err != nil {
		return "", err
	}
	var blob bytes.Buffer
	blob.WriteString(sigMagic)
	binary.Write(&blob, binary.BigEndian, uint32(1))
	writeString(&blob, signer.PublicKey().Marshal())
	writeString(&blob, []byte(sigNamespace))
	writeString(&blob, nil)
	writeString(&blob, []byte(sigHash))
	writeString(&blob, ssh.Marshal(sig))
	return string(pem.EncodeToMemory(&pem.Block{Type: sigPEMType, Bytes: blob.Bytes()})), nil
}

// Verify checks an armored SSHSIG signature over msg and returns the key that
// made it.
func Verify(armored string, msg []byte) (ssh.PublicKey, error) {
	block, _ := pem.Decode([]byte(armored))
	if block == nil || block.Type != sigPEMType {
		return nil, errors.New("signature is not an SSH SIGNATURE block")
	}
	r := bytes.NewReader(block.Bytes)
	magic := make([]byte, len(sigMagic))
	var version uint32
	if _, err := io.ReadFull(r, magic); err != nil || string(magic) != sigMagic {
		return nil, errors.New("signature has bad magic")
	}
	if err := binary.Read(r, binary.BigEndian, &version); err != nil || version != 1 {
		return nil, errors.New("unsupported signature version")
	}
	fields := make([][]byte, 5)
	for i := range fields {
		f, err := readString(r)
		if err != nil {
			return nil, fmt.Errorf("signature is truncated: %w", err)
		}
		fields[i] = f
	}
	pubBytes, namespace, hashAlg, sigBytes := fields[0], fields[1], fields[3], fields[4]
	if string(namespace) != sigNamespace || string(hashAlg) != sigHash {
		return nil, fmt.Errorf("signature is for namespace %q / hash %q, want %q / %q", namespace, hashAlg, sigNamespace, sigHash)
	}
	pub, err := ssh.ParsePublicKey(pubBytes)
	if err != nil {
		return nil, err
	}
	var sig ssh.Signature
	if err := ssh.Unmarshal(sigBytes, &sig); err != nil {
		return nil, err
	}
	if sig.Format == ssh.KeyAlgoRSA {
		return nil, errors.New("SHA-1 RSA signatures are not accepted")
	}
	if err := pub.Verify(signedData(msg), &sig); err != nil {
		return nil, errors.New("signature does not match the bundle")
	}
	return pub, nil
}

func signedData(msg []byte) []byte {
	h := sha512.Sum512(msg)
	var b bytes.Buffer
	b.WriteString(sigMagic)
	writeString(&b, []byte(sigNamespace))
	writeString(&b, nil)
	writeString(&b, []byte(sigHash))
	writeString(&b, h[:])
	return b.Bytes()
}

func writeString(b *bytes.Buffer, s []byte) {
	binary.Write(b, binary.BigEndian, uint32(len(s)))
	b.Write(s)
}

func readString(r *bytes.Reader) ([]byte, error) {
	var n uint32
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return nil, err
	}
	if int64(n) > int64(r.Len()) {
		return nil, errors.New("length out of range")
	}
	s := make([]byte, n)
	_, err := io.ReadFull(r, s)
	return s, err
}
