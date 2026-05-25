---
title: "Hermes Integrations Claim Audit"
weight: 24
---

# Hermes Integrations Claim Audit

This audit classifies the sanitized 2026-05-24 Reddit/WebAfterAI claim set
(`12 Hermes Integrations That Actually Matter`) against checked-in Hermes and
Gormes sources. It is not an implementation batch and it does not treat social
copy as a source of truth.

Rules used here:

- Direct integration claims require current Hermes source: bundled skill,
  bundled plugin/backend, gateway/platform adapter, or model-visible tool.
- Some workflows are real without being direct integrations: generic
  `web_search`, `web_extract`, `web_crawl`, Browser Use/CDP, MCP, webhooks, or
  skill workflows can satisfy a user task without proving a named native plugin.
- Unsupported claims stay excluded until exact current Hermes source or an
  external plugin repository URL appears. Do not create Gormes runtime rows for
  unsupported social-post claims.
- Evidence below uses repository files only; no live `~/.hermes`, credentials,
  browser sessions, private token stores, memory databases, or external accounts
  were read.

## Summary Classification

| # | Claimed integration / workflow | Source-backed class | Hermes evidence | Gormes action |
|---:|---|---|---|---|
| 1 | Google Workspace / Gmail / Calendar / Drive / Docs / Sheets | First-party bundled skill | `hermes-agent/skills/productivity/google-workspace/SKILL.md` declares Gmail, Calendar, Drive, Contacts, Sheets, Docs, OAuth files, setup script, and `scripts/google_api.py`. | Keep under skills parity and Google OAuth/provider rows; do not advertise it as a core Go tool until Gormes has an equivalent source-backed skill/runtime surface. |
| 2 | Obsidian | First-party bundled skill | `hermes-agent/skills/note-taking/obsidian/SKILL.md` defines filesystem-first vault read/search/create/edit workflows using file tools and `OBSIDIAN_VAULT_PATH`. | Source-backed skill workflow. Gormes can map it through skill catalog/file-tool parity, not a separate Obsidian API plugin. |
| 3 | Firecrawl | Bundled plugin/backend | `hermes-agent/plugins/web/firecrawl/plugin.yaml` provides `firecrawl`; `provider.py` implements direct API, self-hosted URL, and Nous tool-gateway routing for search/extract/crawl. `tools/web_tools.py` re-exports Firecrawl helpers for compatibility. | Gormes browser/web rows already own Firecrawl fallback and web backend matrix behavior; keep claims specific to web search/extract/crawl. |
| 4 | Generic web scraping / competitor or Reddit-style research | Indirect web/browser workflow, not a named Reddit integration | `hermes-agent/tools/web_tools.py` exposes backend-selected `web_search_tool`, `web_extract_tool`, and `web_crawl_tool` across Firecrawl/Tavily/Exa/Parallel/SearXNG/Brave/DDGS. Browser automation is mapped separately in the feature map. | Valid as an achievable workflow through web/browser tools. Do not claim `hermes plugins install reddit` or a native Reddit plugin. |
| 5 | GitHub | First-party bundled skill family | `hermes-agent/skills/github/DESCRIPTION.md` plus `skills/github/{github-auth,github-repo-management,github-issues,github-pr-workflow,github-code-review,codebase-inspection}/SKILL.md` cover gh/git workflows for repos, issues, PRs, reviews, and CI. | Source-backed skill family. Gormes should keep GitHub parity row-backed via skills/tooling, not convert this audit into a broad GitHub implementation batch. |
| 6 | YouTube | First-party bundled skill | `hermes-agent/skills/media/youtube-content/SKILL.md` uses `youtube-transcript-api` helper scripts to fetch transcripts and transform them into summaries, chapters, threads, blogs, or quotes. | Source-backed media skill. Preserve as a skill/workflow parity target. |
| 7 | Discord | Gateway/platform plus model-visible admin tool | `hermes-agent/gateway/platforms/discord.py` implements Discord message/thread/voice adapter behavior; `hermes-agent/tools/discord_tool.py` exposes server introspection/admin actions gated by intents and config. | Source-backed channel/tool. Gormes Discord channel/tool rows should remain the owner, not docs marketing copy. |
| 8 | Twilio / SMS / phone calls | Optional skill plus SMS gateway/platform | `hermes-agent/optional-skills/productivity/telephony/SKILL.md` covers Twilio number/SMS/MMS/direct calls and Bland.ai/Vapi outbound calls; `hermes-agent/gateway/platforms/sms.py` is the Twilio SMS adapter. | Source-backed optional skill/gateway. Gormes SMS rows cover gateway parity; telephony call features need separate optional-skill/tool rows if pursued. |
| 9 | Stripe | Unsupported as a direct Hermes plugin/tool; indirect via webhooks/MCP/docs examples | Repository search found Stripe in webhook docs, Notion skill docs, MCP examples, redaction patterns, and generic automation examples, but no first-party Stripe API plugin/tool source. | Do not claim native Stripe integration. Any Stripe work must be framed as webhook/MCP/user-configured API workflow or get a new source-backed row. |
| 10 | InsForge | Unsupported/excluded | Repository search for `insforge` in Hermes source returned no first-party skill, plugin, tool, gateway, or docs source. | Exclude until exact Hermes source or an external plugin repository URL appears. |
| 11 | Graphiti / Zep memory | Unsupported as a direct Hermes memory plugin/tool; Graphiti appears only as profile env metadata | `hermes-agent/hermes_cli/profile_distribution.py` and website profile docs mention `GRAPHITI_MCP_URL`; repository search found no Zep integration source and no direct Graphiti plugin/tool implementation. | Do not treat Graphiti/Zep as native Hermes memory parity. If Gormes wants it, route through MCP/profile docs or a separate discovery row. |
| 12 | Fireflies meeting notes | Unsupported as a Fireflies integration; indirect Google Meet/transcription alternative exists | Repository search found Fireflies as a comparison word in pixel-art assets and in `website/docs/user-guide/features/built-in-plugins.md`, where Google Meet is described as useful when one would otherwise need Fireflies/Otter/Grain. No Fireflies API plugin/tool source was found. | Do not claim Fireflies support. Mention only the source-backed Google Meet/meeting-transcription plugin if that surface is being mapped. |

## Source Notes

### Plugin install is not a short-name integration registry

`hermes-agent/hermes_cli/plugins_cmd.py` installs plugins by Git URL or
`owner/repo` shorthand into `~/.hermes/plugins/`. The installer validates a
local plugin directory/name and reads a `plugin.yaml`, but the checked-in code
is not evidence that every marketing-name token is a bundled plugin. Treat
`hermes plugins install reddit|stripe|insforge|graphiti|fireflies` as unproven
unless an actual plugin repository URL or bundled source exists.

`hermes-agent/hermes_cli/plugins.py` discovers bundled plugins, user plugins,
project plugins, and pip entry-point plugins. It requires a plugin manifest and
register hook. Discovery semantics prove extensibility, not that a named
integration is first-party.

### Web/browser/MCP workflows are separate from direct integrations

Hermes can perform many integration-like tasks through generic tools:

- `tools/web_tools.py` handles search, extraction, and crawling through web
  backends such as Firecrawl and Tavily.
- Browser automation/CDP has its own feature-map lane and tool names.
- MCP allows user-configured servers, including examples that may interact with
  third-party APIs, but user MCP configuration is not a bundled Hermes plugin.
- Webhooks can receive third-party events, but a webhook event from Stripe does
  not imply a native Stripe API client.

Gormes docs and README wording should preserve that distinction: say
"webhook/MCP/web/browser workflow" when that is what the source proves, and
say "bundled plugin/skill/tool/gateway" only when source proves that direct
surface.

## Unsupported / Excluded Claims

These claims remain unsupported/excluded for direct integration messaging:

- Reddit native plugin/tool
- Stripe native API plugin/tool
- InsForge plugin/tool
- Graphiti/Zep native memory integration
- Fireflies API/plugin/tool

They may still be achievable as generic workflows when the user supplies
credentials, endpoints, MCP servers, browser access, webhook events, or URLs;
that does not turn them into Hermes-native integration parity requirements.

## Follow-up Rows

No implementation rows were created by this audit. Existing source-backed lanes
already cover the direct classes at their proper granularity:

- skills and skill-manager parity for Google Workspace, Obsidian, GitHub, and
  YouTube;
- web/browser/tool backend parity for Firecrawl and generic scraping;
- channel/tool rows for Discord;
- SMS gateway and optional-skill rows for Twilio/telephony;
- MCP/webhook/config rows for user-provided third-party integrations.

Future rows should be added only when a concrete, source-backed behavior is
missing or when Gormes intentionally adopts a Gormes-owned enhancement with its
own contract and tests.
