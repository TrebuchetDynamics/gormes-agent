# Phase 3 Identity + Lineage Execution Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close Phase 3 memory identity/lineage delivery safely by sequencing `3.E.6.1`, `3.E.7.2`, `3.E.8.1`, and `3.E.8.2` into small TDD slices that preserve same-chat default safety, keep GONCHO/Honcho boundaries coherent, and make the remaining freshness/lineage/operator gaps explicit before this area is treated as fully closed.

**Architecture:** Treat `internal/session` as the canonical live identity directory, keep `internal/memory` responsible for fenced recall and session/message search semantics, and keep `internal/goncho` plus `internal/tools/honcho_tools.go` as the Honcho-compatible edge. The key delivery rule is: no new cross-chat reach becomes “real” until freshness (`last_seen`), deny-by-default fence behavior, lineage metadata, and operator evidence are all green in that order.

**Tech Stack:** Go 1.25+, SQLite/FTS5, bbolt, `internal/memory`, `internal/session`, `internal/goncho`, Cobra CLI, docs/progress generator

---

## 1. Current baseline

This execution plan covers these exact backlog items:

- `3.E.6.1 Relationship last_seen tracking`
- `3.E.7.2 Cross-chat entity merge + recall fence`
- `3.E.8.1 parent_session_id lineage for compression splits`
- `3.E.8.2 Source-filtered FTS/session search across chats`

### Already frozen or shipped

- `3.E.7.1` is already shipped in `internal/session/directory.go`:
  - canonical `user_id > chat_id > session_id` metadata exists
  - chat-to-user binding conflicts are rejected
  - resumed sessions inherit prior chat bindings
- `3.E.7.2` is shipped in code and ledger, but still needs regression locking around the tool edge:
  - `internal/memory/recall.go` already supports `UserID`, `CrossChat`, and `Sources`
  - same-chat default fencing is already covered in `internal/memory/recall_test.go`
  - `internal/goncho/service.go` already supports `Scope: "user"` turn-search fallback
- `3.E.8.2` is shipped in code and ledger, but still needs lineage-aware/operator-safe closeout around the helper boundary:
  - `internal/memory/session_catalog.go` already provides `SearchMessages` and `SearchSessions`
  - source-filtered user-scope search already has passing tests

### Still underspecified or not delivered end-to-end

- `3.E.6.1` is not landed:
  - no `relationships.last_seen` column exists in `internal/memory/schema.go`
  - recall decay still keys off `updated_at`, not interaction freshness
- `3.E.7.2` still has delivery-edge debt even though the core runtime path is shipped:
  - `internal/tools/honcho_tools.go` schemas still do not advertise `scope`/`sources`
  - deny/fallback behavior is not yet surfaced to operators
  - no explicit completion gate ties memory, goncho, and tool schemas together
- `3.E.8.1` is not landed:
  - `session.Metadata` has no `parent_session_id` or `lineage_kind`
  - `sessions/index.yaml` only mirrors raw key -> session_id mappings
- `3.E.8.2` still has closeout debt even though the helper/search path is shipped:
  - session search results are not lineage-aware
  - search semantics are implemented at helper level but not yet fully visible/auditable
  - current search depends on split `source` + `chat_id` metadata while `turns.chat_id` is stored as a combined `source:chat` key

### Boundary collision risks to keep in view

1. **Memory vs session normalization**
   - `session.Metadata` stores `Source="telegram"` and `ChatID="42"`, while `turns.chat_id` is `"telegram:42"`.
   - Any search or recall path that forgets to normalize these forms will leak or silently miss results.
2. **GONCHO peer vs canonical user**
   - `honcho_*` surfaces still expose `peer`, but user-scope search assumes `peer == user_id`.
   - Without explicit schema/documentation alignment, callers cannot tell when `peer` means transport identity versus canonical user.
3. **Recall fence vs search helpers**
   - `internal/memory/recall.go` and `internal/memory/session_catalog.go` now both know about cross-chat scope.
   - If their source filtering or fallback semantics drift, prompt injection and tool search will disagree on what “same user” means.
4. **Operator visibility lag**
   - `gormes memory status` only reports extractor health today.
   - `SessionIndexMirror` does not show identity or lineage metadata yet.
   - That means future cross-chat or lineage regressions could exist without an operator-readable audit trail.

## 2. Delivery sequence

The execution order is fixed:

`3.E.6.1 -> 3.E.7.2 -> 3.E.8.1 -> 3.E.8.2`

Reasoning:

1. `3.E.6.1` first so freshness exists before any wider recall/search behavior is treated as reliable.
2. `3.E.7.2` second so the already-landed recall/search widening paths are regression-locked before lineage makes the blast radius larger.
3. `3.E.8.1` third so lineage metadata lands after identity/fence semantics are stable.
4. `3.E.8.2` last so the shipped session/message search helpers become lineage-aware and operator-auditable instead of remaining a partially isolated helper surface.

### Dependency graph

```text
3.E.6.1 Relationship last_seen tracking
  -> 3.E.7.2 Cross-chat entity merge + recall fence closeout
    -> 3.E.8.1 parent_session_id lineage for compression splits
      -> 3.E.8.2 source-filtered FTS/session search across chats
        -> operator evidence + docs/progress closeout
```

## 3. Safety model

### Recall fence safety

- Same-chat is the default.
- `opt-in cross-chat` is the only widening mode.
- Cross-chat requires all of:
  - `CrossChat == true`
  - non-empty canonical `UserID`
  - resolvable session metadata directory
- Missing user resolution or empty metadata list must fall back to same-chat, not global scope.
- `Sources` is always an allowlist; empty means “all sources bound to this user,” never “all chats in the database.”

### Lineage safety

- `parent_session_id` is append-only descendant metadata.
- Root sessions remain `parent_session_id=""` and `lineage_kind="primary"`.
- Self-parenting and trivial loops are invalid.
- Orphans are visible and tolerated; they are not a fatal runtime condition.

### Search safety

- Cross-chat message/session search is always scoped to one canonical `user_id`.
- Search never expands scope purely because a session has a parent.
- Lineage is returned as metadata, not as an automatic search-scope widener.

## 4. Data migration and backfill plan

### 4.1 `relationships.last_seen`

- Add `last_seen INTEGER NOT NULL DEFAULT 0` to `relationships`.
- One-time migration backfill:

```sql
ALTER TABLE relationships ADD COLUMN last_seen INTEGER NOT NULL DEFAULT 0;
UPDATE relationships SET last_seen = updated_at WHERE last_seen = 0;
```

- During closeout, relationship upserts must set:
  - `updated_at` for structural mutation freshness
  - `last_seen` for observation freshness
- Recall decay must use:

```sql
COALESCE(NULLIF(r.last_seen, 0), r.updated_at)
```

until every migrated row is guaranteed to have `last_seen`.

### 4.2 Lineage fields

- Extend `session.Metadata` with:
  - `ParentSessionID string`
  - `LineageKind string`
- Backfill strategy:
  - legacy rows default to root lineage
  - no attempt to infer historical parents from transcripts
  - only future compression/fork paths set non-root lineage explicitly
- Mirror/status surfaces should count orphans instead of mutating or deleting them.

## 5. Package/file touch map

### Slice A: freshness

- Modify: `internal/memory/schema.go`
- Modify: `internal/memory/migrate.go`
- Modify: `internal/memory/migrate_test.go`
- Modify: `internal/memory/graph.go`
- Modify: `internal/memory/graph_test.go`
- Modify: `internal/memory/recall_sql.go`
- Modify: `internal/memory/recall_sql_test.go`

### Slice B: recall fence closeout

- Modify: `internal/memory/recall.go`
- Modify: `internal/memory/recall_test.go`
- Modify: `internal/memory/recall_sql.go`
- Modify: `internal/memory/recall_sql_test.go`

### Slice C: Honcho-compatible edge closeout

- Modify: `internal/goncho/types.go`
- Modify: `internal/goncho/service.go`
- Modify: `internal/goncho/contracts_test.go`
- Modify: `internal/goncho/service_test.go`
- Modify: `internal/tools/honcho_tools.go`

### Slice D: lineage metadata

- Modify: `internal/session/directory.go`
- Modify: `internal/session/directory_test.go`
- Modify: `internal/session/mem.go`
- Modify: `internal/session/index_mirror.go`
- Modify: `internal/session/index_mirror_test.go`
- Modify: `cmd/gormes/session_mirror_test.go`

### Slice E: lineage-aware search

- Modify: `internal/memory/session_catalog.go`
- Modify: `internal/memory/session_catalog_test.go`
- Modify: `internal/goncho/service.go`
- Modify: `internal/goncho/service_test.go`

### Slice F: operator evidence

- Modify: `internal/memory/status.go`
- Modify: `internal/memory/status_test.go`
- Modify: `cmd/gormes/memory.go`
- Modify: `internal/session/index_mirror.go`
- Modify docs under `docs/content/building-gormes/architecture_plan/`

## 6. TDD slices

### Task 1: `3.E.6.1` relationship freshness (`last_seen`)

**Files:**
- Modify: `internal/memory/schema.go`
- Modify: `internal/memory/migrate.go`
- Modify: `internal/memory/migrate_test.go`
- Modify: `internal/memory/graph.go`
- Modify: `internal/memory/graph_test.go`
- Modify: `internal/memory/recall_sql.go`
- Modify: `internal/memory/recall_sql_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestMigrate_3fTo3g_AddsRelationshipLastSeenColumn(t *testing.T) {}
func TestUpsertRelationship_BumpsLastSeenWithoutBreakingUpdatedAt(t *testing.T) {}
func TestEnumerateRelationships_DecayUsesLastSeenWhenPresent(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/memory -run 'Test(Migrate_3fTo3g_AddsRelationshipLastSeenColumn|UpsertRelationship_BumpsLastSeenWithoutBreakingUpdatedAt|EnumerateRelationships_DecayUsesLastSeenWhenPresent)' -count=1`

Expected: FAIL because `relationships.last_seen` does not exist and recall SQL still uses `updated_at` only.

- [ ] **Step 3: Write the minimal implementation**

```go
// schema.go
ALTER TABLE relationships ADD COLUMN last_seen INTEGER NOT NULL DEFAULT 0;
UPDATE relationships SET last_seen = updated_at WHERE last_seen = 0;
```

```go
// recall_sql.go
func freshnessExpr() string {
	return "COALESCE(NULLIF(r.last_seen, 0), r.updated_at)"
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/memory -run 'Test(Migrate_3fTo3g_AddsRelationshipLastSeenColumn|UpsertRelationship_BumpsLastSeenWithoutBreakingUpdatedAt|EnumerateRelationships_DecayUsesLastSeenWhenPresent)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/memory/schema.go internal/memory/migrate.go internal/memory/migrate_test.go internal/memory/graph.go internal/memory/graph_test.go internal/memory/recall_sql.go internal/memory/recall_sql_test.go
git commit -m "feat: track relationship last_seen for recall decay"
```

### Task 2: `3.E.7.2` recall fence hardening in `internal/memory`

**Files:**
- Modify: `internal/memory/recall.go`
- Modify: `internal/memory/recall_test.go`
- Modify: `internal/memory/recall_sql.go`
- Modify: `internal/memory/recall_sql_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestProvider_GetContext_CrossChatWithoutUserIDFallsBackToSameChat(t *testing.T) {}
func TestProvider_GetContext_CrossChatMissingDirectoryFallsBackToSameChat(t *testing.T) {}
func TestProvider_GetContext_CrossChatUnknownUserFallsBackToSameChat(t *testing.T) {}
func TestProvider_GetContext_CrossChatSourceFilterAppliesToExactAndSemanticSeeds(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/memory -run 'Test(Provider_GetContext_CrossChatWithoutUserIDFallsBackToSameChat|Provider_GetContext_CrossChatMissingDirectoryFallsBackToSameChat|Provider_GetContext_CrossChatUnknownUserFallsBackToSameChat|Provider_GetContext_CrossChatSourceFilterAppliesToExactAndSemanticSeeds)' -count=1`

Expected: FAIL until all three seed paths and fallback branches obey the same deny-by-default rule.

- [ ] **Step 3: Write the minimal implementation**

```go
type RecallInput struct {
	UserMessage string
	ChatKey     string
	SessionID   string
	UserID      string
	CrossChat   bool
	Sources     []string
}
```

```go
func (p *Provider) allowedChatKeys(ctx context.Context, in RecallInput) []string {
	// same-chat unless explicit cross-chat + resolved user metadata
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/memory -run 'Test(Provider_GetContext_CrossChatWithoutUserIDFallsBackToSameChat|Provider_GetContext_CrossChatMissingDirectoryFallsBackToSameChat|Provider_GetContext_CrossChatUnknownUserFallsBackToSameChat|Provider_GetContext_CrossChatSourceFilterAppliesToExactAndSemanticSeeds)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/memory/recall.go internal/memory/recall_test.go internal/memory/recall_sql.go internal/memory/recall_sql_test.go
git commit -m "feat: harden same-chat default recall fence"
```

### Task 3: `3.E.7.2` Honcho-compatible tool boundary closeout

**Files:**
- Modify: `internal/goncho/types.go`
- Modify: `internal/goncho/service.go`
- Modify: `internal/goncho/contracts_test.go`
- Modify: `internal/goncho/service_test.go`
- Modify: `internal/tools/honcho_tools.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestContractContextParamsOptionalScopeAndSourcesJSONShape(t *testing.T) {}
func TestHonchoSearchToolSchema_AdvertisesScopeAndSources(t *testing.T) {}
func TestHonchoContextToolSchema_AdvertisesScopeAndSources(t *testing.T) {}
func TestService_SearchUserScopeRequiresExplicitScopeValue(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/goncho ./internal/tools -run 'Test(ContractContextParamsOptionalScopeAndSourcesJSONShape|HonchoSearchToolSchema_AdvertisesScopeAndSources|HonchoContextToolSchema_AdvertisesScopeAndSources|Service_SearchUserScopeRequiresExplicitScopeValue)' -count=1`

Expected: FAIL because the tool schemas do not yet advertise the scope/source contract and the service edge does not yet document/lock the explicit-scope requirement well enough.

- [ ] **Step 3: Write the minimal implementation**

```go
type SearchParams struct {
	Peer       string   `json:"peer"`
	Query      string   `json:"query"`
	MaxTokens  int      `json:"max_tokens,omitempty"`
	SessionKey string   `json:"session_key,omitempty"`
	Scope      string   `json:"scope,omitempty"`
	Sources    []string `json:"sources,omitempty"`
}
```

```json
{"type":"object","properties":{"peer":{"type":"string"},"query":{"type":"string"},"scope":{"type":"string","enum":["same_chat","user"]},"sources":{"type":"array","items":{"type":"string"}}}}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/goncho ./internal/tools -run 'Test(ContractContextParamsOptionalScopeAndSourcesJSONShape|HonchoSearchToolSchema_AdvertisesScopeAndSources|HonchoContextToolSchema_AdvertisesScopeAndSources|Service_SearchUserScopeRequiresExplicitScopeValue)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/goncho/types.go internal/goncho/service.go internal/goncho/contracts_test.go internal/goncho/service_test.go internal/tools/honcho_tools.go
git commit -m "feat: expose scoped goncho search contract at tool edge"
```

### Task 4: `3.E.8.1` lineage metadata in `internal/session`

**Files:**
- Modify: `internal/session/directory.go`
- Modify: `internal/session/directory_test.go`
- Modify: `internal/session/mem.go`
- Modify: `internal/session/index_mirror.go`
- Modify: `internal/session/index_mirror_test.go`
- Modify: `cmd/gormes/session_mirror_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestBoltMap_MetadataRoundTripIncludesParentSessionAndLineageKind(t *testing.T) {}
func TestBoltMap_PutMetadataRejectsSelfParent(t *testing.T) {}
func TestSessionIndexMirror_WriteIncludesIdentityAndLineageFields(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/session ./cmd/gormes -run 'Test(BoltMap_MetadataRoundTripIncludesParentSessionAndLineageKind|BoltMap_PutMetadataRejectsSelfParent|SessionIndexMirror_WriteIncludesIdentityAndLineageFields)' -count=1`

Expected: FAIL because `session.Metadata` has no lineage fields and the mirror output still renders only key -> session_id pairs.

- [ ] **Step 3: Write the minimal implementation**

```go
type Metadata struct {
	SessionID       string `json:"session_id"`
	Source          string `json:"source,omitempty"`
	ChatID          string `json:"chat_id,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	ParentSessionID string `json:"parent_session_id,omitempty"`
	LineageKind     string `json:"lineage_kind,omitempty"`
	UpdatedAt       int64  `json:"updated_at"`
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/session ./cmd/gormes -run 'Test(BoltMap_MetadataRoundTripIncludesParentSessionAndLineageKind|BoltMap_PutMetadataRejectsSelfParent|SessionIndexMirror_WriteIncludesIdentityAndLineageFields)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/session/directory.go internal/session/directory_test.go internal/session/mem.go internal/session/index_mirror.go internal/session/index_mirror_test.go cmd/gormes/session_mirror_test.go
git commit -m "feat: add lineage metadata to session directory"
```

### Task 5: `3.E.8.2` lineage-aware source-filtered search

**Files:**
- Modify: `internal/memory/session_catalog.go`
- Modify: `internal/memory/session_catalog_test.go`
- Modify: `internal/goncho/service.go`
- Modify: `internal/goncho/service_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestSearchSessions_IncludesParentSessionMetadataWhenPresent(t *testing.T) {}
func TestSearchMessages_NormalizesSplitSourceAndCombinedChatKey(t *testing.T) {}
func TestService_SearchUserScopeReturnsLineageAwareSessionKeys(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/memory ./internal/goncho -run 'Test(SearchSessions_IncludesParentSessionMetadataWhenPresent|SearchMessages_NormalizesSplitSourceAndCombinedChatKey|Service_SearchUserScopeReturnsLineageAwareSessionKeys)' -count=1`

Expected: FAIL because session search hits are not yet lineage-aware and the helper contract does not yet prove normalization behavior explicitly.

- [ ] **Step 3: Write the minimal implementation**

```go
type SessionSearchHit struct {
	SessionID       string
	ChatID          string
	Source          string
	ParentSessionID string
	LineageKind     string
	LatestTurnUnix  int64
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/memory ./internal/goncho -run 'Test(SearchSessions_IncludesParentSessionMetadataWhenPresent|SearchMessages_NormalizesSplitSourceAndCombinedChatKey|Service_SearchUserScopeReturnsLineageAwareSessionKeys)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/memory/session_catalog.go internal/memory/session_catalog_test.go internal/goncho/service.go internal/goncho/service_test.go
git commit -m "feat: make cross-chat search lineage-aware"
```

### Task 6: operator evidence and closeout gate

**Files:**
- Modify: `internal/memory/status.go`
- Modify: `internal/memory/status_test.go`
- Modify: `cmd/gormes/memory.go`
- Modify: `internal/session/index_mirror.go`
- Modify docs under `docs/content/building-gormes/architecture_plan/`

- [ ] **Step 1: Write the failing tests**

```go
func TestReadExtractorStatus_IncludesIdentityAndLineageSummary(t *testing.T) {}
func TestFormatExtractorStatus_ReportsFreshnessAndOrphanCounts(t *testing.T) {}
func TestSessionIndexMirror_WriteIncludesUserAndLineageAuditFields(t *testing.T) {}
```

- [ ] **Step 2: Run the RED tests**

Run: `go test ./internal/memory ./cmd/gormes ./internal/session -run 'Test(ReadExtractorStatus_IncludesIdentityAndLineageSummary|FormatExtractorStatus_ReportsFreshnessAndOrphanCounts|SessionIndexMirror_WriteIncludesUserAndLineageAuditFields)' -count=1`

Expected: FAIL because the operator surfaces currently only expose extractor health and raw session mappings.

- [ ] **Step 3: Write the minimal implementation**

```go
type IdentityLineageStatus struct {
	MissingLastSeen     int
	ResolvedSessions    int
	UnresolvedSessions  int
	OrphanLineageChains int
}
```

- [ ] **Step 4: Run the GREEN tests**

Run: `go test ./internal/memory ./cmd/gormes ./internal/session -run 'Test(ReadExtractorStatus_IncludesIdentityAndLineageSummary|FormatExtractorStatus_ReportsFreshnessAndOrphanCounts|SessionIndexMirror_WriteIncludesUserAndLineageAuditFields)' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/memory/status.go internal/memory/status_test.go cmd/gormes/memory.go internal/session/index_mirror.go docs/content/building-gormes/architecture_plan
git commit -m "feat: expose identity and lineage evidence to operators"
```

## 7. Operator observability plan

Minimum operator-readable evidence before any of these items move to complete:

- `gormes memory status`
  - count of relationships missing `last_seen`
  - count of sessions with resolved `user_id`
  - count of sessions missing `user_id`
  - count of lineage orphans
  - explicit statement that cross-chat is still opt-in
- `sessions/index.yaml`
  - `source`
  - `chat_id`
  - `user_id`
  - `parent_session_id`
  - `lineage_kind`
- docs/progress
  - status notes explain what is landed versus what still gates completion

## 8. Rollback and failure containment

### Rollback

- `last_seen` migration is additive; if the write path regresses, fall back to `updated_at` in recall and stop promoting the subphase.
- lineage fields in `session.Metadata` are additive; old sessions remain valid roots.
- scope/source fields in GONCHO/Honcho contracts are optional; callers that omit them keep same-chat behavior.

### Failure containment

- unresolved user lookup => same-chat fallback, never global
- empty source filter resolution => same-chat fallback, never global
- orphan parent session => visible in status/mirror, never fatal
- missing `last_seen` backfill => visible in status; decay must continue using the fallback expression until counts hit zero

## 9. Definition of Done

These sub-items are done only when all of the following are true:

- `3.E.6.1`
  - `relationships.last_seen` exists
  - backfill is deterministic
  - recall decay uses `last_seen` when present
- `3.E.7.2`
  - same-chat default is locked for exact, FTS, and semantic paths
  - user-scope recall/search requires explicit opt-in
  - `honcho_search` and `honcho_context` schemas advertise `scope` and `sources`
- `3.E.8.1`
  - `session.Metadata` carries `parent_session_id` and `lineage_kind`
  - session mirror renders lineage safely
  - self-parent and trivial loop cases are rejected
- `3.E.8.2`
  - session/message search is source-filtered and deterministic
  - session hits are lineage-aware
  - operator surfaces can prove what search scope was allowed

## 10. Evidence checklist

Run these commands before marking the delivery complete:

```bash
go test ./internal/memory ./internal/session ./internal/goncho -count=1
go test ./cmd/gormes -count=1
go test ./internal/progress -count=1
go run ./cmd/progress-gen -write
go run ./cmd/progress-gen -validate
go test ./docs -count=1
```

The subphase does not move to complete until all commands above are green and the operator evidence surfaces are manually reviewed.
