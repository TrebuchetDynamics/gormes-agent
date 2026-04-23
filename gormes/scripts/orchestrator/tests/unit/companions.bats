#!/usr/bin/env bats

load '../lib/test_env'

setup() {
  load_helpers
  source_lib common
  source_lib companions

  TMP_WS="$(mktmp_workspace)"
  export RUN_ROOT="$TMP_WS/run"
  export STATE_DIR="$RUN_ROOT/state"
  export LOGS_DIR="$RUN_ROOT/logs"
  mkdir -p "$STATE_DIR" "$LOGS_DIR"

  export PLANNER_EVERY_N_CYCLES=4
  export DOC_IMPROVER_EVERY_N_CYCLES=6
  export LANDINGPAGE_EVERY_N_HOURS=24
  export PLANNER_ROOT="$TMP_WS/planner"
  mkdir -p "$PLANNER_ROOT"

  export CANDIDATES_FILE="$TMP_WS/cands.json"
  echo '[]' > "$CANDIDATES_FILE"

  # Ensure companion_state_dir is computed
  export ORCH_COMPANION_STATE_DIR="$(companion_state_dir)"
  mkdir -p "$ORCH_COMPANION_STATE_DIR"
}

write_companion_state() {
  local name="$1" ts="$2" cycle="$3"
  jq -n --arg ts "$ts" --argjson cycle "$cycle" --argjson rc 0 \
    '{ts_epoch:($ts|tonumber),cycle:$cycle,rc:$rc}' \
    > "$ORCH_COMPANION_STATE_DIR/${name}.last.json"
}

@test "companion_cycles_since returns large N when never run" {
  run companion_cycles_since planner 10
  assert_success
  (( output >= 10 ))
}

@test "companion_cycles_since returns diff since last run" {
  write_companion_state planner "$(date +%s)" 5
  run companion_cycles_since planner 9
  assert_output "4"
}

@test "should_run_planner fires on exhaustion (unclaimed<10%)" {
  # 10 candidates total, 0 unclaimed
  echo '[]' > "$CANDIDATES_FILE"
  write_companion_state planner "$(date +%s)" 0
  export _TOTAL_PROGRESS_ITEMS=10
  run should_run_planner 1
  assert_success
}

@test "should_run_planner fires on cycle interval" {
  echo '[]' > "$CANDIDATES_FILE"
  write_companion_state planner "$(date +%s)" 0
  export _TOTAL_PROGRESS_ITEMS=100
  echo '[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]' > "$CANDIDATES_FILE"
  run should_run_planner 4
  assert_success
}

@test "should_run_planner skips if external systemd ran recently" {
  cp "$FIXTURES_DIR/planner_state.fixture.json" "$PLANNER_ROOT/planner_state.json"
  # Set the fixture's last_run to just now
  jq --arg now "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '.last_run_utc = $now' "$PLANNER_ROOT/planner_state.json" \
    > "$PLANNER_ROOT/planner_state.json.tmp" && mv "$PLANNER_ROOT/planner_state.json.tmp" "$PLANNER_ROOT/planner_state.json"
  write_companion_state planner 0 0
  run should_run_planner 99
  assert_failure
}

@test "should_run_doc_improver skips when no promotions last cycle" {
  write_companion_state doc_improver 0 0
  export PROMOTED_LAST_CYCLE=0
  run should_run_doc_improver 10
  assert_failure
}

@test "should_run_doc_improver fires when interval reached + promotion happened" {
  write_companion_state doc_improver 0 0
  export PROMOTED_LAST_CYCLE=1
  run should_run_doc_improver 10
  assert_success
}

@test "should_run_landingpage fires after 24h" {
  # 25 hours ago
  write_companion_state landingpage "$(( $(date +%s) - 25 * 3600 ))" 0
  run should_run_landingpage
  assert_success
}

@test "should_run_landingpage skips within 24h" {
  write_companion_state landingpage "$(( $(date +%s) - 3600 ))" 0
  run should_run_landingpage
  assert_failure
}
