#!/usr/bin/env bats

load '../lib/test_env'

setup() {
  load_helpers
  source_lib candidates
  export PROGRESS_JSON="$FIXTURES_DIR/progress.fixture.json"
}

@test "normalize_candidates drops complete items" {
  export ACTIVE_FIRST=1
  run normalize_candidates
  assert_success
  # Should include 4 non-complete items: A2, B1, B2, 2A1, 2A2 = 5 items (not 6)
  local count
  count=$(echo "$output" | jq 'length')
  assert_equal "$count" "5"
  echo "$output" | jq -e '.[].status != "complete"'
}

@test "normalize_candidates orders in_progress before planned when ACTIVE_FIRST=1" {
  export ACTIVE_FIRST=1
  run normalize_candidates
  assert_success
  local first_status
  first_status=$(echo "$output" | jq -r '.[0].status')
  assert_equal "$first_status" "in_progress"
}

@test "normalize_candidates keeps agent handoff metadata and skips blocked umbrella rows" {
  local tmp
  tmp="$(mktmp_workspace)"
  cat > "$tmp/progress.json" <<'JSON'
{
  "meta": {
    "autoloop": {
      "entrypoint": "scripts/gormes-auto-codexu-orchestrator.sh",
      "candidate_policy": ["skip blocked rows"]
    }
  },
  "phases": {
    "1": {
      "subphases": {
        "1.A": {
          "items": [
            {
              "name": "Ready slice",
              "status": "planned",
              "contract": "Ready contract",
              "slice_size": "small",
              "execution_owner": "gateway",
              "write_scope": ["internal/gateway/"],
              "test_commands": ["go test ./internal/gateway -count=1"],
              "done_signal": ["gateway fixtures pass"],
              "source_refs": ["docs/content/building-gormes/autoloop/agent-queue.md"]
            },
            {
              "name": "Blocked slice",
              "status": "planned",
              "blocked_by": ["Ready slice"]
            },
            {
              "name": "Umbrella slice",
              "status": "planned",
              "slice_size": "umbrella"
            }
          ]
        }
      }
    }
  }
}
JSON
  export PROGRESS_JSON="$tmp/progress.json"
  run normalize_candidates
  assert_success
  local names
  names="$(echo "$output" | jq -r '[.[].item_name] | join(",")')"
  assert_equal "$names" "Ready slice"
  echo "$output" | jq -e '.[0].agent_ready == true'
  echo "$output" | jq -e '.[0].contract == "Ready contract"'
  echo "$output" | jq -e '.[0].write_scope == ["internal/gateway/"]'
  echo "$output" | jq -e '.[0].test_commands == ["go test ./internal/gateway -count=1"]'
  echo "$output" | jq -e '.[0].done_signal == ["gateway fixtures pass"]'
  echo "$output" | jq -e '.[0].autoloop.entrypoint == "scripts/gormes-auto-codexu-orchestrator.sh"'
  echo "$output" | jq -e '.[0].autoloop.candidate_policy == ["skip blocked rows"]'
}

@test "normalize_candidates does not prioritize when ACTIVE_FIRST=0" {
  export ACTIVE_FIRST=0
  run normalize_candidates
  assert_success
  # Must not group by status — lexical phase/subphase/item order instead
  local order
  order=$(echo "$output" | jq -r '[.[] | .phase_id + "/" + .subphase_id + "/" + .item_name] | join(",")')
  [[ "$order" == "1/1.A/Item A2,1/1.B/Item B1,1/1.B/Item B2,2/2.A/Item 2A1,2/2.A/Item 2A2" ]]
}

@test "normalize_candidates returns empty array for progress.empty.json" {
  export PROGRESS_JSON="$FIXTURES_DIR/progress.empty.json"
  run normalize_candidates
  assert_success
  assert_output "[]"
}

@test "normalize_candidates returns empty array when all items complete" {
  export PROGRESS_JSON="$FIXTURES_DIR/progress.all-complete.json"
  run normalize_candidates
  assert_success
  assert_output "[]"
}

@test "task_slug lowercases and sanitizes" {
  run task_slug "3" "3.E" "Cross Chat Merge (v2)"
  assert_output "3__3.e__cross-chat-merge-v2"
}

@test "candidate_count reports length from CANDIDATES_FILE" {
  local tmp
  tmp="$(mktmp_workspace)"
  echo '[{"a":1},{"b":2},{"c":3}]' > "$tmp/cands.json"
  export CANDIDATES_FILE="$tmp/cands.json"
  run candidate_count
  assert_output "3"
}

@test "candidate_at returns JSON object at index" {
  local tmp
  tmp="$(mktmp_workspace)"
  echo '[{"k":"a"},{"k":"b"}]' > "$tmp/cands.json"
  export CANDIDATES_FILE="$tmp/cands.json"
  run candidate_at 1
  assert_output '{"k":"b"}'
}

@test "PHASE_FLOOR=2 filters out phase 3+ candidates" {
  export PHASE_FLOOR=2
  run normalize_candidates
  assert_success
  # Manually build a file that mimics what write_candidates_file would see
  local tmp="$(mktmp_workspace)/c.json"
  normalize_candidates > "$tmp"
  local out="$(mktmp_workspace)/out.json"
  apply_phase_floor "$tmp" "$out"
  local phase3_count
  phase3_count="$(jq '[.[] | select(.phase_id == "3")] | length' "$out")"
  local phase2_count
  phase2_count="$(jq '[.[] | select(.phase_id == "2")] | length' "$out")"
  assert_equal "$phase3_count" "0"
  (( phase2_count > 0 )) || (( $(jq 'length' "$tmp") == 0 ))
}

@test "PHASE_FLOOR unset is a no-op" {
  unset PHASE_FLOOR || true
  local tmp="$(mktmp_workspace)/c.json"
  normalize_candidates > "$tmp"
  local out="$(mktmp_workspace)/out.json"
  apply_phase_floor "$tmp" "$out"
  local before after
  before="$(jq 'length' "$tmp")"
  after="$(jq 'length' "$out")"
  assert_equal "$before" "$after"
}

@test "PHASE_FLOOR with garbage value is a no-op (defensive)" {
  export PHASE_FLOOR="abc"
  local tmp="$(mktmp_workspace)/c.json"
  normalize_candidates > "$tmp"
  local out="$(mktmp_workspace)/out.json"
  apply_phase_floor "$tmp" "$out"
  local before after
  before="$(jq 'length' "$tmp")"
  after="$(jq 'length' "$out")"
  assert_equal "$before" "$after"
}

@test "PHASE_SKIP_SUBPHASES excludes listed subphases" {
  # Use progress.fixture.json which has phases 1 and 2. Skip 1.A.
  export PHASE_SKIP_SUBPHASES="1.A"
  local tmp="$(mktmp_workspace)/c.json"
  normalize_candidates > "$tmp"
  local out="$(mktmp_workspace)/out.json"
  apply_phase_skip "$tmp" "$out"
  local n
  n="$(jq '[.[] | select(.subphase_id == "1.A")] | length' "$out")"
  assert_equal "$n" "0"
}

@test "PHASE_SKIP_SUBPHASES is case-insensitive and trims whitespace" {
  export PHASE_SKIP_SUBPHASES=" 1.a , 1.B "
  local tmp="$(mktmp_workspace)/c.json"
  normalize_candidates > "$tmp"
  local out="$(mktmp_workspace)/out.json"
  apply_phase_skip "$tmp" "$out"
  local excluded
  excluded="$(jq '[.[] | select(.subphase_id | ascii_downcase | IN("1.a","1.b"))] | length' "$out")"
  assert_equal "$excluded" "0"
}

@test "PHASE_PRIORITY_BOOST lifts matching subphase above everything else" {
  export PHASE_PRIORITY_BOOST="2.A"
  local tmp="$(mktmp_workspace)/c.json"
  normalize_candidates > "$tmp"
  run jq -rc '.[0] | .subphase_id' "$tmp"
  assert_output "2.A"
}
