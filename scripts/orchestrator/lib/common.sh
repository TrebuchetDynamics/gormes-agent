#!/usr/bin/env bash
# Common logging, path, and small utility helpers for the orchestrator.
# Sourced by gormes-auto-codexu-orchestrator.sh and its tests.
# Depends on: $VERBOSE (reads; default 0 if unset).

# Verbose logging functions
log_info() {
  if [[ "$VERBOSE" == "1" ]]; then
    echo "[INFO] $(date '+%H:%M:%S') $*" >&2
  fi
}

log_debug() {
  if [[ "$VERBOSE" == "1" ]]; then
    echo "[DEBUG] $(date '+%H:%M:%S') $*" >&2
  fi
}

log_warn() {
  echo "[WARN] $(date '+%H:%M:%S') $*" >&2
}

log_error() {
  echo "[ERROR] $(date '+%H:%M:%S') $*" >&2
}

# Progress indicator
show_progress() {
  local current=$1
  local total=$2
  local label="${3:-Progress}"
  local width=50
  local percentage=$((current * 100 / total))
  local filled=$((width * current / total))
  local empty=$((width - filled))

  printf "\r%s [%s%s] %3d%% (%d/%d)" \
    "$label" \
    "$(printf '%*s' "$filled" '' | tr ' ' '=')" \
    "$(printf '%*s' "$empty" '' | tr ' ' ' ')" \
    "$percentage" \
    "$current" \
    "$total"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: missing required command: $1" >&2
    exit 1
  }
}

safe_path_token() {
  printf '%s' "$1" | sed -E 's#[^A-Za-z0-9._-]+#-#g; s#^-+##; s#-+$##'
}

available_mem_mb() {
  free -m | awk '/^Mem:/ { print $7 }'
}

classify_worker_failure() {
  local rc="$1"
  if [[ "$rc" == "124" ]]; then
    printf 'timeout\n'
  elif [[ "$rc" == "137" ]]; then
    printf 'killed\n'
  elif [[ "$rc" == "1" ]]; then
    printf 'contract_or_test_failure\n'
  else
    printf 'worker_error\n'
  fi
}

# Check if process is alive and not a zombie.
proc_alive() {
  local pid="$1"
  [[ "$pid" =~ ^[0-9]+$ ]] || return 1
  [[ -d "/proc/$pid" ]] && ! grep -q 'Z)' "/proc/$pid/stat" 2>/dev/null
}

abort_worker_pids() {
  local reason="${1:-worker failure}"
  shift || true

  local pid
  for pid in "$@"; do
    if proc_alive "$pid"; then
      kill -TERM "$pid" 2>/dev/null || true
      log_warn "Aborted worker pid $pid after $reason"
    fi
  done
}

should_pause_after_cycle() {
  local cycle_rc="$1"
  [[ "${PAUSE_ON_RUN_FAILURE:-1}" == "1" ]] || return 1
  [[ "$cycle_rc" != "0" && "$cycle_rc" != "75" ]]
}

should_run_post_cycle_companions() {
  local cycle_rc="$1"
  [[ "${SKIP_COMPANIONS_ON_RUN_FAILURE:-1}" == "1" ]] || return 0
  [[ "$cycle_rc" == "0" ]]
}

worker_status_outcome() {
  local line="$1"
  if [[ "$line" =~ ^worker\[[0-9]+\]:[[:space:]]+(success|soft-success) ]]; then
    printf 'success\n'
  elif [[ "$line" =~ ^worker\[[0-9]+\]:[[:space:]]+quota-exhausted ]]; then
    printf 'quota\n'
  elif [[ "$line" =~ ^worker\[[0-9]+\]:[[:space:]]+timeout ]]; then
    printf 'timeout\n'
  elif [[ "$line" =~ ^worker\[[0-9]+\]:[[:space:]]+aborted-fail-fast ]]; then
    printf 'aborted\n'
  elif [[ "$line" =~ ^worker\[[0-9]+\]:[[:space:]]+failed ]]; then
    printf 'failed\n'
  else
    printf 'other\n'
  fi
}

provider_quota_message() {
  local file
  for file in "$@"; do
    [[ -n "$file" && -f "$file" ]] || continue
    grep -Eaim1 \
      "You've hit your limit|usage limit|rate limit|quota exceeded|too many requests|HTTP 429|429 Too Many Requests" \
      "$file" && return 0
  done
  return 1
}

provider_quota_exhausted() {
  provider_quota_message "$@" >/dev/null
}
