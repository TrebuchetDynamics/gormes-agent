# Gormes Documentation Website Upgrade Research

Date: 2026-05-01

Scope: research and proposal only. No Hugo content, layout, CSS, or runtime files were changed.

Local verification run:

```bash
go test ./docs -count=1
go run ./cmd/progress validate
go run ./cmd/gormes --help
go run ./cmd/gormes gateway --help
go run ./cmd/gormes config --help
go run ./cmd/gormes setup --help
go run ./cmd/gormes config path
go run ./cmd/gormes config env-path
go run ./cmd/gormes version
```

Observed outputs used in this report:

- `go test ./docs -count=1`: passed.
- `go run ./cmd/progress validate`: `progress: validated 7 phases`.
- `gormes version`: `gormes 0.2.0-scout`.
- `gormes config path`: `/home/xel/.gormes/config.toml`.
- `gormes config env-path`: `/home/xel/.gormes/.env`.

## Executive Recommendation

Make the docs site an operator-grade product manual, not a roadmap archive. The homepage should answer this in the first screen:

> Gormes is a Go-native Hermes-compatible agent runtime for local TUI use and persistent chat gateways.

The docs architecture should split stable user tasks from internal parity work:

- `getting-started`: install, first run, first useful provider-backed turn.
- `guides`: Telegram/gateway, web tools, browser/CDP, debugging, production operation.
- `reference`: CLI, config schema, env vars, providers, web backends, paths, logs.
- `architecture`: runtime flow, tool execution, providers, gateway, memory, Hermes parity.
- `development`: repo layout, tests, progress rows, parity workflow.
- `parity`: current Hermes/Honcho/GBrain contract status, with strict "implemented", "row-backed", "planned", and "unverified" labels.

Do not lead with "operational moat", binary-size claims, or internal phase language. These are useful later, but they weaken first-contact trust.

## Current Hugo Site Audit

Exact Hugo project path:

- `/home/xel/git/sages-openclaw/workspace-mineru/gormes-agent/docs`

Current Hugo config:

- `docs/hugo.toml`
- `baseURL = "https://docs.gormes.ai/"`
- `title = "Gormes Docs"`
- No theme is configured. The site uses custom Hugo layouts under `docs/layouts`.
- Goldmark `unsafe = true` is enabled.
- Chroma config exists, but the current code block render hook bypasses `transform.HighlightCodeBlock`.
- Home/section/page outputs are HTML only.

Current theme/layout/assets:

```text
docs/layouts/index.html
docs/layouts/_default/baseof.html
docs/layouts/_default/list.html
docs/layouts/_default/single.html
docs/layouts/_default/_markup/render-codeblock.html
docs/layouts/partials/{breadcrumbs,footer,prevnext,search,sidebar,toc,topbar}.html
docs/static/site.css
docs/static/site.js
docs/static/social-card.png
docs/static/favicon*
docs/static/img/docs/cli-layout.svg
docs/static/img/docs/session-recap.svg
docs/assets/gormes-tui-demo.gif
```

Current content structure, summarized:

```text
docs/content/_index.md
docs/content/why-gormes.md
docs/content/using-gormes/
docs/content/building-gormes/
docs/content/building-gormes/architecture_plan/
docs/content/building-gormes/builder-loop/
docs/content/building-gormes/core-systems/
docs/content/building-gormes/gateway-donor-map/
docs/content/building-gormes/goncho_honcho_memory/
docs/content/upstream-hermes/
docs/content/upstream-gbrain/
```

Strengths:

- Fast static Hugo site with custom design, Pagefind indexing in CI, copy buttons, sidebar, ToC, Open Graph metadata, favicons, and docs build tests.
- The docs already separate "Using Gormes", "Building Gormes", and upstream reference material.
- CI deploys docs to Cloudflare Pages and checks the homepage avoids stale curl-pipe and Homebrew install copy.
- The repository has rich architecture and progress evidence, including `progress.json`, upstream feature maps, and contract-readiness docs.

Problems:

- The first screen says "The Go operator shell for Hermes", which frames Gormes as a wrapper instead of a Go-native runtime.
- `docs/content/_index.md` says `~16.2 MB`, while `docs/content/using-gormes/install.md` says `~34 MB`. The docs must pick measured current evidence or label benchmark data by build profile/date.
- The homepage says install/run flow stays in README. That fails the docs-site job; install and first-run must be obvious inside docs.
- `docs/content/using-gormes/configuration.md` points to `$XDG_CONFIG_HOME/gormes/config.toml`, but the current binary reports `/home/xel/.gormes/config.toml` and source says `GORMES_HOME` wins, defaulting to `~/.gormes`.
- That same configuration page lists memory/log state under `~/.hermes`, while current config source says native paths are under `~/.gormes` for config, `.env`, sessions, memory, and logs.
- `docs/content/using-gormes/telegram-adapter.md` documents `gormes telegram`, which exists, but it does not explain the preferred multi-channel `gormes gateway` runtime and `gateway status/stop` operator commands.
- The Pagefind mount is duplicated: `topbar.html` has `id="search"` and `partials/search.html` also renders `id="search"`. Duplicate IDs can break search initialization and accessibility.
- The sidebar is hardcoded to only `using-gormes`, `building-gormes`, and `upstream-hermes`; `upstream-gbrain` exists but is not a primary nav group.
- The custom code block render hook outputs `.Inner | safeHTML` and does not call Hugo's highlighter. Copy buttons work, but syntax highlighting and code-block attributes are not being used as Hugo intends.
- The build uses Hugo `0.140.0` in docs deploy CI, while Go docs tests and Playwright use Hugo `0.160.1`. Pin one version.
- The docs are too internal too early. Many pages are valuable for maintainers, but a cold user sees roadmap/archive language before stable install/config/operator guidance.

Broken, stale, or missing areas:

- Missing native Gormes reference pages for CLI, config schema, environment variables, paths, logs, provider support, web backend support, gateway operations, browser/CDP setup, and memory behavior.
- Missing status taxonomy. `progress.json` rows can be validated without proving full product readiness, so public docs need "runtime-verified", "fixture-backed", "row-backed", "planned", and "unverified".
- Missing copy-pasteable "first useful run" that includes provider setup and explains no-provider offline mode.
- Missing debugging page that starts from actual operator symptoms: bad Telegram formatting, gateway stopped, stale binary, sessions DB lock, missing Chrome/CDP, provider auth failure, web tools unavailable.
- Missing "what costs money" page for providers and web tools.
- Missing screenshots or diagrams on the homepage and architecture pages.

First-screen critique:

- Current hook is short but not precise enough. It should state the product, the audience, and the first action.
- Current quickstart only builds from source. It does not show `doctor`, `config`, `setup/onboard`, `gateway status`, or provider setup.
- The three cards are useful, but they underserve operators: "configure, run, debug, persist" should be visible immediately.

## Reference Documentation Patterns

| Project | URL | Homepage hook | Navigation model | Getting-started flow | Visual/media strategy | Lessons for Gormes |
|---|---|---|---|---|---|---|
| LangChain docs | https://docs.langchain.com/ | Presents the platform and routes users by product/framework. | Product index plus search and GitHub/Ask AI links. | Multiple entry cards, not one linear path. | Product icons and concise capability cards. | Use cards for distinct audiences, but avoid mixing product marketing with operator truth. |
| LlamaIndex | https://developers.llamaindex.ai/python/framework/ | Developer documentation with clear framework/product tabs. | Deep sidebar: Getting Started, Learn, Use Cases, Component Guides, Integrations. | Installation, high-level concepts, starter tutorials including local LLM path. | Dense reference nav, examples, integration catalog. | Gormes needs separate concepts, guides, components, and integrations so the sidebar scales. |
| Microsoft AutoGen | https://microsoft.github.io/autogen/stable/ | "A framework for building AI agents and applications." | Product tabs: AgentChat, Core, Extensions, Studio, API Reference, versions. | Routes new users to no-code Studio or Python AgentChat depending on intent. | Copyable code on homepage. | Split Gormes paths by persona: TUI user, Telegram operator, contributor, maintainer. |
| Dagger | https://docs.dagger.io/ | "What is Dagger?" then why, features, runs-anywhere promise. | Introduction, Getting Started, Features, Building, Cookbook, Reference. | Install, core concepts, quickstarts. | Text-first, strong operator properties: local-first, repeatable, observable. | Copy its "what/why/features" clarity and operator language. |
| Temporal | https://docs.temporal.io/ | Reliability promise for applications that resume after failures. | Quickstarts, Evaluate, Develop, Deploy, CLI, References, Troubleshooting, Best practices. | Starts with local quickstart and deploy path. | Strong CTA cards and community/trust links. | Gormes should have first-class "Deploy/run gateway" and "Troubleshooting" nav. |
| Kubernetes | https://kubernetes.io/docs/home/ | Broad project docs with version and localization controls. | Getting started, Concepts, Tasks, Tutorials, Reference, Contribute. | Separates learning and production environments. | Huge docs tree, versioned docs, localization. | Gormes should separate local evaluation from fleet/production operation and version docs once releases stabilize. |
| Terraform | https://developer.hashicorp.com/terraform/docs | Defines Terraform as IaC, then groups docs by adoption jobs. | Install, Tutorials, Documentation, Registry, Sandbox, product references. | Install and tutorial library are always visible. | Structured index pages and registry/reference split. | Gormes needs a real reference section separate from guides, especially config/CLI/providers. |
| Astral uv | https://docs.astral.sh/uv/ | "An extremely fast Python package and project manager, written in Rust." | Introduction, Getting started, Guides, Concepts, Reference, Policies. | Install command, first steps, then practical workflows. | Benchmarks, copyable terminal blocks, concise feature bullets. | Gormes should copy this concise first screen: one-line identity, current install, first commands, then docs map. |

Hugo-specific sources used:

- Hugo config settings: https://gohugo.io/configuration/all/
- Hugo menus: https://gohugo.io/content-management/menus/
- Hugo code block render hooks: https://gohugo.io/render-hooks/code-blocks/
- Hugo embedded Open Graph template: https://gohugo.io/templates/embedded/

## Proposed Documentation Positioning

Primary audiences:

- New user evaluating whether Gormes is worth trying.
- Developer installing and running Gormes locally.
- Operator running persistent Telegram/Discord/Slack gateway processes.
- Contributor extending runtime behavior while preserving Hermes parity.
- Maintainer auditing parity, progress rows, and release readiness.

Homepage one-line pitch:

> Gormes is a Go-native Hermes-compatible agent runtime for local TUI work, provider-backed turns, and persistent chat gateways.

Alternative homepage hooks:

1. "Run a Hermes-compatible agent runtime from one Go binary."
2. "Local TUI, provider turns, tools, memory, and chat gateways in a Go-native runtime."
3. "A Go rewrite of Hermes built for operators who need installable, inspectable, persistent agents."

Core promise of the docs site:

> A developer can install Gormes, run a first useful turn, configure provider and gateway credentials, debug failures, and understand what is stable versus still being ported.

First five minutes user journey:

1. Land on homepage and understand Gormes in one sentence.
2. Copy source-build or installer command.
3. Run `gormes doctor --offline`.
4. Run `gormes setup --non-interactive` or `gormes config set ...`.
5. Run `gormes --oneshot "hello from Gormes"` or `gormes --offline`.
6. If operating chat, go to `Run the gateway` and run `gormes gateway status`.
7. If blocked, use troubleshooting by symptom.

## Proposed Site Map

Recommended Hugo content structure:

```text
docs/content/
  _index.md
  getting-started/
    _index.md
    installation.md
    first-run.md
    configuration.md
    troubleshooting.md
  concepts/
    _index.md
    runtime-model.md
    providers.md
    tools-and-web.md
    transports.md
    sessions-and-memory.md
    hermes-parity.md
  guides/
    _index.md
    telegram-bot.md
    gateway-operations.md
    provider-setup.md
    web-tools.md
    browser-cdp.md
    debugging.md
    production-ops.md
  reference/
    _index.md
    cli.md
    config.md
    environment.md
    providers.md
    web-backends.md
    transports.md
    paths-and-logs.md
    progress-status.md
  architecture/
    _index.md
    overview.md
    runtime-flow.md
    provider-routing.md
    tool-execution.md
    gateway.md
    memory.md
  development/
    _index.md
    repo-layout.md
    testing.md
    adding-a-feature.md
    parity-workflow.md
  parity/
    _index.md
    hermes.md
    honcho-goncho.md
    command-surface.md
    gateway-ux.md
```

Migration note:

- Keep existing URLs with aliases where possible.
- Move or relabel `upstream-hermes` and `upstream-gbrain` as reference/archive material, not primary first-run docs.
- Keep `building-gormes` but lower it below user/operator paths.

## Homepage Draft

Proposed `docs/content/_index.md`:

```markdown
---
title: "Gormes Documentation"
description: "Install, configure, operate, and extend the Go-native Hermes-compatible Gormes runtime."
weight: 0
slug: "/"
---

# Gormes

Gormes is a Go-native Hermes-compatible agent runtime for local TUI work, provider-backed turns, tools, memory, and persistent chat gateways.

It is built for developers and operators who want an inspectable Go binary instead of a Python runtime stack, while preserving Hermes behavior where that behavior is the contract.

## Try it locally

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes doctor --offline
./bin/gormes --offline
```

Current source evidence: `go.mod` declares Go 1.25.0, and the installer currently requires Go 1.25+. TODO: verify release binary availability before documenting a no-Go install path as stable.

## First useful paths

| Goal | Start here |
|---|---|
| Install and smoke-test Gormes | [Getting Started](getting-started/) |
| Configure model/provider credentials | [Provider setup](guides/provider-setup/) |
| Run a persistent Telegram/Discord/Slack gateway | [Gateway operations](guides/gateway-operations/) |
| Debug a failed run | [Troubleshooting](getting-started/troubleshooting/) |
| Understand the runtime | [Architecture overview](architecture/) |
| Check Hermes parity status | [Parity status](parity/) |

## What works today

- Local TUI and one-shot runs.
- Offline diagnostics with `gormes doctor --offline`.
- Native config under `GORMES_HOME`, defaulting to `~/.gormes`.
- Provider registry with implemented, owned, row-backed, and planned statuses.
- Gateway runtime commands for configured messaging channels.
- Page-backed development progress through `progress.json`.

TODO: replace this list with generated evidence from runtime smoke tests and `progress.json` status labels before publishing.

## What is still being ported

Gormes is not claiming byte-for-byte Hermes parity yet. Some features are runtime-ready, some are fixture-backed, some are row-backed, and some are planned. The docs label those states explicitly so users can tell product behavior from roadmap evidence.
```

## Key Page Drafts

### Getting Started Overview

```markdown
---
title: "Getting Started"
weight: 10
---

# Getting Started

This path gets you from source checkout to a local Gormes smoke test, then to a provider-backed turn when credentials are available.

1. [Install](installation/) builds or installs the binary.
2. [First run](first-run/) verifies local runtime behavior.
3. [Configuration](configuration/) explains `GORMES_HOME`, `config.toml`, `.env`, and provider credentials.
4. [Troubleshooting](troubleshooting/) maps common symptoms to commands.

If you only want a no-network smoke test:

```bash
./bin/gormes doctor --offline
./bin/gormes --offline
```
```

### Installation

```markdown
---
title: "Installation"
weight: 20
---

# Installation

## Source build

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes version
./bin/gormes doctor --offline
```

The current `go.mod` declares Go 1.25.0 and the installer checks for Go 1.25+. Use the version required by the branch you are building.

## Inspectable installer

```bash
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh
less install.sh
sh install.sh
```

TODO: document signed release archives after checksums/signatures and release URLs are verified.
```

### First Run

```markdown
---
title: "First Run"
weight: 30
---

# First Run

Verify local runtime checks without contacting a model provider:

```bash
gormes doctor --offline
```

Run the local TUI without provider calls:

```bash
gormes --offline
```

Run a one-shot provider-backed turn after configuring a provider:

```bash
gormes --oneshot "hello from Gormes"
```

Check available commands:

```bash
gormes --help
gormes config --help
gormes gateway --help
```
```

### Configuration

```markdown
---
title: "Configuration"
weight: 40
---

# Configuration

Gormes loads native configuration from `GORMES_HOME`, defaulting to `~/.gormes`.

```bash
gormes config path
gormes config env-path
gormes config show
```

Current default paths:

| Item | Default |
|---|---|
| Config | `~/.gormes/config.toml` |
| Secrets dotenv | `~/.gormes/.env` |
| Sessions DB | `~/.gormes/sessions.db` |
| Memory DB | `~/.gormes/memory.db` |
| Log | `~/.gormes/gormes.log` |

Use `gormes config set` for values. Secret-like keys are written to `.env` instead of `config.toml`.
```

### Troubleshooting

```markdown
---
title: "Troubleshooting"
weight: 50
---

# Troubleshooting

Start with:

```bash
gormes version
gormes doctor --offline
gormes config check
gormes gateway status
```

| Symptom | Check | Likely fix |
|---|---|---|
| The command uses an old binary | `which -a gormes` | Rebuild and ensure PATH points at the intended binary. |
| Provider turn fails | `gormes config show` | Set provider/model/API credentials; verify secrets are in `.env`. |
| Gateway is silent | `gormes gateway status` | Check configured channels and runtime status. |
| Telegram formatting is wrong | Capture exact bot output | File a gateway UX parity bug with transcript evidence. |
| Browser tools fail | `gormes doctor --offline` | Install Chrome/Chromium, start CDP, or install go-browser-harness. |
| Web extract returns thin content | Try browser/CDP path | Use browser tools when API extract backends cannot read a dynamic site. |
```

### Architecture Overview

```markdown
---
title: "Architecture Overview"
weight: 10
---

# Architecture Overview

Gormes is organized around one Go runtime boundary:

```text
CLI/TUI/Gateway
  -> config and auth
  -> provider client
  -> agent loop
  -> tool registry
  -> session and memory stores
  -> renderers for terminal and chat channels
```

The parity rule is: Hermes defines user-visible behavior when it is current and intentional; Gormes implements that behavior in Go unless a Gormes-native choice is explicitly documented.

Key packages:

| Package | Role |
|---|---|
| `cmd/gormes` | CLI, TUI startup, gateway commands, setup/config commands. |
| `internal/config` | Native config, `GORMES_HOME`, dotenv, schema checks. |
| `internal/hermes` | Provider/runtime compatibility surfaces. |
| `internal/gateway` | Shared channel command and rendering runtime. |
| `internal/tools` | Tool registry and tool implementations. |
| `internal/goncho` | Honcho-compatible memory facade. |
```

### CLI Reference Skeleton

```markdown
---
title: "CLI Reference"
weight: 10
---

# CLI Reference

Generated from `gormes --help` on 2026-05-01. TODO: replace with generated docs from Cobra.

## Top-level commands

| Command | Purpose |
|---|---|
| `gormes` | Start TUI or run with flags such as `--offline` and `--oneshot`. |
| `gormes doctor` | Verify runtime readiness. |
| `gormes config` | Inspect or update native config. |
| `gormes setup` / `gormes onboard` | Configure runtime sections. Full wizard is not complete in this slice. |
| `gormes gateway` | Run or inspect configured messaging channels. |
| `gormes telegram` | Telegram bot adapter path. |
| `gormes model` | Select model/provider. |
| `gormes auth` / `gormes logout` | Manage provider credentials. |
| `gormes mcp` | Manage Hermes-compatible MCP servers. |
| `gormes goncho` | Inspect local Goncho diagnostics. |
| `gormes memory` | Inspect persisted memory/extractor state. |
| `gormes session` | Inspect/export sessions. |
| `gormes dashboard` | Start local web dashboard. |
| `gormes migrate` | Migrate state from upstream agents. |

## Gateway commands

| Command | Status |
|---|---|
| `gormes gateway status` | Available. |
| `gormes gateway stop` | Available. |
| `gormes gateway start/restart/install/uninstall` | Unavailable in CLI; use the service restart helper. |
```

## Hugo Implementation Plan

Safe step-by-step builder plan:

1. Keep the custom Hugo theme. It is already fast and tested. Do not swap themes unless the custom layout becomes a maintenance problem.
2. Pin Hugo version consistently across `.github/workflows/deploy-gormes-docs.yml`, `docs/build_test.go`, and `docs/www-tests/playwright.config.mjs`.
3. Fix duplicate search IDs by rendering one `#search` element or using separate IDs like `#topbar-search` and `#sidebar-search`.
4. Replace hardcoded sidebar section list with either Hugo menus or a data file. Hugo supports menus in front matter/config and nested rendering.
5. Update `docs/content/_index.md` and `docs/layouts/index.html` together so the homepage promise and rendered first screen match.
6. Add the proposed `getting-started`, `guides`, `reference`, `architecture`, `development`, and `parity` sections with only pages that have evidence.
7. Move stale or upstream-only content below a clearly labeled "Upstream Reference" or "Parity Archive" section.
8. Fix current stale docs:
   - binary size claims;
   - config paths;
   - `~/.hermes` state references;
   - Telegram vs gateway runtime wording;
   - provider/web/backend status claims.
9. Replace the code block render hook with `transform.HighlightCodeBlock` plus the existing copy button. Add Mermaid support with a language-specific render hook if diagrams are added.
10. Add link checking. Suggested command:

```bash
lychee docs/content docs/layouts --no-progress
```

11. Add accessibility checks to Playwright. Suggested package:

```bash
cd docs/www-tests
npm install --save-dev @axe-core/playwright
npm run test:e2e
```

12. Add generated references where possible:
   - `gormes --help` and subcommands to `reference/cli.md`;
   - `internal/config.Config` tags to `reference/config.md`;
   - provider manifest to `reference/providers.md`;
   - web backend resolver to `reference/web-backends.md`;
   - `internal/config` path helpers to `reference/paths-and-logs.md`.

Verification commands:

```bash
go test ./docs -count=1
(cd docs/www-tests && npm run test:e2e)
go run ./cmd/progress validate
git diff --check
```

Commit plan:

1. Commit 1: docs IA and stale-claim fixes.
2. Commit 2: Hugo layout/search/codeblock improvements.
3. Commit 3: generated reference pages and tests.
4. Commit 4: visual assets and diagrams.

## Visual Asset Plan

Recommended assets:

| Asset | Purpose | Path | Alt text |
|---|---|---|---|
| Homepage runtime diagram | Explain what Gormes is in 5 seconds. | `docs/static/img/docs/gormes-runtime-overview.png` | "Gormes runtime connecting CLI, TUI, gateways, providers, tools, sessions, and memory." |
| Architecture flow diagram | Show request flow through runtime. | `docs/static/img/docs/runtime-flow.svg` | "Request flow from user input through Gormes config, provider client, tool registry, session store, and renderer." |
| Provider/config diagram | Explain config and credential storage. | `docs/static/img/docs/provider-config-flow.svg` | "Gormes configuration and dotenv credentials feeding provider selection." |
| Terminal demo GIF | Show build, doctor, offline run. | `docs/assets/gormes-first-run.gif` | "Terminal recording of building Gormes, running doctor, and opening offline mode." |
| Gateway status screenshot | Operator trust proof. | `docs/static/img/docs/gateway-status.png` | "Gormes gateway status output showing configured channel readiness." |
| Telegram screenshot | Show real channel UX after formatting fixes. | `docs/static/img/docs/telegram-help.png` | "Telegram bot help response rendered without Markdown artifacts." |

Capture commands:

```bash
# Terminal GIF with vhs, if installed
cat > /tmp/gormes-first-run.tape <<'EOF'
Output docs/assets/gormes-first-run.gif
Set Shell "bash"
Type "make build"
Enter
Sleep 1s
Type "./bin/gormes doctor --offline"
Enter
Sleep 1s
Type "./bin/gormes --offline"
Enter
Sleep 2s
EOF
vhs /tmp/gormes-first-run.tape
```

Generated hero prompt:

```text
Create a clean documentation hero graphic for a Go-native agent runtime named Gormes. Show a central terminal window connected to provider APIs, tools, memory, and chat gateways. Style: technical, crisp, dark UI, restrained gold accent, no mascots, no fantasy elements, no fake text, no logos except the word Gormes. 16:9, high contrast, accessible composition.
```

Generated architecture prompt:

```text
Create a simple isometric technical diagram for Gormes docs: CLI/TUI and Telegram gateway enter a Go runtime, then flow to config/auth, provider client, tool registry, session store, memory store, and renderer outputs. Use labeled blocks, straight connectors, dark background, high readability, restrained colors, no decorative blobs.
```

Prefer SVG/Mermaid for architecture diagrams when labels need to stay exact. Use generated bitmap art only for the homepage hero/social image.

## Hugo-Specific Review

- Theme: keep the custom theme for now; it already has tests and product identity.
- Homepage layout: update copy and cards; do not rely only on Markdown because `layouts/index.html` currently controls the first screen.
- Sidebar/navigation: replace hardcoded sections with Hugo menus or front matter weights. Hugo menus support config/front-matter entries and nesting.
- Search: Pagefind is already generated in deploy CI; fix duplicate IDs before expanding search.
- Code blocks: current copy buttons work, but switch render hook to Hugo's `transform.HighlightCodeBlock` so Chroma options and code attributes work.
- Mermaid/diagrams: add `render-codeblock-mermaid.html` only when diagrams land.
- SEO/Open Graph: current manual metadata is adequate. Consider Hugo's embedded Open Graph partial after adding per-page images/descriptions.
- Static assets: keep `docs/static/img/docs/` for screenshots/SVGs and `docs/assets/` for source/processed assets.
- Accessibility: add unique IDs, alt text, reduced-motion handling for GIFs, keyboard-visible focus, and Playwright axe checks.
- Mobile readability: grouped sections and shorter command lines are more important than more cards.
- Build/deploy: align Hugo versions and add link checking.

## Risks and Open Questions

- Is `gormes setup/onboard` intended to be documented as a real wizard now, or as a minimal compatibility command until the full wizard ships?
- Which channel runtime should docs recommend first: `gormes telegram` for one adapter, or `gormes gateway` for all configured channels?
- What is the current release story: source build only, installer, or precompiled binaries? Docs should not imply stable signed releases until verified.
- What binary size should public docs claim? Current docs contain both `~16.2 MB` and `~34 MB`.
- Which providers are truly runtime-implemented versus row-backed? The provider manifest has distinct statuses; docs should expose them instead of flattening to "supported".
- Memory/Goncho docs may overstate readiness. The user has observed poor short-term/long-term memory behavior, so docs need runtime limitations and known gaps.
- Web tools need a cost/routing page: API-key providers may cost money; DuckDuckGo fallback is free but limited; CDP/browser fallback needs Chrome/go-browser-harness.
- Chrome/CDP docs need OS-specific install/detect guidance before asking operators to use browser tools.
- `progress.json` validation is not product verification. Public docs must distinguish schema validation, fixture-backed tests, and live operator success.
- Current dirty worktree includes unrelated modified files. Do not mix this docs implementation with runtime parity edits.

## Sources

Local repository evidence:

- `docs/hugo.toml`
- `docs/layouts/index.html`
- `docs/layouts/_default/baseof.html`
- `docs/layouts/_default/_markup/render-codeblock.html`
- `docs/layouts/partials/topbar.html`
- `docs/layouts/partials/search.html`
- `docs/layouts/partials/sidebar.html`
- `docs/content/_index.md`
- `docs/content/using-gormes/quickstart.md`
- `docs/content/using-gormes/install.md`
- `docs/content/using-gormes/configuration.md`
- `docs/content/using-gormes/telegram-adapter.md`
- `.github/workflows/deploy-gormes-docs.yml`
- `docs/build_test.go`
- `docs/www-tests/playwright.config.mjs`
- `internal/config/config.go`
- `internal/hermes/provider_registry_manifest.go`

External documentation references:

- LangChain docs: https://docs.langchain.com/
- LlamaIndex docs: https://developers.llamaindex.ai/python/framework/
- AutoGen docs: https://microsoft.github.io/autogen/stable/
- Dagger docs: https://docs.dagger.io/
- Temporal docs: https://docs.temporal.io/
- Kubernetes docs: https://kubernetes.io/docs/home/
- Terraform docs: https://developer.hashicorp.com/terraform/docs
- uv docs: https://docs.astral.sh/uv/
- Hugo configuration: https://gohugo.io/configuration/all/
- Hugo menus: https://gohugo.io/content-management/menus/
- Hugo code block render hooks: https://gohugo.io/render-hooks/code-blocks/
- Hugo embedded templates/Open Graph: https://gohugo.io/templates/embedded/
