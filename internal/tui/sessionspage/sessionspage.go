package sessionspage

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/transientpage"
)

type Entry struct {
	ID           string
	Title        string
	Preview      string
	Source       string
	LastActiveAt int64
	MessageCount int
}

func SlashName(input string) string {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(fields[0], "/"))
}

func SlashArg(input string) string {
	trimmed := strings.TrimSpace(input)
	fields := strings.Fields(trimmed)
	if len(fields) <= 1 {
		return ""
	}
	idx := strings.Index(trimmed, fields[1])
	if idx < 0 {
		return strings.Join(fields[1:], " ")
	}
	return strings.TrimSpace(trimmed[idx:])
}

func Limit(input string) int {
	fields := strings.Fields(strings.TrimSpace(input))
	limit := 20
	if len(fields) > 1 {
		if n, err := strconv.Atoi(fields[1]); err == nil {
			limit = n
		}
	}
	if limit < 1 {
		return 1
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func Build(entries []Entry) (transientpage.State, bool) {
	if len(entries) == 0 {
		return transientpage.State{}, false
	}
	blocks := make([]string, 0, len(entries))
	for i, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			id = "(unknown session)"
		}
		title := firstNonEmptyString(strings.TrimSpace(entry.Title), strings.TrimSpace(entry.Preview), id)
		preview := strings.TrimSpace(entry.Preview)
		if preview == "" {
			preview = "(no preview)"
		}
		meta := []string{MessageCountLabel(entry.MessageCount)}
		if source := strings.TrimSpace(entry.Source); source != "" {
			meta = append(meta, "source: "+source)
		}
		if when := TimeLabel(entry.LastActiveAt); when != "" {
			meta = append(meta, "last active: "+when)
		}
		blocks = append(blocks, fmt.Sprintf("%2d. %s\nID: %s\nPreview: %s\n%s", i+1, title, id, preview, strings.Join(meta, " · ")))
	}
	return transientpage.State{Title: "Sessions", Body: strings.Join(blocks, "\n\n")}, true
}

func ResumeSuccessStatus(sessionID string, messages int) string {
	return fmt.Sprintf("resumed %s (%s)", sessionID, MessageCountLabel(messages))
}

func CloneResumeHistory(in []llm.Message) []llm.Message {
	if len(in) == 0 {
		return nil
	}
	out := make([]llm.Message, len(in))
	for i, msg := range in {
		out[i] = msg
		out[i].ContentParts = append([]llm.MessageContentPart(nil), msg.ContentParts...)
	}
	return out
}

func MessageCountLabel(count int) string {
	if count == 1 {
		return "1 message"
	}
	if count < 0 {
		count = 0
	}
	return fmt.Sprintf("%d messages", count)
}

func TimeLabel(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format("2006-01-02 15:04 UTC")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
