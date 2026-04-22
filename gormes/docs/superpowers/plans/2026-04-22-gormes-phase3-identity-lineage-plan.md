# Phase 3 Identity + Lineage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add canonical GONCHO identity and lineage primitives for `3.E.7` and `3.E.8` so Gormes can safely unify one user across multiple chats, preserve compression lineage, and search across sessions without breaking same-chat recall defaults or Honcho-compatible tools.

**Architecture:** Keep `internal/session` authoritative for live session routing metadata, add a GONCHO identity/session catalog read model in SQLite for memory queries, and resolve external Honcho-compatible `peer` values to canonical internal `user_id` values only inside `internal/goncho`. The default prompt fence remains same-chat; cross-chat recall is explicit and only allowed when a canonical `user_id` is known.

**Tech Stack:** Go 1.25+, SQLite/FTS5, bbolt, existing `internal/goncho`, `internal/memory`, `internal/session`, `cmd/gormes`

---

## 1. Problem statement

Phase 3 memory already stores `session_id` and `chat_id`, and GONCHO already exposes workspace/peer/session-key surfaces. What does **not** exist yet is a canonical identity model that says which chats belong to the same person, how a compressed child session relates to its parent, and how cross-chat search should be fenced.

Current code has three concrete gaps:

1. `internal/session` only persists `(platform, chat_id) -> session_id`; there is no `user_id` or `parent_session_id` concept.
2. `internal/memory` stores `turns.session_id` and `turns.chat_id`, but recall still has a global exact-name path, so a named entity can leak across chats even when the caller did not opt into cross-chat recall.
3. `internal/goncho` tools accept `peer` and optional `session_key`, but there is no canonical resolution step from external peer aliases to an internal GONCHO `user_id`.

The purpose of `3.E.7` and `3.E.8` is to close those gaps without flattening all sessions into one global stream.

## 2. Non-goals

- No remote Honcho provider parity work.
- No Phase 4 agent-loop implementation.
- No heuristic identity merge based only on entity-name similarity.
- No automatic cross-chat recall by default.
- No retroactive mutation of ancestor turns when a child compression session is created.
- No gateway/platform feature work outside the metadata needed to populate identity and lineage fields.

## 3. Baseline snapshot before implementation

### Ledger state

- `3.E.7.1 user_id concept above chat_id` — `planned`
- `3.E.7.2 Cross-chat entity merge + recall fence` — `planned`
- `3.E.8.1 parent_session_id lineage for compression splits` — `planned`
- `3.E.8.2 Source-filtered FTS/session search across chats` — `planned`

### Contracts that already exist

- `internal/session/session.go`
  - `Map` only supports `Get(ctx, key) -> session_id` and `Put(ctx, key, sessionID)`.
  - Canonical key is still transport-scoped, for example `telegram:<chat_id>` or `discord:<channel_id>`.
- `internal/memory/schema.go`
  - `turns` contains `session_id` and `chat_id`, but no `user_id`, `source`, or `parent_session_id`.
  - GONCHO persistence currently stores `goncho_peer_cards` and `goncho_conclusions`.
- `internal/memory/recall.go` and `internal/memory/recall_test.go`
  - FTS fallback is chat-scoped.
  - Exact-name seeds are global and currently bypass chat fencing.
- `internal/goncho/types.go` and `internal/goncho/service.go`
  - Tool-facing contracts use `peer` and optional `session_key`.
  - There is no resolver from external `peer` aliases to canonical internal identity.

### Missing contracts

- No canonical `user_id > chat_id > session_id` hierarchy.
- No durable lineage metadata for compression/fork descendants.
- No source-filtered session search API.
- No operator-safe visibility into unresolved identity bindings or orphan lineage chains.

### Risks if implemented without a unified identity model

- Cross-chat recall will either leak unrelated facts or silently miss legitimate same-user facts.
- Session lineage will become write-only metadata that cannot be debugged when compression splits misbehave.
- Honcho-compatible tools will drift into inconsistent behavior because `peer` will mean “chat” in some flows and “user” in others.
- Search semantics will diverge across packages (`internal/goncho`, `internal/memory`, `internal/session`) and become impossible to reason about.

## 4. Contract decisions

### 4.1 Canonical identity model

The canonical GONCHO hierarchy is:

`user_id > chat_id > session_id`

- `user_id`
  - Durable participant identity inside GONCHO.
  - May correspond to one or more transport chats or sessions.
  - Becomes the internal target for cross-chat recall and Honcho-compatible peer resolution.
- `chat_id`
  - Transport/channel boundary such as `telegram:42`, `discord:abc`, or `slack:C123`.
  - Remains the default recall fence unless cross-chat scope is explicitly requested.
- `session_id`
  - One conversation instance or branch.
  - May change over time for compression or fork events even when `chat_id` stays the same.

### 4.2 Recall fence rules

- Default mode is **same-chat only**.
- Cross-chat recall is **opt-in**, never implicit.
- Opt-in cross-chat recall requires:
  - a resolved canonical `user_id`
  - an explicit scope request from the caller
  - optional source filters expressed as canonical source names (`telegram`, `discord`, `slack`, etc.)
- `parent_session_id` does **not** widen prompt recall automatically. Lineage is query/debug metadata first.

### 4.3 Lineage model

- `parent_session_id` is nullable.
- Root sessions have `parent_session_id = NULL`.
- Only descendant sessions created by compression split or explicit branch/fork carry a non-null parent.
- Parent linkage is append-only metadata. A child session points at an ancestor; ancestors are not rewritten.
- A lineage chain is valid even if some legacy ancestors predate the schema; missing parents surface as operator-visible orphans, not silent data loss.

### 4.4 Source-filtered search contract

- Search inputs operate on canonical `user_id` plus optional `sources[]`.
- Default search scope:
  - if `session_key` is present and no scope override is set, search is same-chat
  - if cross-chat scope is set and `user_id` resolves, search spans that user’s sessions
- `sources[]` is an allowlist of canonical source names and never matches raw `chat_id` prefixes ad hoc.
- Message search sorts by `ts_unix DESC, id DESC`.
- Session search sorts by latest observed turn timestamp DESC, then `session_id ASC` for determinism.

## 5. Data model and migration plan

### 5.1 SQLite additions in `internal/memory`

Add a new migration after schema `3f`:

- `goncho_identity_aliases`
  - `workspace_id TEXT NOT NULL`
  - `alias_kind TEXT NOT NULL CHECK(alias_kind IN ('peer','chat','session'))`
  - `alias_value TEXT NOT NULL`
  - `user_id TEXT NOT NULL`
  - `binding_kind TEXT NOT NULL CHECK(binding_kind IN ('manual','derived','legacy_self'))`
  - `created_at INTEGER NOT NULL`
  - `updated_at INTEGER NOT NULL`
  - `PRIMARY KEY(workspace_id, alias_kind, alias_value)`
- `goncho_session_catalog`
  - `workspace_id TEXT NOT NULL`
  - `session_id TEXT NOT NULL`
  - `source TEXT NOT NULL`
  - `chat_id TEXT NOT NULL`
  - `user_id TEXT`
  - `parent_session_id TEXT`
  - `lineage_kind TEXT NOT NULL CHECK(lineage_kind IN ('primary','compression_split','fork')) DEFAULT 'primary'`
  - `created_at INTEGER NOT NULL`
  - `updated_at INTEGER NOT NULL`
  - `PRIMARY KEY(workspace_id, session_id)`

Recommended indexes:

- `idx_goncho_identity_aliases_user` on `(workspace_id, user_id, alias_kind)`
- `idx_goncho_session_catalog_user` on `(workspace_id, user_id, updated_at DESC)`
- `idx_goncho_session_catalog_chat` on `(workspace_id, chat_id, updated_at DESC)`
- `idx_goncho_session_catalog_source` on `(workspace_id, source, updated_at DESC)`
- `idx_goncho_session_catalog_parent` on `(workspace_id, parent_session_id)`

### 5.2 bbolt additions in `internal/session`

Add a new bucket alongside `sessions_v1`:

- `session_meta_v1`
  - keyed by `session_id`
  - value is a stable JSON envelope with:
    - `session_id`
    - `source`
    - `chat_id`
    - `user_id`
    - `parent_session_id`
    - `lineage_kind`
    - `created_at`
    - `updated_at`

This keeps live routing metadata close to the existing session map while SQLite remains the query/read model for memory and search.

### 5.3 Backfill strategy

1. Backfill `goncho_session_catalog` from distinct rows in `turns`.
2. Derive `source` from the stable transport prefix in `chat_id` where present; use `unknown` for legacy rows that cannot be classified.
3. Leave `user_id` null when there is no explicit safe binding.
4. Seed `alias_kind='peer'` rows with `binding_kind='legacy_self'` only for existing GONCHO `peer_id` values when preserving tool behavior requires it.
5. Never infer `user_id` by entity-name coincidence.

### 5.4 Rollback strategy

- The migration is additive.
- Runtime behavior stays legacy until new scope fields are explicitly used.
- Rollback means disabling the new code paths and ignoring the new tables/bucket.
- If a hard rollback is required, restore the pre-migration SQLite/bbolt backups instead of attempting destructive in-place down-migrations.

## 6. API and service design

### 6.1 `internal/session`

Add a metadata-oriented surface without breaking `Map`:

- New file: `internal/session/directory.go`
- New types:
  - `Metadata`
  - `Directory`
  - `SearchFilter`
- Responsibilities:
  - write/read `user_id`, `parent_session_id`, and `source` metadata
  - list sessions by `user_id`
  - walk lineage for one `session_id`
  - expose read-only data to `SessionIndexMirror`

`Map` remains the low-level resume primitive. `Directory` is the richer metadata contract layered beside it.

### 6.2 `internal/memory`

Add read-model and query primitives:

- New file: `internal/memory/session_catalog.go`
- Responsibilities:
  - migrate/backfill `goncho_identity_aliases` and `goncho_session_catalog`
  - resolve recall scope to same-chat vs cross-chat
  - search sessions/messages with source filters
  - provide identity/lineage summaries for status surfaces

`RecallInput` should evolve into a richer scope shape, for example:

```go
type RecallScope struct {
	SessionID  string
	ChatID     string
	UserID     string
	CrossChat  bool
	Sources    []string
}
```

The implementation must preserve legacy callers by adapting the current `RecallInput` into `RecallScope`.

### 6.3 `internal/goncho`

Keep external Honcho-compatible names, add internal resolution:

- New file: `internal/goncho/identity.go`
- Responsibilities:
  - resolve external `peer` and optional `session_key` into canonical `user_id`
  - keep `peer` stable in JSON I/O while using resolved `user_id` internally
  - apply same-chat default / cross-chat opt-in rules to `Search` and `Context`

Recommended additive request fields:

```go
type SearchParams struct {
	Peer       string   `json:"peer"`
	Query      string   `json:"query"`
	MaxTokens  int      `json:"max_tokens,omitempty"`
	SessionKey string   `json:"session_key,omitempty"`
	Scope      string   `json:"scope,omitempty"`   // "", "same_chat", "user"
	Sources    []string `json:"sources,omitempty"` // canonical transport names
}
```

Same for `ContextParams`. Existing clients remain valid because the new fields are optional.

### 6.4 Honcho-compatible tool surfaces

`internal/tools/honcho_tools.go` should stay backward-compatible:

- keep tool names: `honcho_profile`, `honcho_search`, `honcho_context`, `honcho_reasoning`, `honcho_conclude`
- keep existing required fields unchanged
- allow optional scope/source fields to flow through once implemented

## 7. File map for implementation

### Create

- `internal/session/directory.go`
- `internal/session/directory_test.go`
- `internal/memory/session_catalog.go`
- `internal/memory/session_catalog_test.go`
- `internal/goncho/identity.go`
- `internal/goncho/identity_test.go`

### Modify

- `internal/session/bolt.go`
- `internal/session/bolt_test.go`
- `internal/session/index_mirror.go`
- `internal/session/index_mirror_test.go`
- `internal/memory/schema.go`
- `internal/memory/migrate.go`
- `internal/memory/migrate_test.go`
- `internal/memory/worker.go`
- `internal/memory/recall.go`
- `internal/memory/recall_sql.go`
- `internal/memory/recall_test.go`
- `internal/memory/status.go`
- `internal/memory/status_test.go`
- `internal/goncho/types.go`
- `internal/goncho/service.go`
- `internal/goncho/sql.go`
- `internal/goncho/contracts_test.go`
- `internal/goncho/service_test.go`
- `internal/tools/honcho_tools.go`
- `cmd/gormes/memory.go`
- `cmd/gormes/session_mirror.go`
- `cmd/gormes/session_mirror_test.go`

## 8. TDD execution slices

### Task 1: `3.E.7.1` freeze session metadata and canonical identity scaffolding

**Files:**
- Create: `internal/session/directory.go`, `internal/session/directory_test.go`
- Modify: `internal/session/bolt.go`, `internal/session/bolt_test.go`
- Modify: `internal/memory/schema.go`, `internal/memory/migrate_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestBoltDirectory_MetadataRoundTrip(t *testing.T) {}
func TestBoltDirectory_RejectsConflictingUserBinding(t *testing.T) {}
func TestMigrate_AddsGonchoIdentityAndSessionCatalogTables(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/session ./internal/memory -run 'Test(BoltDirectory_|Migrate_AddsGonchoIdentityAndSessionCatalogTables)' -count=1`

Expected: FAIL because `Directory` types, metadata bucket handling, and new schema objects do not exist yet.

- [ ] **Step 3: Write the minimal implementation**

```go
type Metadata struct {
	SessionID       string
	Source          string
	ChatID          string
	UserID          string
	ParentSessionID string
	LineageKind     string
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/session ./internal/memory -run 'Test(BoltDirectory_|Migrate_AddsGonchoIdentityAndSessionCatalogTables)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/session internal/memory
git commit -m "feat: scaffold goncho identity and session catalog"
```

### Task 2: `3.E.7.2` same-chat default fence plus opt-in cross-chat recall

**Files:**
- Create: `internal/goncho/identity.go`, `internal/goncho/identity_test.go`
- Modify: `internal/memory/recall.go`, `internal/memory/recall_sql.go`, `internal/memory/recall_test.go`
- Modify: `internal/goncho/types.go`, `internal/goncho/service.go`, `internal/goncho/service_test.go`, `internal/goncho/contracts_test.go`
- Modify: `internal/tools/honcho_tools.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestProvider_GetContext_SameChatDefaultDoesNotLeakNamedEntityAcrossChats(t *testing.T) {}
func TestProvider_GetContext_OptInCrossChatUsesResolvedUserScope(t *testing.T) {}
func TestService_Search_DefaultsToSameChatScope(t *testing.T) {}
func TestService_Search_UserScopeAllowsCrossChatWhenUserResolved(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/memory ./internal/goncho -run 'Test(Provider_GetContext_|Service_Search_)' -count=1`

Expected: FAIL because exact-name recall is currently global and GONCHO has no identity resolver or scope fields.

- [ ] **Step 3: Write the minimal implementation**

```go
type RecallScope struct {
	SessionID string
	ChatID    string
	UserID    string
	CrossChat bool
	Sources   []string
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/memory ./internal/goncho -run 'Test(Provider_GetContext_|Service_Search_)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/memory internal/goncho internal/tools
git commit -m "feat: add goncho identity resolution and scoped recall"
```

### Task 3: `3.E.8.1` parent lineage for compression splits

**Files:**
- Modify: `internal/session/directory.go`, `internal/session/directory_test.go`
- Modify: `internal/session/index_mirror.go`, `internal/session/index_mirror_test.go`
- Modify: `internal/memory/session_catalog.go`, `internal/memory/session_catalog_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestBoltDirectory_LineageRoundTrip(t *testing.T) {}
func TestSessionIndexMirror_RendersUserAndParentSessionFields(t *testing.T) {}
func TestSessionCatalog_LineageQueryReturnsAncestorChain(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/session ./internal/memory -run 'Test(BoltDirectory_Lineage|SessionIndexMirror_RendersUserAndParentSessionFields|SessionCatalog_LineageQueryReturnsAncestorChain)' -count=1`

Expected: FAIL because no lineage metadata or enriched mirror output exists yet.

- [ ] **Step 3: Write the minimal implementation**

```go
type Metadata struct {
	SessionID       string
	ParentSessionID string
	LineageKind     string
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/session ./internal/memory -run 'Test(BoltDirectory_Lineage|SessionIndexMirror_RendersUserAndParentSessionFields|SessionCatalog_LineageQueryReturnsAncestorChain)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/session internal/memory
git commit -m "feat: add session lineage metadata and mirror output"
```

### Task 4: `3.E.8.2` source-filtered FTS/session search across chats

**Files:**
- Modify: `internal/memory/session_catalog.go`, `internal/memory/session_catalog_test.go`
- Modify: `internal/goncho/service.go`, `internal/goncho/sql.go`, `internal/goncho/service_test.go`
- Modify: `internal/goncho/types.go`, `internal/goncho/contracts_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestSessionCatalog_SearchMessagesFiltersBySource(t *testing.T) {}
func TestSessionCatalog_SearchSessionsOrdersByLatestTurn(t *testing.T) {}
func TestService_Context_UserScopeRespectsSourceFilter(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/memory ./internal/goncho -run 'Test(SessionCatalog_Search|Service_Context_UserScopeRespectsSourceFilter)' -count=1`

Expected: FAIL because there is no source-aware session catalog search contract.

- [ ] **Step 3: Write the minimal implementation**

```go
type SearchFilter struct {
	UserID  string
	Sources []string
	Query   string
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/memory ./internal/goncho -run 'Test(SessionCatalog_Search|Service_Context_UserScopeRespectsSourceFilter)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/memory internal/goncho
git commit -m "feat: add source-filtered cross-chat search"
```

### Task 5: operator visibility and safe rollout

**Files:**
- Modify: `internal/memory/status.go`, `internal/memory/status_test.go`
- Modify: `cmd/gormes/memory.go`
- Modify: `cmd/gormes/session_mirror.go`, `cmd/gormes/session_mirror_test.go`
- Modify docs under `docs/content/building-gormes/architecture_plan/`

- [ ] **Step 1: Write the failing tests**

```go
func TestReadExtractorStatus_ReportsIdentityAndLineageDrift(t *testing.T) {}
func TestStartSessionIndexMirror_UsesEnrichedMetadataWhenAvailable(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./cmd/gormes ./internal/memory -run 'Test(ReadExtractorStatus_|StartSessionIndexMirror_)' -count=1`

Expected: FAIL because status and mirror output do not report unresolved users/orphan lineage yet.

- [ ] **Step 3: Write the minimal implementation**

```go
type IdentityLineageStatus struct {
	ResolvedUsers     int
	UnresolvedSessions int
	OrphanLineages    int
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./cmd/gormes ./internal/memory -run 'Test(ReadExtractorStatus_|StartSessionIndexMirror_)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/gormes internal/memory docs
git commit -m "feat: expose identity and lineage status surfaces"
```

## 9. Observability and operator surfaces

These features should not be dark launches. Operators need to see the new metadata.

- `gormes memory status`
  - report resolved user count
  - report unresolved session count
  - report orphan lineage count
  - report cross-chat scope disabled cases when a user is unresolved
- `sessions/index.yaml`
  - enrich entries with `source`, `user_id`, `parent_session_id`, and `lineage_kind` when known
- `gormes session export <id> --format=markdown`
  - include front matter for `user_id`, `chat_id`, `source`, `parent_session_id`

None of these outputs should dump raw alias tables wholesale. Safe summaries first, raw internals only behind deliberate debug surfaces later.

## 10. Risk matrix and sequencing

### Highest-risk edges

| Risk | Why it matters | Mitigation |
|---|---|---|
| Same-chat leakage persists through exact-name recall | Violates prompt fence expectations today | Fix recall scoping before enabling any cross-chat opt-in path |
| False user merges | Causes privacy bugs and bad memory contamination | No heuristic merge; require explicit alias binding or safe legacy-self mapping |
| Divergent session metadata across bbolt and SQLite | Makes debugging impossible | Treat bbolt as routing source of truth and SQLite as derived query model; add mirror/status drift checks |
| Orphan child sessions after compression | Makes exported history and search ambiguous | Surface orphan counts in status; keep lineage queryable even when incomplete |

### Dependency ordering with existing Phase 3 work

1. `3.E.1 Session Index Mirror`
   - already shipped and must remain stable
   - only enrich its output after metadata writes are deterministic
2. `3.E.4 Extraction State Visibility`
   - already shipped and should gain identity/lineage drift counters before cross-chat rollout
3. `3.E.7.1 user_id concept above chat_id`
   - first real prerequisite for every later slice
4. `3.E.7.2 cross-chat entity merge + recall fence`
   - second, because current global exact-name leakage must be corrected before opt-in widening
5. `3.E.8.1 parent_session_id lineage`
   - third, once identity contracts are stable
6. `3.E.8.2 source-filtered search`
   - last, because it depends on both catalog data and stable scope semantics

## 11. Definition of done

`3.E.7` and `3.E.8` are done when all of the following are true:

- `internal/session` persists and reads `user_id` and `parent_session_id` metadata without breaking `sessions_v1`
- `internal/memory` stores a queryable GONCHO session catalog plus identity aliases
- same-chat remains the default recall fence for both memory recall and Honcho-compatible tools
- cross-chat recall only activates when explicitly requested and a canonical `user_id` resolves
- compression/fork descendants expose `parent_session_id` through mirror/export/debug surfaces
- source-filtered session/message search returns deterministic results across mixed-platform fixtures
- docs, ledger, and progress generator outputs are synchronized

## 12. Validation checklist

Run these commands at the end of implementation:

```bash
go test ./internal/session ./internal/memory ./internal/goncho ./cmd/gormes -count=1
go test ./internal/progress -count=1
go run ./cmd/progress-gen -write
go run ./cmd/progress-gen -validate
go test ./docs -count=1
```

If any command fails, `3.E.7` / `3.E.8` are not done.
