#!/bin/sh
# install.sh - release-first Unix installer for Gormes, with source fallback.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | bash
#   curl -fsSL https://gormes.ai/install.sh | bash
#   sh install.sh --branch main
#   sh install.sh --uninstall
#
# Environment overrides:
#   GORMES_BRANCH        target branch (default: main)
#   GORMES_INSTALL_HOME  managed install home (default: $HOME/.gormes)
#   GORMES_INSTALL_DIR   managed source checkout directory
#   GORMES_BIN_DIR       published command directory
#                        default (non-root): $HOME/.local/bin
#                        default (root Linux): /usr/local/bin
#   GORMES_PREFIX        compatibility prefix; publishes into $GORMES_PREFIX/bin
#   GORMES_RESTART_GATEWAY restart policy: auto, always, never (default: auto)
#   GORMES_SKIP_SETUP     set to 1/true/yes/on to skip the setup wizard
#   GORMES_GO_SHA256      optional expected SHA-256 for managed Go download
#   GORMES_INSTALL_VERBOSE set to 1/true/yes/on for verbose installer diagnostics
#
# Native Windows shells are not supported here. Use:
#   Invoke-WebRequest https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
#   Get-Content .\install.ps1
#   powershell -ExecutionPolicy Bypass -File .\install.ps1

set -eu

REPO_URL_SSH="${GORMES_REPO_URL_SSH:-git@github.com:TrebuchetDynamics/gormes-agent.git}"
REPO_URL_HTTPS="${GORMES_REPO_URL_HTTPS:-https://github.com/TrebuchetDynamics/gormes-agent.git}"
RELEASES_API_URL="${GORMES_RELEASES_API_URL:-https://api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest}"
RELEASES_DOWNLOAD_BASE="${GORMES_RELEASES_DOWNLOAD_BASE:-https://github.com/TrebuchetDynamics/gormes-agent/releases/download}"
RELEASES_LATEST_URL="${GORMES_RELEASES_LATEST_URL:-https://github.com/TrebuchetDynamics/gormes-agent/releases/latest}"
BRANCH="${GORMES_BRANCH:-main}"
GO_VERSION="${GORMES_GO_VERSION:-1.26.0}"
RESTART_GATEWAY="${GORMES_RESTART_GATEWAY:-auto}"
RUN_SETUP=true
VERBOSE="${GORMES_INSTALL_VERBOSE:-0}"
DRY_RUN=0
UNINSTALL=0
UNINSTALL_ARGS=""
LOCAL_SOURCE_DIR=""
FROM_SOURCE="${GORMES_INSTALL_FROM_SOURCE:-0}"
INSTALL_METHOD=""
INSTALL_METHOD_DETAIL=""
INSTALL_LOCK_DIR=""
OLD_BUILD_TAG=""
BUILD_TAG=""
SOURCE_ROOT_DIR=""
INSTALL_SOURCE_DESC=""
PREVIOUS_GATEWAY_PID=""
NEW_GATEWAY_PID=""
TMP_DIRS=""
TMP_DIR_COUNT=0
OS=""
DISTRO=""

# Detect non-interactive mode (e.g. curl | bash). Under `set -eu`, `read` on a
# closed stdin can silently abort the entire script. prompt_yes_no falls back
# to /dev/tty when stdin is a pipe so curl|bash users still get prompts.
if [ -t 0 ]; then
  IS_INTERACTIVE=true
else
  IS_INTERACTIVE=false
fi

log() {
  if [ "$#" -eq 0 ] || [ -z "$*" ]; then
    printf '\n' >&2
    return
  fi
  printf '%s\n' "$*" >&2
}
log_info() { printf '→ %s\n' "$*" >&2; }
log_success() { printf '✓ %s\n' "$*" >&2; }
log_warn() { printf '⚠ %s\n' "$*" >&2; }
log_error() { printf '✗ %s\n' "$*" >&2; }
fail() { log_error "$*"; exit 1; }
log_blue() { printf '\033[1;34m%s\033[0m\n' "$*" >&2; }
verbose() {
  [ "$VERBOSE" -eq 1 ] || return 0
  log "$@"
}

# prompt_yes_no QUESTION [DEFAULT]
# DEFAULT is "yes" or "no" (default: "no"). Returns 0 for yes, 1 for no.
# Reads from stdin when interactive, /dev/tty when stdin is a pipe (curl|bash),
# or returns the default when no terminal is reachable.
prompt_yes_no() {
  pyn_q="$1"
  pyn_default="${2:-no}"
  case "$pyn_default" in
    y|Y|yes|YES|true|1) pyn_suffix="[Y/n]" ;;
    *) pyn_suffix="[y/N]" ;;
  esac

  pyn_answer=""
  if [ "$IS_INTERACTIVE" = "true" ]; then
    printf '%s %s ' "$pyn_q" "$pyn_suffix" >&2
    IFS= read -r pyn_answer || pyn_answer=""
  elif (: < /dev/tty) >/dev/null 2>&1; then
    printf '%s %s ' "$pyn_q" "$pyn_suffix" > /dev/tty
    IFS= read -r pyn_answer < /dev/tty || pyn_answer=""
  fi

  case "$pyn_answer" in
    "")
      case "$pyn_default" in
        y|Y|yes|YES|true|1) return 0 ;;
        *) return 1 ;;
      esac
      ;;
    y|Y|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

print_banner() {
  log ""
  log_blue " ██████╗  ██████╗ ██████╗ ███╗   ███╗███████╗███████╗       █████╗  ██████╗ ███████╗███╗   ██╗████████╗"
  log_blue "██╔════╝ ██╔═══██╗██╔══██╗████╗ ████║██╔════╝██╔════╝      ██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝"
  log_blue "██║  ███╗██║   ██║██████╔╝██╔████╔██║█████╗  ███████╗█████╗███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║"
  log_blue "██║   ██║██║   ██║██╔══██╗██║╚██╔╝██║██╔══╝  ╚════██║╚════╝██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║"
  log_blue "╚██████╔╝╚██████╔╝██║  ██║██║ ╚═╝ ██║███████╗███████║      ██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║"
  log_blue " ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝╚══════╝╚══════╝      ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝"
  log_blue "Gormes Agent Installer"
  log_blue "An open source AI agent for Hermes-compatible runtime."
  log ""
}

usage() {
  cat <<'EOF'
Gormes Unix installer

Usage:
  install.sh [--branch NAME] [--home DIR] [--dir DIR] [--bin-dir DIR]
  install.sh --build                   # build from source instead of fetching release binary
  install.sh --local [--bin-dir DIR]
  install.sh --dry-run
  install.sh --uninstall [gormes uninstall flags]

Options:
  --branch NAME  Git branch to clone/update (default: main)
  --home DIR     Managed install home (default: $HOME/.gormes)
  --dir DIR      Managed source checkout directory
  --bin-dir DIR  Published command directory
                   default (non-root): $HOME/.local/bin
                   default (root Linux): /usr/local/bin
   --build, --from-source  Build gormes from source instead of downloading the pre-built
                 release binary from GitHub Releases. Slower but works for
                 unsupported platforms or pre-release branches.
                 (Env: GORMES_INSTALL_FROM_SOURCE=1)
  --local        Build from the current checkout instead of the managed
                 installer checkout
  --dry-run      Print the resolved plan without cloning, building, publishing,
                 or restarting the gateway
  --skip-setup   Skip the post-install setup wizard
  -v, --verbose  Print resolved paths, platform details, and step diagnostics
  --uninstall    Delegate to an existing "gormes uninstall" command and exit.
                 Flags after --uninstall are passed through, for example:
                 install.sh --uninstall --dry-run
                 install.sh --uninstall --dry-run=false --yes
  --restart-gateway auto|always|never
                 Restart a live gateway after update (default: auto)
  --no-restart   Alias for --restart-gateway never
  -h, --help     Show this help

Default installs fetch the latest signed release binary on supported main-branch
platforms. If a release asset is unavailable, verification fails, --from-source
is set, --local is used, or --branch is not main, the installer clones or
updates a managed source checkout, builds gormes locally, and publishes the
resulting command. Rerun this installer to update the command while keeping
Gormes a single Go binary with no Python or Node runtime.

Root Linux installs use an FHS command path like Hermes Agent:
/usr/local/bin/gormes, with data under $GORMES_INSTALL_HOME.
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

machine_name() {
  if [ -n "${GORMES_INSTALL_TEST_UNAME_M:-}" ]; then
    printf '%s\n' "$GORMES_INSTALL_TEST_UNAME_M"
    return
  fi
  uname -m
}

detect_os() {
  case "$(platform_name)" in
    Linux*)
      if is_termux; then
        OS="android"
        DISTRO="termux"
      else
        OS="linux"
        if [ -n "${GORMES_INSTALL_TEST_DISTRO:-}" ]; then
          DISTRO="$GORMES_INSTALL_TEST_DISTRO"
        elif [ -f /etc/os-release ]; then
          # shellcheck disable=SC1091
          . /etc/os-release
          DISTRO="${ID:-unknown}"
        else
          DISTRO="unknown"
        fi
      fi
      ;;
    Darwin*)
      OS="macos"
      DISTRO="macos"
      ;;
    MINGW*|MSYS*|CYGWIN*)
      OS="windows"
      DISTRO="windows"
      ;;
    *)
      OS="unknown"
      DISTRO="unknown"
      ;;
  esac
  log_success "Detected: ${OS} (${DISTRO})"
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
  # CRITICAL: scope the uninstall to the managed home dir resolved from
  # GORMES_INSTALL_HOME. Without this, an operator running install.sh
  # --uninstall from inside a sandbox (GORMES_INSTALL_HOME=/tmp/...) would
  # see the gormes binary use its default HOME-derived ~/.gormes path and
  # delete the operator's REAL install (live regression 2026-05-10:
  # confirmed data loss of ~/.gormes/.env, ~/.gormes/memory.db,
  # ~/.gormes/config.toml, ~/.local/bin/gormes during a sandbox uninstall
  # test). Exporting GORMES_HOME pins the uninstall blast radius to the
  # SAME directory tree install.sh manages; sandbox uninstalls now stay in
  # the sandbox.
  GORMES_HOME="$(managed_home_dir)"
  export GORMES_HOME
  # Mirror install.sh's default-to-apply UX: `install.sh` actually
  # installs by default, so `install.sh --uninstall` should actually
  # uninstall by default. Without this, the operator saw a dry-run
  # preview and assumed cleanup happened. Caller's `--dry-run` opt-in
  # still wins (preserved verbatim in "$@").
  apply_default=1
  for arg; do
    case "$arg" in
      --dry-run|--dry-run=*) apply_default=0; break ;;
    esac
  done
  if [ "$apply_default" -eq 1 ]; then
    log "running GORMES_HOME=${GORMES_HOME} ${uninstall_bin} uninstall --yes --dry-run=false $*"
    "$uninstall_bin" uninstall --yes --dry-run=false "$@"
  else
    log "running GORMES_HOME=${GORMES_HOME} ${uninstall_bin} uninstall $*"
    "$uninstall_bin" uninstall "$@"
  fi
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
  go_version_supported "$goversion" || fail "Go 1.26+ required; found ${goversion}"
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
    go1.2[6-9]*|go1.[3-9][0-9]*|go[2-9]*)
      return 0 ;;
    *)
      return 1 ;;
  esac
}

parse_args() {
  case "$VERBOSE" in
    1|true|TRUE|yes|YES|on|ON) VERBOSE=1 ;;
    *) VERBOSE=0 ;;
  esac

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
      --from-source|--build)
        FROM_SOURCE=1
        shift
        ;;
      --dry-run)
        DRY_RUN=1
        shift
        ;;
      --skip-setup)
        RUN_SETUP=false
        shift
        ;;
      -v|--verbose)
        VERBOSE=1
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
  case "${GORMES_SKIP_SETUP:-}" in
    1|true|TRUE|yes|YES|on|ON) RUN_SETUP=false ;;
  esac
}

ensure_base_prerequisites() {
  verbose "checking base prerequisites"
  need uname
  need mkdir
  need rm
  need ln
  need mv
  need cp
  need chmod
  need sleep

  check_platform
}

ensure_source_prerequisites() {
  verbose "checking source-build prerequisites"
  if is_termux; then
    ensure_termux_core_packages
  else
    ensure_go
    ensure_git
  fi
  check_node_optional
  check_ripgrep_optional
  check_ffmpeg_optional
}

ensure_prerequisites() {
  ensure_base_prerequisites
  ensure_source_prerequisites
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

  if ! has sudo; then
    fail "administrator permission is needed to install missing OS packages; install them manually or rerun with sudo available"
  fi

  # Passwordless sudo — just run.
  if sudo -n true 2>/dev/null; then
    sudo "$@"
    return
  fi

  # Password sudo with a TTY available — confirm, then read password from /dev/tty.
  if (: < /dev/tty) >/dev/null 2>&1; then
    log_info "sudo is needed to install missing OS packages."
    log_info "Gormes itself does not require or retain root access."
    if prompt_yes_no "Run: sudo $*" "yes"; then
      sudo "$@" < /dev/tty
      return
    fi
    fail "skipped sudo step; install required packages manually and rerun"
  fi

  fail "sudo password prompt is not available in this shell; install the required OS packages manually and rerun"
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
    # env keeps DEBIAN_FRONTEND/NEEDRESTART_MODE across the sudo barrier so
    # whiptail/needrestart dialogs don't block unattended (curl|bash) installs.
    run_privileged env DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a apt-get update
    run_privileged env DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a apt-get install -y "$@"
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
  log_info "Checking Git..."
  if has git; then
    git_version=$(git --version 2>/dev/null || true)
    set -- $git_version
    version="${3:-unknown}"
    log_success "Git ${version} found"
    return
  fi

  log_info "Git not found; attempting to install it"
  install_os_packages git || fail "Git is required and could not be installed automatically"
  has git || fail "Git install completed but git is still not on PATH"
  git_version=$(git --version 2>/dev/null || true)
  set -- $git_version
  version="${3:-unknown}"
  log_success "Git ${version} installed"
}

ensure_go() {
  log_info "Checking Go ${GO_VERSION}..."
  if has go; then
    goversion=$(current_go_version)
    if go_version_supported "$goversion"; then
      log_success "Go ${goversion} found"
      return
    fi
    log_info "found ${goversion}; installing managed Go ${GO_VERSION}"
  else
    log_info "Go not found; installing managed Go ${GO_VERSION}"
  fi

  install_managed_go
  check_go_version
  log_success "Go $(current_go_version) installed"
}

check_node_optional() {
  log_info "Checking Node.js (for browser tools)..."
  if has node; then
    node_version=$(node --version 2>/dev/null || true)
    log_success "Node.js ${node_version:-unknown} found"
    return
  fi
  log_warn "Node.js not found (browser-adjacent tools may be limited)"
}

check_ripgrep_optional() {
  log_info "Checking ripgrep (fast file search)..."
  if has rg; then
    rg_version=$(rg --version 2>/dev/null | sed -n '1p')
    log_success "${rg_version:-ripgrep unknown} found"
    return
  fi
  log_warn "ripgrep not found (file search will use slower fallbacks)"
  # Offer a per-distro install hint so operators don't have to guess the
  # package name. The package name is `ripgrep` everywhere except Alpine
  # (which calls the binary `rg` from a `ripgrep` package too) and macOS.
  case "${DISTRO:-}" in
    ubuntu|debian|raspbian|pop|linuxmint)
      log "  install:  sudo apt install ripgrep"
      ;;
    fedora|rhel|centos|rocky|almalinux)
      log "  install:  sudo dnf install ripgrep"
      ;;
    arch|manjaro|endeavouros)
      log "  install:  sudo pacman -S ripgrep"
      ;;
    alpine)
      log "  install:  sudo apk add ripgrep"
      ;;
    *)
      case "$(platform_name)" in
        Darwin*) log "  install:  brew install ripgrep" ;;
        *)       log "  install:  see https://github.com/BurntSushi/ripgrep#installation" ;;
      esac
      ;;
  esac
}

check_ffmpeg_optional() {
  log_info "Checking ffmpeg (TTS voice messages)..."
  if has ffmpeg; then
    ffmpeg_line=$(ffmpeg -version 2>/dev/null | sed -n '1p')
    set -- $ffmpeg_line
    version="${3:-unknown}"
    log_success "ffmpeg ${version} found"
    return
  fi
  log_warn "ffmpeg not found (voice/TTS media support may be limited)"
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

  # Enforce HTTPS-only with TLS 1.2+, retry transient failures, and bound
  # connection waits — the Go tarball download is the riskiest network step.
  if has curl; then
    curl -fsSL --proto '=https' --tlsv1.2 \
      --retry 3 --retry-delay 1 --retry-connrefused \
      -o "$out" "$url"
    return
  fi

  wget -q --https-only --secure-protocol=TLSv1_2 \
    --tries=3 --timeout=20 \
    -O "$out" "$url"
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

new_tmp_dir() {
  base="${TMPDIR:-/tmp}"
  base="${base%/}"
  if has mktemp; then
    dir=$(mktemp -d "${base}/gormes-install.XXXXXX") ||
      fail "could not create temporary directory under ${base}"
  else
    TMP_DIR_COUNT=$((TMP_DIR_COUNT + 1))
    dir="${base}/gormes-install.$$.$TMP_DIR_COUNT"
    mkdir -p "$dir" || fail "could not create temporary directory ${dir}"
  fi
  TMP_DIRS="${TMP_DIRS}${TMP_DIRS:+ }${dir}"
  printf '%s\n' "$dir"
}

# release_platform_arch maps the host platform to a published release-asset
# arch slug. Returns empty for unsupported platforms — caller MUST treat empty
# as "no published binary; fall back to source build".
#
# Supported asset slugs match the release matrix:
#   linux-amd64, linux-arm64, darwin-amd64, darwin-arm64,
#   windows-amd64, windows-arm64, android-arm64
release_platform_arch() {
  rpa_pn=$(platform_name)
  rpa_m=$(machine_name 2>/dev/null || printf 'unknown\n')
  case "$rpa_pn" in
    Linux)
      if is_termux; then
        case "$rpa_m" in
          aarch64|arm64) printf 'android-arm64\n' ;;
          *) printf '\n' ;;
        esac
      else
        case "$rpa_m" in
          x86_64|amd64) printf 'linux-amd64\n' ;;
          aarch64|arm64) printf 'linux-arm64\n' ;;
          *) printf '\n' ;;
        esac
      fi
      ;;
    Darwin)
      case "$rpa_m" in
        x86_64|amd64) printf 'darwin-amd64\n' ;;
        arm64) printf 'darwin-arm64\n' ;;
        *) printf '\n' ;;
      esac
      ;;
    *) printf '\n' ;;
  esac
}

# decide_install_method picks between binary-fetch (download a pre-built
# release artifact from GitHub Releases) and source-build (clone + go build).
# Sets globals INSTALL_METHOD and INSTALL_METHOD_DETAIL exactly once per run.
# Idempotent: re-running with the same args is a no-op.
#
# Decision rules (first match wins):
#   1. --local    -> source-build (operator wants their working tree).
#   2. --from-source / GORMES_INSTALL_FROM_SOURCE=1 -> source-build.
#   3. Branch is non-default (not "main") -> source-build (release binaries
#      are only published from main).
#   4. Host arch has no published asset -> source-build (with reason).
#   5. Otherwise -> binary-fetch (the new fast path).
decide_install_method() {
  if [ -n "$INSTALL_METHOD" ]; then
    return 0
  fi
  if [ -n "$LOCAL_SOURCE_DIR" ]; then
    INSTALL_METHOD="source-build"
    INSTALL_METHOD_DETAIL="--local: build from current checkout (${LOCAL_SOURCE_DIR})"
    return 0
  fi
  if [ "${FROM_SOURCE:-0}" -eq 1 ]; then
    INSTALL_METHOD="source-build"
    INSTALL_METHOD_DETAIL="--from-source flag set (build from cloned source instead of downloading release binary)"
    return 0
  fi
  if [ "$BRANCH" != "main" ]; then
    INSTALL_METHOD="source-build"
    INSTALL_METHOD_DETAIL="--branch ${BRANCH}: release binaries are only published from main"
    return 0
  fi
  dim_arch=$(release_platform_arch)
  if [ -z "$dim_arch" ]; then
    INSTALL_METHOD="source-build"
    INSTALL_METHOD_DETAIL="platform $(platform_name)/$(machine_name 2>/dev/null || printf unknown) has no published release asset"
    return 0
  fi
  INSTALL_METHOD="binary-fetch"
  INSTALL_METHOD_DETAIL="${dim_arch} from latest release (no Go toolchain or git clone needed)"
}

# fetch_release_binary downloads the latest release asset for the host
# platform, verifies its SHA-256 against the published .sha256 sidecar, and
# extracts the gormes binary into managed_bin_dir.
#
# Returns non-zero on any failure (network, missing asset, hash mismatch);
# caller is expected to fall back to source-build on failure.
fetch_release_binary() {
  frb_arch=$(release_platform_arch)
  if [ -z "$frb_arch" ]; then
    log "fetch_release_binary: no asset for platform $(platform_name)/$(machine_name); aborting"
    return 1
  fi

  # Resolve the latest release tag. Prefer api.github.com (versioned, durable),
  # but fall back to the public releases/latest redirect on github.com when the
  # API is blocked, throttled, or unreachable — that endpoint serves a plain
  # 302 to /releases/tag/<tag> and only requires github.com itself, which the
  # asset downloads need anyway.
  log_info "Resolving latest release tag from ${RELEASES_API_URL}"
  frb_api_body=""
  if has curl; then
    frb_api_body=$(curl -fsSL --connect-timeout 5 --max-time 20 \
      --retry 2 --retry-delay 1 --retry-connrefused \
      -H 'Accept: application/vnd.github+json' "$RELEASES_API_URL" 2>/dev/null) || frb_api_body=""
  elif has wget; then
    frb_api_body=$(wget -q --tries=2 --timeout=20 --header='Accept: application/vnd.github+json' \
      -O - "$RELEASES_API_URL" 2>/dev/null) || frb_api_body=""
  fi
  frb_tag=""
  if [ -n "$frb_api_body" ]; then
    frb_tag=$(printf '%s\n' "$frb_api_body" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
  fi
  if [ -z "$frb_tag" ]; then
    log_warn "GitHub API unreachable or unparseable; trying ${RELEASES_LATEST_URL} redirect"
    if has curl; then
      frb_tag=$(curl -fsLI --connect-timeout 5 --max-time 20 -o /dev/null \
        -w '%{url_effective}\n' "$RELEASES_LATEST_URL" 2>/dev/null \
        | sed -n 's|.*/releases/tag/\([^/[:space:]]\{1,\}\).*|\1|p' \
        | tail -1)
    elif has wget; then
      # wget -S prints response headers including Location: lines while following redirects.
      frb_tag=$(wget -S --tries=2 --timeout=20 -O /dev/null "$RELEASES_LATEST_URL" 2>&1 \
        | sed -n 's|.*Location:[[:space:]]*.*/releases/tag/\([^/[:space:]]\{1,\}\).*|\1|p' \
        | tail -1)
    fi
  fi
  if [ -z "$frb_tag" ]; then
    log "fetch_release_binary: could not resolve latest release tag (API and redirect both failed); aborting"
    return 1
  fi
  frb_ver="${frb_tag#v}"
  frb_asset="gormes-${frb_ver}-${frb_arch}.tar.gz"
  frb_url="${RELEASES_DOWNLOAD_BASE}/${frb_tag}/${frb_asset}"
  frb_sha_url="${frb_url}.sha256"

  frb_tmp=$(new_tmp_dir)
  log_info "Downloading ${frb_asset} (${frb_tag})"
  if has curl; then
    curl -fsSL --connect-timeout 10 --retry 3 --retry-delay 1 --retry-connrefused \
      -o "${frb_tmp}/${frb_asset}" "$frb_url" || {
        log "fetch_release_binary: download failed for ${frb_url}"
        return 1
      }
    curl -fsSL --connect-timeout 10 --retry 3 --retry-delay 1 --retry-connrefused \
      -o "${frb_tmp}/${frb_asset}.sha256" "$frb_sha_url" || {
        log "fetch_release_binary: download failed for ${frb_sha_url}"
        return 1
      }
  elif has wget; then
    wget -q --tries=3 --timeout=30 -O "${frb_tmp}/${frb_asset}" "$frb_url" || {
      log "fetch_release_binary: download failed for ${frb_url}"
      return 1
    }
    wget -q --tries=3 --timeout=30 -O "${frb_tmp}/${frb_asset}.sha256" "$frb_sha_url" || {
      log "fetch_release_binary: download failed for ${frb_sha_url}"
      return 1
    }
  else
    log "fetch_release_binary: neither curl nor wget available; aborting"
    return 1
  fi

  log_info "Verifying SHA-256 checksum"
  frb_expected=$(awk '{print $1}' "${frb_tmp}/${frb_asset}.sha256" 2>/dev/null)
  if [ -z "$frb_expected" ]; then
    log "fetch_release_binary: empty .sha256 file; aborting"
    return 1
  fi
  if has sha256sum; then
    frb_actual=$(sha256sum "${frb_tmp}/${frb_asset}" | awk '{print $1}')
  elif has shasum; then
    frb_actual=$(shasum -a 256 "${frb_tmp}/${frb_asset}" | awk '{print $1}')
  else
    log "fetch_release_binary: no SHA-256 utility (need sha256sum or shasum); aborting"
    return 1
  fi
  if [ "$frb_expected" != "$frb_actual" ]; then
    log "fetch_release_binary: SHA-256 mismatch (expected ${frb_expected}, got ${frb_actual}); aborting"
    return 1
  fi
  log_success "SHA-256 verified"

  log_info "Extracting ${frb_asset}"
  tar -xzf "${frb_tmp}/${frb_asset}" -C "$frb_tmp" || {
    log "fetch_release_binary: tar extract failed; aborting"
    return 1
  }
  # The release archive contains a top-level dir named after the asset slug,
  # e.g. gormes-0.1.06-linux-amd64/gormes. Look there first; fall back to a
  # flat layout for forward compatibility.
  frb_extracted_bin="${frb_tmp}/gormes-${frb_ver}-${frb_arch}/gormes"
  if [ ! -f "$frb_extracted_bin" ] && [ -f "${frb_tmp}/gormes" ]; then
    frb_extracted_bin="${frb_tmp}/gormes"
  fi
  if [ ! -f "$frb_extracted_bin" ]; then
    log "fetch_release_binary: extracted archive does not contain a gormes binary; aborting"
    return 1
  fi

  frb_bin_target="$(managed_bin_dir)/gormes"
  mkdir -p "$(parent_dir "$frb_bin_target")"
  mv -f "$frb_extracted_bin" "$frb_bin_target" || {
    log "fetch_release_binary: could not install binary at ${frb_bin_target}"
    return 1
  }
  chmod +x "$frb_bin_target"
  INSTALL_SOURCE_DESC="GitHub Releases (${frb_tag}/${frb_asset})"
  log_success "Installed gormes ${frb_ver} from release ${frb_tag} (${frb_arch})"
  return 0
}

clone_checkout() {
  checkout_dir="${SOURCE_ROOT_DIR:-$(managed_checkout_dir)}"
  mkdir -p "$(parent_dir "$checkout_dir")"

  log_info "Installing to ${checkout_dir}..."
  verbose "clone branch: ${BRANCH}"
  verbose "clone ssh url: ${REPO_URL_SSH}"
  verbose "clone https fallback: ${REPO_URL_HTTPS}"
  log_info "Trying SSH clone"
  if GIT_SSH_COMMAND="ssh -o BatchMode=yes -o ConnectTimeout=5" \
    git clone --branch "$BRANCH" "$REPO_URL_SSH" "$checkout_dir"; then
    log_success "Cloned via SSH"
    log_success "Repository ready"
    return
  fi

  log_info "SSH failed, trying HTTPS"
  rm -rf "$checkout_dir"
  git clone --branch "$BRANCH" "$REPO_URL_HTTPS" "$checkout_dir" ||
    fail "could not clone Gormes from SSH or HTTPS"
  log_success "Cloned via HTTPS"
  log_success "Repository ready"
}

ensure_checkout() {
  if [ -n "$LOCAL_SOURCE_DIR" ]; then
    if [ ! -f "$LOCAL_SOURCE_DIR/go.mod" ] || [ ! -d "$LOCAL_SOURCE_DIR/cmd/gormes" ]; then
      fail "--local must be run from a Gormes source checkout"
    fi
    SOURCE_ROOT_DIR="$LOCAL_SOURCE_DIR"
    INSTALL_SOURCE_DESC="$LOCAL_SOURCE_DIR"
    log "using local source checkout ${LOCAL_SOURCE_DIR}"
    return
  fi

  SOURCE_ROOT_DIR="$(managed_checkout_dir)"
  INSTALL_SOURCE_DESC="$SOURCE_ROOT_DIR"
  if [ -d "$SOURCE_ROOT_DIR" ]; then
    if [ ! -d "$SOURCE_ROOT_DIR/.git" ]; then
      fail "$SOURCE_ROOT_DIR exists but is not a git checkout; remove it or rerun with GORMES_INSTALL_DIR"
    fi
    update_checkout
    return
  fi

  clone_checkout
}

update_checkout() {
  checkout_dir="${SOURCE_ROOT_DIR:-$(managed_checkout_dir)}"

  log_info "Installing to ${checkout_dir}..."
  log_info "Existing Gormes checkout found; updating"
  (
    cd "$checkout_dir" || exit 1

    autostash_ref=""
    if [ -n "$(git status --porcelain)" ]; then
      stash_name="gormes-install-autostash-$$"
      log_info "Local changes detected; stashing before update"
      git stash push --include-untracked -m "$stash_name"
      autostash_ref="$(git rev-parse --verify refs/stash 2>/dev/null || true)"
    fi

    # Gormes is a public repo, so SSH access is never required to fetch updates.
    # If the existing origin is SSH (or any remote is unreachable), make the
    # SSH attempt fail fast and fall back to the public HTTPS URL rather than
    # hanging on a 2-minute TCP timeout.
    GIT_SSH_COMMAND="${GIT_SSH_COMMAND:-ssh -o BatchMode=yes -o ConnectTimeout=5}"
    export GIT_SSH_COMMAND

    if ! git fetch origin; then
      log_warn "git fetch origin failed; falling back to public HTTPS (${REPO_URL_HTTPS})"
      git fetch "$REPO_URL_HTTPS" "$BRANCH" || exit 1
      git checkout "$BRANCH"
      git merge --ff-only FETCH_HEAD || exit 1
    else
      git checkout "$BRANCH"
      git pull --ff-only origin "$BRANCH"
    fi

    if [ -n "$autostash_ref" ]; then
      log_info "Restoring stashed local changes"
      if git stash apply "$autostash_ref"; then
        git stash drop "$autostash_ref" >/dev/null
        log_warn "Local changes restored on top of updated checkout"
      else
        log_error "Update succeeded, but restoring local changes failed"
        log_info "Restore manually with: git stash apply $autostash_ref"
        exit 1
      fi
    fi
  ) || fail "could not update Gormes checkout ${checkout_dir}"
  log_success "Repository ready"
}

build_root_dir() {
  if [ -n "$SOURCE_ROOT_DIR" ]; then
    printf '%s\n' "$SOURCE_ROOT_DIR"
    return
  fi

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

  return 1
}

install_source_description() {
  if [ -n "$INSTALL_SOURCE_DESC" ]; then
    printf '%s\n' "$INSTALL_SOURCE_DESC"
    return
  fi
  build_root_dir 2>/dev/null || printf 'managed-checkout\n'
}

build_gormes() {
  build_bin="$(managed_bin_dir)/gormes"
  build_root=$(build_root_dir) || fail "could not find a Gormes Go module to build"
  cache_tag=$(git -C "$build_root" rev-parse --short HEAD 2>/dev/null || echo "unknown")
  BUILD_TAG="$cache_tag"
  verbose "build root: ${build_root}"
  verbose "build output: ${build_bin}"
  verbose "source commit: ${cache_tag}"

  if [ -x "$build_bin" ]; then
    cached_tag=""
    if [ -f "${build_bin}.build-tag" ]; then
      cached_tag=$(cat "${build_bin}.build-tag" 2>/dev/null || true)
    fi
    OLD_BUILD_TAG="$cached_tag"
    if [ "$cached_tag" = "$cache_tag" ]; then
      log_success "Gormes binary ready (${cache_tag})"
      return
    fi
    if [ -n "$cached_tag" ]; then
      log_info "source changed (${cached_tag} → ${cache_tag}); rebuilding"
    fi
  fi

  mkdir -p "$(managed_bin_dir)"
  # Embed git metadata and version so `gormes version` reports the real
  # values instead of compiled-in defaults.
  build_commit="$(git -C "$build_root" rev-parse --short HEAD 2>/dev/null || true)"
  [ -n "$build_commit" ] || build_commit="unknown"
  build_dirty="false"
  if ! git -C "$build_root" diff --quiet 2>/dev/null \
    || ! git -C "$build_root" diff --cached --quiet 2>/dev/null; then
    build_dirty="true"
  fi
  build_version="$(grep '^\s*var Version\s*=' "$build_root/cmd/gormes/version.go" 2>/dev/null | sed 's/.*"\(.*\)".*/\1/' || true)"
  [ -n "$build_version" ] || build_version="0.0.0"
  build_ldflags="-s -w -X main.Version=${build_version} -X main.GitCommit=${build_commit} -X main.GitDirty=${build_dirty}"
  log_info "Building gormes from ${build_root} (${cache_tag})"
  (
    cd "$build_root" || exit 1
    CGO_ENABLED=0 go build -trimpath -ldflags "$build_ldflags" -o "$build_bin" ./cmd/gormes
  ) || fail "go build failed"

  printf '%s\n' "$cache_tag" > "${build_bin}.build-tag"
  [ -x "$build_bin" ] || fail "build completed but ${build_bin} was not created"
  log_success "Gormes binary ready"
}

PATH_CONFIG_FILES=""
PATH_CONFIG_RESULT=""

write_path_to_shell_config() {
  wpsc_config="$1"
  wpsc_bin_dir="$2"
  wpsc_marker="# Gormes installer — added ${wpsc_bin_dir} to PATH"
  if grep -F "$wpsc_marker" "$wpsc_config" >/dev/null 2>&1; then
    return 1
  fi
  {
    printf '\n%s\n' "$wpsc_marker"
    printf 'export PATH="%s:$PATH"\n' "$wpsc_bin_dir"
  } >> "$wpsc_config"
  return 0
}

write_path_to_fish_config() {
  wpfc_config="$1"
  wpfc_bin_dir="$2"
  wpfc_marker="# Gormes installer — added ${wpfc_bin_dir} to PATH"
  if grep -F "$wpfc_marker" "$wpfc_config" >/dev/null 2>&1; then
    return 1
  fi
  {
    printf '\n%s\n' "$wpfc_marker"
    printf 'fish_add_path "%s"\n' "$wpfc_bin_dir"
  } >> "$wpfc_config"
  return 0
}

ensure_path_in_shell_config() {
  epsc_bin_dir="$1"
  PATH_CONFIG_FILES=""
  if path_contains_dir "$epsc_bin_dir"; then
    PATH_CONFIG_RESULT="already_on_path"
    return 0
  fi

  # iso-shellrc-leak: when the operator declared a sandbox bin dir
  # (GORMES_BIN_DIR / GORMES_PREFIX), do NOT mutate ~/.bashrc, ~/.profile,
  # ~/.zshrc, or fish config. The sandbox path will be reaped on the next
  # /tmp cleanup; persistent rc lines pointing at it become dangling cruft
  # the operator has to sed out by hand. Make the sandbox bin dir visible
  # to downstream steps in this same install run only.
  if sandbox_bin_dir_set; then
    PATH_CONFIG_RESULT="sandbox_skipped"
    PATH="${epsc_bin_dir}:${PATH}"
    export PATH
    log "skipping shell rc PATH edits (sandbox bin dir set via ${GORMES_BIN_DIR:+GORMES_BIN_DIR}${GORMES_PREFIX:+ GORMES_PREFIX}; respecting boundary — ~/.bashrc, ~/.profile, ~/.zshrc, fish config left untouched)"
    return 0
  fi

  PATH_CONFIG_RESULT="written"

  epsc_login_shell="${SHELL:-/bin/sh}"
  epsc_shell_name="${epsc_login_shell##*/}"

  case "$epsc_shell_name" in
    fish)
      epsc_fish_dir="$HOME/.config/fish"
      epsc_fish_config="${epsc_fish_dir}/config.fish"
      mkdir -p "$epsc_fish_dir"
      [ -f "$epsc_fish_config" ] || : > "$epsc_fish_config"
      if write_path_to_fish_config "$epsc_fish_config" "$epsc_bin_dir"; then
        PATH_CONFIG_FILES="$epsc_fish_config"
      fi
      ;;
    zsh)
      for epsc_config in "$HOME/.zshrc" "$HOME/.zprofile"; do
        [ -f "$epsc_config" ] || continue
        if write_path_to_shell_config "$epsc_config" "$epsc_bin_dir"; then
          PATH_CONFIG_FILES="${PATH_CONFIG_FILES}${PATH_CONFIG_FILES:+ }${epsc_config}"
        fi
      done
      if [ -z "$PATH_CONFIG_FILES" ] && [ ! -f "$HOME/.zshrc" ]; then
        : > "$HOME/.zshrc"
        if write_path_to_shell_config "$HOME/.zshrc" "$epsc_bin_dir"; then
          PATH_CONFIG_FILES="$HOME/.zshrc"
        fi
      fi
      ;;
    *)
      # bash, sh, dash, ksh, unknown — write to the standard bash files plus
      # ~/.profile (which login bash/sh sources on Ubuntu/Debian/WSL).
      for epsc_config in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile"; do
        [ -f "$epsc_config" ] || continue
        if write_path_to_shell_config "$epsc_config" "$epsc_bin_dir"; then
          PATH_CONFIG_FILES="${PATH_CONFIG_FILES}${PATH_CONFIG_FILES:+ }${epsc_config}"
        fi
      done
      ;;
  esac

  # FHS root layout on Linux: /etc/profile pathmunges /usr/local/bin into PATH
  # for login shells, but RHEL/Rocky/Alma 8+ non-login interactive root shells
  # (sudo -s, su, tmux, web terminals) skip /etc/profile and lose it. Probe
  # with `bash -i -c` so we only patch ~/.bashrc when actually needed.
  if is_root_linux_install && ! has_legacy_checkout && has bash; then
    if ! env -i HOME="$HOME" TERM="${TERM:-dumb}" PATH="/usr/bin:/bin" \
        bash -i -c 'command -v gormes' >/dev/null 2>&1; then
      epsc_rhel_marker="# Gormes installer — ensure /usr/local/bin is on PATH (RHEL non-login shells)"
      epsc_bashrc="$HOME/.bashrc"
      [ -f "$epsc_bashrc" ] || : > "$epsc_bashrc"
      if ! grep -F "$epsc_rhel_marker" "$epsc_bashrc" >/dev/null 2>&1; then
        {
          printf '\n%s\n' "$epsc_rhel_marker"
          printf 'export PATH="/usr/local/bin:$PATH"\n'
        } >> "$epsc_bashrc"
        PATH_CONFIG_FILES="${PATH_CONFIG_FILES}${PATH_CONFIG_FILES:+ }${epsc_bashrc}"
      fi
    fi
  fi

  # Make the bin dir visible to downstream steps in this same install run.
  PATH="${epsc_bin_dir}:${PATH}"
  export PATH

  if [ -n "$PATH_CONFIG_FILES" ]; then
    log_success "added ${epsc_bin_dir} to PATH in: ${PATH_CONFIG_FILES}"
  fi
}

publish_command() {
  bin_dir=$(pick_bin_dir)
  build_bin="$(managed_bin_dir)/gormes"
  published_bin="${bin_dir}/gormes"

  log_info "Setting up gormes command"
  verbose "publish source: ${build_bin}"
  verbose "publish target: ${published_bin}"
  publish_built_binary "$build_bin" "$published_bin"
  update_active_command "$build_bin" "$published_bin"
  ensure_path_in_shell_config "$bin_dir"
  log_success "gormes command ready"
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

# sandbox_bin_dir_set returns true (exit 0) when the operator explicitly set
# GORMES_BIN_DIR or GORMES_PREFIX. When true, the installer treats the
# resolved bin dir as an authoritative isolation boundary and must not reach
# outside it (e.g., must not overwrite an existing gormes binary discovered
# elsewhere on PATH such as ~/.local/bin/gormes). Closes iso-bin-hijack.
sandbox_bin_dir_set() {
  [ -n "${GORMES_BIN_DIR:-}" ] || [ -n "${GORMES_PREFIX:-}" ]
}

update_active_command() {
  build_bin="$1"
  published_bin="$2"
  if sandbox_bin_dir_set; then
    log "skipping active PATH command update (sandbox bin dir set via ${GORMES_BIN_DIR:+GORMES_BIN_DIR}${GORMES_PREFIX:+ GORMES_PREFIX}; respecting boundary)"
    return 0
  fi
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

  verbose "verifying published command: ${published_bin}"
  [ -x "$published_bin" ] || fail "published command is not executable: ${published_bin}"
  "$published_bin" version >/dev/null 2>&1 || fail "verification failed: ${published_bin} version"

  active_bin=$(active_command_path)
  if [ -n "$active_bin" ]; then
    verbose "verifying active PATH command: ${active_bin}"
    "$active_bin" version >/dev/null 2>&1 || fail "verification failed: active PATH command ${active_bin} version"
  fi

  build_root=$(build_root_dir 2>/dev/null || true)
  if [ -n "$build_root" ] && [ -f "$build_root/go.mod" ]; then
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

setup_tty_available() {
  case "${GORMES_INSTALL_TEST_HAS_TTY:-}" in
    1) return 0 ;;
    0) return 1 ;;
  esac
  (: < /dev/tty) >/dev/null 2>&1
}

run_setup_wizard() {
  if [ "$RUN_SETUP" = "false" ]; then
    log_info "Skipping setup wizard (--skip-setup)"
    return 0
  fi

  if ! setup_tty_available; then
    log_info "Setup wizard skipped (no terminal available)."
    log "Run 'gormes setup' after install."
    return 0
  fi

  published_bin="$(pick_bin_dir)/gormes"
  [ -x "$published_bin" ] || fail "setup wizard could not find installed gormes command: ${published_bin}"

  log ""
  log_info "Starting setup wizard"
  log ""
  if [ "${GORMES_INSTALL_TEST_HAS_TTY:-}" = "1" ]; then
    "$published_bin" setup || fail "setup wizard failed"
    return 0
  fi
  "$published_bin" setup < /dev/tty || fail "setup wizard failed"
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
  source=$(install_source_description)
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
  log "  source: $(install_source_description)"
  log "  config: $(managed_home_dir)/config.toml"
  active_bin=$(active_command_path)
  if [ -n "$active_bin" ] && [ "$active_bin" != "$published_bin" ] && ! same_binary "$active_bin" "$published_bin"; then
    log "  active: ${active_bin} (older install; run 'hash -r' or restart shell)"
  fi

  if path_contains_dir "$bin_dir"; then
    log "  PATH:   ${bin_dir} ✓"
  elif [ -n "${PATH_CONFIG_FILES:-}" ]; then
    log "  PATH:   ${bin_dir} (added to ${PATH_CONFIG_FILES})"
    log ""
    case "${SHELL##*/}" in
      zsh)  log "Reload your shell to pick up the new PATH: source ~/.zshrc" ;;
      bash) log "Reload your shell to pick up the new PATH: source ~/.bashrc" ;;
      fish) log "Reload your shell to pick up the new PATH: source ~/.config/fish/config.fish" ;;
      *)    log "Reload your shell or open a new terminal to pick up the new PATH." ;;
    esac
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
  if sandbox_bin_dir_set; then
    log "skipping system service install (sandbox bin dir set via ${GORMES_BIN_DIR:+GORMES_BIN_DIR}${GORMES_PREFIX:+ GORMES_PREFIX}; respecting boundary — ~/.config/systemd/user/ and ~/Library/LaunchAgents/ left untouched)"
    return 0
  fi
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
# RestartSec=30 + StartLimitIntervalSec=300 + StartLimitBurst=5 stop a
# misconfigured gateway (e.g. missing [hermes].endpoint) from crash-looping
# indefinitely: 5 restarts in 5 minutes trips the burst limit.
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
ExecStart=${build_bin} gateway
ExecReload=${build_bin} gateway stop && ${build_bin} gateway
Restart=on-failure
RestartSec=30
Environment=GORMES_HOME=$(managed_home_dir)

[Install]
WantedBy=default.target
SYSTEMDUNIT

  systemctl --user daemon-reload
  # Skip auto-enable when the wizard was skipped: enabling a unit that
  # requires [hermes].endpoint without configuring it first would crash-loop
  # the gateway on next login. Operators who actually want auto-start can
  # run `systemctl --user enable --now gormes-gateway` once setup completes.
  if [ "$RUN_SETUP" = "false" ]; then
    log ""
    log "systemd user service file installed (NOT auto-enabled under --skip-setup):"
    log "  ${service_file}"
    log "After configuring [hermes].endpoint, enable with:"
    log "  systemctl --user enable --now gormes-gateway"
    log "Then check:"
    log "  systemctl --user status gormes-gateway"
    log "  journalctl --user -u gormes-gateway -f"
    return 0
  fi

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

print_install_plan_body() {
  decide_install_method
  log "  branch: ${BRANCH}"
  log "  install_method: ${INSTALL_METHOD} (${INSTALL_METHOD_DETAIL})"
  if [ "$INSTALL_METHOD" = "binary-fetch" ]; then
    log "  source: github releases (no git clone, no Go toolchain)"
  elif [ -n "$LOCAL_SOURCE_DIR" ]; then
    log "  source: ${LOCAL_SOURCE_DIR}"
  else
    log "  source: managed git checkout of ${BRANCH}"
    log "  checkout: $(managed_checkout_dir)"
  fi
  log "  install_home: $(managed_home_dir)"
  log "  managed_binary: $(managed_bin_dir)/gormes"
  log "  published_binary: $(pick_bin_dir)/gormes"
  if sandbox_bin_dir_set; then
    log "  update_active_path_command: skipped (sandbox bin dir set via ${GORMES_BIN_DIR:+GORMES_BIN_DIR}${GORMES_PREFIX:+ GORMES_PREFIX}; respecting boundary)"
    log "  edit_shell_rc_files: skipped (sandbox bin dir set; ~/.bashrc, ~/.profile, ~/.zshrc, fish config left untouched)"
    log "  install_system_service: skipped (sandbox bin dir set; ~/.config/systemd/user/ and ~/Library/LaunchAgents/ left untouched)"
  else
    log "  update_active_path_command: yes (default install; will adopt any existing gormes on PATH)"
    log "  edit_shell_rc_files: yes (writes export PATH lines to ~/.bashrc, ~/.profile, or shell-appropriate config when bin dir is not already on PATH)"
    if [ "$RUN_SETUP" = "false" ]; then
      log "  install_system_service: yes (writes ~/.config/systemd/user/gormes-gateway.service on Linux with systemctl --user, or ~/Library/LaunchAgents/com.gormes.gateway.plist on macOS; not auto-enabled under --skip-setup — run \`systemctl --user enable --now gormes-gateway\` after configuring [hermes].endpoint)"
    else
      log "  install_system_service: yes (writes ~/.config/systemd/user/gormes-gateway.service on Linux with systemctl --user, or ~/Library/LaunchAgents/com.gormes.gateway.plist on macOS)"
    fi
  fi
  log "  restart_gateway: ${RESTART_GATEWAY}"
  log "  setup_wizard: ${RUN_SETUP}"
}

print_dry_run() {
  log "dry run"
  print_install_plan_body
}

print_install_plan() {
  log "install plan"
  print_install_plan_body
  log ""
}

yes_no() {
  if "$@" >/dev/null 2>&1; then
    printf 'yes\n'
  else
    printf 'no\n'
  fi
}

print_verbose_plan() {
  decide_install_method
  active_bin=$(active_command_path)
  [ -n "$active_bin" ] || active_bin="<none>"
  log "resolved install plan"
  log "  verbose: true"
  log "  platform: $(platform_name)"
  log "  arch: $(machine_name 2>/dev/null || printf unknown)"
  log "  termux: $(yes_no is_termux)"
  log "  root_linux_install: $(yes_no is_root_linux_install)"
  log "  effective_uid: $(effective_uid)"
  log "  branch: ${BRANCH}"
  log "  install_method: ${INSTALL_METHOD}"
  log "  install_method_reason: ${INSTALL_METHOD_DETAIL}"
  if [ "$INSTALL_METHOD" = "binary-fetch" ]; then
    log "  source_mode: github-releases"
    log "  release_arch: $(release_platform_arch)"
    log "  release_api: ${RELEASES_API_URL}"
  elif [ -n "$LOCAL_SOURCE_DIR" ]; then
    log "  source_mode: local"
    log "  source: ${LOCAL_SOURCE_DIR}"
  else
    log "  source_mode: managed-checkout"
    log "  checkout: $(managed_checkout_dir)"
  fi
  log "  install_home: $(managed_home_dir)"
  log "  managed_binary: $(managed_bin_dir)/gormes"
  log "  published_binary: $(pick_bin_dir)/gormes"
  log "  active_command: ${active_bin}"
  if sandbox_bin_dir_set; then
    log "  update_active_path_command: skipped (sandbox bin dir set via ${GORMES_BIN_DIR:+GORMES_BIN_DIR}${GORMES_PREFIX:+ GORMES_PREFIX}; respecting boundary)"
    log "  edit_shell_rc_files: skipped (sandbox bin dir set; ~/.bashrc, ~/.profile, ~/.zshrc, fish config left untouched)"
    log "  install_system_service: skipped (sandbox bin dir set; ~/.config/systemd/user/ and ~/Library/LaunchAgents/ left untouched)"
  else
    log "  update_active_path_command: yes (default install; will adopt any existing gormes on PATH)"
    log "  edit_shell_rc_files: yes (writes export PATH lines to ~/.bashrc, ~/.profile, or shell-appropriate config when bin dir is not already on PATH)"
    if [ "$RUN_SETUP" = "false" ]; then
      log "  install_system_service: yes (writes ~/.config/systemd/user/gormes-gateway.service on Linux with systemctl --user, or ~/Library/LaunchAgents/com.gormes.gateway.plist on macOS; not auto-enabled under --skip-setup — run \`systemctl --user enable --now gormes-gateway\` after configuring [hermes].endpoint)"
    else
      log "  install_system_service: yes (writes ~/.config/systemd/user/gormes-gateway.service on Linux with systemctl --user, or ~/Library/LaunchAgents/com.gormes.gateway.plist on macOS)"
    fi
  fi
  log "  restart_gateway: ${RESTART_GATEWAY}"
  log "  setup_wizard: ${RUN_SETUP}"
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

prepare_gormes_binary() {
  decide_install_method
  if [ "$INSTALL_METHOD" = "binary-fetch" ]; then
    if fetch_release_binary; then
      return
    fi
    log "binary-fetch failed; falling back to source build"
    INSTALL_METHOD="source-build"
    INSTALL_METHOD_DETAIL="binary-fetch failed at runtime; fallback to source build"
  fi

  ensure_source_prerequisites
  ensure_checkout
  build_gormes
}

main() {
  parse_args "$@"
  if [ "$VERBOSE" -eq 1 ]; then
    print_verbose_plan
  fi
  if [ "$UNINSTALL" -eq 1 ]; then
    # shellcheck disable=SC2086
    run_uninstall $UNINSTALL_ARGS
    return
  fi
  if [ "$DRY_RUN" -eq 1 ]; then
    print_dry_run
    return
  fi
  print_banner
  detect_os
  if [ "$VERBOSE" -ne 1 ]; then
    print_install_plan
  fi
  acquire_install_lock
  PREVIOUS_GATEWAY_PID=$(running_gateway_pid 2>/dev/null || true)
  ensure_base_prerequisites
  prepare_gormes_binary
  publish_command
  bootstrap_config
  verify_install
  run_setup_wizard
  restart_gateway_if_running "$PREVIOUS_GATEWAY_PID"
  append_install_ledger
  print_summary
}

if [ "${GORMES_INSTALL_TEST_MODE:-}" != "1" ]; then
  main "$@"
fi
