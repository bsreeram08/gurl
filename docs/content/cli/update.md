---
title: "gurl update"
description: "Update gurl to the latest release"
---


Replace the running binary with the latest GitHub release for your OS and CPU.

## Synopsis

```bash
gurl update
```

The updater validates the latest release metadata before choosing an asset; releases publish macOS and Linux downloads for amd64 and arm64 (tarballs, raw binaries and `SHA256SUMS`). If the metadata is missing a usable version or platform asset, it exits with an error instead of downloading an invalid URL. If GitHub's release API is temporarily unavailable, it falls back to the public latest-release redirect and can still report `Already up to date!`.

If you installed with Homebrew, use `brew upgrade gurl` instead; for the container image, pull a newer tag.

## Flags

None.

## Examples

### Update in place

```bash
gurl update
```

### Check the installed version

```bash
gurl --version
```

### Homebrew installs

```bash
brew upgrade gurl
```

## Related commands

- [`gurl docs`]({{< relref "/cli/docs" >}}) - Reference for the installed version
