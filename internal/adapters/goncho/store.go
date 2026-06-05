package goncho

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	extgoncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// NewStore adapts the external Goncho service to the kernel GonchoStore
// interface used by Gormes channel and gateway runtimes.
func NewStore(svc *extgoncho.Service) kernel.GonchoStore {
	return &Store{svc: svc}
}

// Store bridges kernel memory hooks to the external Goncho service.
type Store struct{ svc *extgoncho.Service }

func (a *Store) AppendTurn(ctx context.Context, peer, sessionKey, role, content string) error {
	if a == nil || a.svc == nil || sessionKey == "" || content == "" {
		return nil
	}
	_, err := a.svc.CreateMessages(ctx, extgoncho.CreateMessagesParams{
		SessionKey: sessionKey,
		Messages:   []extgoncho.CreateMessage{{Peer: peer, Role: role, Content: content}},
	})
	return err
}

func (a *Store) GetContext(ctx context.Context, sessionKey string, maxTokens int) (string, error) {
	if a == nil || a.svc == nil || sessionKey == "" {
		return "", nil
	}
	result, err := a.svc.Context(ctx, extgoncho.ContextParams{
		Peer:       "gormes",
		SessionKey: sessionKey,
		MaxTokens:  maxTokens,
	})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, m := range result.RecentMessages {
		role := "User"
		if m.Role == "assistant" {
			role = "Gormes"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func (a *Store) OnSessionEnd(ctx context.Context, sessionKey string, messages []llm.Message) error {
	if a == nil || a.svc == nil || sessionKey == "" {
		return nil
	}
	gonchoMsgs := make([]extgoncho.Message, len(messages))
	for i, m := range messages {
		gonchoMsgs[i] = extgoncho.Message{Role: m.Role, Content: m.Content}
	}
	return a.svc.OnSessionEnd(ctx, sessionKey, gonchoMsgs)
}

func (a *Store) Observe(ctx context.Context, obs kernel.GonchoObservation) error {
	if a == nil || a.svc == nil {
		return nil
	}
	success := obs.Success
	_, err := a.svc.Observe(ctx, extgoncho.ObservationParams{
		ID:         gormesObservationID(obs),
		Kind:       mapGormesObservationKind(obs.Kind),
		PeerID:     obs.PeerID,
		SessionKey: obs.SessionKey,
		ContextID:  obs.ContextID,
		Input:      obs.Input,
		Output:     obs.Output,
		Success:    success,
		Metadata:   cloneGormesObservationMetadata(obs.Metadata),
		ObservedAt: obs.ObservedAt,
		Reason:     obs.Reason,
	})
	return err
}

func mapGormesObservationKind(kind kernel.GonchoObservationKind) extgoncho.ObservationKind {
	switch kind {
	case kernel.GonchoObservationSessionStart:
		return extgoncho.ObservationKindSessionStart
	case kernel.GonchoObservationUserPrompt:
		return extgoncho.ObservationKindUserPrompt
	case kernel.GonchoObservationToolCall:
		return extgoncho.ObservationKindToolCall
	case kernel.GonchoObservationToolResult:
		return extgoncho.ObservationKindToolResult
	case kernel.GonchoObservationToolError:
		return extgoncho.ObservationKindToolError
	case kernel.GonchoObservationAssistantResponse:
		return extgoncho.ObservationKindAssistantResponse
	case kernel.GonchoObservationCompact:
		return extgoncho.ObservationKindCompact
	case kernel.GonchoObservationSessionEnd:
		return extgoncho.ObservationKindSessionEnd
	default:
		return extgoncho.ObservationKindCustom
	}
}

func gormesObservationID(obs kernel.GonchoObservation) string {
	h := sha256.New()
	write := func(value string) {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	write(string(obs.Kind))
	write(obs.PeerID)
	write(obs.SessionKey)
	write(obs.ContextID)
	write(obs.Input)
	write(obs.Output)
	if obs.Success == nil {
		write("success:nil")
	} else if *obs.Success {
		write("success:true")
	} else {
		write("success:false")
	}
	keys := make([]string, 0, len(obs.Metadata))
	for key := range obs.Metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		write(key)
		write(obs.Metadata[key])
	}
	sum := hex.EncodeToString(h.Sum(nil))
	return "gormes-" + string(obs.Kind) + "-" + sum[:24]
}

func cloneGormesObservationMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
