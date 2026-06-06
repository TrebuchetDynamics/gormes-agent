#!/usr/bin/env python3
"""Audit repo-local Gormes skill hygiene.

This is intentionally lightweight: it catches loader-view mistakes and the
high-impact wording regressions that have caused bad routing in past Gormes
sessions. It does not replace human review of the skill workflow.
"""

from __future__ import annotations

import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SKILLS_DIR = ROOT / "development-skills"
LOADER_DIRS = [ROOT / ".agents" / "skills", ROOT / ".claude" / "skills", ROOT / ".codex" / "skills"]
MAX_DESCRIPTION_CHARS = 240

BAD_PATTERNS = {
    r"atoms are the only\s+executable queue": "progress rows, not behavior atoms, are the executable backlog",
    r"progress\.json`? \(legacy": "progress.json/logical progress data is not legacy",
    r"implementation intent goes into `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS\.md`": "implementation intent must go into the progress control plane",
    r"Put implementation intent in the parity evidence doc": "implementation intent must go into the progress control plane",
    r"represented in `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS\.md` only": "progress rows are the backlog-safe representation",
    r"parity evidence doc captures what remains to be built": "progress rows capture backlog work; parity docs are evidence",
    r"internal/progress": "the Go package path is internal/planning/progress",
    r"go test \./internal/progress": "use go test ./internal/planning/progress",
    r"continue\s+with the next safe domain": "cmd refactor passes must stop after one domain",
    r"/home/xel/git/sages-openclaw/workspace-mineru/gormes-agent": "use the current git root, not a stale hard-coded repo path",
    r"/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os": "resolve donor roots dynamically; do not require this stale path",
}

REQUIRED_SNIPPETS = {
    "cmd-internal-refactor": {
        "SKILL.md": [
            "internal/platform/cli/gormescli",
            "internal/app/<domain>",
            "hermes-knowledge-graph.json",
            "Stop after one domain",
            "go test ./internal/support/repochecks",
            "folder_refactor_audit",
            "codemap.md",
            "Bug-finding refactor oracle",
            "introduced regression",
            "references/bug-finding.md",
        ],
        "references/bug-finding.md": [
            "Baseline oracle",
            "Common refactor bug traps",
            "preexisting bug",
            "introduced regression",
            "parity drift",
            "bug oracle before",
        ],
        "references/domain-folder-topology.md": [
            "cmd/gormes/main.go -> gormescli -> internal/app/<domain>",
            "folder_refactor_scan",
            "direct internal imports",
            "Do not use an empty `-run` selector",
            "codemap.md` cannot remain",
            "Tests-only slices are valid",
        ],
    }
}


def parse_frontmatter(path: Path) -> tuple[str | None, str | None]:
    lines = path.read_text(encoding="utf-8").splitlines()
    if not lines or lines[0].strip() != "---":
        return None, None
    name = None
    description = None
    for line in lines[1:]:
        if line.strip() == "---":
            break
        if line.startswith("name:"):
            name = line.split(":", 1)[1].strip()
        if line.startswith("description:"):
            description = line.split(":", 1)[1].strip()
    return name, description


def skill_dirs() -> list[Path]:
    return sorted(p.parent for p in SKILLS_DIR.glob("*/SKILL.md"))


def audit_frontmatter(errors: list[str]) -> None:
    for directory in skill_dirs():
        path = directory / "SKILL.md"
        name, description = parse_frontmatter(path)
        if not name:
            errors.append(f"{path.relative_to(ROOT)}: missing frontmatter name")
        elif name != directory.name:
            errors.append(f"{path.relative_to(ROOT)}: name {name!r} != directory {directory.name!r}")
        if not description:
            errors.append(f"{path.relative_to(ROOT)}: missing frontmatter description")
        elif len(description) > MAX_DESCRIPTION_CHARS:
            errors.append(
                f"{path.relative_to(ROOT)}: description is {len(description)} chars; keep <= {MAX_DESCRIPTION_CHARS}"
            )


def audit_bad_patterns(errors: list[str]) -> None:
    for path in sorted(SKILLS_DIR.rglob("*")):
        if path == Path(__file__).resolve():
            continue
        if not path.is_file() or path.suffix.lower() not in {".md", ".py"}:
            continue
        text = path.read_text(encoding="utf-8")
        for pattern, message in BAD_PATTERNS.items():
            for match in re.finditer(pattern, text, flags=re.IGNORECASE | re.MULTILINE):
                line_no = text.count("\n", 0, match.start()) + 1
                errors.append(f"{path.relative_to(ROOT)}:{line_no}: {message}")


def audit_loader_views(errors: list[str]) -> None:
    names = [p.name for p in skill_dirs()]
    for loader in LOADER_DIRS:
        if not loader.is_dir():
            errors.append(f"{loader.relative_to(ROOT)}: missing loader directory")
            continue
        for name in names:
            entry = loader / name
            if not entry.exists():
                errors.append(f"{entry.relative_to(ROOT)}: missing loader symlink")
                continue
            if not entry.is_symlink():
                errors.append(f"{entry.relative_to(ROOT)}: must be a symlink to development-skills/{name}")
                continue
            target = os.readlink(entry)
            expected = f"../../development-skills/{name}"
            if target != expected:
                errors.append(f"{entry.relative_to(ROOT)}: target {target!r}, want {expected!r}")
            real = entry.resolve()
            want = (SKILLS_DIR / name).resolve()
            if real != want:
                errors.append(f"{entry.relative_to(ROOT)}: resolves to {real}, want {want}")


def audit_required_snippets(errors: list[str]) -> None:
    for skill_name, files in REQUIRED_SNIPPETS.items():
        skill_root = SKILLS_DIR / skill_name
        for rel_path, snippets in files.items():
            path = skill_root / rel_path
            if not path.exists():
                errors.append(f"{path.relative_to(ROOT)}: missing required skill file")
                continue
            text = path.read_text(encoding="utf-8")
            for snippet in snippets:
                if snippet not in text:
                    errors.append(f"{path.relative_to(ROOT)}: missing required snippet {snippet!r}")


def main() -> int:
    errors: list[str] = []
    audit_frontmatter(errors)
    audit_bad_patterns(errors)
    audit_required_snippets(errors)
    audit_loader_views(errors)
    if errors:
        print("local skill audit failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1
    print(f"local skill audit passed: {len(skill_dirs())} skills, {len(LOADER_DIRS)} loader views")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
