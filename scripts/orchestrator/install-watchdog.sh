#!/usr/bin/env sh
set -eu
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
cd "$REPO_ROOT"
printf '%s\n' "builder-loop watchdog service install is retired."
printf '%s\n' "Use skill-driven work sessions plus go run ./cmd/progress validate."
exit 2
