#!/usr/bin/env bash
# record-benchmark.sh — measures bin/gormes and records metrics to benchmarks.json
# Called automatically by 'make build' after producing the binary.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GORMES_DIR="$(dirname "$SCRIPT_DIR")"
BENCHMARKS_FILE="$GORMES_DIR/benchmarks.json"
BINARY_PATH="${BINARY_PATH:-"$GORMES_DIR/bin/gormes"}"

# ── validation ────────────────────────────────────────────────────────────────
if [[ ! -f "$BINARY_PATH" ]]; then
    echo "record-benchmark: binary not found at $BINARY_PATH — skipping" >&2
    exit 0  # don't fail the build if called before binary exists
fi

# ── measure ─────────────────────────────────────────────────────────────────
size_bytes=$(stat -c%s "$BINARY_PATH")
size_mb=$(awk "BEGIN {printf \"%.1f\", $size_bytes / 1048576}")
today=$(date +%Y-%m-%d)

# ── read existing benchmarks.json ─────────────────────────────────────────────
if [[ -f "$BENCHMARKS_FILE" ]]; then
    # Use python3 for reliable JSON manipulation (available everywhere)
    python3 << PYEOF
import json
import os
import subprocess

with open("$BENCHMARKS_FILE") as f:
    data = json.load(f)

size_bytes = $size_bytes
size_mb = $size_mb

# Update binary metrics
data["binary"]["size_bytes"] = size_bytes
data["binary"]["size_mb"] = str(size_mb)
data["binary"]["last_measured"] = "$today"

# Check if last history entry is from today (avoid duplicate entries)
history = data.get("history", [])
data["history"] = history

def phase_from_arch_plan(path):
    if not os.path.exists(path):
        return ""
    with open(path) as f:
        text = f.read()
    marker = "## Phase"
    if marker not in text:
        return ""
    return "Phase " + text.split(marker, 1)[1].split("\n", 1)[0].strip()

def subphase_complete(subphase):
    items = subphase.get("items")
    if items:
        return all(item.get("status") == "complete" for item in items)
    return subphase.get("status") == "complete"

def phase_from_progress(path):
    if not os.path.exists(path):
        return ""
    with open(path) as f:
        progress = json.load(f)
    phases = progress.get("phases", {})
    if not phases:
        return ""
    ordered = sorted(phases.items(), key=lambda item: int(item[0]))
    for _, phase in ordered:
        subphases = phase.get("subphases", {})
        if not subphases:
            return phase.get("name", "")
        if not all(subphase_complete(sp) for sp in subphases.values()):
            return phase.get("name", "")
    return ordered[-1][1].get("name", "")

if not history or history[0].get("date") != "$today":
    # Prepend new entry (most recent first)
    commit = subprocess.check_output(
        ["git", "rev-parse", "--short", "HEAD"],
        text=True
    ).strip()
    phase = (
        phase_from_arch_plan("$GORMES_DIR/docs/ARCH_PLAN.md")
        or phase_from_progress("$GORMES_DIR/docs/content/building-gormes/architecture_plan/progress.json")
        or next((entry.get("phase", "") for entry in history if entry.get("phase")), "unknown")
    )
    data["history"].insert(0, {
        "date": "$today",
        "size_mb": size_mb,
        "commit": commit,
        "phase": phase
    })

with open("$BENCHMARKS_FILE", "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")

print(f"benchmarks.json updated: {size_mb} MB")
PYEOF
else
    # Create new file
    cat > "$BENCHMARKS_FILE" << EOF
{
  "binary": {
    "name": "gormes",
    "path": "bin/gormes",
    "size_bytes": $size_bytes,
    "size_mb": "$size_mb",
    "build_flags": "CGO_ENABLED=0 -trimpath -ldflags=\\"-s -w\\"",
    "linker": "static",
    "stripped": true,
    "go_version": "1.25+",
    "last_measured": "$today"
  },
  "properties": {
    "cgo": false,
    "dependencies": "zero (no dynamic library deps)",
    "platforms": ["linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64"]
  },
  "history": [
    {
      "date": "$today",
      "size_mb": $size_mb,
      "phase": "unknown"
    }
  ]
}
EOF
    echo "benchmarks.json created: $size_mb MB"
fi

# ── copy to Hugo data directory for docs site ────────────────────────────────
docs_data="$GORMES_DIR/docs/data"
mkdir -p "$docs_data"
cp "$BENCHMARKS_FILE" "$docs_data/benchmarks.json"
echo "copied to docs/data/benchmarks.json"
