package callbacks

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const ApprovalTextLimit = 4096

func ApprovalText(command, description string) string {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		cmd = "(empty command)"
	}
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "dangerous command"
	}
	desc = truncateApprovalRunes(desc, 500)
	cmd = truncateApprovalRunes(cmd, 3800)

	for {
		text := fmt.Sprintf(
			"⚠️ <b>Command Approval Required</b>\n\n<pre>%s</pre>\n\nReason: %s",
			html.EscapeString(cmd),
			html.EscapeString(desc),
		)
		if len([]rune(text)) <= ApprovalTextLimit {
			return text
		}
		if len([]rune(desc)) > 120 {
			desc = truncateApprovalRunes(desc, 120)
			continue
		}
		cmd = truncateApprovalRunes(cmd, len([]rune(cmd))-128)
	}
}

func ApprovalKeyboard(approvalID uint64) tgbotapi.InlineKeyboardMarkup {
	id := strconv.FormatUint(approvalID, 10)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Allow Once", "ea:once:"+id),
			tgbotapi.NewInlineKeyboardButtonData("✅ Session", "ea:session:"+id),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Always", "ea:always:"+id),
			tgbotapi.NewInlineKeyboardButtonData("❌ Deny", "ea:deny:"+id),
		),
	)
}

func ParseApprovalCallbackData(data string) (gateway.ApprovalChoice, uint64, bool) {
	parts := strings.Split(strings.TrimSpace(data), ":")
	if len(parts) != 3 || parts[0] != "ea" {
		return "", 0, false
	}
	choice, ok := gateway.ParseApprovalChoice(parts[1])
	if !ok {
		return "", 0, false
	}
	id, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil || id == 0 {
		return "", 0, false
	}
	return choice, id, true
}

func ApprovalDecisionLabel(choice gateway.ApprovalChoice) string {
	switch choice {
	case gateway.ApprovalChoiceOnce:
		return "✅ Approved once"
	case gateway.ApprovalChoiceSession:
		return "✅ Approved for session"
	case gateway.ApprovalChoiceAlways:
		return "✅ Approved permanently"
	case gateway.ApprovalChoiceDeny:
		return "❌ Denied"
	default:
		return "Resolved"
	}
}

func CallbackActor(query *tgbotapi.CallbackQuery) (string, string) {
	if query == nil || query.From == nil {
		return "", ""
	}
	actorID := strconv.FormatInt(query.From.ID, 10)
	name := strings.TrimSpace(query.From.FirstName)
	if name == "" {
		name = strings.TrimSpace(query.From.UserName)
	}
	return actorID, name
}

func truncateApprovalRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
