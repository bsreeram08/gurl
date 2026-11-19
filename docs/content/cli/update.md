---
title: "apigent update"
description: "Update apigent to the latest release"
---


Replace the running binary with the latest GitHub release for your OS and CPU.

## Synopsis

```bash
apigent update
```

The updater validates the latest release metadata before choosing an asset; releases publish macOS and Linux downloads for amd64 and arm64 (tarballs, raw binaries and `SHA256SUMS`). If the metadata is missing a usable version or platform asset, it exits with an error instead of downloading an invalid URL. If GitHub's release API is temporarily unavailable, it falls back to the public latest-release redirect and can still report `Already up to date!`.

If you installed with Homebrew, use `brew upgrade apigent` instead; for the container image, pull a newer tag.

## Flags

None.

## Examples

### Update in place

```bash
apigent update
```

### Check the installed version

```bash
apigent --version
```

### Homebrew installs

```bash
brew upgrade apigent
```

## Related commands

- [`apigent docs`]({{< relref "/cli/docs" >}}) - Reference for the installed version
