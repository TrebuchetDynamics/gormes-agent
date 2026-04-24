#!/usr/bin/env bats

load '../lib/test_env'

setup() {
  load_helpers
  TMP_WS="$(mktmp_workspace)"
  export TMP_WS
  export GORMES_ORCHESTRATOR_SOURCE_ONLY=1
  export REPO_ROOT="$TMP_WS/repo"
  export RUN_ROOT="$TMP_WS/run"
  export PATH="$FIXTURES_DIR/bin:$PATH"
  export FINAL_REPORT_GRACE_SECONDS=0
  export ORCHESTRATOR_LIB_DIR
  source "$ENTRY_SCRIPT"
  trap - ERR
}

@test "legacy entry script can be source-loaded for direct soft-success tests" {
  run env \
    GORMES_ORCHESTRATOR_SOURCE_ONLY=1 \
    REPO_ROOT="$TMP_WS/repo" \
    RUN_ROOT="$TMP_WS/run" \
    PATH="$FIXTURES_DIR/bin:$PATH" \
    bash -c 'source "$1"; declare -F try_soft_success_nonzero >/dev/null' _ "$ENTRY_SCRIPT"

  assert_success
}

make_fixture_repo() {
  local repo="$1"
  git init -q -b main "$repo"
  git -C "$repo" -c user.email=t@t -c user.name=T commit -q --allow-empty -m init
}

make_orchestrator_repo() {
  local repo="$1"
  git init -q -b main "$repo"
  cp "$FIXTURES_DIR/progress.fixture.json" "$repo/progress.json"
  mkdir -p "$repo/docs/content/building-gormes/architecture_plan"
  cp "$FIXTURES_DIR/progress.fixture.json" \
    "$repo/docs/content/building-gormes/architecture_plan/progress.json"
  (
    cd "$repo"
    git -c user.email=t@t -c user.name=T add -A
    git -c user.email=t@t -c user.name=T commit -q -m init
  )
}

write_valid_final_report() {
  local report="$1"
  local branch="$2"
  local commit="$3"

  cat > "$report" <<REPORT
1) Selected task
Task: 1 / 1.C / Orchestrator failure-row stabilization for 4-8 workers

2) Pre-doc baseline
Files:
- scripts/gormes-auto-codexu-orchestrator.sh

3) RED proof
Command: bash scripts/orchestrator/tests/run.sh unit/soft-success.bats
Exit: 1
Snippet: failing direct soft-success guard

4) GREEN proof
Command: bash scripts/orchestrator/tests/run.sh unit/soft-success.bats
Exit: 0
Snippet: ok

5) REFACTOR proof
Command: bash scripts/orchestrator/tests/run.sh unit/soft-success.bats
Exit: 0
Snippet: ok

6) Regression proof
Command: bash scripts/orchestrator/tests/run.sh unit
Exit: 0
Snippet: ok

7) Post-doc closeout
Files:
- docs/content/building-gormes/architecture_plan/progress.json

8) Commit
Branch: $branch
Commit: $commit
Files:
- soft-success.txt

9) Acceptance check
Criterion: final report validates - PASS
Criterion: worker commit validates - PASS
Criterion: original non-zero exit can be recovered - PASS
REPORT
}

prepare_clean_worker_report() {
  export GIT_ROOT="$REPO_ROOT"
  export REPO_SUBDIR="."
  export RUN_ID="soft-success-run"
  export WORKTREES_DIR="$TMP_WS/worktrees"
  mkdir -p "$WORKTREES_DIR"

  make_fixture_repo "$GIT_ROOT"
  export BASE_COMMIT
  BASE_COMMIT="$(git -C "$GIT_ROOT" rev-parse HEAD)"

  create_worker_worktree 1
  local wt
  wt="$(worker_worktree_root 1)"
  (
    cd "$wt"
    printf 'soft-success\n' > soft-success.txt
    git -c user.email=t@t -c user.name=T add soft-success.txt
    git -c user.email=t@t -c user.name=T commit -q -m "test: soft success"
  )

  SOFT_SUCCESS_HEAD="$(git -C "$wt" rev-parse HEAD)"
  SOFT_SUCCESS_REPORT="$TMP_WS/final.md"
  SOFT_SUCCESS_STDERR="$TMP_WS/stderr.log"
  SOFT_SUCCESS_JSONL="$TMP_WS/run.jsonl"
  : > "$SOFT_SUCCESS_STDERR"
  : > "$SOFT_SUCCESS_JSONL"
  write_valid_final_report "$SOFT_SUCCESS_REPORT" "$(worker_branch_name 1)" "$SOFT_SUCCESS_HEAD"
}

@test "try_soft_success_nonzero rejects timeout and OOM exits despite a valid report and commit" {
  prepare_clean_worker_report

  ALLOW_SOFT_SUCCESS_NONZERO=1 run try_soft_success_nonzero 1 124 "$SOFT_SUCCESS_REPORT" "$SOFT_SUCCESS_STDERR" "$SOFT_SUCCESS_JSONL"
  assert_failure

  ALLOW_SOFT_SUCCESS_NONZERO=1 run try_soft_success_nonzero 1 137 "$SOFT_SUCCESS_REPORT" "$SOFT_SUCCESS_STDERR" "$SOFT_SUCCESS_JSONL"
  assert_failure
}

@test "try_soft_success_nonzero rejects when the recovery guard is disabled" {
  prepare_clean_worker_report

  ALLOW_SOFT_SUCCESS_NONZERO=0 run try_soft_success_nonzero 1 2 "$SOFT_SUCCESS_REPORT" "$SOFT_SUCCESS_STDERR" "$SOFT_SUCCESS_JSONL"

  assert_failure
}

@test "try_soft_success_nonzero rejects invalid final reports" {
  prepare_clean_worker_report
  printf 'incomplete report\n' > "$SOFT_SUCCESS_REPORT"

  ALLOW_SOFT_SUCCESS_NONZERO=1 run try_soft_success_nonzero 1 2 "$SOFT_SUCCESS_REPORT" "$SOFT_SUCCESS_STDERR" "$SOFT_SUCCESS_JSONL"

  assert_failure
}

@test "try_soft_success_nonzero rejects dirty worker commits" {
  prepare_clean_worker_report
  printf 'dirty\n' > "$(worker_worktree_root 1)/untracked.txt"

  ALLOW_SOFT_SUCCESS_NONZERO=1 run try_soft_success_nonzero 1 2 "$SOFT_SUCCESS_REPORT" "$SOFT_SUCCESS_STDERR" "$SOFT_SUCCESS_JSONL"

  assert_failure
}

@test "try_soft_success_nonzero accepts non-timeout non-OOM exits with valid report and clean commit" {
  prepare_clean_worker_report

  ALLOW_SOFT_SUCCESS_NONZERO=1 run try_soft_success_nonzero 1 2 "$SOFT_SUCCESS_REPORT" "$SOFT_SUCCESS_STDERR" "$SOFT_SUCCESS_JSONL"

  assert_success
}

@test "run_worker records soft_success_nonzero mode with original_rc after recovery" {
  make_orchestrator_repo "$REPO_ROOT"

  run env \
    GORMES_ORCHESTRATOR_SOURCE_ONLY=0 \
    REPO_ROOT="$REPO_ROOT" \
    RUN_ROOT="$RUN_ROOT" \
    RUN_ID="soft-success-worker-run" \
    PATH="$FIXTURES_DIR/bin:$PATH" \
    FAKE_CODEXU_MODE=success_nonzero \
    MAX_AGENTS=1 \
    MODE=safe \
    ORCHESTRATOR_ONCE=1 \
    AUTO_PROMOTE_SUCCESS=0 \
    HEARTBEAT_SECONDS=1 \
    FINAL_REPORT_GRACE_SECONDS=0 \
    WORKER_TIMEOUT_SECONDS=60 \
    MIN_AVAILABLE_MEM_MB=1 \
    MIN_MEM_PER_WORKER_MB=1 \
    MAX_EXISTING_CHROMIUM=9999 \
    FORCE_RUN_UNDER_PRESSURE=1 \
    "$ENTRY_SCRIPT"

  assert_success
  run jq -r '.status, .mode, (.original_rc | tostring)' "$RUN_ROOT/state/workers/soft-success-worker-run/worker_1.json"
  assert_success
  assert_line --index 0 "success"
  assert_line --index 1 "soft_success_nonzero"
  assert_line --index 2 "2"
  run jq -r 'select(.event == "worker_success" and .worker_id == 1) | .status' "$RUN_ROOT/state/runs.jsonl"
  assert_success
  assert_output --partial "soft_success_nonzero"
}
