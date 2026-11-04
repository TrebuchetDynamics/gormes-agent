+++
schema = 1
id = "NAVIVOX-PROGRESS"
title = "Retire Navivox progress module and navivox-telegram-ui skill"
short = "Retire Navivox backlog"
status = "open"
goal = "PHASE-9"
section = "Next"
rank = 60
depends_on = ["NAVIVOX-DOCS"]
touches = ["webpages/docs/content/building-gormes/architecture_plan/progress.json", "development-skills/", ".agents/skills/", ".claude/skills/", ".codex/skills/", "AGENTS.md"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Retire the Navivox backlog and skill surface. Remove the navivox progress module rows or mark them superseded through the typed backlog, delete development-skills/navivox-telegram-ui with its .agents, .claude, and .codex symlink views, and update skill indexes and validation lists so no loader advertises it."
acceptance = ["The navivox module has no active rows.", "development-skills/navivox-telegram-ui and its three loader symlinks are gone.", "Progress validation and docs regeneration pass."]
checks = ["go run ./cmd/progress validate", "go run ./cmd/progress write", "go test ./internal/planning/progress -count=1"]
exclusions = ["Removing the module would break progress docs regeneration or alias pages.", "Symlink loader views cannot be recreated safely from development-skills."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
