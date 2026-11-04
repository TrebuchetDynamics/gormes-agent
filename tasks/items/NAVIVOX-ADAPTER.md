+++
schema = 1
id = "NAVIVOX-ADAPTER"
title = "Remove Navivox adapter, pairing, and setup packages"
short = "Remove Navivox packages"
status = "open"
goal = "PHASE-9"
section = "Next"
rank = 40
depends_on = ["NAVIVOX-CONFIG"]
touches = ["internal/adapters/channels/", "internal/config/channels/", "internal/app/setupnavivox/", "internal/app/gateway/", "internal/persistence/session/", "internal/platform/cli/gormescli/setupnavivox.go", "internal/platform/cli/gormescli/setup_gateway.go", "internal/platform/cli/gormescli/setupfirst.go", "internal/platform/cli/gormescli/setup_gateway_test.go", "internal/platform/cli/gormescli/setup_profiles_test_helpers_test.go"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Delete the remaining Navivox runtime packages: internal/adapters/channels/navivox (HTTP/WebSocket channel, pairing, device credentials, profile admin, voice profiles), internal/config/channels/navivox, and internal/app/setupnavivox. Remove the factory wiring in DefaultChannelFactories and any session credential persistence that only serves paired Navivox devices, keeping unrelated channel and session behavior intact. Remove the Navivox setup-profile section and its seams in internal/platform/cli/gormescli (setupnavivox.go, setup_gateway.go, setupfirst.go) plus their focused tests, so setup gateway and setup-first no longer offer a Navivox step."
acceptance = ["The three Navivox packages are gone and the repository builds.", "No non-Navivox channel or session test changes meaning.", "Stored Navivox artifacts are ignored without deletion.", "`gormes setup gateway` and setup-first no longer offer a Navivox section."]
checks = ["go test ./internal/adapters/channels/... -count=1", "go test ./internal/persistence/session -count=1", "go build ./..."]
exclusions = ["The session credential store cannot separate Navivox-owned records from shared records.", "Removal would change unrelated channel registration or session behavior."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
