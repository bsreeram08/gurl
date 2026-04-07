#!/usr/bin/env bash
#
# Gurl Installer
# GURL = Gurl's Universal Request Library
# Usage: curl -sL https://raw.githubusercontent.com/bsreeram08/gurl/master/scripts/install.sh | bash
#

set -euo pipefail

# Configuration
REPO="bsreeram08/gurl"
INSTALL_DIR="${HOME}/.local/bin"
VERSION="${VERSION:-latest}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() { echo -e "${GREEN}[gurl]${NC} $1"; }
warn() { echo -e "${YELLOW}[gurl]${NC} $1"; }
error() { echo -e "${RED}[gurl]${NC} $1" >&2; }

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
    local version
    version=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    echo "${version#v}"
}

# Download and install
install() {
    local os arch version filename url
    os=$(detect_os)
    arch=$(detect_arch)
    
    if [[ "${VERSION}" == "latest" ]]; then
        version=$(get_latest_version)
    else
        version="${VERSION#v}"
    fi
    
    filename="gurl-${os}-${arch}"
    if [[ "${os}" == "windows" ]]; then
        filename="${filename}.exe"
    fi
    
    url="https://github.com/${REPO}/releases/download/v${version}/${filename}"
    
    log "Downloading ${filename} (v${version})..."
    
    # Create install directory
    mkdir -p "${INSTALL_DIR}"
    
    # Download
    if command -v curl >/dev/null 2>&1; then
        curl -sL "${url}" -o "${INSTALL_DIR}/gurl"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "${url}" -O "${INSTALL_DIR}/gurl"
    else
        error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
    
    # Make executable
    chmod +x "${INSTALL_DIR}/gurl"
    
    log "Installed gurl v${version} to ${INSTALL_DIR}/gurl"
    
    # Check if install dir is in PATH
    if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
        warn "${INSTALL_DIR} is not in your PATH."
        warn "Add this to your ~/.bashrc or ~/.zshrc:"
        warn "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
}

# Verify installation
verify() {
    log "Verifying installation..."
    if "${INSTALL_DIR}/gurl" --version >/dev/null 2>&1; then
        log "Installation successful!"
        "${INSTALL_DIR}/gurl" --version
    else
        error "Installation verification failed."
        exit 1
    fi
}

# Main
main() {
    log "Gurl Installer - GURL = Gurl's Universal Request Library"
    log "Repository: https://github.com/${REPO}"
    echo
    
    install
    verify
    
    echo
    log "Run 'gurl --help' to get started!"
}

main "$@"
