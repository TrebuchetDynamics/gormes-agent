+++
schema = 1
id = "PUB-SOCIAL"
title = "TD social presence connected to blog feed"
short = "Social presence to blog"
status = "open"
goal = "PHASE-8"
section = "Needs decision"
rank = 10
depends_on = []
touches = ["scripts/blog-feed-social/", "webpages/blog/", "webpages/docs/content/building-gormes/strategy/success-plan.md", "webpages/docs/content/building-gormes/strategy/success-plan.md", "webpages/docs/content/building-gormes/architecture_plan/progress.json"]
resources = []
kind = "implementation"
ownership = "owner"
scope = "After the TrebuchetDynamics engineering blog feed is live and the operator selects a social account, provide a reproducible feed-to-social automation with a dry-run preview, idempotency guard, and documented publish method so blog posts can be announced without click-only console configuration."
acceptance = ["A dry-run against a repo-local or freshly built blog feed emits the exact social post text, canonical blog URL, and idempotency key without network publication.", "With operator-provided credentials outside the repository, a test post on the blog produces one corresponding public social post within 1 hour.", "The automation is reproducible from a script and documented inputs, not click-configured in a provider console.", "success-plan.md records the social account URL, automation method, and secret-handling boundary without storing tokens."]
checks = ["(cd webpages/blog && npm run test)", "node --test scripts/blog-feed-social/publish.test.mjs", "node scripts/blog-feed-social/publish.mjs --dry-run --feed webpages/blog/dist/feed.xml --out /tmp/gormes-social-post.json", "go run ./cmd/progress validate", "git diff --check"]
exclusions = ["Posts are announced manually each time.", "Automation depends on a paid-per-post service when free options exist.", "The implementation requires storing raw social tokens in the repository or progress backlog.", "The builder cannot run a dry-run preview without publishing to the live social account."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
