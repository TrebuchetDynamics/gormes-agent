#!/bin/sh
# install.sh — install Gormes from source via go install.
#
# Usage:
#   curl -fsSL https://gormes.ai/install.sh | sh
#
# Requirements:
#   - Go 1.25+ on PATH
#
# Environment overrides:
#   GORMES_MODULE   — go install target (default: github.com/TrebuchetDynamics/gormes-agent/gormes/cmd/gormes)
#   GORMES_VERSION  — version suffix passed to go install (default: latest)
#   GORMES_PREFIX   — optional install prefix; when set, install into $GORMES_PREFIX/bin via GOBIN
#
# Native Windows is not supported. Install WSL2 and rerun inside it.

set -eu

MODULE="${GORMES_MODULE:-github.com/TrebuchetDynamics/gormes-agent/gormes/cmd/gormes}"
VERSION="${GORMES_VERSION:-latest}"
PREFIX="${GORMES_PREFIX:-}"

log()  { printf '[gormes] %s\n' "$*" >&2; }
fail() { printf '[gormes] error: %s\n' "$*" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || fail "required tool not found: $1"; }

check_platform() {
  case "$(uname -s)" in
    Linux*|Darwin*) ;;
    MINGW*|MSYS*|CYGWIN*)
      fail "native Windows is not supported — install WSL2 and rerun this script inside it" ;;
    *) fail "unsupported OS: $(uname -s)" ;;
  esac
}

check_go_version() {
  goversion=$(go env GOVERSION 2>/dev/null || go version | awk '{print $3}')
  case "$goversion" in
    go1.2[5-9]*|go1.[3-9][0-9]*|go[2-9]*)
      ;;
    *)
      fail "Go 1.25+ required today; found ${goversion}" ;;
  esac
}

pick_bin_dir() {
  if [ -n "$PREFIX" ]; then
    printf '%s/bin\n' "$PREFIX"
    return
  fi

  gobin=$(go env GOBIN 2>/dev/null || true)
  if [ -n "$gobin" ]; then
    printf '%s\n' "$gobin"
    return
  fi

  gopath=$(go env GOPATH 2>/dev/null || true)
  [ -n "$gopath" ] || fail "go env GOPATH returned empty"
  printf '%s/bin\n' "$gopath"
}

main() {
  need go
  need mkdir
  need uname
  need awk

  check_platform
  check_go_version

  BIN_DIR=$(pick_bin_dir)
  mkdir -p "$BIN_DIR"
  if [ ! -w "$BIN_DIR" ]; then
    fail "cannot write to ${BIN_DIR} — set GORMES_PREFIX to a writable path"
  fi

  if [ -n "$PREFIX" ]; then
    export GOBIN="$BIN_DIR"
  fi

  log "installing ${MODULE}@${VERSION}"
  go install "${MODULE}@${VERSION}"

  BINARY="${BIN_DIR}/gormes"
  if [ ! -x "$BINARY" ]; then
    fail "go install completed but ${BINARY} was not created"
  fi

  log "installed ${BINARY}"

  case ":${PATH:-}:" in
    *":${BIN_DIR}:"*) ;;
    *)
      log "note: ${BIN_DIR} is not in your PATH"
      log "add it:  export PATH=\"${BIN_DIR}:\$PATH\""
      ;;
  esac

  log "verify:  gormes version"
  log "doctor:  gormes doctor --offline"
}

main "$@"
