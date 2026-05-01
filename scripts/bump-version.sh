#!/usr/bin/env bash
set -euo pipefail

# bump-version.sh — Bump Gormes version, update CHANGELOG, and regenerate docs.
# Usage: ./scripts/bump-version.sh <patch|minor|major> [prerelease-suffix]
# Example: ./scripts/bump-version.sh patch scout

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
VERSION_FILE="${ROOT_DIR}/cmd/gormes/version.go"
CHANGELOG="${ROOT_DIR}/CHANGELOG.md"

BUMP="${1:-patch}"
PRERELEASE="${2:-}"

current=$(grep 'var Version =' "${VERSION_FILE}" | sed 's/.*"\(.*\)".*/\1/')
echo "Current version: ${current}"

base="${current%%-*}"
IFS='.' read -r major minor patch <<< "${base}"

case "${BUMP}" in
  major) major=$((major + 1)); minor=0; patch=0 ;;
  minor) minor=$((minor + 1)); patch=0 ;;
  patch) patch=$((patch + 1)) ;;
  *)
    echo "Unknown bump type: ${BUMP}. Use patch, minor, or major."
    exit 1
    ;;
esac

next="${major}.${minor}.${patch}"
if [ -n "${PRERELEASE}" ]; then
  next="${next}-${PRERELEASE}"
fi

echo "Next version: ${next}"

sed -i.bak "s/var Version = \".*\"/var Version = \"${next}\"/" "${VERSION_FILE}"
rm -f "${VERSION_FILE}.bak"

date=$(date +%Y-%m-%d)

awk -v ver="${next}" -v dt="${date}" '
  /## \[Unreleased\]/ {
    print
    print ""
    print "## [" ver "] - " dt
    next
  }
  { print }
' "${CHANGELOG}" > "${CHANGELOG}.tmp" && mv "${CHANGELOG}.tmp" "${CHANGELOG}"

cd "${ROOT_DIR}"
go run ./cmd/progress write

echo ""
echo "Version bumped to ${next}. Review the changes, then commit:"
echo "  git add cmd/gormes/version.go CHANGELOG.md docs/"
echo "  git commit -m 'release: bump version to ${next}'"
