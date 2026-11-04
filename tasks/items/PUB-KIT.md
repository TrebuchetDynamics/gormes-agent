+++
schema = 1
id = "PUB-KIT"
title = "Agentic-porting-kit public repo scaffold"
short = "Porting kit repo scaffold"
status = "open"
goal = "PHASE-8"
section = "Needs decision"
rank = 30
depends_on = []
touches = ["(separate repo) README.md", "(separate repo) LICENSE", "(separate repo) schemas/progress.schema.json", "(separate repo) scripts/validate-example.sh", "(separate repo) skills/", "(separate repo) examples/python-greeter-to-go/", "README.md", "webpages/docs/content/building-gormes/strategy/success-plan.md", "webpages/docs/content/building-gormes/architecture_plan/progress.json"]
resources = []
kind = "implementation"
ownership = "owner"
scope = "Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout."
acceptance = ["Public repo exists with README.md, LICENSE, schemas/progress.schema.json, scripts/validate-example.sh, skills/, and examples/python-greeter-to-go/.", "README.md explains the kit independent of Gormes/Hermes and includes Codex plus Claude Code loading instructions.", "Each copied skill uses the porting-* name from the extraction spec and replaces hard-coded Gormes paths with target-repo variables.", "scripts/validate-example.sh validates the example progress file and runs the example tests without cloning Gormes.", "Gormes README.md and success-plan.md record the public repo URL after the repo is reachable."]
checks = ["cd ${AGENTIC_PORTING_KIT_REPO:-../agentic-porting-kit} && ./scripts/validate-example.sh", "go run ./cmd/progress validate", "go test ./webpages/docs -count=1"]
exclusions = ["No authenticated path exists to create or update the public TrebuchetDynamics repo.", "The builder plans to edit Gormes' repo-local skills in place instead of copied kit skills.", "The standalone example still requires cloning Gormes or running cmd/progress."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
