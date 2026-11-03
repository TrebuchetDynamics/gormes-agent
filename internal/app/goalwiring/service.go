// Package goalwiring connects the gateway goal loop to the provider client.
package goalwiring

import (
	"context"
	"errors"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// JudgeSystemPrompt mirrors the Hermes goal judge contract in the subset the
// gateway loop consumes: an explicit done/continue verdict plus one reason.
const JudgeSystemPrompt = "You judge whether an autonomous agent has satisfied a user's goal. " +
	"Reply with ONLY one JSON object on one line, with no markdown: " +
	`{"done": true, "reason": "<one sentence>"}` + " or " +
	`{"done": false, "reason": "<one sentence>"}` + ". " +
	"done=true only when the response explicitly confirms completion or shows the final deliverable exists. " +
	"If the response is blocked, asks the user for input, or only reports progress, done=false."

const (
	maxJudgeResponseChars = 4000
	maxJudgeOutputBytes   = 64 * 1024
)

// BuildGoalJudgeFunc returns a gateway goal judge backed by the provider
// client. A nil client or a failed stream makes the judge fail open through
// the gateway's continue path.
func BuildGoalJudgeFunc(client llm.Client, model string) gateway.GoalJudge {
	model = strings.TrimSpace(model)
	return gateway.GoalJudgeFunc(func(ctx context.Context, goal, lastResponse string) (string, error) {
		if client == nil {
			return "", errors.New("goal judge client unavailable")
		}
		stream, err := client.OpenStream(ctx, llm.ChatRequest{
			Model:  model,
			Stream: true,
			Messages: []llm.Message{
				{Role: "system", Content: JudgeSystemPrompt},
				{Role: "user", Content: BuildJudgeUserPrompt(goal, lastResponse)},
			},
		})
		if err != nil {
			return "", err
		}
		defer stream.Close() //nolint:errcheck // best-effort close

		var buf strings.Builder
		for {
			event, err := stream.Recv(ctx)
			if err != nil {
				return "", err
			}
			switch event.Kind {
			case llm.EventToken:
				buf.WriteString(event.Token)
				if buf.Len() > maxJudgeOutputBytes {
					return "", errors.New("goal judge response exceeded output cap")
				}
			case llm.EventDone:
				return strings.TrimSpace(buf.String()), nil
			}
		}
	})
}

// BuildJudgeUserPrompt bounds the response snippet and preserves the goal.
func BuildJudgeUserPrompt(goal, lastResponse string) string {
	response := strings.TrimSpace(lastResponse)
	if len(response) > maxJudgeResponseChars {
		response = response[len(response)-maxJudgeResponseChars:]
	}
	return "Goal:\n" + strings.TrimSpace(goal) + "\n\nAgent's most recent response:\n" + response
}
