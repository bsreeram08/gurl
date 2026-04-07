#!/usr/bin/env bash
set -euo pipefail

VERSION=${1:-}
if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

# Validate version format
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format vX.Y.Z"
    exit 1
fi

echo "Creating release $VERSION..."

# Update version in code
# sed -i "s/var version = \".*\"/var version = \"$VERSION\"/" internal/version.go

# Create tag
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

# CI will build and create GitHub release
echo "Tagged and pushed. CI will build and create release."
