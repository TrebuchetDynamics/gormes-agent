#!/bin/sh
# install.sh - source-backed Unix installer for Gormes.
#
# Usage:
#   curl -fsSL https://gormes.ai/install.sh | sh
#   curl -fsSL https://gormes.ai/install.sh | sh -s -- --branch main
#
# Environment overrides:
#   GORMES_BRANCH        target branch (default: main)
#   GORMES_INSTALL_HOME  managed install home (default: $HOME/.gormes)
#   GORMES_INSTALL_DIR   managed checkout directory (default: $GORMES_INSTALL_HOME/gormes-agent)
#   GORMES_BIN_DIR       published command directory (default: $HOME/.local/bin)
#   GORMES_PREFIX        compatibility prefix; publishes into $GORMES_PREFIX/bin
#
# Native Windows shells are not supported here. Use:
#   irm https://gormes.ai/install.ps1 | iex

set -eu

REPO_URL_SSH="${GORMES_REPO_URL_SSH:-git@github.com:TrebuchetDynamics/gormes-agent.git}"
REPO_URL_HTTPS="${GORMES_REPO_URL_HTTPS:-https://github.com/TrebuchetDynamics/gormes-agent.git}"
BRANCH="${GORMES_BRANCH:-main}"
GO_VERSION="${GORMES_GO_VERSION:-1.25.0}"

log() { printf '[gormes] %s\n' "$*" >&2; }
fail() { printf '[gormes] error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Gormes Unix installer

Usage:
  install.sh [--branch NAME] [--home DIR] [--dir DIR] [--bin-dir DIR]

Options:
  --branch NAME  Git branch to install or update (default: main)
  --home DIR     Managed install home (default: $HOME/.gormes)
  --dir DIR      Managed checkout directory (default: $HOME/.gormes/gormes-agent)
  --bin-dir DIR  Published command directory (default: $HOME/.local/bin)
  -h, --help     Show this help
EOF
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found: $1"
}

has() {
  command -v "$1" >/dev/null 2>&1
}

platform_name() {
  if [ -n "${UNAME:-}" ]; then
    printf '%s\n' "$UNAME"
    return
  fi
  uname -s
}

is_termux() {
  [ -n "${TERMUX_VERSION:-}" ] || case "${PREFIX:-}" in
    *com.termux/files/usr*) return 0 ;;
    *) return 1 ;;
  esac
}

managed_home_dir() {
  printf '%s\n' "${GORMES_INSTALL_HOME:-$HOME/.gormes}"
}

managed_checkout_dir() {
  if [ -n "${GORMES_INSTALL_DIR:-}" ]; then
    printf '%s\n' "$GORMES_INSTALL_DIR"
    return
  fi
  printf '%s/gormes-agent\n' "$(managed_home_dir)"
}

managed_bin_dir() {
  printf '%s/bin\n' "$(managed_home_dir)"
}

pick_bin_dir() {
  if [ -n "${GORMES_BIN_DIR:-}" ]; then
    printf '%s\n' "$GORMES_BIN_DIR"
    return
  fi
  if [ -n "${GORMES_PREFIX:-}" ]; then
    printf '%s/bin\n' "$GORMES_PREFIX"
    return
  fi
  if is_termux && [ -n "${PREFIX:-}" ]; then
    printf '%s/bin\n' "$PREFIX"
    return
  fi
  printf '%s/.local/bin\n' "$HOME"
}

parent_dir() {
  case "$1" in
    */*) printf '%s\n' "${1%/*}" ;;
    *) printf '.\n' ;;
  esac
}

path_contains_dir() {
  case ":${PATH:-}:" in
    *":$1:"*) return 0 ;;
    *) return 1 ;;
  esac
}

check_platform() {
  case "$(platform_name)" in
    Linux*|Darwin*) ;;
    MINGW*|MSYS*|CYGWIN*)
      fail "native Windows shells are not supported by install.sh; use PowerShell: irm https://gormes.ai/install.ps1 | iex" ;;
    *) fail "unsupported OS: $(platform_name)" ;;
  esac
}

check_go_version() {
  goversion=$(current_go_version)
  go_version_supported "$goversion" || fail "Go 1.25+ required; found ${goversion}"
}

current_go_version() {
  goversion=$(go env GOVERSION 2>/dev/null || true)
  if [ -z "$goversion" ]; then
    set -- $(go version 2>/dev/null || true)
    goversion="${3:-unknown}"
  fi
  printf '%s\n' "$goversion"
}

go_version_supported() {
  case "$1" in
    go1.2[5-9]*|go1.[3-9][0-9]*|go[2-9]*)
      return 0 ;;
    *)
      return 1 ;;
  esac
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --branch)
        [ "$#" -ge 2 ] || fail "--branch requires a value"
        BRANCH="$2"
        shift 2
        ;;
      --home)
        [ "$#" -ge 2 ] || fail "--home requires a value"
        GORMES_INSTALL_HOME="$2"
        export GORMES_INSTALL_HOME
        shift 2
        ;;
      --dir)
        [ "$#" -ge 2 ] || fail "--dir requires a value"
        GORMES_INSTALL_DIR="$2"
        export GORMES_INSTALL_DIR
        shift 2
        ;;
      --bin-dir)
        [ "$#" -ge 2 ] || fail "--bin-dir requires a value"
        GORMES_BIN_DIR="$2"
        export GORMES_BIN_DIR
        shift 2
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        fail "unknown option: $1" ;;
    esac
  done
}

ensure_prerequisites() {
  need uname
  need mkdir
  need rm
  need ln
  need mv
  need cp
  need chmod

  check_platform

  if is_termux; then
    ensure_termux_core_packages
  else
    ensure_git
    ensure_go
  fi
}

run_privileged() {
  if [ "${GORMES_INSTALL_TEST_MODE:-}" = "1" ]; then
    "$@"
    return
  fi

  if has id && [ "$(id -u)" = "0" ]; then
    "$@"
    return
  fi

  if has sudo; then
    sudo "$@"
    return
  fi

  fail "administrator permission is needed to install missing OS packages; install them manually or rerun with sudo available"
}

install_os_packages() {
  [ "$#" -gt 0 ] || return 0

  if is_termux && has pkg; then
    pkg install -y "$@"
    return
  fi

  if has brew; then
    brew install "$@"
    return
  fi

  if has apt-get; then
    run_privileged apt-get update
    run_privileged apt-get install -y "$@"
    return
  fi

  if has dnf; then
    run_privileged dnf install -y "$@"
    return
  fi

  if has pacman; then
    run_privileged pacman -S --noconfirm "$@"
    return
  fi

  return 1
}

ensure_termux_core_packages() {
  has pkg || fail "Termux package manager not found; install pkg support before rerunning"

  packages=""

  if ! has git; then
    packages="git"
  fi

  if ! has go; then
    packages="${packages}${packages:+ }golang"
  else
    goversion=$(current_go_version)
    if ! go_version_supported "$goversion"; then
      packages="${packages}${packages:+ }golang"
    fi
  fi

  if [ -n "$packages" ]; then
    log "installing missing Termux packages: ${packages}"
    # shellcheck disable=SC2086
    pkg install -y $packages || fail "could not install required Termux packages: ${packages}"
  fi

  has git || fail "Git is required and could not be installed with pkg"
  has go || fail "Go is required and could not be installed with pkg"
  check_go_version
}

ensure_git() {
  if has git; then
    return
  fi

  log "Git not found; attempting to install it"
  install_os_packages git || fail "Git is required and could not be installed automatically"
  has git || fail "Git install completed but git is still not on PATH"
}

ensure_go() {
  if has go; then
    goversion=$(current_go_version)
    if go_version_supported "$goversion"; then
      return
    fi
    log "found ${goversion}; installing managed Go ${GO_VERSION}"
  else
    log "Go not found; installing managed Go ${GO_VERSION}"
  fi

  install_managed_go
  check_go_version
}

go_platform() {
  case "$(platform_name)" in
    Linux*) printf 'linux\n' ;;
    Darwin*) printf 'darwin\n' ;;
    *) fail "managed Go download is not supported on this OS" ;;
  esac
}

go_arch() {
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) printf 'amd64\n' ;;
    aarch64|arm64) printf 'arm64\n' ;;
    i386|i686) printf '386\n' ;;
    armv6l|armv7l) printf 'armv6l\n' ;;
    *) fail "managed Go download is not supported for architecture: ${arch}" ;;
  esac
}

ensure_download_tools() {
  if has curl || has wget; then
    :
  else
    log "download tool not found; attempting to install curl"
    install_os_packages curl || fail "curl or wget is required to download Go"
  fi

  if has tar; then
    :
  else
    log "tar not found; attempting to install tar"
    install_os_packages tar || fail "tar is required to install Go"
  fi

  if ! has curl && ! has wget; then
    fail "curl or wget is required to download Go"
  fi
  has tar || fail "tar is required to install Go"
}

download_file() {
  url="$1"
  out="$2"

  if has curl; then
    curl -fsSL "$url" -o "$out"
    return
  fi

  wget -q "$url" -O "$out"
}

install_managed_go() {
  home=$(managed_home_dir)
  managed_go="${home}/go/bin/go"

  if [ -x "$managed_go" ]; then
    PATH="${home}/go/bin:${PATH}"
    export PATH
    goversion=$(current_go_version)
    if go_version_supported "$goversion"; then
      log "using managed ${goversion}"
      return
    fi
  fi

  ensure_download_tools

  os=$(go_platform)
  arch=$(go_arch)
  tarball_dir="${home}/tmp"
  tarball="${tarball_dir}/go${GO_VERSION}.${os}-${arch}.tar.gz"
  url="https://go.dev/dl/go${GO_VERSION}.${os}-${arch}.tar.gz"

  mkdir -p "$tarball_dir"
  log "downloading Go ${GO_VERSION} for ${os}/${arch}"
  download_file "$url" "$tarball" || fail "could not download Go ${GO_VERSION}"

  rm -rf "${home}/go"
  tar -C "$home" -xzf "$tarball" || fail "could not extract Go ${GO_VERSION}"

  PATH="${home}/go/bin:${PATH}"
  export PATH
  has go || fail "managed Go install completed but go is not on PATH"
}

clone_checkout() {
  checkout_dir=$(managed_checkout_dir)
  mkdir -p "$(parent_dir "$checkout_dir")"

  log "cloning Gormes into ${checkout_dir}"
  if GIT_SSH_COMMAND="ssh -o BatchMode=yes -o ConnectTimeout=5" \
    git clone --branch "$BRANCH" "$REPO_URL_SSH" "$checkout_dir"; then
    return
  fi

  log "SSH clone failed; retrying HTTPS"
  rm -rf "$checkout_dir"
  git clone --branch "$BRANCH" "$REPO_URL_HTTPS" "$checkout_dir" ||
    fail "could not clone Gormes from SSH or HTTPS"
}

update_checkout() {
  checkout_dir=$(managed_checkout_dir)

  if [ ! -d "$checkout_dir/.git" ]; then
    fail "${checkout_dir} exists but is not a git checkout; remove it or rerun with --dir"
  fi

  log "updating managed checkout ${checkout_dir}"
  (
    cd "$checkout_dir" || exit 1

    stashed=0
    if [ -n "$(git status --porcelain)" ]; then
      log "local changes detected; stashing before update"
      git stash push -u -m "gormes installer autostash" >/dev/null ||
        fail "could not stash local changes in ${checkout_dir}"
      stashed=1
    fi

    git fetch origin "$BRANCH" || fail "could not fetch origin/${BRANCH}"
    git checkout "$BRANCH" || fail "could not checkout ${BRANCH}"
    git pull --ff-only origin "$BRANCH" || fail "could not fast-forward ${BRANCH}"

    if [ "$stashed" -eq 1 ]; then
      git stash pop >/dev/null ||
        fail "updated checkout but could not reapply stashed changes; inspect: cd ${checkout_dir} && git stash list"
      log "local changes restored after update"
    fi
  )
}

ensure_checkout() {
  checkout_dir=$(managed_checkout_dir)

  if [ -d "$checkout_dir" ]; then
    update_checkout
    return
  fi

  if [ -e "$checkout_dir" ]; then
    fail "${checkout_dir} exists but is not a directory; remove it or rerun with --dir"
  fi

  clone_checkout
}

build_root_dir() {
  checkout_dir=$(managed_checkout_dir)

  if [ -f "$checkout_dir/go.mod" ] && [ -d "$checkout_dir/cmd/gormes" ]; then
    printf '%s\n' "$checkout_dir"
    return
  fi

  if [ -f "$checkout_dir/gormes/go.mod" ] && [ -d "$checkout_dir/gormes/cmd/gormes" ]; then
    printf '%s/gormes\n' "$checkout_dir"
    return
  fi

  fail "could not find a Gormes Go module under ${checkout_dir}"
}

build_gormes() {
  build_bin="$(managed_bin_dir)/gormes"
  build_root=$(build_root_dir)

  mkdir -p "$(managed_bin_dir)"
  log "building gormes from ${build_root}"
  (
    cd "$build_root" || exit 1
    go build -o "$build_bin" ./cmd/gormes
  ) || fail "go build failed"

  [ -x "$build_bin" ] || fail "build completed but ${build_bin} was not created"
}

publish_command() {
  bin_dir=$(pick_bin_dir)
  build_bin="$(managed_bin_dir)/gormes"
  published_bin="${bin_dir}/gormes"

  mkdir -p "$bin_dir"
  if [ ! -w "$bin_dir" ]; then
    fail "cannot write to ${bin_dir}; rerun with --bin-dir or GORMES_BIN_DIR"
  fi

  if [ "$published_bin" = "$build_bin" ]; then
    chmod +x "$published_bin"
    return
  fi

  tmp="${published_bin}.tmp.$$"
  rm -f "$tmp"
  if ln -s "$build_bin" "$tmp" 2>/dev/null; then
    :
  else
    cp "$build_bin" "$tmp" || fail "could not copy ${build_bin} to ${tmp}"
    chmod +x "$tmp"
  fi
  mv -f "$tmp" "$published_bin" || fail "could not publish ${published_bin}"
}

verify_install() {
  published_bin="$(pick_bin_dir)/gormes"

  [ -x "$published_bin" ] || fail "published command is not executable: ${published_bin}"
  "$published_bin" version >/dev/null 2>&1 || fail "verification failed: ${published_bin} version"

  if "$published_bin" doctor --offline >/dev/null 2>&1; then
    log "offline doctor passed"
  else
    log "note: offline doctor did not pass; core version smoke check succeeded"
  fi
}

print_summary() {
  bin_dir=$(pick_bin_dir)
  published_bin="${bin_dir}/gormes"

  log "Core install: succeeded"
  log "Managed checkout: $(managed_checkout_dir)"
  log "Published command: ${published_bin}"
  log "Verification: succeeded"

  if path_contains_dir "$bin_dir"; then
    log "PATH: ${bin_dir} is already available"
  else
    log "PATH: add this line to your shell profile:"
    log "  export PATH=\"${bin_dir}:\$PATH\""
  fi

  log "Update: rerun this installer to update Gormes"
}

main() {
  parse_args "$@"
  ensure_prerequisites
  ensure_checkout
  build_gormes
  publish_command
  verify_install
  print_summary
}

if [ "${GORMES_INSTALL_TEST_MODE:-}" != "1" ]; then
  main "$@"
fi
