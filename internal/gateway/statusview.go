package gateway

import gatewaystatusview "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/statusview"

// StatusChannel is a configured channel row for the read-only gateway status
// view. Detail should be non-secret operator context such as an allowlist ID.
type StatusChannel = gatewaystatusview.StatusChannel

// StatusSummary is the pure input model for RenderStatusSummary.
type StatusSummary struct {
	Channels []StatusChannel
	Pairing  PairingStatus
	Runtime  RuntimeStatus
}

// RenderStatusSummary renders the operator-facing gateway status text without
// touching transports, clients, stores, or process state.
func RenderStatusSummary(summary StatusSummary) string {
	return gatewaystatusview.RenderStatusSummary(gatewaystatusview.StatusSummary{
		Channels: summary.Channels,
		Pairing:  statusViewPairing(summary.Pairing),
		Runtime:  statusViewRuntime(summary.Runtime),
	})
}

func statusViewRuntime(runtime RuntimeStatus) gatewaystatusview.RuntimeStatus {
	platforms := make(map[string]gatewaystatusview.PlatformRuntimeStatus, len(runtime.Platforms))
	for name, platform := range runtime.Platforms {
		platforms[name] = gatewaystatusview.PlatformRuntimeStatus{
			State:        string(platform.State),
			ErrorMessage: platform.ErrorMessage,
		}
	}
	return gatewaystatusview.RuntimeStatus{
		Kind:                      runtime.Kind,
		PID:                       runtime.PID,
		GatewayState:              string(runtime.GatewayState),
		ExitReason:                runtime.ExitReason,
		RestartRequested:          runtime.RestartRequested,
		ActiveAgents:              runtime.ActiveAgents,
		Platforms:                 platforms,
		Proxy:                     gatewaystatusview.ProxyRuntimeStatus{State: runtime.Proxy.State, URL: runtime.Proxy.URL, ErrorMessage: runtime.Proxy.ErrorMessage, UpdatedAt: runtime.Proxy.UpdatedAt},
		TokenLocks:                anySlice(len(runtime.TokenLocks)),
		MemoryPressure:            statusViewMemoryPressure(runtime.MemoryPressure),
		ExpiryFinalized:           statusViewExpiryFinalized(runtime.ExpiryFinalized),
		ExpiryFinalize:            statusViewExpiryFinalize(runtime.ExpiryFinalize),
		TakeoverMarkers:           make([]gatewaystatusview.RuntimeRestartTakeoverEvidence, len(runtime.TakeoverMarkers)),
		DuplicateRestarts:         make([]gatewaystatusview.RuntimeRestartDuplicateEvidence, len(runtime.DuplicateRestarts)),
		ServiceManagerUnavailable: make([]gatewaystatusview.RuntimeServiceManagerUnavailableEvidence, len(runtime.ServiceManagerUnavailable)),
		ConfigReload: gatewaystatusview.RuntimeConfigReloadEvidence{
			Status:     string(runtime.ConfigReload.Status),
			Generation: runtime.ConfigReload.Generation,
			Error:      runtime.ConfigReload.Error,
			AppliedAt:  runtime.ConfigReload.AppliedAt,
			FailedAt:   runtime.ConfigReload.FailedAt,
			Redacted:   runtime.ConfigReload.Redacted,
		},
		UpdatedAt: runtime.UpdatedAt,
	}
}

func statusViewMemoryPressure(evidence RuntimeMemoryPressureEvidence) gatewaystatusview.RuntimeMemoryPressureEvidence {
	return gatewaystatusview.RuntimeMemoryPressureEvidence{
		Status:          string(evidence.Status),
		RSSMB:           evidence.RSSMB,
		WarnRSSMB:       evidence.WarnRSSMB,
		CriticalRSSMB:   evidence.CriticalRSSMB,
		UptimeSeconds:   evidence.UptimeSeconds,
		GoRoutines:      evidence.GoRoutines,
		GCCollections:   evidence.GCCollections,
		Action:          string(evidence.Action),
		TargetPID:       evidence.TargetPID,
		TargetStartTime: evidence.TargetStartTime,
		Evidence:        append([]string(nil), evidence.Evidence...),
		Message:         evidence.Message,
		CheckedAt:       evidence.CheckedAt,
		Redacted:        evidence.Redacted,
	}
}

func statusViewExpiryFinalized(records []RuntimeExpiryFinalizedEvidence) []gatewaystatusview.RuntimeExpiryFinalizedEvidence {
	out := make([]gatewaystatusview.RuntimeExpiryFinalizedEvidence, 0, len(records))
	for _, record := range records {
		out = append(out, gatewaystatusview.RuntimeExpiryFinalizedEvidence{
			SessionID:             record.SessionID,
			Source:                record.Source,
			ChatID:                record.ChatID,
			UserID:                record.UserID,
			ExpiryFinalized:       record.ExpiryFinalized,
			MigratedMemoryFlushed: record.MigratedMemoryFlushed,
		})
	}
	return out
}

func statusViewExpiryFinalize(records []RuntimeExpiryFinalizeEvidence) []gatewaystatusview.RuntimeExpiryFinalizeEvidence {
	out := make([]gatewaystatusview.RuntimeExpiryFinalizeEvidence, 0, len(records))
	for _, record := range records {
		out = append(out, gatewaystatusview.RuntimeExpiryFinalizeEvidence{
			SessionID: record.SessionID,
			Source:    record.Source,
			ChatID:    record.ChatID,
			UserID:    record.UserID,
			Status:    record.Status,
			Attempts:  record.Attempts,
			Error:     record.Error,
		})
	}
	return out
}

func statusViewPairing(pairing PairingStatus) gatewaystatusview.PairingStatus {
	return gatewaystatusview.PairingStatus{
		Platforms: statusViewPairingPlatforms(pairing.Platforms),
		Pending:   statusViewPairingPending(pairing.Pending),
		Approved:  statusViewPairingApproved(pairing.Approved),
		Degraded:  statusViewPairingDegraded(pairing.Degraded),
	}
}

func statusViewPairingPlatforms(records []PairingPlatformStatus) []gatewaystatusview.PairingPlatformStatus {
	out := make([]gatewaystatusview.PairingPlatformStatus, 0, len(records))
	for _, record := range records {
		out = append(out, gatewaystatusview.PairingPlatformStatus{
			Platform:      record.Platform,
			State:         string(record.State),
			PendingCount:  record.PendingCount,
			ApprovedCount: record.ApprovedCount,
		})
	}
	return out
}

func statusViewPairingPending(records []PairingPendingRecord) []gatewaystatusview.PairingPendingRecord {
	out := make([]gatewaystatusview.PairingPendingRecord, 0, len(records))
	for _, record := range records {
		out = append(out, gatewaystatusview.PairingPendingRecord{
			Platform:   record.Platform,
			Code:       record.Code,
			UserID:     record.UserID,
			AgeSeconds: record.AgeSeconds,
		})
	}
	return out
}

func statusViewPairingApproved(records []PairingApprovedRecord) []gatewaystatusview.PairingApprovedRecord {
	out := make([]gatewaystatusview.PairingApprovedRecord, 0, len(records))
	for _, record := range records {
		out = append(out, gatewaystatusview.PairingApprovedRecord{
			Platform: record.Platform,
			UserID:   record.UserID,
			UserName: record.UserName,
		})
	}
	return out
}

func statusViewPairingDegraded(records []PairingDegradedEvidence) []gatewaystatusview.PairingDegradedEvidence {
	out := make([]gatewaystatusview.PairingDegradedEvidence, 0, len(records))
	for _, record := range records {
		out = append(out, gatewaystatusview.PairingDegradedEvidence{
			Reason:   string(record.Reason),
			Message:  record.Message,
			Platform: record.Platform,
			UserID:   record.UserID,
			Code:     record.Code,
		})
	}
	return out
}

func anySlice(length int) []any {
	if length == 0 {
		return nil
	}
	return make([]any, length)
}
