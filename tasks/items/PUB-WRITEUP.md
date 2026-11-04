+++
schema = 1
id = "PUB-WRITEUP"
title = "Engineering writeup #1: autonomous Hermes-porting loop"
short = "Engineering writeup one"
status = "open"
goal = "PHASE-8"
section = "Needs decision"
rank = 20
depends_on = []
touches = ["webpages/blog/ (or external blog repo path)", "webpages/docs/content/building-gormes/strategy/writeups/", "webpages/docs/content/building-gormes/architecture_plan/progress.json"]
resources = []
kind = "implementation"
ownership = "owner"
scope = "A 1500–2500 word engineering writeup published on the TD blog explaining how the autonomous loop ships 1–2 Hermes-parity features per hour with TDD discipline. Includes: an architecture diagram of the loop, a real diff walkthrough of 3+ recent loop-shipped commits (e.g., Telegram thread fallback, browser cleanup, skill validator), the cost-per-feature numbers, and the validation-gate flow. Writes for a reader who has not seen Gormes before."
acceptance = ["Post is published at a public URL with the TD blog feed; a stranger can read it without auth.", "Post contains a loop architecture diagram (image or ASCII).", "Post walks through 3+ real diff examples with PR/commit links.", "Post includes real cost-per-feature numbers from 8.F telemetry.", "Post has been submitted to HN/Lobsters at least once (success metric is the submission, not front-page placement)."]
checks = ["(cd webpages/blog && npm run test)", "go run ./cmd/progress validate", "git diff --check"]
exclusions = ["Cost numbers are estimated rather than measured.", "The diff walkthrough cherry-picks a single perfect commit; readers see only good outcomes and not honest tradeoffs.", "The post is published without operator review of voice and tone."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
