#!/usr/bin/env sh
set -eu
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
cd "$REPO_ROOT"
printf '%s\n' "builder-loop legacy timer management is retired."
printf '%s\n' "Remove any old gormes-builder-loop/planner-loop user units manually if present."
exit 2
