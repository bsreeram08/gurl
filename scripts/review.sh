#!/usr/bin/env bash
#
# Hound Code Review - Internal AI Reviewer
# Usage: ./scripts/review.sh [files...]
#        ./scripts/review.sh --all
#

set -euo pipefail

REPO="${GITHUB_REPOSITORY:-sreeram/gurl}"
BRANCH="${GITHUB_REF_NAME:-master}"

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[HOUND]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }
info() { echo -e "${BLUE}[INFO]${NC} $1"; }

echo ""
log "═══════════════════════════════════════════════════════"
log "     HOUND CODE REVIEW - Bloodhound for Bugs"
log "═══════════════════════════════════════════════════════"
echo ""

# Check if we have files to review
if [[ $# -eq 0 ]]; then
    warn "No files specified. Use: $0 [files...] or $0 --all"
    exit 0
fi

# Determine files to review
if [[ "$1" == "--all" ]]; then
    info "Reviewing all Go files..."
    FILES=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | head -20)
elif [[ "$1" == "--changed" ]]; then
    info "Reviewing changed files..."
    FILES=$(git diff --name-only HEAD~1 --diff-filter=AM | grep "\.go$" || true)
else
    FILES="$@"
fi

if [[ -z "$FILES" ]]; then
    info "No Go files to review."
    exit 0
fi

echo ""
info "Files to review:"
echo "$FILES" | while read -r f; do echo "  - $f"; done
echo ""

# Run review checks
REVIEW_PASSED=true

for FILE in $FILES; do
    [[ -f "$FILE" ]] || continue
    
    echo ""
    warn "Checking: $FILE"
    
    # Security: Shell command injection
    if grep -q 'exec\.Command.*+.*"' "$FILE" 2>/dev/null; then
        error "POSSIBLE SHELL INJECTION: String concatenation in exec.Command"
        REVIEW_PASSED=false
    fi
    
    # Security: exec.Command with shell=true pattern
    if grep -q 'Shell.*true' "$FILE" 2>/dev/null; then
        error "SHELL INJECTION RISK: Shell=true in exec"
        REVIEW_PASSED=false
    fi
    
    # Architecture: Map iteration for logic
    if grep -q 'for.*range.*map' "$FILE" 2>/dev/null && grep -q 'if.*==' "$FILE"; then
        warn "MAP ITERATION: Verify iteration order doesn't affect logic"
    fi
    
    # Reliability: Error swallowing
    if grep -q '_.*=.*err' "$FILE" 2>/dev/null; then
        COUNT=$(grep -c '_.*=.*err' "$FILE" 2>/dev/null || echo 0)
        warn "ERROR SWALLOWING: $COUNT ignored errors"
    fi
    
    # Reliability: os.Exit in main
    if grep -q 'os\.Exit' "$FILE" 2>/dev/null && [[ "$FILE" == *"main.go" ]]; then
        warn "DEFERRAL: os.Exit bypasses deferred cleanup"
    fi
    
    # Correctness: Missing error wrapping
    if grep -q 'fmt\.Errorf.*"%v"' "$FILE" 2>/dev/null && ! grep -q '%w' "$FILE"; then
        warn "ERROR HANDLING: fmt.Errorf without %w wrapper"
    fi
    
done

echo ""
log "═══════════════════════════════════════════════════════"

if [[ "$REVIEW_PASSED" == "true" ]]; then
    log "  REVIEW PASSED - No critical issues found"
else
    error "  REVIEW FAILED - Issues found above"
fi

log "═══════════════════════════════════════════════════════"
echo ""

# Run go vet
info "Running go vet..."
if go vet ./... 2>&1; then
    log "go vet: PASSED"
else
    warn "go vet: WARNINGS"
fi

# Run static analysis
info "Checking for common issues..."
WARNINGS=0

# Check for TODO/FIXME without tracking
TODOS=$(grep -r "TODO\|FIXME\|XXX\|HACK" --include="*.go" . 2>/dev/null | grep -v "_test.go" | wc -l || echo 0)
if [[ "$TODOS" -gt 0 ]]; then
    warn "Found $TODOS TODO/FIXME comments (should be tracked)"
fi

echo ""
log "Review complete."
