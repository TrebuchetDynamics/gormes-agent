+++
schema = 1
id = "NAVIVOX-CHANNEL"
title = "Retire Navivox channel registration and setup plan"
short = "Retire Navivox channel"
status = "open"
goal = "PHASE-9"
section = "Next"
rank = 20
depends_on = ["NAVIVOX-CLI"]
touches = ["internal/app/gateway/", "internal/gateway/channelsetup/", "internal/app/gormescmd/"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Remove Navivox from gateway channel registration and setup planning: drop the Navivox factory from ChannelFactories, the registration branch and logs, the channelsetup entry, ProfileSetupChannelIDs membership, and gateway allowed-user synthesis. Gateway startup must not fail when a profile still lists navivox config."
acceptance = ["No ChannelFactories.Navivox field or registration branch remains.", "ProfileSetupChannelIDs excludes navivox.", "A config with [channels.navivox] enabled starts and logs one deprecation warning."]
checks = ["go test ./internal/app/gateway -count=1", "go test ./internal/gateway/channelsetup -count=1"]
exclusions = ["Removing registration would make an existing paired install fail to start instead of degrading.", "Setup guidance still requires a Navivox entry for stable channel IDs."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
