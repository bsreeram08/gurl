#!/usr/bin/env bash
set -euo pipefail

VERSION=${1:-}
IMAGE=${IMAGE:-ghcr.io/bsreeram08/gurl}
PLATFORMS=${PLATFORMS:-linux/amd64,linux/arm64}

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.4.0"
    exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format vX.Y.Z"
    exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
    echo "Error: docker is required"
    exit 1
fi

if ! docker buildx inspect >/dev/null 2>&1; then
    docker buildx create --use >/dev/null
fi

echo "Publishing ${IMAGE}:${VERSION} and ${IMAGE}:latest for ${PLATFORMS}"
docker buildx build \
    --platform "$PLATFORMS" \
    --build-arg "VERSION=$VERSION" \
    --label "org.opencontainers.image.source=https://github.com/bsreeram08/gurl" \
    --label "org.opencontainers.image.version=$VERSION" \
    --label "org.opencontainers.image.description=Smart curl saver and API companion for the terminal" \
    --tag "${IMAGE}:${VERSION}" \
    --tag "${IMAGE}:latest" \
    --push \
    .
