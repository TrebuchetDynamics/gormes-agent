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
with_hermes_refs=1
hermes_src=""
execute=1
include_journal=1
max_turns=120
max_tool_lines=160
max_session_lines=220
max_session_files=120
max_journal_lines=220
executor_sandbox="workspace-write"

usage() {
  cat <<'USAGE'
Usage: scripts/audit-profile-sessions.sh [options]

Collect redacted evidence from every Gormes profile/session touched inside the
lookback window, then ask codexu to audit response quality, memory, learning
loop, token usage, accuracy, formatting, web search, tool execution, runtime,
profile/channel routing, TTS/media, and security issues before mapping fixes
back to docs/content/building-gormes/modules.

Options:
  --hours N             Look back N hours. Default: 12.
  --gormes-home PATH    Gormes runtime home. Default: $GORMES_HOME or ~/.gormes.
  --hermes-home PATH    Hermes runtime home for --include-hermes. Default: ~/.hermes.
  --modules-dir PATH    Building-Gormes module docs directory.
  --out PATH            Output directory. Default: .codex/profile-session-audits/<utc>.
  --codexu PATH         codexu binary. Default: CODEXU_BIN or codexu.
  --include-hermes      Also scan ~/.hermes and ~/.hermes/profiles.
  --with-hermes         Let planner/executor compare against upstream Hermes source. Default on.
  --no-hermes           Disable upstream Hermes source comparison.
  --hermes-src PATH     Hermes source root. Implies --with-hermes.
  --execute             After planner output, run a second codexu executor pass. Default on.
  --plan-only           Stop after the planner report; do not run executor.
  --executor-sandbox S  Sandbox for --execute pass. Default: workspace-write.
  --no-journal          Skip user-systemd journal snippets.
  --dry-run             Build bundle and prompt, but do not invoke codexu.
  --max-turns N         Max recent memory turns per profile. Default: 120.
  --max-tool-lines N    Max tool audit rows per profile. Default: 160.
  --max-session-lines N Max lines per session file excerpt. Default: 220.
  --max-session-files N Max session files listed per profile. Default: 120.
  --max-journal-lines N Max user journal lines per profile. Default: 220.
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
    --with-hermes)
      with_hermes_refs=1
      shift
      ;;
    --no-hermes)
      with_hermes_refs=0
      hermes_src=""
      shift
      ;;
    --hermes-src)
      [ "$#" -ge 2 ] || die "--hermes-src requires a value"
      hermes_src="$2"
      with_hermes_refs=1
      shift 2
      ;;
    --execute)
      execute=1
      shift
      ;;
    --plan-only)
      execute=0
      shift
      ;;
    --executor-sandbox)
      [ "$#" -ge 2 ] || die "--executor-sandbox requires a value"
      executor_sandbox="$2"
      shift 2
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
    --max-session-lines)
      [ "$#" -ge 2 ] || die "--max-session-lines requires a value"
      max_session_lines="$2"
      shift 2
      ;;
    --max-session-files)
      [ "$#" -ge 2 ] || die "--max-session-files requires a value"
      max_session_files="$2"
      shift 2
      ;;
    --max-journal-lines)
      [ "$#" -ge 2 ] || die "--max-journal-lines requires a value"
      max_journal_lines="$2"
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

for numeric in "$max_turns" "$max_tool_lines" "$max_session_lines" "$max_session_files" "$max_journal_lines"; do
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
executor_prompt="$out_dir/codexu-execute-prompt.md"
executor_report="$out_dir/codexu-execution.md"
cutoff_epoch=$(date -u -d "$hours hours ago" +%s)
cutoff_iso=$(date -u -d "@$cutoff_epoch" +%Y-%m-%dT%H:%M:%SZ)
minutes=$((hours * 60))

resolve_hermes_src() {
  local candidate
  if [ -n "$hermes_src" ]; then
    candidate=$hermes_src
    if [ ! -f "$candidate/hermes_cli/main.py" ]; then
      die "Hermes source root does not look valid: $candidate"
    fi
    (cd "$candidate" && pwd)
    return 0
  fi

  for candidate in "$repo_root/hermes-agent" "$repo_root/../hermes-agent" "$repo_root/references/hermes-agent"; do
    if [ -f "$candidate/hermes_cli/main.py" ]; then
      (cd "$candidate" && pwd)
      return 0
    fi
  done
  return 1
}

resolved_hermes_src=""
if [ "$with_hermes_refs" -eq 1 ]; then
  if ! resolved_hermes_src=$(resolve_hermes_src); then
    die "--with-hermes requested, but no Hermes source was found"
  fi
fi

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
    s/\bbot[0-9]{8,10}:[A-Za-z0-9_-]{30,}\b/bot[REDACTED_TELEGRAM_TOKEN]/g;
    s/\b[0-9]{8,10}:[A-Za-z0-9_-]{30,}\b/[REDACTED_TELEGRAM_TOKEN]/g;
    s/\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b/[REDACTED_JWT]/g;
    s/\b(agent:[A-Za-z0-9_:-]*telegram:[A-Za-z0-9_:-]*:)[0-9]{6,}\b/${1}[REDACTED_ID]/g;
    s/\b(telegram:[A-Za-z0-9_:-]*:)[0-9]{6,}\b/${1}[REDACTED_ID]/g;
    s/\b(telegram:)[0-9]{6,}\b/${1}[REDACTED_ID]/g;
    s/\b([A-Za-z0-9_]*(?:chat_id|user_id)|peer_id|workspace_id|observer_peer_id)([":[:space:]=]+)[0-9]{6,}\b/${1}${2}[REDACTED_ID]/gi;
    s/\b[A-Fa-f0-9]{64,}\b/[REDACTED_HEX]/g;
    s/\b[A-Za-z0-9+_-]{96,}={0,2}\b/[REDACTED_LONG_TOKEN]/g;
  '
}

append_redacted() {
  redact_stream >> "$bundle"
}

append_section() {
  printf '\n## %s\n\n' "$1" >> "$bundle"
}

limit_lines() {
  local limit=$1
  sed -n "1,${limit}p"
}

list_command_paths() {
  local name=$1
  if command -v which >/dev/null 2>&1; then
    which -a "$name" 2>/dev/null | sort -u
  else
    command -v "$name" 2>/dev/null || true
  fi
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
      select(epoch >= $cutoff)
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
      rg -n '^#{1,3}[[:space:]]' "$modules_dir"/*.md | head -n 500 || true
    else
      grep -HnE '^#{1,3}[[:space:]]' "$modules_dir"/*.md | head -n 500 || true
    fi
  } | append_redacted
}

collect_audit_dimensions() {
  append_section "Audit Dimensions"
  cat <<'DIMENSIONS' >> "$bundle"
The planner must audit every dimension below for all profile/session evidence inside the lookback window:

- response_quality: helpfulness, directness, completeness, over/under-answering, language match, accessibility, and user preference fit.
- memory: memory tool behavior, memory.db writes, USER.md/MEMORY.md context, provenance, recall, persistence acknowledgements, and stale/contradictory facts.
- learning_loop: whether feedback becomes durable improvement, whether mistakes recur, and whether proposed fixes map to progress/module docs instead of side queues.
- token_usage: token/cost accounting, excessive prompt/response size, runaway context, compression quality, and high-latency or high-token turns.
- accuracy: factual correctness, workspace/path identity, source grounding, uncertainty handling, date/time handling, and contradictions against tool/runtime evidence.
- response_format: Markdown/channel rendering, screen-reader friendliness, concise text, TTS-friendly wording, attachment metadata, and leaked implementation details.
- web_search_quality: query choice, source quality, recency when needed, citation/evidence quality, unnecessary searches, and missing searches for unstable facts.
- tool_execution: tool choice, arguments, failures, duration, retries, status reporting, terminal safety, and whether tool results were interpreted correctly.
- providers_runtime: provider/auth errors, fallback behavior, model routing, latency, HTTP errors, token invalidation, and stale binary/home/profile surfaces.
- profiles_sessions_channels: profile ownership, session routing, channel delivery, gateway service ownership, context roots, and cross-profile contamination.
- tts_stt_media: STT transcript quality, TTS attachment freshness, audio/text consistency, media delivery, and local-path leakage.
- privacy_security: secret redaction, private identifiers, auth files, risky commands, and overexposed logs.

DIMENSIONS
}

collect_runtime_surface() {
  append_section "Runtime Surface"
  local git_status
  {
    printf 'Repository working directory: %s\n' "$repo_root"
    printf 'Git branch: '
    git -C "$repo_root" branch --show-current 2>/dev/null || printf 'unknown\n'
    printf 'Git tracked status:\n'
    git_status=$(git -C "$repo_root" status --porcelain=v1 -uno 2>/dev/null || true)
    if [ -n "$git_status" ]; then
      printf '%s\n' "$git_status"
    else
      printf '(clean)\n'
    fi
    printf '\nEnvironment homes:\n'
    printf 'GORMES_HOME=%s\n' "${GORMES_HOME:-}"
    printf 'HERMES_HOME=%s\n' "${HERMES_HOME:-}"
    printf 'CODEX_HOME=%s\n' "${CODEX_HOME:-}"
    printf 'XDG_CONFIG_HOME=%s\n' "${XDG_CONFIG_HOME:-}"
    printf 'XDG_DATA_HOME=%s\n' "${XDG_DATA_HOME:-}"
    printf '\nGormes command discovery:\n'
    if command -v gormes >/dev/null 2>&1; then
      list_command_paths gormes
      if command -v readlink >/dev/null 2>&1; then
        while IFS= read -r candidate; do
          [ -n "$candidate" ] || continue
          printf 'realpath\t%s\t' "$candidate"
          readlink -f "$candidate" 2>/dev/null || printf 'unknown\n'
        done < <(list_command_paths gormes)
      fi
    else
      printf 'gormes not found on PATH\n'
    fi
    printf '\nCodexu command discovery:\n'
    if command -v "$codexu_bin" >/dev/null 2>&1; then
      command -v "$codexu_bin" 2>/dev/null || true
      "$codexu_bin" --version 2>/dev/null || true
    else
      printf 'codexu not found on PATH: %s\n' "$codexu_bin"
    fi
    printf '\nGateway status from discovered gormes command:\n'
    if command -v gormes >/dev/null 2>&1 && command -v timeout >/dev/null 2>&1; then
      timeout 5s gormes gateway status --json 2>/dev/null || printf 'gateway status unavailable or timed out\n'
    elif command -v gormes >/dev/null 2>&1; then
      printf 'skipped; timeout command unavailable\n'
    else
      printf 'skipped; gormes not found on PATH\n'
    fi
  } | append_redacted
}

collect_hermes_refs() {
  [ "$with_hermes_refs" -eq 1 ] || return 0
  [ -n "$resolved_hermes_src" ] || return 0

  append_section "Optional Hermes Upstream Reference"
  {
    printf 'Hermes source root: %s\n\n' "$resolved_hermes_src"
    printf 'Suggested upstream anchors for this audit:\n'
    for rel in \
      "agent/prompt_builder.py" \
      "agent/memory_manager.py" \
      "tools/memory_tool.py" \
      "tools/skills_tool.py" \
      "hermes_cli/profiles.py" \
      "gateway/run.py"; do
      if [ -f "$resolved_hermes_src/$rel" ]; then
        printf '%s\n' "- $rel"
      fi
    done

    printf '\nRelevant grep hits:\n'
    if command -v rg >/dev/null 2>&1; then
      rg -n \
        'USER\.md|MEMORY\.md|memory|context|profile|text_to_speech|tts|telegram|MarkdownV2|session' \
        "$resolved_hermes_src/agent" \
        "$resolved_hermes_src/tools" \
        "$resolved_hermes_src/gateway" \
        "$resolved_hermes_src/hermes_cli" 2>/dev/null |
        head -n 500 || true
    else
      grep -RInE \
        'USER\.md|MEMORY\.md|memory|context|profile|text_to_speech|tts|telegram|MarkdownV2|session' \
        "$resolved_hermes_src/agent" \
        "$resolved_hermes_src/tools" \
        "$resolved_hermes_src/gateway" \
        "$resolved_hermes_src/hermes_cli" 2>/dev/null |
        head -n 500 || true
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
    sqlite_query "$db" "
      SELECT 'recent_role', role, count(*), round(avg(length(content))), max(length(content))
      FROM turns
      WHERE ts_unix >= $cutoff_epoch
      GROUP BY role;
    " 2>/dev/null || true
    sqlite_query "$db" "
      SELECT 'memory_sync_status', coalesce(memory_sync_status, ''), count(*)
      FROM turns
      WHERE ts_unix >= $cutoff_epoch
      GROUP BY coalesce(memory_sync_status, '');
    " 2>/dev/null || true
    sqlite_query "$db" "
      SELECT 'extraction_status', extracted, count(*)
      FROM turns
      WHERE ts_unix >= $cutoff_epoch
      GROUP BY extracted;
    " 2>/dev/null || true
    sqlite_query "$db" "
      SELECT 'response_signal', 'assistant_local_media_paths', count(*)
      FROM turns
      WHERE ts_unix >= $cutoff_epoch AND role = 'assistant' AND content LIKE '%MEDIA:%';
    " 2>/dev/null || true
    sqlite_query "$db" "
      SELECT 'response_signal', 'assistant_long_over_3000_chars', count(*)
      FROM turns
      WHERE ts_unix >= $cutoff_epoch AND role = 'assistant' AND length(content) > 3000;
    " 2>/dev/null || true
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

collect_session_inventory() {
  local root=$1
  local sessions_dir="$root/sessions"
  [ -d "$sessions_dir" ] || return 0

  printf '\n### Session inventory within lookback\n\n' >> "$bundle"
  {
    printf '```text\n'
    printf 'All session files listed here are limited by --hours (%s hours) and exclude request_dump_* payloads.\n' "$hours"
    local total
    total=$(find "$sessions_dir" -maxdepth 1 -type f -mmin "-$minutes" \
      \( -name '*.jsonl' -o -name '*.json' -o -name '*.yaml' -o -name '*.yml' \) \
      ! -name 'request_dump_*' | wc -l | tr -d ' ')
    printf 'recent_session_files: %s\n' "$total"
    find "$sessions_dir" -maxdepth 1 -type f -mmin "-$minutes" \
      \( -name '*.jsonl' -o -name '*.json' -o -name '*.yaml' -o -name '*.yml' \) \
      ! -name 'request_dump_*' \
      -printf '%TY-%Tm-%Td %TH:%TM %s %p\n' |
      sort |
      limit_lines "$max_session_files"
    if [ "$total" -gt "$max_session_files" ]; then
      printf 'truncated_session_files: %s of %s shown; increase --max-session-files to inspect more within this --hours window.\n' "$max_session_files" "$total"
    fi
    printf '\n```\n'
  } | append_redacted

  local sessions_json="$sessions_dir/sessions.json"
  if [ -f "$sessions_json" ] && [ "$(find "$sessions_json" -mmin "-$minutes" -print 2>/dev/null)" ]; then
    printf '\n#### Session token/cost summaries\n\n' >> "$bundle"
    {
      printf '```tsv\n'
      if command -v jq >/dev/null 2>&1; then
        jq -r '
          .. | objects
          | select(has("session_id") or has("input_tokens") or has("total_tokens"))
          | [
              (.session_id // .id // ""),
              (.platform // ""),
              (.chat_type // ""),
              (.created_at // ""),
              (.updated_at // ""),
              (.input_tokens // 0),
              (.output_tokens // 0),
              (.cache_read_tokens // 0),
              (.cache_write_tokens // 0),
              (.total_tokens // 0),
              (.last_prompt_tokens // 0),
              (.estimated_cost_usd // ""),
              (.cost_status // "")
            ]
          | @tsv
        ' "$sessions_json" 2>/dev/null | limit_lines "$max_session_files" || true
      else
        sed -n '1,120p' "$sessions_json"
      fi
      printf '\n```\n'
    } | append_redacted
  fi
}

collect_session_files() {
  local root=$1
  local sessions_dir="$root/sessions"
  [ -d "$sessions_dir" ] || return 0

  printf '\n### Session files\n\n' >> "$bundle"
  local seen=0
  while IFS= read -r -d '' file; do
    seen=$((seen + 1))
    if [ "$seen" -gt "$max_session_files" ]; then
      continue
    fi
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
  done < <(find "$sessions_dir" -maxdepth 1 -type f -mmin "-$minutes" \
    \( -name '*.jsonl' -o -name '*.json' -o -name '*.yaml' -o -name '*.yml' \) \
    ! -name 'request_dump_*' -print0 |
    sort -z)
  if [ "$seen" -gt "$max_session_files" ]; then
    printf '\nSession file excerpts truncated at %s files inside this --hours window.\n' "$max_session_files" >> "$bundle"
  fi

  local skipped
  skipped=$(find "$sessions_dir" -maxdepth 1 -type f -mmin "-$minutes" -name 'request_dump_*' | wc -l | tr -d ' ')
  if [ "$skipped" -gt 0 ]; then
    printf '\nSkipped %s request_dump_* file(s); they can contain raw provider request payloads.\n' "$skipped" >> "$bundle"
  fi
}

collect_tool_audit_summary() {
  local root=$1
  local file="$root/tools/audit.jsonl"
  [ -f "$file" ] || return 0

  printf '\n### Tool audit summary within lookback\n\n' >> "$bundle"
  {
    printf '```tsv\n'
    if command -v jq >/dev/null 2>&1; then
      jq -rs --argjson cutoff "$cutoff_epoch" '
        def epoch($r):
          ($r.timestamp // $r.ts // $r.created_at // $r.time // null) as $t
          | if ($t | type) == "number" then $t
            elif ($t | type) == "string" then ($t | fromdateiso8601? // 0)
            else 0 end;
        [ .[] | select(epoch(.) >= $cutoff) ] as $rows
        | [ .[] | select(epoch(.) == 0) ] as $undated
        | "recent_tool_events\t\($rows | length)",
          "undated_tool_events_excluded_from_lookback\t\($undated | length)",
          ($rows
            | sort_by(.tool // "unknown")
            | group_by(.tool // "unknown")[]?
            | [
                (.[0].tool // "unknown"),
                length,
                (map(select((.status // "") == "completed")) | length),
                (map(select((.status // "") != "completed")) | length),
                (((map(.duration_ms // 0) | add) // 0) / (length | if . == 0 then 1 else . end) | floor)
              ]
            | @tsv),
          "web_search_like_events_within_lookback",
          ($rows[]
            | select((.tool // "") | test("web_search|web_extract|browser|search"; "i"))
            | [
                (.timestamp // ""),
                (.tool // ""),
                (.status // ""),
                ((.args // {}) | tostring | gsub("\r"; " ") | gsub("\n"; " ") | .[0:500])
              ]
            | @tsv),
          "undated_tool_event_samples",
          ($undated[0:10][]
            | [
                (.timestamp // .ts // .created_at // .time // ""),
                (.tool // ""),
                (.status // ""),
                ((.args // .error // {}) | tostring | gsub("\r"; " ") | gsub("\n"; " ") | .[0:500])
              ]
            | @tsv)
      ' "$file" 2>/dev/null || true
    else
      printf 'jq not found; showing tail without timestamp filtering.\n'
      tail -n "$max_tool_lines" "$file"
    fi
    printf '\n```\n'
  } | append_redacted
}

collect_tool_audit() {
  local root=$1
  local file="$root/tools/audit.jsonl"
  [ -f "$file" ] || return 0

  printf '\n### Tool audit events within lookback\n\n' >> "$bundle"
  {
    printf '```tsv\n'
    if ! command -v jq >/dev/null 2>&1; then
      printf 'jq not found; showing tail without timestamp filtering.\n'
    fi
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
  collect_session_inventory "$root"
  collect_session_files "$root"
  collect_tool_audit_summary "$root"
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
  printf '%s\n' "- Hermes upstream comparison: $with_hermes_refs"
  if [ -n "$resolved_hermes_src" ]; then
    printf '%s\n' "- Hermes source root: $resolved_hermes_src"
  fi
  printf '%s\n' "- Collection policy: all profiles and all matching session/tool/memory evidence inside --hours; read-only, redacted, auth/env/request-dump files excluded."
  printf '%s\n' "- Important: this bundle is evidence for audit, not a backlog. Fixes must map to progress.json/module docs."
} | append_redacted

collect_audit_dimensions
collect_runtime_surface
collect_modules
collect_hermes_refs

if [ "${#profile_labels[@]}" -eq 0 ]; then
  append_section "Profiles"
  printf 'No profile roots found.\n' >> "$bundle"
else
  for i in "${!profile_labels[@]}"; do
    collect_profile "${profile_labels[$i]}" "${profile_roots[$i]}"
  done
fi

if [ -n "$resolved_hermes_src" ]; then
  hermes_prompt_note="Hermes upstream source: $resolved_hermes_src. Compare against Hermes before proposing fixes; cite exact files/symbols when Hermes defines the expected behavior."
else
  hermes_prompt_note="Hermes upstream source: disabled. Use only the bundle and Gormes repo docs/code."
fi

cat > "$prompt" <<PROMPT
You are the PLANNER stage for a lookback-limited, all-aspects Gormes/Hermes-compatible agent-behavior audit.

Repository: $repo_root
Evidence bundle: $bundle
Module docs: $modules_dir
Lookback: last $hours hours since $cutoff_iso UTC
$hermes_prompt_note

Task:
1. Read the evidence bundle and the relevant files under docs/content/building-gormes/modules. The evidence is limited by --hours; audit every profile/session represented inside that window.
2. If Hermes upstream is enabled, inspect the Hermes source for the same behavior before proposing a fix. Use Hermes as a behavior oracle, not as runtime dependency.
3. Audit every aspect of the agent behavior inside the lookback window: response quality, instruction following, memory, learning loop, token/cost usage, accuracy, response format, web-search quality, tool execution, provider/runtime health, profile/session/channel routing, TTS/STT/media, privacy/security, latency, and reliability.
4. Find issues where the visible answer, tool use, runtime state, token usage, search behavior, or persisted state conflicts with user intent, tool output, module contracts, or Hermes behavior.
5. Classify each issue under one or more module docs, for example memory, learning-loop, profiles, gateway, channels, providers, sessions, tools, tts, kanban, runtime, config, fleet, or docs.
6. Produce a solution plan that can be translated into progress.json rows or one bounded executor task. Do not create a side backlog.

Pay special attention to this known failure shape:
- The agent says persistent memory stores have zero entries while durable context files or recent turns contain remembered facts.
- The response treats loaded context as "memory" without explaining the difference between memory.db, markdown/context files, profile state, and session transcript state.
- Tool progress, TTS, terminal, or channel wrappers leak implementation details or make the user-facing answer noisy.
- Per-profile migration or service ownership causes state to land in the wrong profile.

Output format:
- Findings first, highest severity first.
- For each finding include: severity, audit dimension(s), profile/session/time evidence, what the user saw or what the runtime did, why it is wrong, module doc path(s), likely root cause, and proposed fix.
- Include a short scorecard for every audit dimension, including "no issue found" when the bundle has enough evidence and no problem is visible.
- Then list progress-row-ready recommendations: row title, target module, write scope, acceptance, and test/smoke commands.
- Then choose exactly one next executor task. It must be the smallest safe step after this plan and must name allowed write scope and verification commands.
- Then list any evidence gaps the runtime should expose better next time.

Constraints:
- Use evidence from the bundle, Gormes repo docs/code, and Hermes source only when enabled above.
- Do not expose secrets, tokens, private raw auth, or full unredacted user identifiers.
- Do not edit files. This is an audit and planning pass.
PROMPT

cat > "$executor_prompt" <<PROMPT
You are the EXECUTOR stage for the Gormes profile-session audit.

Repository: $repo_root
Evidence bundle: $bundle
Planner report: $report
Module docs: $modules_dir
$hermes_prompt_note

Task:
1. Read the planner report first, then the evidence bundle and relevant module docs/code.
2. If Hermes upstream is enabled, inspect Hermes before editing whenever the selected fix has a Hermes parity surface.
3. Execute exactly one bounded task from the planner report. Prefer the planner's "next executor task". If it is too broad or unsafe, update the canonical progress/module plan instead of attempting a large runtime fix.
4. Preserve the repository rule: use the existing development branch only and do not create a side backlog.
5. Run the focused verification commands named by the planner or the nearest focused tests for the touched scope. Always run git diff --check.

Constraints:
- Do not expose secrets, tokens, private raw auth, or full unredacted user identifiers.
- Do not edit unrelated files.
- Do not revert user or previous agent changes.
- Final output must list changed files, verification commands, and any remaining blocker.
PROMPT

if [ "$dry_run" -eq 1 ]; then
  printf 'dry-run: wrote bundle: %s\n' "$bundle"
  printf 'dry-run: wrote prompt: %s\n' "$prompt"
  if [ "$execute" -eq 1 ]; then
    printf 'dry-run: wrote executor prompt: %s\n' "$executor_prompt"
  fi
  exit 0
fi

if ! command -v "$codexu_bin" >/dev/null 2>&1; then
  die "codexu binary not found: $codexu_bin"
fi

run_codexu_stage() {
  local stage=$1
  local stage_prompt=$2
  local stage_report=$3
  local stage_sandbox=$4
  local pre_tracked_status post_tracked_status

  pre_tracked_status=$(git -C "$repo_root" status --porcelain=v1 -uno)
  "$codexu_bin" exec --ephemeral --sandbox "$stage_sandbox" -C "$repo_root" -o "$stage_report" - < "$stage_prompt"
  post_tracked_status=$(git -C "$repo_root" status --porcelain=v1 -uno)
  if [ "$pre_tracked_status" != "$post_tracked_status" ]; then
    printf 'warning: tracked git status changed during %s stage; inspect git diff before continuing.\n' "$stage" >&2
  fi
}

run_codexu_stage "planner" "$prompt" "$report" "read-only"

if [ "$execute" -eq 1 ]; then
  run_codexu_stage "executor" "$executor_prompt" "$executor_report" "$executor_sandbox"
  if git -C "$repo_root" diff --check > "$out_dir/git-diff-check.txt" 2>&1; then
    printf 'executor verification: git diff --check passed\n'
  else
    printf 'warning: executor verification failed; see %s\n' "$out_dir/git-diff-check.txt" >&2
  fi
fi

printf 'wrote bundle: %s\n' "$bundle"
printf 'wrote prompt: %s\n' "$prompt"
printf 'wrote codexu audit: %s\n' "$report"
if [ "$execute" -eq 1 ]; then
  printf 'wrote executor prompt: %s\n' "$executor_prompt"
  printf 'wrote codexu execution report: %s\n' "$executor_report"
fi
