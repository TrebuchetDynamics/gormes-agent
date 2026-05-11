#!/usr/bin/env bash
# build.sh — Build Gormes from source with release-quality flags.
# Usage:
#   ./build.sh              # build + benchmark + install
#   ./build.sh --slim       # slim build (no WASM whisper, TTS, image gen)
#   ./build.sh --no-install # build only, don't install to ~/.gormes/bin
set -euo pipefail

cd "$(git rev-parse --show-toplevel 2>/dev/null || dirname "$0")"

# ── Read version from source ────────────────────────────────────────────
VERSION="$(grep -E '^\s*Version\s*=\s*"' cmd/gormes/version.go | sed 's/.*"\(.*\)".*/\1/')"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
GIT_DIRTY="$(git diff --quiet 2>/dev/null && git diff --cached --quiet 2>/dev/null && echo false || echo true)"
BUILD_FLAGS="-trimpath -ldflags=-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.GitDirty=${GIT_DIRTY}"
BINARY_PATH="bin/gormes"
SLIM=false

for arg in "$@"; do
  case "$arg" in
    --slim) SLIM=true; BINARY_PATH="bin/gormes-slim"; BUILD_FLAGS="$BUILD_FLAGS -tags slim" ;;
    --no-install) NO_INSTALL=true ;;
  esac
done

echo "==> Building Gormes ${VERSION} (commit=${GIT_COMMIT}, dirty=${GIT_DIRTY})"
echo "    Flags: ${BUILD_FLAGS}"
echo "    Output: ${BINARY_PATH}"

# ── Build ───────────────────────────────────────────────────────────────
mkdir -p bin
CGO_ENABLED=0 go build $BUILD_FLAGS -o "$BINARY_PATH" ./cmd/gormes

# ── Size ────────────────────────────────────────────────────────────────
RAW_SIZE=$(stat -c%s "$BINARY_PATH" 2>/dev/null || stat -f%z "$BINARY_PATH" 2>/dev/null)
echo "    Binary size: $(( RAW_SIZE / 1024 / 1024 )) MB ($(numfmt --to=iec $RAW_SIZE 2>/dev/null || echo "${RAW_SIZE} bytes"))"

# ── Post-build steps (skip for --slim, benchmarks compare full builds) ──
if [ "$SLIM" = false ]; then
  echo "==> Recording benchmark..."
  go run ./cmd/repoctl benchmark record 2>/dev/null || echo "    (benchmark record skipped)"

  echo "==> Regenerating progress docs..."
  go run ./cmd/progress write 2>/dev/null || echo "    (progress write skipped)"

  echo "==> Updating README..."
  go run ./cmd/repoctl readme update 2>/dev/null || echo "    (readme update skipped)"

  echo "==> Syncing landing page assets..."
  node webpages/landing/scripts/sync-assets.mjs 2>/dev/null || echo "    (sync-assets skipped)"
fi

# ── Install to ~/.gormes/bin ────────────────────────────────────────────
if [ "${NO_INSTALL:-false}" = false ]; then
  INSTALL_DIR="${GORMES_HOME:-$HOME/.gormes}/bin"
  mkdir -p "$INSTALL_DIR"

  # Stop gateway if running (so we can replace the binary)
  if pgrep -f "gormes gateway" >/dev/null 2>&1; then
    echo "==> Stopping gormes gateway..."
    GATEWAY_PID=$(pgrep -f "gormes gateway" | head -1)
    kill "$GATEWAY_PID" 2>/dev/null || true
    sleep 1
  fi

  cp "$BINARY_PATH" "$INSTALL_DIR/gormes"
  echo "==> Installed to ${INSTALL_DIR}/gormes"

  # Update ~/.local/bin/gormes symlink
  LOCAL_BIN="$HOME/.local/bin"
  if [ -d "$LOCAL_BIN" ]; then
    ln -sf "$INSTALL_DIR/gormes" "$LOCAL_BIN/gormes"
    echo "==> Symlinked to ${LOCAL_BIN}/gormes"
  fi

  # Restart gateway if it was running before
  if [ -n "${GATEWAY_PID:-}" ]; then
    echo "==> Restarting gormes gateway..."
    nohup "$INSTALL_DIR/gormes" gateway >> "$HOME/.local/state/gormes-agent/gateway.log" 2>&1 &
    echo "    Gateway PID $!"
  fi
fi

echo ""
echo "✅ Build complete: ${BINARY_PATH} ($(stat -c%s "$BINARY_PATH" 2>/dev/null | numfmt --to=iec 2>/dev/null || echo "${RAW_SIZE} bytes"))"
