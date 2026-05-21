package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func (a *gonchoAdapter) Observe(ctx context.Context, obs kernel.GonchoObservation) error {
	if a == nil || a.svc == nil {
		return nil
	}
	success := obs.Success
	_, err := a.svc.Observe(ctx, goncho.ObservationParams{
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

func mapGormesObservationKind(kind kernel.GonchoObservationKind) goncho.ObservationKind {
	switch kind {
	case kernel.GonchoObservationSessionStart:
		return goncho.ObservationKindSessionStart
	case kernel.GonchoObservationUserPrompt:
		return goncho.ObservationKindUserPrompt
	case kernel.GonchoObservationToolCall:
		return goncho.ObservationKindToolCall
	case kernel.GonchoObservationToolResult:
		return goncho.ObservationKindToolResult
	case kernel.GonchoObservationToolError:
		return goncho.ObservationKindToolError
	case kernel.GonchoObservationAssistantResponse:
		return goncho.ObservationKindAssistantResponse
	case kernel.GonchoObservationCompact:
		return goncho.ObservationKindCompact
	case kernel.GonchoObservationSessionEnd:
		return goncho.ObservationKindSessionEnd
	default:
		return goncho.ObservationKindCustom
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
