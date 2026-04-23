#!/usr/bin/env bats

load '../lib/test_env'

make_fixture_repo() {
  local repo="$1"
  git init -q -b main "$repo"
  git -C "$repo" -c user.email=t@t -c user.name=T commit -q --allow-empty -m init
}

write_worker_state() {
  local id="$1" slug="$2" commit="$3" status="$4"
  local dir="$RUN_WORKER_STATE_DIR"
  mkdir -p "$dir"
  jq -n --arg run "$RUN_ID" --arg s "$status" --arg slug "$slug" --arg c "$commit" \
    '{run_id:$run,status:$s,slug:$slug,commit:$c}' > "$dir/worker_${id}.json"
}

setup() {
  load_helpers
  source_lib common
  source_lib promote
  TMP_WS="$(mktmp_workspace)"
  export GIT_ROOT="$TMP_WS/int"
  export INTEGRATION_BRANCH="codexu/autoloop"
  export AUTO_PROMOTE_SUCCESS=1
  export RUN_ID="prom-1"
  export RUN_WORKER_STATE_DIR="$TMP_WS/workers/$RUN_ID"
  export STATE_DIR="$TMP_WS/state"
  export RUNS_LEDGER="$STATE_DIR/runs.jsonl"
  export AUTO_PUSH=0
  mkdir -p "$STATE_DIR"
  make_fixture_repo "$GIT_ROOT"
  git -C "$GIT_ROOT" checkout -q -b "$INTEGRATION_BRANCH"
  # Re-source load_worker_state + log_event — they live in entry script until those extractions;
  # for promote.bats we define lightweight stubs if absent.
  type load_worker_state >/dev/null 2>&1 || load_worker_state() { cat "$RUN_WORKER_STATE_DIR/worker_$1.json" 2>/dev/null; }
  type log_event >/dev/null 2>&1 || log_event() { :; }
}

@test "promote_successful_workers skips when feature disabled" {
  export AUTO_PROMOTE_SUCCESS=0
  run promote_successful_workers 2
  assert_success
}

@test "promote_successful_workers cherry-picks one success" {
  # Build a branch that modifies a file, record its commit, then reset integration
  ( cd "$GIT_ROOT" && git -c user.email=t@t -c user.name=T checkout -q -b feat )
  ( cd "$GIT_ROOT" && echo a > a && git -c user.email=t@t -c user.name=T add a && git -c user.email=t@t -c user.name=T commit -q -m add-a )
  local commit
  commit="$(git -C "$GIT_ROOT" rev-parse HEAD)"
  ( cd "$GIT_ROOT" && git checkout -q "$INTEGRATION_BRANCH" )
  write_worker_state 1 "foo__bar" "$commit" "success"
  run promote_successful_workers 1
  assert_success
  local head
  head="$(git -C "$GIT_ROOT" log --format=%s -n1 "$INTEGRATION_BRANCH")"
  assert_equal "$head" "add-a"
}

@test "promote_successful_workers aborts cherry-pick on conflict" {
  # Worker 1 commits a→"one"; integration then commits a→"two"; worker's cherry-pick will conflict.
  ( cd "$GIT_ROOT" && git -c user.email=t@t -c user.name=T checkout -q -b feat )
  ( cd "$GIT_ROOT" && echo one > a && git -c user.email=t@t -c user.name=T add a && git -c user.email=t@t -c user.name=T commit -q -m feat-a )
  local worker_commit
  worker_commit="$(git -C "$GIT_ROOT" rev-parse HEAD)"
  ( cd "$GIT_ROOT" && git checkout -q "$INTEGRATION_BRANCH" )
  ( cd "$GIT_ROOT" && echo two > a && git -c user.email=t@t -c user.name=T add a && git -c user.email=t@t -c user.name=T commit -q -m int-a )
  write_worker_state 1 "foo__bar" "$worker_commit" "success"
  run promote_successful_workers 1
  assert_failure
  [[ ! -f "$GIT_ROOT/.git/CHERRY_PICK_HEAD" ]]
}

@test "promote_successful_workers exports PROMOTED_LAST_CYCLE" {
  ( cd "$GIT_ROOT" && git -c user.email=t@t -c user.name=T checkout -q -b feat )
  ( cd "$GIT_ROOT" && echo a > a && git -c user.email=t@t -c user.name=T add a && git -c user.email=t@t -c user.name=T commit -q -m add-a )
  local commit
  commit="$(git -C "$GIT_ROOT" rev-parse HEAD)"
  ( cd "$GIT_ROOT" && git checkout -q "$INTEGRATION_BRANCH" )
  write_worker_state 1 "foo__bar" "$commit" "success"
  promote_successful_workers 1
  assert_equal "$PROMOTED_LAST_CYCLE" "1"
}
