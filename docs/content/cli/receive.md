---
title: "gurl receive"
description: "Decrypt a share bundle and import it"
---


Decrypt a bundle made by [`gurl share`]({{< relref "/cli/share" >}}) with your private key, preview its contents and import them.

## Synopsis

```bash
gurl receive <path|-> [--identity <key>...] [--dry-run] [--yes] [--force] [--require-signed]
```

Pass `-` to read the bundle from stdin. Armored and binary bundles are both accepted.

## Keys

`receive` tries, in order: each `--identity`, `$GURL_SHARE_IDENTITY`, `~/.ssh/id_ed25519` and `~/.ssh/id_rsa`. ed25519 and RSA SSH keys and age identity files (`AGE-SECRET-KEY-...`) work. A passphrase-protected key prompts for its passphrase; `ssh-agent` can't be used, because decryption needs the private key itself.

## Who sent it

The preview ends with the sender check:

| Line | Meaning |
|------|---------|
| `✓ Signed by github:bob (SHA256:...)` | The signature is valid and the key is one GitHub publishes for `bob` |
| `Signed by key SHA256:...` | The signature is valid; the sender didn't claim a GitHub account |
| `Signed by key SHA256:..., claims github:bob (couldn't check: ...)` | Valid signature, but GitHub couldn't be reached or bob's keys changed since they were pinned |
| `⚠ Unsigned: the sender can't be verified` | No signature |

A broken signature, or a bundle that claims `github:bob` but is signed by a key that isn't bob's, is refused outright. `--require-signed` also refuses anything that isn't verified against GitHub. Sender keys are pinned in `~/.config/gurl/known_keys.json` the same way [`share`]({{< relref "/cli/share" >}}) pins recipients; `--update-keys` accepts a changed set.

## What gets imported

Before importing, `receive` prints the collection, each request's method, name and URL (marking pre/post scripts and auth), and whether secrets are included, then asks to confirm. Nothing runs on receive, but a request's scripts run when you later run it, so check them with `gurl show` first. Requests are matched to local ones by name only: the sender's internal IDs are dropped, and a loose request (not part of a shared collection) is imported without a collection, so it can't join one of yours and read its secrets. Existing requests and collections are kept unless you pass `--force`. Imported secret values are encrypted at rest with your local key, like any other collection secret.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--identity` | `-i` | see Keys | Private key to decrypt with (repeatable) |
| `--dry-run` | | `false` | Show the bundle's contents without importing |
| `--yes` | `-y` | `false` | Import without asking (required when reading from stdin or with `--no-input`) |
| `--force` | `-f` | `false` | Overwrite existing requests and collections with the same name |
| `--require-signed` | | `false` | Refuse bundles whose sender isn't verified against GitHub |
| `--update-keys` | | `false` | Accept and re-pin the sender's changed GitHub keys |

## Examples

### Preview, then import

```bash
gurl receive payments.gurl.age --dry-run
gurl receive payments.gurl.age
```

### Use a key with a custom name

```bash
gurl receive payments.gurl.age -i ~/.ssh/work_ed25519
```

### Import a pasted bundle

```bash
pbpaste | gurl receive --yes -
```

## Related commands

- [`gurl share`]({{< relref "/cli/share" >}}) - Encrypt requests for someone
- [`gurl import`]({{< relref "/cli/import" >}}) - Import unencrypted exports and other formats
