#!/usr/bin/env bash
# Git worktree lifecycle + post-run verification helpers.
# Depends on: $GIT_ROOT, $WORKTREES_DIR, $REPO_SUBDIR, $RUN_ID, $BASE_COMMIT,
#             $KEEP_WORKTREES, $PINNED_RUNS_FILE, $MAX_RUN_WORKTREE_DIRS, $RUN_ROOT.

branch_worktree_path() {
  local git_root="$1"
  local branch="$2"

  git -C "$git_root" worktree list --porcelain \
    | awk -v branch_ref="refs/heads/${branch}" '
        /^worktree / { path = substr($0, 10) }
        /^branch / {
          if (!found && substr($0, 8) == branch_ref) {
            print path
            found = 1
          }
        }
      '
}

worker_branch_name() {
  local worker_id="$1"
  printf 'codexu/%s/worker%d' "$RUN_ID" "$worker_id"
}

worker_worktree_root() {
  local worker_id="$1"
  printf '%s/worker%d' "$WORKTREES_DIR" "$worker_id"
}

worker_repo_root() {
  local worker_id="$1"
  local worktree_root
  worktree_root="$(worker_worktree_root "$worker_id")"
  if [[ "$REPO_SUBDIR" == "." ]]; then
    printf '%s\n' "$worktree_root"
  else
    printf '%s/%s\n' "$worktree_root" "$REPO_SUBDIR"
  fi
}

create_worker_worktree() {
  local worker_id="$1"
  local worktree_root branch
  worktree_root="$(worker_worktree_root "$worker_id")"
  branch="$(worker_branch_name "$worker_id")"

  mkdir -p "$(dirname "$worktree_root")"
  git -C "$GIT_ROOT" worktree add -b "$branch" "$worktree_root" "$BASE_COMMIT" >/dev/null 2>&1
}

maybe_remove_worker_worktree() {
  local worker_id="$1"
  local worktree_root
  worktree_root="$(worker_worktree_root "$worker_id")"

  if [[ "$KEEP_WORKTREES" == "0" && -d "$worktree_root" ]]; then
    git -C "$GIT_ROOT" worktree remove --force "$worktree_root" >/dev/null 2>&1 || true
  fi
}

enforce_worktree_dir_cap() {
  local keep="$MAX_RUN_WORKTREE_DIRS"
  local dirs=()
  local d

  while IFS= read -r d; do
    [[ -n "$d" ]] && dirs+=("$d")
  done < <(find "$RUN_ROOT/worktrees" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' 2>/dev/null | sort -nr | awk '{print $2}')

  local idx=0
  for d in "${dirs[@]}"; do
    idx=$((idx + 1))
    if (( idx <= keep )); then
      continue
    fi
    if [[ "$(basename "$d")" == "$RUN_ID" ]]; then
      continue
    fi
    if grep -Fxq "$(basename "$d")" "$PINNED_RUNS_FILE" 2>/dev/null; then
      continue
    fi
    git -C "$GIT_ROOT" worktree remove --force "$d" >/dev/null 2>&1 || true
    rm -rf "$d" 2>/dev/null || true
  done
}

verify_worker_commit() {
  local worker_id="$1"
  local final_file="$2"
  local worktree_root branch head_commit report_commit report_branch commit_count status_output changed_files file

  # Reset reason so a successful run doesn't leak a stale value from
  # a prior invocation.
  LAST_VERIFY_REASON=""
  export LAST_VERIFY_REASON

  worktree_root="$(worker_worktree_root "$worker_id")"
  branch="$(worker_branch_name "$worker_id")"
  head_commit="$(git -C "$worktree_root" rev-parse HEAD)"
  if [[ "$head_commit" == "$BASE_COMMIT" ]]; then
    LAST_VERIFY_REASON="no_commit_made"
    echo "worker[$worker_id]: HEAD did not advance beyond $BASE_COMMIT" >&2
    return 1
  fi

  commit_count="$(git -C "$worktree_root" rev-list --count "${BASE_COMMIT}..HEAD")"
  if [[ "$commit_count" != "1" ]]; then
    LAST_VERIFY_REASON="wrong_commit_count"
    echo "worker[$worker_id]: commit count = $commit_count, want exactly 1" >&2
    return 1
  fi

  status_output="$(git -C "$worktree_root" status --short)"
  if [[ -n "$status_output" ]]; then
    LAST_VERIFY_REASON="worktree_dirty"
    echo "worker[$worker_id]: worktree not clean after run" >&2
    printf '%s\n' "$status_output" >&2
    return 1
  fi

  report_commit="$(extract_report_commit "$final_file")"
  if [[ -z "$report_commit" || "$head_commit" != "$report_commit"* ]]; then
    LAST_VERIFY_REASON="report_commit_mismatch"
    echo "worker[$worker_id]: report commit does not match HEAD ($report_commit vs $head_commit)" >&2
    return 1
  fi

  report_branch="$(extract_report_branch "$final_file")"
  if [[ "$report_branch" != "$branch" ]]; then
    LAST_VERIFY_REASON="branch_mismatch"
    echo "worker[$worker_id]: report branch does not match expected branch ($report_branch vs $branch)" >&2
    return 1
  fi

  changed_files="$(git -C "$worktree_root" diff --name-only "${BASE_COMMIT}..HEAD")"
  if [[ -z "$changed_files" ]]; then
    LAST_VERIFY_REASON="no_commit_made"
    echo "worker[$worker_id]: commit contains no file changes" >&2
    return 1
  fi

  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    if [[ "$REPO_SUBDIR" != "." && "$file" != "$REPO_SUBDIR/"* ]]; then
      LAST_VERIFY_REASON="scope_violation"
      echo "worker[$worker_id]: changed file escaped allowed scope: $file" >&2
      return 1
    fi
  done <<< "$changed_files"

  return 0
}
