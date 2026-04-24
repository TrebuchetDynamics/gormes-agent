#!/usr/bin/env bats

load '../lib/test_env'

setup() {
  load_helpers
  source_lib common
}

@test "classify_worker_failure maps 124 to timeout" {
  run classify_worker_failure 124
  assert_success
  assert_output "timeout"
}

@test "classify_worker_failure maps 137 to killed" {
  run classify_worker_failure 137
  assert_output "killed"
}

@test "classify_worker_failure maps 1 to contract_or_test_failure" {
  run classify_worker_failure 1
  assert_output "contract_or_test_failure"
}

@test "classify_worker_failure maps other to worker_error" {
  run classify_worker_failure 42
  assert_output "worker_error"
}

@test "worker_status_outcome does not count failed soft-success task slug as success" {
  run worker_status_outcome "worker[3]: failed(1) -> 1__1.c__soft-success-nonzero-bats-coverage"
  assert_success
  assert_output "failed"
}

@test "worker_status_outcome maps explicit soft-success status to success" {
  run worker_status_outcome "worker[3]: soft-success(nonzero=1) -> task-slug (abcdef1)"
  assert_success
  assert_output "success"
}

@test "provider_quota_exhausted detects codex usage limit final message" {
  local final_file
  final_file="$BATS_TEST_TMPDIR/quota.final.md"
  printf "You've hit your limit resets 8:20am (America/Monterrey)\n" > "$final_file"

  run provider_quota_exhausted "$final_file" "" ""
  assert_success
}

@test "provider_quota_message returns the matched quota line" {
  local final_file
  final_file="$BATS_TEST_TMPDIR/quota-message.final.md"
  printf "You've hit your limit resets 8:20am (America/Monterrey)\n" > "$final_file"

  run provider_quota_message "$final_file" "" ""
  assert_success
  assert_output --partial "You've hit your limit"
}

@test "safe_path_token strips unsafe characters" {
  run safe_path_token "Feat/sub phase: X_Y.Z@v1"
  assert_output "Feat-sub-phase-X_Y.Z-v1"
}

@test "safe_path_token trims leading and trailing dashes" {
  run safe_path_token "///foo///"
  assert_output "foo"
}

@test "require_cmd succeeds for a real command" {
  run require_cmd bash
  assert_success
}

@test "require_cmd fails for a bogus command" {
  run require_cmd bogus_cmd_that_does_not_exist_xyz
  assert_failure
}

@test "available_mem_mb returns a positive integer" {
  run available_mem_mb
  assert_success
  [[ "$output" =~ ^[0-9]+$ ]]
  (( output > 0 ))
}
