#!/usr/bin/env bash
# Companion-script periodic invocation helpers.
# Depends on: $RUN_ROOT, $STATE_DIR, $PLANNER_EVERY_N_CYCLES, $DOC_IMPROVER_EVERY_N_CYCLES,
#             $LANDINGPAGE_EVERY_N_HOURS, $PLANNER_ROOT, $LOOP_SLEEP_SECONDS,
#             $PROMOTED_LAST_CYCLE, $DISABLE_COMPANIONS, $COMPANION_ON_IDLE,
#             $COMPANION_TIMEOUT_SECONDS, $COMPANION_PLANNER_CMD,
#             $COMPANION_DOC_IMPROVER_CMD, $COMPANION_LANDINGPAGE_CMD.
# Reads the candidates file + optional $_TOTAL_PROGRESS_ITEMS override (tests only).

companion_state_dir() {
  printf '%s/companions\n' "$RUN_ROOT"
}

companion_last_ts() {
  local name="$1"
  local f
  f="$(companion_state_dir)/${name}.last.json"
  if [[ -f "$f" ]]; then
    jq -r '.ts_epoch // 0' "$f"
  else
    printf '0\n'
  fi
}

companion_last_cycle() {
  local name="$1"
  local f
  f="$(companion_state_dir)/${name}.last.json"
  if [[ -f "$f" ]]; then
    jq -r '.cycle // 0' "$f"
  else
    printf '0\n'
  fi
}

companion_cycles_since() {
  local name="$1"
  local current_cycle="$2"
  local last
  last="$(companion_last_cycle "$name")"
  printf '%d\n' $(( current_cycle - last ))
}

_candidates_remaining() {
  [[ -f "$CANDIDATES_FILE" ]] || { printf '0\n'; return; }
  jq 'length' "$CANDIDATES_FILE"
}

_planner_external_recent() {
  local state="$PLANNER_ROOT/planner_state.json"
  [[ -f "$state" ]] || return 1
  local ts
  ts="$(jq -r '.last_run_utc // empty' "$state")"
  [[ -n "$ts" ]] || return 1
  local epoch
  epoch="$(date -d "$ts" +%s 2>/dev/null || true)"
  [[ -n "$epoch" ]] || return 1
  local threshold=$(( PLANNER_EVERY_N_CYCLES * ${LOOP_SLEEP_SECONDS:-30} * 2 ))
  local now
  now="$(date +%s)"
  (( now - epoch < threshold ))
}

should_run_planner() {
  local cycle="$1"
  [[ "${DISABLE_COMPANIONS:-0}" == "1" ]] && return 1
  _planner_external_recent && return 1
  local remaining total
  remaining="$(_candidates_remaining)"
  total="${_TOTAL_PROGRESS_ITEMS:-$remaining}"
  (( total > 0 )) || return 0
  # Exhaustion trigger: unclaimed < 10%
  if (( remaining * 10 < total )); then
    return 0
  fi
  local since
  since="$(companion_cycles_since planner "$cycle")"
  (( since >= PLANNER_EVERY_N_CYCLES ))
}

should_run_doc_improver() {
  local cycle="$1"
  [[ "${DISABLE_COMPANIONS:-0}" == "1" ]] && return 1
  local promoted="${PROMOTED_LAST_CYCLE:-0}"
  (( promoted >= 1 )) || return 1
  local since
  since="$(companion_cycles_since doc_improver "$cycle")"
  (( since >= DOC_IMPROVER_EVERY_N_CYCLES ))
}

should_run_landingpage() {
  [[ "${DISABLE_COMPANIONS:-0}" == "1" ]] && return 1
  local last
  last="$(companion_last_ts landingpage)"
  local now
  now="$(date +%s)"
  local delta=$(( now - last ))
  (( delta >= LANDINGPAGE_EVERY_N_HOURS * 3600 ))
}

run_companion() {
  local name="$1"
  local cmd_var_name
  case "$name" in
    planner)       cmd_var_name="COMPANION_PLANNER_CMD" ;;
    doc_improver)  cmd_var_name="COMPANION_DOC_IMPROVER_CMD" ;;
    landingpage)   cmd_var_name="COMPANION_LANDINGPAGE_CMD" ;;
    *) echo "run_companion: unknown companion '$name'" >&2; return 1 ;;
  esac
  local cmd="${!cmd_var_name:-}"
  if [[ -z "$cmd" ]]; then
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
    case "$name" in
      planner)       cmd="$script_dir/gormes-architecture-planner-tasks-manager.sh" ;;
      doc_improver)  cmd="$script_dir/documentation-improver.sh" ;;
      landingpage)   cmd="$script_dir/landingpage-improver.sh" ;;
    esac
  fi

  mkdir -p "$(companion_state_dir)"
  local ts_start
  ts_start="$(date +%s)"
  local ts_utc
  ts_utc="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  local rc=0
  (
    cd "$GIT_ROOT"
    AUTO_COMMIT=1 AUTO_PUSH=0 PLANNER_INSTALL_SCHEDULE=0 \
      timeout "${COMPANION_TIMEOUT_SECONDS:-1800}" bash "$cmd" \
      >"$LOGS_DIR/companion_${name}.out.log" 2>"$LOGS_DIR/companion_${name}.err.log"
  ) || rc=$?

  jq -n \
    --arg name "$name" \
    --argjson ts_epoch "$ts_start" \
    --arg ts_utc "$ts_utc" \
    --argjson cycle "${ORCH_CURRENT_CYCLE:-0}" \
    --argjson rc "$rc" \
    '{name:$name,ts_epoch:$ts_epoch,ts_utc:$ts_utc,cycle:$cycle,rc:$rc}' \
    > "$(companion_state_dir)/${name}.last.json"

  type log_event >/dev/null 2>&1 && log_event "companion_${name}_completed" null "rc=$rc" "completed" || true
  return "$rc"
}

maybe_run_companions() {
  local cycle="$1"
  local promoted="${2:-0}"
  export ORCH_CURRENT_CYCLE="$cycle"
  export PROMOTED_LAST_CYCLE="$promoted"

  [[ "${DISABLE_COMPANIONS:-0}" == "1" ]] && return 0

  local exhausted=0
  local remaining
  remaining="$(_candidates_remaining)"
  local total="${_TOTAL_PROGRESS_ITEMS:-$remaining}"
  if (( total > 0 )) && (( remaining * 10 < total )); then
    exhausted=1
  fi

  if [[ "${COMPANION_ON_IDLE:-1}" == "1" && "$exhausted" == "0" && "$promoted" == "0" ]]; then
    return 0
  fi

  if should_run_planner "$cycle"; then
    run_companion planner || true
    export EXHAUSTION_TRIGGERED="$exhausted"
  fi
  if should_run_doc_improver "$cycle"; then
    run_companion doc_improver || true
  fi
  if should_run_landingpage; then
    run_companion landingpage || true
  fi
}
