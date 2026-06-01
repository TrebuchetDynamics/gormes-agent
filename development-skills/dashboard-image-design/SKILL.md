---
name: dashboard-image-design
description: Use when designing, critiquing, or polishing dashboard screenshots, hero images, social cards, visual mockups, or image-based dashboard UI assets for agent/operator products.
---

# Dashboard Image Design

## Mission

Turn dashboard screenshots or dashboard concepts into clear, trustworthy visual assets without inventing product claims or hiding operational state.

## Use For

- Dashboard screenshot critique, crop, annotation, or redesign direction.
- Landing-page hero images, social cards, README images, or launch visuals that show a dashboard.
- Image-based review of hierarchy, contrast, density, empty/loading/error states, and trust signals.
- Text-only design specs for future visual or frontend implementation.

Do not use this for backend dashboard API behavior, auth/security, or runtime status contracts; route those through the relevant Gormes dashboard/API or parity skill.

## Workflow

1. **Anchor the source.** Identify the exact screenshot, mockup, or page path. If none exists, state assumptions and produce a wireframe/spec instead of pretending to inspect an image.
2. **Define the job.** Name audience, surface, aspect ratio, and success criterion: e.g. README proof, landing hero, operator status, incident debug, or social preview.
3. **Prioritize truth.** Preserve real status, timestamps, errors, and limitations. Redact secrets and personal data; never beautify failures into success.
4. **Improve hierarchy.** Check first-read headline, primary metric, active state, next action, and secondary detail. Remove decorative clutter before adding style.
5. **Design for accessibility.** Require readable contrast, non-color status labels, large-enough text, and a short alt-text description.
6. **Output implementable guidance.** Provide concise bullets: crop/framing, layout, copy, color/status treatment, annotations, export sizes, and acceptance checks.

## Quality Checklist

- Source image or explicit no-image assumption is named.
- No secrets, tokens, private chats, emails, or user IDs are visible.
- The image communicates one main point in under five seconds.
- Status colors have text labels; red/yellow/green are not the only signal.
- Empty, degraded, loading, and error states are truthful if present.
- Alt text is included for every final image recommendation.
- Claims match shipped product evidence or are marked as mockup/future.

## Output Shape

```text
source:
purpose:
audience:
main message:
recommended crop/framing:
hierarchy changes:
copy/annotation changes:
accessibility notes:
redactions:
export sizes:
acceptance checks:
```

## Validation

Run from the Gormes repo root after creating, editing, or routing this skill:

```sh
python3 /home/xel/.codex/skills/.system/skill-creator/scripts/quick_validate.py development-skills/dashboard-image-design
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -path '*/dashboard-image-design/SKILL.md' -print | sort
python3 - <<'PY'
from pathlib import Path
required = {
    'AGENTS.md': ['dashboard-image-design', 'dashboard screenshots'],
    'development-skills/gormes-skill-manager/SKILL.md': ['dashboard-image-design', 'dashboard screenshot'],
    'references/skill-routing.md': ['dashboard-image-design'],
    'development-skills/gormes-skill-manager/references/skill-routing.md': ['dashboard-image-design'],
}
missing = []
for path, needles in required.items():
    text = Path(path).read_text().lower()
    for needle in needles:
        if needle.lower() not in text:
            missing.append(f'{path}: missing {needle!r}')
if missing:
    raise SystemExit('\n'.join(missing))
print('dashboard-image-design routing coverage present')
PY
git diff --check
```

Expected loader output includes `.agents/skills/dashboard-image-design/SKILL.md`,
`.claude/skills/dashboard-image-design/SKILL.md`, and
`.codex/skills/dashboard-image-design/SKILL.md` when the local ignored Codex
loader view exists.

## Common Mistakes

- Designing a pretty fake dashboard instead of an honest operator artifact.
- Showing secrets, private messages, or irreversible actions in screenshots.
- Using tiny terminal text that is unreadable in README/social-card contexts.
- Omitting degraded/error states that would make the product feel more trustworthy.
- Giving vague taste notes instead of implementation-ready crop, copy, and contrast guidance.
