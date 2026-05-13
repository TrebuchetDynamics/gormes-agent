#!/usr/bin/env bash
# tests/install/e2e.sh — End-to-end probes for the install.sh shell script.
#
# Why this exists: Go tests under internal/installtest/ assert installer
# *behaviors* in isolation, but nothing actually invokes install.sh end-to-end
# against a sandbox HOME. This runner does, so regressions in plan accuracy,
# binary-fetch, the GitHub API redirect fallback, the SSH-origin update
# fallback, and the uninstall lifecycle are caught before a tagged release
# breaks first-ten-minutes for operators.
#
# Sandbox isolation: every case runs against a unique /tmp/gormes-install-e2e/
# sandbox with GORMES_INSTALL_HOME + GORMES_BIN_DIR set. Cases that touch
# production state explicitly diff pre/post and fail loudly if they do.
#
# Usage:
#   tests/install/e2e.sh                # run all cases
#   tests/install/e2e.sh <case-name>    # run a single case (no "case_" prefix)
#   GORMES_INSTALL_E2E_KEEP=1 …         # keep sandbox dirs after success
#
# Exit code is 0 only if every selected case passed.

set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
INSTALL_SH="$REPO_ROOT/install.sh"

if [ ! -f "$INSTALL_SH" ]; then
  echo "error: install.sh not found at $INSTALL_SH" >&2
  exit 2
fi

TS=$(date -u +%Y%m%dT%H%M%SZ)
SANDBOX_ROOT="${GORMES_INSTALL_E2E_ROOT:-/tmp/gormes-install-e2e/$TS}"
mkdir -p "$SANDBOX_ROOT/tmp"

# Pin TMPDIR to the sandbox so curl/go-build/mktemp don't fight whatever the
# user's $TMPDIR happens to be (it might be on a low-space partition or a
# shared cache dir polluted by other tools). Sandbox tmp lives under /tmp
# tmpfs and is GC'd with the rest of SANDBOX_ROOT.
export TMPDIR="$SANDBOX_ROOT/tmp"

PASS=0
FAIL=0
FAILED_CASES=""

# ----- assertions ------------------------------------------------------------

assert_eq() {
  local actual=$1 expected=$2 label=${3:-value}
  if [ "$actual" != "$expected" ]; then
    printf '    ! %s: expected=%q actual=%q\n' "$label" "$expected" "$actual" >&2
    return 1
  fi
}

assert_file_exists() {
  local path=$1
  if [ ! -e "$path" ]; then
    printf '    ! missing: %s\n' "$path" >&2
    return 1
  fi
}

assert_not_exists() {
  local path=$1
  if [ -e "$path" ]; then
    printf '    ! still exists: %s\n' "$path" >&2
    return 1
  fi
}

assert_executable() {
  local path=$1
  if [ ! -x "$path" ]; then
    printf '    ! not executable: %s\n' "$path" >&2
    return 1
  fi
}

assert_grep() {
  local pat=$1 file=$2
  if ! grep -qE "$pat" "$file" 2>/dev/null; then
    printf '    ! pattern not found: %s\n      in: %s\n' "$pat" "$file" >&2
    return 1
  fi
}

assert_no_grep() {
  local pat=$1 file=$2
  if grep -qE "$pat" "$file" 2>/dev/null; then
    printf '    ! pattern unexpectedly present: %s\n      in: %s\n' "$pat" "$file" >&2
    grep -nE "$pat" "$file" | head -5 >&2
    return 1
  fi
}

# ----- production-state capture/diff -----------------------------------------
#
# Each case can opt in via capture_prestate / assert_no_production_state_changes
# to fail loudly if the installer touches anything outside the sandbox.

capture_state() {
  local where=$1
  local out="$SANDBOX/$where"
  mkdir -p "$out"
  ls -la "$HOME/.local/bin/gormes" 2>/dev/null > "$out/local-bin-gormes.txt" || true
  grep -nE "GORMES|gormes" "$HOME/.bashrc" "$HOME/.profile" "$HOME/.zshrc" 2>/dev/null \
    > "$out/shell-rc.txt" || true
  ls -la "$HOME/.config/systemd/user/gormes-gateway.service" 2>/dev/null \
    > "$out/systemd-file.txt" || true
  systemctl --user is-enabled gormes-gateway 2>/dev/null > "$out/systemd-enabled.txt" || true
  stat -c '%y %n' "$HOME/.gormes/config.toml" "$HOME/.gormes/auth.json" 2>/dev/null \
    > "$out/gormes-home.txt" || true
  crontab -l 2>/dev/null | grep -E "gormes-codexu|gormes-opencode" \
    > "$out/cron.txt" || true
}

capture_prestate()  { capture_state pre-state;  }
capture_poststate() { capture_state post-state; }

assert_no_production_state_changes() {
  local failed=0 surface
  for surface in local-bin-gormes shell-rc systemd-file systemd-enabled gormes-home cron; do
    if ! diff -q "$SANDBOX/pre-state/$surface.txt" \
                 "$SANDBOX/post-state/$surface.txt" >/dev/null 2>&1; then
      printf '    ! production state changed: %s\n' "$surface" >&2
      diff "$SANDBOX/pre-state/$surface.txt" \
           "$SANDBOX/post-state/$surface.txt" >&2 || true
      failed=1
    fi
  done
  [ "$failed" -eq 0 ]
}

# ----- test cases ------------------------------------------------------------
#
# Each case runs in a subshell with set -e, so any failed assertion stops the
# case immediately. The case function reads $SANDBOX (its private dir) and
# uses $INSTALL_SH for the script under test.

case_plan_dryrun_inside_sandbox() {
  local log="$SANDBOX/logs/dryrun.log"

  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --dry-run > "$log" 2>&1

  assert_grep "install_home: $SANDBOX/home" "$log"
  assert_grep "managed_binary: $SANDBOX/home/bin/gormes" "$log"
  assert_grep "published_binary: $SANDBOX/bin/gormes" "$log"
  assert_grep "update_active_path_command: skipped" "$log"
  assert_grep "edit_shell_rc_files: skipped" "$log"
  assert_grep "install_system_service: skipped" "$log"
  assert_grep "restart_gateway: never" "$log"

  # No plan line should announce writes outside the sandbox. Guard the three
  # production paths the installer normally edits.
  assert_no_grep "managed_binary: $HOME/" "$log"
  assert_no_grep "published_binary: $HOME/" "$log"
  assert_no_grep "install_home: $HOME/" "$log"
}

case_binary_fetch_happy_path() {
  capture_prestate

  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  GORMES_INSTALL_VERBOSE=1 \
  sh "$INSTALL_SH" --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  assert_grep "Resolving latest release tag" "$SANDBOX/logs/install.log"
  assert_grep "Installed gormes" "$SANDBOX/logs/install.log"
  assert_executable "$SANDBOX/bin/gormes"

  local ver
  ver=$(GORMES_HOME="$SANDBOX/home" "$SANDBOX/bin/gormes" version 2>&1)
  printf '%s\n' "$ver" > "$SANDBOX/logs/version.log"
  assert_grep '^gormes ' "$SANDBOX/logs/version.log"

  capture_poststate
  assert_no_production_state_changes
}

case_api_failure_redirect_fallback() {
  # Point the API URL at a DNS-unresolvable host so the first curl/wget call
  # fails fast. The fix should then resolve the latest tag via the public
  # releases/latest redirect on github.com and complete the install.
  GORMES_RELEASES_API_URL="https://api.github.invalid/repos/TrebuchetDynamics/gormes-agent/releases/latest" \
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  assert_grep "GitHub API unreachable.*releases/latest redirect" "$SANDBOX/logs/install.log"
  assert_grep "Installed gormes" "$SANDBOX/logs/install.log"
  assert_executable "$SANDBOX/bin/gormes"
}

case_hash_mismatch_aborts() {
  # Spin up a local HTTP server that serves a wrong tarball + a .sha256
  # sidecar whose hash doesn't match it. Point GORMES_RELEASES_DOWNLOAD_BASE
  # at that server (the tag resolver still hits the real api/redirect, which
  # is fine — we just intercept the download step). install.sh must detect
  # the mismatch and fall back to source-build. Operators must never get a
  # silently-corrupted binary, so this contract is load-bearing.
  if ! command -v python3 >/dev/null 2>&1; then
    printf '    ! python3 required for this case\n' >&2
    return 1
  fi

  local arch
  case $(uname -s) in
    Linux)
      case $(uname -m) in
        x86_64|amd64) arch="linux-amd64" ;;
        aarch64|arm64) arch="linux-arm64" ;;
        *) printf '    ! unsupported test arch: %s\n' "$(uname -m)" >&2; return 1 ;;
      esac ;;
    Darwin)
      case $(uname -m) in
        x86_64|amd64) arch="darwin-amd64" ;;
        arm64)        arch="darwin-arm64" ;;
        *) printf '    ! unsupported test arch: %s\n' "$(uname -m)" >&2; return 1 ;;
      esac ;;
    *) printf '    ! unsupported test platform: %s\n' "$(uname -s)" >&2; return 1 ;;
  esac

  # Resolve the latest release tag via the public redirect (same path the
  # installer uses when api.github.com is unreachable).
  local tag
  tag=$(curl -fsLI --connect-timeout 5 --max-time 20 -o /dev/null \
        -w '%{url_effective}\n' \
        https://github.com/TrebuchetDynamics/gormes-agent/releases/latest 2>/dev/null \
      | sed -n 's|.*/releases/tag/\([^/[:space:]]\{1,\}\).*|\1|p' | tail -1)
  if [ -z "$tag" ]; then
    printf '    ! could not resolve latest release tag\n' >&2
    return 1
  fi
  local ver="${tag#v}"
  local asset="gormes-${ver}-${arch}.tar.gz"

  local webroot="$SANDBOX/web"
  mkdir -p "$webroot/$tag"
  printf 'this is not a gormes release tarball\n' > "$webroot/$tag/$asset"
  # Plausible hex string that obviously won't match the asset content.
  printf 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef  %s\n' \
    "$asset" > "$webroot/$tag/$asset.sha256"

  # Pre-allocate a free port (parsing python http.server's startup line is
  # brittle: it's block-buffered when redirected and may flush after we'd
  # already given up). Race window between bind/close and the server binding
  # is microscopic on a localhost test machine.
  local port
  port=$(python3 -c 'import socket
s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
  if [ -z "$port" ]; then
    printf '    ! could not pre-allocate free port\n' >&2
    return 1
  fi

  python3 -m http.server "$port" --bind 127.0.0.1 --directory "$webroot" \
    > "$SANDBOX/logs/httpd.log" 2>&1 &
  local httpd_pid=$!
  # `|| true` after each command: under `set -e` in the subshell, both `kill`
  # (when the pid is already gone) and `wait` (when the process exited with
  # signal 143) return non-zero and would abort the trap action mid-flow,
  # leaving wait's 143 as the subshell exit and producing a spurious FAIL
  # after every assertion passed. Trailing `; true` alone wasn't enough —
  # set -e caused the action to abort *before* reaching `true`.
  trap "kill $httpd_pid 2>/dev/null || true; wait $httpd_pid 2>/dev/null || true" EXIT

  local ready=0 i
  for i in 1 2 3 4 5 6 7 8 9 10; do
    if curl -fsS --max-time 1 "http://127.0.0.1:$port/" >/dev/null 2>&1; then
      ready=1
      break
    fi
    sleep 0.2
  done
  if [ "$ready" -eq 0 ]; then
    printf '    ! local httpd not responding on port %s within 2s\n' "$port" >&2
    return 1
  fi

  # Bad SSH/HTTPS git URLs so the source-build fallback fails fast (DNS
  # NXDOMAIN). We don't care whether source-build succeeds — the contract
  # under test is "SHA-256 mismatch is detected and binary-fetch aborts."
  GORMES_RELEASES_DOWNLOAD_BASE="http://127.0.0.1:$port" \
  GORMES_REPO_URL_SSH="git@invalid.invalid:foo/bar.git" \
  GORMES_REPO_URL_HTTPS="https://invalid.invalid/foo/bar.git" \
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1 || true

  assert_grep "SHA-256 mismatch" "$SANDBOX/logs/install.log"
  assert_grep "binary-fetch failed; falling back to source build" "$SANDBOX/logs/install.log"
  # Mismatched asset must never reach managed_bin_dir.
  assert_not_exists "$SANDBOX/home/bin/gormes"
}

case_no_go_toolchain_binary_fetch() {
  # Operator guarantee: a host without Go installed must still be able to
  # install gormes via the binary-fetch path. ensure_go() is only invoked
  # by ensure_source_prerequisites, which runs only when binary-fetch fails.
  #
  # Stripping dirs containing `go` from PATH is unsafe on Ubuntu: apt's
  # golang-1.22 installs symlinks at /bin/go and /usr/bin/go, so naive
  # stripping nukes sh/curl/tar. Build a curated shim dir with every tool
  # binary-fetch needs, set PATH to just that dir, and confirm `command -v
  # go` resolves to nothing.
  capture_prestate

  local shim="$SANDBOX/path-no-go"
  mkdir -p "$shim"
  # Required base (need) + binary-fetch deps + commonly-invoked coreutils.
  # ripgrep/node/ffmpeg are optional — install.sh logs warnings if absent.
  local needed tool real missing=""
  needed=(
    sh bash uname mkdir rm ln mv cp chmod sleep
    curl wget tar gzip gunzip awk grep sed head tail cat tr cut sort
    id hostname printf date dirname basename true false test '['
    sha256sum shasum stat readlink which env mktemp file
    git tput touch chown find xargs ps kill wait
  )
  for tool in "${needed[@]}"; do
    real=$(command -v "$tool" 2>/dev/null || true)
    if [ -n "$real" ]; then
      ln -sf "$real" "$shim/$tool"
    else
      missing="$missing $tool"
    fi
  done

  if PATH="$shim" command -v go >/dev/null 2>&1; then
    printf '    ! go is still resolvable on shim PATH (bug in case): %s\n' \
      "$(PATH=$shim command -v go)" >&2
    return 1
  fi

  # GOROOT/GOPATH unset for extra safety: a stale GOROOT could make `go`
  # appear "installed" via $GOROOT/bin even if not on PATH.
  PATH="$shim" \
  GOROOT="" \
  GOPATH="" \
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  # Took binary-fetch path; ensure_go was never invoked.
  assert_grep "Installed gormes" "$SANDBOX/logs/install.log"
  assert_no_grep "Checking Go" "$SANDBOX/logs/install.log"
  assert_no_grep "installing managed Go" "$SANDBOX/logs/install.log"

  assert_executable "$SANDBOX/bin/gormes"
  PATH="$shim" GORMES_HOME="$SANDBOX/home" \
    "$SANDBOX/bin/gormes" version > "$SANDBOX/logs/version.log" 2>&1
  assert_grep '^gormes ' "$SANDBOX/logs/version.log"

  capture_poststate
  assert_no_production_state_changes
}

case_ssh_origin_update_fallback() {
  # Pre-seed a real checkout via HTTPS so we have history + a valid .git dir.
  if ! git clone --depth 50 --branch main \
      https://github.com/TrebuchetDynamics/gormes-agent.git \
      "$SANDBOX/home/gormes-agent" > "$SANDBOX/logs/seed.log" 2>&1; then
    printf '    ! seed clone failed (no public HTTPS access?); see %s\n' \
      "$SANDBOX/logs/seed.log" >&2
    return 1
  fi
  # Hijack origin to SSH so a naive `git fetch origin` would hit port 22.
  git -C "$SANDBOX/home/gormes-agent" remote set-url origin \
    git@github.com:TrebuchetDynamics/gormes-agent.git

  # Source install.sh in TEST_MODE so we can call update_checkout directly
  # without going through the full install (which would need a Go toolchain).
  # GIT_SSH_COMMAND=false makes every SSH attempt return non-zero instantly,
  # which is what we want: it proves the HTTPS fallback runs regardless of
  # whether the test machine has SSH keys for github.com.
  (
    set +e
    export GORMES_INSTALL_TEST_MODE=1
    # shellcheck source=/dev/null
    . "$INSTALL_SH"
    export SOURCE_ROOT_DIR="$SANDBOX/home/gormes-agent"
    export GORMES_INSTALL_HOME="$SANDBOX/home"
    export GORMES_BIN_DIR="$SANDBOX/bin"
    export GIT_SSH_COMMAND=false
    update_checkout > "$SANDBOX/logs/update.log" 2>&1
  )
  local rc=$?
  assert_eq "$rc" 0 "update_checkout exit code"

  assert_grep "git fetch origin failed; falling back to public HTTPS" \
    "$SANDBOX/logs/update.log"
  assert_grep "Repository ready" "$SANDBOX/logs/update.log"

  # The fallback must not silently rewrite the user's origin remote.
  local origin_url
  origin_url=$(git -C "$SANDBOX/home/gormes-agent" remote get-url origin)
  assert_eq "$origin_url" "git@github.com:TrebuchetDynamics/gormes-agent.git" "origin url preserved"
}

case_source_build_local() {
  # --local sets LOCAL_SOURCE_DIR=$(pwd), so the case must run install.sh from
  # the repo root (where go.mod lives). This exercises the source-build path
  # without any clone — the operator's existing checkout is the source of truth.
  capture_prestate

  (
    cd "$REPO_ROOT"
    GORMES_INSTALL_HOME="$SANDBOX/home" \
    GORMES_BIN_DIR="$SANDBOX/bin" \
    GORMES_SKIP_SETUP=1 \
    GORMES_RESTART_GATEWAY=never \
    sh "$INSTALL_SH" --local --skip-setup --restart-gateway never
  ) > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  # Took the source-build path, not binary-fetch.
  assert_grep "install_method: source-build" "$SANDBOX/logs/install.log"
  assert_grep "using local source checkout $REPO_ROOT" "$SANDBOX/logs/install.log"
  assert_no_grep "Resolving latest release tag" "$SANDBOX/logs/install.log"
  assert_no_grep "Installed gormes .* from release" "$SANDBOX/logs/install.log"

  # No clone went into the sandbox — --local should not write a checkout.
  assert_not_exists "$SANDBOX/home/gormes-agent"

  assert_executable "$SANDBOX/bin/gormes"
  local ver
  ver=$(GORMES_HOME="$SANDBOX/home" "$SANDBOX/bin/gormes" version 2>&1)
  printf '%s\n' "$ver" > "$SANDBOX/logs/version.log"
  assert_grep '^gormes ' "$SANDBOX/logs/version.log"

  capture_poststate
  assert_no_production_state_changes
}

case_source_build_clone() {
  # --from-source forces a clone + build (skip binary-fetch). With a fresh
  # sandbox the checkout lands at $SANDBOX/home/gormes-agent. This exercises
  # the path the user originally hit: binary-fetch failed and the installer
  # fell back to building from a cloned source, which required SSH or HTTPS
  # to actually reach github.com.
  capture_prestate

  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --from-source --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  # Took the source-build path via clone, not binary-fetch.
  assert_grep "install_method: source-build" "$SANDBOX/logs/install.log"
  assert_grep "Cloned via" "$SANDBOX/logs/install.log"
  assert_no_grep "Resolving latest release tag" "$SANDBOX/logs/install.log"
  assert_no_grep "Installed gormes .* from release" "$SANDBOX/logs/install.log"

  assert_file_exists "$SANDBOX/home/gormes-agent/.git"
  assert_file_exists "$SANDBOX/home/gormes-agent/go.mod"
  assert_executable "$SANDBOX/bin/gormes"

  local ver
  ver=$(GORMES_HOME="$SANDBOX/home" "$SANDBOX/bin/gormes" version 2>&1)
  printf '%s\n' "$ver" > "$SANDBOX/logs/version.log"
  assert_grep '^gormes ' "$SANDBOX/logs/version.log"

  capture_poststate
  assert_no_production_state_changes
}

case_uninstall_lifecycle() {
  # Install first.
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"
  assert_executable "$SANDBOX/bin/gormes"

  capture_prestate

  # Now uninstall and verify the sandbox is torn down.
  # GORMES_UNINSTALL_FORCE_PURGE=1 opts out of the default "gio trash" mover.
  # That default is deliberate for human operators (protects against accidental
  # real-~/.gormes wipes after the 2026-05-10 incident), but freedesktop trash
  # doesn't span the /tmp sandbox, so the test must opt into plain rm.
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  GORMES_UNINSTALL_FORCE_PURGE=1 \
  sh "$INSTALL_SH" --uninstall --yes > "$SANDBOX/logs/uninstall.log" 2>&1
  rc=$?
  assert_eq "$rc" 0 "uninstall exit code"

  assert_not_exists "$SANDBOX/bin/gormes"
  assert_not_exists "$SANDBOX/home/bin/gormes"

  capture_poststate
  assert_no_production_state_changes
}

# ----- runner ----------------------------------------------------------------

run_case() {
  local name=$1
  local fn=case_$1
  local sandbox="$SANDBOX_ROOT/$name"
  mkdir -p "$sandbox/home" "$sandbox/bin" "$sandbox/logs"

  if ! declare -F "$fn" >/dev/null; then
    printf 'FAIL %s (no such case)\n' "$name"
    FAIL=$((FAIL+1))
    FAILED_CASES="$FAILED_CASES $name"
    return
  fi

  printf '\n=== %s ===\n' "$name"
  local start end rc
  start=$(date +%s)
  # Run the case in a subshell with `set -e`. NOTE: do not wrap this in an
  # `if (...); then` because Bash silently disables `set -e` inside functions
  # called from a conditional context — assertion failures would then no-op
  # and the case would appear to PASS. Capture $? after the subshell instead.
  ( set -e; SANDBOX="$sandbox"; "$fn" )
  rc=$?
  end=$(date +%s)
  if [ "$rc" -eq 0 ]; then
    PASS=$((PASS+1))
    printf '  PASS %s (%ds)\n' "$name" "$((end-start))"
    [ -z "${GORMES_INSTALL_E2E_KEEP:-}" ] && rm -rf "$sandbox"
  else
    FAIL=$((FAIL+1))
    FAILED_CASES="$FAILED_CASES $name"
    printf '  FAIL %s (%ds) — sandbox kept: %s\n' "$name" "$((end-start))" "$sandbox"
  fi
}

CASES=(
  plan_dryrun_inside_sandbox
  binary_fetch_happy_path
  no_go_toolchain_binary_fetch
  api_failure_redirect_fallback
  hash_mismatch_aborts
  ssh_origin_update_fallback
  source_build_local
  source_build_clone
  uninstall_lifecycle
)

SELECTED=${1:-all}
printf 'install.sh e2e — sandbox root: %s\n' "$SANDBOX_ROOT"
printf 'install.sh path: %s\n' "$INSTALL_SH"

if [ "$SELECTED" = "all" ]; then
  for c in "${CASES[@]}"; do run_case "$c"; done
else
  run_case "$SELECTED"
fi

printf '\n=== summary ===\n  pass: %d\n  fail: %d%s\n' \
  "$PASS" "$FAIL" "${FAILED_CASES:+ (failed:$FAILED_CASES)}"

[ "$FAIL" -eq 0 ]
