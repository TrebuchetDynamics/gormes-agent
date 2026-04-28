#!/usr/bin/env sh
set -eu
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
cd "$REPO_ROOT"
printf '%s\n' "orchestrator audit is retired with cmd/builder-loop."
printf '%s\n' "Use repo-local skills for planning/building and go run ./cmd/progress validate for roadmap validation."
exit 2
