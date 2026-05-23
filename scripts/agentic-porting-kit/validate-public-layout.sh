#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
kit_dir="$repo_root/examples/agentic-porting-kit"
tmp_parent="${TMPDIR:-/tmp}"
work_dir="$(mktemp -d "${tmp_parent%/}/agentic-porting-kit-public-layout.XXXXXX")"
cleanup() {
  if [[ "${KEEP_AGENTIC_PORTING_KIT_LAYOUT:-}" != "1" ]]; then
    rm -rf "$work_dir"
  else
    printf 'kept public layout: %s\n' "$public_dir"
  fi
}
trap cleanup EXIT

public_dir="$work_dir/agentic-porting-kit"
mkdir -p "$public_dir/schemas" "$public_dir/scripts" "$public_dir/examples"

cp "$kit_dir/README.md" "$public_dir/README.md"
cp "$kit_dir/LICENSE" "$public_dir/LICENSE"
cp "$repo_root/schemas/agentic-porting-kit-progress.schema.json" "$public_dir/schemas/progress.schema.json"
cp "$kit_dir/scripts/validate-example.sh" "$public_dir/scripts/validate-example.sh"
chmod +x "$public_dir/scripts/validate-example.sh"
cp -R "$kit_dir/skills" "$public_dir/skills"
cp -R "$kit_dir/python-greeter-to-go" "$public_dir/examples/python-greeter-to-go"

python3 - "$public_dir" <<'PY'
import sys
from pathlib import Path

public_dir = Path(sys.argv[1])
required_paths = [
    "README.md",
    "LICENSE",
    "schemas/progress.schema.json",
    "scripts/validate-example.sh",
    "skills/porting-skill-manager/SKILL.md",
    "skills/porting-planner/SKILL.md",
    "skills/porting-builder/SKILL.md",
    "skills/porting-tdd-slice/SKILL.md",
    "skills/porting-parity-auditor/SKILL.md",
    "skills/porting-references/SKILL.md",
    "examples/python-greeter-to-go/progress.json",
    "examples/python-greeter-to-go/source/greeter.py",
    "examples/python-greeter-to-go/target/go.mod",
    "examples/python-greeter-to-go/target/greeter.go",
    "examples/python-greeter-to-go/target/greeter_test.go",
]
for rel in required_paths:
    path = public_dir / rel
    if not path.exists():
        raise SystemExit(f"assembled public layout missing {rel}")
for path in public_dir.rglob("*"):
    if not path.is_file():
        continue
    text = path.read_text(encoding="utf-8", errors="ignore")
    if "/home/xel/" in text:
        raise SystemExit(f"assembled public layout contains local path in {path.relative_to(public_dir)}")
print("public layout shape: ok")
PY

(cd "$public_dir" && ./scripts/validate-example.sh)

echo "agentic-porting-kit public layout: ok"
