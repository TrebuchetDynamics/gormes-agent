package kernel_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho"
)

func TestAgentTurnUsesGonchoMemory(t *testing.T) {
	t.Run("recall_tools_and_final_persistence", func(t *testing.T) {
		mem, svc, cleanup := newKernelGonchoService(t)
		defer cleanup()
		seedKernelGonchoMemory(t, svc)

		reg := tools.NewRegistry()
		integration := gonchotools.NewTurnIntegration(svc, "telegram:6586915095")
		status := integration.RegisterTools(reg)
		if status.ToolsEvidence != "honcho_tools_ready" {
			t.Fatalf("tools evidence = %q, want honcho_tools_ready", status.ToolsEvidence)
		}

		provider := hermes.NewMockClient()
		provider.Script([]hermes.Event{{
			Kind:         hermes.EventDone,
			FinishReason: "tool_calls",
			ToolCalls: []hermes.ToolCall{{
				ID:        "call_honcho_search",
				Name:      "honcho_search",
				Arguments: json.RawMessage(`{"peer":"telegram:6586915095","query":"evidence-first","session_key":"sess-goncho-turn"}`),
			}},
			TokensIn:  40,
			TokensOut: 1,
		}}, "provider-goncho-turn")
		provider.Script([]hermes.Event{
			{Kind: hermes.EventToken, Token: "Goncho memory confirms evidence-first reporting.", TokensOut: 6},
			{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 80, TokensOut: 6},
		}, "provider-goncho-turn")

		k, done, cancel := runKernelForGonchoTurn(t, mem, provider, kernel.Config{
			Model:             "goncho-turn-model",
			Endpoint:          "mock://goncho-turn",
			Admission:         kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
			Tools:             reg,
			MaxToolIterations: 2,
			MaxToolDuration:   time.Second,
			InitialSessionID:  "sess-goncho-turn",
			ChatKey:           "telegram:6586915095",
			Recall:            integration.RecallProvider(),
			RecallDeadline:    time.Second,
		})
		defer stopKernelForGonchoTurn(t, done, cancel)

		final := submitAndWaitForGonchoFinal(t, k, "Use Goncho memory and tools for this turn.")
		if got := final.History[len(final.History)-1].Content; got != "Goncho memory confirms evidence-first reporting." {
			t.Fatalf("final assistant = %q, want Goncho answer", got)
		}
		status = integration.Status()
		if status.RecallEvidence != "goncho_recall_ready" {
			t.Fatalf("recall evidence = %q, want goncho_recall_ready", status.RecallEvidence)
		}

		requests := provider.Requests()
		if len(requests) != 2 {
			t.Fatalf("provider requests = %d, want initial + tool continuation", len(requests))
		}
		if !requestContainsGonchoRecall(requests[0]) {
			t.Fatalf("initial request messages = %+v, want Goncho recall system context", requests[0].Messages)
		}
		for _, toolName := range []string{"honcho_context", "honcho_search", "honcho_profile"} {
			if !requestHasTool(requests[0], toolName) {
				t.Fatalf("initial request tools = %+v, want %s", requests[0].Tools, toolName)
			}
		}
		if !continuationHasHonchoToolReply(requests[1], "call_honcho_search", "evidence-first") {
			t.Fatalf("continuation messages = %+v, want honcho_search tool reply", requests[1].Messages)
		}
		assertGonchoTurnPersistedOnce(t, mem.DB(), "assistant", "Goncho memory confirms evidence-first reporting.")
	})

	t.Run("degraded_recall_and_tools_report_evidence", func(t *testing.T) {
		mem, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 32, nil)
		if err != nil {
			t.Fatalf("OpenSqlite: %v", err)
		}
		defer func() {
			if err := mem.Close(context.Background()); err != nil {
				t.Fatalf("Close: %v", err)
			}
		}()

		reg := tools.NewRegistry()
		integration := gonchotools.NewTurnIntegration(nil, "telegram:6586915095")
		status := integration.RegisterTools(reg)
		if status.ToolsEvidence != "honcho_tools_unavailable" {
			t.Fatalf("tools evidence = %q, want honcho_tools_unavailable", status.ToolsEvidence)
		}

		provider := hermes.NewMockClient()
		provider.Script([]hermes.Event{
			{Kind: hermes.EventToken, Token: "Turn stayed explainable without Goncho.", TokensOut: 5},
			{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 10, TokensOut: 5},
		}, "provider-degraded")

		k, done, cancel := runKernelForGonchoTurn(t, mem, provider, kernel.Config{
			Model:            "goncho-turn-model",
			Endpoint:         "mock://goncho-turn",
			Admission:        kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
			Tools:            reg,
			InitialSessionID: "sess-goncho-degraded",
			ChatKey:          "telegram:6586915095",
			Recall:           integration.RecallProvider(),
			RecallDeadline:   time.Second,
		})
		defer stopKernelForGonchoTurn(t, done, cancel)

		final := submitAndWaitForGonchoFinal(t, k, "Continue without Goncho if needed.")
		if got := final.History[len(final.History)-1].Content; got != "Turn stayed explainable without Goncho." {
			t.Fatalf("final assistant = %q, want degraded answer", got)
		}
		status = integration.Status()
		if status.RecallEvidence != "goncho_recall_unavailable" {
			t.Fatalf("recall evidence = %q, want goncho_recall_unavailable", status.RecallEvidence)
		}
		if len(provider.Requests()[0].Tools) != 0 {
			t.Fatalf("degraded tools = %+v, want no honcho tool descriptors", provider.Requests()[0].Tools)
		}
		assertGonchoTurnPersistedOnce(t, mem.DB(), "assistant", "Turn stayed explainable without Goncho.")
	})
}

func newKernelGonchoService(t *testing.T) (*memory.SqliteStore, *goncho.Service, func()) {
	t.Helper()
	mem, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 32, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	svc := goncho.NewService(mem.DB(), goncho.Config{
		WorkspaceID:    "default",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)
	return mem, svc, func() {
		if err := mem.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

func seedKernelGonchoMemory(t *testing.T, svc *goncho.Service) {
	t.Helper()
	ctx := context.Background()
	if err := svc.SetProfile(ctx, "telegram:6586915095", []string{"Prefers exact evidence-first reports"}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       "telegram:6586915095",
		Conclusion: "The user prefers exact evidence-first reports.",
		SessionKey: "sess-goncho-turn",
	}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if _, err := svc.CreateMessages(ctx, goncho.CreateMessagesParams{
		SessionKey: "sess-goncho-turn",
		Messages: []goncho.CreateMessage{{
			Peer:    "telegram:6586915095",
			Role:    "user",
			Content: "Please keep reports exact and evidence-first.",
		}},
	}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}
}

func runKernelForGonchoTurn(t *testing.T, mem *memory.SqliteStore, provider *hermes.MockClient, cfg kernel.Config) (*kernel.Kernel, chan error, context.CancelFunc) {
	t.Helper()
	k := kernel.New(cfg, provider, mem, telemetry.New(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- k.Run(ctx)
	}()
	waitForGonchoFrame(t, k.Render(), func(f kernel.RenderFrame) bool { return f.Phase == kernel.PhaseIdle }, time.Second)
	return k, done, cancel
}

func stopKernelForGonchoTurn(t *testing.T, done chan error, cancel context.CancelFunc) {
	t.Helper()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("kernel run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("kernel did not stop after cleanup cancel")
	}
}

func submitAndWaitForGonchoFinal(t *testing.T, k *kernel.Kernel, text string) kernel.RenderFrame {
	t.Helper()
	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: text}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return waitForGonchoFrame(t, k.Render(), func(f kernel.RenderFrame) bool {
		return f.Phase == kernel.PhaseIdle &&
			len(f.History) >= 2 &&
			f.History[len(f.History)-1].Role == "assistant"
	}, 5*time.Second)
}

func waitForGonchoFrame(t *testing.T, ch <-chan kernel.RenderFrame, pred func(kernel.RenderFrame) bool, timeout time.Duration) kernel.RenderFrame {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var last kernel.RenderFrame
	for {
		select {
		case frame, ok := <-ch:
			if !ok {
				t.Fatalf("render channel closed before expected frame; last=%+v", last)
			}
			last = frame
			if pred(frame) {
				return frame
			}
		case <-timer.C:
			t.Fatalf("timeout waiting for frame; last=%+v", last)
		}
	}
}

func requestContainsGonchoRecall(req hermes.ChatRequest) bool {
	if len(req.Messages) == 0 || req.Messages[0].Role != "system" {
		return false
	}
	return strings.Contains(req.Messages[0].Content, "Goncho memory context") &&
		strings.Contains(req.Messages[0].Content, "evidence-first")
}

func requestHasTool(req hermes.ChatRequest, name string) bool {
	for _, desc := range req.Tools {
		if desc.Name == name {
			return true
		}
	}
	return false
}

func continuationHasHonchoToolReply(req hermes.ChatRequest, callID, want string) bool {
	for _, msg := range req.Messages {
		if msg.Role == "tool" && msg.ToolCallID == callID && strings.Contains(msg.Content, want) {
			return true
		}
	}
	return false
}

func assertGonchoTurnPersistedOnce(t *testing.T, db *sql.DB, role, content string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var count int
		err := db.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM turns WHERE role = ? AND content = ? AND memory_sync_status = 'ready'`,
			role, content,
		).Scan(&count)
		if err == nil && count == 1 {
			return
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("query persisted turn: %v", err)
			}
			t.Fatalf("persisted %s turn count = %d, want 1 for %q", role, count, content)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
