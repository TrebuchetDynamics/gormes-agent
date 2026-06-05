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

func RenderReply(req ReplyRequest) string {
	degraded := strings.TrimSpace(req.ScanError) != ""
	for _, refresh := range req.Refreshes {
		if strings.TrimSpace(refresh.Error) != "" {
			degraded = true
			break
		}
	}
	header := "Skills Reloaded"
	if degraded {
		header = "Skills reload degraded"
	}
	lines := []string{header}
	if req.ScanError != "" {
		lines = append(lines, "skill scan: "+req.ScanError)
	} else {
		lines = append(lines, fmt.Sprintf("%d skill(s) available", req.SkillCount))
	}
	for _, refresh := range req.Refreshes {
		channel := strings.TrimSpace(refresh.Channel)
		if channel == "" {
			channel = "unknown"
		}
		if strings.TrimSpace(refresh.Error) != "" {
			lines = append(lines, fmt.Sprintf("%s: refresh error: %s", channel, refresh.Error))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: refreshed %d command(s), %d hidden", channel, refresh.Count, refresh.Hidden))
	}
	if len(req.Refreshes) == 0 {
		lines = append(lines, "adapter refresh: none")
	}
	return strings.Join(lines, "\n")
}
