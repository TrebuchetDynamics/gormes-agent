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
  # `--` separates options from positional args so patterns starting with `-`
  # (e.g. `--branch development`) aren't parsed by grep as flags.
  if ! grep -qE -- "$pat" "$file" 2>/dev/null; then
    printf '    ! pattern not found: %s\n      in: %s\n' "$pat" "$file" >&2
    return 1
  fi
}

assert_no_grep() {
  local pat=$1 file=$2
  if grep -qE -- "$pat" "$file" 2>/dev/null; then
    printf '    ! pattern unexpectedly present: %s\n      in: %s\n' "$pat" "$file" >&2
    grep -nE -- "$pat" "$file" | head -5 >&2
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

case_no_curl_or_wget_falls_back_to_git_source_build() {
  # Operator guarantee: a host with neither curl nor wget on PATH still gets
  # a working install via the source-build fallback. install.sh's binary-fetch
  # path needs curl or wget for both tag resolution (API call OR releases/latest
  # redirect) and the asset download; git uses its own HTTPS implementation and
  # doesn't depend on either. With curl/wget absent, tag resolution fails first
  # at install.sh:1093 ("could not resolve latest release tag"), binary-fetch
  # aborts, source-build kicks in, git clones, Go builds, install succeeds.
  capture_prestate

  local shim="$SANDBOX/path-no-curl-wget"
  mkdir -p "$shim"
  # Include git + go so the source-build fallback can run. Omit curl and wget.
  local needed tool real
  needed=(
    sh bash uname mkdir rm ln mv cp chmod sleep
    tar gzip gunzip awk grep sed head tail cat tr cut sort
    id hostname printf date dirname basename true false test '['
    sha256sum shasum stat readlink which env mktemp file
    git go tput touch chown find xargs ps kill wait
  )
  for tool in "${needed[@]}"; do
    real=$(command -v "$tool" 2>/dev/null || true)
    if [ -n "$real" ]; then
      ln -sf "$real" "$shim/$tool"
    fi
  done

  if PATH="$shim" command -v curl >/dev/null 2>&1; then
    printf '    ! shim PATH still has curl: %s\n' "$(PATH=$shim command -v curl)" >&2
    return 1
  fi
  if PATH="$shim" command -v wget >/dev/null 2>&1; then
    printf '    ! shim PATH still has wget: %s\n' "$(PATH=$shim command -v wget)" >&2
    return 1
  fi
  if ! PATH="$shim" command -v git >/dev/null 2>&1; then
    printf '    ! shim PATH missing git (needed for source-build clone)\n' >&2
    return 1
  fi
  if ! PATH="$shim" command -v go >/dev/null 2>&1; then
    printf '    ! shim PATH missing go (needed for source-build compile)\n' >&2
    return 1
  fi

  PATH="$shim" \
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  # binary-fetch must have failed at tag resolution (the redirect fallback
  # also needs curl/wget; both branches abort, the tag stays empty, and
  # install.sh logs "could not resolve latest release tag").
  assert_grep "could not resolve latest release tag" "$SANDBOX/logs/install.log"
  assert_grep "binary-fetch failed; falling back to source build" "$SANDBOX/logs/install.log"

  # Source-build must have run: clone via git, build via go.
  assert_grep "Cloned via" "$SANDBOX/logs/install.log"
  assert_file_exists "$SANDBOX/home/gormes-agent/.git"
  assert_executable "$SANDBOX/bin/gormes"

  PATH="$shim" GORMES_HOME="$SANDBOX/home" \
    "$SANDBOX/bin/gormes" version > "$SANDBOX/logs/version.log" 2>&1
  assert_grep '^gormes ' "$SANDBOX/logs/version.log"

  capture_poststate
  assert_no_production_state_changes
}

case_branch_forces_source_build() {
  # install.sh:1030 — any --branch other than main forces source-build because
  # release binaries are only published from main. Use a real non-main branch
  # (development) so the clone actually has something to fetch, and verify
  # the checkout lands on that branch.
  capture_prestate

  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --branch development --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  # Plan and method should both reflect source-build with the --branch reason.
  assert_grep "install_method: source-build" "$SANDBOX/logs/install.log"
  assert_grep "--branch development.*release binaries are only published from main" \
    "$SANDBOX/logs/install.log"
  assert_grep "branch: development" "$SANDBOX/logs/install.log"
  # Binary-fetch must NOT have been attempted.
  assert_no_grep "Resolving latest release tag" "$SANDBOX/logs/install.log"
  assert_no_grep "Installed gormes .* from release" "$SANDBOX/logs/install.log"

  # Clone must land on the requested branch.
  assert_file_exists "$SANDBOX/home/gormes-agent/.git"
  local branch
  branch=$(git -C "$SANDBOX/home/gormes-agent" rev-parse --abbrev-ref HEAD 2>/dev/null)
  assert_eq "$branch" "development" "checked-out branch"

  assert_executable "$SANDBOX/bin/gormes"
  local ver
  ver=$(GORMES_HOME="$SANDBOX/home" "$SANDBOX/bin/gormes" version 2>&1)
  printf '%s\n' "$ver" > "$SANDBOX/logs/version.log"
  assert_grep '^gormes ' "$SANDBOX/logs/version.log"

  capture_poststate
  assert_no_production_state_changes
}

case_no_systemd_install_skips_service() {
  # Operator guarantee: a host without systemctl (Docker minimal images, WSL1,
  # busybox-init Linux) gets a clean install with the system service step
  # silently skipped. install.sh:1884 guards the systemd-install branch on
  # `has systemctl && systemctl --user >/dev/null 2>&1`; on a no-systemctl
  # host both halves fail and install.sh falls through to `elif Darwin` (no)
  # and exits the function without writing anything.
  #
  # This case CANNOT use GORMES_BIN_DIR — that env triggers the boundary
  # short-circuit at install.sh:1881 which skips the systemd branch entirely,
  # bypassing the no-systemctl check we want to exercise. Instead we redirect
  # HOME into the sandbox so install.sh's normal production writes
  # ($HOME/.local/bin/gormes, $HOME/.bashrc, $HOME/.config/systemd/user/) all
  # land inside SANDBOX without touching the operator's real home.
  capture_prestate

  local shim="$SANDBOX/path-no-systemctl"
  mkdir -p "$shim"
  # Full toolset MINUS systemctl. Need curl/wget for binary-fetch happy path.
  local needed tool real
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
    fi
  done

  if PATH="$shim" command -v systemctl >/dev/null 2>&1; then
    printf '    ! shim PATH still has systemctl: %s\n' \
      "$(PATH=$shim command -v systemctl)" >&2
    return 1
  fi
  if ! PATH="$shim" command -v curl >/dev/null 2>&1; then
    printf '    ! shim PATH missing curl (need it for binary-fetch)\n' >&2
    return 1
  fi

  local fake_home="$SANDBOX/fake-home"
  mkdir -p "$fake_home"

  # NO GORMES_BIN_DIR / GORMES_PREFIX here — those would trigger
  # install.sh's sandbox-boundary short-circuit and skip the systemd
  # branch we want to exercise. HOME redirection alone keeps every
  # production write inside the sandbox.
  PATH="$shim" \
  HOME="$fake_home" \
  GORMES_INSTALL_HOME="$SANDBOX/managed" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --skip-setup --restart-gateway never \
    > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  # The plan still announces the systemd intent (plan output is platform-
  # agnostic), but runtime must NOT actually create the unit file because
  # systemctl wasn't on PATH.
  assert_not_exists "$fake_home/.config/systemd/user/gormes-gateway.service"

  # And install.sh's sandbox-boundary skip message must NOT have fired,
  # because we deliberately did NOT set GORMES_BIN_DIR — that confirms the
  # test actually exercised the no-systemctl branch rather than skipping it.
  assert_no_grep "skipping system service install" "$SANDBOX/logs/install.log"

  # Binary must be runnable from its real published location (under fake HOME).
  assert_executable "$fake_home/.local/bin/gormes"
  PATH="$shim" GORMES_HOME="$SANDBOX/managed" \
    "$fake_home/.local/bin/gormes" version > "$SANDBOX/logs/version.log" 2>&1
  assert_grep '^gormes ' "$SANDBOX/logs/version.log"

  # The operator's REAL home was never touched — capture_prestate /
  # capture_poststate run with the test runner's HOME (unmodified), so
  # this diff catches any production leak.
  capture_poststate
  assert_no_production_state_changes
}

case_termux_detection_drives_plan() {
  # Operator running install.sh in Termux: install.sh:240 detects via
  # TERMUX_VERSION or $PREFIX containing /com.termux/files/usr. Detection
  # branches:
  #   - pick_bin_dir uses $PREFIX/bin  (install.sh:309)
  #   - prereqs call ensure_termux_core_packages instead of ensure_go/git
  #     (install.sh:636) — guarded so non-termux CI doesn't try pkg
  #   - release_platform_arch returns android-arm64 on aarch64/arm64 or
  #     empty (forcing source-build) on x86_64  (install.sh:980)
  #
  # We can't run a real Termux install on a generic Linux runner, but we
  # CAN exercise the dry-run plan with TERMUX_VERSION + PREFIX set and
  # prove the detection branches still produce a coherent plan. Locks in
  # the detection contract against accidental removal of Termux support
  # in a future install.sh refactor.

  local fake_prefix="$SANDBOX/termux-prefix"
  mkdir -p "$fake_prefix/bin"

  TERMUX_VERSION="0.118.1" \
  PREFIX="$fake_prefix" \
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --dry-run > "$SANDBOX/logs/dryrun.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "--dry-run exit code"

  # The bin dir on Termux is $PREFIX/bin (install.sh:309 — `is_termux &&
  # [ -n "$PREFIX" ]`). That's the clearest single signal that detection
  # fired correctly.
  assert_grep "published_binary: $fake_prefix/bin/gormes" "$SANDBOX/logs/dryrun.log"

  # Asset-slug logic must reflect host arch within the Termux branch.
  local arch
  arch=$(uname -m)
  case "$arch" in
    aarch64|arm64)
      assert_grep "install_method: binary-fetch" "$SANDBOX/logs/dryrun.log"
      assert_grep "android-arm64" "$SANDBOX/logs/dryrun.log"
      ;;
    *)
      # Termux-detected on a non-arm64 host: no android-amd64 asset is
      # published, so install.sh must fall to source-build with a
      # "no published release asset" reason.
      assert_grep "install_method: source-build" "$SANDBOX/logs/dryrun.log"
      assert_grep "no published release asset" "$SANDBOX/logs/dryrun.log"
      ;;
  esac

  # Detection via $PREFIX path pattern (without TERMUX_VERSION) must also
  # work — same path operators get from termux-app itself.
  PREFIX="/data/data/com.termux/files/usr" \
  GORMES_INSTALL_HOME="$SANDBOX/home2" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --dry-run > "$SANDBOX/logs/dryrun-prefix-only.log" 2>&1
  assert_eq "$?" 0 "--dry-run exit code (PREFIX-only detection)"
  assert_grep "published_binary: /data/data/com.termux/files/usr/bin/gormes" \
    "$SANDBOX/logs/dryrun-prefix-only.log"
}

case_termux_local_install_doctor_smoke() {
  # Synthetic Termux e2e: build from the current checkout, publish into
  # $PREFIX/bin, skip desktop service install even if a host systemctl is
  # available, then run the installed binary's Termux-aware doctor check.
  capture_prestate

  local fake_prefix="$SANDBOX/com.termux/files/usr"
  local fake_home="$SANDBOX/com.termux/files/home"
  local fake_gocache="$SANDBOX/go-cache"
  local fake_gomodcache="$SANDBOX/go-modcache"
  local shim="$SANDBOX/shim"
  mkdir -p "$fake_prefix/bin" "$fake_home" "$fake_gocache" "$fake_gomodcache" "$shim"

  cat > "$shim/pkg" <<'SH'
#!/bin/sh
exit 0
SH
  chmod +x "$shim/pkg"

  cat > "$shim/systemctl" <<'SH'
#!/bin/sh
exit 0
SH
  chmod +x "$shim/systemctl"

  for cmd in termux-wake-lock termux-notification; do
    cat > "$fake_prefix/bin/$cmd" <<'SH'
#!/bin/sh
exit 0
SH
    chmod +x "$fake_prefix/bin/$cmd"
  done

  (
    cd "$REPO_ROOT"
    TERMUX_VERSION="0.119.0" \
    PREFIX="$fake_prefix" \
    HOME="$fake_home" \
    PATH="$shim:$fake_prefix/bin:$PATH" \
    GOCACHE="$fake_gocache" \
    GOMODCACHE="$fake_gomodcache" \
    GOFLAGS="${GOFLAGS:+$GOFLAGS }-modcacherw" \
    GORMES_INSTALL_HOME="$SANDBOX/home" \
    GORMES_SKIP_SETUP=1 \
    GORMES_RESTART_GATEWAY=never \
    sh "$INSTALL_SH" --local --skip-setup --restart-gateway never
  ) > "$SANDBOX/logs/install.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "install exit code"

  assert_grep "install_method: source-build" "$SANDBOX/logs/install.log"
  assert_grep "using local source checkout $REPO_ROOT" "$SANDBOX/logs/install.log"
  assert_grep "skipping system service install .*Termux" "$SANDBOX/logs/install.log"
  assert_executable "$fake_prefix/bin/gormes"
  assert_not_exists "$fake_home/.config/systemd/user/gormes-gateway.service"

  TERMUX_VERSION="0.119.0" \
  PREFIX="$fake_prefix" \
  HOME="$fake_home" \
  PATH="$fake_prefix/bin:$PATH" \
  GORMES_HOME="$SANDBOX/home" \
    "$fake_prefix/bin/gormes" doctor --offline --json \
    > "$SANDBOX/logs/doctor.json" 2>&1

  assert_grep '"name": "Termux runtime"' "$SANDBOX/logs/doctor.json"
  assert_grep "desktop-like command path ready" "$SANDBOX/logs/doctor.json"
  assert_grep "termux-api commands available" "$SANDBOX/logs/doctor.json"
  assert_grep "run long gateway sessions inside tmux" "$SANDBOX/logs/doctor.json"

  capture_poststate
  assert_no_production_state_changes
}

case_root_linux_install_uses_usr_local() {
  # Operator running install.sh as root on Linux lands in a different bin
  # dir (/usr/local/bin) and managed checkout (/usr/local/lib/gormes-agent)
  # than the default user install. install.sh:265 reads the effective UID
  # via effective_uid(), which honors GORMES_INSTALL_EFFECTIVE_UID — spoof
  # root for the plan check without needing sudo. The actual install would
  # write to /usr/local/bin which a non-root test can't do; --dry-run
  # validates the plan branch.
  #
  # Combine --verbose with --from-source --dry-run so the assertions can
  # see both:
  #   - "root_linux_install: yes" / "effective_uid: 0" (verbose plan
  #     diagnostics from print_verbose_plan)
  #   - "published_binary: /usr/local/bin/gormes" (pick_bin_dir:314)
  #   - "checkout: /usr/local/lib/gormes-agent" (managed_checkout_dir:290,
  #     only printed under source-build, so we force --from-source)

  GORMES_INSTALL_EFFECTIVE_UID=0 \
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --verbose --from-source --dry-run \
    > "$SANDBOX/logs/dryrun.log" 2>&1
  local rc=$?
  assert_eq "$rc" 0 "--verbose --from-source --dry-run exit code"

  # Verbose plan explicitly reports the root-install detection.
  assert_grep "root_linux_install: yes" "$SANDBOX/logs/dryrun.log"
  assert_grep "effective_uid: 0" "$SANDBOX/logs/dryrun.log"

  # Root + no-legacy-checkout overrides pick_bin_dir (install.sh:313) and
  # managed_checkout_dir (install.sh:290).
  assert_grep "published_binary: /usr/local/bin/gormes" "$SANDBOX/logs/dryrun.log"
  assert_grep "checkout: /usr/local/lib/gormes-agent" "$SANDBOX/logs/dryrun.log"

  # The user-install default bin dir must NOT appear when root is detected
  # and no GORMES_BIN_DIR/GORMES_PREFIX is set.
  assert_no_grep "published_binary: $HOME/.local/bin/gormes" "$SANDBOX/logs/dryrun.log"
}

case_uninstall_dry_run_previews_no_deletion() {
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

  # Snapshot every installed file/symlink so we can prove dry-run touched none.
  find "$SANDBOX/bin" "$SANDBOX/home" \( -type f -o -type l \) 2>/dev/null \
    | sort > "$SANDBOX/logs/pre-inventory.txt"
  local pre_count
  pre_count=$(wc -l < "$SANDBOX/logs/pre-inventory.txt")

  # Dry-run uninstall. install.sh detects --dry-run and skips its
  # apply-by-default behavior — the underlying `gormes uninstall --dry-run`
  # prints the artifacts that *would* be removed and returns without
  # touching the filesystem.
  GORMES_INSTALL_HOME="$SANDBOX/home" \
  GORMES_BIN_DIR="$SANDBOX/bin" \
  GORMES_SKIP_SETUP=1 \
  GORMES_RESTART_GATEWAY=never \
  sh "$INSTALL_SH" --uninstall --dry-run \
    > "$SANDBOX/logs/uninstall.log" 2>&1
  rc=$?
  assert_eq "$rc" 0 "uninstall --dry-run exit code"

  # Preview markers from `gormes uninstall --dry-run`: header announces
  # artifact count, and at minimum the published-binary section appears.
  assert_grep "uninstall dry-run: [0-9]+ artifact" "$SANDBOX/logs/uninstall.log"
  assert_grep "\[published-binary\]" "$SANDBOX/logs/uninstall.log"
  # An actual deletion would log "uninstall complete: N removed". Dry-run
  # must not — its absence is the contract under test.
  assert_no_grep "uninstall complete:" "$SANDBOX/logs/uninstall.log"

  # Binary must still be executable.
  assert_executable "$SANDBOX/bin/gormes"

  # Post-inventory must be byte-identical to pre-inventory.
  find "$SANDBOX/bin" "$SANDBOX/home" \( -type f -o -type l \) 2>/dev/null \
    | sort > "$SANDBOX/logs/post-inventory.txt"
  local post_count
  post_count=$(wc -l < "$SANDBOX/logs/post-inventory.txt")
  assert_eq "$post_count" "$pre_count" "sandbox file count unchanged"
  if ! diff -q "$SANDBOX/logs/pre-inventory.txt" \
                "$SANDBOX/logs/post-inventory.txt" >/dev/null 2>&1; then
    printf '    ! sandbox inventory changed during --uninstall --dry-run\n' >&2
    diff "$SANDBOX/logs/pre-inventory.txt" "$SANDBOX/logs/post-inventory.txt" >&2 || true
    return 1
  fi
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
    if [ -z "${GORMES_INSTALL_E2E_KEEP:-}" ]; then
      chmod -R u+w "$sandbox" 2>/dev/null || true
      rm -rf "$sandbox"
    fi
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
  branch_forces_source_build
  no_curl_or_wget_falls_back_to_git_source_build
  no_systemd_install_skips_service
  termux_detection_drives_plan
  termux_local_install_doctor_smoke
  root_linux_install_uses_usr_local
  uninstall_dry_run_previews_no_deletion
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
