package gateway

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestRenderCompatibilityWrappersStayAvailable(t *testing.T) {
	frame := kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "hello"}
	if got := FormatStreamPlain(frame); got != "hello ▉" {
		t.Fatalf("FormatStreamPlain wrapper = %q, want rendering implementation", got)
	}
	if got := FormatFinalTelegramText("Use a_b(c)!"); got != `Use a\_b\(c\)\!` {
		t.Fatalf("FormatFinalTelegramText wrapper = %q, want Telegram MarkdownV2 output", got)
	}
}
