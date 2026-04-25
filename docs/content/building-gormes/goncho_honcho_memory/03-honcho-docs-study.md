---
title: "Honcho Docs Study Plan"
weight: 3
---

# 03 - Honcho Docs Study Plan

Last studied: 2026-04-25.

Source root: `/home/xel/git/sages-openclaw/workspace-mineru/honcho/docs`.

This page records the implementation plan produced from studying Honcho's
current v3 documentation. It complements the source-level port notes in the
main Goncho page: source files explain how Honcho works internally; docs explain
the behavior SDK users expect at the public edge.

## Source Corpus

Read these local docs before editing this page:

- `v3/documentation/introduction/overview.mdx`
- `v3/documentation/core-concepts/architecture.mdx`
- `v3/documentation/core-concepts/representation.mdx`
- `v3/documentation/core-concepts/reasoning.mdx`
- `v3/documentation/core-concepts/design-patterns.mdx`
- `v3/documentation/features/storing-data.mdx`
- `v3/documentation/features/get-context.mdx`
- `v3/documentation/features/chat.mdx`
- `v3/documentation/features/advanced/search.mdx`
- `v3/documentation/features/advanced/using-filters.mdx`
- `v3/documentation/features/advanced/representation-scopes.mdx`
- `v3/documentation/features/advanced/reasoning-configuration.mdx`
- `v3/documentation/features/advanced/queue-status.mdx`
- `v3/documentation/features/advanced/dreaming.mdx`
- `v3/documentation/features/advanced/summarizer.mdx`
- `v3/documentation/features/advanced/peer-card.mdx`
- `v3/documentation/features/advanced/file-uploads.mdx`
- `v3/documentation/features/advanced/streaming-response.mdx`
- `v3/documentation/reference/cli.mdx`
- `v3/documentation/reference/sdk.mdx`
- `v3/documentation/reference/platform.mdx`
- `v3/documentation/reference/storage.mdx` (empty as of this study)
- `v3/contributing/self-hosting.mdx`
- `v3/contributing/configuration.mdx`
- `v3/contributing/troubleshooting.mdx`
- `v3/guides/integrations/hermes.mdx`
- `v3/guides/integrations/opencode.mdx`
- `v3/guides/integrations/claude-code.mdx`
- `v3/guides/integrations/openclaw.mdx`
- `v3/guides/integrations/mcp.mdx`
- `v3/guides/integrations/crewai.mdx`
- `v3/guides/integrations/langgraph.mdx`
- `v3/guides/integrations/n8n.mdx`
- `v3/guides/integrations/paperclip.mdx`
- `v3/guides/integrations/sillytavern.mdx`
- `v3/guides/integrations/zo-computer.mdx`
- `v3/guides/migrations/mem0.mdx`
- `v3/migrations/from-mem0.mdx`
- `v3/openapi.json`
- `docs/changelog/compatibility-guide.mdx`

The work-packet version of this study lives in
[Agent Work Packets](../04-agent-work-packets/). Use that page when assigning an
implementation slice; use this page to understand the reasoning behind the
slice boundaries. Runtime and operator decisions live in
[Operator Playbook](../05-operator-playbook/).

## Memory Contract Learned From The Docs

Honcho v3 presents memory as four storage primitives plus background
reasoning:

1. **Workspace** isolates applications, environments, or tenants.
2. **Peer** is the durable identity being modeled. Humans, agents, groups, and
   imported entities are all peers.
3. **Session** is the temporal interaction boundary. A session can have one or
   many peers.
4. **Message** is the event that gets stored immediately and then triggers
   background reasoning.
5. **Representation** is the derived, queryable memory for a peer. It includes
   conclusions, summaries, and peer cards, not just stored text.

Goncho already has parts of this contract: `internal/goncho.Service`,
`honcho_*` tools, peer cards, manual conclusions, same-chat recall, and
`scope=user` / `sources[]` plumbing. It does not yet have the full docs-visible
contract for representation options, directional peer cards, filters, queue
status, configuration inheritance, summaries, or dreaming.

## Docs-Driven Gaps

### 1. Context Retrieval Options

Honcho `session.context()` supports these public controls:

- `summary` (default true);
- `tokens`;
- `peer_target`;
- `peer_perspective`;
- `search_query`;
- `limit_to_session`;
- `search_top_k`;
- `search_max_distance`;
- `include_most_frequent`;
- `max_conclusions`.

Goncho currently exposes `peer`, `query`, `max_tokens`, `session_key`,
`scope`, and `sources` in `internal/goncho.ContextParams`, while
`honcho_context` only advertises the first four in the tool schema. The next
context slice should not jump straight to full SDK parity. It should first add
typed fields for the Honcho v3 representation options and fixture-lock how
unsupported options degrade.

Planned fixture shape:

- omitted options preserve the current same-chat behavior;
- `peer_target` and `peer_perspective` map to `(observer, observed)` without
  crossing workspaces;
- `limit_to_session=true` cannot widen recall through `scope=user`;
- `search_top_k`, `search_max_distance`, `include_most_frequent`, and
  `max_conclusions` are accepted only when the observation table exists;
- unsupported options return structured "not implemented" evidence instead of
  silent ignore.

### 2. Search And Filter Grammar

Honcho docs expose workspace, session, and peer search with a common
`{ query, filters?, limit }` shape. The default limit is 10 and the documented
maximum is 100.

The filter grammar is larger than Goncho's current `sources[]` allowlist:

- logical operators: `AND`, `OR`, `NOT`;
- comparison operators: `gt`, `gte`, `lt`, `lte`, `ne`, `in`;
- text operators: `contains`, `icontains`;
- nested `metadata` filters;
- wildcard `"*"`.

Goncho should add a typed filter AST before it adds more endpoints. The first
implementation can support a smaller allowlist, but it must reject unsupported
operators visibly. Silent fallback would create privacy bugs because a caller
could believe a peer/session filter applied when it did not.

### 3. Directional Representations

Honcho's representation scopes are `(workspace, observer, observed)`.
Self-representation is the default path. Directional representation only
appears when `observe_others=true` for the observer in a session.

Important docs rules:

- `observe_me=true` is the default for a peer;
- `observe_others` is session-level and creates peer-to-peer views;
- reasoning is not retroactive for peers that join a session late;
- a `target` parameter selects which observed peer to query;
- `peer.chat()` is for query-specific reasoning; `representation()` is for
  stored conclusions and dashboard-style hydration.

Goncho has an `observer` service default, but peer cards are still keyed as
`(workspace, peer)` in `goncho_peer_cards`. The next storage slice must change
cards and observations to `(workspace, observer, observed)` before claiming
representation-scope parity.

### 4. Peer Cards

Honcho peer cards are stable, biographical facts. They are not summaries and
not arbitrary recent context.

Docs-visible rules:

- max 40 facts;
- facts are `list[str]`;
- manual `set_card` replaces the entire card instead of merging;
- directional cards use the same `target` model as representations;
- `peer_card.use` and `peer_card.create` can be configured independently.

Goncho should enforce the 40-fact cap and replacement semantics in the service
layer, then add target-aware card tests. Current `SetProfile()` accepts any
slice size and has no directional target.

### 5. Configuration Hierarchy

Honcho docs define configuration inheritance as:

`message > session > workspace > global defaults`

Peer observation configuration is separate: peer-level `observe_me` overrides
defaults and workspace configuration, but not session or message configuration.

The docs-visible config blocks are:

- `reasoning.enabled`;
- `peer_card.use`;
- `peer_card.create`;
- `summary.enabled`;
- `summary.messages_per_short_summary`;
- `summary.messages_per_long_summary`;
- `dream.enabled`.

Goncho needs an explicit `[goncho]` namespace in Gormes config before dialectic
or dreamer slices are enabled. Keep the existing Phase 3 `[memory]` knobs for
local recall, decay, and mirrors; use `[goncho]` for Honcho-shaped reasoning
behavior.

### 6. Queue Status

Honcho queue status is observability, not synchronization.

Docs-visible fields:

- `completed_work_units`;
- `in_progress_work_units`;
- `pending_work_units`;
- `total_work_units`;
- optional per-session `sessions`.

Tracked task types are `representation`, `summary`, and `dream`. Internal
webhook, deletion, and vector reconciliation work is not included in queue
status counts.

Critical rule: do not wait for the queue to be empty. New messages can always
arrive, and completion is not a durable application state. Goncho should expose
queue status through `gormes memory status` or a dedicated `gormes goncho
queue-status` command as evidence only.

### 7. Summaries

Honcho summary docs sharpen the context budget plan:

- short summaries run every 20 messages by default and target 1000 tokens;
- long summaries run every 60 messages by default and target 4000 tokens;
- each summary replaces the previous slot of the same type;
- each new summary receives the prior summary plus messages since that summary;
- `context()` reserves 40% of the token budget for summary and 60% for recent
  messages;
- without a token limit, context tries for exhaustive conversation coverage;
- if the newest messages or summaries exceed the budget, summary can be absent.

Goncho should implement summaries as their own table and queue task. Do not
fold this into the existing last-N-turns recall path; the summary slot is a
separate prompt component with its own budget.

### 8. Dreaming

Honcho docs define dreaming as experimental but specific enough to plan:

- dream scope is `(workspace, observer, observed)`;
- a dream needs at least 50 new conclusions since the last dream;
- cooldown is at least 8 hours;
- idle timeout defaults to 60 minutes;
- new activity cancels a pending dream for the observed peer;
- manual schedule is allowed but duplicate pending/in-progress dreams are
  deduplicated;
- deduction runs before induction.

Goncho should not port surprisal sampling first. The first dream slice should
implement the scheduler contract and the deduction/induction sequencing with
fixtures, then add tree-based surprisal later.

### 9. Dialectic Chat Endpoint

The chat docs and OpenAPI contract define `peer.chat()` as query-specific
reasoning over a peer representation. The request accepts:

- `query` (required, 1 to 10000 chars);
- `session_id`;
- `target`;
- `stream`;
- `reasoning_level` (`minimal`, `low`, `medium`, `high`, `max`, default
  `low`).

The non-streaming response shape is `{content: string|null}`. Streaming returns
`text/event-stream`.

Goncho currently exposes `honcho_reasoning`, but the Honcho host docs for
OpenCode, Claude Code, and MCP use `honcho_chat` or `chat` naming. The next
chat slice should add a host-compatible `honcho_chat` alias while preserving
`honcho_reasoning`. The first implementation can reuse deterministic synthesis,
but it must keep this path separate from `honcho_context` and must report
`stream=true` as unsupported until streaming exists.

### 10. File Uploads And Legacy Memory Imports

Honcho's upload path converts files into ordinary session messages and queues
the same background reasoning used by normal messages. Docs advertise PDF,
text, and JSON support. Source and OpenAPI show the runtime request is multipart
form data with required `file` and `peer_id`, optional JSON-ish `metadata`,
optional `configuration`, and optional `created_at`.

Important source-level correction: the file-upload prose mentions chunks around
49,500 characters, but `src/config.py` sets `MAX_MESSAGE_SIZE=25000`, and
`process_file_uploads_for_messages()` receives that value as its runtime
`max_chars` default. Goncho should use the source runtime limit unless Honcho
changes its config.

The first Gormes slice should support text, Markdown, and JSON imports before
PDF. It should never persist original file bytes. It should persist file
metadata on generated messages:

- `file_id`;
- `filename`;
- `chunk_index`;
- `total_chunks`;
- `original_file_size`;
- `content_type`;
- `chunk_character_range`.

This is also the non-destructive migration path for legacy `USER.md`,
`MEMORY.md`, `IDENTITY.md`, `SOUL.md`, workspace `memory/`, and similar files.

### 11. Host Integration Contract

The integration docs are planning inputs, not packages to vendor. Gormes should
not install the Node/Bun plugins, but it must preserve the memory concepts they
depend on:

- shared config roots such as `~/.honcho/config.json`;
- host-scoped workspace and AI peer names;
- session strategies like `per-directory`, `per-repo`, `git-branch`,
  `per-session`, `chat-instance`, and `global`;
- recall modes `hybrid`, `context`, and `tools`;
- directional observation mode;
- status/config tools that explain degraded behavior;
- MCP headers for user, assistant, and workspace identity.

OpenClaw's integration adds two Gormes-relevant details: legacy memory files are
uploaded instead of moved or deleted, and parent agents can observe subagent
sessions through `observeOthers` while staying silent with `observeMe=false`.

### 12. OpenAPI As The Optional HTTP Contract

The MDX API reference pages mostly point at `v3/openapi.json`. Use that JSON as
the exact route/schema source when the optional HTTP adapter becomes active.
Endpoint groups are:

- workspaces: create/list/update/delete/search, queue status, schedule dream;
- peers: create/list/update/sessions/chat/representation/card/context/search;
- sessions: create/list/update/delete/clone/peers/config/context/summaries/search;
- messages: batch create, upload, list, get, update;
- conclusions: create, list, query, delete;
- webhooks and keys: managed-platform features, not local Goncho MVP features.

The HTTP adapter must stay a thin adapter over `goncho.Service`. It must not add
a sidecar, a second store, or a loopback dependency.

### 13. Operational Lessons From Honcho Docs

The design-pattern, CLI, self-hosting, configuration, troubleshooting,
streaming, and integration docs add operational rules that source files alone do
not make obvious:

- Workspaces are application or hard-isolation boundaries, not user records.
- Peers model durable participants. Assistants or bots with deterministic
  behavior usually should not be observed.
- Sessions must follow real context boundaries. Too many tiny sessions fragment
  summaries and `session.context()`.
- Message storage is the primary memory trigger. Manual conclusions are useful
  for quick migrations, but they are not equivalent to raw message import.
- `stream=true` belongs to dialectic chat. The caller accumulates the final
  assistant response and stores it once; partial chunks must not enter memory.
- Operator surfaces need a diagnostic ladder: config, database, migrations,
  provider reachability for enabled model callers, queue backlog, peer cards,
  conclusions, summaries, and context preview.
- Goncho should add a `[goncho]` config namespace through the existing Gormes
  config loader. Do not copy Honcho's Python `.env`/`config.toml` runtime.

## Progress Measurement

For memory planning, progress is measured in `progress.json`, not by prose
length. A row is usable by autoloop only when it has:

- a concrete contract;
- `contract_status`;
- `slice_size`;
- `execution_owner`;
- `trust_class`;
- `degraded_mode`;
- a fixture path;
- source references;
- `ready_when` and `not_ready_when`;
- acceptance criteria;
- write scope;
- test commands;
- a done signal.

Future work quality is measured by how little a worker has to infer. Each row
should name the exact Honcho docs or source files that justify it, the exact
Gormes packages it may edit, and the exact test command that proves the slice.

Parity is not one number. Track it as four gates:

1. **Schema parity**: public tools or HTTP endpoints accept the documented
   fields and reject unsupported ones visibly.
2. **Storage parity**: the SQLite tables can represent Honcho's workspace,
   observer, observed, session, message, card, conclusion, summary, and queue
   concepts.
3. **Runtime parity**: derivation, summary, dream, and dialectic work run
   asynchronously without blocking the kernel recall budget.
4. **Operator parity**: `memory status`, mirrors, logs, or docs explain whether
   a request used same-chat recall, user-scope recall, summaries, observations,
   or a degraded fallback.

## Planner Queue Added From This Study

The corresponding executable roadmap rows live in `architecture_plan/progress.json`
under `3.F - Goncho Honcho Memory Parity`:

- `Goncho context representation options`;
- `Goncho search filter grammar`;
- `Directional peer cards and representation scopes`;
- `Goncho queue status read model`;
- `Goncho summary context budget`;
- `Goncho dialectic chat contract`;
- `Goncho file upload import ingestion`;
- `Goncho topology design fixtures`;
- `Goncho operator diagnostics contract`;
- `Goncho streaming chat persistence contract`;
- `Goncho configuration namespace`.

Those rows are intentionally smaller than "port Honcho." Each one gives the
autoloop source refs, write scope, fixtures, and done signals so it can build
the memory system without rereading every upstream doc. The exact worker packet
for each row is in [Agent Work Packets](../04-agent-work-packets/).
