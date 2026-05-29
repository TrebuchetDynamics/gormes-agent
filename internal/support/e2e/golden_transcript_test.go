package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
)

func TestNormalTurnGoldenTranscripts(t *testing.T) {
	for _, scenario := range []goldenScenario{
		{
			Name:       "text_only",
			SessionID:  "sess-golden-text",
			ChatKey:    "golden:text",
			UserInput:  "Answer with a plain text-only final response.",
			FinalText:  "Plain text-only final response.",
			ProviderID: "provider-text",
			Scripts: [][]hermes.Event{{
				{Kind: hermes.EventToken, Token: "Plain text-only final response.", TokensOut: 5},
				{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 12, TokensOut: 5},
			}},
		},
		{
			Name:        "tool_call",
			SessionID:   "sess-golden-tool",
			ChatKey:     "golden:tool",
			UserInput:   "Inspect the fixture tool and summarize the result.",
			FinalText:   "The fixture tool returned a stable observation.",
			ProviderID:  "provider-tool",
			EnableTools: true,
			Scripts: [][]hermes.Event{
				{
					{Kind: hermes.EventToken, Token: "Inspecting fixture. ", TokensOut: 3},
					{
						Kind:         hermes.EventDone,
						FinishReason: "tool_calls",
						TokensIn:     18,
						TokensOut:    3,
						ToolCalls: []hermes.ToolCall{{
							ID:        "call_golden_echo",
							Name:      "golden_echo",
							Arguments: json.RawMessage(`{"query":"fixture observation"}`),
						}},
					},
				},
				{
					{Kind: hermes.EventToken, Token: "The fixture tool returned a stable observation.", TokensOut: 7},
					{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 31, TokensOut: 7},
				},
			},
		},
		{
			Name:         "memory_backed",
			SessionID:    e2eSessionKey,
			ChatKey:      e2eChatKey,
			UserInput:    "Use Goncho memory to summarize my reporting preference.",
			FinalText:    "Goncho memory says the user prefers exact evidence-first reports.",
			ProviderID:   "provider-memory",
			EnableGoncho: true,
			Scripts: [][]hermes.Event{{
				{Kind: hermes.EventToken, Token: "Goncho memory says the user prefers exact evidence-first reports.", TokensOut: 9},
				{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 24, TokensOut: 9},
			}},
		},
	} {
		t.Run(scenario.Name, func(t *testing.T) {
			got := runGoldenScenario(t, scenario)
			assertGoldenTranscript(t, scenario.Name, got)
		})
	}
}

type goldenScenario struct {
	Name         string
	SessionID    string
	ChatKey      string
	UserInput    string
	FinalText    string
	ProviderID   string
	EnableTools  bool
	EnableGoncho bool
	Scripts      [][]hermes.Event
}

type goldenTranscript struct {
	Fixture          string                  `json:"fixture"`
	UserInput        string                  `json:"user_input"`
	FinalAssistant   string                  `json:"final_assistant"`
	Status           goldenStatus            `json:"status"`
	ProviderRequests []goldenProviderRequest `json:"provider_requests"`
	ToolExecution    []goldenToolExecution   `json:"tool_execution"`
	Memory           goldenMemoryEvidence    `json:"memory"`
	Audit            []goldenAuditRecord     `json:"audit"`
}

type goldenStatus struct {
	Phase         string `json:"phase"`
	StatusText    string `json:"status_text"`
	SessionID     string `json:"session_id"`
	HistoryLength int    `json:"history_length"`
}

type goldenProviderRequest struct {
	Model     string          `json:"model"`
	SessionID string          `json:"session_id"`
	Stream    bool            `json:"stream"`
	Messages  []goldenMessage `json:"messages"`
	Tools     []string        `json:"tools"`
}

type goldenMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []goldenToolCall `json:"tool_calls,omitempty"`
}

type goldenToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type goldenToolExecution struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Args    json.RawMessage `json:"args"`
	Result  json.RawMessage `json:"result"`
	Status  string          `json:"status"`
	Audited bool            `json:"audited"`
}

type goldenMemoryEvidence struct {
	RecallInjected bool                  `json:"recall_injected"`
	RecallTerms    []string              `json:"recall_terms,omitempty"`
	PersistedTurns []goldenPersistedTurn `json:"persisted_turns"`
}

type goldenPersistedTurn struct {
	Role             string           `json:"role"`
	Content          string           `json:"content"`
	MemorySyncStatus string           `json:"memory_sync_status"`
	ToolCalls        []goldenToolCall `json:"tool_calls,omitempty"`
}

type goldenAuditRecord struct {
	Source          string          `json:"source"`
	SessionID       string          `json:"session_id"`
	Tool            string          `json:"tool"`
	Args            json.RawMessage `json:"args"`
	Status          string          `json:"status"`
	ResultSizeBytes int             `json:"result_size_bytes"`
	Error           string          `json:"error"`
}

func runGoldenScenario(t *testing.T, scenario goldenScenario) goldenTranscript {
	t.Helper()

	dir := t.TempDir()
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

	var recall kernel.RecallProvider
	if scenario.EnableGoncho {
		svc := goncho.NewService(mem.DB(), goncho.Config{
			WorkspaceID:    "default",
			ObserverPeerID: "gormes",
			RecentMessages: 4,
		}, nil)
		seedGonchoState(t, mem.DB(), svc)
		recall = &gonchoRecall{svc: svc, peer: e2ePeer, evidence: &turnEvidence{}}
	}

	var registry *tools.Registry
	if scenario.EnableTools {
		registry = tools.NewRegistry()
		registry.MustRegister(goldenEchoTool{})
	}

	provider := hermes.NewMockClient()
	for _, script := range scenario.Scripts {
		provider.Script(script, scenario.ProviderID)
	}
	auditPath := filepath.Join(dir, "tool-audit.jsonl")
	k := kernel.New(kernel.Config{
		Model:             "golden-model",
		Endpoint:          "mock://golden",
		Admission:         kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             registry,
		MaxToolIterations: 2,
		MaxToolDuration:   2 * time.Second,
		InitialSessionID:  scenario.SessionID,
		ChatKey:           scenario.ChatKey,
		Recall:            recall,
		RecallDeadline:    time.Second,
		ToolAudit:         audit.NewJSONLWriter(auditPath),
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
	waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool { return f.Phase == kernel.PhaseIdle }, time.Second)

	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: scenario.UserInput}); err != nil {
		t.Fatalf("submit turn: %v", err)
	}
	final := waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool {
		return f.Phase == kernel.PhaseIdle &&
			len(f.History) >= 2 &&
			f.History[len(f.History)-1].Role == "assistant"
	}, 5*time.Second)
	if got := final.History[len(final.History)-1].Content; got != scenario.FinalText {
		t.Fatalf("final assistant = %q, want %q", got, scenario.FinalText)
	}

	persisted := waitGoldenPersistedTurns(t, mem.DB(), scenario.ChatKey, scenario.UserInput, scenario.FinalText)
	requests := normalizeGoldenRequests(provider.Requests())
	toolExecution := normalizeGoldenToolExecution(requests, readAuditRecords(t, auditPath))
	recallTerms := recallTermsFromRequests(requests)

	return goldenTranscript{
		Fixture:        scenario.Name,
		UserInput:      scenario.UserInput,
		FinalAssistant: scenario.FinalText,
		Status: goldenStatus{
			Phase:         final.Phase.String(),
			StatusText:    final.StatusText,
			SessionID:     final.SessionID,
			HistoryLength: len(final.History),
		},
		ProviderRequests: requests,
		ToolExecution:    toolExecution,
		Memory: goldenMemoryEvidence{
			RecallInjected: len(recallTerms) > 0,
			RecallTerms:    recallTerms,
			PersistedTurns: persisted,
		},
		Audit: normalizeGoldenAudit(readAuditRecords(t, auditPath)),
	}
}

func assertGoldenTranscript(t *testing.T, name string, got goldenTranscript) {
	t.Helper()
	path := filepath.Join("testdata", "normal_turn", name+".json")
	gotRaw, err := transcript.MarshalStableJSON(got)
	if err != nil {
		t.Fatalf("marshal generated transcript: %v", err)
	}
	if os.Getenv("GORMES_UPDATE_GOLDEN_TRANSCRIPTS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create golden transcript dir: %v", err)
		}
		if err := os.WriteFile(path, gotRaw, 0o644); err != nil {
			t.Fatalf("write golden transcript %s: %v", path, err)
		}
		return
	}
	wantRaw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture_missing %s at $: %s", name, path)
	}
	if err != nil {
		t.Fatalf("read golden transcript %s: %v", path, err)
	}
	if err := transcript.CompareGoldenJSON(wantRaw, gotRaw); err != nil {
		var diff transcript.JSONDiff
		if errors.As(err, &diff) {
			t.Fatalf("%s %s at %s: %s", classifyGoldenMismatch(diff.Path), name, diff.Path, diff.Message)
		}
		t.Fatalf("transcript_mismatch %s: %v", name, err)
	}
}

func normalizeGoldenRequests(requests []hermes.ChatRequest) []goldenProviderRequest {
	out := make([]goldenProviderRequest, 0, len(requests))
	for _, req := range requests {
		tools := make([]string, 0, len(req.Tools))
		for _, desc := range req.Tools {
			tools = append(tools, desc.Name)
		}
		messages := make([]goldenMessage, 0, len(req.Messages))
		for _, msg := range req.Messages {
			messages = append(messages, goldenMessage{
				Role:       msg.Role,
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				Name:       msg.Name,
				ToolCalls:  normalizeGoldenToolCalls(msg.ToolCalls),
			})
		}
		out = append(out, goldenProviderRequest{
			Model:     req.Model,
			SessionID: req.SessionID,
			Stream:    req.Stream,
			Messages:  messages,
			Tools:     tools,
		})
	}
	return out
}

func normalizeGoldenToolExecution(requests []goldenProviderRequest, records []audit.Record) []goldenToolExecution {
	if len(requests) < 2 {
		return []goldenToolExecution{}
	}
	auditByTool := make(map[string]audit.Record, len(records))
	for _, rec := range records {
		auditByTool[rec.Tool] = rec
	}
	var out []goldenToolExecution
	for _, msg := range requests[len(requests)-1].Messages {
		if msg.Role != "tool" {
			continue
		}
		exec := goldenToolExecution{
			ID:     msg.ToolCallID,
			Name:   msg.Name,
			Result: json.RawMessage(msg.Content),
		}
		for _, prior := range requests[len(requests)-1].Messages {
			if prior.Role != "assistant" {
				continue
			}
			for _, call := range prior.ToolCalls {
				if call.ID == msg.ToolCallID {
					exec.Args = call.Arguments
					break
				}
			}
		}
		if rec, ok := auditByTool[msg.Name]; ok && rec.Status == "completed" {
			exec.Status = rec.Status
			exec.Audited = true
		}
		out = append(out, exec)
	}
	return out
}

func normalizeGoldenAudit(records []audit.Record) []goldenAuditRecord {
	out := make([]goldenAuditRecord, 0, len(records))
	for _, rec := range records {
		out = append(out, goldenAuditRecord{
			Source:          rec.Source,
			SessionID:       rec.SessionID,
			Tool:            rec.Tool,
			Args:            rec.Args,
			Status:          rec.Status,
			ResultSizeBytes: rec.ResultSizeBytes,
			Error:           rec.Error,
		})
	}
	return out
}

func waitGoldenPersistedTurns(t *testing.T, db *sql.DB, chatKey, userInput, finalText string) []goldenPersistedTurn {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		turns := queryGoldenPersistedTurns(t, db, chatKey, userInput, finalText)
		if len(turns) == 2 && turns[0].MemorySyncStatus == "ready" && turns[1].MemorySyncStatus == "ready" {
			return turns
		}
		if time.Now().After(deadline) {
			t.Fatalf("persisted ready turns not found for %q/%q: %+v", userInput, finalText, turns)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func queryGoldenPersistedTurns(t *testing.T, db *sql.DB, chatKey, userInput, finalText string) []goldenPersistedTurn {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		`SELECT role, content, memory_sync_status, COALESCE(meta_json, '')
		   FROM turns
		  WHERE chat_id = ? AND content IN (?, ?)
		  ORDER BY id`,
		chatKey, userInput, finalText,
	)
	if err != nil {
		t.Fatalf("query persisted turns: %v", err)
	}
	defer rows.Close()

	var out []goldenPersistedTurn
	for rows.Next() {
		var turn goldenPersistedTurn
		var meta string
		if err := rows.Scan(&turn.Role, &turn.Content, &turn.MemorySyncStatus, &meta); err != nil {
			t.Fatalf("scan persisted turn: %v", err)
		}
		turn.ToolCalls = toolCallsFromMeta(t, meta)
		out = append(out, turn)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate persisted turns: %v", err)
	}
	return out
}

func toolCallsFromMeta(t *testing.T, meta string) []goldenToolCall {
	t.Helper()
	if strings.TrimSpace(meta) == "" {
		return nil
	}
	var decoded struct {
		ToolCalls []hermes.ToolCall `json:"tool_calls"`
	}
	if err := json.Unmarshal([]byte(meta), &decoded); err != nil {
		t.Fatalf("decode persisted meta_json: %v", err)
	}
	return normalizeGoldenToolCalls(decoded.ToolCalls)
}

func normalizeGoldenToolCalls(calls []hermes.ToolCall) []goldenToolCall {
	out := make([]goldenToolCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, goldenToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}
	return out
}

func recallTermsFromRequests(requests []goldenProviderRequest) []string {
	if len(requests) == 0 || len(requests[0].Messages) == 0 {
		return nil
	}
	content := requests[0].Messages[0].Content
	if requests[0].Messages[0].Role != "system" || !strings.Contains(content, "Goncho memory context") {
		return nil
	}
	terms := []string{"Goncho memory context"}
	for _, candidate := range []string{"evidence-first", "exact"} {
		if strings.Contains(content, candidate) {
			terms = append(terms, candidate)
		}
	}
	return terms
}

func classifyGoldenMismatch(path string) string {
	switch {
	case strings.HasPrefix(path, "$.memory"):
		return "memory_evidence_mismatch"
	case strings.HasPrefix(path, "$.tool_execution"):
		return "tool_continuation_mismatch"
	case strings.Contains(path, ".tool_calls") || strings.Contains(path, ".tool_call_id"):
		return "tool_continuation_mismatch"
	case strings.HasPrefix(path, "$.provider_requests"):
		return "provider_part_mismatch"
	default:
		return "transcript_mismatch"
	}
}

type goldenEchoTool struct{}

func (goldenEchoTool) Name() string        { return "golden_echo" }
func (goldenEchoTool) Description() string { return "stable golden transcript echo tool" }
func (goldenEchoTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`)
}
func (goldenEchoTool) Timeout() time.Duration { return time.Second }
func (goldenEchoTool) Execute(context.Context, json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{"observation":"stable fixture observation","source":"golden_echo"}`), nil
}
