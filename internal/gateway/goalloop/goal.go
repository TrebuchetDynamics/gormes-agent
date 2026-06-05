package goalloop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

const (
	JudgeVerdictDone     = "done"
	JudgeVerdictContinue = "continue"
	JudgeVerdictSkipped  = "skipped"
)

type Judge interface {
	JudgeGoal(context.Context, string, string) (string, error)
}

type JudgeFunc func(context.Context, string, string) (string, error)

func (f JudgeFunc) JudgeGoal(ctx context.Context, goal, lastResponse string) (string, error) {
	return f(ctx, goal, lastResponse)
}

type JudgeVerdict struct {
	Verdict string
	Reason  string
}

func EvaluateJudge(ctx context.Context, judge Judge, goal, lastResponse string) JudgeVerdict {
	if strings.TrimSpace(goal) == "" {
		return JudgeVerdict{Verdict: JudgeVerdictSkipped, Reason: "empty goal"}
	}
	if strings.TrimSpace(lastResponse) == "" {
		return JudgeVerdict{Verdict: JudgeVerdictContinue, Reason: "empty response (nothing to evaluate)"}
	}
	if judge == nil {
		return JudgeVerdict{Verdict: JudgeVerdictContinue, Reason: "auxiliary client unavailable"}
	}
	raw, err := judge.JudgeGoal(ctx, goal, lastResponse)
	if err != nil {
		return JudgeVerdict{Verdict: JudgeVerdictContinue, Reason: "judge error: " + err.Error()}
	}
	return ParseJudgeResponse(raw)
}

func ParseJudgeResponse(raw string) JudgeVerdict {
	if strings.TrimSpace(raw) == "" {
		return JudgeVerdict{Verdict: JudgeVerdictContinue, Reason: "judge returned empty response"}
	}
	text := judgeResponseJSONCandidate(raw)

	var data struct {
		Done   any    `json:"done"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return JudgeVerdict{Verdict: JudgeVerdictContinue, Reason: fmt.Sprintf("judge reply was not JSON: %q", truncateReason(raw, 200))}
	}
	reason := strings.TrimSpace(data.Reason)
	if reason == "" {
		reason = "no reason provided"
	}
	if judgeDone(data.Done) {
		return JudgeVerdict{Verdict: JudgeVerdictDone, Reason: reason}
	}
	return JudgeVerdict{Verdict: JudgeVerdictContinue, Reason: reason}
}

func judgeResponseJSONCandidate(raw string) string {
	text := strings.TrimSpace(raw)
	if body, ok := fencedJudgeResponseBody(text); ok {
		text = body
	}
	if start := strings.IndexByte(text, '{'); start >= 0 {
		if end := strings.LastIndexByte(text, '}'); end >= start {
			return strings.TrimSpace(text[start : end+1])
		}
	}
	return text
}

func fencedJudgeResponseBody(text string) (string, bool) {
	if !strings.HasPrefix(text, "```") {
		return "", false
	}
	firstLineEnd := strings.IndexByte(text, '\n')
	if firstLineEnd < 0 {
		return "", false
	}
	body := text[firstLineEnd+1:]
	if fence := strings.Index(body, "```"); fence >= 0 {
		body = body[:fence]
	}
	return strings.TrimSpace(body), true
}

func judgeDone(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "yes", "1", "done":
			return true
		default:
			return false
		}
	case float64:
		return v != 0
	default:
		return false
	}
}

func truncateReason(s string, limit int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func CommandArgs(text string) string {
	body := strings.TrimSpace(text)
	if body == "" {
		return ""
	}
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return ""
	}
	first := strings.TrimPrefix(strings.ToLower(fields[0]), "/")
	if first != "goal" {
		return body
	}
	if len(body) <= len(fields[0]) {
		return ""
	}
	return strings.TrimSpace(body[len(fields[0]):])
}

func FormatStatusLine(state *session.GoalState) string {
	if state == nil {
		return "No goal set."
	}
	status := strings.ToLower(strings.TrimSpace(string(state.Status)))
	if status == "" {
		status = string(session.GoalStatusActive)
	}
	line := fmt.Sprintf("Goal %s (%d/%d): %s", status, state.TurnsUsed, state.MaxTurns, state.Goal)
	if reason := strings.TrimSpace(state.LastReason); reason != "" {
		line += "\nLast verdict: " + strings.TrimSpace(state.LastVerdict) + " — " + reason
	}
	if reason := strings.TrimSpace(state.PausedReason); reason != "" {
		line += "\nPaused reason: " + reason
	}
	return line
}
