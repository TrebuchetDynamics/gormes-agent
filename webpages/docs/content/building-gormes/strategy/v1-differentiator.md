# Gormes v1.0 Differentiator

**Ratified on**: 2026-05-07

## Differentiator (one paragraph, <50 words)

> Gormes runs the 30 most-used Hermes skills unchanged, in a single 30 MB Go
> binary, on Termux, Windows-without-Python, and locked-down corp Linux.
> No pip. No venv. No Docker daemon. One command.

## What Gormes v1.0 Will Be

A bounded, single-binary Go runtime that preserves Hermes-compatible agent
behavior for the skills operators actually use every day. The parity with
Hermes is the *receipt* — proof the autonomous-porting methodology works.
The product is the methodology.

## Curated 30-Skill List

The skills below represent the most-used Hermes skill families, selected by
operator judgement and in-repo evidence (skill file count, category breadth,
and Gormes implementation readiness). All 30 run unchanged from their
upstream Hermes SKILL.md definitions.

### Software Development (6)
1. **github** — PR creation, issue management, repo exploration
2. **git-master** — atomic commits, rebase, blame, bisect
3. **code-review** — structured PR review with plan verification
4. **shell-scripting** — bash and shell automation
5. **docker** — container build, run, compose management
6. **testing** — test generation, coverage analysis, debugging

### Productivity (5)
7. **note-taking** — structured notes, summaries, knowledge capture
8. **writing** — prose, documentation, technical writing
9. **diagramming** — Mermaid, Graphviz, architecture diagrams
10. **web-search** — research queries, fact-checking, documentation lookup
11. **todo-management** — task tracking, priority management

### AI & Agents (5)
12. **autonomous-agent** — multi-turn autonomous task execution
13. **subagent-driven-development** — parallel agent dispatch
14. **brainstorming** — creative ideation, design exploration
15. **debugging** — systematic bug investigation and fix
16. **prompt-engineering** — prompt design, refinement, testing

### Media & Creative (4)
17. **image-generation** — AI image creation and editing
18. **video** — video analysis, editing, clip generation
19. **audio** — transcription, TTS, voice processing
20. **creative** — design, copywriting, brand work

### Data & Research (4)
21. **data-analysis** — CSV, JSON, SQL, charting, statistics
22. **mlops** — model evaluation, deployment, monitoring
23. **research** — literature review, paper analysis, citation
24. **api-integration** — REST, GraphQL, SDK exploration

### Operations (3)
25. **devops** — CI/CD, infra-as-code, cloud management
26. **monitoring** — log analysis, alert triage, health checks
27. **system-administration** — server management, security audit

### Communication (3)
28. **email** — email drafting, inbox management, threading
29. **slack** — channel messaging, thread management, bot integration
30. **discord** — server management, bot responses, channel ops

## Explicit Exclusion List

Gormes v1.0 will **not** chase:

- **Full TUI parity beyond core** — the Bubble Tea TUI covers onboarding,
  doctor, slack, dashboard, and gateway status. Hermes' React/Ink TUI with
  session pickers, model pickers, FPS overlays, and themed components is out
  of scope for v1.0.

- **Dashboard / web app** — Hermes' web UI is not a v1.0 target.

- **Full i18n** — English-only for v1.0. Non-English channel adapters
  (WeChat, QQ, DingTalk, Feishu) are deferred.

- **Plugin ecosystem** — Hermes' plugin registry for memory providers
  (Mem0, Supermemory, Holographic) and platform adapters (IRC, Viber)
  is deferred beyond the core 30-skill runtime.

- **Voice/TTS beyond basic** — `gormes --oneshot` supports provider-backed
  turns. Full voice-mode state, TTS synthesis, and streaming transcription
  are post-v1.

- **Kanban multi-board and dashboard widgets** — kanban core board operations
  work; Hermes' full dashboard with tooltips, docs links, and multi-board
  management is deferred.

- **MCP server management** — MCP client login/auth works; full MCP serve,
  configure, and plugin lifecycle is deferred.

- **Cron scheduling beyond basic** — cron job management, webhook
  subscriptions, and multi-target delivery are deferred.

- **Every Hermes OAuth provider** — OpenAI Codex, Anthropic, and OpenAI API
  key auth work. Nous, MiniMax, Google Gemini CLI, Qwen, and Spotify OAuth
  are deferred.

- **Browser automation beyond basic** — `browser_navigate`, `browser_snapshot`,
  `browser_click`, `browser_type` work. Browserbase, Camofox, CDP supervisor,
  and `/browser connect` are post-v1.

- **Release signing and package-manager distribution** — tagged GitHub
  releases with SHA-256 checksums exist. Homebrew formula, apt/yum repos,
  and code signing are post-v1.

This exclusion list is the Pareto filter. Adding anything from this list to
v1.0 requires updating this document with the new ratified-on date and the
reason for inclusion.

## Cross-Reference

- `docs/content/building-gormes/strategy/success-plan.md` — full strategy
- `docs/content/building-gormes/architecture_plan/progress.json` — executable
  rows, Phase 8
- `README.md` — operator-facing messaging
- `webpages/landing/` — public site messaging
- `hermes-agent/skills/` — upstream skill definitions
