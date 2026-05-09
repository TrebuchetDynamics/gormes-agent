#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"

PATH="$HOME/.local/bin:$HOME/go/bin:$HOME/.cargo/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export PATH

BRANCH="${GORMES_BUILDER_BRANCH:-${GORMES_CODEXU_BRANCH:-development}}"
RUN_TIMEOUT="${GORMES_BUILDER_TIMEOUT:-${GORMES_CODEXU_TIMEOUT:-3h}}"
STATE_DIR="${GORMES_BUILDER_STATE_DIR:-${GORMES_CODEXU_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/gormes-agent/builder}}"
LOG_DIR="${GORMES_BUILDER_LOG_DIR:-${GORMES_CODEXU_LOG_DIR:-$STATE_DIR/logs}}"
LOCK_FILE="${GORMES_BUILDER_LOCK_FILE:-${GORMES_CODEXU_LOCK_FILE:-$STATE_DIR/run.lock}}"
CODEXU_BIN="${CODEXU_BIN:-codexu}"
SKIP_REMOTE_SYNC="${GORMES_BUILDER_SKIP_REMOTE_SYNC:-${GORMES_CODEXU_SKIP_REMOTE_SYNC:-0}}"
BACKEND="${GORMES_BUILDER_BACKEND:-codexu}"
OPENCODE_BIN="${GORMES_OPENCODE_BIN:-opencode}"
OPENCODE_MODEL="${GORMES_OPENCODE_MODEL:-opencode-go/deepseek-v4-pro}"

mkdir -p "$STATE_DIR" "$LOG_DIR"
RUN_ID="$(date -u +%Y%m%dT%H%M%SZ)"
RUN_LOG="$LOG_DIR/$RUN_ID.log"
LAST_MESSAGE="$STATE_DIR/last-message.txt"

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" | tee -a "$RUN_LOG"
}

die() {
  log "error: $*"
  exit 1
}

resolve_codexu() {
  if command -v "$CODEXU_BIN" >/dev/null 2>&1; then
    command -v "$CODEXU_BIN"
    return 0
  fi
  if [[ -x "$HOME/.local/bin/codexu" ]]; then
    printf '%s\n' "$HOME/.local/bin/codexu"
    return 0
  fi
  if [[ -x "$REPO_ROOT/scripts/orchestrator/codexu" ]]; then
    printf '%s\n' "$REPO_ROOT/scripts/orchestrator/codexu"
    return 0
  fi
  return 1
}

resolve_opencode() {
  if command -v "$OPENCODE_BIN" >/dev/null 2>&1; then
    command -v "$OPENCODE_BIN"
    return 0
  fi
  if [[ -x "$HOME/.opencode/bin/opencode" ]]; then
    printf '%s\n' "$HOME/.opencode/bin/opencode"
    return 0
  fi
  return 1
}

worktree_dirty() {
  ! git diff --quiet ||
    ! git diff --cached --quiet ||
    [[ -n "$(git ls-files --others --exclude-standard)" ]]
}

ensure_development_branch() {
  local current
  current="$(git rev-parse --abbrev-ref HEAD)"
  [[ "$current" == "$BRANCH" ]] || die "current branch is $current, expected $BRANCH"
}

sync_development_branch() {
  [[ "$SKIP_REMOTE_SYNC" == "1" ]] && return 0

  git fetch origin "$BRANCH" >>"$RUN_LOG" 2>&1 || die "git fetch origin $BRANCH failed"

  local local_head remote_head merge_base
  local_head="$(git rev-parse HEAD)"
  remote_head="$(git rev-parse "origin/$BRANCH")"
  merge_base="$(git merge-base HEAD "origin/$BRANCH")"

  if [[ "$local_head" == "$remote_head" ]]; then
    return 0
  fi
  if [[ "$local_head" == "$merge_base" ]]; then
    log "fast-forwarding $BRANCH to origin/$BRANCH"
    git merge --ff-only "origin/$BRANCH" >>"$RUN_LOG" 2>&1 || die "fast-forward merge failed"
    return 0
  fi
  if [[ "$remote_head" == "$merge_base" ]]; then
    log "$BRANCH is ahead of origin/$BRANCH; continuing"
    return 0
  fi

  die "$BRANCH diverged from origin/$BRANCH; skipping automated builder run"
}

codex_prompt() {
  cat <<'PROMPT'
You are Codex running from cron inside the Gormes repository.

Use repo-local skills before substantive work:
- gormes-skill-manager to route the task.
- gormes-hermes-parity to sweep progress.json for new or stale parity work.
- gormes-planner to turn parity findings into builder-ready progress rows.
- gormes-builder to select and implement one row.
- gormes-tdd-slice for red-green-refactor when tests are required.
- gormes-git first, before row selection, again after implementation is complete, and for CI/CD-green repair.

Task:
First invoke gormes-git. If the worktree is dirty, commit every current safe change, make development green, and push origin development before selecting a new row. If the worktree is already clean, record that and continue.

Then invoke gormes-hermes-parity against progress.json. Run a bounded all-topic weakness sweep: find source-backed missing tasks to add, stale complete rows to revisit, vague rows to sharpen, or existing builder-ready rows whose priority should rise. Cover the major Gormes/Hermes surfaces rather than only the most recent subsystem: CLI/TUI, provider/auth, gateway/channels, tools, sessions/memory/Goncho, install/runtime, browser automation, docs/public surfaces, and release/operator flows.

Then invoke gormes-planner on the parity findings and current planned-row count. Convert findings into builder-ready progress.json row changes before implementation. Keep the queue completion-biased: add at most one new source-backed row per cycle, add no new P3/P4 rows while planned rows are 90 or higher, and prefer sharpening, de-duplicating, or reprioritizing existing rows when that is enough. Record implementation intent only in docs/content/building-gormes/architecture_plan/progress.json and regenerate derived progress surfaces when it changes.

Then implement the highest-priority builder-ready new/planned progress.json row after that parity sweep. If the top candidate is not actually ready, pick the next highest-priority builder-ready row or fix the highest-priority failing row that is already in scope. Do exactly one bounded row.

After the final gormes-git push, verify CI/CD for the pushed development HEAD when GitHub status is available. Use gh run/status commands to wait for in-progress checks, inspect failing logs, and repair red CI/CD on development before selecting another parity row. If remote CI/CD cannot be queried, report ci_status=unknown with the exact command failure and do not claim CI/CD is green from local gates alone.

Hard constraints:
- Read AGENTS.md and the selected skill files before editing.
- Stay on the existing development branch only. Do not create branches or worktrees.
- Do not recreate cmd/planner-loop, cmd/builder-loop, or any autonomous loop binary.
- Do not create a backlog outside docs/content/building-gormes/architecture_plan/progress.json.
- Do not skip because the worktree is dirty. Dirty state means gormes-git is the first task.
- Do not treat the parity sweep as implementation. It may add, revisit, prioritize, or sharpen progress rows; runtime code belongs to the subsequent builder/TDD step.
- Preserve any user or parallel-agent changes if they appear while working.
- Use the row write_scope. If the row is wrong or vague, refine the row through the planner workflow and stop after validation.
- Use TDD when the row has tests or observable behavior.
- When a row is complete, update progress evidence/status and regenerate derived progress surfaces with go run ./cmd/progress write.
- Run the row test commands, then the required repo gate: go test ./... -count=1, go run ./cmd/progress validate, and git diff --check. Run docs/landing public-surface gates when docs or web assets changed.
- Finish by invoking gormes-git to commit all current work coherently and push origin development. If this cannot be done, leave the worktree and log in a clearly recoverable state; do not force push.
- If CI/CD is red after a push, make the next loop task a focused CI repair. Do not start new parity or builder work until the development branch is locally green and remote CI/CD is green or explicitly ci_status=unknown due to an unavailable GitHub status query.
PROMPT
}

main() {
  exec 9>"$LOCK_FILE"
  if ! flock -n 9; then
    log "another codexu builder run is active; skipping"
    exit 0
  fi

  cd "$REPO_ROOT"
  log "starting codexu builder cron in $REPO_ROOT"

  ensure_development_branch
  if worktree_dirty; then
    log "worktree is dirty; codexu will run gormes-git first"
  else
    sync_development_branch
  fi

  case "$BACKEND" in
    codexu|opencode) ;;
    *) die "unsupported GORMES_BUILDER_BACKEND: $BACKEND (expected codexu|opencode)" ;;
  esac

  local resolved_bin
  if [[ "$BACKEND" == "opencode" ]]; then
    resolved_bin="$(resolve_opencode)" || die "opencode command not found"
    log "using opencode at $resolved_bin with model $OPENCODE_MODEL"
  else
    resolved_bin="$(resolve_codexu)" || die "codexu command not found"
    log "using codexu at $resolved_bin"
  fi

  if [[ "${1:-}" == "--dry-run" ]]; then
    log "dry run: would execute $BACKEND builder prompt with timeout $RUN_TIMEOUT"
    codex_prompt
    exit 0
  fi

  set +e
  local status
  if [[ "$BACKEND" == "opencode" ]]; then
    local opencode_log="$LOG_DIR/$RUN_ID.opencode.jsonl"
    codex_prompt | timeout "$RUN_TIMEOUT" "$resolved_bin" run \
      --model "$OPENCODE_MODEL" \
      --dir "$REPO_ROOT" \
      --format json \
      --dangerously-skip-permissions \
      >"$opencode_log" 2>>"$RUN_LOG"
    status=$?
    cat "$opencode_log" >>"$RUN_LOG"
    if command -v jq >/dev/null 2>&1 && [[ -s "$opencode_log" ]]; then
      jq -r 'select(.type=="text") | .part.text' "$opencode_log" 2>/dev/null \
        | awk 'NF' \
        | tail -n 1 >"$LAST_MESSAGE" || true
    else
      printf 'opencode backend: see %s for full transcript\n' "$opencode_log" >"$LAST_MESSAGE"
    fi
  else
    codex_prompt | timeout "$RUN_TIMEOUT" "$resolved_bin" exec \
      --sandbox danger-full-access \
      -c 'approval_policy="never"' \
      -C "$REPO_ROOT" \
      --color never \
      --output-last-message "$LAST_MESSAGE" \
      - >>"$RUN_LOG" 2>&1
    status=$?
  fi
  set -e

  if [[ "$status" -eq 124 ]]; then
    die "$BACKEND run timed out after $RUN_TIMEOUT"
  fi
  if [[ "$status" -ne 0 ]]; then
    die "$BACKEND run failed with status $status"
  fi

  log "$BACKEND builder cron finished"
}

main "$@"
