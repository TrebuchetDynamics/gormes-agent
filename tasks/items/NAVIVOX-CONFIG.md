+++
schema = 1
id = "NAVIVOX-CONFIG"
title = "Retire Navivox config schema with tolerant parsing"
short = "Retire Navivox config"
status = "open"
goal = "PHASE-9"
section = "Next"
rank = 30
depends_on = ["NAVIVOX-CHANNEL"]
touches = ["internal/config/"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Retire the Navivox config schema. Remove config.NavivoxCfg, the config/channels/navivox package, normalization and VPN bind validation, and profile config materialization. Keep config loading tolerant: `[channels.navivox]` and `[navivox]` keys are accepted-and-ignored with one deprecation warning so existing config.toml files do not fail to load."
acceptance = ["No config.NavivoxCfg type or navivox package remains.", "A config.toml with [channels.navivox] loads, warns once, and keeps other settings.", "Profile config output contains no navivox keys."]
checks = ["go test ./internal/config/... -run 'Navivox|Legacy|Unknown' -count=1", "go test ./internal/config -count=1"]
exclusions = ["Legacy navivox keys would abort config load or reset unrelated settings.", "Profile config writers still emit navivox defaults."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
