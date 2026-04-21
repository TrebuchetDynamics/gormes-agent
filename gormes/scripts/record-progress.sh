#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GORMES_DIR="$(dirname "$SCRIPT_DIR")"
PROGRESS_FILE="$GORMES_DIR/docs/data/progress.json"
LANDING_PROGRESS_FILE="$GORMES_DIR/docs/content/building-gormes/architecture_plan/progress.json"

if [[ ! -f "$PROGRESS_FILE" ]]; then
    echo "record-progress: progress.json not found — skipping"
    exit 0
fi

today=$(date +%Y-%m-%d)

python3 << PYEOF
import json
with open("$PROGRESS_FILE") as f:
    data = json.load(f)

data["meta"]["last_updated"] = "$today"

with open("$PROGRESS_FILE", "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")

print(f"progress.json updated: $today")
PYEOF

cp "$LANDING_PROGRESS_FILE" "$GORMES_DIR/www.gormes.ai/internal/site/data/progress.json"
