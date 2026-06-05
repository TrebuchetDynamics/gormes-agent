package kernel

import (
	"context"
	"errors"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/goncho"
)

var ErrGonchoUnavailable = goncho.ErrUnavailable

type GonchoStore = goncho.Store

type GonchoObservationKind = goncho.ObservationKind

const (
	GonchoObservationSessionStart      = goncho.ObservationSessionStart
	GonchoObservationUserPrompt        = goncho.ObservationUserPrompt
	GonchoObservationToolCall          = goncho.ObservationToolCall
	GonchoObservationToolResult        = goncho.ObservationToolResult
	GonchoObservationToolError         = goncho.ObservationToolError
	GonchoObservationAssistantResponse = goncho.ObservationAssistantResponse
	GonchoObservationCompact           = goncho.ObservationCompact
	GonchoObservationSessionEnd        = goncho.ObservationSessionEnd
	GonchoObservationCustom            = goncho.ObservationCustom
)

type GonchoObservation = goncho.Observation

func (k *Kernel) writeGonchoUserTurn(ctx context.Context, text, turnKey string) {
	if k.cfg.Goncho == nil {
		return
	}
	if err := k.cfg.Goncho.AppendTurn(ctx, "user", k.sessionID, "user", text); err != nil {
		k.log.Warn("goncho user turn write failed", "err", err)
	}
	k.observeGoncho(ctx, GonchoObservation{
		Kind:       GonchoObservationUserPrompt,
		PeerID:     k.gonchoUserPeerID(),
		SessionKey: k.sessionID,
		ContextID:  turnKey,
		Input:      text,
		Metadata:   k.gonchoTurnMetadata("user", turnKey),
		Reason:     "gormes user prompt capture",
	})
}

func (k *Kernel) writeGonchoAssistantTurn(ctx context.Context, content, turnKey string) {
	if k.cfg.Goncho == nil {
		return
	}
	if err := k.cfg.Goncho.AppendTurn(ctx, "gormes", k.sessionID, "assistant", content); err != nil {
		k.log.Warn("goncho assistant turn write failed", "err", err)
	}
	success := true
	k.observeGoncho(ctx, GonchoObservation{
		Kind:       GonchoObservationAssistantResponse,
		PeerID:     "gormes",
		SessionKey: k.sessionID,
		ContextID:  turnKey,
		Output:     content,
		Success:    &success,
		Metadata:   k.gonchoTurnMetadata("assistant", turnKey),
		Reason:     "gormes assistant response capture",
	})
}

func (k *Kernel) gonchoContext(ctx context.Context) string {
	if k.cfg.Goncho == nil {
		return ""
	}
	ctxStr, err := k.cfg.Goncho.GetContext(ctx, k.sessionID, 2000)
	if err != nil {
		if !errors.Is(err, ErrGonchoUnavailable) {
			k.log.Warn("goncho context read failed", "err", err)
		}
		return ""
	}
	if ctxStr == "" {
		return ""
	}
	return fmt.Sprintf("## Recent Conversation History\n%s", ctxStr)
}

func (k *Kernel) observeGoncho(ctx context.Context, obs GonchoObservation) {
	if k.cfg.Goncho == nil {
		return
	}
	if obs.SessionKey == "" {
		obs.SessionKey = k.sessionID
	}
	if err := k.cfg.Goncho.Observe(ctx, obs); err != nil {
		k.log.Warn("goncho observation write failed", "kind", obs.Kind, "err", err)
	}
}

func (k *Kernel) gonchoUserPeerID() string {
	if k.cfg.ChatKey != "" {
		return k.cfg.ChatKey
	}
	return "user"
}

func (k *Kernel) gonchoTurnMetadata(role, turnKey string) map[string]string {
	metadata := map[string]string{
		"role":   role,
		"source": "kernel",
	}
	if turnKey != "" {
		metadata["turn_key"] = turnKey
	}
	if k.cfg.ChatKey != "" {
		metadata["chat_key"] = k.cfg.ChatKey
	}
	if k.activeModel != "" {
		metadata["model"] = k.activeModel
	}
	return metadata
}
