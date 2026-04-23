#!/usr/bin/env bats
# Pins the 79% promotion-failure behavior measured in the 24h audit:
# two workers edit the same line of progress.fixture.json from the same
# BASE_COMMIT, both produce valid commits, but sequential cherry-pick
# promotion only lands the first one. The second should log
# cherry_pick_failed and leave no CHERRY_PICK_HEAD behind.

load '../lib/test_env'

setup() {
  load_helpers
  TMP_WS="$(mktmp_workspace)"

  export PATH="$FIXTURES_DIR/bin:$PATH"
  export FAKE_CODEXU_MODE=conflict
  export FAKE_CODEXU_LOG="$TMP_WS/fake-codexu.log"

  export REPO_ROOT="$TMP_WS/repo"
  git init -q -b main "$REPO_ROOT"

  # progress.fixture.json must be tracked at BASE_COMMIT so each worker's
  # worktree starts from the same line-level content. The conflict mode in
  # fake-codexu overwrites this file with a PID-tagged payload, so the two
  # parallel workers will end up with different commits that can't both be
  # cherry-picked onto the integration branch.
  echo '{}' > "$REPO_ROOT/progress.fixture.json"

  mkdir -p "$REPO_ROOT/docs/content/building-gormes/architecture_plan"
  cp "$FIXTURES_DIR/progress.fixture.json" \
     "$REPO_ROOT/docs/content/building-gormes/architecture_plan/progress.json"
  (
    cd "$REPO_ROOT"
    git -c user.email=t@t -c user.name=T add -A
    git -c user.email=t@t -c user.name=T commit -q -m init
  )

  export RUN_ROOT="$TMP_WS/run"
  export MAX_AGENTS=2
  export MODE=safe
  export ORCHESTRATOR_ONCE=1
  export HEARTBEAT_SECONDS=1
  export FINAL_REPORT_GRACE_SECONDS=1
  export WORKER_TIMEOUT_SECONDS=60
  export MIN_AVAILABLE_MEM_MB=1
  export MIN_MEM_PER_WORKER_MB=1
  export MAX_EXISTING_CHROMIUM=9999
  export FORCE_RUN_UNDER_PRESSURE=1
  export AUTO_PROMOTE_SUCCESS=1
  # Different branch from happy-path.bats so runs don't stomp each other.
  export INTEGRATION_BRANCH="codexu/test-autoloop-conflict"
  export KEEP_WORKTREES=0
}

@test "two conflicting workers: one promotes, other emits cherry_pick_failed, integration clean" {
  run "$ENTRY_SCRIPT"
  # The run itself may return non-zero because one worker fails promotion;
  # we care about the ledger + git state, not the exit code.

  [ -f "$RUN_ROOT/state/runs.jsonl" ]

  # Both workers should have produced a successful final report (worker_success).
  local success_count
  success_count="$(grep -c 'worker_success' "$RUN_ROOT/state/runs.jsonl" || true)"
  assert_equal "$success_count" "2"

  # Sequential promotion: one cherry-pick lands, the other conflicts.
  grep -q 'worker_promoted' "$RUN_ROOT/state/runs.jsonl"
  grep -q 'cherry_pick_failed' "$RUN_ROOT/state/runs.jsonl"

  # Exactly one promoted-row (not 0, not 2). We don't pin which worker wins
  # because the promotion loop order depends on which worker_state file is
  # iterated first.
  local promoted_count
  promoted_count="$(grep -c 'worker_promoted.*promoted' "$RUN_ROOT/state/runs.jsonl" || true)"
  assert_equal "$promoted_count" "1"

  # Exactly one cherry_pick_failed.
  local failed_count
  failed_count="$(grep -c 'cherry_pick_failed' "$RUN_ROOT/state/runs.jsonl" || true)"
  assert_equal "$failed_count" "1"

  # Clean abort: no lingering CHERRY_PICK_HEAD in the integration worktree.
  # The integration worktree lives under $RUN_ROOT/integration/... and its
  # per-worktree git dir is at $REPO_ROOT/.git/worktrees/<safe_branch>/.
  run bash -c "find '$REPO_ROOT/.git/worktrees' -name CHERRY_PICK_HEAD 2>/dev/null"
  assert_success
  assert_output ""

  # Integration branch advanced by exactly one commit beyond init.
  run git -C "$REPO_ROOT" log --oneline "$INTEGRATION_BRANCH"
  assert_success
  [ "$(echo "$output" | wc -l)" -eq 2 ]
}
