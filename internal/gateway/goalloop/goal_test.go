package goalloop

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func TestEvaluateJudgeFailOpen(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		judge Judge
		want  string
	}{
		{name: "missing auxiliary client", judge: nil, want: JudgeVerdictContinue},
		{name: "empty judge response", judge: JudgeFunc(func(context.Context, string, string) (string, error) { return "", nil }), want: JudgeVerdictContinue},
		{name: "malformed judge response", judge: JudgeFunc(func(context.Context, string, string) (string, error) { return "not json", nil }), want: JudgeVerdictContinue},
		{name: "judge error", judge: JudgeFunc(func(context.Context, string, string) (string, error) { return "", errors.New("boom") }), want: JudgeVerdictContinue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateJudge(ctx, tc.judge, "ship it", "made progress")
			if got.Verdict != tc.want {
				t.Fatalf("EvaluateJudge verdict = %q, want %q (reason=%q)", got.Verdict, tc.want, got.Reason)
			}
			if strings.TrimSpace(got.Reason) == "" {
				t.Fatal("EvaluateJudge reason is empty, want fail-open evidence")
			}
		})
	}

	done := EvaluateJudge(ctx, JudgeFunc(func(context.Context, string, string) (string, error) {
		return "```json\n{\"done\": \"yes\", \"reason\": \"delivered\"}\n```", nil
	}), "ship it", "delivered")
	if done.Verdict != JudgeVerdictDone || done.Reason != "delivered" {
		t.Fatalf("EvaluateJudge done = %+v, want done delivered", done)
	}
}

func TestCommandArgs(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"/goal":            "",
		"/goal ship it":    "ship it",
		"goal   ship it":   "ship it",
		"ship it directly": "ship it directly",
	}
	for input, want := range cases {
		if got := CommandArgs(input); got != want {
			t.Fatalf("CommandArgs(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatStatusLine(t *testing.T) {
	if got := FormatStatusLine(nil); got != "No goal set." {
		t.Fatalf("FormatStatusLine(nil) = %q", got)
	}
	got := FormatStatusLine(&session.GoalState{
		Status:       session.GoalStatusPaused,
		Goal:         "ship it",
		TurnsUsed:    2,
		MaxTurns:     5,
		LastVerdict:  "continue",
		LastReason:   "more work",
		PausedReason: "operator asked",
	})
	for _, want := range []string{"Goal paused (2/5): ship it", "Last verdict: continue — more work", "Paused reason: operator asked"} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatStatusLine = %q, want substring %q", got, want)
		}
	}
}
