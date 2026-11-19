#!/usr/bin/env bash
#
# valetkey installer
# Usage: curl -sL https://raw.githubusercontent.com/bsreeram08/valetkey/master/scripts/install.sh | bash
#

set -euo pipefail

# Configuration
REPO="bsreeram08/valetkey"
BINARY="valetkey"
# Pre-rename name; an existing gurl from this project becomes a link to valetkey.
LEGACY_BINARY="gurl"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
VERSION="${VERSION:-latest}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() { echo -e "${GREEN}[valetkey]${NC} $1"; }
warn() { echo -e "${YELLOW}[valetkey]${NC} $1"; }
error() { echo -e "${RED}[valetkey]${NC} $1" >&2; }

validate_release_tag() {
    [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

resolve_release_tag() {
    local version="${1:-}"
    local release_tag

    version="${version#v}"
    release_tag="v${version}"

    if ! validate_release_tag "$release_tag"; then
        error "Invalid version: ${1:-<empty>}"
        return 1
    fi

    echo "$release_tag"
}

latest_release_tag_from_api() {
    local body release_tag

    if ! body=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest"); then
        return 1
    fi

    release_tag=$(printf '%s\n' "$body" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)
    release_tag="${release_tag//$'\r'/}"

    if ! validate_release_tag "$release_tag"; then
        return 1
    fi

    echo "$release_tag"
}

latest_release_tag_from_redirect() {
    local location release_tag

    location=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")
    release_tag="${location%%[?#]*}"
    release_tag="${release_tag#*/releases/tag/}"

    if [[ "$release_tag" == "$location" ]]; then
        return 1
    fi

    release_tag="${release_tag%%/*}"

    if ! validate_release_tag "$release_tag"; then
        return 1
    fi

    echo "$release_tag"
}

# Detect OS and architecture
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)          error "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        *)              error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
}

# Get latest version tag
get_latest_version() {
    local release_tag

    if release_tag=$(latest_release_tag_from_api); then
        echo "$release_tag"
        return 0
    fi

    if release_tag=$(latest_release_tag_from_redirect); then
        echo "$release_tag"
        return 0
    fi

    error "Unable to determine latest release version from GitHub"
    return 1
}

# Download and install
install() {
    local os arch release_tag filename url
    os=$(detect_os)
    arch=$(detect_arch)
    
    if [[ "${VERSION}" == "latest" ]]; then
        release_tag=$(get_latest_version) || return 1
    else
        release_tag=$(resolve_release_tag "${VERSION}") || return 1
    fi
    
    filename="${BINARY}-${os}-${arch}"
    if [[ "${os}" == "windows" ]]; then
        filename="${filename}.exe"
    fi
    
    url="https://github.com/${REPO}/releases/download/${release_tag}/${filename}"
    
    log "Downloading ${filename} (${release_tag})..."
    
    # Create install directory
    mkdir -p "${INSTALL_DIR}"
    
    # Download
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "${url}" -o "${INSTALL_DIR}/${BINARY}" || return 1
    elif command -v wget >/dev/null 2>&1; then
        wget -q "${url}" -O "${INSTALL_DIR}/${BINARY}" || return 1
    else
        error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
    
    # Make executable
    chmod +x "${INSTALL_DIR}/${BINARY}"
    
    log "Installed ${BINARY} ${release_tag} to ${INSTALL_DIR}/${BINARY}"
    link_legacy_binary
    
    # Check if install dir is in PATH
    if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
        warn "${INSTALL_DIR} is not in your PATH."
        warn "Add this to your ~/.bashrc or ~/.zshrc:"
        warn "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
}

# link_legacy_binary points an existing gurl from this project at valetkey,
# so scripts that call gurl run the new binary. A gurl that is some other
# tool is left alone.
link_legacy_binary() {
    local legacy="${INSTALL_DIR}/${LEGACY_BINARY}"
    if [[ -e "$legacy" ]] && "$legacy" --version 2>/dev/null | grep -Eq "^(gurl|valetkey) version"; then
        ln -sf "${BINARY}" "$legacy"
        log "Linked ${legacy} -> ${BINARY} (the old command name keeps working)"
    fi
}

# Verify installation
verify() {
    log "Verifying installation..."
    if "${INSTALL_DIR}/${BINARY}" --version >/dev/null 2>&1; then
        log "Installation successful!"
        "${INSTALL_DIR}/${BINARY}" --version
    else
        error "Installation verification failed."
        exit 1
    fi
}

# Main
main() {
    log "valetkey installer"
    log "Repository: https://github.com/${REPO}"
    echo
    
    install
    verify
    
    echo
    log "Run '${BINARY} --help' to get started!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
