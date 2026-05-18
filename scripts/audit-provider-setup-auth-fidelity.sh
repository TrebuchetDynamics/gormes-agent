#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

output="${GORMES_PROVIDER_SETUP_AUTH_FIDELITY_OUT:-${TMPDIR:-/tmp}/gormes-provider-setup-auth-fidelity.json}"
with_hermes=1
with_real_homes=0

usage() {
  cat <<'USAGE'
Usage: scripts/audit-provider-setup-auth-fidelity.sh [options]

Run a hermetic provider/setup/auth fidelity pack for the highest-risk Hermes
parity surfaces:
  - OpenAI Codex must use OAuth/import/device-code guidance, not generic API-key setup.
  - OpenRouter live model catalog parsing must not silently truncate the underlying data path.
  - auth status, config check, doctor, setup help, and model help must stay parseable/redacted.

Options:
  --output PATH       JSON report path. Default: $TMPDIR/gormes-provider-setup-auth-fidelity.json.
  --with-hermes       Record upstream Hermes source SHA and source refs. Default on.
  --no-hermes         Skip Hermes source SHA/source-ref lookup.
  --with-real-homes   Record that real homes were explicitly allowed. This pack still uses temp homes.
  -h, --help          Show this help.
USAGE
}

die() {
  printf 'audit-provider-setup-auth-fidelity: %s\n' "$*" >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      [ "$#" -ge 2 ] || die "--output requires a value"
      output="$2"
      shift 2
      ;;
    --with-hermes)
      with_hermes=1
      shift
      ;;
    --no-hermes)
      with_hermes=0
      shift
      ;;
    --with-real-homes)
      with_real_homes=1
      shift
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

[ -d "$repo_root/.git" ] || die "repo root not found: $repo_root"
cd "$repo_root"

resolve_hermes_src() {
  local candidate
  for candidate in "$repo_root/hermes-agent" "$repo_root/../hermes-agent" "$repo_root/references/hermes-agent"; do
    if [ -f "$candidate/hermes_cli/main.py" ]; then
      (cd "$candidate" && pwd)
      return 0
    fi
  done
  return 1
}

json_escape() {
  local value=${1//\\/\\\\}
  value=${value//\"/\\\"}
  value=${value//$'\n'/\\n}
  printf '%s' "$value"
}

assert_contains() {
  local file=$1
  local needle=$2
  if ! grep -Fq -- "$needle" "$file"; then
    printf '%s\n' "--- $file ---" >&2
    sed -n '1,220p' "$file" >&2
    die "expected output to contain: $needle"
  fi
}

assert_not_contains() {
  local file=$1
  local needle=$2
  if grep -Fq -- "$needle" "$file"; then
    printf '%s\n' "--- $file ---" >&2
    sed -n '1,220p' "$file" >&2
    die "output contained forbidden text: $needle"
  fi
}

run_step() {
  local label=$1
  shift
  printf 'provider-fidelity: %s\n' "$label"
  "$@"
}

hermes_src=""
hermes_sha="not_checked"
if [ "$with_hermes" -eq 1 ]; then
  if ! hermes_src=$(resolve_hermes_src); then
    die "--with-hermes requested, but no Hermes source checkout was found"
  fi
  hermes_sha=$(git -C "$hermes_src" rev-parse --short=12 HEAD)
fi

gormes_sha=$(git rev-parse --short=12 HEAD 2>/dev/null || printf 'unknown')
gormes_branch=$(git branch --show-current 2>/dev/null || printf 'unknown')
generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)

go_cache=$(go env GOCACHE)
go_mod_cache=$(go env GOMODCACHE)
[ -n "$go_cache" ] || die "go env GOCACHE returned empty"
[ -n "$go_mod_cache" ] || die "go env GOMODCACHE returned empty"

run_step "focused provider/setup/auth Go regressions" \
  env GOCACHE="$go_cache" GOMODCACHE="$go_mod_cache" \
  go test ./cmd/gormes -run 'Auth|Provider|OpenRouter|Codex|Doctor|Model|Fallback|Setup' -count=1

run_step "internal config/provider/doctor Go regressions" \
  env GOCACHE="$go_cache" GOMODCACHE="$go_mod_cache" \
  go test ./internal/config ./internal/provider ./internal/doctor -count=1

sandbox=$(mktemp -d "${TMPDIR:-/tmp}/gormes-provider-fidelity.XXXXXX")
cleanup() {
  chmod -R u+w "$sandbox" 2>/dev/null || true
  rm -rf "$sandbox"
}
trap cleanup EXIT

mkdir -p "$sandbox/home" "$sandbox/config" "$sandbox/data" "$sandbox/state" "$sandbox/gormes" "$sandbox/hermes" "$sandbox/codex"
gormes_env=(
  env
  HOME="$sandbox/home"
  XDG_CONFIG_HOME="$sandbox/config"
  XDG_DATA_HOME="$sandbox/data"
  XDG_STATE_HOME="$sandbox/state"
  GORMES_HOME="$sandbox/gormes"
  HERMES_HOME="$sandbox/hermes"
  CODEX_HOME="$sandbox/codex"
  GOCACHE="$go_cache"
  GOMODCACHE="$go_mod_cache"
)

run_gormes() {
  "${gormes_env[@]}" go run ./cmd/gormes "$@"
}

run_step "configure temp OpenAI Codex provider" run_gormes config set hermes.provider openai-codex
run_step "configure temp Codex endpoint" run_gormes config set hermes.endpoint https://chatgpt.com/backend-api/codex
run_step "configure temp Codex model" run_gormes config set hermes.model gpt-5.2

auth_json="$sandbox/auth-status.json"
doctor_json="$sandbox/doctor.json"
config_json="$sandbox/config-check.json"
setup_help="$sandbox/setup-help.txt"
model_help="$sandbox/model-help.txt"
onboard_help="$sandbox/onboard-help.txt"

printf 'provider-fidelity: auth status openai-codex --json smoke\n'
run_gormes auth status openai-codex --json > "$auth_json"
assert_contains "$auth_json" '"provider": "openai-codex"'
assert_contains "$auth_json" '"auth_type": "oauth_external"'
assert_contains "$auth_json" '"status": "logged_out"'
assert_contains "$auth_json" '"reason": "codex_auth_missing"'
assert_contains "$auth_json" '"redacted": true'
assert_not_contains "$auth_json" "api_key"

printf 'provider-fidelity: doctor --offline --json smoke\n'
run_gormes doctor --offline --json > "$doctor_json"
assert_contains "$doctor_json" '"name": "Custom endpoint"'
assert_contains "$doctor_json" '"summary": "configured provider=openai-codex endpoint=https://chatgpt.com/backend-api/codex missing=auth"'
assert_contains "$doctor_json" '"name": "auth"'
assert_contains "$doctor_json" 'run `gormes auth add openai-codex`'
assert_not_contains "$doctor_json" '"name": "api_key"'
assert_not_contains "$doctor_json" "plain-codex-access"
assert_not_contains "$doctor_json" "plain-codex-refresh"

printf 'provider-fidelity: config check --json smoke\n'
run_gormes config check --json > "$config_json"
assert_contains "$config_json" '"ok": true'
assert_contains "$config_json" '"issues": []'

printf 'provider-fidelity: setup help smoke\n'
run_gormes setup --help > "$setup_help"
assert_contains "$setup_help" "Guided interactive setup"
assert_contains "$setup_help" "--target"

printf 'provider-fidelity: model help smoke\n'
run_gormes model --help > "$model_help"
assert_contains "$model_help" "Interactively select the active model/provider"

if run_gormes onboard --help > "$onboard_help" 2>&1; then
  die "top-level onboard unexpectedly exists; update this fidelity pack to include it as a truth surface"
fi
assert_contains "$onboard_help" "gormes setup"
assert_contains "$onboard_help" "gormes doctor --offline --target terminal --json"

mkdir -p "$(dirname -- "$output")"
cat > "$output" <<JSON
{
  "schema_version": 1,
  "scope": "provider_setup_auth_fidelity",
  "generated_at": "$(json_escape "$generated_at")",
  "status": "pass",
  "real_homes_inspected": false,
  "real_homes_explicitly_allowed": $([ "$with_real_homes" -eq 1 ] && printf 'true' || printf 'false'),
  "gormes": {
    "branch": "$(json_escape "$gormes_branch")",
    "sha": "$(json_escape "$gormes_sha")"
  },
  "hermes": {
    "enabled": $([ "$with_hermes" -eq 1 ] && printf 'true' || printf 'false'),
    "src": "$(json_escape "$hermes_src")",
    "sha": "$(json_escape "$hermes_sha")",
    "source_refs": [
      "hermes_cli/main.py::_model_flow_openrouter",
      "hermes_cli/models.py::fetch_openrouter_models",
      "hermes_cli/main.py::_model_flow_openai_codex",
      "hermes_cli/status.py::OpenAI Codex auth status"
    ]
  },
  "behavior_atoms": [
    {
      "id": "codex-auth-no-creds",
      "status": "pass",
      "evidence": "auth status reports oauth_external logged_out/codex_auth_missing, and doctor asks for gormes auth add openai-codex without an api_key item.",
      "proposed_solution_if_regressed": "Fix the Codex auth resolver/doctor readiness path so openai-codex is OAuth-only and points operators to import/device-code login."
    },
    {
      "id": "codex-auth-configured-truth-surfaces",
      "status": "pass",
      "evidence": "Go regression TestProviderSetupAuthFidelityCodexTruthSurfacesConsistent ties auth status, config check, and doctor together with synthetic Codex OAuth tokens.",
      "proposed_solution_if_regressed": "Repair the shared credential resolver before changing individual command copy; all truth surfaces must read the same redacted auth state."
    },
    {
      "id": "openrouter-full-model-data-path",
      "status": "pass",
      "evidence": "Go regression TestFetchOpenRouterModelCatalogPreservesLargeModelsAPI proves a 64-model /models fixture is preserved end-to-end.",
      "proposed_solution_if_regressed": "Keep any picker pagination/search separate from fetchOpenRouterModelCatalog and ParseOpenRouterModelRegistry; never cap the underlying catalog."
    },
    {
      "id": "setup-model-help-truth-surfaces",
      "status": "pass",
      "evidence": "setup/model help smokes pass in a temp Gormes home; top-level onboard remains absent and points operators to setup plus doctor.",
      "proposed_solution_if_regressed": "Either register an onboard command as a real truth surface with tests, or keep the unknown-command guidance aligned with setup/doctor."
    }
  ],
  "validation_commands": [
    "go test ./cmd/gormes -run 'Auth|Provider|OpenRouter|Codex|Doctor|Model|Fallback|Setup' -count=1",
    "go test ./internal/config ./internal/provider ./internal/doctor -count=1",
    "temp-home gormes auth status openai-codex --json",
    "temp-home gormes doctor --offline --json",
    "temp-home gormes config check --json",
    "temp-home gormes setup --help",
    "temp-home gormes model --help"
  ]
}
JSON

printf 'provider-fidelity: report=%s\n' "$output"
