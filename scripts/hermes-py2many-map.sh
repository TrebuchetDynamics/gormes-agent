#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HERMES_SRC="${HERMES_SRC:-$ROOT/hermes-agent}"
REPORT="${REPORT:-$ROOT/docs/content/building-gormes/architecture_plan/py2many-hermes-map.md}"
ARTIFACT_ROOT="${ARTIFACT_ROOT:-}"
TIMEOUT_SECONDS="${PY2MANY_TIMEOUT_SECONDS:-8}"
LIMIT=""
DRY_RUN=0

usage() {
  cat <<'USAGE'
usage: scripts/hermes-py2many-map.sh [--dry-run] [--limit N]

Runs py2many's Go backend across the Hermes Python checkout as a parity
mapping aid. Generated Go is written only to a temporary artifact directory and
is never imported into Gormes runtime packages.

Environment:
  HERMES_SRC                 Hermes checkout path (default: ./hermes-agent)
  REPORT                     Markdown report path
  ARTIFACT_ROOT              Output artifact root (default: mktemp under /tmp)
  PY2MANY_BIN                Existing py2many binary to use
  PY2MANY_TIMEOUT_SECONDS    Per-file timeout (default: 8)
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --limit)
      LIMIT="${2:-}"
      if [[ -z "$LIMIT" || ! "$LIMIT" =~ ^[0-9]+$ ]]; then
        echo "error: --limit requires a non-negative integer" >&2
        exit 2
      fi
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ ! -d "$HERMES_SRC" || ! -f "$HERMES_SRC/hermes_cli/main.py" ]]; then
  echo "error: HERMES_SRC does not look like hermes-agent: $HERMES_SRC" >&2
  exit 2
fi

mapfile -t FILES < <(
  cd "$HERMES_SRC"
  find . \
    -path '*/.git' -prune -o \
    -path '*/__pycache__' -prune -o \
    -path '*/node_modules' -prune -o \
    -path '*/.venv' -prune -o \
    -type f -name '*.py' -print |
    sed 's#^\./##' |
    sort
)

if [[ -n "$LIMIT" && "$LIMIT" -lt "${#FILES[@]}" ]]; then
  FILES=("${FILES[@]:0:$LIMIT}")
fi

HERMES_SHA="$(git -C "$HERMES_SRC" rev-parse --short HEAD 2>/dev/null || true)"
if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "Hermes source: $HERMES_SRC"
  echo "Hermes SHA: ${HERMES_SHA:-unknown}"
  echo "Python files selected: ${#FILES[@]}"
  echo "Report: $REPORT"
  echo "Generated Go artifact root: ${ARTIFACT_ROOT:-/tmp/gormes-hermes-py2many-map.*}"
  echo "py2many command: py2many --go --outdir <temp-dir> --comment-unsupported --no-strict --ignore-formatter-errors <file.py>"
  exit 0
fi

if [[ -z "$ARTIFACT_ROOT" ]]; then
  ARTIFACT_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/gormes-hermes-py2many-map.XXXXXX")"
else
  mkdir -p "$ARTIFACT_ROOT"
fi

PY2MANY_BIN="${PY2MANY_BIN:-}"
INSTALL_NOTE=""
if [[ -n "$PY2MANY_BIN" && -x "$PY2MANY_BIN" ]]; then
  :
elif command -v py2many >/dev/null 2>&1; then
  PY2MANY_BIN="$(command -v py2many)"
elif [[ -x /tmp/gormes-py2many-map-venv/bin/py2many ]]; then
  PY2MANY_BIN="/tmp/gormes-py2many-map-venv/bin/py2many"
  INSTALL_NOTE="reused /tmp/gormes-py2many-map-venv"
else
  python3 -m venv /tmp/gormes-py2many-map-venv
  /tmp/gormes-py2many-map-venv/bin/python -m pip install -q --upgrade pip
  /tmp/gormes-py2many-map-venv/bin/python -m pip install -q py2many
  PY2MANY_BIN="/tmp/gormes-py2many-map-venv/bin/py2many"
  INSTALL_NOTE="python3 -m venv /tmp/gormes-py2many-map-venv && /tmp/gormes-py2many-map-venv/bin/python -m pip install -q --upgrade pip py2many"
fi

PY2MANY_VERSION="$("$PY2MANY_BIN" --version 2>&1 | tail -n 1 || true)"
if [[ -z "$PY2MANY_VERSION" ]]; then
  PY2MANY_VERSION="unknown"
fi

RESULTS_TSV="$ARTIFACT_ROOT/results.tsv"
printf 'path\tclassification\tstatus\texit_code\tgenerated\tsymbols\tfirst_error\ttarget\n' > "$RESULTS_TSV"

md_escape() {
  sed -e 's/\\/\\\\/g' -e 's/|/\\|/g' -e 's/\[/\&#91;/g' -e 's/\]/\&#93;/g'
}

redact_sensitive() {
  sed -E \
    -e 's/sk-proj-[A-Za-z0-9_-]+/sk-proj-[REDACTED]/g' \
    -e 's/sk-ant-[A-Za-z0-9_-]+/sk-ant-[REDACTED]/g' \
    -e 's/sk-[A-Za-z0-9_-]{12,}/sk-[REDACTED]/g' \
    -e 's/ghp_[A-Za-z0-9_]+/ghp_[REDACTED]/g' \
    -e 's/AKIA[0-9A-Z]{16}/AKIA[REDACTED]/g' \
    -e 's/xox[baprs]-[A-Za-z0-9-]+/xox[REDACTED]/g' \
    -e 's/AIza[0-9A-Za-z_-]+/AIza[REDACTED]/g'
}

one_line() {
  tr '\n' ' ' | tr '\t' ' ' | sed -E 's/[[:space:]]+/ /g' | redact_sensitive | cut -c1-220
}

classify_file() {
  local rel="$1"
  local src="$2"
  local status="$3"
  if [[ "$rel" == tests/* ]]; then
    echo "test-fixture"
  elif grep -Eq '(^|[[:space:]])(async def|await |asyncio|FastAPI|websocket|aiohttp|subprocess|click|typer|rich|prompt_toolkit|textual|playwright|sqlalchemy|boto3|httpx|requests)' "$src"; then
    echo "manual-go-native-rewrite"
  elif grep -Eq '^([A-Z][A-Z0-9_]+|DEFAULT_[A-Z0-9_]+)[[:space:]]*=|@dataclass|TypedDict|BaseModel|Enum\)' "$src"; then
    echo "schema-constant-candidate"
  elif [[ "$status" == "ok" ]]; then
    echo "transpiles"
  elif [[ "$status" == "timeout" ]]; then
    echo "timeout-review"
  else
    echo "unsupported-or-partial"
  fi
}

target_for_file() {
  local rel="$1"
  case "$rel" in
    hermes_cli/*) echo "cmd/gormes or internal/config";;
    agent/*|run_agent.py|cli.py) echo "internal/hermes or internal/kernel";;
    tools/*|toolsets.py|toolset_distributions.py) echo "internal/tools";;
    gateway/*|tui_gateway/*) echo "internal/gateway or cmd/gormes";;
    providers/*) echo "internal/provider";;
    plugins/*) echo "internal/plugins";;
    cron/*) echo "internal/cron";;
    tests/*) echo "test parity fixture";;
    skills/*|optional-skills/*) echo "internal/skills or bundled skill docs";;
    *) echo "parity review";;
  esac
}

total="${#FILES[@]}"
index=0
for rel in "${FILES[@]}"; do
  index=$((index + 1))
  src="$HERMES_SRC/$rel"
  rel_dir="$(dirname "$rel")"
  if [[ "$rel_dir" == "." ]]; then
    rel_dir=""
  fi
  out_dir="$ARTIFACT_ROOT/generated/$rel_dir"
  log_dir="$ARTIFACT_ROOT/logs/$rel_dir"
  mkdir -p "$out_dir" "$log_dir"
  stdout="$log_dir/$(basename "$rel" .py).stdout"
  stderr="$log_dir/$(basename "$rel" .py).stderr"

  status="ok"
  set +e
  timeout "$TIMEOUT_SECONDS" "$PY2MANY_BIN" --go --outdir "$out_dir" --comment-unsupported --no-strict --ignore-formatter-errors "$src" >"$stdout" 2>"$stderr"
  code=$?
  set -e
  if [[ "$code" -ne 0 ]]; then
    if [[ "$code" -eq 124 ]]; then
      status="timeout"
    else
      status="failed"
    fi
  fi

  generated="$(find "$out_dir" -maxdepth 1 -type f -name "$(basename "$rel" .py).go" -print -quit 2>/dev/null || true)"
  generated_name=""
  symbols=""
  if [[ -n "$generated" && -s "$generated" ]]; then
    generated_name="${generated#"$ARTIFACT_ROOT/"}"
    symbols="$(grep -aE '^(func|type|const|var)[[:space:]]+' "$generated" | head -8 | one_line || true)"
  elif [[ "$status" == "ok" ]]; then
    status="no-output"
  fi
  first_error="$(cat "$stderr" "$stdout" 2>/dev/null | one_line)"
  classification="$(classify_file "$rel" "$src" "$status")"
  target="$(target_for_file "$rel")"
  generated_name="${generated_name:-none}"
  symbols="${symbols:-none}"
  first_error="${first_error:-none}"
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$rel" "$classification" "$status" "$code" "$generated_name" "$symbols" "$first_error" "$target" >> "$RESULTS_TSV"

  if (( index % 100 == 0 )); then
    printf 'mapped %d/%d\n' "$index" "$total" >&2
  fi
done

status_counts="$(tail -n +2 "$RESULTS_TSV" | cut -f3 | sort | uniq -c | awk '{print "- `" $2 "`: " $1}')"
class_counts="$(tail -n +2 "$RESULTS_TSV" | cut -f2 | sort | uniq -c | awk '{print "- `" $2 "`: " $1}')"
top_dirs="$(printf '%s\n' "${FILES[@]}" | cut -d/ -f1 | sort | uniq -c | sort -nr | awk '{print "- `" $2 "`: " $1}')"

important_refs=(
  "hermes_cli/default_soul.py"
  "hermes_cli/config.py"
  "hermes_cli/profiles.py"
  "agent/prompt_builder.py"
  "hermes_cli/auth_commands.py"
  "hermes_cli/commands.py"
  "hermes_cli/main.py"
  "hermes_cli/claw.py"
  "agent/skill_commands.py"
  "agent/skill_preprocessing.py"
  "tools/skills_tool.py"
  "tools/skill_manager_tool.py"
  "tools/skills_sync.py"
  "gateway/run.py"
  "run_agent.py"
  "cli.py"
)

mkdir -p "$(dirname "$REPORT")"
{
  echo "# Hermes py2many Parity Map"
  echo
  echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo
  echo "## Scope"
  echo
  echo "- Hermes source: \`$HERMES_SRC\`"
  echo "- Hermes SHA: \`${HERMES_SHA:-unknown}\`"
  echo "- Python files mapped: \`${#FILES[@]}\`"
  echo "- py2many binary: \`$PY2MANY_BIN\`"
  echo "- py2many version: \`$PY2MANY_VERSION\`"
  echo "- Install evidence: \`${INSTALL_NOTE:-existing py2many binary}\`"
  echo "- Command shape: \`py2many --go --outdir <temp-dir> --comment-unsupported --no-strict --ignore-formatter-errors <file.py>\`"
  echo "- Per-file timeout: \`${TIMEOUT_SECONDS}s\`"
  echo "- Generated artifact root: \`$ARTIFACT_ROOT\`"
  echo
  echo "Generated Go remains an evidence artifact under \`$ARTIFACT_ROOT\`; it is not runtime code and should not be copied into \`cmd/\` or \`internal/\` without a source-backed parity row and Go tests."
  echo
  echo "## Top-Level Source Distribution"
  echo
  echo "$top_dirs"
  echo
  echo "## py2many Status Counts"
  echo
  echo "$status_counts"
  echo
  echo "## Classification Counts"
  echo
  echo "$class_counts"
  echo
  echo "## High-Priority Parity Contracts"
  echo
  echo "| Hermes file | classification | py2many status | generated | symbols | Gormes target |"
  echo "|---|---|---|---|---|---|"
  for ref in "${important_refs[@]}"; do
    awk -F '\t' -v ref="$ref" '$1 == ref {print}' "$RESULTS_TSV" |
      while IFS=$'\t' read -r path classification status code generated symbols first_error target; do
        printf '| `%s` | `%s` | `%s` | `%s` | %s | `%s` |\n' \
          "$(printf '%s' "$path" | md_escape)" \
          "$(printf '%s' "$classification" | md_escape)" \
          "$(printf '%s' "$status" | md_escape)" \
          "$(printf '%s' "${generated:-none}" | md_escape)" \
          "$(printf '%s' "${symbols:-none}" | md_escape)" \
          "$(printf '%s' "$target" | md_escape)"
      done
  done
  echo
  echo "## Immediate Findings"
  echo
  echo "- \`hermes_cli/default_soul.py\` is now covered by \`internal/hermes/default_soul.go\`; this map exists to prevent similar constants/templates from staying hidden inside unrelated files."
  echo "- Files classified as \`schema-constant-candidate\` are the best next source for paired Go constants, config defaults, manifests, enum tables, and prompt text drift tests."
  echo "- Files classified as \`manual-go-native-rewrite\` should not be treated as transpilation candidates even when py2many emits Go; they need behavior tests and native Go subsystem design."
  echo "- \`tests/\` files are included because they reveal upstream behavior contracts and fixture names, not because their generated Go should be kept."
  echo
  echo "## Full File Map"
  echo
  echo "| Hermes file | classification | py2many status | exit | generated | symbols | first output/error | Gormes target |"
  echo "|---|---|---:|---:|---|---|---|---|"
  tail -n +2 "$RESULTS_TSV" |
    while IFS=$'\t' read -r path classification status code generated symbols first_error target; do
      printf '| `%s` | `%s` | `%s` | `%s` | `%s` | %s | %s | `%s` |\n' \
        "$(printf '%s' "$path" | md_escape)" \
        "$(printf '%s' "$classification" | md_escape)" \
        "$(printf '%s' "$status" | md_escape)" \
        "$(printf '%s' "$code" | md_escape)" \
        "$(printf '%s' "${generated:-none}" | md_escape)" \
        "$(printf '%s' "${symbols:-none}" | md_escape)" \
        "$(printf '%s' "${first_error:-none}" | md_escape)" \
        "$(printf '%s' "$target" | md_escape)"
    done
} > "$REPORT"

echo "report: $REPORT"
echo "artifacts: $ARTIFACT_ROOT"
