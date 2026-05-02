#!/bin/sh
# install.sh - source-backed Unix installer for Gormes.
#
# Usage:
#   curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh
#   less install.sh
#   sh install.sh
#   sh install.sh --branch main
#   sh install.sh --uninstall
#
# Environment overrides:
#   GORMES_BRANCH        target branch (default: main)
#   GORMES_INSTALL_HOME  managed install home (default: $HOME/.gormes)
#   GORMES_INSTALL_DIR   managed checkout directory
#                        default (non-root): $GORMES_INSTALL_HOME/gormes-agent
#                        default (root Linux): /usr/local/lib/gormes-agent
#   GORMES_BIN_DIR       published command directory
#                        default (non-root): $HOME/.local/bin
#                        default (root Linux): /usr/local/bin
#   GORMES_PREFIX        compatibility prefix; publishes into $GORMES_PREFIX/bin
#   GORMES_RESTART_GATEWAY restart policy: auto, always, never (default: auto)
#   GORMES_GO_SHA256      optional expected SHA-256 for managed Go download
#
# Native Windows shells are not supported here. Use:
#   Invoke-WebRequest https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
#   Get-Content .\install.ps1
#   powershell -ExecutionPolicy Bypass -File .\install.ps1

set -eu

REPO_URL_SSH="${GORMES_REPO_URL_SSH:-git@github.com:TrebuchetDynamics/gormes-agent.git}"
REPO_URL_HTTPS="${GORMES_REPO_URL_HTTPS:-https://github.com/TrebuchetDynamics/gormes-agent.git}"
BRANCH="${GORMES_BRANCH:-main}"
GO_VERSION="${GORMES_GO_VERSION:-1.25.0}"
RESTART_GATEWAY="${GORMES_RESTART_GATEWAY:-auto}"
DRY_RUN=0
UNINSTALL=0
UNINSTALL_ARGS=""
LOCAL_SOURCE_DIR=""
INSTALL_LOCK_DIR=""
OLD_BUILD_TAG=""
BUILD_TAG=""
PREVIOUS_GATEWAY_PID=""
NEW_GATEWAY_PID=""
TMP_DIRS=""

log() { printf '[gormes] %s\n' "$*" >&2; }
fail() { printf '[gormes] error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Gormes Unix installer

Usage:
  install.sh [--branch NAME] [--home DIR] [--dir DIR] [--bin-dir DIR]
  install.sh --local [--bin-dir DIR]
  install.sh --dry-run
  install.sh --uninstall [gormes uninstall flags]

Options:
  --branch NAME  Git branch to install or update (default: main)
  --home DIR     Managed install home (default: $HOME/.gormes)
  --dir DIR      Managed checkout directory
                   default (non-root): $HOME/.gormes/gormes-agent
                   default (root Linux): /usr/local/lib/gormes-agent
  --bin-dir DIR  Published command directory
                   default (non-root): $HOME/.local/bin
                   default (root Linux): /usr/local/bin
  --local        Build from the current checkout instead of updating the
                 managed source checkout
  --dry-run      Print the resolved plan without cloning, building, publishing,
                 or restarting the gateway
  --uninstall    Delegate to an existing "gormes uninstall" command and exit.
                 Flags after --uninstall are passed through, for example:
                 install.sh --uninstall --dry-run
                 install.sh --uninstall --dry-run=false --yes
  --restart-gateway auto|always|never
                 Restart a live gateway after update (default: auto)
  --no-restart   Alias for --restart-gateway never
  -h, --help     Show this help

Root Linux installs use an FHS layout like Hermes Agent: code under
/usr/local/lib/gormes-agent, command in /usr/local/bin/gormes, and data under
$GORMES_INSTALL_HOME. Existing legacy root checkouts at
$GORMES_INSTALL_HOME/gormes-agent are preserved in place.
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

effective_uid() {
  if [ -n "${GORMES_INSTALL_EFFECTIVE_UID:-}" ]; then
    printf '%s\n' "$GORMES_INSTALL_EFFECTIVE_UID"
    return
  fi
  if has id; then
    id -u
    return
  fi
  printf '1\n'
}

is_root_linux_install() {
  case "$(platform_name)" in
    Linux*) ;;
    *) return 1 ;;
  esac
  is_termux && return 1
  [ "$(effective_uid)" = "0" ]
}

legacy_checkout_dir() {
  printf '%s/gormes-agent\n' "$(managed_home_dir)"
}

has_legacy_checkout() {
  [ -d "$(legacy_checkout_dir)/.git" ]
}

managed_home_dir() {
  printf '%s\n' "${GORMES_INSTALL_HOME:-$HOME/.gormes}"
}

managed_checkout_dir() {
  if [ -n "${GORMES_INSTALL_DIR:-}" ]; then
    printf '%s\n' "$GORMES_INSTALL_DIR"
    return
  fi
  if is_root_linux_install; then
    if has_legacy_checkout; then
      printf '%s\n' "$(legacy_checkout_dir)"
      return
    fi
    printf '/usr/local/lib/gormes-agent\n'
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
  if is_root_linux_install && ! has_legacy_checkout; then
    printf '/usr/local/bin\n'
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

active_command_path() {
  found=$(command -v gormes 2>/dev/null || true)
  case "$found" in
    /*|*/*) printf '%s\n' "$found" ;;
    *) printf '\n' ;;
  esac
}

uninstall_command_path() {
  published="$(pick_bin_dir)/gormes"
  if [ -x "$published" ]; then
    printf '%s\n' "$published"
    return
  fi

  active=$(active_command_path)
  if [ -n "$active" ] && [ -x "$active" ]; then
    printf '%s\n' "$active"
    return
  fi

  managed="$(managed_bin_dir)/gormes"
  if [ -x "$managed" ]; then
    printf '%s\n' "$managed"
    return
  fi

  return 1
}

run_uninstall() {
  uninstall_bin=$(uninstall_command_path 2>/dev/null || true)
  [ -n "$uninstall_bin" ] || fail "could not find an installed gormes command; rerun with --bin-dir or put gormes on PATH"
  log "running ${uninstall_bin} uninstall"
  "$uninstall_bin" uninstall "$@"
}

readlink_f() {
  if has readlink; then
    readlink -f "$1" 2>/dev/null && return
  fi
  printf '%s\n' "$1"
}

file_sha256() {
  path="$1"
  if has sha256sum; then
    sum=$(sha256sum "$path")
    printf '%s\n' "${sum%% *}"
    return
  fi
  if has shasum; then
    sum=$(shasum -a 256 "$path")
    printf '%s\n' "${sum%% *}"
    return
  fi
  return 1
}

same_binary() {
  a="$1"
  b="$2"
  [ -n "$a" ] && [ -n "$b" ] || return 1
  [ -e "$a" ] || [ -L "$a" ] || return 1
  [ -e "$b" ] || [ -L "$b" ] || return 1
  areal=$(readlink_f "$a")
  breal=$(readlink_f "$b")
  if [ "$areal" = "$breal" ]; then
    return 0
  fi
  asum=$(file_sha256 "$a" 2>/dev/null || true)
  bsum=$(file_sha256 "$b" 2>/dev/null || true)
  [ -n "$asum" ] && [ "$asum" = "$bsum" ]
}

running_gateway_pid_from_status() {
  status="$1"
  case "$status" in
    *'"gateway_state"'*'"running"'*'"pid"'*)
      rest=${status#*'"pid"'}
      rest=${rest#*:}
      while :; do
        case "$rest" in
          [0123456789]*) break ;;
          "") return 1 ;;
          *) rest=${rest#?} ;;
        esac
      done
      pid=${rest%%[!0123456789]*}
      case "$pid" in
        ""|*[!0123456789]*) return 1 ;;
        *) printf '%s\n' "$pid" ;;
      esac
      ;;
    *"runtime: running (pid="*)
      rest=${status#*"runtime: running (pid="}
      pid=${rest%%[!0123456789]*}
      case "$pid" in
        ""|*[!0123456789]*) return 1 ;;
        *) printf '%s\n' "$pid" ;;
      esac
      ;;
    *) return 1 ;;
  esac
}

gateway_status_output() {
  status_bin=$(active_command_path)
  if [ -z "$status_bin" ]; then
    status_bin="$(pick_bin_dir)/gormes"
  fi
  [ -x "$status_bin" ] || return 1
  "$status_bin" gateway status --json 2>/dev/null && return 0
  "$status_bin" gateway status 2>/dev/null || return 1
}

running_gateway_pid() {
  status=$(gateway_status_output 2>/dev/null || true)
  [ -n "$status" ] || return 1
  running_gateway_pid_from_status "$status"
}

pid_is_running() {
  [ -n "${1:-}" ] || return 1
  kill -0 "$1" 2>/dev/null
}

wait_for_pid_exit() {
  pid="$1"
  i=0
  while [ "$i" -lt 5 ]; do
    if ! pid_is_running "$pid"; then
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  return 1
}

check_platform() {
  case "$(platform_name)" in
    Linux*|Darwin*) ;;
    MINGW*|MSYS*|CYGWIN*)
      fail "native Windows shells are not supported by install.sh; download and inspect scripts/install.ps1, then run it with PowerShell" ;;
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
      --local)
        LOCAL_SOURCE_DIR=$(pwd)
        shift
        ;;
      --dry-run)
        DRY_RUN=1
        shift
        ;;
      --uninstall)
        UNINSTALL=1
        shift
        if [ "${1:-}" = "--" ]; then
          shift
        fi
        UNINSTALL_ARGS="$*"
        break
        ;;
      --restart-gateway)
        [ "$#" -ge 2 ] || fail "--restart-gateway requires auto, always, or never"
        RESTART_GATEWAY="$2"
        shift 2
        ;;
      --no-restart)
        RESTART_GATEWAY="never"
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        fail "unknown option: $1" ;;
    esac
  done
  case "$RESTART_GATEWAY" in
    auto|always|never) ;;
    *) fail "--restart-gateway must be auto, always, or never" ;;
  esac
}

ensure_prerequisites() {
  need uname
  need mkdir
  need rm
  need ln
  need mv
  need cp
  need chmod
  need sleep

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
  verify_managed_go_download "$tarball"

  rm -rf "${home}/go"
  tar -C "$home" -xzf "$tarball" || fail "could not extract Go ${GO_VERSION}"

  PATH="${home}/go/bin:${PATH}"
  export PATH
  has go || fail "managed Go install completed but go is not on PATH"
}

verify_managed_go_download() {
  tarball="$1"
  expected="${GORMES_GO_SHA256:-}"
  if [ -z "$expected" ]; then
    log "Go download sha256 verification skipped; set GORMES_GO_SHA256 to enforce it"
    return 0
  fi
  log "verifying Go download sha256"
  actual=$(file_sha256 "$tarball" 2>/dev/null || true)
  if [ -z "$actual" ]; then
    fail "could not compute sha256 for ${tarball}"
  fi
  if [ "$actual" != "$expected" ]; then
    fail "Go download sha256 mismatch: expected ${expected}, got ${actual}"
  fi
  log "Go download sha256 verified"
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
  if [ -n "$LOCAL_SOURCE_DIR" ]; then
    if [ ! -f "$LOCAL_SOURCE_DIR/go.mod" ] || [ ! -d "$LOCAL_SOURCE_DIR/cmd/gormes" ]; then
      fail "--local must be run from a Gormes source checkout"
    fi
    log "using local source checkout ${LOCAL_SOURCE_DIR}"
    return
  fi

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
  if [ -n "$LOCAL_SOURCE_DIR" ]; then
    printf '%s\n' "$LOCAL_SOURCE_DIR"
    return
  fi

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
  cache_tag=$(git -C "$build_root" rev-parse --short HEAD 2>/dev/null || echo "unknown")
  BUILD_TAG="$cache_tag"

  if [ -x "$build_bin" ]; then
    cached_tag=""
    if [ -f "${build_bin}.build-tag" ]; then
      cached_tag=$(cat "${build_bin}.build-tag" 2>/dev/null || true)
    fi
    OLD_BUILD_TAG="$cached_tag"
    if [ "$cached_tag" = "$cache_tag" ]; then
      log "binary up to date (${cache_tag}); skipping rebuild"
      return
    fi
    if [ -n "$cached_tag" ]; then
      log "source changed (${cached_tag} → ${cache_tag}); rebuilding"
    fi
  fi

  mkdir -p "$(managed_bin_dir)"
  log "building gormes from ${build_root} (${cache_tag})"
  (
    cd "$build_root" || exit 1
    go build -o "$build_bin" ./cmd/gormes
  ) || fail "go build failed"

  printf '%s\n' "$cache_tag" > "${build_bin}.build-tag"
  [ -x "$build_bin" ] || fail "build completed but ${build_bin} was not created"
}

publish_command() {
  bin_dir=$(pick_bin_dir)
  build_bin="$(managed_bin_dir)/gormes"
  published_bin="${bin_dir}/gormes"

  publish_built_binary "$build_bin" "$published_bin"
  update_active_command "$build_bin" "$published_bin"
}

publish_built_binary() {
  build_bin="$1"
  published_bin="$2"
  bin_dir=$(parent_dir "$published_bin")
  backup="${published_bin}.rollback.$$"

  [ ! -d "$published_bin" ] || fail "cannot replace directory with gormes command: ${published_bin}"
  mkdir -p "$bin_dir"
  if [ ! -w "$bin_dir" ]; then
    fail "cannot write to ${bin_dir}; rerun with --bin-dir or GORMES_BIN_DIR"
  fi

  if [ "$published_bin" = "$build_bin" ]; then
    chmod +x "$published_bin"
    return
  fi

  existed=0
  if [ -e "$published_bin" ] || [ -L "$published_bin" ]; then
    cp -P "$published_bin" "$backup" || fail "could not prepare rollback for ${published_bin}"
    existed=1
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
  if ! "$published_bin" version >/dev/null 2>&1; then
    if [ "$existed" -eq 1 ]; then
      mv -f "$backup" "$published_bin" || true
    else
      rm -f "$published_bin"
    fi
    fail "published command verification failed for ${published_bin}; rolled back"
  fi
  rm -f "$backup"
}

update_active_command() {
  build_bin="$1"
  published_bin="$2"
  paths=""
  if has which; then
    paths=$(which -a gormes 2>/dev/null || true)
  fi
  active_bin=$(active_command_path)
  if [ -n "$active_bin" ]; then
    paths="${paths}${paths:+
}${active_bin}"
  fi
  for candidate in "$HOME/.local/bin/gormes" "$HOME/go/bin/gormes"; do
    if [ -e "$candidate" ] || [ -L "$candidate" ]; then
      paths="${paths}${paths:+
}${candidate}"
    fi
  done

  seen=""
  printf '%s\n' "$paths" | while IFS= read -r active_bin; do
    [ -n "$active_bin" ] || continue
    case "$active_bin" in
      */gormes) ;;
      *) continue ;;
    esac
    case "
$seen
" in
      *"
$active_bin
"*) continue ;;
    esac
    seen="${seen}${seen:+
}${active_bin}"
    [ "$active_bin" != "$published_bin" ] || continue
    [ "$active_bin" != "$build_bin" ] || continue
    if same_binary "$active_bin" "$build_bin"; then
      continue
    fi
    log "updating active PATH command ${active_bin}"
    publish_built_binary "$build_bin" "$active_bin"
  done
}

verify_install() {
  published_bin="$(pick_bin_dir)/gormes"

  [ -x "$published_bin" ] || fail "published command is not executable: ${published_bin}"
  "$published_bin" version >/dev/null 2>&1 || fail "verification failed: ${published_bin} version"

  active_bin=$(active_command_path)
  if [ -n "$active_bin" ]; then
    "$active_bin" version >/dev/null 2>&1 || fail "verification failed: active PATH command ${active_bin} version"
  fi

  build_root=$(build_root_dir 2>/dev/null || managed_checkout_dir)
  if [ -f "$build_root/go.mod" ]; then
    log ""
    if (cd "$build_root" && go run ./cmd/progress validate) >/dev/null 2>&1; then
      log "  progress ✓"
    else
      log "  progress ⚠ (run 'go run ./cmd/progress validate' in ${build_root})"
    fi
  fi

  if "$published_bin" doctor --offline >/dev/null 2>&1; then
    log "  TUI ✓  Tools ✓  Gateway ✓  Goncho ✓  Web ✓"
  else
    doctor_out=$("$published_bin" doctor --offline 2>&1 || true)
    printf '%s\n' "$doctor_out" | while IFS= read -r line; do
      case "$line" in
        *"[PASS]"*|*"[OK]"*) log "  ✓ ${line}" ;;
        *"[FAIL]"*|*"[WARN]"*|*"[ERROR]"*) log "  ✗ ${line}" ;;
        *) log "    ${line}" ;;
      esac
    done
    log ""
    log "Free web backends: export CHROME_REMOTE_DEBUGGING_URL=http://localhost:9222 (CDP extract)"
    log "                     Gormes auto-enables DuckDuckGo for search when no API keys are set"
    log "Paid API backends: export FIRECRAWL_API_KEY=fc-xxx for full search+extract+crawl"
    log "Run 'gormes doctor' for detailed diagnostics"
  fi
}

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

install_ledger_path() {
  printf '%s/install.log.jsonl\n' "$(managed_home_dir)"
}

append_install_ledger() {
  ledger=$(install_ledger_path)
  mkdir -p "$(parent_dir "$ledger")"
  build_bin="$(managed_bin_dir)/gormes"
  hash=$(file_sha256 "$build_bin" 2>/dev/null || true)
  source=$(build_root_dir 2>/dev/null || managed_checkout_dir)
  timestamp=""
  if has date; then
    timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || true)
  fi
  {
    printf '{'
    printf '"event":"install"'
    [ -n "$timestamp" ] && printf ',"timestamp":"%s"' "$(json_escape "$timestamp")"
    printf ',"source":"%s"' "$(json_escape "$source")"
    printf ',"branch":"%s"' "$(json_escape "$BRANCH")"
    printf ',"old_commit":"%s"' "$(json_escape "$OLD_BUILD_TAG")"
    printf ',"new_commit":"%s"' "$(json_escape "$BUILD_TAG")"
    printf ',"binary_sha256":"%s"' "$(json_escape "$hash")"
    if [ -n "$PREVIOUS_GATEWAY_PID" ]; then
      printf ',"old_gateway_pid":%s' "$PREVIOUS_GATEWAY_PID"
    fi
    if [ -n "$NEW_GATEWAY_PID" ]; then
      printf ',"new_gateway_pid":%s' "$NEW_GATEWAY_PID"
    fi
    printf ',"restart_gateway":"%s"' "$(json_escape "$RESTART_GATEWAY")"
    printf '}\n'
  } >> "$ledger"
}

stop_gateway_for_restart() {
  old_pid="$1"
  stop_bin=$(active_command_path)
  if [ -z "$stop_bin" ] || [ ! -x "$stop_bin" ]; then
    stop_bin="$(managed_bin_dir)/gormes"
  fi
  [ -x "$stop_bin" ] || return 1
  "$stop_bin" gateway stop >/dev/null 2>&1 || return 1
  wait_for_pid_exit "$old_pid" || return 1
}

start_gateway_for_restart() {
  build_bin="$(managed_bin_dir)/gormes"
  home="$(managed_home_dir)"
  log_path="${home}/gateway.log"

  [ -x "$build_bin" ] || return 1
  mkdir -p "$home"

  if has setsid; then
    setsid -f "$build_bin" gateway >> "$log_path" 2>&1 < /dev/null
    return
  fi
  if has nohup; then
    nohup "$build_bin" gateway >> "$log_path" 2>&1 < /dev/null &
    return
  fi
  "$build_bin" gateway >> "$log_path" 2>&1 < /dev/null &
}

wait_for_gateway_restart() {
  old_pid="$1"
  build_bin="$(managed_bin_dir)/gormes"
  i=0
  while [ "$i" -lt 8 ]; do
    status=$("$build_bin" gateway status --json 2>/dev/null || "$build_bin" gateway status 2>/dev/null || true)
    new_pid=$(running_gateway_pid_from_status "$status" 2>/dev/null || true)
    if [ -n "$new_pid" ] && [ "$new_pid" != "$old_pid" ]; then
      printf '%s\n' "$new_pid"
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  return 1
}

restart_gateway_if_running() {
  old_pid="$1"
  case "$RESTART_GATEWAY" in
    never)
      log "gateway restart skipped by policy=never"
      return 0
      ;;
    auto)
      [ -n "$old_pid" ] || return 0
      ;;
    always)
      ;;
  esac

  if [ -n "$old_pid" ]; then
    log "restarting live gateway pid=${old_pid}"
    if ! stop_gateway_for_restart "$old_pid"; then
      log "gateway restart skipped: could not stop pid=${old_pid}"
      return 0
    fi
  else
    log "starting gateway by policy=always"
  fi
  if ! start_gateway_for_restart; then
    log "gateway restart failed: could not start $(managed_bin_dir)/gormes gateway"
    return 0
  fi
  new_pid=$(wait_for_gateway_restart "$old_pid" || true)
  NEW_GATEWAY_PID="$new_pid"
  if [ -n "$new_pid" ]; then
    if [ -n "$old_pid" ]; then
      log "gateway restarted pid=${old_pid} -> ${new_pid}"
    else
      log "gateway started pid=${new_pid}"
    fi
  else
    log "gateway restart requested; status did not report a new live pid yet"
  fi
}

bootstrap_config() {
  home=$(managed_home_dir)
  config="${home}/config.toml"

  if [ -f "$config" ]; then
    return 0
  fi

  mkdir -p "$home"
  cat > "$config" <<'GORMESCFG'
# Gormes configuration — generated by install.sh
# See: https://docs.gormes.ai/using-gormes/configuration/

[hermes]
# endpoint = "https://api.openai.com/v1"
# api_key = "sk-xxx"
# model = "gpt-4o"

# [web]
# backend = "firecrawl"
# Uncomment to use a paid web backend, or leave empty to auto-enable
# DuckDuckGo (free search) + CDP (free extract with Chrome debug port).

# [gateway]
# telegram_token = ""
# discord_token = ""
GORMESCFG

  log ""
  log "Bootstrapped ${config}"
  log "Edit it to set your provider and gateway credentials."
}

print_summary() {
  bin_dir=$(pick_bin_dir)
  published_bin="${bin_dir}/gormes"

  log ""
  log "═══ Gormes installed ═══"
  log "  binary: ${published_bin}"
  log "  source: $(build_root_dir 2>/dev/null || managed_checkout_dir)"
  log "  config: $(managed_home_dir)/config.toml"
  active_bin=$(active_command_path)
  if [ -n "$active_bin" ] && [ "$active_bin" != "$published_bin" ] && ! same_binary "$active_bin" "$published_bin"; then
    log "  active: ${active_bin} (older install; run 'hash -r' or restart shell)"
  fi

  if path_contains_dir "$bin_dir"; then
    log "  PATH:   ${bin_dir} ✓"
  else
    log ""
    log "Add to PATH (copy one):"
    case "${SHELL##*/}" in
      zsh)  log "  echo 'export PATH=\"${bin_dir}:\$PATH\"' >> ~/.zshrc" ;;
      bash) log "  echo 'export PATH=\"${bin_dir}:\$PATH\"' >> ~/.bashrc" ;;
      fish) log "  fish_add_path ${bin_dir}" ;;
      *)    log "  export PATH=\"${bin_dir}:\$PATH\"" ;;
    esac
  fi

  if [ "${GORMES_SKIP_SERVICE:-0}" != "1" ]; then
    print_service_instructions
  fi

  log ""
  log "Quick start:"
  log "  gormes --offline        # smoke test TUI"
  log "  gormes doctor --offline # check everything"
  log "  gormes dashboard        # web UI at http://127.0.0.1:43827/dashboard"
  log ""
  log "Free web backends (no API keys needed):"
  log "  Search: auto-enabled via DuckDuckGo"
  log "  Extract: export CHROME_REMOTE_DEBUGGING_URL=http://localhost:9222"
  log "           (Chrome needs --remote-debugging-port=9222)"
  log ""
  log "Update: rerun this installer."
}

print_service_instructions() {
  if has systemctl && systemctl --user >/dev/null 2>&1; then
    install_systemd_user_service
  elif [ "$(platform_name)" = "Darwin" ] && has launchctl; then
    install_launchd_service
  fi
}

install_systemd_user_service() {
  unit_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
  service_file="${unit_dir}/gormes-gateway.service"
  mkdir -p "$unit_dir"

  if [ -f "$service_file" ]; then
    systemctl --user stop gormes-gateway.service 2>/dev/null || true
  fi

  build_bin="$(managed_bin_dir)/gormes"
  cat > "$service_file" <<SYSTEMDUNIT
[Unit]
Description=Gormes Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${build_bin} gateway
ExecReload=${build_bin} gateway stop && ${build_bin} gateway
Restart=on-failure
RestartSec=5
Environment=GORMES_HOME=$(managed_home_dir)

[Install]
WantedBy=default.target
SYSTEMDUNIT

  systemctl --user daemon-reload
  systemctl --user enable gormes-gateway.service

  log ""
  log "systemd user service installed:"
  log "  systemctl --user start gormes-gateway    # start now"
  log "  systemctl --user status gormes-gateway   # check status"
  log "  journalctl --user -u gormes-gateway -f   # follow logs"
  log "  (auto-starts on login; survives reboots)"
}

install_launchd_service() {
  plist_dir="${HOME}/Library/LaunchAgents"
  plist_file="${plist_dir}/com.gormes.gateway.plist"
  mkdir -p "$plist_dir"

  if [ -f "$plist_file" ]; then
    launchctl bootout "gui/$(id -u)/com.gormes.gateway" 2>/dev/null || true
  fi

  build_bin="$(managed_bin_dir)/gormes"
  cat > "$plist_file" <<PLISTUNIT
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.gormes.gateway</string>
    <key>ProgramArguments</key>
    <array>
        <string>${build_bin}</string>
        <string>gateway</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>GORMES_HOME</key>
        <string>$(managed_home_dir)</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$(managed_home_dir)/gateway.log</string>
    <key>StandardErrorPath</key>
    <string>$(managed_home_dir)/gateway.log</string>
</dict>
</plist>
PLISTUNIT

  launchctl bootstrap "gui/$(id -u)" "$plist_file" 2>/dev/null || \
    launchctl load "$plist_file" 2>/dev/null || true

  log ""
  log "launchd service installed:"
  log "  launchctl list com.gormes.gateway    # check status"
  log "  tail -f $(managed_home_dir)/gateway.log   # follow logs"
  log "  (auto-starts on login; survives reboots)"
}

print_dry_run() {
  log "dry run"
  log "  branch: ${BRANCH}"
  if [ -n "$LOCAL_SOURCE_DIR" ]; then
    log "  source: ${LOCAL_SOURCE_DIR}"
  else
    log "  source: $(managed_checkout_dir)"
  fi
  log "  install_home: $(managed_home_dir)"
  log "  managed_binary: $(managed_bin_dir)/gormes"
  log "  published_binary: $(pick_bin_dir)/gormes"
  log "  restart_gateway: ${RESTART_GATEWAY}"
}

acquire_install_lock() {
  home=$(managed_home_dir)
  lock="${home}/install.lock"
  mkdir -p "$home"
  if mkdir "$lock" 2>/dev/null; then
    INSTALL_LOCK_DIR="$lock"
    printf '%s\n' "$$" > "${lock}/pid" 2>/dev/null || true
    trap release_install_lock EXIT INT TERM
    return 0
  fi
  fail "another install is already running; remove ${lock} if it is stale"
}

cleanup_tmp_dirs() {
  if [ -n "$TMP_DIRS" ]; then
    for d in $TMP_DIRS; do
      rm -rf "$d" 2>/dev/null || true
    done
  fi
}

release_install_lock() {
  if [ -n "${INSTALL_LOCK_DIR:-}" ] && [ -d "$INSTALL_LOCK_DIR" ]; then
    rm -rf "$INSTALL_LOCK_DIR"
  fi
  cleanup_tmp_dirs
}

main() {
  parse_args "$@"
  if [ "$UNINSTALL" -eq 1 ]; then
    # shellcheck disable=SC2086
    run_uninstall $UNINSTALL_ARGS
    return
  fi
  if [ "$DRY_RUN" -eq 1 ]; then
    print_dry_run
    return
  fi
  acquire_install_lock
  PREVIOUS_GATEWAY_PID=$(running_gateway_pid 2>/dev/null || true)
  ensure_prerequisites
  ensure_checkout
  build_gormes
  publish_command
  bootstrap_config
  verify_install
  restart_gateway_if_running "$PREVIOUS_GATEWAY_PID"
  append_install_ledger
  print_summary
}

if [ "${GORMES_INSTALL_TEST_MODE:-}" != "1" ]; then
  main "$@"
fi
