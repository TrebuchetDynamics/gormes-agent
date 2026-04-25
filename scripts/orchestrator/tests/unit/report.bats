#!/usr/bin/env bats

load '../lib/test_env'

setup() {
  load_helpers
  source_lib common
  source_lib candidates
  source_lib report
  source_lib failures
  source_lib worktree
}

@test "collect_final_report_issues passes on good fixture" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/good.final.md"
  assert_success
  assert_output ""
}

@test "collect_final_report_issues fails on missing section" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/bad-missing-section.final.md"
  assert_failure
  assert_output --partial "REFACTOR proof"
}

@test "collect_final_report_issues fails on missing commit hash" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/bad-no-commit-hash.final.md"
  assert_failure
  assert_output --partial "Commit hash"
}

@test "collect_final_report_issues fails on all-zero exits" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/bad-all-zero-exits.final.md"
  assert_failure
  assert_output --partial "non-zero RED exit"
}

@test "collect_final_report_issues fails on zero RED exit" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/bad-no-red-exit.final.md"
  assert_failure
}

@test "collect_final_report_issues accepts explained non-zero RED exit" {
  local tmp report
  tmp="$(mktmp_workspace)"
  report="$tmp/good-explained-red-exit.final.md"
  sed '0,/^Exit: 1$/s//Exit: 2 (gateway build failure) and 1 (cmd test failure)/' \
    "$FIXTURES_DIR/reports/good.final.md" > "$report"

  run collect_final_report_issues "$report"
  assert_success
  assert_output ""
}

@test "collect_final_report_issues fails on missing branch" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/bad-missing-branch.final.md"
  assert_failure
  assert_output --partial "Branch field"
}

@test "collect_final_report_issues fails on empty report" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/bad-empty.final.md"
  assert_failure
}

@test "collect_final_report_issues errors when report file missing" {
  run collect_final_report_issues "/nonexistent/final.md"
  assert_failure
  assert_output --partial "Missing final report"
}

@test "extract_report_commit strips backticks" {
  local tmp
  tmp="$(mktmp_workspace)"
  printf 'Commit: `abc1234def5678`\n' > "$tmp/r.md"
  run extract_report_commit "$tmp/r.md"
  assert_output "abc1234def5678"
}

@test "extract_report_branch reads plain value" {
  local tmp
  tmp="$(mktmp_workspace)"
  printf 'Branch: codexu/foo/worker1\n' > "$tmp/r.md"
  run extract_report_branch "$tmp/r.md"
  assert_output "codexu/foo/worker1"
}

@test "extract_report_field returns empty when label absent" {
  local tmp
  tmp="$(mktmp_workspace)"
  printf 'hello\n' > "$tmp/r.md"
  run extract_report_field "Commit" "$tmp/r.md"
  assert_output ""
}

@test "collect_final_report_issues accepts optional section 10 Runtime flags" {
  local tmp
  tmp="$(mktmp_workspace)"
  local report="$tmp/good-with-section10.final.md"
  cat "$FIXTURES_DIR/reports/good.final.md" > "$report"
  printf '\n10) Runtime flags\nAllowMultiCommit: true\nTolerateWorktreeUntracked: true\n' >> "$report"
  run collect_final_report_issues "$report"
  assert_success
  assert_output ""
}

@test "collect_final_report_issues accepts sections 9 (Acceptance) and 10 (Runtime flags) together" {
  local tmp
  tmp="$(mktmp_workspace)"
  local report="$tmp/good-with-sections-9-and-10.final.md"
  # good.final.md already has section 9 Acceptance check; append section 10.
  cat "$FIXTURES_DIR/reports/good.final.md" > "$report"
  printf '\n10) Runtime flags\nAllowMultiCommit: true\n' >> "$report"
  run collect_final_report_issues "$report"
  assert_success
  assert_output ""
}

@test "collect_final_report_issues rejects report with no Acceptance section" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/bad-no-acceptance.final.md"
  assert_failure
  assert_output --partial "Acceptance check"
}

@test "collect_final_report_issues rejects Acceptance section with a FAIL criterion" {
  run collect_final_report_issues "$FIXTURES_DIR/reports/bad-acceptance-fail.final.md"
  assert_failure
  assert_output --partial "failing criterion"
}

@test "build_prompt announces acceptance-criteria contract" {
  local tmp
  tmp="$(mktmp_workspace)"
  export STATE_DIR="$tmp/state"
  export WORKTREES_DIR="$tmp/worktrees"
  export REPO_SUBDIR="."
  export RUN_ID="testrun"
  export BASE_COMMIT="abc1234"
  export PROGRESS_JSON_REL="docs/progress.json"
  mkdir -p "$STATE_DIR"
  local selected='{"phase_id":"1","subphase_id":"1.A","item_name":"Item A1","status":"planned"}'
  local prompt_file="$tmp/prompt.txt"
  run build_prompt 1 "$selected" "0:1/1.A/Item A1" "$prompt_file"
  assert_success
  run cat "$prompt_file"
  assert_success
  assert_output --partial "ACCEPTANCE CRITERIA"
  assert_output --partial "9) Acceptance check"
  assert_output --partial "Criterion:"
  assert_output --partial "Autoloop control plane:"
  assert_output --partial "scripts/gormes-auto-codexu-orchestrator.sh"
  assert_output --partial "- Agent queue docs: docs/content/building-gormes/autoloop/agent-queue.md"
  assert_output --partial "- Progress schema docs: docs/content/building-gormes/autoloop/progress-schema.md"
  assert_output --partial "docs/superpowers/plans/2026-04-24-orchestrator-oiling-release-1-plan.md"
  assert_output --partial "Candidate policy:"
  assert_output --partial "Skip rows with blocked_by until ready_when is satisfied."
}

@test "build_prompt includes selected progress handoff fields" {
  local tmp
  tmp="$(mktmp_workspace)"
  export STATE_DIR="$tmp/state"
  export WORKTREES_DIR="$tmp/worktrees"
  export REPO_SUBDIR="."
  export RUN_ID="testrun"
  export BASE_COMMIT="abc1234"
  export PROGRESS_JSON_REL="docs/content/building-gormes/architecture_plan/progress.json"
  mkdir -p "$STATE_DIR"
  local selected
  selected='{"phase_id":"4","subphase_id":"4.A","item_name":"Provider harness","status":"in_progress","contract":"Provider transcript contract","contract_status":"fixture_ready","slice_size":"medium","execution_owner":"provider","fixture":"internal/hermes fixtures","degraded_mode":"status reports gaps","ready_when":["fixtures replay"],"not_ready_when":["live provider call required"],"write_scope":["internal/hermes/"],"test_commands":["go test ./internal/hermes -count=1"],"done_signal":["transcripts replay"],"acceptance":["fixture passes"],"source_refs":["docs/content/upstream-hermes/source-study.md"],"unblocks":["Bedrock"],"note":"Keep provider quirks out of kernel."}'
  local prompt_file="$tmp/prompt.txt"
  run build_prompt 1 "$selected" "0:4/4.A/Provider harness" "$prompt_file"
  assert_success
  run cat "$prompt_file"
  assert_success
  assert_output --partial "Canonical progress handoff:"
  assert_output --partial "- Contract: Provider transcript contract"
  assert_output --partial "- Contract status: fixture_ready"
  assert_output --partial "- Slice size: medium"
  assert_output --partial "- Execution owner: provider"
  assert_output --partial "- internal/hermes/"
  assert_output --partial "- go test ./internal/hermes -count=1"
  assert_output --partial "- transcripts replay"
  assert_output --partial "Prefer the declared Write scope and Test commands"
}

@test "build_prompt reads autoloop control plane from selected candidate metadata" {
  local tmp
  tmp="$(mktmp_workspace)"
  export STATE_DIR="$tmp/state"
  export WORKTREES_DIR="$tmp/worktrees"
  export REPO_SUBDIR="."
  export RUN_ID="testrun"
  export BASE_COMMIT="abc1234"
  export PROGRESS_JSON_REL="fallback/progress.json"
  mkdir -p "$STATE_DIR"
  local selected
  selected='{"phase_id":"1","subphase_id":"1.C","item_name":"Autoloop metadata","status":"planned","autoloop":{"entrypoint":"custom-entry.sh","plan":"docs/superpowers/plans/custom-plan.md","agent_queue":"docs/content/building-gormes/custom-queue.md","progress_schema":"docs/content/building-gormes/custom-schema.md","candidate_source":"docs/content/building-gormes/architecture_plan/progress.json","unit_test":"custom test command","candidate_policy":["Skip blocked rows.","Skip umbrella rows."]}}'
  local prompt_file="$tmp/prompt.txt"
  run build_prompt 1 "$selected" "0:1/1.C/Autoloop metadata" "$prompt_file"
  assert_success
  run cat "$prompt_file"
  assert_success
  assert_output --partial "- Main entrypoint: custom-entry.sh"
  assert_output --partial "- Candidate source: docs/content/building-gormes/architecture_plan/progress.json"
  assert_output --partial "- Agent queue docs: docs/content/building-gormes/custom-queue.md"
  assert_output --partial "- Progress schema docs: docs/content/building-gormes/custom-schema.md"
  assert_output --partial "- Orchestrator plan: docs/superpowers/plans/custom-plan.md"
  assert_output --partial "- Orchestrator tests: custom test command"
  assert_output --partial "Candidate policy:"
  assert_output --partial "- Skip blocked rows."
  assert_output --partial "- Skip umbrella rows."
}

@test "build_prompt omits PRIOR ATTEMPT FEEDBACK when no failure record" {
  local tmp
  tmp="$(mktmp_workspace)"
  export STATE_DIR="$tmp/state"
  export WORKTREES_DIR="$tmp/worktrees"
  export REPO_SUBDIR="."
  export RUN_ID="testrun"
  export BASE_COMMIT="abc1234"
  export PROGRESS_JSON_REL="docs/progress.json"
  mkdir -p "$STATE_DIR"
  local selected
  selected='{"phase_id":"1","subphase_id":"1.A","item_name":"Item A1","status":"planned"}'
  local prompt_file="$tmp/prompt.txt"
  run build_prompt 1 "$selected" "0:1/1.A/Item A1" "$prompt_file"
  assert_success
  run cat "$prompt_file"
  assert_success
  refute_output --partial "PRIOR ATTEMPT FEEDBACK"
  assert_output --partial "Mission:"
}

@test "build_prompt injects PRIOR ATTEMPT FEEDBACK when failure record exists" {
  local tmp
  tmp="$(mktmp_workspace)"
  export STATE_DIR="$tmp/state"
  export WORKTREES_DIR="$tmp/worktrees"
  export REPO_SUBDIR="."
  export RUN_ID="testrun"
  export BASE_COMMIT="abc1234"
  export PROGRESS_JSON_REL="docs/progress.json"
  mkdir -p "$STATE_DIR"

  local stderr_file="$tmp/stderr.log"
  printf 'panic: explosive failure at line 9000\nstack trace blah\n' > "$stderr_file"
  local slug
  slug="$(task_slug "1" "1.A" "Item A1")"
  failure_record_write "$slug" "1" "report_validation_failed" "$stderr_file" '["Missing section GREEN proof","Missing Commit hash"]'

  local selected
  selected='{"phase_id":"1","subphase_id":"1.A","item_name":"Item A1","status":"planned"}'
  local prompt_file="$tmp/prompt.txt"
  run build_prompt 1 "$selected" "0:1/1.A/Item A1" "$prompt_file"
  assert_success
  run cat "$prompt_file"
  assert_success
  assert_output --partial "PRIOR ATTEMPT FEEDBACK"
  assert_output --partial "This task has been attempted 1 times before"
  assert_output --partial "report_validation_failed"
  assert_output --partial "Missing section GREEN proof"
  assert_output --partial "Missing Commit hash"
  assert_output --partial "panic: explosive failure"
  assert_output --partial "Mission:"
}
