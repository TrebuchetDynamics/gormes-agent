#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE=""
FETCH_REMOTE=1

usage() {
  cat <<'USAGE'
usage: scripts/hermes-parity-refresh.sh --check|--update [--no-fetch]

Phase 0 parity refresh.

  --check   Read-only validation. Validates the source-pair manifest against
            the currently checked out Hermes SHA and reports whether
            origin/main is newer when remote metadata is available.
  --update  Safe update. Requires a clean Hermes checkout, fetches origin/main,
            fast-forwards the detached checkout to that commit, syncs the
            source-pair manifest SHA, regenerates reports, and validates
            progress metadata.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check)
      MODE="check"
      shift
      ;;
    --update)
      MODE="update"
      shift
      ;;
    --no-fetch)
      FETCH_REMOTE=0
      shift
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

if [[ "$MODE" != "check" && "$MODE" != "update" ]]; then
  usage >&2
  exit 2
fi

HERMES_SRC="${HERMES_SRC:-$ROOT/hermes-agent}"
if [[ ! -d "$HERMES_SRC" || ! -f "$HERMES_SRC/hermes_cli/main.py" ]]; then
  echo "error: HERMES_SRC does not look like hermes-agent: $HERMES_SRC" >&2
  exit 2
fi

current_sha="$(git -C "$HERMES_SRC" rev-parse HEAD)"
echo "phase0: hermes current_sha=$current_sha"

upstream_sha=""
if [[ "$FETCH_REMOTE" -eq 1 ]]; then
  if git -C "$HERMES_SRC" fetch --quiet origin main; then
    upstream_sha="$(git -C "$HERMES_SRC" rev-parse origin/main)"
    echo "phase0: hermes origin_main_sha=$upstream_sha"
    if [[ "$MODE" == "check" && "$current_sha" != "$upstream_sha" ]]; then
      echo "phase0: stale upstream checkout; run scripts/hermes-parity-refresh.sh --update" >&2
      exit 1
    fi
  else
    if [[ "$MODE" == "update" ]]; then
      echo "error: cannot update without fetching origin/main" >&2
      exit 1
    fi
    echo "phase0: warning: could not fetch origin/main; continuing with local checkout" >&2
  fi
else
  upstream_sha="$(git -C "$HERMES_SRC" rev-parse origin/main 2>/dev/null || true)"
  if [[ "$MODE" == "update" && -n "$upstream_sha" ]]; then
    echo "phase0: hermes cached_origin_main_sha=$upstream_sha"
  fi
fi

if [[ "$MODE" == "update" ]]; then
  if [[ -z "$upstream_sha" ]]; then
    echo "error: cannot update because origin/main is unavailable" >&2
    exit 1
  fi
  if [[ -n "$(git -C "$HERMES_SRC" status --porcelain=v1)" ]]; then
    echo "error: Hermes checkout has uncommitted changes: $HERMES_SRC" >&2
    exit 1
  fi
  if [[ "$current_sha" != "$upstream_sha" ]]; then
    if ! git -C "$HERMES_SRC" merge-base --is-ancestor "$current_sha" "$upstream_sha"; then
      echo "error: refusing non-fast-forward Hermes update current=$current_sha upstream=$upstream_sha" >&2
      exit 1
    fi
    git -C "$HERMES_SRC" checkout --quiet "$upstream_sha"
    current_sha="$(git -C "$HERMES_SRC" rev-parse HEAD)"
    echo "phase0: hermes updated_sha=$current_sha"
  else
    echo "phase0: hermes already current"
  fi
  go run ./cmd/gormes-repo --repo-root "$ROOT" hermes-source-pairs sync-sha --hermes-src "$HERMES_SRC" --hermes-sha "$current_sha"
  HERMES_SRC="$HERMES_SRC" scripts/hermes-py2many-map.sh
  go run ./cmd/gormes-repo --repo-root "$ROOT" hermes-source-pairs report --hermes-src "$HERMES_SRC" --hermes-sha "$current_sha"
  go run ./cmd/progress validate
  echo "phase0: update ok"
  exit 0
fi

go run ./cmd/gormes-repo --repo-root "$ROOT" hermes-source-pairs validate --hermes-src "$HERMES_SRC" --hermes-sha "$current_sha"
HERMES_SRC="$HERMES_SRC" scripts/hermes-py2many-map.sh --dry-run

echo "phase0: check ok"
