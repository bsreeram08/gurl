---
title: "gurl update"
description: "Update gurl to latest version"
---

# gurl update

Update gurl to the latest version available.

## Usage

```bash
gurl update
```

## Description

The `update` command checks for and installs the latest version of gurl. It handles downloading and replacing the current binary.

The updater validates the latest release metadata before choosing an asset. Current releases publish complete macOS and Linux downloads for amd64 and arm64, including tarballs, raw binaries, and `SHA256SUMS`.

## Flags

None.

## Examples

### Update gurl

```bash
gurl update
```

Checks for and installs the latest version.

If the release metadata is missing a usable version or platform asset, the command exits with an error instead of downloading an invalid URL.

## See also

- [Gurl GitHub Repository](https://github.com/bsreeram08/gurl) - Release notes and version information
