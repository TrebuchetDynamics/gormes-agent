+++
schema = 1
id = "NAVIVOX-DOCS"
title = "Remove Navivox docs, landing mirrors, install residue, and CI jobs"
short = "Remove Navivox surfaces"
status = "open"
goal = "PHASE-9"
section = "Next"
rank = 50
depends_on = ["NAVIVOX-ADAPTER"]
touches = ["webpages/docs/", "webpages/landing/", ".github/workflows/", "Makefile", "README.md", "install.sh", "webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md", "webpages/docs/content/building-gormes/architecture_plan/runtime/profile-config-v2.md", "webpages/docs/content/building-gormes/architecture_plan/profile-config-v2.md"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Remove the remaining Navivox surfaces: delete the Navivox docs pages and test file, drop Navivox rows and mentions from README, llms.txt, landing data and tests, remove the mirrored install.sh residue, delete the dedicated navivox CI job plus its -race package reference in ci.yml, release.yml, and release-prep.yml, and drop the Makefile Navivox test pattern. Also retire the remaining planning docs: mark the Hermes/Honcho feature-map row as retired and remove the leftover navivox config examples from the profile-config v2 pages."
acceptance = ["No Navivox docs page, landing claim, or CI job remains.", "Docs and landing builds pass without Navivox references.", "The Makefile no longer names a Navivox test pattern.", "The feature map no longer presents Navivox as a current first-party channel, and profile-config v2 shows no navivox keys."]
checks = ["go test ./webpages/docs -count=1", "cd webpages/landing/legacy/go-renderer && go test ./... -count=1", "cd webpages/landing && npm run test:data && npm run test:sync-assets && npm run build"]
exclusions = ["Docs or landing tests still assert the deprecated Navivox copy.", "CI removal would drop coverage for a still-present package."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
