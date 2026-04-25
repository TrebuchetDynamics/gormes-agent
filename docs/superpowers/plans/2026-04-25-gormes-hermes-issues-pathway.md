# Gormes Hermes-Issues Pathway Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an optional `hermes-issues` pathway to `cmd/architecture-planner-loop` that periodically downloads issues from `NousResearch/hermes-agent`, rule-filters ~2400 issues down to ~50–150 high-signal candidates, writes a ranked markdown digest, extracts a grounded keyword vocabulary, and surfaces both in the planner prompt so the LLM can pick the ~20 worth turning into Gormes roadmap rows.

**Architecture:** New isolated subpackage `internal/hermesissues` owns fetch, filter, rank, digest, and keyword extraction. CLI gains a peer subcommand `hermes-issues` that refreshes the cache. The existing `run` subcommand opportunistically reads digest + suggested keywords from the cache directory (no hard dependency — pathway stays optional). The wrapper script gates the refresh behind `HERMES_ISSUES_PATHWAY=1`. Filtering is two-stage rule-based (scope → signal) with reaction-weighted ranking; final issue selection is delegated to the planner LLM via the prompt. **The pathway reinforces the planner's existing `Keywords` feature in two directions:** (1) `Refresh` *generates* `suggested_keywords.json` — labels and top-frequency title nouns extracted from high-signal items — surfaced as a grounded vocabulary the LLM may honor, fixing the disuse pain (the systemd timer never supplies keywords, so the topical clause is dormant); (2) when the user *does* supply keywords positionally, those keywords additionally narrow the digest section so the topical model is coherent across internal context and upstream-issue signal.

**Tech Stack:** Go 1.22, stdlib `net/http`/`encoding/json`/`time`, existing `internal/architectureplanner` integration points, no new third-party deps. GitHub REST API v3 (`/repos/{owner}/{repo}/issues`) with optional `GITHUB_TOKEN` auth.

**Source spec:** Conversation 2026-04-25 — strangler-fig signal extraction, target ~20 useful issues from ~2400, plus reinforcement of the dormant `Keywords` feature.

---

## Prerequisites

- `cmd/architecture-planner-loop` is the long-term planning entry (see its README).
- `internal/architectureplanner.ContextBundle` is the integration surface for new prompt context.
- `scripts/architecture-planner-loop.sh` is the wrapper invoked by the systemd timer.
- `.codex/architecture-planner/` is the existing artifact root (overridden by `RUN_ROOT`).
- Anonymous GitHub API allows 60 req/hr; `GITHUB_TOKEN` raises this to 5000. ~2400 issues = ~24 pages = fine cold-start anonymously, but iterative work needs a token.

## Existing-Keyword-Feature Pain Analysis (reinforce, don't fork)

Surveyed before adding adjacent code:

- **`run [keyword ...]`** — positional args feed `architectureplanner.RunOptions.Keywords`.
- **`FilterContextByKeywords`** (`internal/architectureplanner/topics.go`) narrows `QuarantinedRows` and `ImplInventory.{GormesOriginalPaths,RecentlyChanged,OwnedSubphases}` by case-insensitive substring match against item Name, Contract, Fixture, SourceRefs, WriteScope, parent phase/subphase names.
- **`topicalClauseTemplate`** (`internal/architectureplanner/prompt.go:113`) appends a `TOPICAL FOCUS` block instructing the LLM to honor scope.

**Three pains:**

1. **Disuse.** The systemd timer never passes keywords, so the topical mechanism is effectively dead in autonomous mode.
2. **Substring-only matching.** `auth` misses `OAuth`, `streaming` misses `SSE`. Mechanical, not semantic.
3. **No grounded vocabulary.** Keywords are 100% user-supplied; no source-of-truth for which terms matter.

**Mitigation in this plan (no separate feature, no rewrite of the keyword filter):**

- Pain #1 (disuse) → `Refresh` writes `suggested_keywords.json` extracted from high-signal Hermes issues. Prompt surfaces these as suggestions the LLM may apply. Systemd timer now produces signal-grounded suggestions every run.
- Pain #3 (no grounded vocabulary) → suggested keywords come from labels and title-noun frequency over the top-N digest items, by construction terms users care about.
- Pain #2 (substring matching) → out of scope. Substring is fine for label-derived terms; if production shows false negatives we layer fuzzy matching as a follow-up. Doing it now is YAGNI.

**Coherence:** when user *also* passes positional keywords, those keywords additionally filter the digest section (Task 9b). Same vocabulary narrows internal context (existing) and upstream signal (new) — topical clause has consistent meaning end-to-end.

## File Structure Map

```
gormes-agent/
├── internal/
│   └── hermesissues/
│       ├── types.go                # NEW — Issue, Reactions, Filters, RankedIssue, KeywordCandidate, Digest
│       ├── client.go               # NEW — GitHub client (pagination)
│       ├── client_test.go          # NEW — httptest client tests
│       ├── filter.go               # NEW — scope+signal filters, FilterDigestByKeywords
│       ├── filter_test.go          # NEW — table-driven filter tests
│       ├── rank.go                 # NEW — Score, Rank
│       ├── rank_test.go            # NEW
│       ├── digest.go               # NEW — RenderDigest markdown writer
│       ├── digest_test.go          # NEW — golden digest test
│       ├── keywords.go             # NEW — ExtractKeywords from ranked items
│       ├── keywords_test.go        # NEW
│       ├── refresh.go              # NEW — Refresh orchestrator + readers
│       └── refresh_test.go         # NEW — end-to-end orchestrator test
├── internal/architectureplanner/
│   ├── context.go                  # MODIFY — HermesIssuesDigest + HermesSuggestedKeywords on ContextBundle
│   ├── context_test.go             # MODIFY — passthrough tests
│   ├── prompt.go                   # MODIFY — emit hermes section + suggested-keywords section
│   └── prompt_test.go              # MODIFY — assert sections appear/omit
├── cmd/architecture-planner-loop/
│   ├── main.go                     # MODIFY — hermes-issues subcommand
│   ├── main_test.go                # MODIFY — subcommand wiring
│   └── README.md                   # MODIFY — document the pathway
└── scripts/
    └── architecture-planner-loop.sh # MODIFY — gate refresh on HERMES_ISSUES_PATHWAY=1
```

No changes to `internal/autoloop`, `cmd/autoloop`, `pkg/`, `docs/content/`, or `www.gormes.ai`.

---

## Task 1: Scaffold types

**Files:** Create `internal/hermesissues/types.go`.

- [ ] **Step 1:** Create `internal/hermesissues/types.go`:

```go
// Package hermesissues fetches, filters, ranks, and digests GitHub issues
// from upstream Hermes (NousResearch/hermes-agent) so the architecture
// planner can mine them for Gormes feature/pain signals.
package hermesissues

import "time"

type Issue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	StateReason string    `json:"state_reason"`
	HTMLURL     string    `json:"html_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClosedAt    time.Time `json:"closed_at,omitempty"`
	Labels      []string  `json:"labels"`
	Comments    int       `json:"comments"`
	Reactions   Reactions `json:"reactions"`
	IsPullReq   bool      `json:"is_pull_request"`
	AuthorAssoc string    `json:"author_association"`
}

type Reactions struct {
	Total    int `json:"total_count"`
	PlusOne  int `json:"+1"`
	MinusOne int `json:"-1"`
	Heart    int `json:"heart"`
	Hooray   int `json:"hooray"`
	Rocket   int `json:"rocket"`
	Eyes     int `json:"eyes"`
	Laugh    int `json:"laugh"`
	Confused int `json:"confused"`
}

type Filters struct {
	MaxAgeMonths           int
	IncludeClosed          bool
	ExcludeStateNotPlanned bool
	MinReactionsTotal      int
	MinComments            int
	NoiseLabels            []string
	BoostLabels            []string
	RequireBodyChars       int
}

func DefaultFilters() Filters {
	return Filters{
		MaxAgeMonths:           18,
		IncludeClosed:          true,
		ExcludeStateNotPlanned: true,
		MinReactionsTotal:      1,
		MinComments:            1,
		NoiseLabels:            []string{"duplicate", "wontfix", "invalid", "question", "spam", "stale", "needs-info"},
		BoostLabels:            []string{"enhancement", "feature", "feature-request", "good first issue", "help wanted", "rfc"},
		RequireBodyChars:       40,
	}
}

type RankedIssue struct {
	Issue     Issue   `json:"issue"`
	Score     float64 `json:"score"`
	Rationale string  `json:"rationale"`
}

type KeywordCandidate struct {
	Term   string `json:"term"`
	Source string `json:"source"` // "label" | "title"
	Hits   int    `json:"hits"`
}

type Digest struct {
	GeneratedAt       time.Time          `json:"generated_at"`
	SourceRepo        string             `json:"source_repo"`
	TotalSeen         int                `json:"total_seen"`
	AfterScope        int                `json:"after_scope_filter"`
	AfterSignal       int                `json:"after_signal_filter"`
	TopN              int                `json:"top_n"`
	Items             []RankedIssue      `json:"items"`
	SuggestedKeywords []KeywordCandidate `json:"suggested_keywords,omitempty"`
}
```

- [ ] **Step 2:** `go build ./internal/hermesissues/` → exit 0.
- [ ] **Step 3:** Commit `feat(hermesissues): scaffold core types`.

---

## Task 2: GitHub client — single page fetch

**Files:** Create `internal/hermesissues/client.go`, `internal/hermesissues/client_test.go`.

- [ ] **Step 1:** Create `client_test.go`:

```go
package hermesissues

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_FetchPage_ParsesIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept = %q", got)
		}
		if r.URL.Query().Get("state") != "all" {
			t.Errorf("state query missing")
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"number": 42, "title": "Add streaming", "body": "we need SSE for tool output",
			"state": "open", "state_reason": nil,
			"html_url":   "https://github.com/NousResearch/hermes-agent/issues/42",
			"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-04-01T00:00:00Z",
			"comments":           7,
			"labels":             []map[string]any{{"name": "enhancement"}},
			"reactions":          map[string]any{"total_count": 12, "+1": 9, "heart": 2, "rocket": 1},
			"author_association": "CONTRIBUTOR",
		}})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "x", "y", "")
	page, _, err := c.fetchPage(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].Number != 42 || page[0].Reactions.PlusOne != 9 {
		t.Fatalf("bad parse: %+v", page)
	}
	if page[0].UpdatedAt.Year() != 2026 || page[0].UpdatedAt.Month() != time.April {
		t.Errorf("UpdatedAt = %v", page[0].UpdatedAt)
	}
	if len(page[0].Labels) != 1 || page[0].Labels[0] != "enhancement" {
		t.Errorf("labels = %v", page[0].Labels)
	}
	if page[0].IsPullReq {
		t.Errorf("IsPullReq must be false (no pull_request key)")
	}
}

func TestClient_FetchPage_FlagsPullRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"number":1,"title":"PR","state":"open","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","pull_request":{"url":"x"}}]`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "x", "y", "")
	page, _, err := c.fetchPage(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !page[0].IsPullReq {
		t.Fatalf("IsPullReq must be true")
	}
}

func TestClient_FetchPage_AuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "x", "y", "secret")
	if _, _, err := c.fetchPage(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
}

func TestClient_FetchPage_StatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"rate limited"}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "x", "y", "")
	_, _, err := c.fetchPage(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("want 403 error, got %v", err)
	}
}
```

- [ ] **Step 2:** Run — fails (`undefined: NewClient`).

```bash
go test ./internal/hermesissues/ -run TestClient_FetchPage -v
```

- [ ] **Step 3:** Create `internal/hermesissues/client.go`:

```go
package hermesissues

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL  string
	owner    string
	repo     string
	token    string
	http     *http.Client
	MaxPages int
}

func NewClient(baseURL, owner, repo, token string) *Client {
	return &Client{
		baseURL:  baseURL,
		owner:    owner,
		repo:     repo,
		token:    token,
		http:     &http.Client{Timeout: 30 * time.Second},
		MaxPages: 60,
	}
}

func (c *Client) fetchPage(ctx context.Context, page int) ([]Issue, bool, error) {
	u, err := url.Parse(fmt.Sprintf("%s/repos/%s/%s/issues", c.baseURL, c.owner, c.repo))
	if err != nil {
		return nil, false, err
	}
	q := u.Query()
	q.Set("state", "all")
	q.Set("per_page", "100")
	q.Set("page", strconv.Itoa(page))
	q.Set("sort", "updated")
	q.Set("direction", "desc")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "gormes-architecture-planner")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	hasNext := linkHasNext(resp.Header.Get("Link"))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, hasNext, fmt.Errorf("github api %d: %s", resp.StatusCode, string(body))
	}

	var raw []rawIssue
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, hasNext, fmt.Errorf("decode issues: %w", err)
	}
	out := make([]Issue, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.toIssue())
	}
	return out, hasNext, nil
}

func linkHasNext(link string) bool {
	if link == "" {
		return false
	}
	for _, part := range strings.Split(link, ",") {
		if strings.Contains(part, `rel="next"`) {
			return true
		}
	}
	return false
}

type rawIssue struct {
	Number      int              `json:"number"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	State       string           `json:"state"`
	StateReason *string          `json:"state_reason"`
	HTMLURL     string           `json:"html_url"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	ClosedAt    *time.Time       `json:"closed_at"`
	Comments    int              `json:"comments"`
	Labels      []rawLabel       `json:"labels"`
	Reactions   *rawReactions    `json:"reactions"`
	PullReq     *json.RawMessage `json:"pull_request"`
	AuthorAssoc string           `json:"author_association"`
}

type rawLabel struct {
	Name string `json:"name"`
}

type rawReactions struct {
	Total    int `json:"total_count"`
	PlusOne  int `json:"+1"`
	MinusOne int `json:"-1"`
	Heart    int `json:"heart"`
	Hooray   int `json:"hooray"`
	Rocket   int `json:"rocket"`
	Eyes     int `json:"eyes"`
	Laugh    int `json:"laugh"`
	Confused int `json:"confused"`
}

func (r rawIssue) toIssue() Issue {
	out := Issue{
		Number:      r.Number,
		Title:       r.Title,
		Body:        r.Body,
		State:       r.State,
		HTMLURL:     r.HTMLURL,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Comments:    r.Comments,
		IsPullReq:   r.PullReq != nil,
		AuthorAssoc: r.AuthorAssoc,
	}
	if r.StateReason != nil {
		out.StateReason = *r.StateReason
	}
	if r.ClosedAt != nil {
		out.ClosedAt = *r.ClosedAt
	}
	for _, l := range r.Labels {
		out.Labels = append(out.Labels, l.Name)
	}
	if r.Reactions != nil {
		out.Reactions = Reactions(*r.Reactions)
	}
	return out
}
```

Note the test signature `c.fetchPage(ctx, 1)` returns `([]Issue, bool, error)` — the second return is `hasNext`. Update the test calls to use `_` for the bool placeholder if compiler complains.

- [ ] **Step 4:** Run client tests → all PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): single-page GitHub fetch + PR detection`.

---

## Task 3: Pagination — `FetchAll`

**Files:** Modify `internal/hermesissues/client.go`, `internal/hermesissues/client_test.go`.

- [ ] **Step 1:** Add tests:

```go
func TestClient_FetchAll_FollowsPagination(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Query().Get("page") {
		case "1":
			w.Header().Set("Link", `<x?page=2>; rel="next", <x?page=3>; rel="last"`)
			_, _ = w.Write([]byte(`[{"number":1,"title":"a","state":"open","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`))
		case "2":
			w.Header().Set("Link", `<x?page=3>; rel="next"`)
			_, _ = w.Write([]byte(`[{"number":2,"title":"b","state":"open","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`))
		case "3":
			_, _ = w.Write([]byte(`[{"number":3,"title":"c","state":"open","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`))
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "x", "y", "")
	all, err := c.FetchAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 || len(all) != 3 {
		t.Fatalf("hits=%d len=%d", hits, len(all))
	}
}

func TestClient_FetchAll_HitsMaxPagesCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<x>; rel="next"`)
		_, _ = w.Write([]byte(`[{"number":1,"title":"x","state":"open","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "x", "y", "")
	c.MaxPages = 3
	_, err := c.FetchAll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "MaxPages") {
		t.Fatalf("want MaxPages cap error, got %v", err)
	}
}
```

- [ ] **Step 2:** Run — fails on `undefined: FetchAll`.
- [ ] **Step 3:** Append to `client.go`:

```go
func (c *Client) FetchAll(ctx context.Context) ([]Issue, error) {
	var all []Issue
	for page := 1; page <= c.MaxPages; page++ {
		issues, hasNext, err := c.fetchPage(ctx, page)
		if err != nil {
			return nil, err
		}
		all = append(all, issues...)
		if !hasNext {
			return all, nil
		}
	}
	return all, fmt.Errorf("FetchAll: hit MaxPages=%d safety cap", c.MaxPages)
}
```

- [ ] **Step 4:** Run all client tests → PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): paginated FetchAll with Link walk`.

---

## Task 4: Scope filter

**Files:** Create `internal/hermesissues/filter.go`, `internal/hermesissues/filter_test.go`.

- [ ] **Step 1:** Tests:

```go
package hermesissues

import (
	"testing"
	"time"
)

func TestScopeFilter_DropsPullRequests(t *testing.T) {
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	body := "long enough body to pass the minimum threshold for scope filtering here"
	in := []Issue{
		{Number: 1, Body: body, State: "open", UpdatedAt: now},
		{Number: 2, Body: body, State: "open", UpdatedAt: now, IsPullReq: true},
	}
	out := ScopeFilter(in, DefaultFilters(), now)
	if len(out) != 1 || out[0].Number != 1 {
		t.Fatalf("got %+v", out)
	}
}

func TestScopeFilter_RecencyWindow(t *testing.T) {
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	body := "long enough body for scope filtering minimum threshold check, no problem"
	in := []Issue{
		{Number: 1, Body: body, State: "open", UpdatedAt: now.AddDate(0, -24, 0)},
		{Number: 2, Body: body, State: "open", UpdatedAt: now.AddDate(0, -3, 0)},
	}
	out := ScopeFilter(in, DefaultFilters(), now)
	if len(out) != 1 || out[0].Number != 2 {
		t.Fatalf("got %+v", out)
	}
}

func TestScopeFilter_DropsClosedNotPlanned(t *testing.T) {
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	body := "long enough body to pass the scope filter minimum threshold check pls"
	in := []Issue{
		{Number: 1, Body: body, State: "closed", StateReason: "not_planned", UpdatedAt: now},
		{Number: 2, Body: body, State: "closed", StateReason: "completed", UpdatedAt: now},
	}
	out := ScopeFilter(in, DefaultFilters(), now)
	if len(out) != 1 || out[0].Number != 2 {
		t.Fatalf("got %+v", out)
	}
}

func TestScopeFilter_DropsShortBodies(t *testing.T) {
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	in := []Issue{
		{Number: 1, Body: "this body is plenty long to clear the minimum length floor", State: "open", UpdatedAt: now},
		{Number: 2, Body: "yes", State: "open", UpdatedAt: now},
	}
	out := ScopeFilter(in, DefaultFilters(), now)
	if len(out) != 1 || out[0].Number != 1 {
		t.Fatalf("got %+v", out)
	}
}
```

- [ ] **Step 2:** Run — fails on `undefined: ScopeFilter`.
- [ ] **Step 3:** Create `filter.go`:

```go
package hermesissues

import (
	"strings"
	"time"
)

func ScopeFilter(in []Issue, f Filters, now time.Time) []Issue {
	out := make([]Issue, 0, len(in))
	cutoff := now.AddDate(0, -f.MaxAgeMonths, 0)
	for _, it := range in {
		if it.IsPullReq {
			continue
		}
		if !f.IncludeClosed && it.State == "closed" {
			continue
		}
		if f.ExcludeStateNotPlanned && it.State == "closed" && it.StateReason == "not_planned" {
			continue
		}
		if it.UpdatedAt.Before(cutoff) {
			continue
		}
		if len(it.Body) < f.RequireBodyChars {
			continue
		}
		out = append(out, it)
	}
	return out
}

func lowerSet(in []string) map[string]struct{} {
	m := make(map[string]struct{}, len(in))
	for _, s := range in {
		m[strings.ToLower(s)] = struct{}{}
	}
	return m
}

func hasAny(labels []string, set map[string]struct{}) bool {
	for _, l := range labels {
		if _, ok := set[strings.ToLower(l)]; ok {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4:** Run scope tests → PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): ScopeFilter (PR/recency/state-reason/body)`.

---

## Task 5: Signal filter

**Files:** Modify `internal/hermesissues/filter.go`, `internal/hermesissues/filter_test.go`.

- [ ] **Step 1:** Tests:

```go
func TestSignalFilter_RequiresEngagementOrBoost(t *testing.T) {
	in := []Issue{
		{Number: 1, Body: "x", Comments: 0, Reactions: Reactions{Total: 0}},
		{Number: 2, Body: "x", Comments: 3},
		{Number: 3, Body: "x", Reactions: Reactions{Total: 5}},
		{Number: 4, Body: "x", Labels: []string{"feature-request"}},
	}
	out := SignalFilter(in, DefaultFilters())
	if len(out) != 3 {
		t.Fatalf("expected 3 (#2 comments, #3 reactions, #4 boost), got %+v", out)
	}
}

func TestSignalFilter_DropsNoiseLabels(t *testing.T) {
	in := []Issue{
		{Number: 1, Body: "x", Comments: 5, Labels: []string{"enhancement"}},
		{Number: 2, Body: "x", Comments: 5, Labels: []string{"Duplicate"}},
		{Number: 3, Body: "x", Comments: 5, Labels: []string{"question"}},
	}
	out := SignalFilter(in, DefaultFilters())
	if len(out) != 1 || out[0].Number != 1 {
		t.Fatalf("got %+v", out)
	}
}
```

- [ ] **Step 2:** Run — fails.
- [ ] **Step 3:** Append to `filter.go`:

```go
func SignalFilter(in []Issue, f Filters) []Issue {
	noise := lowerSet(f.NoiseLabels)
	boost := lowerSet(f.BoostLabels)
	out := make([]Issue, 0, len(in))
	for _, it := range in {
		if hasAny(it.Labels, noise) {
			continue
		}
		hasBoost := hasAny(it.Labels, boost)
		hasEng := it.Reactions.Total >= f.MinReactionsTotal || it.Comments >= f.MinComments
		if !hasBoost && !hasEng {
			continue
		}
		out = append(out, it)
	}
	return out
}
```

- [ ] **Step 4:** Run signal tests → PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): SignalFilter (engagement + noise labels)`.

---

## Task 6: Score & rank

**Files:** Create `internal/hermesissues/rank.go`, `internal/hermesissues/rank_test.go`.

- [ ] **Step 1:** Tests:

```go
package hermesissues

import (
	"testing"
	"time"
)

func TestScore_ReactionsBeatComments(t *testing.T) {
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	chatter := Issue{Comments: 80, UpdatedAt: now}
	endorsed := Issue{Reactions: Reactions{PlusOne: 10}, UpdatedAt: now}
	if Score(endorsed, DefaultFilters(), now) <= Score(chatter, DefaultFilters(), now) {
		t.Fatalf("endorsed must beat chatter")
	}
}

func TestScore_BoostLabelMatters(t *testing.T) {
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	plain := Issue{Reactions: Reactions{PlusOne: 1}, UpdatedAt: now}
	boosted := Issue{Reactions: Reactions{PlusOne: 1}, Labels: []string{"feature-request"}, UpdatedAt: now}
	if Score(boosted, DefaultFilters(), now) <= Score(plain, DefaultFilters(), now) {
		t.Fatalf("boost label must raise score")
	}
}

func TestScore_RecencyDecay(t *testing.T) {
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	fresh := Issue{Reactions: Reactions{PlusOne: 5}, UpdatedAt: now.AddDate(0, -3, 0)}
	stale := Issue{Reactions: Reactions{PlusOne: 5}, UpdatedAt: now.AddDate(0, -15, 0)}
	if Score(fresh, DefaultFilters(), now) <= Score(stale, DefaultFilters(), now) {
		t.Fatalf("fresh must outscore stale")
	}
}

func TestRank_OrdersByScoreDesc(t *testing.T) {
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	in := []Issue{
		{Number: 1, Reactions: Reactions{PlusOne: 1}, UpdatedAt: now},
		{Number: 2, Reactions: Reactions{PlusOne: 9}, UpdatedAt: now},
		{Number: 3, Reactions: Reactions{PlusOne: 4}, UpdatedAt: now},
	}
	got := Rank(in, DefaultFilters(), now)
	if got[0].Issue.Number != 2 || got[1].Issue.Number != 3 || got[2].Issue.Number != 1 {
		t.Fatalf("order = %d/%d/%d", got[0].Issue.Number, got[1].Issue.Number, got[2].Issue.Number)
	}
	if got[0].Rationale == "" {
		t.Fatalf("rationale must be set")
	}
}
```

- [ ] **Step 2:** Run — fails.
- [ ] **Step 3:** Create `rank.go`:

```go
package hermesissues

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	weightPlusOne     = 3.0
	weightStrongReact = 2.0
	weightEyes        = 1.0
	weightCommentEach = 0.5
	commentCap        = 40
	boostLabelBonus   = 5.0
)

func Score(it Issue, f Filters, now time.Time) float64 {
	r := it.Reactions
	s := float64(r.PlusOne)*weightPlusOne +
		float64(r.Heart+r.Hooray+r.Rocket)*weightStrongReact +
		float64(r.Eyes)*weightEyes
	c := it.Comments
	if c > commentCap {
		c = commentCap
	}
	s += float64(c) * weightCommentEach
	if hasAny(it.Labels, lowerSet(f.BoostLabels)) {
		s += boostLabelBonus
	}
	return s * recencyMultiplier(it.UpdatedAt, now)
}

func recencyMultiplier(updated, now time.Time) float64 {
	m := monthsBetween(updated, now)
	switch {
	case m <= 6:
		return 1.0
	case m <= 12:
		return 0.7
	case m <= 18:
		return 0.4
	default:
		return 0.2
	}
}

func monthsBetween(a, b time.Time) int {
	if a.After(b) {
		a, b = b, a
	}
	return (b.Year()-a.Year())*12 + int(b.Month()) - int(a.Month())
}

func Rank(in []Issue, f Filters, now time.Time) []RankedIssue {
	out := make([]RankedIssue, 0, len(in))
	for _, it := range in {
		out = append(out, RankedIssue{
			Issue:     it,
			Score:     Score(it, f, now),
			Rationale: rationaleFor(it, f),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Issue.Number < out[j].Issue.Number
	})
	return out
}

func rationaleFor(it Issue, f Filters) string {
	parts := []string{}
	r := it.Reactions
	if r.PlusOne > 0 {
		parts = append(parts, fmt.Sprintf("👍 %d", r.PlusOne))
	}
	if r.Heart+r.Hooray+r.Rocket > 0 {
		parts = append(parts, fmt.Sprintf("strong %d", r.Heart+r.Hooray+r.Rocket))
	}
	if it.Comments > 0 {
		parts = append(parts, fmt.Sprintf("💬 %d", it.Comments))
	}
	boost := lowerSet(f.BoostLabels)
	for _, l := range it.Labels {
		if _, ok := boost[strings.ToLower(l)]; ok {
			parts = append(parts, "label:"+l)
		}
	}
	if it.State == "closed" && it.StateReason == "completed" {
		parts = append(parts, "closed-completed")
	}
	if len(parts) == 0 {
		return "(no notable signal)"
	}
	return strings.Join(parts, ", ")
}
```

- [ ] **Step 4:** Run rank tests → PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): deterministic Score+Rank with rationales`.

---

## Task 7: Markdown digest

**Files:** Create `internal/hermesissues/digest.go`, `internal/hermesissues/digest_test.go`.

- [ ] **Step 1:** Tests:

```go
package hermesissues

import (
	"strings"
	"testing"
	"time"
)

func TestRenderDigest_HeaderAndItems(t *testing.T) {
	d := Digest{
		GeneratedAt: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		SourceRepo:  "NousResearch/hermes-agent",
		TotalSeen:   2400, AfterScope: 300, AfterSignal: 80, TopN: 2,
		Items: []RankedIssue{
			{Issue: Issue{Number: 42, Title: "Streaming", HTMLURL: "https://github.com/NousResearch/hermes-agent/issues/42", State: "open", Labels: []string{"enhancement"}}, Score: 27.0, Rationale: "👍 9, 💬 7"},
			{Issue: Issue{Number: 88, Title: "Retry", HTMLURL: "https://github.com/NousResearch/hermes-agent/issues/88", State: "closed", StateReason: "completed"}, Score: 14.5, Rationale: "👍 4"},
		},
	}
	got := RenderDigest(d)
	for _, want := range []string{
		"# Hermes Upstream Issue Signal Digest",
		"NousResearch/hermes-agent", "Total seen: 2400",
		"After scope filter: 300", "After signal filter: 80", "Top N shown: 2",
		"#42", "Streaming", "score 27.0", "https://github.com/NousResearch/hermes-agent/issues/42",
		"#88", "closed-completed-state", // we render state as closed/completed
	} {
		if want == "closed-completed-state" {
			if !strings.Contains(got, "closed/completed") {
				t.Errorf("missing closed/completed marker: %s", got)
			}
			continue
		}
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderDigest_Empty(t *testing.T) {
	d := Digest{SourceRepo: "x/y", GeneratedAt: time.Now()}
	if !strings.Contains(RenderDigest(d), "(no candidates after filtering)") {
		t.Fatalf("expected empty marker")
	}
}
```

- [ ] **Step 2:** Run — fails.
- [ ] **Step 3:** Create `digest.go`:

```go
package hermesissues

import (
	"fmt"
	"strings"
)

func RenderDigest(d Digest) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Hermes Upstream Issue Signal Digest")
	fmt.Fprintf(&b, "Generated: %s\n", d.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"))
	fmt.Fprintf(&b, "Source repo: %s\n\n", d.SourceRepo)
	fmt.Fprintln(&b, "Funnel:")
	fmt.Fprintf(&b, "- Total seen: %d\n", d.TotalSeen)
	fmt.Fprintf(&b, "- After scope filter: %d\n", d.AfterScope)
	fmt.Fprintf(&b, "- After signal filter: %d\n", d.AfterSignal)
	fmt.Fprintf(&b, "- Top N shown: %d\n\n", d.TopN)
	fmt.Fprintln(&b, "Planner instruction: scan the items below and lift the ~20 most relevant pain points / feature requests into Gormes roadmap rows. Skip items already covered; cite the issue URL in `source_refs` when promoting one.")
	fmt.Fprintln(&b)

	if len(d.Items) == 0 {
		fmt.Fprintln(&b, "(no candidates after filtering)")
		return b.String()
	}

	for _, ri := range d.Items {
		labels := ""
		if len(ri.Issue.Labels) > 0 {
			labels = " [" + strings.Join(ri.Issue.Labels, ", ") + "]"
		}
		state := ri.Issue.State
		if ri.Issue.State == "closed" && ri.Issue.StateReason != "" {
			state = "closed/" + ri.Issue.StateReason
		}
		fmt.Fprintf(&b, "## #%d %s\n", ri.Issue.Number, ri.Issue.Title)
		fmt.Fprintf(&b, "- score %.1f — %s\n", ri.Score, ri.Rationale)
		fmt.Fprintf(&b, "- state: %s%s\n", state, labels)
		fmt.Fprintf(&b, "- url: %s\n", ri.Issue.HTMLURL)
		body := strings.TrimSpace(ri.Issue.Body)
		if len(body) > 480 {
			body = body[:480] + "…"
		}
		if body != "" {
			fmt.Fprintf(&b, "- excerpt: %s\n", strings.ReplaceAll(body, "\n", " "))
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}
```

- [ ] **Step 4:** Run digest tests → PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): RenderDigest markdown writer`.

---

## Task 7b: Keyword extraction

**Files:** Create `internal/hermesissues/keywords.go`, `internal/hermesissues/keywords_test.go`.

This task closes pain #1 (the planner's `Keywords` feature is dormant in autonomous mode) by mining a grounded vocabulary from ranked items.

- [ ] **Step 1:** Tests:

```go
package hermesissues

import "testing"

func TestExtractKeywords_LabelsDedupedCaseInsensitive(t *testing.T) {
	items := []RankedIssue{
		{Issue: Issue{Title: "x", Labels: []string{"enhancement", "Streaming"}}},
		{Issue: Issue{Title: "y", Labels: []string{"streaming", "tools"}}},
		{Issue: Issue{Title: "z", Labels: []string{"duplicate"}}},
	}
	out := ExtractKeywords(items, DefaultFilters(), 25)
	got := map[string]int{}
	for _, kw := range out {
		if kw.Source == "label" {
			got[kw.Term] = kw.Hits
		}
	}
	if got["streaming"] != 2 {
		t.Errorf("streaming hits = %d, want 2", got["streaming"])
	}
	if _, ok := got["duplicate"]; ok {
		t.Errorf("duplicate is a noise label, must be dropped")
	}
}

func TestExtractKeywords_TitleNounsRequireFreq2(t *testing.T) {
	items := []RankedIssue{
		{Issue: Issue{Title: "Add streaming for tool output"}},
		{Issue: Issue{Title: "Streaming retry on disconnect"}},
		{Issue: Issue{Title: "Fix tool output parser"}},
	}
	out := ExtractKeywords(items, DefaultFilters(), 25)
	titleHits := map[string]int{}
	for _, kw := range out {
		if kw.Source == "title" {
			titleHits[kw.Term] = kw.Hits
		}
	}
	for _, want := range []string{"streaming", "tool", "output"} {
		if titleHits[want] < 2 {
			t.Errorf("%s freq = %d, want ≥2", want, titleHits[want])
		}
	}
	for _, drop := range []string{"add", "for", "on", "the"} {
		if _, ok := titleHits[drop]; ok {
			t.Errorf("stopword %q must be dropped", drop)
		}
	}
}

func TestExtractKeywords_DropsSingletons(t *testing.T) {
	out := ExtractKeywords([]RankedIssue{{Issue: Issue{Title: "completely unique words appear once"}}}, DefaultFilters(), 25)
	for _, kw := range out {
		if kw.Source == "title" {
			t.Errorf("singleton %q must be dropped", kw.Term)
		}
	}
}

func TestExtractKeywords_MaxCap(t *testing.T) {
	items := []RankedIssue{}
	for i := 0; i < 50; i++ {
		items = append(items, RankedIssue{Issue: Issue{Labels: []string{
			"label-" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26)),
		}}})
	}
	if got := ExtractKeywords(items, DefaultFilters(), 10); len(got) > 10 {
		t.Errorf("len = %d, want ≤10", len(got))
	}
}

func TestExtractKeywords_DeterministicOrder(t *testing.T) {
	items := []RankedIssue{
		{Issue: Issue{Labels: []string{"alpha"}}},
		{Issue: Issue{Labels: []string{"beta"}}},
		{Issue: Issue{Labels: []string{"alpha"}}},
		{Issue: Issue{Labels: []string{"beta"}}},
	}
	a := ExtractKeywords(items, DefaultFilters(), 25)
	b := ExtractKeywords(items, DefaultFilters(), 25)
	for i := range a {
		if a[i].Term != b[i].Term {
			t.Fatalf("nondeterministic at %d", i)
		}
	}
}
```

- [ ] **Step 2:** Run — fails.
- [ ] **Step 3:** Create `keywords.go`:

```go
package hermesissues

import (
	"sort"
	"strings"
)

func ExtractKeywords(items []RankedIssue, f Filters, maxKeywords int) []KeywordCandidate {
	if maxKeywords <= 0 {
		maxKeywords = 25
	}
	noise := lowerSet(f.NoiseLabels)

	labelHits := map[string]int{}
	for _, ri := range items {
		seen := map[string]bool{}
		for _, l := range ri.Issue.Labels {
			term := strings.ToLower(strings.TrimSpace(l))
			if term == "" || seen[term] {
				continue
			}
			if _, isNoise := noise[term]; isNoise {
				continue
			}
			seen[term] = true
			labelHits[term]++
		}
	}

	titleHits := map[string]int{}
	for _, ri := range items {
		seen := map[string]bool{}
		for _, tok := range tokenizeTitle(ri.Issue.Title) {
			if seen[tok] {
				continue
			}
			seen[tok] = true
			titleHits[tok]++
		}
	}
	for k, v := range titleHits {
		if v < 2 {
			delete(titleHits, k)
		}
	}

	out := make([]KeywordCandidate, 0, len(labelHits)+len(titleHits))
	for term, hits := range labelHits {
		out = append(out, KeywordCandidate{Term: term, Source: "label", Hits: hits})
	}
	for term, hits := range titleHits {
		if _, isLabel := labelHits[term]; isLabel {
			continue
		}
		out = append(out, KeywordCandidate{Term: term, Source: "title", Hits: hits})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Hits != out[j].Hits {
			return out[i].Hits > out[j].Hits
		}
		return out[i].Term < out[j].Term
	})
	if len(out) > maxKeywords {
		out = out[:maxKeywords]
	}
	return out
}

var titleStopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "with": {}, "from": {}, "into": {},
	"add": {}, "fix": {}, "use": {}, "make": {}, "new": {}, "old": {},
	"on": {}, "in": {}, "of": {}, "to": {}, "at": {}, "by": {},
	"a": {}, "an": {}, "is": {}, "are": {}, "be": {}, "or": {},
	"this": {}, "that": {}, "when": {}, "how": {}, "why": {}, "what": {},
}

func tokenizeTitle(s string) []string {
	s = strings.ToLower(s)
	out := []string{}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if len(w) < 3 {
			continue
		}
		if _, isStop := titleStopwords[w]; isStop {
			continue
		}
		out = append(out, w)
	}
	return out
}
```

- [ ] **Step 4:** Run keyword tests → PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): ExtractKeywords mines grounded vocab from ranked items`.

---

## Task 8: Refresh orchestrator

**Files:** Create `internal/hermesissues/refresh.go`, `internal/hermesissues/refresh_test.go`.

- [ ] **Step 1:** Tests:

```go
package hermesissues

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRefresh_WritesAllArtifacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"number":1,"title":"PR thing","state":"open","created_at":"2026-04-01T00:00:00Z","updated_at":"2026-04-01T00:00:00Z","pull_request":{"url":"x"},"comments":3,"reactions":{"total_count":2,"+1":2}},
			{"number":2,"title":"Dup","body":"this body is plenty long to pass scope filtering no problem","state":"open","created_at":"2026-04-01T00:00:00Z","updated_at":"2026-04-01T00:00:00Z","comments":3,"reactions":{"total_count":2,"+1":2},"labels":[{"name":"duplicate"}]},
			{"number":3,"title":"Short","body":"x","state":"open","created_at":"2026-04-01T00:00:00Z","updated_at":"2026-04-01T00:00:00Z","comments":3,"reactions":{"total_count":2,"+1":2}},
			{"number":4,"title":"Real ask","body":"please add streaming tool output, this is critical for our agent loop","state":"open","created_at":"2026-04-01T00:00:00Z","updated_at":"2026-04-01T00:00:00Z","comments":7,"reactions":{"total_count":12,"+1":9},"labels":[{"name":"enhancement"}]}
		]`))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	d, err := Refresh(context.Background(), RefreshOptions{
		BaseURL: srv.URL, Owner: "NousResearch", Repo: "hermes-agent",
		CacheDir: tmp, TopN: 10, Filters: DefaultFilters(), Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.TotalSeen != 4 || d.AfterScope != 2 || d.AfterSignal != 1 {
		t.Errorf("funnel = seen=%d scope=%d signal=%d", d.TotalSeen, d.AfterScope, d.AfterSignal)
	}
	if len(d.Items) != 1 || d.Items[0].Issue.Number != 4 {
		t.Fatalf("items: %+v", d.Items)
	}
	for _, name := range []string{"issues.json", "digest.json", "digest.md", "suggested_keywords.json"} {
		if _, err := os.Stat(filepath.Join(tmp, name)); err != nil {
			t.Errorf("missing %s", name)
		}
	}
	md, _ := os.ReadFile(filepath.Join(tmp, "digest.md"))
	if !strings.Contains(string(md), "#4 Real ask") {
		t.Errorf("digest.md content wrong: %s", md)
	}

	kws, err := ReadSuggestedKeywords(tmp)
	if err != nil {
		t.Fatal(err)
	}
	foundEnh := false
	for _, k := range kws {
		if k.Term == "enhancement" && k.Source == "label" {
			foundEnh = true
		}
	}
	if !foundEnh {
		t.Errorf("expected enhancement label keyword, got %+v", kws)
	}

	// digest.json round-trips
	raw, _ := os.ReadFile(filepath.Join(tmp, "digest.json"))
	var decoded Digest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Errorf("digest.json unmarshal: %v", err)
	}
}

func TestRefresh_TopNCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"number":1,"title":"a","body":"long body well past the scope-filter minimum bytes please","state":"open","updated_at":"2026-04-20T00:00:00Z","created_at":"2026-04-20T00:00:00Z","reactions":{"total_count":3,"+1":3}},
			{"number":2,"title":"b","body":"long body well past the scope-filter minimum bytes please","state":"open","updated_at":"2026-04-20T00:00:00Z","created_at":"2026-04-20T00:00:00Z","reactions":{"total_count":9,"+1":9}},
			{"number":3,"title":"c","body":"long body well past the scope-filter minimum bytes please","state":"open","updated_at":"2026-04-20T00:00:00Z","created_at":"2026-04-20T00:00:00Z","reactions":{"total_count":6,"+1":6}}
		]`))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	now := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	d, err := Refresh(context.Background(), RefreshOptions{
		BaseURL: srv.URL, Owner: "x", Repo: "y", CacheDir: tmp, TopN: 2,
		Filters: DefaultFilters(), Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Items) != 2 || d.Items[0].Issue.Number != 2 {
		t.Fatalf("got %+v", d.Items)
	}
}
```

- [ ] **Step 2:** Run — fails.
- [ ] **Step 3:** Create `refresh.go`:

```go
package hermesissues

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type RefreshOptions struct {
	BaseURL  string
	Owner    string
	Repo     string
	Token    string
	CacheDir string
	TopN     int
	Filters  Filters
	Now      time.Time
}

func Refresh(ctx context.Context, opts RefreshOptions) (Digest, error) {
	if opts.TopN <= 0 {
		opts.TopN = 50
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.CacheDir == "" {
		return Digest{}, fmt.Errorf("Refresh: CacheDir is required")
	}
	if err := os.MkdirAll(opts.CacheDir, 0o755); err != nil {
		return Digest{}, err
	}

	c := NewClient(opts.BaseURL, opts.Owner, opts.Repo, opts.Token)
	all, err := c.FetchAll(ctx)
	if err != nil {
		return Digest{}, fmt.Errorf("Refresh: fetch: %w", err)
	}

	scoped := ScopeFilter(all, opts.Filters, opts.Now)
	signal := SignalFilter(scoped, opts.Filters)
	ranked := Rank(signal, opts.Filters, opts.Now)
	if len(ranked) > opts.TopN {
		ranked = ranked[:opts.TopN]
	}
	keywords := ExtractKeywords(ranked, opts.Filters, 25)

	d := Digest{
		GeneratedAt:       opts.Now,
		SourceRepo:        fmt.Sprintf("%s/%s", opts.Owner, opts.Repo),
		TotalSeen:         len(all),
		AfterScope:        len(scoped),
		AfterSignal:       len(signal),
		TopN:              len(ranked),
		Items:             ranked,
		SuggestedKeywords: keywords,
	}

	if err := writeJSON(filepath.Join(opts.CacheDir, "issues.json"), all); err != nil {
		return d, err
	}
	if err := writeJSON(filepath.Join(opts.CacheDir, "digest.json"), d); err != nil {
		return d, err
	}
	if err := writeJSON(filepath.Join(opts.CacheDir, "suggested_keywords.json"), keywords); err != nil {
		return d, err
	}
	if err := os.WriteFile(filepath.Join(opts.CacheDir, "digest.md"), []byte(RenderDigest(d)), 0o644); err != nil {
		return d, err
	}
	return d, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ReadDigestMarkdown returns ("", nil) when digest.md does not exist.
func ReadDigestMarkdown(cacheDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(cacheDir, "digest.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// ReadSuggestedKeywords returns (nil, nil) when the file does not exist.
func ReadSuggestedKeywords(cacheDir string) ([]KeywordCandidate, error) {
	data, err := os.ReadFile(filepath.Join(cacheDir, "suggested_keywords.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []KeywordCandidate
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode suggested_keywords.json: %w", err)
	}
	return out, nil
}
```

- [ ] **Step 4:** Run refresh tests → PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): Refresh orchestrator + readers`.

---

## Task 9: CLI subcommand `hermes-issues`

**Files:** Modify `cmd/architecture-planner-loop/main.go`, `cmd/architecture-planner-loop/main_test.go`.

- [ ] **Step 1:** Add to `main_test.go`:

```go
func TestRun_HermesIssuesSubcommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"number":1,"title":"streaming","body":"please add streaming tool output already, real signal text","state":"open","created_at":"2026-04-01T00:00:00Z","updated_at":"2026-04-20T00:00:00Z","comments":5,"reactions":{"total_count":7,"+1":7},"labels":[{"name":"enhancement"}]}]`))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	t.Setenv("RUN_ROOT", tmp)
	t.Setenv("HERMES_ISSUES_API_BASE", srv.URL)
	t.Setenv("HERMES_ISSUES_TOKEN", "")

	var out bytes.Buffer
	prev := commandStdout
	commandStdout = &out
	defer func() { commandStdout = prev }()

	if err := run([]string{"hermes-issues"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "hermes-issues", "digest.md")); err != nil {
		t.Fatalf("digest.md not written: %v", err)
	}
	if !strings.Contains(out.String(), "wrote") {
		t.Errorf("stdout = %q", out.String())
	}
}
```

Imports needed: `bytes`, `net/http`, `net/http/httptest`, `os`, `path/filepath`, `strings`, `testing`.

- [ ] **Step 2:** Run — fails.
- [ ] **Step 3:** Edit `cmd/architecture-planner-loop/main.go`:

(a) Update `usage`:

```go
const usage = "usage: architecture-planner-loop run [--dry-run] [--codexu|--claudeu] [--mode safe|full|unattended] [keyword ...] | status | show-report | doctor | hermes-issues | service install [--force]"
```

(b) Add case:

```go
case "hermes-issues":
    cfg, err := architectureplanner.ConfigFromEnv(root, plannerEnv(runOptions{}))
    if err != nil {
        return err
    }
    return runHermesIssues(context.Background(), cfg)
```

(c) Add imports for `strconv`, `time`, and `internal/hermesissues`. Add helper:

```go
func runHermesIssues(ctx context.Context, cfg architectureplanner.Config) error {
	cacheDir := filepath.Join(cfg.RunRoot, "hermes-issues")

	apiBase := os.Getenv("HERMES_ISSUES_API_BASE")
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	token := os.Getenv("HERMES_ISSUES_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	topN := 50
	if v := os.Getenv("HERMES_ISSUES_TOP_N"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return fmt.Errorf("HERMES_ISSUES_TOP_N must be a positive integer, got %q", v)
		}
		topN = n
	}

	d, err := hermesissues.Refresh(ctx, hermesissues.RefreshOptions{
		BaseURL: apiBase, Owner: "NousResearch", Repo: "hermes-agent",
		Token: token, CacheDir: cacheDir, TopN: topN,
		Filters: hermesissues.DefaultFilters(), Now: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("hermes-issues refresh: %w", err)
	}
	_, err = fmt.Fprintf(commandStdout,
		"hermes-issues: wrote %s (seen=%d, scope=%d, signal=%d, top=%d)\n",
		filepath.Join(cacheDir, "digest.md"),
		d.TotalSeen, d.AfterScope, d.AfterSignal, d.TopN)
	return err
}
```

- [ ] **Step 4:** Run all `cmd/architecture-planner-loop` tests → PASS.
- [ ] **Step 5:** Commit `feat(planner-cli): hermes-issues subcommand`.

---

## Task 9b: Filter digest by user-supplied keywords

**Files:** Modify `internal/hermesissues/filter.go`, `internal/hermesissues/filter_test.go`.

This task closes the coherence gap: when the user passes `run streaming`, the existing `FilterContextByKeywords` narrows internal context — `FilterDigestByKeywords` does the same to the upstream-issue signal so the topical clause has consistent meaning end-to-end.

- [ ] **Step 1:** Tests:

```go
func TestFilterDigestByKeywords_MatchesTitleBodyOrLabel(t *testing.T) {
	d := Digest{
		TotalSeen: 100, AfterScope: 50, AfterSignal: 10, TopN: 4,
		Items: []RankedIssue{
			{Issue: Issue{Number: 1, Title: "Streaming tool output", Body: "x", Labels: []string{"enhancement"}}},
			{Issue: Issue{Number: 2, Title: "Memory backend", Body: "we need streaming saves", Labels: []string{"feature"}}},
			{Issue: Issue{Number: 3, Title: "Auth flow", Body: "x", Labels: []string{"streaming"}}},
			{Issue: Issue{Number: 4, Title: "Unrelated", Body: "no signal", Labels: []string{"misc"}}},
		},
	}
	got := FilterDigestByKeywords(d, []string{"streaming"})
	if len(got.Items) != 3 {
		t.Fatalf("len = %d, want 3 (title/body/label hits); got %+v", len(got.Items), got.Items)
	}
	if got.TopN != 3 {
		t.Errorf("TopN = %d, want 3 (recomputed)", got.TopN)
	}
	if got.TotalSeen != 100 || got.AfterScope != 50 || got.AfterSignal != 10 {
		t.Errorf("funnel mutated: %+v", got)
	}
}

func TestFilterDigestByKeywords_EmptyPassthrough(t *testing.T) {
	d := Digest{Items: []RankedIssue{{Issue: Issue{Number: 1}}}, TopN: 1}
	got := FilterDigestByKeywords(d, nil)
	if len(got.Items) != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestFilterDigestByKeywords_CaseInsensitive(t *testing.T) {
	d := Digest{Items: []RankedIssue{{Issue: Issue{Number: 1, Title: "STREAMING tools"}}}, TopN: 1}
	got := FilterDigestByKeywords(d, []string{"streaming"})
	if len(got.Items) != 1 {
		t.Fatalf("case-insensitive failed: %+v", got)
	}
}
```

- [ ] **Step 2:** Run — fails.
- [ ] **Step 3:** Append to `filter.go`:

```go
// FilterDigestByKeywords narrows d.Items by case-insensitive substring match
// against Title, Body, or any Label. Funnel counts (TotalSeen, AfterScope,
// AfterSignal) are preserved — they describe the unfiltered upstream pipeline.
// TopN is recomputed from the surviving Items length. Mirrors
// architectureplanner.FilterContextByKeywords semantics so the keyword
// vocabulary has consistent meaning across internal context and upstream-issue
// signal.
func FilterDigestByKeywords(d Digest, keywords []string) Digest {
	if len(keywords) == 0 {
		return d
	}
	needles := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if kw = strings.TrimSpace(kw); kw != "" {
			needles = append(needles, strings.ToLower(kw))
		}
	}
	if len(needles) == 0 {
		return d
	}
	contains := func(haystack string) bool {
		h := strings.ToLower(haystack)
		for _, n := range needles {
			if strings.Contains(h, n) {
				return true
			}
		}
		return false
	}

	out := d
	out.Items = nil
	for _, ri := range d.Items {
		if contains(ri.Issue.Title) || contains(ri.Issue.Body) {
			out.Items = append(out.Items, ri)
			continue
		}
		for _, l := range ri.Issue.Labels {
			if contains(l) {
				out.Items = append(out.Items, ri)
				break
			}
		}
	}
	out.TopN = len(out.Items)
	return out
}
```

- [ ] **Step 4:** Run filter tests → PASS.
- [ ] **Step 5:** Commit `feat(hermesissues): FilterDigestByKeywords mirrors planner topical scope`.

---

## Task 10: ContextBundle + prompt integration

**Files:** Modify `internal/architectureplanner/context.go`, `context_test.go`, `prompt.go`, `prompt_test.go`.

- [ ] **Step 1:** Add fields to `ContextBundle` (`internal/architectureplanner/context.go`):

```go
HermesIssuesDigest      string                          `json:"hermes_issues_digest,omitempty"`
HermesSuggestedKeywords []hermesissues.KeywordCandidate `json:"hermes_suggested_keywords,omitempty"`
```

Add import `"github.com/TrebuchetDynamics/gormes-agent/internal/hermesissues"`.

- [ ] **Step 2:** In `CollectContext`, before the final `return ContextBundle{...}`:

```go
hermesCacheDir := filepath.Join(cfg.RunRoot, "hermes-issues")
hermesDigest, err := hermesissues.ReadDigestMarkdown(hermesCacheDir)
if err != nil {
    return ContextBundle{}, fmt.Errorf("read hermes-issues digest: %w", err)
}
hermesKeywords, err := hermesissues.ReadSuggestedKeywords(hermesCacheDir)
if err != nil {
    return ContextBundle{}, fmt.Errorf("read hermes-issues suggested keywords: %w", err)
}
```

Add `HermesIssuesDigest: hermesDigest,` and `HermesSuggestedKeywords: hermesKeywords,` to the returned struct literal.

- [ ] **Step 3:** Add tests to `context_test.go` (adapt the fixture helper to whatever already exists):

```go
func TestCollectContext_HermesDigestPassthrough(t *testing.T) {
	tmp := t.TempDir()
	cfg := makeMinimalTestConfig(t, tmp) // adapt to existing helper

	dir := filepath.Join(cfg.RunRoot, "hermes-issues")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const want = "# Hermes Upstream Issue Signal Digest\n(test fixture)\n"
	if err := os.WriteFile(filepath.Join(dir, "digest.md"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	kws := []hermesissues.KeywordCandidate{{Term: "streaming", Source: "label", Hits: 3}}
	if err := writeJSONFixture(filepath.Join(dir, "suggested_keywords.json"), kws); err != nil {
		t.Fatal(err)
	}

	bundle, err := CollectContext(cfg, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if bundle.HermesIssuesDigest != want {
		t.Errorf("digest = %q", bundle.HermesIssuesDigest)
	}
	if len(bundle.HermesSuggestedKeywords) != 1 || bundle.HermesSuggestedKeywords[0].Term != "streaming" {
		t.Errorf("keywords = %+v", bundle.HermesSuggestedKeywords)
	}
}

func TestCollectContext_HermesAbsentEmpty(t *testing.T) {
	tmp := t.TempDir()
	cfg := makeMinimalTestConfig(t, tmp)
	bundle, err := CollectContext(cfg, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if bundle.HermesIssuesDigest != "" || len(bundle.HermesSuggestedKeywords) != 0 {
		t.Errorf("expected empties, got %+v", bundle)
	}
}
```

If `makeMinimalTestConfig` and `writeJSONFixture` don't exist, replace with whatever helper convention `context_test.go` already uses; the assertion behaviour is the only mandatory part.

- [ ] **Step 4:** Update `internal/architectureplanner/prompt.go` `BuildPrompt`. After the upstream-source block, before the autoloop audit, append:

```go
if bundle.HermesIssuesDigest != "" {
    fmt.Fprintln(&b)
    fmt.Fprintln(&b, "## Hermes Upstream Issue Signal")
    fmt.Fprintln(&b, "The block below is the most recent hermes-issues digest. Lift the ~20 highest-signal items into Gormes roadmap rows where they fit; cite the issue URL in source_refs when promoting one. Skip items already covered.")
    fmt.Fprintln(&b)
    fmt.Fprintln(&b, bundle.HermesIssuesDigest)
}

if len(bundle.HermesSuggestedKeywords) > 0 {
    fmt.Fprintln(&b)
    fmt.Fprintln(&b, "## Suggested Topical Keywords (from upstream issue signal)")
    fmt.Fprintln(&b, "These terms were extracted from the highest-signal Hermes issues (labels + frequency-counted title nouns). Treat them as a vocabulary suggestion: prefer phrasing that matches user-pain language when refining or splitting roadmap rows. Top 10:")
    fmt.Fprintln(&b)
    n := len(bundle.HermesSuggestedKeywords)
    if n > 10 {
        n = 10
    }
    for _, k := range bundle.HermesSuggestedKeywords[:n] {
        fmt.Fprintf(&b, "- %s (%s, hits=%d)\n", k.Term, k.Source, k.Hits)
    }
}
```

(`b` is the existing strings.Builder in the function — adjust variable name if it differs.)

- [ ] **Step 5:** Add tests to `prompt_test.go`:

```go
func TestBuildPrompt_HermesDigestSection(t *testing.T) {
	bundle := minimalBundleForPromptTest(t) // adapt
	bundle.HermesIssuesDigest = "# Hermes Upstream Issue Signal Digest\n#42 streaming\n"
	got := BuildPrompt(bundle, nil)
	if !strings.Contains(got, "## Hermes Upstream Issue Signal") || !strings.Contains(got, "#42 streaming") {
		t.Errorf("missing digest section: %s", got)
	}
}

func TestBuildPrompt_OmitsHermesDigestWhenAbsent(t *testing.T) {
	bundle := minimalBundleForPromptTest(t)
	got := BuildPrompt(bundle, nil)
	if strings.Contains(got, "## Hermes Upstream Issue Signal") {
		t.Errorf("digest section must not appear when empty")
	}
}

func TestBuildPrompt_SuggestedKeywordsSection(t *testing.T) {
	bundle := minimalBundleForPromptTest(t)
	bundle.HermesSuggestedKeywords = []hermesissues.KeywordCandidate{
		{Term: "streaming", Source: "label", Hits: 7},
		{Term: "tools", Source: "title", Hits: 5},
	}
	got := BuildPrompt(bundle, nil)
	if !strings.Contains(got, "## Suggested Topical Keywords") {
		t.Errorf("missing keywords section")
	}
	if !strings.Contains(got, "streaming") || !strings.Contains(got, "hits=7") {
		t.Errorf("missing keyword detail: %s", got)
	}
}

func TestBuildPrompt_OmitsSuggestedKeywordsWhenEmpty(t *testing.T) {
	bundle := minimalBundleForPromptTest(t)
	got := BuildPrompt(bundle, nil)
	if strings.Contains(got, "## Suggested Topical Keywords") {
		t.Errorf("keywords section must not appear when empty")
	}
}
```

Add `hermesissues` import where needed.

- [ ] **Step 6:** Run `go test ./internal/architectureplanner/ -count=1` → PASS.
- [ ] **Step 7:** Commit `feat(planner): surface hermes digest + suggested keywords in prompt`.

---

## Task 11: Wrapper script gating

**Files:** Modify `scripts/architecture-planner-loop.sh`.

- [ ] **Step 1:** Read current wrapper. Locate the `go run ./cmd/architecture-planner-loop run ...` line.
- [ ] **Step 2:** Insert before it:

```bash
if [ "${HERMES_ISSUES_PATHWAY:-0}" = "1" ]; then
    echo "hermes-issues: refreshing digest"
    if ! go run ./cmd/architecture-planner-loop hermes-issues; then
        echo "hermes-issues: refresh failed; continuing with existing digest if any" >&2
    fi
fi
```

Refresh failures must NOT abort the planner run — the pathway is opt-in and degrading gracefully on rate limits / network is required.

- [ ] **Step 3:** Syntax-check.

```bash
bash -n scripts/architecture-planner-loop.sh
```

- [ ] **Step 4:** Commit `feat(planner-wrapper): opt-in hermes-issues refresh via HERMES_ISSUES_PATHWAY=1`.

---

## Task 12: README documentation

**Files:** Modify `cmd/architecture-planner-loop/README.md`.

- [ ] **Step 1:** Insert before `## Service Timer`:

````markdown
## Hermes Upstream Issues Pathway (optional)

Periodically mines `NousResearch/hermes-agent` GitHub issues for feature
requests and pain-point signals to inform Gormes roadmap rows.

Refresh manually:

```sh
go run ./cmd/architecture-planner-loop hermes-issues
```

Writes under `<RUN_ROOT>/hermes-issues/`:

- `issues.json` — raw fetched payload (re-rank without re-fetching)
- `digest.json` — structured ranked items + suggested keywords
- `digest.md` — markdown digest the planner prompt consumes
- `suggested_keywords.json` — labels + frequency-counted title nouns

Configuration:

| Variable | Default | Purpose |
|---|---|---|
| `HERMES_ISSUES_API_BASE` | `https://api.github.com` | Override for testing |
| `HERMES_ISSUES_TOKEN` (or `GITHUB_TOKEN`) | unset | Raises rate limit 60→5000 req/hr |
| `HERMES_ISSUES_TOP_N` | `50` | Maximum items in the digest |
| `HERMES_ISSUES_PATHWAY` | `0` | Set to `1` to enable refresh in the systemd timer |

Two-stage rule-based filter:

1. **Scope:** drops PRs, items older than 18 months, closed-as-not-planned, bodies <40 chars.
2. **Signal:** drops noise labels (`duplicate`, `wontfix`, `invalid`, `question`, `spam`, `stale`, `needs-info`); requires ≥1 reaction OR ≥1 comment, OR a boost label (`enhancement`, `feature`, `feature-request`, `good first issue`, `help wanted`, `rfc`).

Final selection of which ~20 issues become roadmap rows is delegated to the planner LLM via the prompt — the rule-based filter hands it ~50–150 high-signal candidates.

### Keyword reinforcement

The pathway also reinforces the planner's existing `Keywords` feature by writing `suggested_keywords.json` — a grounded vocabulary (labels + ≥2-occurrence title nouns) extracted from top-N digest items. The planner prompt surfaces these as suggestions the LLM may honor. This addresses the disuse of the topical mechanism in autonomous mode (the systemd timer never supplies positional keywords).

When the user *does* pass positional keywords (`run streaming`), `FilterDigestByKeywords` narrows the digest section in addition to the existing `FilterContextByKeywords` narrowing internal context — same vocabulary, consistent meaning across both signal sources.
````

- [ ] **Step 2:** `go test ./docs -count=1` → PASS (no regression in doc tests).
- [ ] **Step 3:** Commit `docs(planner): document hermes-issues optional pathway`.

---

## Task 13: Final verification

- [ ] **Step 1:** `go test ./... -count=1` → all PASS.
- [ ] **Step 2:** `go build ./...` → exit 0.
- [ ] **Step 3:** End-to-end smoke (live GitHub):

```bash
RUN_ROOT=/tmp/gormes-hermes-smoke go run ./cmd/architecture-planner-loop hermes-issues
ls /tmp/gormes-hermes-smoke/hermes-issues/
head -40 /tmp/gormes-hermes-smoke/hermes-issues/digest.md
```

Expected: all four artifacts present, `Total seen:` >2000, top items have non-zero scores. Set `GITHUB_TOKEN` if anonymous rate-limit was already burned.

- [ ] **Step 4:** Verify prompt integration:

```bash
RUN_ROOT=/tmp/gormes-hermes-smoke go run ./cmd/architecture-planner-loop run --dry-run
grep -c "Hermes Upstream Issue Signal" /tmp/gormes-hermes-smoke/latest_prompt.txt
grep -c "Suggested Topical Keywords" /tmp/gormes-hermes-smoke/latest_prompt.txt
```

Expected: both `1`.

- [ ] **Step 5:** Verify graceful failure when API is unreachable:

```bash
HERMES_ISSUES_API_BASE=http://127.0.0.1:1 go run ./cmd/architecture-planner-loop hermes-issues || echo "expected non-zero exit"
RUN_ROOT=/tmp/gormes-hermes-smoke go run ./cmd/architecture-planner-loop run --dry-run
```

The first command must exit non-zero with a clear network error. The planner `run --dry-run` must still succeed using the previously cached digest.

---

## Notes for the implementer

- **Rate limits.** Anonymous = 60/hr; one cold `FetchAll` = ~24 requests = fine. If iterating on filter weights, set `GITHUB_TOKEN`.
- **No ETag caching.** YAGNI for a slow timer (hours, not minutes). Re-fetching the full set every run is fine.
- **Don't tune weights without data.** First production run produces the ground-truth dataset for tuning.
- **Pathway must stay non-blocking.** `Refresh` errors must never break the planner run. Verified in Task 13 Step 5.
- **Keyword reinforcement is suggestion, not auto-application.** `BuildPrompt` surfaces `HermesSuggestedKeywords` to the LLM; we do *not* auto-feed them into `FilterContextByKeywords`. Auto-application would silently change planner behaviour every time upstream issue activity shifts — too aggressive. The LLM picks.
