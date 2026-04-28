# Gormes Development Skills

This directory is the canonical source for repo-local Gormes skills.

Each skill lives at:

```text
docs/development-skills/<name>/SKILL.md
```

Loader directories are symlink views, not separate sources of truth:

```text
.agents/skills/<name> -> docs/development-skills/<name>
.claude/skills/<name> -> docs/development-skills/<name>
.codex/skills/<name>  -> docs/development-skills/<name>
```

Edit the `docs/development-skills` copy first. If a symlink is missing, recreate
the symlink instead of copying the skill.
