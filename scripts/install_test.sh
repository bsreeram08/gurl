#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/install.sh"

assert_eq() {
    local expected="$1"
    local actual="$2"
    local message="$3"
    if [[ "$actual" != "$expected" ]]; then
        printf 'FAIL: %s\nexpected: %s\nactual:   %s\n' "$message" "$expected" "$actual" >&2
        exit 1
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="$3"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'FAIL: %s\nmissing: %s\nvalue:   %s\n' "$message" "$needle" "$haystack" >&2
        exit 1
    fi
}

assert_not_contains() {
    local haystack="$1"
    local needle="$2"
    local message="$3"
    if [[ "$haystack" == *"$needle"* ]]; then
        printf 'FAIL: %s\nunexpected: %s\nvalue:      %s\n' "$message" "$needle" "$haystack" >&2
        exit 1
    fi
}

make_temp_install_dir() {
    local tmp_root="/tmp/opencode/gurl-install-tests"
    mkdir -p "$tmp_root"
    mktemp -d "$tmp_root/install-XXXXXX"
}

test_install_honors_install_dir_environment_override() {
    local workspace home_dir install_dir bin_dir default_dir
    workspace="$(mktemp -d /tmp/opencode/gurl-install-tests/full-run-XXXXXX)"
    home_dir="$workspace/home"
    install_dir="$workspace/custom-bin"
    default_dir="$home_dir/.local/bin"
    bin_dir="$workspace/bin"

    mkdir -p "$bin_dir"

    cat >"$bin_dir/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

output=""
while (($#)); do
    case "$1" in
        -o)
            output="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

cat >"$output" <<'BIN'
#!/usr/bin/env bash
if [[ "${1:-}" == "--version" ]]; then
  echo "gurl version v0.3.0"
  exit 0
fi
exit 0
BIN
exit 0
EOF
    chmod +x "$bin_dir/curl"

    mkdir -p "$home_dir"

    set +e
    HOME="$home_dir" INSTALL_DIR="$install_dir" VERSION="0.3.0" PATH="$bin_dir:$PATH" \
        bash "$SCRIPT_DIR/install.sh" >/dev/null 2>&1
    local status=$?
    set -e

    if [[ $status -ne 0 ]]; then
        printf 'FAIL: installer exited %s with INSTALL_DIR override\n' "$status" >&2
        exit 1
    fi
    if [[ ! -x "$install_dir/gurl" ]]; then
        printf 'FAIL: expected installer to write to override dir %s\n' "$install_dir/gurl" >&2
        exit 1
    fi
    if [[ -e "$default_dir/gurl" ]]; then
        printf 'FAIL: expected installer not to use default dir %s\n' "$default_dir/gurl" >&2
        exit 1
    fi
}

test_get_latest_version_falls_back_to_public_redirect() {
    local tag

    curl() {
        local url=""
        while (($#)); do
            case "$1" in
                http*)
                    url="$1"
                    shift
                    ;;
                *)
                    shift
                    ;;
            esac
        done

        case "$url" in
            *api.github.com/repos/bsreeram08/gurl/releases/latest)
                printf '%s' '{"tag_name":""}'
                ;;
            *github.com/bsreeram08/gurl/releases/latest)
                printf '%s' 'https://github.com/bsreeram08/gurl/releases/tag/v0.3.0'
                ;;
            *)
                printf 'unexpected curl url: %s\n' "$url" >&2
                return 1
                ;;
        esac
    }

    tag="$(get_latest_version)"
    assert_eq "v0.3.0" "$tag" "latest release should fall back to public redirect"
}

test_get_latest_version_rejects_bare_v_from_redirect() {
    curl() {
        local url=""
        while (($#)); do
            case "$1" in
                http*)
                    url="$1"
                    shift
                    ;;
                *)
                    shift
                    ;;
            esac
        done

        case "$url" in
            *api.github.com/repos/bsreeram08/gurl/releases/latest)
                printf '%s' '{"message":"rate limited"}'
                ;;
            *github.com/bsreeram08/gurl/releases/latest)
                printf '%s' 'https://github.com/bsreeram08/gurl/releases/tag/v'
                ;;
            *)
                printf 'unexpected curl url: %s\n' "$url" >&2
                return 1
                ;;
        esac
    }

    if get_latest_version >/dev/null 2>&1; then
        printf 'FAIL: expected bare v redirect to be rejected\n' >&2
        exit 1
    fi
}

test_install_normalizes_explicit_versions() {
    local install_dir
    install_dir="$(make_temp_install_dir)"

    for version in "v0.3.0" "0.3.0"; do
        local download_url=""
        local output_path=""

        curl() {
            local output=""
            local url=""
            while (($#)); do
                case "$1" in
                    -o)
                        output="$2"
                        shift 2
                        ;;
                    http*)
                        url="$1"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done

            download_url="$url"
            output_path="$output"

            cat >"$output" <<'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == "--version" ]]; then
  echo "gurl version v0.3.0"
  exit 0
fi
exit 0
EOF
            return 0
        }

        VERSION="$version" INSTALL_DIR="$install_dir" \
        install

        assert_contains "$download_url" "/releases/download/v0.3.0/gurl-" "explicit version $version should normalize to v0.3.0"
        if [[ ! -x "$output_path" ]]; then
            printf 'FAIL: expected installed binary to be executable for version %s\n' "$version" >&2
            exit 1
        fi
    done
}

test_install_rejects_bare_v_before_download() {
    local install_dir
    install_dir="$(make_temp_install_dir)"
    local download_requested=false

    curl() {
        download_requested=true
        local output=""
        while (($#)); do
            case "$1" in
                -o)
                    output="$2"
                    shift 2
                    ;;
                *)
                    shift
                    ;;
            esac
        done

        if [[ -n "$output" ]]; then
            cat >"$output" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
        fi
        return 0
    }

    set +e
    VERSION="v" INSTALL_DIR="$install_dir" install >/dev/null 2>&1
    local status=$?
    set -e

    if [[ $status -eq 0 ]]; then
        printf 'FAIL: expected VERSION=v to be rejected before download\n' >&2
        exit 1
    fi

    if [[ "$download_requested" == true ]]; then
        printf 'FAIL: expected VERSION=v to be rejected before any download\n' >&2
        exit 1
    fi
}

test_install_fails_fast_on_download_errors() {
    local install_dir
    install_dir="$(make_temp_install_dir)"
    local download_args=""

    curl() {
        local output=""
        local args=""
        local saw_fail_flag=false
        while (($#)); do
            args+=" $1"
            case "$1" in
                -o)
                    output="$2"
                    args+=" $2"
                    shift 2
                    ;;
                -*f*)
                    saw_fail_flag=true
                    shift
                    ;;
                http*)
                    shift
                    ;;
                *)
                    shift
                    ;;
            esac
        done

        download_args="$args"

        if [[ "$saw_fail_flag" == true ]]; then
            return 22
        fi

        cat >"$output" <<'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == "--version" ]]; then
  echo "gurl version v0.3.0"
  exit 0
fi
exit 0
EOF
        return 0
    }

    set +e
    VERSION="0.3.0" INSTALL_DIR="$install_dir" install >/dev/null 2>&1
    local status=$?
    set -e

    if [[ $status -eq 0 ]]; then
        printf 'FAIL: expected download errors to stop the installer\n' >&2
        exit 1
    fi

    assert_contains "$download_args" "-f" "curl should fail fast on download errors"
}

main() {
    test_install_honors_install_dir_environment_override
    test_get_latest_version_falls_back_to_public_redirect
    test_get_latest_version_rejects_bare_v_from_redirect
    test_install_normalizes_explicit_versions
    test_install_rejects_bare_v_before_download
    test_install_fails_fast_on_download_errors
    printf 'install.sh tests passed\n'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
