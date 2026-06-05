package callbacks

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestApprovalTextEscapesAndTruncates(t *testing.T) {
	text := ApprovalText("echo <danger>", strings.Repeat("needs review ", 300))
	if len([]rune(text)) > ApprovalTextLimit {
		t.Fatalf("text length = %d, want <= %d", len([]rune(text)), ApprovalTextLimit)
	}
	if !strings.Contains(text, "echo &lt;danger&gt;") {
		t.Fatalf("text = %q, want escaped command", text)
	}
	if !strings.Contains(text, "...") {
		t.Fatalf("text = %q, want truncation marker", text)
	}
}

func TestApprovalKeyboardAndParseCallbackData(t *testing.T) {
	markup := ApprovalKeyboard(42)
	if got := len(markup.InlineKeyboard); got != 2 {
		t.Fatalf("rows = %d, want 2", got)
	}
	button := markup.InlineKeyboard[0][0]
	if button.CallbackData == nil {
		t.Fatal("callback data nil")
	}
	choice, id, ok := ParseApprovalCallbackData(*button.CallbackData)
	if !ok || choice != gateway.ApprovalChoiceOnce || id != 42 {
		t.Fatalf("parsed = %q/%d/%v, want once/42/true", choice, id, ok)
	}
	if _, _, ok := ParseApprovalCallbackData("ea:bogus:42"); ok {
		t.Fatal("bogus choice parsed successfully")
	}
}

func TestApprovalDecisionLabelAndCallbackActor(t *testing.T) {
	if got := ApprovalDecisionLabel(gateway.ApprovalChoiceDeny); got != "❌ Denied" {
		t.Fatalf("deny label = %q", got)
	}
	actorID, actorName := CallbackActor(&tgbotapi.CallbackQuery{From: &tgbotapi.User{ID: 7, UserName: "fallback"}})
	if actorID != "7" || actorName != "fallback" {
		t.Fatalf("actor = %q/%q, want 7/fallback", actorID, actorName)
	}
}
