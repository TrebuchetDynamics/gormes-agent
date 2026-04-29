#!/bin/sh
# Gormes OCI entrypoint: bootstrap config volume and forward to the binary.
# Pure POSIX sh so it runs on distroless static images without bash.

set -eu

GORMES_HOME="${GORMES_HOME:-/opt/data}"
INSTALL_DIR="/opt/gormes"

mkdir -p "$GORMES_HOME"

if [ ! -f "$GORMES_HOME/config.yaml" ]; then
    # Seed an empty offline-safe config so `gormes doctor --offline`
    # can read the volume without contacting providers or registries.
    printf 'gormes:\n  offline: true\n' > "$GORMES_HOME/config.yaml"
fi

# No args: deterministic offline doctor smoke against the seeded volume.
if [ "$#" -eq 0 ]; then
    exec /usr/local/bin/gormes doctor --offline
fi

exec /usr/local/bin/gormes "$@"
