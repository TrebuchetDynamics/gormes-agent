#!/usr/bin/env bash
# Candidate list normalization helpers.
# Depends on: $PROGRESS_JSON, $ACTIVE_FIRST, $CANDIDATES_FILE (reads only).

normalize_candidates() {
  jq -c --arg active_first "$ACTIVE_FIRST" '
    def status_rank(s):
      if ($active_first == "1") then
        if (s == "in_progress") then 0
        elif (s == "planned") then 1
        else 2 end
      else 0 end;

    [
      (.phases // {})
      | to_entries[]
      | .key as $phase_id
      | (.value.subphases // .value.sub_phases // {})
      | to_entries[]
      | .key as $subphase_id
      | (.value.items // [])[]
      | {
          phase_id: $phase_id,
          subphase_id: $subphase_id,
          item_name: (.item_name // .name // .title // .id),
          status: ((.status // "unknown") | tostring | ascii_downcase)
        }
      | select(.item_name != null and .item_name != "")
      | select(.status != "complete")
      | . + {status_rank: status_rank(.status)}
    ]
    | unique_by([.phase_id, .subphase_id, .item_name])
    | sort_by([.status_rank, .phase_id, .subphase_id, .item_name])
    | map(del(.status_rank))
  ' "$PROGRESS_JSON"
}

write_candidates_file() {
  normalize_candidates > "$CANDIDATES_FILE"
}

candidate_count() {
  jq 'length' "$CANDIDATES_FILE"
}

candidate_at() {
  local idx="$1"
  jq -c ".[$idx]" "$CANDIDATES_FILE"
}

task_slug() {
  local phase_id="$1"
  local subphase_id="$2"
  local item_name="$3"

  printf '%s__%s__%s' "$phase_id" "$subphase_id" "$item_name" \
    | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[^a-z0-9._-]+/-/g; s/^-+//; s/-+$//; s/--+/-/g'
}
