#!/usr/bin/env bats

load '../lib/test_env'

make_fixture_repo() {
  local repo="$1"
  git init -q -b main "$repo"
  git -C "$repo" -c user.email=t@t -c user.name=T commit -q --allow-empty -m init
}

setup() {
  load_helpers
  source_lib common
  source_lib report
  source_lib worktree
  TMP_WS="$(mktmp_workspace)"
  export GIT_ROOT="$TMP_WS/repo"
  export WORKTREES_DIR="$TMP_WS/wt"
  export REPO_SUBDIR="."
  export RUN_ID="wrt-run-1"
  export PROGRESS_JSON_REL="progress.json"
  make_fixture_repo "$GIT_ROOT"
  export BASE_COMMIT="$(git -C "$GIT_ROOT" rev-parse HEAD)"
  mkdir -p "$WORKTREES_DIR"
}

@test "worker_branch_name format" {
  run worker_branch_name 3
  assert_output "codexu/wrt-run-1/worker3"
}

@test "worker_worktree_root format" {
  run worker_worktree_root 2
  assert_output "$WORKTREES_DIR/worker2"
}

@test "create_worker_worktree checks out base commit on new branch" {
  run create_worker_worktree 1
  assert_success
  [[ -d "$WORKTREES_DIR/worker1" ]]
  local head
  head="$(git -C "$WORKTREES_DIR/worker1" rev-parse HEAD)"
  assert_equal "$head" "$BASE_COMMIT"
  local branch
  branch="$(git -C "$WORKTREES_DIR/worker1" rev-parse --abbrev-ref HEAD)"
  assert_equal "$branch" "codexu/wrt-run-1/worker1"
}

@test "verify_worker_commit rejects unchanged HEAD" {
  create_worker_worktree 1
  local report="$TMP_WS/f.md"
  printf 'Branch: codexu/wrt-run-1/worker1\nCommit: %s\n' "$BASE_COMMIT" > "$report"
  run verify_worker_commit 1 "$report"
  assert_failure
  assert_output --partial "HEAD did not advance"
}

@test "verify_worker_commit rejects multiple commits" {
  create_worker_worktree 1
  local wt="$WORKTREES_DIR/worker1"
  ( cd "$wt" && echo a > a && git -c user.email=t@t -c user.name=T add a && git -c user.email=t@t -c user.name=T commit -q -m a )
  ( cd "$wt" && echo b > b && git -c user.email=t@t -c user.name=T add b && git -c user.email=t@t -c user.name=T commit -q -m b )
  local head
  head="$(git -C "$wt" rev-parse HEAD)"
  local report="$TMP_WS/f.md"
  printf 'Branch: codexu/wrt-run-1/worker1\nCommit: %s\n' "$head" > "$report"
  run verify_worker_commit 1 "$report"
  assert_failure
  assert_output --partial "commit count"
}

@test "verify_worker_commit rejects dirty worktree" {
  create_worker_worktree 1
  local wt="$WORKTREES_DIR/worker1"
  ( cd "$wt" && echo a > a && git -c user.email=t@t -c user.name=T add a && git -c user.email=t@t -c user.name=T commit -q -m a )
  ( cd "$wt" && echo stray > stray )
  local head
  head="$(git -C "$wt" rev-parse HEAD)"
  local report="$TMP_WS/f.md"
  printf 'Branch: codexu/wrt-run-1/worker1\nCommit: %s\n' "$head" > "$report"
  run verify_worker_commit 1 "$report"
  assert_failure
  assert_output --partial "not clean"
}

@test "verify_worker_commit accepts single valid commit" {
  create_worker_worktree 1
  local wt="$WORKTREES_DIR/worker1"
  ( cd "$wt" && echo a > a && git -c user.email=t@t -c user.name=T add a && git -c user.email=t@t -c user.name=T commit -q -m a )
  local head
  head="$(git -C "$wt" rev-parse HEAD)"
  local report="$TMP_WS/f.md"
  printf 'Branch: codexu/wrt-run-1/worker1\nCommit: %s\n' "$head" > "$report"
  run verify_worker_commit 1 "$report"
  assert_success
}
