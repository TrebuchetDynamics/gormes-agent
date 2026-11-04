+++
schema = 1
id = "MCP-LIFECYCLE"
title = "Hermes MCP catalog install/configure lifecycle"
short = "MCP catalog lifecycle"
status = "open"
goal = "PHASE-5"
section = "Needs decision"
rank = 20
depends_on = []
touches = ["internal/tools/mcp/catalog/", "internal/tools/mcp/config/", "internal/tools/mcp/probe/", "internal/platform/cli/gormescli/", "webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md", "webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md", "webpages/docs/content/building-gormes/architecture_plan/progress.json"]
resources = []
kind = "implementation"
ownership = "coordinator"
scope = "Port the remaining approved MCP catalog lifecycle after the read-only manifest/list row: install config-only stdio/http entries, collect declared env values through existing secret-safe operator seams, preserve prior tool selections on reinstall, probe tools through injected clients, and explicitly update/disable/remove entries. Pinned git-install/bootstrap manifests require a separate Go-native process and network policy; Hermes shell=True bootstrap is not an acceptable implementation contract."
acceptance = ["A future planner split provides fixture-ready acceptance for config-only install/update/disable/remove before any git/bootstrap behavior.", "Reinstall preserves prior user tool selection, probe failures apply manifest defaults deterministically, and secrets never enter config/output/logs.", "Git/bootstrap entries remain disabled until the separately approved process/network policy is implemented with rollback evidence."]
checks = ["go test ./internal/tools/mcp/catalog ./internal/tools/mcp/config ./internal/tools/mcp/probe -count=1", "go test ./internal/platform/cli/gormescli -run 'MCP.*Catalog|MCPCommand' -count=1", "go run ./cmd/progress validate"]
exclusions = ["Owner has not approved outbound URL/ref allowlisting, redirect policy, clone size/depth, install root, symlink/atomic replacement, bootstrap executable allowlist/argv/cwd/environment, timeout/cancellation/output caps, and rollback policy.", "Secret prompts and dotenv/config writes lack explicit profile-root confinement, 0600/atomicity, redaction, duplicate/update semantics, and crash recovery.", "Implementation would port Hermes shell=True bootstrap, auto-update catalog entries, use live MCP/network/process fixtures, or combine interactive picker UI with persistence mechanics in one untestable slice."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
