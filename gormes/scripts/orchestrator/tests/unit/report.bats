#!/usr/bin/env bats

load '../lib/test_env'

setup() {
  load_helpers
  source_lib report
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
