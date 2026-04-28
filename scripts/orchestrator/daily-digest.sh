#!/usr/bin/env sh
set -eu
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
cd "$REPO_ROOT"
printf '%s\n' "orchestrator daily digest is retired with cmd/builder-loop."
printf '%s\n' "Use repo-local skills and progress.json evidence for handoff summaries."
exit 2
