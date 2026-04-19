#!/usr/bin/env bash
#
# Check whether the Gormes binary builds under Go 1.22.
#
# Exit codes:
#   0 — build succeeded; Termux/LTS portability preserved
#   1 — build failed under Go 1.22; offending packages listed
#   2 — neither Docker nor `go1.22.10` toolchain is available
#
# Preferred path: Docker (golang:1.22-alpine).
# Fallback path:  golang.org/dl/go1.22.10 downloadable toolchain.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
LOG="$(mktemp)"
trap 'rm -f "${LOG}"' EXIT

print_decision_summary() {
    local status="$1"
    echo
    echo "=== Decision data for 'Portability vs. Progress' ==="
    if [[ "${status}" -eq 0 ]]; then
        echo "  ✓ Go 1.22 builds cleanly — no action needed"
        return
    fi
    local offenders
    offenders=$(grep -E 'requires go1\.[0-9]+' "${LOG}" | sort -u || true)
    if [[ -n "${offenders}" ]]; then
        echo "Incompatible dependencies:"
        echo "${offenders}" | sed 's/^/  /'
    fi
    local symbols
    symbols=$(grep -E '^.*: undefined: .+' "${LOG}" | sort -u || true)
    if [[ -n "${symbols}" ]]; then
        echo
        echo "Undefined symbols in 1.22 toolchain:"
        echo "${symbols}" | sed 's/^/  /'
    fi
    echo
    echo "Options:"
    echo "  a) Accept Go 1.24 floor as the Gormes minimum"
    echo "  b) Downgrade the offending dependencies to 1.22-compatible versions"
    echo
    echo "Raw build log (last 40 lines):"
    tail -40 "${LOG}" | sed 's/^/  /'
}

try_docker() {
    if ! command -v docker >/dev/null 2>&1; then
        return 2
    fi
    echo "=== Go 1.22 compatibility check (Docker path) ==="
    docker run --rm \
        -v "${REPO_ROOT}:/src:ro" \
        -w /src/gormes \
        --tmpfs /tmp \
        -e GOCACHE=/tmp/gocache \
        -e GOMODCACHE=/tmp/gomodcache \
        golang:1.22-alpine \
        sh -c 'go build ./cmd/gormes' \
        > "${LOG}" 2>&1
    return $?
}

try_fallback() {
    if ! command -v go >/dev/null 2>&1; then
        return 2
    fi
    if ! command -v go1.22.10 >/dev/null 2>&1; then
        echo "Installing golang.org/dl/go1.22.10 …"
        GO111MODULE=on go install golang.org/dl/go1.22.10@latest >> "${LOG}" 2>&1 || return 2
        go1.22.10 download >> "${LOG}" 2>&1 || return 2
    fi
    echo "=== Go 1.22 compatibility check (fallback path) ==="
    (cd "${REPO_ROOT}/gormes" && go1.22.10 build ./cmd/gormes) >> "${LOG}" 2>&1
    return $?
}

main() {
    local status
    try_docker
    status=$?
    if [[ "${status}" -eq 2 ]]; then
        try_fallback
        status=$?
    fi

    case "${status}" in
        0)
            echo "PASS: gormes builds under Go 1.22"
            print_decision_summary 0
            exit 0
            ;;
        2)
            echo "UNAVAILABLE: neither Docker nor golang.org/dl/go1.22.10 could be used."
            echo "  Install one:"
            echo "    - Docker Desktop / docker.io"
            echo "    - go install golang.org/dl/go1.22.10@latest && go1.22.10 download"
            exit 2
            ;;
        *)
            echo "FAIL: gormes does NOT build under Go 1.22 (exit ${status})"
            print_decision_summary "${status}"
            exit 1
            ;;
    esac
}

main "$@"
