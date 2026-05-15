package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestRenderConvViewportBinding_UsesHeightBudget(t *testing.T) {
	history := make([]hermes.Message, 0, 120)
	for i := 0; i < 120; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		history = append(history, hermes.Message{
			Role:    role,
			Content: fmt.Sprintf("turn-%03d-body-marker", i),
		})
	}

	got := renderConv(kernel.RenderFrame{History: history}, 80, 8)

	for _, want := range []string{
		"turn-117-body-marker",
		"turn-118-body-marker",
		"turn-119-body-marker",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderConv() missing latest turn %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "turn-000-body-marker") {
		t.Fatalf("renderConv() included earliest turn body in:\n%s", got)
	}
	if !strings.Contains(got, "earlier history messages omitted") {
		t.Fatalf("renderConv() missing omitted-history sentinel in:\n%s", got)
	}
}

func TestRenderConvViewportBinding_DraftAndErrorPreserved(t *testing.T) {
	history := make([]hermes.Message, 0, 120)
	for i := 0; i < 120; i++ {
		history = append(history, hermes.Message{
			Role:    "user",
			Content: fmt.Sprintf("history-%03d", i),
		})
	}
	frame := kernel.RenderFrame{
		History:   history,
		DraftText: "render-conv draft survives clipping",
		LastError: "render-conv last error survives clipping",
	}

	got := renderConv(frame, 72, 6)

	for _, want := range []string{
		"render-conv draft survives clipping",
		"render-conv last error survives clipping",
		"earlier history messages omitted",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderConv() missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "history-000") {
		t.Fatalf("renderConv() included earliest history under clipping in:\n%s", got)
	}
}

func TestRenderConvViewportBinding_TinyTerminalCompactFallback(t *testing.T) {
	frame := kernel.RenderFrame{
		History: []hermes.Message{
			{Role: "user", Content: "earliest body should stay hidden"},
			{Role: "assistant", Content: "latest tiny terminal marker"},
		},
		DraftText: "tiny draft",
		LastError: "tiny error",
	}

	got := renderConv(frame, 5, 1)

	if !strings.Contains(got, "latest") {
		t.Fatalf("renderConv() compact tiny view did not include latest marker in:\n%s", got)
	}
	if strings.Contains(got, "earliest body should stay hidden") {
		t.Fatalf("renderConv() compact tiny view leaked clipped earliest body in:\n%s", got)
	}
	if !strings.Contains(got, "tiny draft") {
		t.Fatalf("renderConv() compact tiny view dropped DraftText in:\n%s", got)
	}
	if !strings.Contains(got, "tiny error") {
		t.Fatalf("renderConv() compact tiny view dropped LastError in:\n%s", got)
	}
}

func TestRenderConvViewportBinding_WrapsProviderAuthError(t *testing.T) {
	const width = 72
	frame := kernel.RenderFrame{
		History: []hermes.Message{
			{Role: "user", Content: "as"},
		},
		LastError: "Unauthorized: Your authentication token has been invalidated. Please reauthenticate with the provider before continuing.",
	}

	got := renderConv(frame, width, 8)

	for _, want := range []string{
		"Unauthorized: Your authentication token has been invalidated.",
		"Please reauthenticate with the provider before continuing.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderConv() missing wrapped auth error text %q in:\n%s", want, got)
		}
	}
	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Fatalf("renderConv() error line width = %d, want <= %d:\n%q\n\nfull output:\n%s", w, width, line, got)
		}
	}
}

func TestRenderConvViewportBinding_RenderedLineBudget(t *testing.T) {
	history := make([]hermes.Message, 0, 120)
	for i := 0; i < 120; i++ {
		history = append(history, hermes.Message{
			Role:    "assistant",
			Content: fmt.Sprintf("budget-turn-%03d", i),
		})
	}
	frame := kernel.RenderFrame{
		History:   history,
		DraftText: "budget draft evidence",
		LastError: "budget error evidence",
	}
	const height = 8

	got := renderConv(frame, 80, height)

	for _, want := range []string{
		"budget-turn-119",
		"budget draft evidence",
		"budget error evidence",
		"earlier history messages omitted",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderConv() missing %q in:\n%s", want, got)
		}
	}
	if gotLines, maxLines := renderedLineCount(got), height+3; gotLines > maxLines {
		t.Fatalf("renderConv() rendered %d lines, want <= %d:\n%s", gotLines, maxLines, got)
	}
}

func TestRenderConvViewportBinding_EmptyFramePlaceholder(t *testing.T) {
	got := renderConv(kernel.RenderFrame{}, 80, 8)
	if !strings.Contains(got, "Type your message or /help for commands.") {
		t.Fatalf("renderConv() empty frame missing intro copy in:\n%s", got)
	}
	if strings.Contains(got, "start typing below to begin") {
		t.Fatalf("renderConv() empty frame leaked legacy placeholder in:\n%s", got)
	}
}
