#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

hours=12
gormes_home="${GORMES_HOME:-$HOME/.gormes}"
hermes_home="${HERMES_HOME:-$HOME/.hermes}"
modules_dir="$repo_root/docs/content/building-gormes/modules"
codexu_bin="${CODEXU_BIN:-codexu}"
out_dir=""
dry_run=0
include_hermes=0
include_journal=1
max_turns=120
max_tool_lines=160
max_session_lines=220
max_journal_lines=220

usage() {
  cat <<'USAGE'
Usage: scripts/audit-profile-sessions.sh [options]

Collect redacted evidence from every Gormes profile touched in the last 12
hours, then ask codexu to audit agent-response issues and map fixes back to
docs/content/building-gormes/modules.

Options:
  --hours N             Look back N hours. Default: 12.
  --gormes-home PATH    Gormes runtime home. Default: $GORMES_HOME or ~/.gormes.
  --hermes-home PATH    Hermes runtime home for --include-hermes. Default: ~/.hermes.
  --modules-dir PATH    Building-Gormes module docs directory.
  --out PATH            Output directory. Default: .codex/profile-session-audits/<utc>.
  --codexu PATH         codexu binary. Default: CODEXU_BIN or codexu.
  --include-hermes      Also scan ~/.hermes and ~/.hermes/profiles.
  --no-journal          Skip user-systemd journal snippets.
  --dry-run             Build bundle and prompt, but do not invoke codexu.
  --max-turns N         Max recent memory turns per profile. Default: 120.
  --max-tool-lines N    Max tool audit rows per profile. Default: 160.
  -h, --help            Show this help.
USAGE
}

die() {
  printf 'audit-profile-sessions: %s\n' "$*" >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --hours)
      [ "$#" -ge 2 ] || die "--hours requires a value"
      hours="$2"
      shift 2
      ;;
    --gormes-home)
      [ "$#" -ge 2 ] || die "--gormes-home requires a value"
      gormes_home="$2"
      shift 2
      ;;
    --hermes-home)
      [ "$#" -ge 2 ] || die "--hermes-home requires a value"
      hermes_home="$2"
      shift 2
      ;;
    --modules-dir)
      [ "$#" -ge 2 ] || die "--modules-dir requires a value"
      modules_dir="$2"
      shift 2
      ;;
    --out)
      [ "$#" -ge 2 ] || die "--out requires a value"
      out_dir="$2"
      shift 2
      ;;
    --codexu)
      [ "$#" -ge 2 ] || die "--codexu requires a value"
      codexu_bin="$2"
      shift 2
      ;;
    --include-hermes)
      include_hermes=1
      shift
      ;;
    --no-journal)
      include_journal=0
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --max-turns)
      [ "$#" -ge 2 ] || die "--max-turns requires a value"
      max_turns="$2"
      shift 2
      ;;
    --max-tool-lines)
      [ "$#" -ge 2 ] || die "--max-tool-lines requires a value"
      max_tool_lines="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

case "$hours" in
  ''|*[!0-9]*) die "--hours must be a positive integer" ;;
esac
[ "$hours" -gt 0 ] || die "--hours must be greater than zero"

for numeric in "$max_turns" "$max_tool_lines" "$max_session_lines" "$max_journal_lines"; do
  case "$numeric" in
    ''|*[!0-9]*) die "limits must be positive integers" ;;
  esac
  [ "$numeric" -gt 0 ] || die "limits must be greater than zero"
done

[ -d "$repo_root/.git" ] || die "repo root not found: $repo_root"
[ -d "$modules_dir" ] || die "modules dir not found: $modules_dir"

stamp=$(date -u +%Y%m%dT%H%M%SZ)
if [ -z "$out_dir" ]; then
  out_dir="$repo_root/.codex/profile-session-audits/$stamp"
fi
mkdir -p "$out_dir"

bundle="$out_dir/bundle.md"
prompt="$out_dir/codexu-prompt.md"
report="$out_dir/codexu-audit.md"
cutoff_epoch=$(date -u -d "$hours hours ago" +%s)
cutoff_iso=$(date -u -d "@$cutoff_epoch" +%Y-%m-%dT%H:%M:%SZ)
minutes=$((hours * 60))

redact_stream() {
  perl -pe '
    BEGIN {
      $home = quotemeta($ENV{"HOME"} // "");
    }
    s/$home/~/g if $home ne "";
    s/(Authorization:[[:space:]]*Bearer[[:space:]]+)[A-Za-z0-9._~+\/=-]+/${1}[REDACTED_BEARER]/gi;
    s/(api[_-]?key|token|secret|password|client[_-]?secret|refresh[_-]?token|access[_-]?token)([[:space:]]*["=:]+[[:space:]]*)[^,}\s"]+/${1}${2}[REDACTED]/gi;
    s/\bsk-[A-Za-z0-9_-]{20,}\b/[REDACTED_OPENAI_KEY]/g;
    s/\bgh[pousr]_[A-Za-z0-9_]{20,}\b/[REDACTED_GITHUB_TOKEN]/g;
    s/\b[0-9]{8,10}:[A-Za-z0-9_-]{30,}\b/[REDACTED_TELEGRAM_TOKEN]/g;
    s/\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b/[REDACTED_JWT]/g;
    s/\b(agent:[A-Za-z0-9_:-]*telegram:[A-Za-z0-9_:-]*:)[0-9]{6,}\b/${1}[REDACTED_ID]/g;
    s/\b(telegram:[A-Za-z0-9_:-]*:)[0-9]{6,}\b/${1}[REDACTED_ID]/g;
    s/\b(telegram:)[0-9]{6,}\b/${1}[REDACTED_ID]/g;
    s/\b(chat_id|user_id|peer_id|workspace_id|observer_peer_id)([":[:space:]=]+)[0-9]{6,}\b/${1}${2}[REDACTED_ID]/gi;
    s/\b[A-Fa-f0-9]{64,}\b/[REDACTED_HEX]/g;
    s/\b[A-Za-z0-9+\/_-]{96,}={0,2}\b/[REDACTED_LONG_TOKEN]/g;
  '
}

append_redacted() {
  redact_stream >> "$bundle"
}

append_section() {
  printf '\n## %s\n\n' "$1" >> "$bundle"
}

display_path() {
  local path=$1
  if [[ "$path" == "$HOME"* ]]; then
    printf '~%s' "${path#"$HOME"}"
  else
    printf '%s' "$path"
  fi
}

is_sensitive_path() {
  case "$1" in
    *auth.json|*.env|*.pem|*.key|*credentials*|*secret*|*password*|*token*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

sqlite_query() {
  local db_path=$1
  local sql=$2
  sqlite3 -readonly -separator $'\t' "$db_path" "$sql"
}

jsonl_events() {
  local file=$1
  local limit=$2
  if command -v jq >/dev/null 2>&1; then
    if jq -r --argjson cutoff "$cutoff_epoch" '
      def raw_ts: (.timestamp // .ts // .created_at // .time // null);
      def epoch:
        raw_ts as $t
        | if ($t | type) == "number" then $t
          elif ($t | type) == "string" then ($t | fromdateiso8601? // 0)
          else 0 end;
      def compact:
        tostring
        | gsub("\r"; " ")
        | gsub("\n"; " ")
        | gsub("[[:space:]]+"; " ")
        | .[0:1800];
      select(epoch >= $cutoff or epoch == 0)
      | [
          (raw_ts // ""),
          (.role // .type // .source // .tool // ""),
          (.session_id // .agent_id // ""),
          ((.content // .message // .text // .result // .error // .args // .) | compact)
        ]
      | @tsv
    ' "$file" 2>/dev/null | tail -n "$limit"; then
      return 0
    fi
  fi
  tail -n "$limit" "$file"
}

tool_audit_events() {
  local file=$1
  if command -v jq >/dev/null 2>&1; then
    if jq -r --argjson cutoff "$cutoff_epoch" '
      def raw_ts: (.timestamp // .ts // .created_at // .time // null);
      def epoch:
        raw_ts as $t
        | if ($t | type) == "number" then $t
          elif ($t | type) == "string" then ($t | fromdateiso8601? // 0)
          else 0 end;
      def compact:
        tostring
        | gsub("\r"; " ")
        | gsub("\n"; " ")
        | gsub("[[:space:]]+"; " ")
        | .[0:1800];
      select(epoch >= $cutoff or epoch == 0)
      | [
          (raw_ts // ""),
          (.source // ""),
          (.session_id // ""),
          (.agent_id // ""),
          (.tool // ""),
          (.status // ""),
          (.duration_ms // ""),
          ((.args // {}) | compact),
          ((.error // "") | compact)
        ]
      | @tsv
    ' "$file" 2>/dev/null | tail -n "$max_tool_lines"; then
      return 0
    fi
  fi
  tail -n "$max_tool_lines" "$file"
}

collect_modules() {
  append_section "Building-Gormes Module Map"
  {
    printf 'Module docs path: %s\n\n' "$modules_dir"
    printf 'Modules:\n'
    find "$modules_dir" -maxdepth 1 -type f -name '*.md' | sort | while IFS= read -r file; do
      title=$(awk '/^# / { sub(/^# /, ""); print; exit }' "$file")
      [ -n "$title" ] || title=$(basename "$file" .md)
      printf '%s\n' "- $(basename "$file" .md): $title"
    done
    printf '\nKey headings:\n'
    if command -v rg >/dev/null 2>&1; then
      rg -n '^#{1,3}[[:space:]]' "$modules_dir"/*.md | head -n 500
    else
      grep -HnE '^#{1,3}[[:space:]]' "$modules_dir"/*.md | head -n 500
    fi
  } | append_redacted
}

collect_memory_markdown() {
  local root=$1
  local memory_dir="$root/memory"
  [ -d "$memory_dir" ] || return 0

  find "$memory_dir" -maxdepth 1 -type f -name '*.md' -mmin "-$minutes" -print0 |
    sort -z |
    while IFS= read -r -d '' file; do
      is_sensitive_path "$file" && continue
      printf '\n### Memory markdown: %s\n\n' "$(display_path "$file")" >> "$bundle"
      {
        printf '```text\n'
        sed -n '1,120p' "$file"
        printf '\n```\n'
      } | append_redacted
    done
}

collect_memory_db() {
  local root=$1
  local db="$root/memory.db"
  [ -f "$db" ] || return 0

  printf '\n### memory.db status and recent turns\n\n' >> "$bundle"
  if ! command -v sqlite3 >/dev/null 2>&1; then
    printf 'sqlite3 not found; skipped memory.db extraction.\n' >> "$bundle"
    return 0
  fi
  if ! sqlite3 -readonly "$db" 'SELECT 1;' >/dev/null 2>&1; then
    printf 'memory.db exists but could not be opened read-only; likely locked or corrupt: %s\n' "$db" | append_redacted
    return 0
  fi

  {
    printf '```tsv\n'
    sqlite_query "$db" "SELECT 'turns_recent', count(*) FROM turns WHERE ts_unix >= $cutoff_epoch;" 2>/dev/null || true
    sqlite_query "$db" "SELECT 'turns_total', count(*) FROM turns;" 2>/dev/null || true
    sqlite_query "$db" "SELECT 'active_memory_items', count(*) FROM goncho_memory_items WHERE active = 1;" 2>/dev/null || true
    sqlite_query "$db" "SELECT 'memory_eval_artifacts', count(*) FROM goncho_memory_eval_artifacts;" 2>/dev/null || true
    printf '```\n\n'
    printf 'Recent turns:\n\n```tsv\n'
    sqlite_query "$db" "
      SELECT
        datetime(ts_unix, 'unixepoch') || 'Z',
        role,
        session_id,
        coalesce(chat_id, ''),
        coalesce(memory_sync_status, ''),
        coalesce(extracted, ''),
        substr(replace(replace(content, char(13), ' '), char(10), ' '), 1, 1800)
      FROM turns
      WHERE ts_unix >= $cutoff_epoch
      ORDER BY ts_unix ASC, id ASC
      LIMIT $max_turns;
    " 2>/dev/null || true
    printf '\n```\n'
  } | append_redacted
}

collect_session_files() {
  local root=$1
  local sessions_dir="$root/sessions"
  [ -d "$sessions_dir" ] || return 0

  printf '\n### Session files\n\n' >> "$bundle"
  find "$sessions_dir" -maxdepth 1 -type f -mmin "-$minutes" \
    \( -name '*.jsonl' -o -name '*.json' -o -name '*.yaml' -o -name '*.yml' \) \
    ! -name 'request_dump_*' -print0 |
    sort -z |
    while IFS= read -r -d '' file; do
      is_sensitive_path "$file" && continue
      printf '\n#### %s\n\n' "$(display_path "$file")" >> "$bundle"
      {
        printf '```tsv\n'
        case "$file" in
          *.jsonl|*.json)
            jsonl_events "$file" "$max_session_lines"
            ;;
          *)
            sed -n '1,220p' "$file"
            ;;
        esac
        printf '\n```\n'
      } | append_redacted
    done

  local skipped
  skipped=$(find "$sessions_dir" -maxdepth 1 -type f -mmin "-$minutes" -name 'request_dump_*' | wc -l | tr -d ' ')
  if [ "$skipped" -gt 0 ]; then
    printf '\nSkipped %s request_dump_* file(s); they can contain raw provider request payloads.\n' "$skipped" >> "$bundle"
  fi
}

collect_tool_audit() {
  local root=$1
  local file="$root/tools/audit.jsonl"
  [ -f "$file" ] || return 0

  printf '\n### Tool audit events\n\n' >> "$bundle"
  {
    printf '```tsv\n'
    tool_audit_events "$file"
    printf '\n```\n'
  } | append_redacted
}

collect_recent_file_manifest() {
  local root=$1
  printf '\n### Recent non-secret file manifest\n\n' >> "$bundle"
  {
    printf '```text\n'
    find "$root" -maxdepth 4 -type f -mmin "-$minutes" \
      ! -path '*/cache/audio/*' \
      ! -path '*/.git/*' \
      ! -path "$root/profiles/*" \
      ! -name 'auth.json' \
      ! -name 'request_dump_*' \
      ! -name '.env' \
      ! -name '*.pem' \
      ! -name '*.key' \
      ! -iname '*secret*' \
      ! -iname '*token*' \
      ! -iname '*password*' \
      -printf '%TY-%Tm-%Td %TH:%TM %s %p\n' |
      sort |
      tail -n 200
    printf '\n```\n'
  } | append_redacted
}

collect_journal() {
  local label=$1
  [ "$include_journal" -eq 1 ] || return 0
  command -v journalctl >/dev/null 2>&1 || return 0

  local product profile unit
  product=${label%%/*}
  profile=${label#*/}
  case "$product/$profile" in
    gormes/default) unit="gormes-gateway.service" ;;
    gormes/*) unit="gormes-gateway-$profile.service" ;;
    hermes/default) unit="hermes-gateway.service" ;;
    hermes/*) unit="hermes-gateway-$profile.service" ;;
    *) return 0 ;;
  esac

  printf '\n### User journal: %s\n\n' "$unit" >> "$bundle"
  {
    printf '```text\n'
    journalctl --user -u "$unit" --since "$cutoff_iso" --no-pager -n "$max_journal_lines" 2>/dev/null || true
    printf '\n```\n'
  } | append_redacted
}

collect_profile() {
  local label=$1
  local root=$2
  [ -d "$root" ] || return 0

  append_section "Profile: $label"
  {
    printf '%s\n' "- Root: $root"
    printf '%s\n' "- Cutoff: $cutoff_iso ($hours hours)"
  } | append_redacted

  collect_recent_file_manifest "$root"
  collect_session_files "$root"
  collect_tool_audit "$root"
  collect_memory_db "$root"
  collect_memory_markdown "$root"
  collect_journal "$label"
}

profile_labels=()
profile_roots=()

if [ -d "$gormes_home" ]; then
  profile_labels+=("gormes/default")
  profile_roots+=("$gormes_home")
fi
if [ -d "$gormes_home/profiles" ]; then
  while IFS= read -r -d '' dir; do
    profile_labels+=("gormes/$(basename "$dir")")
    profile_roots+=("$dir")
  done < <(find "$gormes_home/profiles" -mindepth 1 -maxdepth 1 -type d -print0 | sort -z)
fi

if [ "$include_hermes" -eq 1 ]; then
  if [ -d "$hermes_home" ]; then
    profile_labels+=("hermes/default")
    profile_roots+=("$hermes_home")
  fi
  if [ -d "$hermes_home/profiles" ]; then
    while IFS= read -r -d '' dir; do
      profile_labels+=("hermes/$(basename "$dir")")
      profile_roots+=("$dir")
    done < <(find "$hermes_home/profiles" -mindepth 1 -maxdepth 1 -type d -print0 | sort -z)
  fi
fi

{
  printf '# Gormes Profile Session Audit Bundle\n\n'
  printf '%s\n' "- Generated: $stamp"
  printf '%s\n' "- Repository: $repo_root"
  printf '%s\n' "- Lookback: $hours hours"
  printf '%s\n' "- Cutoff UTC: $cutoff_iso"
  printf '%s\n' "- Gormes home: $gormes_home"
  printf '%s\n' "- Hermes included: $include_hermes"
  printf '%s\n' "- Collection policy: read-only, redacted, auth/env/request-dump files excluded."
  printf '%s\n' "- Important: this bundle is evidence for audit, not a backlog. Fixes must map to progress.json/module docs."
} | append_redacted

collect_modules

if [ "${#profile_labels[@]}" -eq 0 ]; then
  append_section "Profiles"
  printf 'No profile roots found.\n' >> "$bundle"
else
  for i in "${!profile_labels[@]}"; do
    collect_profile "${profile_labels[$i]}" "${profile_roots[$i]}"
  done
fi

cat > "$prompt" <<PROMPT
You are auditing recent Gormes/Hermes-compatible agent behavior from a redacted evidence bundle.

Repository: $repo_root
Evidence bundle: $bundle
Module docs: $modules_dir
Lookback: last $hours hours since $cutoff_iso UTC

Task:
1. Read the evidence bundle and the relevant files under docs/content/building-gormes/modules.
2. Find agent-response issues, especially cases where the visible answer conflicts with persisted state, tool output, runtime state, or user preferences.
3. Classify each issue under one or more module docs, for example memory, learning-loop, profiles, gateway, channels, providers, sessions, tools, tts, kanban, runtime, config, or fleet.
4. Produce a fix/improvement plan that can be translated into progress.json rows. Do not create a side backlog.

Pay special attention to this known failure shape:
- The agent says persistent memory stores have zero entries while durable context files or recent turns contain remembered facts.
- The response treats loaded context as "memory" without explaining the difference between memory.db, markdown/context files, profile state, and session transcript state.
- Tool progress, TTS, terminal, or channel wrappers leak implementation details or make the user-facing answer noisy.
- Per-profile migration or service ownership causes state to land in the wrong profile.

Output format:
- Findings first, highest severity first.
- For each finding include: severity, profile/session/time evidence, what the user saw, why it is wrong, module doc path(s), likely root cause, and proposed fix.
- Then list progress-row-ready recommendations: row title, target module, write scope, acceptance, and test/smoke commands.
- Then list any evidence gaps the runtime should expose better next time.

Constraints:
- Use evidence from the bundle and repo docs only.
- Do not expose secrets, tokens, private raw auth, or full unredacted user identifiers.
- Do not edit files. This is an audit and planning pass.
PROMPT

if [ "$dry_run" -eq 1 ]; then
  printf 'dry-run: wrote bundle: %s\n' "$bundle"
  printf 'dry-run: wrote prompt: %s\n' "$prompt"
  exit 0
fi

if ! command -v "$codexu_bin" >/dev/null 2>&1; then
  die "codexu binary not found: $codexu_bin"
fi

"$codexu_bin" exec --ephemeral --sandbox read-only -C "$repo_root" -o "$report" - < "$prompt"

printf 'wrote bundle: %s\n' "$bundle"
printf 'wrote prompt: %s\n' "$prompt"
printf 'wrote codexu audit: %s\n' "$report"
