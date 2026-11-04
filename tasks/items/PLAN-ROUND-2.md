+++
schema = 1
id = "PLAN-ROUND-2"
title = "Plan round 2: split upstream delta into builder rows"
short = "Split upstream delta rows"
status = "open"
goal = "PHASE-4"
section = "Next"
rank = 20
depends_on = ["PARITY-GATE"]
touches = ["webpages/docs/parity-evidence/", "webpages/docs/content/building-gormes/architecture_plan/"]
resources = []
kind = "plan"
ownership = "coordinator"
scope = "After the rolling gate, baseline advance, and delta report land, reconcile HERMES-BEHAVIOR-ATOMS against the new upstream SHA and split every new feature-bearing family into builder-ready progress rows. This is a planning pass: no runtime implementation. Stop only when every new feature family has an atom, an owner, an acceptance test, and a row or a justified exclusion."
acceptance = ["TestUpstreamCoverageLedgerMatchesSourceClasses passes for the new upstream SHA when the source checkouts are available.", "TestCompletionPlanCurrentFinishLedgerMatchesProgress passes after the new rows are saved.", "Every new feature-bearing family in the delta report maps to a saved row or a source-backed exclusion."]
checks = ["go run ./cmd/progress validate", "go test ./webpages/docs -run 'TestUpstreamCoverage|TestCompletionPlanCurrentFinishLedgerMatchesProgress' -count=1", "git diff --check"]
exclusions = ["Rows are created from unclassified or stale evidence.", "Runtime code, tests, or fixtures are edited in this planning pass.", "The round claims complete parity while any new atom is missing or partial."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
