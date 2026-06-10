package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (m *Manager) handleRestartCommand(ctx context.Context, ch Channel, ev InboundEvent) error {
	now := m.now()
	if marker, duplicate, err := m.restartDuplicate(ctx, ev); err != nil {
		m.log.Warn("read restart takeover marker", "err", err)
	} else if duplicate {
		evidence := restartDuplicateEvidence(marker, ev, now)
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{DuplicateRestartEvidence: &evidence})
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "duplicate_restart_suppressed")
		return nil
	}

	restartRequested := true
	activeAgents := m.activeAgentCount()
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		RestartRequested: &restartRequested,
		ActiveAgents:     &activeAgents,
	})

	serviceManagerAvailable := m.restartServiceManagerAvailable()
	selfRestartAvailable := m.cfg.Restart.SelfRestart != nil
	if !serviceManagerAvailable && !m.restartMarkerStoreAvailable() && !selfRestartAvailable {
		evidence := RuntimeServiceManagerUnavailableEvidence{
			Source:   ev.Platform,
			ChatID:   ev.ChatID,
			ThreadID: ev.ThreadID,
			Reason:   "restart marker store and service-manager restart exit path are unavailable",
			At:       now.Format(time.RFC3339Nano),
		}
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{ServiceManagerUnavailableEvidence: &evidence})
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Restart unavailable: no restart manager is configured. Restart the gateway process manually or rerun install.sh with gateway restart enabled.")
		return nil
	}

	marker, err := m.writeRestartTakeoverMarker(ctx, ev, now)
	if err != nil {
		evidence := RuntimeServiceManagerUnavailableEvidence{
			Source:   ev.Platform,
			ChatID:   ev.ChatID,
			ThreadID: ev.ThreadID,
			Reason:   "restart takeover marker write failed: " + err.Error(),
			At:       now.Format(time.RFC3339Nano),
		}
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{ServiceManagerUnavailableEvidence: &evidence})
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Restart marker could not be written; restart was not started. Fix runtime state permissions or restart the gateway process manually.")
		return nil
	}
	if m.restartMarkerStoreAvailable() {
		takeoverEvidence := restartTakeoverEvidence(marker, RestartTakeoverMarkerStatusWritten, now)
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{TakeoverMarkerEvidence: &takeoverEvidence})
	}

	if activeAgents > 0 {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("restart_requested: draining %d active agent(s) before restart.", activeAgents))
	} else if serviceManagerAvailable {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "restart_requested: handing off to service manager.")
	} else {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "restart_requested: handing off to gateway restart path.")
	}

	drainCtx, cancel := context.WithTimeout(context.Background(), m.restartDrainTimeout())
	defer cancel()
	if err := m.ShutdownWithDrainReason(drainCtx, DrainReasonRestartTimeout); err != nil &&
		!errors.Is(err, context.DeadlineExceeded) &&
		!errors.Is(err, context.Canceled) {
		m.log.Warn("gateway restart drain", "err", err)
	}

	if !serviceManagerAvailable && selfRestartAvailable {
		if err := m.cfg.Restart.SelfRestart(); err != nil {
			m.log.Warn("gateway self-restart failed", "err", err)
			return RestartRequestedError{
				Code:    GatewayServiceRestartExitCode,
				Message: fmt.Sprintf("self-restart failed: %v; process will exit and must be restarted manually", err),
			}
		}
		return nil
	}
	return RestartRequestedError{
		Code:    GatewayServiceRestartExitCode,
		Message: "gateway restart requested",
	}
}

func (m *Manager) restartDuplicate(ctx context.Context, ev InboundEvent) (RestartTakeoverMarker, bool, error) {
	store := m.cfg.Restart.MarkerStore
	if store == nil {
		return RestartTakeoverMarker{}, false, nil
	}
	return store.SuppressDuplicate(ctx, ev)
}

func (m *Manager) writeRestartTakeoverMarker(ctx context.Context, ev InboundEvent, now time.Time) (RestartTakeoverMarker, error) {
	store := m.cfg.Restart.MarkerStore
	if store == nil {
		return RestartTakeoverMarker{}, nil
	}
	marker := RestartTakeoverMarker{
		SourcePlatform: strings.ToLower(strings.TrimSpace(ev.Platform)),
		ChatID:         strings.TrimSpace(ev.ChatID),
		ThreadID:       strings.TrimSpace(ev.ThreadID),
		UpdateID:       restartUpdateID(ev),
		MessageID:      strings.TrimSpace(ev.MsgID),
		Generation:     m.runtimeStatusGeneration(ctx),
		RequestedAt:    now.Format(time.RFC3339Nano),
	}
	if err := store.Write(ctx, marker); err != nil {
		return marker, err
	}
	return marker, nil
}

func (m *Manager) restartServiceManagerAvailable() bool {
	if m.cfg.Restart.ServiceManagerAvailable == nil {
		return false
	}
	return m.cfg.Restart.ServiceManagerAvailable()
}

func (m *Manager) restartMarkerStoreAvailable() bool {
	store := m.cfg.Restart.MarkerStore
	return store != nil && strings.TrimSpace(store.path) != ""
}

func (m *Manager) restartDrainTimeout() time.Duration {
	if m.cfg.Restart.DrainTimeout > 0 {
		return m.cfg.Restart.DrainTimeout
	}
	return time.Minute
}

func (m *Manager) runtimeStatusGeneration(ctx context.Context) uint64 {
	reader, ok := m.cfg.RuntimeStatus.(interface {
		ReadRuntimeStatus(context.Context) (RuntimeStatus, error)
	})
	if !ok {
		return 0
	}
	status, err := reader.ReadRuntimeStatus(ctx)
	if err != nil {
		m.log.Debug("read gateway runtime status generation", "err", err)
		return 0
	}
	return status.Generation
}

func restartTakeoverEvidence(marker RestartTakeoverMarker, status RestartTakeoverMarkerStatus, at time.Time) RuntimeRestartTakeoverEvidence {
	return RuntimeRestartTakeoverEvidence{
		Status:     status,
		Source:     marker.SourcePlatform,
		ChatID:     marker.ChatID,
		ThreadID:   marker.ThreadID,
		UpdateID:   marker.UpdateID,
		MessageID:  marker.MessageID,
		Generation: marker.Generation,
		At:         at.UTC().Format(time.RFC3339Nano),
	}
}

func restartDuplicateEvidence(marker RestartTakeoverMarker, ev InboundEvent, at time.Time) RuntimeRestartDuplicateEvidence {
	source := marker.SourcePlatform
	if source == "" {
		source = ev.Platform
	}
	chatID := marker.ChatID
	if chatID == "" {
		chatID = ev.ChatID
	}
	threadID := marker.ThreadID
	if threadID == "" {
		threadID = ev.ThreadID
	}
	updateID := marker.UpdateID
	if updateID == "" {
		updateID = restartUpdateID(ev)
	}
	messageID := marker.MessageID
	if messageID == "" {
		messageID = ev.MsgID
	}
	return RuntimeRestartDuplicateEvidence{
		Status:     RestartDuplicateStatusSuppressed,
		Source:     source,
		ChatID:     chatID,
		ThreadID:   threadID,
		UpdateID:   updateID,
		MessageID:  messageID,
		Generation: marker.Generation,
		At:         at.UTC().Format(time.RFC3339Nano),
	}
}

func (m *Manager) ConsumeRestartTakeoverMarker(ctx context.Context) error {
	store := m.cfg.Restart.MarkerStore
	if store == nil {
		return nil
	}
	marker, ok, expired, err := store.Read(ctx)
	if err != nil {
		return err
	}
	now := m.now()
	if expired {
		evidence := restartTakeoverEvidence(marker, RestartTakeoverMarkerStatusExpired, now)
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{TakeoverMarkerEvidence: &evidence})
		return nil
	}
	if !ok || marker.NotificationSentAt != "" {
		return nil
	}
	marker.SourcePlatform = strings.TrimSpace(marker.SourcePlatform)
	marker.ChatID = strings.TrimSpace(marker.ChatID)
	marker.ThreadID = strings.TrimSpace(marker.ThreadID)
	marker.UpdateID = strings.TrimSpace(marker.UpdateID)
	marker.MessageID = strings.TrimSpace(marker.MessageID)
	ch := m.lookupChannel(marker.SourcePlatform)
	if ch == nil {
		return nil
	}
	evidence := restartTakeoverEvidence(marker, RestartTakeoverMarkerStatusSeen, now)
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{TakeoverMarkerEvidence: &evidence})
	if !m.restartNotificationEnabled(marker.SourcePlatform) {
		m.log.Info("restart notification suppressed", "platform", marker.SourcePlatform, "chat_id", marker.ChatID)
		return store.MarkNotificationSent(ctx, marker, now)
	}
	if _, err := m.sendWithHooks(ctx, ch, marker.ChatID, "Gateway restarted successfully. Your session continues."); err != nil {
		return err
	}
	return store.MarkNotificationSent(ctx, marker, now)
}

func (m *Manager) restartNotificationEnabled(platform string) bool {
	key := normalizedPlatformName(platform)
	if key == "" || len(m.cfg.RestartNotifications) == 0 {
		return true
	}
	if enabled, ok := m.cfg.RestartNotifications[key]; ok {
		return enabled
	}
	base := platformBaseName(key)
	if base != key {
		if enabled, ok := m.cfg.RestartNotifications[base]; ok {
			return enabled
		}
	}
	return true
}
