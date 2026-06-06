#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
kit_dir="$repo_root/examples/agentic-porting-kit"
example_dir="$kit_dir/python-greeter-to-go"
skills_dir="$kit_dir/skills"
readme_path="$kit_dir/README.md"
license_path="$kit_dir/LICENSE"
schema_path="$repo_root/schemas/agentic-porting-kit-progress.schema.json"
progress_path="$example_dir/progress.json"

python3 - "$schema_path" "$progress_path" "$skills_dir" "$readme_path" "$license_path" <<'PY'
import json
import re
import sys
from pathlib import Path

schema_path = Path(sys.argv[1])
progress_path = Path(sys.argv[2])
skills_dir = Path(sys.argv[3])
readme_path = Path(sys.argv[4])
license_path = Path(sys.argv[5])
schema = json.loads(schema_path.read_text(encoding="utf-8"))
progress = json.loads(progress_path.read_text(encoding="utf-8"))

if "cmd/progress" in progress_path.read_text(encoding="utf-8"):
    raise SystemExit("example progress must not depend on Gormes cmd/progress")
if schema.get("title") != "Agentic Porting Kit Progress":
    raise SystemExit("unexpected schema title")
for field in ("project", "source", "target", "rows"):
    if field not in progress:
        raise SystemExit(f"progress.json missing {field}")
if not isinstance(progress["rows"], list) or not progress["rows"]:
    raise SystemExit("progress.json rows must be a non-empty array")
required_row_fields = {
    "name",
    "status",
    "contract",
    "source_refs",
    "write_scope",
    "test_commands",
    "acceptance",
    "done_signal",
}
for index, row in enumerate(progress["rows"], start=1):
    missing = sorted(required_row_fields.difference(row))
    if missing:
        raise SystemExit(f"row {index} missing required fields: {', '.join(missing)}")
    for list_field in ("source_refs", "write_scope", "test_commands", "acceptance", "done_signal"):
        if not isinstance(row[list_field], list) or not row[list_field]:
            raise SystemExit(f"row {index} field {list_field} must be a non-empty list")
print("progress schema smoke: ok")

expected_skills = [
    "porting-skill-manager",
    "porting-planner",
    "porting-builder",
    "porting-tdd-slice",
    "porting-parity-auditor",
    "porting-references",
]
for name in expected_skills:
    skill_path = skills_dir / name / "SKILL.md"
    if not skill_path.exists():
        raise SystemExit(f"missing skill: {skill_path}")
    text = skill_path.read_text(encoding="utf-8")
    frontmatter = re.match(r"^---\n(.*?)\n---\n", text, re.DOTALL)
    if not frontmatter:
        raise SystemExit(f"{skill_path}: missing YAML frontmatter")
    yaml = frontmatter.group(1)
    if f"name: {name}" not in yaml:
        raise SystemExit(f"{skill_path}: frontmatter name must be {name}")
    description = next((line.split(":", 1)[1].strip() for line in yaml.splitlines() if line.startswith("description:")), "")
    if not description.startswith("Use when"):
        raise SystemExit(f"{skill_path}: description must start with 'Use when'")
    for required in ("source implementation", "target implementation", "PORTING_PROGRESS_PATH"):
        if required not in text:
            raise SystemExit(f"{skill_path}: missing required phrase {required!r}")
    forbidden = [
        "webpages/docs/content/building-gormes",
        "cmd/progress",
        "github.com/TrebuchetDynamics/gormes-agent",
        "/home/xel/",
        "Gormes",
        "Hermes",
    ]
    for token in forbidden:
        if token in text:
            raise SystemExit(f"{skill_path}: forbidden Gormes-specific token {token!r}")
print("porting skill skeletons: ok")

if not readme_path.exists():
    raise SystemExit(f"missing README fixture: {readme_path}")
if not license_path.exists():
    raise SystemExit(f"missing LICENSE fixture: {license_path}")
readme = readme_path.read_text(encoding="utf-8")
license_text = license_path.read_text(encoding="utf-8")
required_readme_phrases = [
    "Validation-gated agentic porting for teams moving behavior from one runtime to another without losing tests, traceability, or source attribution.",
    "Codex",
    "Claude Code",
    "examples/python-greeter-to-go",
    "PORTING_PROGRESS_PATH",
    "schemas/progress.schema.json",
    "scripts/validate-example.sh",
    "Gormes proves the method",
    "## License",
]
for phrase in required_readme_phrases:
    if phrase not in readme:
        raise SystemExit(f"README fixture missing required phrase: {phrase}")
for token in ("/home/xel/", "webpages/docs/content/building-gormes", "cmd/progress"):
    if token in readme:
        raise SystemExit(f"README fixture contains Gormes-local token: {token}")
required_license_phrases = ["MIT License", "Trebuchet Dynamics", "Permission is hereby granted"]
for phrase in required_license_phrases:
    if phrase not in license_text:
        raise SystemExit(f"LICENSE fixture missing required phrase: {phrase}")
print("readme license fixture: ok")
PY

(cd "$example_dir/target" && go test ./...)
"$script_dir/validate-public-layout.sh"

echo "agentic-porting-kit local example: ok"
