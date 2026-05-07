#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

PATH="$HOME/.local/bin:$HOME/go/bin:$HOME/.cargo/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export PATH

REPO_ROOT="${GORMES_CODEXU_REPO:-/home/xel/git/sages-openclaw/workspace-mineru/gormes-agent}"
DEFAULT_RUNNER="$REPO_ROOT/scripts/codexu-gormes-builder-cron.sh"
RUNNER="${GORMES_CODEXU_RUNNER:-$DEFAULT_RUNNER}"
STATE_DIR="${GORMES_CODEXU_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/gormes-agent/codexu-builder}"
LOOP_LOCK="${GORMES_CODEXU_LOOP_LOCK:-$STATE_DIR/loop.lock}"
INTERVAL_SECONDS="${GORMES_CODEXU_LOOP_INTERVAL:-60}"
FAIL_BACKOFF_SECONDS="${GORMES_CODEXU_FAIL_BACKOFF:-300}"
PAUSE_POLL_SECONDS="${GORMES_CODEXU_PAUSE_POLL:-60}"
DEFAULT_PAUSE_TTL_SECONDS="${GORMES_CODEXU_PAUSE_TTL:-1800}"
LEGACY_PAUSE_MAX_AGE_SECONDS="${GORMES_CODEXU_LEGACY_PAUSE_MAX_AGE:-$DEFAULT_PAUSE_TTL_SECONDS}"
NODE_VERSION="${GORMES_CODEXU_NODE_VERSION:-22.21.1}"
HEALTH_FILE="$STATE_DIR/loop-health.env"
PID_FILE="$STATE_DIR/loop.pid"
CURRENT_RUN_FILE="$STATE_DIR/current-run.env"
LAST_SUCCESS_FILE="$STATE_DIR/last-success.env"
LAST_FAILURE_FILE="$STATE_DIR/last-failure.env"
PAUSE_FILE="$STATE_DIR/pause"
STOP_FILE="$STATE_DIR/stop-after-current"
LOG_DIR="$STATE_DIR/logs"
MAX_LOGS="${GORMES_CODEXU_MAX_LOGS:-80}"
LOOP_STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
CURRENT_CHILD=""
CURRENT_CHILD_OWN_SESSION=0
CONSECUTIVE_FAILURES=0

mkdir -p "$STATE_DIR" "$LOG_DIR"

timestamp() {
  date -u +%Y-%m-%dT%H:%M:%SZ
}

log() {
  printf '[%s] %s\n' "$(timestamp)" "$*"
}

atomic_kv_write() {
  local target="$1"
  local tmp="$target.$$"
  shift
  {
    while (($#)); do
      printf '%s=%q\n' "$1" "$2"
      shift 2
    done
  } >"$tmp"
  mv -f "$tmp" "$target"
}

parse_duration_seconds() {
  local value="$1"
  if [[ "$value" =~ ^[0-9]+$ ]]; then
    printf '%s\n' "$value"
    return 0
  fi
  if [[ "$value" =~ ^([0-9]+)([smhd])$ ]]; then
    local amount="${BASH_REMATCH[1]}"
    local unit="${BASH_REMATCH[2]}"
    case "$unit" in
      s) printf '%s\n' "$amount" ;;
      m) printf '%s\n' "$((amount * 60))" ;;
      h) printf '%s\n' "$((amount * 3600))" ;;
      d) printf '%s\n' "$((amount * 86400))" ;;
    esac
    return 0
  fi
  return 1
}

repo_branch() {
  git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || true
}

repo_head() {
  git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || true
}

pause_field() {
  local field="$1"
  local timestamp="" reason="" expires_at="" expires_at_epoch="" ttl_seconds="" forever=""
  [[ -f "$PAUSE_FILE" ]] || return 1
  # The pause file is private state written by this script with umask 077.
  # shellcheck source=/dev/null
  source "$PAUSE_FILE" 2>/dev/null || true
  case "$field" in
    timestamp) printf '%s\n' "$timestamp" ;;
    reason) printf '%s\n' "$reason" ;;
    expires_at) printf '%s\n' "$expires_at" ;;
    expires_at_epoch) printf '%s\n' "$expires_at_epoch" ;;
    ttl_seconds) printf '%s\n' "$ttl_seconds" ;;
    forever) printf '%s\n' "$forever" ;;
    *) return 1 ;;
  esac
}

pause_mtime_epoch() {
  stat -c %Y "$PAUSE_FILE" 2>/dev/null || printf '0\n'
}

pause_expired() {
  [[ -f "$PAUSE_FILE" ]] || return 1

  local forever expires_at_epoch now mtime age
  forever="$(pause_field forever || true)"
  [[ "$forever" == "1" ]] && return 1

  now="$(date +%s)"
  expires_at_epoch="$(pause_field expires_at_epoch || true)"
  if [[ -n "$expires_at_epoch" && "$expires_at_epoch" =~ ^[0-9]+$ ]]; then
    [[ "$now" -ge "$expires_at_epoch" ]]
    return $?
  fi

  if [[ "$LEGACY_PAUSE_MAX_AGE_SECONDS" =~ ^[0-9]+$ && "$LEGACY_PAUSE_MAX_AGE_SECONDS" -gt 0 ]]; then
    mtime="$(pause_mtime_epoch)"
    age=$((now - mtime))
    [[ "$age" -ge "$LEGACY_PAUSE_MAX_AGE_SECONDS" ]]
    return $?
  fi

  return 1
}

pause_state() {
  if [[ ! -f "$PAUSE_FILE" ]]; then
    printf 'absent\n'
    return 0
  fi
  if [[ "$(pause_field forever || true)" == "1" ]]; then
    printf 'forever\n'
    return 0
  fi
  if pause_expired; then
    printf 'expired\n'
    return 0
  fi
  printf 'active\n'
}

clear_expired_pause() {
  if pause_expired; then
    log "pause file expired; auto-resuming $PAUSE_FILE"
    rm -f "$PAUSE_FILE"
    return 0
  fi
  return 1
}

write_health() {
  atomic_kv_write "$HEALTH_FILE" \
    timestamp "$(timestamp)" \
    status "${1:-running}" \
    detail "${2:-}" \
    repo "$REPO_ROOT" \
    branch "$(repo_branch)" \
    head "$(repo_head)" \
    runner "$RUNNER" \
    loop_pid "$$" \
    loop_started_at "$LOOP_STARTED_AT" \
    current_child "$CURRENT_CHILD" \
    consecutive_failures "$CONSECUTIVE_FAILURES" \
    node "$(command -v node >/dev/null 2>&1 && node --version || true)" \
    codexu "$(command -v codexu 2>/dev/null || true)" \
    pause_file "$PAUSE_FILE" \
    pause_state "$(pause_state)" \
    stop_file "$STOP_FILE"
}

write_current_run() {
  atomic_kv_write "$CURRENT_RUN_FILE" \
    timestamp "$(timestamp)" \
    status "${1:-running}" \
    runner "$RUNNER" \
    pid "$CURRENT_CHILD" \
    started_at "${2:-}" \
    branch "$(repo_branch)" \
    head "$(repo_head)"
}

write_last_success() {
  atomic_kv_write "$LAST_SUCCESS_FILE" \
    timestamp "$(timestamp)" \
    elapsed_seconds "${1:-0}" \
    branch "$(repo_branch)" \
    head "$(repo_head)"
}

write_last_failure() {
  atomic_kv_write "$LAST_FAILURE_FILE" \
    timestamp "$(timestamp)" \
    elapsed_seconds "${1:-0}" \
    status "${2:-unknown}" \
    consecutive_failures "$CONSECUTIVE_FAILURES" \
    branch "$(repo_branch)" \
    head "$(repo_head)"
}

without_loop_lock_fd() {
  (exec 8>&-; "$@")
}

write_current_run_starting() {
  write_current_run "starting" "$(timestamp)"
}

write_current_run_running() {
  write_current_run "running" "$(date -u -d "@$started_at" +%Y-%m-%dT%H:%M:%SZ)"
}

activate_node() {
  local nvm_sh="${NVM_DIR:-$HOME/.nvm}/nvm.sh"
  if [[ -s "$nvm_sh" ]]; then
    # shellcheck disable=SC1090
    . "$nvm_sh"
    if nvm use "$NODE_VERSION" >/dev/null 2>&1; then
      log "using Node $(node --version) from $(command -v node)"
      return 0
    fi
  fi

  local node_bin="$HOME/.nvm/versions/node/v$NODE_VERSION/bin"
  if [[ -x "$node_bin/node" ]]; then
    PATH="$node_bin:$PATH"
    export PATH
    log "using Node $(node --version) from $(command -v node)"
    return 0
  fi

  log "warning: requested Node $NODE_VERSION unavailable; using $(command -v node 2>/dev/null || printf '<missing>') $(node --version 2>/dev/null || true)"
}

cleanup_old_logs() {
  find "$LOG_DIR" -maxdepth 1 -type f -name '*.log' -printf '%T@ %p\n' 2>/dev/null |
    sort -nr |
    awk -v keep="$MAX_LOGS" 'NR > keep {print $2}' |
    xargs -r rm -f --
}

runner_ready() {
  if [[ ! -d "$REPO_ROOT/.git" ]]; then
    log "repo is missing or not a git checkout: $REPO_ROOT"
    return 1
  fi
  if [[ ! -x "$RUNNER" ]]; then
    log "runner missing or not executable: $RUNNER"
    return 1
  fi
  if ! bash -n "$RUNNER"; then
    log "runner syntax check failed: $RUNNER"
    return 1
  fi
  if [[ "$RUNNER" == "$DEFAULT_RUNNER" ]] && ! command -v codexu >/dev/null 2>&1 && [[ ! -x "$HOME/.local/bin/codexu" ]]; then
    log "codexu is not available on PATH or at $HOME/.local/bin/codexu"
    return 1
  fi
  return 0
}

shutdown() {
  local reason="${1:-signal}"
  log "codexu builder loop stopping: $reason"
  without_loop_lock_fd write_health "stopping" "$reason"
  if [[ -n "$CURRENT_CHILD" ]] && kill -0 "$CURRENT_CHILD" 2>/dev/null; then
    log "forwarding stop to active runner pid $CURRENT_CHILD"
    if [[ "$CURRENT_CHILD_OWN_SESSION" == "1" ]]; then
      kill -- "-$CURRENT_CHILD" 2>/dev/null || true
    else
      kill "$CURRENT_CHILD" 2>/dev/null || true
    fi
    wait "$CURRENT_CHILD" 2>/dev/null || true
  fi
  without_loop_lock_fd rm -f "$PID_FILE" "$CURRENT_RUN_FILE"
  without_loop_lock_fd write_health "stopped" "$reason"
  exit 0
}

print_pause_status() {
  printf '\npause:\n'
  printf 'pause_file=%s\n' "$PAUSE_FILE"
  printf 'pause_state=%s\n' "$(pause_state)"
  if [[ -f "$PAUSE_FILE" ]]; then
    printf 'pause_timestamp=%s\n' "$(pause_field timestamp || true)"
    printf 'pause_reason=%s\n' "$(pause_field reason || true)"
    printf 'pause_ttl_seconds=%s\n' "$(pause_field ttl_seconds || true)"
    printf 'pause_expires_at=%s\n' "$(pause_field expires_at || true)"
  fi
}

print_status() {
  printf 'loop health: %s\n' "$HEALTH_FILE"
  if [[ -f "$HEALTH_FILE" ]]; then
    cat "$HEALTH_FILE"
  else
    printf 'status=%q\n' "missing_health_file"
  fi
  print_pause_status
  printf '\nprocesses:\n'
  ps -eo pid,ppid,lstart,etime,cmd | grep -E 'gormes-codexu-builder-loop|codexu-gormes-builder-loop|codexu-gormes-builder-cron|codexu exec' | grep -v grep || true
  printf '\nlatest logs:\n'
  ls -lt "$LOG_DIR" 2>/dev/null | head -n 8 || true
}

doctor() {
  print_status
  activate_node >/dev/null
  printf '\nchecks:\n'
  printf 'repo=%s\n' "$REPO_ROOT"
  printf 'branch=%s\n' "$(repo_branch)"
  printf 'head=%s\n' "$(repo_head)"
  printf 'runner=%s\n' "$RUNNER"
  bash -n "$0" && printf 'loop_syntax=ok\n'
  bash -n "$RUNNER" && printf 'runner_syntax=ok\n'
  printf 'node=%s %s\n' "$(command -v node 2>/dev/null || true)" "$(node --version 2>/dev/null || true)"
  printf 'codexu=%s\n' "$(command -v codexu 2>/dev/null || true)"
  printf '\ncron:\n'
  crontab -l 2>/dev/null | grep -F 'gormes-codexu-builder-loop' || true
}

pause_loop() {
  local ttl="$DEFAULT_PAUSE_TTL_SECONDS"
  local forever="0"
  local reason=()

  while (($#)); do
    case "$1" in
      --ttl)
        [[ $# -ge 2 ]] || {
          printf 'pause: --ttl requires a value\n' >&2
          exit 2
        }
        ttl="$(parse_duration_seconds "$2")" || {
          printf 'pause: invalid --ttl value: %s\n' "$2" >&2
          exit 2
        }
        shift 2
        ;;
      --forever)
        forever="1"
        ttl="0"
        shift
        ;;
      *)
        reason+=("$1")
        shift
        ;;
    esac
  done

  local reason_text="${reason[*]}"
  local now_epoch expires_epoch expires_at
  now_epoch="$(date +%s)"
  expires_epoch=""
  expires_at=""
  if [[ "$forever" != "1" && "$ttl" =~ ^[0-9]+$ && "$ttl" -gt 0 ]]; then
    expires_epoch=$((now_epoch + ttl))
    expires_at="$(date -u -d "@$expires_epoch" +%Y-%m-%dT%H:%M:%SZ)"
  fi

  atomic_kv_write "$PAUSE_FILE" \
    timestamp "$(timestamp)" \
    reason "$reason_text" \
    ttl_seconds "$ttl" \
    expires_at "$expires_at" \
    expires_at_epoch "$expires_epoch" \
    forever "$forever"

  if [[ "$forever" == "1" ]]; then
    printf 'paused forever: active run, if any, will finish; resume is required\n'
  else
    printf 'paused until %s: active run, if any, will finish; future runs wait until resume or expiry\n' "$expires_at"
  fi
}

case "${1:-run}" in
  status)
    print_status
    exit 0
    ;;
  doctor)
    doctor
    exit 0
    ;;
  pause)
    shift
    pause_loop "$@"
    exit 0
    ;;
  resume)
    rm -f "$PAUSE_FILE" "$STOP_FILE"
    printf 'resumed\n'
    exit 0
    ;;
  stop-after-current)
    shift
    atomic_kv_write "$STOP_FILE" timestamp "$(timestamp)" reason "$*"
    printf 'stop requested: active run, if any, will finish before loop exits\n'
    exit 0
    ;;
  run)
    ;;
  *)
    printf 'usage: %s [run|status|doctor|pause [--ttl DURATION|--forever] [reason]|resume|stop-after-current [reason]]\n' "$0" >&2
    exit 2
    ;;
esac

trap 'shutdown signal' INT TERM

cd "$REPO_ROOT"
activate_node

exec 8>"$LOOP_LOCK"
if ! flock -n 8; then
  log "codexu builder loop already running; exiting"
  exit 0
fi

without_loop_lock_fd log "codexu builder loop started for $REPO_ROOT"
printf '%s\n' "$$" >"$PID_FILE"
without_loop_lock_fd write_health "started"

while true; do
  if [[ -f "$STOP_FILE" ]]; then
    without_loop_lock_fd log "stop-after-current file present before next run; exiting"
    shutdown "stop-after-current"
  fi
  while [[ -f "$PAUSE_FILE" ]]; do
    if without_loop_lock_fd clear_expired_pause; then
      break
    fi
    without_loop_lock_fd log "pause file present; waiting ${PAUSE_POLL_SECONDS}s"
    without_loop_lock_fd write_health "paused"
    without_loop_lock_fd sleep "$PAUSE_POLL_SECONDS"
    if [[ -f "$STOP_FILE" ]]; then
      without_loop_lock_fd log "stop-after-current requested while paused; exiting"
      shutdown "stop-after-current"
    fi
  done

  without_loop_lock_fd cleanup_old_logs
  if ! without_loop_lock_fd runner_ready; then
    without_loop_lock_fd write_health "runner_not_ready"
    without_loop_lock_fd sleep "$FAIL_BACKOFF_SECONDS"
    continue
  fi

  started_at="$(without_loop_lock_fd date +%s)"
  without_loop_lock_fd write_current_run_starting
  set +e
  if command -v setsid >/dev/null 2>&1; then
    (exec 8>&-; exec setsid "$RUNNER") &
    CURRENT_CHILD_OWN_SESSION=1
  else
    (exec 8>&-; exec "$RUNNER") &
    CURRENT_CHILD_OWN_SESSION=0
  fi
  CURRENT_CHILD=$!
  without_loop_lock_fd write_current_run_running
  without_loop_lock_fd write_health "running_runner"
  wait "$CURRENT_CHILD"
  status=$?
  set -e
  finished_at="$(without_loop_lock_fd date +%s)"
  elapsed=$((finished_at - started_at))
  CURRENT_CHILD=""
  CURRENT_CHILD_OWN_SESSION=0
  without_loop_lock_fd rm -f "$CURRENT_RUN_FILE"

  if [[ "$status" -eq 0 ]]; then
    CONSECUTIVE_FAILURES=0
    without_loop_lock_fd log "runner completed in ${elapsed}s; sleeping ${INTERVAL_SECONDS}s"
    without_loop_lock_fd write_last_success "$elapsed"
    without_loop_lock_fd write_health "sleeping"
    without_loop_lock_fd sleep "$INTERVAL_SECONDS"
  else
    CONSECUTIVE_FAILURES=$((CONSECUTIVE_FAILURES + 1))
    without_loop_lock_fd log "runner exited with status $status after ${elapsed}s; backing off ${FAIL_BACKOFF_SECONDS}s"
    without_loop_lock_fd write_last_failure "$elapsed" "$status"
    without_loop_lock_fd write_health "runner_failed"
    without_loop_lock_fd sleep "$FAIL_BACKOFF_SECONDS"
  fi
done
