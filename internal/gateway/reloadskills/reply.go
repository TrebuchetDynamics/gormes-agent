package reloadskills

import (
	"fmt"
	"strings"
)

type RefreshResult struct {
	Channel string
	Count   int
	Hidden  int
	Error   string
}

type ReplyRequest struct {
	SkillCount int
	ScanError  string
	Refreshes  []RefreshResult
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func renderReplyField(value string) string {
	msg := strings.TrimSpace(value)
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	compact := compactSecretSeparators(lower)
	for _, marker := range []string{"api_key", "apikey", "authorization", "bearer", "secret", "password", "token="} {
		if strings.Contains(lower, marker) || strings.Contains(compact, marker) {
			return "[redacted]"
		}
	}
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"#", "＃",
	)
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func compactSecretSeparators(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func RenderReply(req ReplyRequest) string {
	scanError := renderReplyField(req.ScanError)
	degraded := scanError != ""
	for _, refresh := range req.Refreshes {
		if renderReplyField(refresh.Error) != "" {
			degraded = true
			break
		}
	}
	header := "Skills Reloaded"
	if degraded {
		header = "Skills reload degraded"
	}
	lines := []string{header}
	if scanError != "" {
		lines = append(lines, "skill scan: "+scanError)
	} else {
		lines = append(lines, fmt.Sprintf("%d skill(s) available", nonNegative(req.SkillCount)))
	}
	for _, refresh := range req.Refreshes {
		channel := renderReplyField(refresh.Channel)
		if channel == "" {
			channel = "unknown"
		}
		if errText := renderReplyField(refresh.Error); errText != "" {
			lines = append(lines, fmt.Sprintf("%s: refresh error: %s", channel, errText))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: refreshed %d command(s), %d hidden", channel, nonNegative(refresh.Count), nonNegative(refresh.Hidden)))
	}
	if len(req.Refreshes) == 0 {
		lines = append(lines, "adapter refresh: none")
	}
	return strings.Join(lines, "\n")
}
