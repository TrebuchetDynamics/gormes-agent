#!/usr/bin/env bash
# Report parsing and prompt construction.
# Depends on: $FINAL_REPORT_GRACE_SECONDS, $PROGRESS_JSON_REL, $BASE_COMMIT.
# Note: build_prompt references worker_repo_root / worker_branch_name which are still
# defined in the entry script until Task 6 extracts worktree.sh — that's fine because
# bash resolves function names at call time, not source time.

build_prompt() {
  local worker_id="$1"
  local selected_json="$2"
  local trail="$3"
  local prompt_file="$4"

  local phase_id subphase_id item_name status branch worker_dir
  phase_id="$(jq -r '.phase_id' <<<"$selected_json")"
  subphase_id="$(jq -r '.subphase_id' <<<"$selected_json")"
  item_name="$(jq -r '.item_name' <<<"$selected_json")"
  status="$(jq -r '.status' <<<"$selected_json")"
  branch="$(worker_branch_name "$worker_id")"
  worker_dir="$(worker_repo_root "$worker_id")"

  cat > "$prompt_file" <<EOF
Repository root:
  $worker_dir

Mission:
Pick exactly ONE unfinished phase task and complete it with strict Test-Driven Development (TDD), while documenting progress and related docs before and after implementation.

Coordinator-selected worker lane:
- WORKER_ID: $worker_id
- Deterministic index trail: $trail

Selected task:
- Phase/Subphase/Item: $phase_id / $subphase_id / $item_name
- Current status: $status

Isolated execution context:
- Git worktree: $worker_dir
- Git branch: $branch
- Base commit: $BASE_COMMIT

Selection contract:
- This task was preselected by the coordinator from:
  $PROGRESS_JSON_REL
- Do not switch to another task in this run.
- Keep all changes inside the current repository root.
- Create exactly one new commit on $branch.
- Do not amend, squash, or rewrite history.
- Leave the worktree clean after your commit.
- If blocked/conflict/not-viable, provide the exact command + exact error and stop without creating extra commits.

==================================================
1) DOCUMENTATION GATE (MANDATORY: BEFORE + AFTER)
==================================================
Before writing tests:
- Record current status of selected item in progress.json
- Identify all related docs/progress files likely impacted
- Add a short implementation intent note in your report

After implementation/tests:
- Update progress/docs that reflect completed behavior
- Regenerate/validate progress artifacts when relevant
- Include exact file paths + summaries of doc/progress edits
- Do not claim completion if docs/progress are stale

==================================================
2) TDD PROTOCOL (MANDATORY)
==================================================
A) RED
- Write failing test(s) first.
- Run targeted tests and capture the failing command, exit code, and a short failure snippet.

B) GREEN
- Implement minimal code to satisfy tests.
- Re-run the targeted test command and capture the passing command, exit code, and a short success snippet.

C) REFACTOR
- Improve structure/naming without changing behavior.
- Re-run targeted tests and capture the command, exit code, and a short success snippet.

D) REGRESSION
- Run broader tests for touched packages.
- If progress/docs changed, also run:
  - go run ./cmd/progress-gen -write
  - go run ./cmd/progress-gen -validate
  - related progress assertions/tests
- Capture the regression command, exit code, and a short success snippet.

==================================================
3) REQUIRED FINAL REPORT FORMAT
==================================================
1) Selected task
Task: <phase / subphase / item>

2) Pre-doc baseline
Files:
- <path>

3) RED proof
Command: <exact command>
Exit: <non-zero integer>
Snippet: <short failure output>

4) GREEN proof
Command: <exact command>
Exit: 0
Snippet: <short passing output>

5) REFACTOR proof
Command: <exact command>
Exit: 0
Snippet: <short passing output>

6) Regression proof
Command: <exact command>
Exit: 0
Snippet: <short passing output>

7) Post-doc closeout
Files:
- <path>

8) Commit
Branch: $branch
Commit: <git hash>
Files:
- <path>
EOF
}

collect_final_report_issues() {
  local final_file="$1"
  local missing=0

  [[ -f "$final_file" ]] || {
    echo "Missing final report: $final_file"
    return 1
  }

  local -a section_titles=(
    "Selected task"
    "Pre-doc baseline"
    "RED proof"
    "GREEN proof"
    "REFACTOR proof"
    "Regression proof"
    "Post-doc closeout"
    "Commit"
  )
  local i section_title section_pattern
  for (( i = 0; i < ${#section_titles[@]}; i++ )); do
    section_title="${section_titles[$i]}"
    # Accept optional leading markdown header prefix (#, ##, ..., ######) and
    # optional trailing/surrounding ** bold markers so claude's markdown-style
    # reports validate. The required content is the "N) <Title>" or "N. <Title>".
    section_pattern="^[[:space:]]*(#{1,6}[[:space:]]+)?(\\*\\*)?$((i + 1))[).][[:space:]]*(\\*\\*)?${section_title}(\\*\\*)?[[:space:]]*$"
    if ! grep -Eq "$section_pattern" "$final_file"; then
      echo "Missing section '$((i + 1))) ${section_title}' in $final_file"
      missing=1
    fi
  done

  local command_count exit_count zero_count
  command_count="$(grep -Ec '^[[:space:]]*Command: .+' "$final_file" || true)"
  exit_count="$(grep -Ec '^[[:space:]]*Exit: ' "$final_file" || true)"
  zero_count="$(grep -Ec '^[[:space:]]*Exit:[[:space:]]*`?0`?[[:space:]]*$' "$final_file" || true)"

  if (( command_count < 4 )); then
    echo "Final report missing command evidence in $final_file"
    missing=1
  fi
  if (( exit_count < 4 )); then
    echo "Final report missing exit-code evidence in $final_file"
    missing=1
  fi
  if ! grep -Eq '^[[:space:]]*Exit:[[:space:]]*`?[1-9][0-9]*`?[[:space:]]*$' "$final_file"; then
    echo "Final report missing non-zero RED exit code in $final_file"
    missing=1
  fi
  if (( zero_count < 3 )); then
    echo "Final report missing GREEN/REFACTOR/REGRESSION zero exits in $final_file"
    missing=1
  fi
  if ! grep -Eq '^[[:space:]]*Branch:[[:space:]]*`?.+`?[[:space:]]*$' "$final_file"; then
    echo "Final report missing Branch field in $final_file"
    missing=1
  fi
  if ! grep -Eq '^[[:space:]]*Commit:[[:space:]]*`?[0-9a-f]{7,40}`?[[:space:]]*$' "$final_file"; then
    echo "Final report missing Commit hash in $final_file"
    missing=1
  fi

  return "$missing"
}

verify_final_report() {
  local final_file="$1"
  local issues=""

  issues="$(collect_final_report_issues "$final_file")"
  if [[ $? -eq 0 ]]; then
    return 0
  fi

  [[ -n "$issues" ]] && printf '%s\n' "$issues" >&2
  return 1
}

print_final_report_diagnostics() {
  local worker_id="$1"
  local final_file="$2"
  local stderr_file="$3"
  local jsonl_file="$4"

  echo "worker[$worker_id]: final report validation failed after ${FINAL_REPORT_GRACE_SECONDS}s" >&2
  echo "worker[$worker_id]: artifacts final=$final_file stderr=$stderr_file jsonl=$jsonl_file" >&2

  if [[ -f "$final_file" ]]; then
    echo "worker[$worker_id]: tail(final report)" >&2
    tail -n 20 "$final_file" >&2 || true
  else
    echo "worker[$worker_id]: final report file not found" >&2
  fi

  if [[ -s "$stderr_file" ]]; then
    echo "worker[$worker_id]: tail(stderr)" >&2
    tail -n 20 "$stderr_file" >&2 || true
  fi

  if [[ -s "$jsonl_file" ]]; then
    echo "worker[$worker_id]: tail(jsonl)" >&2
    tail -n 20 "$jsonl_file" >&2 || true
  fi
}

wait_for_valid_final_report() {
  local worker_id="$1"
  local final_file="$2"
  local stderr_file="$3"
  local jsonl_file="$4"
  local attempts attempt issues rc

  attempts=$(( FINAL_REPORT_GRACE_SECONDS * 10 + 1 ))
  (( attempts < 1 )) && attempts=1

  for (( attempt = 1; attempt <= attempts; attempt++ )); do
    issues="$(collect_final_report_issues "$final_file")"
    rc=$?
    if [[ "$rc" == "0" ]]; then
      return 0
    fi
    if (( attempt < attempts )); then
      sleep 0.1
    fi
  done

  [[ -n "$issues" ]] && printf '%s\n' "$issues" >&2
  print_final_report_diagnostics "$worker_id" "$final_file" "$stderr_file" "$jsonl_file"
  return 1
}

extract_report_field() {
  local label="$1"
  local final_file="$2"
  local value

  value="$(
    awk -v label="$label" '
      $0 ~ "^[[:space:]]*" label ":" {
        sub("^[[:space:]]*" label ":[[:space:]]*", "", $0)
        sub("[[:space:]]*$", "", $0)
        print
        exit
      }
    ' "$final_file"
  )"
  value="${value#\`}"
  value="${value%\`}"
  printf '%s\n' "$value"
}

extract_report_commit() {
  local final_file="$1"
  extract_report_field "Commit" "$final_file"
}

extract_report_branch() {
  local final_file="$1"
  extract_report_field "Branch" "$final_file"
}

extract_session_id() {
  local jsonl_file="$1"
  [[ -f "$jsonl_file" ]] || return 0
  jq -r 'select(.type=="thread.started") | (.thread_id // .session_id // empty)' "$jsonl_file" | head -n1
}
