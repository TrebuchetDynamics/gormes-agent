package e2e

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
)

const (
	gonchoGoldenFixtureName     = "goncho_memory_turn"
	gonchoGoldenPeer            = e2ePeer
	gonchoGoldenChatKey         = "telegram:goncho-golden"
	gonchoGoldenSessionKey      = "sess-goncho-golden"
	gonchoGoldenModel           = "goncho-golden-model"
	gonchoGoldenPipelineVersion = "goncho-golden-e2e-v1"
	gonchoGoldenScoringVersion  = "goncho-golden-scoring-v1"
	gonchoGoldenFirstUser       = "When answering me about Goncho, prefer evidence-first reports and mention RecallTrace when retrieval is involved."
	gonchoGoldenFirstAssistant  = "Captured the Goncho recall preference for later use."
	gonchoGoldenSecondUser      = "How should you answer me about Goncho RecallTrace evidence?"
	gonchoGoldenSecondAssistant = "Use evidence-first reports and mention RecallTrace when retrieval is involved."
	gonchoGoldenMissingContext  = "MISSING_GONCHO_CONTEXT"
	gonchoGoldenNegativeLeak    = "NEGATIVE_GONCHO_CONTEXT_LEAKED"
	gonchoGoldenConclusion      = "When answering about Goncho recall, prefer evidence-first reports and mention RecallTrace when retrieval is involved."
	gonchoGoldenNegative        = "When answering about Goncho recall, prefer vague summaries and hide trace details."
)

func TestGonchoGoldenTranscriptE2E(t *testing.T) {
	got := runGonchoGoldenTranscript(t)
	assertGonchoGoldenTranscript(t, got)
}

type gonchoGoldenTranscript struct {
	Fixture          string                         `json:"fixture"`
	FinalAssistant   string                         `json:"final_assistant"`
	Capture          gonchoGoldenCapture            `json:"capture"`
	ProviderRequests []gonchoGoldenProviderRequest  `json:"provider_requests"`
	PersistedTurns   []goldenPersistedTurn          `json:"persisted_turns"`
	Memory           gonchoGoldenMemoryEvidence     `json:"memory"`
	Trace            gonchoGoldenTraceEvidence      `json:"trace"`
	Diagnostics      gonchoGoldenDiagnosticsSummary `json:"diagnostics"`
	Replay           gonchoGoldenReplaySummary      `json:"replay"`
}

type gonchoGoldenCapture struct {
	ProfileCard       []string `json:"profile_card"`
	ConclusionID      int64    `json:"conclusion_id"`
	NegativeControlID int64    `json:"negative_control_id"`
}

type gonchoGoldenProviderRequest struct {
	Index                 int      `json:"index"`
	Model                 string   `json:"model"`
	SessionID             string   `json:"session_id"`
	LastUserMessage       string   `json:"last_user_message"`
	RecallInjected        bool     `json:"recall_injected"`
	RecallTerms           []string `json:"recall_terms,omitempty"`
	NegativeControlLeaked bool     `json:"negative_control_leaked"`
}

type gonchoGoldenMemoryEvidence struct {
	SearchResultIDs []int64  `json:"search_result_ids"`
	SearchTerms     []string `json:"search_terms"`
	NegativeLeaked  bool     `json:"negative_leaked"`
}

type gonchoGoldenTraceEvidence struct {
	TraceID                 string   `json:"trace_id"`
	PipelineVersion         string   `json:"pipeline_version"`
	ScoringConfigVersion    string   `json:"scoring_config_version"`
	SelectedMemoryIDs       []string `json:"selected_memory_ids"`
	RejectedReasons         []string `json:"rejected_reasons"`
	WarningCodes            []string `json:"warning_codes"`
	StableJSONSHA256        string   `json:"stable_json_sha256"`
	StableJSONDeterministic bool     `json:"stable_json_deterministic"`
}

type gonchoGoldenDiagnosticsSummary struct {
	Status              string `json:"status"`
	SelectedCount       int    `json:"selected_count"`
	RejectedCount       int    `json:"rejected_count"`
	WarningCount        int    `json:"warning_count"`
	ProjectionInvariant string `json:"projection_invariant"`
}

type gonchoGoldenReplaySummary struct {
	EventCount          int      `json:"event_count"`
	EventKinds          []string `json:"event_kinds"`
	ProjectionInvariant string   `json:"projection_invariant"`
	ReplayContract      string   `json:"replay_contract"`
}

func runGonchoGoldenTranscript(t *testing.T) gonchoGoldenTranscript {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(dir, "home"))
	mem, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 32, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mem.Close(ctx); err != nil {
			t.Fatalf("close memory store: %v", err)
		}
	})

	svc := goncho.NewService(mem.DB(), goncho.Config{
		WorkspaceID:    "default",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)
	recall := &gonchoGoldenRecall{svc: svc, peer: gonchoGoldenPeer}
	provider := &gonchoGoldenProvider{}
	auditPath := filepath.Join(dir, "tool-audit.jsonl")
	k := kernel.New(kernel.Config{
		Model:            gonchoGoldenModel,
		Endpoint:         "mock://goncho-golden",
		Admission:        kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
		InitialSessionID: gonchoGoldenSessionKey,
		ChatKey:          gonchoGoldenChatKey,
		Recall:           recall,
		RecallDeadline:   time.Second,
		ToolAudit:        audit.NewJSONLWriter(auditPath),
	}, provider, mem, telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- k.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("kernel run: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("kernel did not stop after cleanup cancel")
		}
	})
	waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool {
		return f.Phase == kernel.PhaseIdle
	}, time.Second)

	runGonchoGoldenTurn(t, k, gonchoGoldenFirstUser, gonchoGoldenFirstAssistant, 2)
	waitGonchoGoldenPersistedTurnCount(t, mem.DB(), 2)
	capture := captureGonchoGoldenMemory(t, mem.DB(), svc)

	final := runGonchoGoldenTurn(t, k, gonchoGoldenSecondUser, gonchoGoldenSecondAssistant, 4)
	if got := final.History[len(final.History)-1].Content; got != gonchoGoldenSecondAssistant {
		t.Fatalf("final assistant = %q, want %q", got, gonchoGoldenSecondAssistant)
	}
	persisted := waitGonchoGoldenPersistedTurns(t, mem.DB())
	search := assertGonchoGoldenSearch(t, svc)
	trace := recall.LastTrace(t)
	diagnostics := goncho.BuildRecallDiagnostics(trace)
	replay := goncho.BuildRecallReplay(trace)

	return gonchoGoldenTranscript{
		Fixture:          gonchoGoldenFixtureName,
		FinalAssistant:   gonchoGoldenSecondAssistant,
		Capture:          capture,
		ProviderRequests: summarizeGonchoGoldenRequests(provider.Requests()),
		PersistedTurns:   persisted,
		Memory:           summarizeGonchoGoldenMemory(search),
		Trace:            summarizeGonchoGoldenTrace(t, trace),
		Diagnostics: gonchoGoldenDiagnosticsSummary{
			Status:              diagnostics.Status,
			SelectedCount:       diagnostics.SelectedCount,
			RejectedCount:       diagnostics.RejectedCount,
			WarningCount:        diagnostics.WarningCount,
			ProjectionInvariant: diagnostics.ProjectionInvariant,
		},
		Replay: gonchoGoldenReplaySummary{
			EventCount:          replay.EventCount,
			EventKinds:          gonchoGoldenReplayKinds(replay),
			ProjectionInvariant: replay.ProjectionInvariant,
			ReplayContract:      replay.ReplayContract,
		},
	}
}

func runGonchoGoldenTurn(t *testing.T, k *kernel.Kernel, text string, want string, minHistory int) kernel.RenderFrame {
	t.Helper()
	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: text}); err != nil {
		t.Fatalf("submit Goncho golden turn: %v", err)
	}
	return waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool {
		if f.Phase != kernel.PhaseIdle || len(f.History) < minHistory {
			return false
		}
		last := f.History[len(f.History)-1]
		return last.Role == "assistant" && last.Content == want
	}, 5*time.Second)
}

func captureGonchoGoldenMemory(t *testing.T, db *sql.DB, svc *goncho.Service) gonchoGoldenCapture {
	t.Helper()
	ctx := context.Background()
	card := []string{
		"Prefers evidence-first reports",
		"Mention RecallTrace when retrieval is involved",
	}
	if err := svc.SetProfile(ctx, gonchoGoldenPeer, card); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
	positive, err := svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       gonchoGoldenPeer,
		Conclusion: gonchoGoldenConclusion,
		SessionKey: gonchoGoldenSessionKey,
	})
	if err != nil {
		t.Fatalf("Conclude positive memory: %v", err)
	}
	other := goncho.NewService(db, goncho.Config{WorkspaceID: "other", ObserverPeerID: "gormes"}, nil)
	negative, err := other.Conclude(ctx, goncho.ConcludeParams{
		Peer:       gonchoGoldenPeer,
		Conclusion: gonchoGoldenNegative,
		SessionKey: gonchoGoldenSessionKey,
	})
	if err != nil {
		t.Fatalf("Conclude negative control: %v", err)
	}
	return gonchoGoldenCapture{
		ProfileCard:       card,
		ConclusionID:      positive.ID,
		NegativeControlID: negative.ID,
	}
}

func assertGonchoGoldenSearch(t *testing.T, svc *goncho.Service) goncho.SearchResultSet {
	t.Helper()
	search, err := svc.Search(context.Background(), goncho.SearchParams{
		Peer:       gonchoGoldenPeer,
		Query:      "Goncho RecallTrace evidence-first",
		SessionKey: gonchoGoldenSessionKey,
		MaxTokens:  400,
		Limit:      5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(search.Results) == 0 {
		t.Fatalf("search results empty, want captured Goncho memory")
	}
	if !strings.Contains(search.Results[0].Content, "evidence-first reports") ||
		!strings.Contains(search.Results[0].Content, "RecallTrace") {
		t.Fatalf("search result = %+v, want captured RecallTrace preference", search.Results[0])
	}
	for _, hit := range search.Results {
		if strings.Contains(hit.Content, "vague summaries") {
			t.Fatalf("search leaked negative control result: %+v", hit)
		}
	}
	return search
}

func waitGonchoGoldenPersistedTurns(t *testing.T, db *sql.DB) []goldenPersistedTurn {
	t.Helper()
	return waitGonchoGoldenPersistedTurnCount(t, db, 4)
}

func waitGonchoGoldenPersistedTurnCount(t *testing.T, db *sql.DB, count int) []goldenPersistedTurn {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		turns := queryGonchoGoldenPersistedTurns(t, db)
		if len(turns) == count && allGonchoGoldenTurnsReady(turns) {
			return turns
		}
		if time.Now().After(deadline) {
			t.Fatalf("persisted Goncho golden turns not ready: got=%d want=%d turns=%+v", len(turns), count, turns)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func queryGonchoGoldenPersistedTurns(t *testing.T, db *sql.DB) []goldenPersistedTurn {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		`SELECT role, content, memory_sync_status, COALESCE(meta_json, '')
		   FROM turns
		  WHERE chat_id = ?
		    AND content IN (?, ?, ?, ?)
		  ORDER BY id`,
		gonchoGoldenChatKey,
		gonchoGoldenFirstUser,
		gonchoGoldenFirstAssistant,
		gonchoGoldenSecondUser,
		gonchoGoldenSecondAssistant,
	)
	if err != nil {
		t.Fatalf("query Goncho golden persisted turns: %v", err)
	}
	defer rows.Close()

	var out []goldenPersistedTurn
	for rows.Next() {
		var turn goldenPersistedTurn
		var meta string
		if err := rows.Scan(&turn.Role, &turn.Content, &turn.MemorySyncStatus, &meta); err != nil {
			t.Fatalf("scan Goncho golden persisted turn: %v", err)
		}
		turn.ToolCalls = toolCallsFromMeta(t, meta)
		out = append(out, turn)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate Goncho golden persisted turns: %v", err)
	}
	return out
}

func allGonchoGoldenTurnsReady(turns []goldenPersistedTurn) bool {
	for _, turn := range turns {
		if turn.MemorySyncStatus != "ready" {
			return false
		}
	}
	return true
}

type gonchoGoldenRecall struct {
	svc  *goncho.Service
	peer string
	mu   sync.Mutex
	last goncho.RecallTrace
}

func (r *gonchoGoldenRecall) GetContext(ctx context.Context, params kernel.RecallParams) string {
	if r.svc == nil {
		return ""
	}
	search, err := r.svc.Search(ctx, goncho.SearchParams{
		Peer:       r.peer,
		Query:      params.UserMessage,
		SessionKey: params.SessionID,
		MaxTokens:  400,
		Limit:      5,
	})
	if err != nil || len(search.Results) == 0 {
		return ""
	}
	trace := buildGonchoGoldenTrace(search, params)
	r.mu.Lock()
	r.last = trace
	r.mu.Unlock()

	contextResult := (&goncho.RecallProjector{}).ProjectContext(trace)
	if strings.TrimSpace(contextResult.Representation) == "" {
		return ""
	}
	return "Goncho memory context:\n" + contextResult.Representation +
		"\n\nRecallTrace: " + trace.TraceID +
		"\nScoringConfig: " + trace.ScoringConfig.Version
}

func (r *gonchoGoldenRecall) LastTrace(t *testing.T) goncho.RecallTrace {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.last.TraceID == "" {
		t.Fatal("Goncho recall trace was not recorded")
	}
	return r.last
}

func buildGonchoGoldenTrace(search goncho.SearchResultSet, params kernel.RecallParams) goncho.RecallTrace {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	scoring := goncho.RecallScoringConfig{
		Version: gonchoGoldenScoringVersion,
		Weights: map[string]float64{
			"keyword":    0.60,
			"semantic":   0.20,
			"graph":      0.10,
			"recency":    0.05,
			"importance": 0.03,
			"scope":      0.02,
		},
		RRFK:          60,
		MMRLambda:     1,
		DiversityKeys: []string{"session_id", "source_type"},
		TokenBudget:   400,
	}
	selected := make([]goncho.ScoredRecallCandidate, 0, len(search.Results))
	for _, hit := range search.Results {
		selected = append(selected, goncho.ScoredRecallCandidate{
			Candidate: goncho.RecallCandidate{
				MemoryID:   strconv.FormatInt(hit.ID, 10),
				SourceType: hit.Source,
				Content:    hit.Content,
				SessionID:  hit.SessionKey,
				ScopeID:    "default",
				CreatedAt:  now,
				Importance: 0.9,
				Provenance: []goncho.EvidenceItem{{
					Kind:   "keyword",
					Source: "goncho_search",
					ID:     strconv.FormatInt(hit.ID, 10),
					Score:  1,
					Metadata: map[string]string{
						"chat_key": params.ChatKey,
					},
				}},
			},
			Score: goncho.RecallScore{
				KeywordScore:    1,
				SemanticScore:   0,
				GraphScore:      0,
				RecencyScore:    1,
				ImportanceScore: 0.9,
				ScopeScore:      1,
				RRFScore:        0.016393,
				FinalScore:      1.016393,
				WhySelected: []string{
					"final_score=1.016393",
					"scoring_config=" + gonchoGoldenScoringVersion,
					"source=goncho_search",
					"projection=trace_only",
				},
			},
		})
	}
	negative := goncho.RejectedRecallCandidate{
		Candidate: goncho.RecallCandidate{
			MemoryID:   "negative-control",
			SourceType: "conclusion",
			Content:    gonchoGoldenNegative,
			SessionID:  params.SessionID,
			ScopeID:    "other",
			CreatedAt:  now,
			Importance: 0.1,
			Provenance: []goncho.EvidenceItem{{
				Kind:   "keyword",
				Source: "negative_control",
				Score:  1,
			}},
		},
		Score: goncho.RecallScore{
			KeywordScore:    1,
			RecencyScore:    1,
			ImportanceScore: 0.1,
			ScopeScore:      0,
			FinalScore:      0.1,
		},
		Reason: goncho.RecallRejectScopeMismatch,
		WhyRejected: []string{
			"candidate_scope=other",
			"query_scope=default",
		},
	}
	trace := goncho.RecallTrace{
		PipelineVersion: gonchoGoldenPipelineVersion,
		CreatedAt:       now,
		Query: goncho.RecallQuery{
			WorkspaceID: "default",
			Peer:        gonchoGoldenPeer,
			Query:       params.UserMessage,
			SessionKey:  params.SessionID,
			ScopeID:     "default",
			Limit:       5,
			MaxTokens:   400,
		},
		ScoringConfig: scoring,
		Candidates:    selected,
		Selected:      selected,
		Rejected:      []goncho.RejectedRecallCandidate{negative},
		Warnings:      []goncho.RecallWarning{},
	}
	trace.TraceID = gonchoGoldenTraceID(trace)
	return trace
}

func gonchoGoldenTraceID(trace goncho.RecallTrace) string {
	view := struct {
		Query           goncho.RecallQuery `json:"query"`
		CandidateIDs    []string           `json:"candidate_ids"`
		ScoringVersion  string             `json:"scoring_version"`
		PipelineVersion string             `json:"pipeline_version"`
		Weights         map[string]float64 `json:"weights"`
		DiversityKeys   []string           `json:"diversity_keys,omitempty"`
		RRFK            int                `json:"rrf_k"`
		MMRLambda       float64            `json:"mmr_lambda"`
		TokenBudget     int                `json:"token_budget,omitempty"`
	}{
		Query:           trace.Query,
		ScoringVersion:  trace.ScoringConfig.Version,
		PipelineVersion: trace.PipelineVersion,
		Weights:         trace.ScoringConfig.Weights,
		DiversityKeys:   trace.ScoringConfig.DiversityKeys,
		RRFK:            trace.ScoringConfig.RRFK,
		MMRLambda:       trace.ScoringConfig.MMRLambda,
		TokenBudget:     trace.ScoringConfig.TokenBudget,
	}
	for _, item := range trace.Candidates {
		view.CandidateIDs = append(view.CandidateIDs, item.Candidate.MemoryID)
	}
	raw, _ := json.Marshal(view)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type gonchoGoldenProvider struct {
	mu       sync.Mutex
	requests []hermes.ChatRequest
	calls    int
}

func (p *gonchoGoldenProvider) Health(context.Context) error { return nil }

func (p *gonchoGoldenProvider) ProviderStatus() hermes.ProviderStatus {
	return hermes.ProviderStatus{Provider: "goncho-golden", Runtime: "test_harness"}
}

func (p *gonchoGoldenProvider) OpenRunEvents(context.Context, string) (hermes.RunEventStream, error) {
	return nil, hermes.ErrRunEventsNotSupported
}

func (p *gonchoGoldenProvider) OpenStream(_ context.Context, req hermes.ChatRequest) (hermes.Stream, error) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	p.requests = append(p.requests, req)
	p.mu.Unlock()

	answer := gonchoGoldenFirstAssistant
	if call >= 2 {
		switch {
		case requestContains(req, gonchoGoldenNegative):
			answer = gonchoGoldenNegativeLeak
		case !requestContains(req, "Goncho memory context") ||
			!requestContains(req, "evidence-first reports") ||
			!requestContains(req, "RecallTrace:"):
			answer = gonchoGoldenMissingContext
		default:
			answer = gonchoGoldenSecondAssistant
		}
	}
	return &gonchoGoldenStream{
		sessionID: req.SessionID,
		events: []hermes.Event{
			{Kind: hermes.EventToken, Token: answer, TokensOut: len(strings.Fields(answer))},
			{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 64, TokensOut: len(strings.Fields(answer))},
		},
	}, nil
}

func (p *gonchoGoldenProvider) Requests() []hermes.ChatRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]hermes.ChatRequest, len(p.requests))
	copy(out, p.requests)
	return out
}

type gonchoGoldenStream struct {
	sessionID string
	events    []hermes.Event
	pos       int
	closed    bool
	mu        sync.Mutex
}

func (s *gonchoGoldenStream) SessionID() string { return s.sessionID }

func (s *gonchoGoldenStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *gonchoGoldenStream) Recv(ctx context.Context) (hermes.Event, error) {
	select {
	case <-ctx.Done():
		return hermes.Event{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.pos >= len(s.events) {
		return hermes.Event{}, io.EOF
	}
	event := s.events[s.pos]
	s.pos++
	return event, nil
}

func requestContains(req hermes.ChatRequest, needle string) bool {
	for _, msg := range req.Messages {
		if strings.Contains(msg.Content, needle) {
			return true
		}
	}
	return false
}

func summarizeGonchoGoldenRequests(requests []hermes.ChatRequest) []gonchoGoldenProviderRequest {
	out := make([]gonchoGoldenProviderRequest, 0, len(requests))
	for i, req := range requests {
		terms := gonchoGoldenRecallTerms(req)
		out = append(out, gonchoGoldenProviderRequest{
			Index:                 i + 1,
			Model:                 req.Model,
			SessionID:             req.SessionID,
			LastUserMessage:       lastUserMessage(req.Messages),
			RecallInjected:        len(terms) > 0,
			RecallTerms:           terms,
			NegativeControlLeaked: requestContains(req, gonchoGoldenNegative),
		})
	}
	return out
}

func gonchoGoldenRecallTerms(req hermes.ChatRequest) []string {
	var combined strings.Builder
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			combined.WriteString(msg.Content)
			combined.WriteByte('\n')
		}
	}
	content := combined.String()
	if !strings.Contains(content, "Goncho memory context") {
		return nil
	}
	terms := []string{"Goncho memory context"}
	for _, term := range []string{"evidence-first reports", "RecallTrace", "trace_id", gonchoGoldenScoringVersion} {
		if strings.Contains(content, term) {
			terms = append(terms, term)
		}
	}
	return terms
}

func lastUserMessage(messages []hermes.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

func summarizeGonchoGoldenMemory(search goncho.SearchResultSet) gonchoGoldenMemoryEvidence {
	out := gonchoGoldenMemoryEvidence{
		SearchTerms: []string{"Goncho", "RecallTrace", "evidence-first"},
	}
	for _, hit := range search.Results {
		out.SearchResultIDs = append(out.SearchResultIDs, hit.ID)
		if strings.Contains(hit.Content, "vague summaries") {
			out.NegativeLeaked = true
		}
	}
	if out.SearchResultIDs == nil {
		out.SearchResultIDs = []int64{}
	}
	return out
}

func summarizeGonchoGoldenTrace(t *testing.T, trace goncho.RecallTrace) gonchoGoldenTraceEvidence {
	t.Helper()
	raw1, err := trace.StableJSON()
	if err != nil {
		t.Fatalf("StableJSON first call: %v", err)
	}
	raw2, err := trace.StableJSON()
	if err != nil {
		t.Fatalf("StableJSON second call: %v", err)
	}
	return gonchoGoldenTraceEvidence{
		TraceID:                 trace.TraceID,
		PipelineVersion:         trace.PipelineVersion,
		ScoringConfigVersion:    trace.ScoringConfig.Version,
		SelectedMemoryIDs:       selectedGonchoGoldenMemoryIDs(trace),
		RejectedReasons:         rejectedGonchoGoldenReasons(trace),
		WarningCodes:            warningGonchoGoldenCodes(trace),
		StableJSONSHA256:        sha256Hex(raw1),
		StableJSONDeterministic: string(raw1) == string(raw2),
	}
}

func selectedGonchoGoldenMemoryIDs(trace goncho.RecallTrace) []string {
	out := make([]string, 0, len(trace.Selected))
	for _, item := range trace.Selected {
		out = append(out, item.Candidate.MemoryID)
	}
	return out
}

func rejectedGonchoGoldenReasons(trace goncho.RecallTrace) []string {
	out := make([]string, 0, len(trace.Rejected))
	for _, item := range trace.Rejected {
		out = append(out, item.Reason)
	}
	return out
}

func warningGonchoGoldenCodes(trace goncho.RecallTrace) []string {
	out := make([]string, 0, len(trace.Warnings))
	for _, warning := range trace.Warnings {
		out = append(out, warning.Code)
	}
	return out
}

func gonchoGoldenReplayKinds(replay goncho.RecallReplay) []string {
	out := make([]string, 0, len(replay.Events))
	for _, event := range replay.Events {
		out = append(out, event.Kind)
	}
	return out
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func assertGonchoGoldenTranscript(t *testing.T, got gonchoGoldenTranscript) {
	t.Helper()
	if got.FinalAssistant != gonchoGoldenSecondAssistant {
		t.Fatalf("final assistant = %q, want %q", got.FinalAssistant, gonchoGoldenSecondAssistant)
	}
	if len(got.ProviderRequests) != 2 {
		t.Fatalf("provider request count = %d, want two deterministic turns", len(got.ProviderRequests))
	}
	if got.ProviderRequests[0].RecallInjected {
		t.Fatalf("first provider request unexpectedly had recall context: %+v", got.ProviderRequests[0])
	}
	if !got.ProviderRequests[1].RecallInjected {
		t.Fatalf("second provider request missing recall context: %+v", got.ProviderRequests[1])
	}
	if got.ProviderRequests[1].NegativeControlLeaked || got.Memory.NegativeLeaked {
		t.Fatalf("negative control leaked into recall: provider=%+v memory=%+v", got.ProviderRequests[1], got.Memory)
	}
	if got.Trace.TraceID == "" || got.Trace.PipelineVersion != gonchoGoldenPipelineVersion || got.Trace.ScoringConfigVersion != gonchoGoldenScoringVersion {
		t.Fatalf("trace summary = %+v, want stable id and scoring config", got.Trace)
	}
	if !got.Trace.StableJSONDeterministic {
		t.Fatalf("trace StableJSON was not deterministic: %+v", got.Trace)
	}
	if len(got.Trace.SelectedMemoryIDs) != 1 {
		t.Fatalf("selected memory ids = %+v, want one selected memory", got.Trace.SelectedMemoryIDs)
	}
	if len(got.Trace.RejectedReasons) != 1 || got.Trace.RejectedReasons[0] != goncho.RecallRejectScopeMismatch {
		t.Fatalf("rejected reasons = %+v, want scope_mismatch", got.Trace.RejectedReasons)
	}
	if got.Diagnostics.ProjectionInvariant != "no_projection_without_recall_trace" {
		t.Fatalf("diagnostics invariant = %q", got.Diagnostics.ProjectionInvariant)
	}
	if got.Replay.ReplayContract != "deterministic_replay_from_recall_trace" {
		t.Fatalf("replay contract = %q", got.Replay.ReplayContract)
	}

	assertGonchoGoldenFixture(t, got)
}

func assertGonchoGoldenFixture(t *testing.T, got gonchoGoldenTranscript) {
	t.Helper()
	path := filepath.Join("testdata", "goncho_memory_turn", "golden.json")
	gotRaw, err := transcript.MarshalStableJSON(got)
	if err != nil {
		t.Fatalf("marshal Goncho golden transcript: %v", err)
	}
	if os.Getenv("GORMES_UPDATE_GONCHO_GOLDEN_TRANSCRIPT") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create Goncho golden transcript dir: %v", err)
		}
		if err := os.WriteFile(path, gotRaw, 0o644); err != nil {
			t.Fatalf("write Goncho golden transcript %s: %v", path, err)
		}
		return
	}
	wantRaw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture_missing %s at $: %s", gonchoGoldenFixtureName, path)
	}
	if err != nil {
		t.Fatalf("read Goncho golden transcript %s: %v", path, err)
	}
	if err := transcript.CompareGoldenJSON(wantRaw, gotRaw); err != nil {
		var diff transcript.JSONDiff
		if errors.As(err, &diff) {
			t.Fatalf("goncho_golden_transcript_mismatch at %s: %s", diff.Path, diff.Message)
		}
		t.Fatalf("goncho_golden_transcript_mismatch: %v", err)
	}
}
