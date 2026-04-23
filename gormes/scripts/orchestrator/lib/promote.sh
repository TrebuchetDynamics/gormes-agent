#!/usr/bin/env bash
# Promotion (cherry-pick) of successful worker commits onto the integration branch.
# Depends on: $AUTO_PROMOTE_SUCCESS, $GIT_ROOT, $INTEGRATION_BRANCH, $ORIGINAL_REPO_ROOT,
#             $RUN_WORKER_STATE_DIR, $AUTO_PUSH, $REMOTE_NAME.
# Exports: PROMOTED_LAST_CYCLE (count of cherry-picks that landed this invocation).

promotion_enabled() {
  [[ "$AUTO_PROMOTE_SUCCESS" == "1" ]]
}

setup_integration_root() {
  promotion_enabled || return 0

  require_cmd git

  local source_git_root source_subdir safe_branch existing_worktree
  source_git_root="$(git -C "$ORIGINAL_REPO_ROOT" rev-parse --show-toplevel)"
  source_subdir="."
  if [[ "$ORIGINAL_REPO_ROOT" != "$source_git_root" ]]; then
    source_subdir="${ORIGINAL_REPO_ROOT#"$source_git_root"/}"
  fi

  if ! git -C "$source_git_root" show-ref --verify --quiet "refs/heads/$INTEGRATION_BRANCH"; then
    git -C "$source_git_root" branch "$INTEGRATION_BRANCH" HEAD
  fi

  existing_worktree="$(branch_worktree_path "$source_git_root" "$INTEGRATION_BRANCH")"
  if [[ -n "$existing_worktree" ]]; then
    INTEGRATION_WORKTREE="$existing_worktree"
  else
    if [[ -z "$INTEGRATION_WORKTREE" ]]; then
      safe_branch="$(safe_path_token "$INTEGRATION_BRANCH")"
      INTEGRATION_WORKTREE="$RUN_ROOT/integration/$safe_branch"
    fi
    mkdir -p "$(dirname "$INTEGRATION_WORKTREE")"
    git -C "$source_git_root" worktree add "$INTEGRATION_WORKTREE" "$INTEGRATION_BRANCH" >/dev/null
  fi

  if [[ -n "$(git -C "$INTEGRATION_WORKTREE" status --short)" ]]; then
    echo "ERROR: integration worktree is dirty: $INTEGRATION_WORKTREE" >&2
    echo "Resolve or remove it before running the forever orchestrator." >&2
    exit 1
  fi

  git -C "$INTEGRATION_WORKTREE" reset --hard "$INTEGRATION_BRANCH" >/dev/null

  if [[ "$source_subdir" == "." ]]; then
    REPO_ROOT="$INTEGRATION_WORKTREE"
  else
    REPO_ROOT="$INTEGRATION_WORKTREE/$source_subdir"
  fi
  refresh_repo_paths
}

# Push integration branch to remote
push_integration_branch() {
  if [[ "$AUTO_PUSH" != "1" ]]; then
    return 0
  fi

  log_info "Pushing $INTEGRATION_BRANCH to $REMOTE_NAME"

  if ! git -C "$GIT_ROOT" push "$REMOTE_NAME" "$INTEGRATION_BRANCH"; then
    log_error "Failed to push $INTEGRATION_BRANCH to $REMOTE_NAME"
    return 1
  fi

  log_info "Successfully pushed $INTEGRATION_BRANCH"
  return 0
}

cmd_promote_commit() {
  local target_run="$1"
  local worker_id="$2"
  local target_branch="${3:-$(git -C "$GIT_ROOT" rev-parse --abbrev-ref HEAD)}"
  local prefix commit
  prefix="$(latest_worker_log_prefix "$target_run" "$worker_id")"
  [[ -n "$prefix" ]] || { echo "No logs for run=$target_run worker=$worker_id" >&2; return 1; }
  commit="$(cat "$LOGS_DIR/${prefix}.head" 2>/dev/null || true)"
  [[ -n "$commit" ]] || { echo "No commit head found for $prefix" >&2; return 1; }

  git -C "$GIT_ROOT" checkout "$target_branch" >/dev/null
  git -C "$GIT_ROOT" cherry-pick "$commit"
  echo "promoted commit $commit onto $target_branch"
}

promote_successful_workers() {
  local workers="$1"
  promotion_enabled || return 0

  local rc=0 promoted=0 i state_json status commit slug

  if [[ -n "$(git -C "$GIT_ROOT" status --short)" ]]; then
    echo "ERROR: integration branch worktree is dirty before promotion: $GIT_ROOT" >&2
    return 1
  fi

  for (( i = 1; i <= workers; i++ )); do
    state_json="$(load_worker_state "$i" 2>/dev/null || true)"
    [[ -n "$state_json" ]] || continue

    status="$(jq -r '.status // ""' <<<"$state_json")"
    [[ "$status" == "success" ]] || continue

    commit="$(jq -r '.commit // ""' <<<"$state_json")"
    slug="$(jq -r '.slug // ""' <<<"$state_json")"
    if [[ -z "$commit" || "$commit" == "null" ]]; then
      echo "worker[$i]: success state missing commit; cannot promote" >&2
      log_event "worker_promotion_failed" "$i" "$slug" "missing_commit"
      rc=1
      continue
    fi

    if git -C "$GIT_ROOT" merge-base --is-ancestor "$commit" HEAD 2>/dev/null; then
      echo "worker[$i]: already promoted -> $slug ($commit)"
      log_event "worker_promoted" "$i" "$slug@$commit" "already_promoted"
      continue
    fi

    echo "worker[$i]: promoting -> $slug ($commit) onto $INTEGRATION_BRANCH"
    if git -C "$GIT_ROOT" cherry-pick "$commit" >/dev/null; then
      promoted=$((promoted + 1))
      log_event "worker_promoted" "$i" "$slug@$commit" "promoted"
    else
      git -C "$GIT_ROOT" cherry-pick --abort >/dev/null 2>&1 || true
      echo "worker[$i]: promotion failed -> $slug ($commit)" >&2
      log_event "worker_promotion_failed" "$i" "$slug@$commit" "cherry_pick_failed"
      rc=1
    fi
  done

  if (( promoted > 0 )); then
    echo "Promoted worker commits: $promoted"
    echo "Integration head: $(git -C "$GIT_ROOT" rev-parse --short HEAD)"

    # Push integration branch to remote if AUTO_PUSH is enabled
    if [[ "$AUTO_PUSH" == "1" ]]; then
      push_integration_branch || rc=1
    fi
  fi

  export PROMOTED_LAST_CYCLE="$promoted"
  return "$rc"
}
