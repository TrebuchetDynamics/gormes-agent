package statusview

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// StatusChannel is a configured channel row for the read-only gateway status
// view. Detail should be non-secret operator context such as an allowlist ID.
type StatusChannel struct {
	Name   string
	Detail string
}

// StatusSummary is the pure input model for RenderStatusSummary.
type StatusSummary struct {
	Channels []StatusChannel
	Pairing  PairingStatus
	Runtime  RuntimeStatus
}

type RuntimeStatus struct {
	Kind                      string
	PID                       int
	GatewayState              string
	ExitReason                string
	RestartRequested          bool
	ActiveAgents              int
	Platforms                 map[string]PlatformRuntimeStatus
	Proxy                     ProxyRuntimeStatus
	TokenLocks                []any
	MemoryPressure            RuntimeMemoryPressureEvidence
	ExpiryFinalized           []RuntimeExpiryFinalizedEvidence
	ExpiryFinalize            []RuntimeExpiryFinalizeEvidence
	TakeoverMarkers           []RuntimeRestartTakeoverEvidence
	DuplicateRestarts         []RuntimeRestartDuplicateEvidence
	ServiceManagerUnavailable []RuntimeServiceManagerUnavailableEvidence
	ConfigReload              RuntimeConfigReloadEvidence
	UpdatedAt                 string
}

type PlatformRuntimeStatus struct {
	State        string
	ErrorMessage string
}

type ProxyRuntimeStatus struct {
	State        string
	URL          string
	ErrorMessage string
	UpdatedAt    string
}

type RuntimeConfigReloadEvidence struct {
	Status     string
	Generation uint64
	Error      string
	AppliedAt  string
	FailedAt   string
	Redacted   bool
}

type RuntimeMemoryPressureEvidence struct {
	Status          string
	RSSMB           int
	WarnRSSMB       int
	CriticalRSSMB   int
	UptimeSeconds   int64
	GoRoutines      int
	GCCollections   uint32
	Action          string
	TargetPID       int
	TargetStartTime int64
	Evidence        []string
	Message         string
	CheckedAt       string
	Redacted        bool
}

type RuntimeExpiryFinalizedEvidence struct {
	SessionID             string
	Source                string
	ChatID                string
	UserID                string
	ExpiryFinalized       bool
	MigratedMemoryFlushed bool
}

type RuntimeExpiryFinalizeEvidence struct {
	SessionID string
	Source    string
	ChatID    string
	UserID    string
	Status    string
	Attempts  int
	Error     string
}

type RuntimeRestartTakeoverEvidence struct{}

type RuntimeRestartDuplicateEvidence struct{}

type RuntimeServiceManagerUnavailableEvidence struct{}

type PairingStatus struct {
	Platforms []PairingPlatformStatus
	Pending   []PairingPendingRecord
	Approved  []PairingApprovedRecord
	Degraded  []PairingDegradedEvidence
}

type PairingPlatformStatus struct {
	Platform      string
	State         string
	PendingCount  int
	ApprovedCount int
}

type PairingPendingRecord struct {
	Platform   string
	Code       string
	UserID     string
	AgeSeconds int64
}

type PairingApprovedRecord struct {
	Platform string
	UserID   string
	UserName string
}

type PairingDegradedEvidence struct {
	Reason   string
	Message  string
	Platform string
	UserID   string
	Code     string
}

// RenderStatusSummary renders the operator-facing gateway status text without
// touching transports, clients, stores, or process state.
func RenderStatusSummary(summary StatusSummary) string {
	var b strings.Builder
	b.WriteString("Gateway status\n")
	b.WriteString(renderRuntimeLine(summary.Runtime))
	b.WriteByte('\n')
	if memoryLine := formatMemoryPressureEvidence(summary.Runtime.MemoryPressure); memoryLine != "" {
		b.WriteString(memoryLine)
		b.WriteByte('\n')
	}

	channels := sortedStatusChannels(summary.Channels)
	if len(channels) == 0 {
		b.WriteString("channels: none configured\n")
	} else {
		b.WriteString("channels:\n")
		pairingByPlatform := pairingPlatformMap(summary.Pairing.Platforms)
		for _, channel := range channels {
			b.WriteString(renderChannelLine(channel, summary.Runtime, pairingByPlatform[channel.Name]))
			b.WriteByte('\n')
		}
	}

	pending := sortedPendingPairingRecords(summary.Pairing.Pending)
	approved := sortedApprovedPairingRecords(summary.Pairing.Approved)
	if len(pending) > 0 || len(approved) > 0 {
		b.WriteString("pairing:\n")
		for _, record := range pending {
			b.WriteString(fmt.Sprintf("- pending %s user=%s code=%s age=%ds\n", record.Platform, record.UserID, record.Code, record.AgeSeconds))
		}
		for _, record := range approved {
			b.WriteString(fmt.Sprintf("- approved %s user=%s", record.Platform, record.UserID))
			if record.UserName != "" {
				b.WriteString(" name=")
				b.WriteString(record.UserName)
			}
			b.WriteByte('\n')
		}
	}

	expiryFinalized := sortedExpiryFinalizedEvidence(summary.Runtime.ExpiryFinalized)
	expiryFinalize := sortedExpiryFinalizeEvidence(summary.Runtime.ExpiryFinalize)
	if len(expiryFinalized) > 0 || len(expiryFinalize) > 0 {
		b.WriteString("session_expiry:\n")
		for _, evidence := range expiryFinalized {
			b.WriteString(renderExpiryFinalizedEvidence(evidence))
			b.WriteByte('\n')
		}
		for _, evidence := range expiryFinalize {
			b.WriteString(renderExpiryFinalizeEvidence(evidence))
			b.WriteByte('\n')
		}
	}

	degraded := sortedPairingDegradedEvidence(summary.Pairing.Degraded)
	if len(degraded) > 0 {
		b.WriteString("degraded:\n")
		for _, evidence := range degraded {
			b.WriteString("- pairing")
			if evidence.Platform != "" {
				b.WriteByte(' ')
				b.WriteString(evidence.Platform)
			}
			if evidence.Reason != "" {
				b.WriteByte(' ')
				b.WriteString(evidence.Reason)
			}
			if evidence.Message != "" {
				b.WriteString(": ")
				b.WriteString(evidence.Message)
			}
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func renderRuntimeLine(runtime RuntimeStatus) string {
	if runtimeStatusMissing(runtime) {
		return "runtime: missing"
	}

	state := runtime.GatewayState
	if state == "" {
		state = "unknown"
	}
	parts := []string{}
	if runtime.PID > 0 {
		parts = append(parts, fmt.Sprintf("pid=%d", runtime.PID))
	}
	parts = append(parts, fmt.Sprintf("active_agents=%d", runtime.ActiveAgents))
	if runtime.ExitReason != "" {
		parts = append(parts, fmt.Sprintf("exit_reason=%q", runtime.ExitReason))
	}
	if runtime.RestartRequested {
		parts = append(parts, "restart_requested=true")
	}
	if len(runtime.TakeoverMarkers) > 0 {
		parts = append(parts, fmt.Sprintf("takeover_marker_seen=%d", len(runtime.TakeoverMarkers)))
	}
	if len(runtime.DuplicateRestarts) > 0 {
		parts = append(parts, fmt.Sprintf("duplicate_restart_suppressed=%d", len(runtime.DuplicateRestarts)))
	}
	if len(runtime.ServiceManagerUnavailable) > 0 {
		parts = append(parts, fmt.Sprintf("service_manager_unavailable=%d", len(runtime.ServiceManagerUnavailable)))
	}
	if runtime.ConfigReload.Status != "" {
		parts = append(parts, fmt.Sprintf("config_reload=%s", runtime.ConfigReload.Status))
	}
	return fmt.Sprintf("runtime: %s (%s)", state, strings.Join(parts, " "))
}

func runtimeStatusMissing(runtime RuntimeStatus) bool {
	return runtime.Kind == "" &&
		runtime.PID == 0 &&
		runtime.GatewayState == "" &&
		runtime.ExitReason == "" &&
		!runtime.RestartRequested &&
		runtime.ActiveAgents == 0 &&
		len(runtime.Platforms) == 0 &&
		len(runtime.TokenLocks) == 0 &&
		len(runtime.ExpiryFinalized) == 0 &&
		len(runtime.ExpiryFinalize) == 0 &&
		len(runtime.TakeoverMarkers) == 0 &&
		len(runtime.DuplicateRestarts) == 0 &&
		len(runtime.ServiceManagerUnavailable) == 0 &&
		memoryPressureEvidenceEmpty(runtime.MemoryPressure) &&
		runtime.ConfigReload == (RuntimeConfigReloadEvidence{}) &&
		runtime.Proxy == (ProxyRuntimeStatus{}) &&
		runtime.UpdatedAt == ""
}

func memoryPressureEvidenceEmpty(evidence RuntimeMemoryPressureEvidence) bool {
	return evidence.Status == "" &&
		evidence.RSSMB == 0 &&
		evidence.WarnRSSMB == 0 &&
		evidence.CriticalRSSMB == 0 &&
		evidence.UptimeSeconds == 0 &&
		evidence.GoRoutines == 0 &&
		evidence.GCCollections == 0 &&
		evidence.Action == "" &&
		evidence.TargetPID == 0 &&
		evidence.TargetStartTime == 0 &&
		len(evidence.Evidence) == 0 &&
		evidence.Message == "" &&
		evidence.CheckedAt == "" &&
		!evidence.Redacted
}

func formatMemoryPressureEvidence(evidence RuntimeMemoryPressureEvidence) string {
	if evidence.Status == "" {
		return ""
	}
	parts := []string{
		fmt.Sprintf("memory_pressure: %s", evidence.Status),
		fmt.Sprintf("rss=%dMB", evidence.RSSMB),
		fmt.Sprintf("warn=%dMB", evidence.WarnRSSMB),
		fmt.Sprintf("critical=%dMB", evidence.CriticalRSSMB),
	}
	if evidence.UptimeSeconds > 0 {
		parts = append(parts, fmt.Sprintf("uptime=%ds", evidence.UptimeSeconds))
	}
	if evidence.GoRoutines > 0 {
		parts = append(parts, fmt.Sprintf("goroutines=%d", evidence.GoRoutines))
	}
	if evidence.GCCollections > 0 {
		parts = append(parts, fmt.Sprintf("gc=%d", evidence.GCCollections))
	}
	if evidence.Action != "" && evidence.Action != "none" {
		parts = append(parts, "action="+evidence.Action)
	}
	if evidence.TargetPID > 0 {
		parts = append(parts, fmt.Sprintf("target_pid=%d", evidence.TargetPID))
	}
	if evidence.TargetStartTime > 0 {
		parts = append(parts, fmt.Sprintf("target_start_time=%d", evidence.TargetStartTime))
	}
	if len(evidence.Evidence) > 0 {
		parts = append(parts, "evidence="+strings.Join(evidence.Evidence, ","))
	}
	if evidence.Message != "" {
		parts = append(parts, "message="+strconv.Quote(evidence.Message))
	}
	return strings.Join(parts, " ")
}

func renderChannelLine(channel StatusChannel, runtime RuntimeStatus, pairing PairingPlatformStatus) string {
	lifecycle := "unknown"
	if runtime.Platforms != nil {
		if platform, ok := runtime.Platforms[channel.Name]; ok && platform.State != "" {
			lifecycle = platform.State
		}
	}

	pairingState := "unpaired"
	pendingCount := 0
	approvedCount := 0
	if pairing.Platform != "" {
		pairingState = pairing.State
		pendingCount = pairing.PendingCount
		approvedCount = pairing.ApprovedCount
	}

	target := channel.Detail
	if target == "" {
		target = "-"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "- %s: lifecycle=%s", channel.Name, lifecycle)
	if runtime.Platforms != nil {
		if platform, ok := runtime.Platforms[channel.Name]; ok && platform.ErrorMessage != "" {
			fmt.Fprintf(&b, " error=%q", platform.ErrorMessage)
		}
	}
	fmt.Fprintf(&b, "; pairing=%s pending=%d approved=%d; target=%s", pairingState, pendingCount, approvedCount, target)
	return b.String()
}

func renderExpiryFinalizedEvidence(evidence RuntimeExpiryFinalizedEvidence) string {
	var b strings.Builder
	b.WriteString("- finalized")
	if evidence.SessionID != "" {
		fmt.Fprintf(&b, " session=%s", evidence.SessionID)
	}
	if evidence.Source != "" {
		fmt.Fprintf(&b, " source=%s", evidence.Source)
	}
	if evidence.ChatID != "" {
		fmt.Fprintf(&b, " chat=%s", evidence.ChatID)
	}
	if evidence.UserID != "" {
		fmt.Fprintf(&b, " user=%s", evidence.UserID)
	}
	fmt.Fprintf(&b, " expiry_finalized=%t", evidence.ExpiryFinalized)
	if evidence.MigratedMemoryFlushed {
		b.WriteString(" migrated_memory_flushed=true")
	}
	return b.String()
}

func renderExpiryFinalizeEvidence(evidence RuntimeExpiryFinalizeEvidence) string {
	var b strings.Builder
	status := evidence.Status
	if status == "" {
		status = "expiry_finalize_pending"
	}
	b.WriteString("- ")
	b.WriteString(status)
	if evidence.SessionID != "" {
		fmt.Fprintf(&b, " session=%s", evidence.SessionID)
	}
	if evidence.Source != "" {
		fmt.Fprintf(&b, " source=%s", evidence.Source)
	}
	if evidence.ChatID != "" {
		fmt.Fprintf(&b, " chat=%s", evidence.ChatID)
	}
	if evidence.UserID != "" {
		fmt.Fprintf(&b, " user=%s", evidence.UserID)
	}
	fmt.Fprintf(&b, " attempts=%d", evidence.Attempts)
	if evidence.Error != "" {
		fmt.Fprintf(&b, " error=%q", evidence.Error)
	}
	return b.String()
}

func sortedStatusChannels(channels []StatusChannel) []StatusChannel {
	out := make([]StatusChannel, 0, len(channels))
	for _, channel := range channels {
		channel.Name = strings.TrimSpace(channel.Name)
		channel.Detail = strings.TrimSpace(channel.Detail)
		if channel.Name == "" {
			continue
		}
		out = append(out, channel)
	}
	slices.SortStableFunc(out, func(a, b StatusChannel) int {
		if byName := cmp.Compare(a.Name, b.Name); byName != 0 {
			return byName
		}
		return cmp.Compare(a.Detail, b.Detail)
	})
	return out
}

func pairingPlatformMap(platforms []PairingPlatformStatus) map[string]PairingPlatformStatus {
	out := make(map[string]PairingPlatformStatus, len(platforms))
	for _, platform := range platforms {
		if platform.Platform == "" {
			continue
		}
		out[platform.Platform] = platform
	}
	return out
}

func sortedPendingPairingRecords(records []PairingPendingRecord) []PairingPendingRecord {
	out := slices.Clone(records)
	slices.SortStableFunc(out, func(left, right PairingPendingRecord) int {
		if byPlatform := cmp.Compare(left.Platform, right.Platform); byPlatform != 0 {
			return byPlatform
		}
		if byUserID := cmp.Compare(left.UserID, right.UserID); byUserID != 0 {
			return byUserID
		}
		if byAgeSeconds := cmp.Compare(left.AgeSeconds, right.AgeSeconds); byAgeSeconds != 0 {
			return byAgeSeconds
		}
		return cmp.Compare(left.Code, right.Code)
	})
	return out
}

func sortedApprovedPairingRecords(records []PairingApprovedRecord) []PairingApprovedRecord {
	out := slices.Clone(records)
	slices.SortStableFunc(out, func(left, right PairingApprovedRecord) int {
		if byPlatform := cmp.Compare(left.Platform, right.Platform); byPlatform != 0 {
			return byPlatform
		}
		if byUserID := cmp.Compare(left.UserID, right.UserID); byUserID != 0 {
			return byUserID
		}
		return cmp.Compare(left.UserName, right.UserName)
	})
	return out
}

func sortedExpiryFinalizedEvidence(records []RuntimeExpiryFinalizedEvidence) []RuntimeExpiryFinalizedEvidence {
	out := slices.Clone(records)
	slices.SortStableFunc(out, func(left, right RuntimeExpiryFinalizedEvidence) int {
		if bySessionID := cmp.Compare(left.SessionID, right.SessionID); bySessionID != 0 {
			return bySessionID
		}
		if bySource := cmp.Compare(left.Source, right.Source); bySource != 0 {
			return bySource
		}
		if byChatID := cmp.Compare(left.ChatID, right.ChatID); byChatID != 0 {
			return byChatID
		}
		return cmp.Compare(left.UserID, right.UserID)
	})
	return out
}

func sortedExpiryFinalizeEvidence(records []RuntimeExpiryFinalizeEvidence) []RuntimeExpiryFinalizeEvidence {
	out := slices.Clone(records)
	slices.SortStableFunc(out, func(left, right RuntimeExpiryFinalizeEvidence) int {
		if bySessionID := cmp.Compare(left.SessionID, right.SessionID); bySessionID != 0 {
			return bySessionID
		}
		if byStatus := cmp.Compare(left.Status, right.Status); byStatus != 0 {
			return byStatus
		}
		if bySource := cmp.Compare(left.Source, right.Source); bySource != 0 {
			return bySource
		}
		if byChatID := cmp.Compare(left.ChatID, right.ChatID); byChatID != 0 {
			return byChatID
		}
		return cmp.Compare(left.UserID, right.UserID)
	})
	return out
}

func sortedPairingDegradedEvidence(records []PairingDegradedEvidence) []PairingDegradedEvidence {
	out := slices.Clone(records)
	slices.SortStableFunc(out, func(left, right PairingDegradedEvidence) int {
		if byPlatform := cmp.Compare(left.Platform, right.Platform); byPlatform != 0 {
			return byPlatform
		}
		if byReason := cmp.Compare(left.Reason, right.Reason); byReason != 0 {
			return byReason
		}
		if byUserID := cmp.Compare(left.UserID, right.UserID); byUserID != 0 {
			return byUserID
		}
		if byCode := cmp.Compare(left.Code, right.Code); byCode != 0 {
			return byCode
		}
		return cmp.Compare(left.Message, right.Message)
	})
	return out
}
