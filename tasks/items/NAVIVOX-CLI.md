+++
schema = 1
id = "NAVIVOX-CLI"
title = "Retire Navivox CLI command tree"
short = "Retire Navivox CLI tree"
status = "open"
goal = "PHASE-9"
section = "Next"
rank = 10
depends_on = []
touches = ["internal/app/gormescmd/", "internal/app/navivox/"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Remove the `gormes navivox` command tree. Unregister it from internal/app/gormescmd/main.go, delete internal/app/navivox, and make `gormes navivox` exit with a stable not-supported message that points at the CLI, the native TUI, and supported chat gateways. Keep unrelated CLI help tests intact."
acceptance = ["`gormes navivox pair` exits non-zero with not-supported guidance.", "`gormes --help` lists no navivox command.", "No package under internal/app/navivox remains."]
checks = ["go test ./internal/app/gormescmd -count=1", "go test ./internal/app/navivox -count=1"]
exclusions = ["Gateway registration or config loading still imports internal/app/navivox.", "A removal would break a documented quickstart or installer path that has not been updated."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
