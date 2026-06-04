package titlewiring

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// BuildTitleModelFunc wraps a llm.Client as a llm.TitleModelFunc. It
// opens an SSE stream, collects EventToken fragments until EventDone, and
// returns the concatenated text. The call is bounded by ctx; provider errors
// surface as non-nil errors so PerformAutoTitle records provider_failed
// evidence without retry.
//
// model is used as the ChatRequest model when non-empty; an empty model falls
// back to the server-configured default.
func BuildTitleModelFunc(client llm.Client, model string) llm.TitleModelFunc {
	return func(ctx context.Context, req llm.TitleModelRequest) (string, error) {
		msgs := make([]llm.Message, 0, len(req.Messages))
		for _, m := range req.Messages {
			msgs = append(msgs, llm.Message{Role: m.Role, Content: m.Content})
		}
		chatReq := llm.ChatRequest{
			Model:    model,
			Messages: msgs,
			Stream:   true,
		}
		stream, err := client.OpenStream(ctx, chatReq)
		if err != nil {
			return "", err
		}
		defer stream.Close() //nolint:errcheck // best-effort close

		var buf strings.Builder
		for {
			ev, err := stream.Recv(ctx)
			if err != nil {
				return "", err
			}
			switch ev.Kind {
			case llm.EventToken:
				buf.WriteString(ev.Token)
			case llm.EventDone:
				return buf.String(), nil
			}
		}
	}
}

// BuildGatewayTitleSeam extracts a SessionTitleStore and TitleModelFunc from
// the live session.Map and provider client. The store is nil when smap is not
// a *session.BoltMap (e.g., MemMap in tests); in that case auto-title
// silently skips via the nil-store short-circuit in maybeRunAutoTitle.
func BuildGatewayTitleSeam(ctx context.Context, smap session.Map, client llm.Client, model string) (session.SessionTitleStore, llm.TitleModelFunc) {
	boltMap, ok := smap.(*session.BoltMap)
	if !ok {
		return nil, nil
	}
	store := session.NewMetadataTitleStore(ctx, boltMap)
	titleModel := BuildTitleModelFunc(client, model)
	return store, titleModel
}
