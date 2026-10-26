---
title: "gurl share"
description: "Encrypt requests to a recipient's public key"
---


Encrypt one request, one collection or every request to another person's public key. Only that person's private key can open the result, so the file is safe to send over Slack, email or a gist. The recipient imports it with [`gurl receive`]({{< relref "/cli/receive" >}}).

## Synopsis

```bash
gurl share (<request> | --name <request> | --collection <name> | --all) --to <recipient> [--to <recipient>...] [--from github:<you>] [--output <file>] [--with-secrets] [--binary]
```

## Recipients

| `--to` value | Resolved by |
|--------------|-------------|
| `github:<user>` | The public SSH keys GitHub publishes at `https://github.com/<user>.keys` |
| a file path | A `.pub` or `authorized_keys`-style file, one key per line |
| `ssh-ed25519 AAAA...` / `ssh-rsa AAAA...` | The literal public key |
| `age1...` | A native [age](https://age-encryption.org) X25519 recipient |

The bundle is encrypted to every usable key of every recipient, so any of their machines can open it. `ecdsa` and security-key (`sk-`) SSH keys can't be used for encryption; they are skipped with a warning. Each recipient key's fingerprint is printed to stderr so you can check it.

## Signing

Bundles are signed with your SSH key so the receiver can tell who sent them. `share` signs with `--sign <key>`, else `$GURL_SHARE_IDENTITY`, else `~/.ssh/id_ed25519` or `~/.ssh/id_rsa`. If none of the default keys can be used, the bundle goes out unsigned with a warning; `--no-sign` skips signing on purpose.

Pass `--from github:<you>` (or set `GURL_SHARE_FROM`) and the receiver checks that the signing key is one of the keys GitHub publishes for you. Signatures use OpenSSH's SSHSIG format with the namespace `gurl-share`, so `ssh-keygen -Y verify` can check them too.

## Key pinning

The first time you share with `github:<user>`, gurl records the fingerprints of their keys in `~/.config/gurl/known_keys.json`. If the set changes later (a new laptop, or a taken-over account), `share` stops and lists the added and removed keys. Re-run with `--update-keys` once you've confirmed the change with the person.

## Secrets

Secret collection variables are sent with an empty value and keep their secret flag; the recipient fills in their own. `--with-secrets` includes the values, protected by the bundle's encryption. Request headers, bodies and auth params are sent as stored: a literal value under a sensitive header such as `Authorization` is shared as-is, and `share` warns about it. Use `{{variables}}` to keep tokens out of shared requests.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | none | Share one request by name (or pass it as the argument) |
| `--collection` | `-c` | none | Share an entire collection with its variables |
| `--all` | `-a` | `false` | Share all requests |
| `--to` | | none | Recipient (repeatable, required) |
| `--output` | `-o` | stdout | Output file, written with mode `0600` |
| `--with-secrets` | | `false` | Include secret collection variable values (requires `--collection`) |
| `--binary` | | `false` | Write raw age output instead of ASCII armor |
| `--sign` | | see Signing | SSH private key to sign with |
| `--no-sign` | | `false` | Send the bundle unsigned |
| `--from` | | `$GURL_SHARE_FROM` | Who you are, e.g. `github:<you>`; checked by the receiver |
| `--update-keys` | | `false` | Accept and re-pin a recipient's changed GitHub keys |

## Examples

### Send one request to a GitHub user

```bash
gurl share get-user --to github:alice --from github:bob
```

The ASCII-armored output can be pasted straight into a chat message.

### Share a collection with two people

```bash
gurl share -c payments-api --to github:alice --to github:bob -o payments.gurl.age
```

### Include secret values

```bash
gurl share -c payments-api --with-secrets --to ~/keys/alice.pub -o payments.gurl.age
```

## How it works

The bundle is the same JSON that [`gurl export`]({{< relref "/cli/export" >}}) writes, plus a `kind: gurl-share` marker, encrypted with [age](https://age-encryption.org) (`filippo.io/age`). Any standard `age` CLI can decrypt it with the recipient's key. A signed bundle wraps that JSON with an SSHSIG signature over its exact bytes before encryption, so the sender's identity is visible only to the recipient.

## Related commands

- [`gurl receive`]({{< relref "/cli/receive" >}}) - Decrypt and import a share bundle
- [`gurl export`]({{< relref "/cli/export" >}}) - Unencrypted export, or passphrase-protected collection secrets
