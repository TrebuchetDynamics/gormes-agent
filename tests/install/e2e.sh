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
mkdir -p "$SANDBOX_ROOT"

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
  api_failure_redirect_fallback
  ssh_origin_update_fallback
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
