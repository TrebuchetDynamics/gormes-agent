package kernel

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

type turnRequestAssemblyInput struct {
	Model          string
	SessionID      string
	UserText       string
	UserMessage    hermes.Message
	SessionContext string
	Reasoning      hermes.ReasoningEffortEvidence
}

func (k *Kernel) buildTurnRequest(ctx context.Context, in turnRequestAssemblyInput) hermes.ChatRequest {
	msgs := []hermes.Message{in.UserMessage}
	systemMsgs := make([]hermes.Message, 0, 8)

	if gonchoCtx := k.gonchoContext(ctx); gonchoCtx != "" {
		systemMsgs = append(systemMsgs, hermes.Message{Role: "system", Content: gonchoCtx})
	}
	if in.SessionContext != "" {
		systemMsgs = append(systemMsgs, hermes.Message{Role: "system", Content: in.SessionContext})
	}
	for _, guidance := range k.liveTurnGuidanceBlocks(in.Model) {
		systemMsgs = append(systemMsgs, hermes.Message{Role: "system", Content: guidance})
	}
	if k.cfg.Recall != nil {
		deadline := k.cfg.RecallDeadline
		if deadline <= 0 {
			deadline = 100 * time.Millisecond
		}
		recallCtx, recallCancel := context.WithTimeout(ctx, deadline)
		ctxStr := k.cfg.Recall.GetContext(recallCtx, RecallParams{
			UserMessage: in.UserText,
			ChatKey:     k.cfg.ChatKey,
			SessionID:   k.sessionID,
		})
		recallCancel()
		if ctxStr != "" {
			systemMsgs = append(systemMsgs, hermes.Message{Role: "system", Content: hermes.MemoryGuidance + "\n\n" + ctxStr})
		}
	}
	if k.cfg.Skills != nil {
		block, skillNames, err := k.cfg.Skills.BuildSkillBlock(ctx, in.UserText)
		if err != nil {
			k.log.Warn("kernel: skill runtime failed; continuing without skills", "err", err)
		} else if block != "" {
			systemMsgs = append(systemMsgs, hermes.Message{Role: "system", Content: hermes.SkillsGuidance})
			systemMsgs = append(systemMsgs, hermes.Message{Role: "system", Content: block})
			if len(skillNames) > 0 && k.cfg.SkillUsage != nil {
				if err := k.cfg.SkillUsage.RecordSkillUsage(ctx, skillNames); err != nil {
					k.log.Warn("kernel: record skill usage failed", "err", err)
				}
			}
		}
	}
	if len(k.cfg.PrefillMessages) > 0 {
		msgs = append(cloneKernelMessages(k.cfg.PrefillMessages), msgs...)
	}
	if len(systemMsgs) > 0 {
		msgs = append(systemMsgs, msgs...)
	}

	request := hermes.ChatRequest{
		Model:     in.Model,
		SessionID: in.SessionID,
		Stream:    true,
		Messages:  msgs,
	}
	if in.Reasoning.Forwarded {
		effort := in.Reasoning.Effort
		request.ReasoningEffort = &effort
	}
	if k.cfg.Tools != nil {
		descs := k.cfg.Tools.Descriptors()
		wireDescs := make([]hermes.ToolDescriptor, len(descs))
		for i, d := range descs {
			wireDescs[i] = hermes.ToolDescriptor{Name: d.Name, Description: d.Description, Schema: d.Schema}
		}
		request.Tools = wireDescs
	}
	if k.cfg.ContextEngine != nil {
		request.Tools = append(request.Tools, k.cfg.ContextEngine.ToolDescriptors()...)
	}
	return request
}
