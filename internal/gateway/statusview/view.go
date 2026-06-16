package statusview

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
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
			b.WriteString("- pending")
			if platform := statusLineValue(record.Platform); platform != "" {
				b.WriteByte(' ')
				b.WriteString(platform)
			}
			if userID := statusLineValue(record.UserID); userID != "" {
				b.WriteString(" user=")
				b.WriteString(userID)
			}
			if code := statusLineValue(record.Code); code != "" {
				b.WriteString(" code=")
				b.WriteString(code)
			}
			b.WriteString(fmt.Sprintf(" age=%ds\n", nonNegativeInt64(record.AgeSeconds)))
		}
		for _, record := range approved {
			b.WriteString("- approved")
			if platform := statusLineValue(record.Platform); platform != "" {
				b.WriteByte(' ')
				b.WriteString(platform)
			}
			if userID := statusLineValue(record.UserID); userID != "" {
				b.WriteString(" user=")
				b.WriteString(userID)
			}
			if userName := statusLineValue(record.UserName); userName != "" {
				b.WriteString(" name=")
				b.WriteString(userName)
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
			if platform := statusLineValue(evidence.Platform); platform != "" {
				b.WriteByte(' ')
				b.WriteString(platform)
			}
			if reason := statusLineValue(evidence.Reason); reason != "" {
				b.WriteByte(' ')
				b.WriteString(reason)
			}
			if message := statusLineValue(evidence.Message); message != "" {
				b.WriteString(": ")
				b.WriteString(message)
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

	state := statusLineValue(runtime.GatewayState)
	if state == "" {
		state = "unknown"
	}
	parts := []string{}
	if runtime.PID > 0 {
		parts = append(parts, fmt.Sprintf("pid=%d", runtime.PID))
	}
	parts = append(parts, fmt.Sprintf("active_agents=%d", nonNegativeInt(runtime.ActiveAgents)))
	if exitReason := statusLineValue(runtime.ExitReason); exitReason != "" {
		parts = append(parts, fmt.Sprintf("exit_reason=%q", exitReason))
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
	if status := statusLineValue(runtime.ConfigReload.Status); status != "" {
		parts = append(parts, fmt.Sprintf("config_reload=%s", status))
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
	status := statusLineValue(evidence.Status)
	if status == "" {
		return ""
	}
	parts := []string{
		fmt.Sprintf("memory_pressure: %s", status),
		fmt.Sprintf("rss=%dMB", nonNegativeInt(evidence.RSSMB)),
		fmt.Sprintf("warn=%dMB", nonNegativeInt(evidence.WarnRSSMB)),
		fmt.Sprintf("critical=%dMB", nonNegativeInt(evidence.CriticalRSSMB)),
	}
	if uptime := nonNegativeInt64(evidence.UptimeSeconds); uptime > 0 {
		parts = append(parts, fmt.Sprintf("uptime=%ds", uptime))
	}
	if goroutines := nonNegativeInt(evidence.GoRoutines); goroutines > 0 {
		parts = append(parts, fmt.Sprintf("goroutines=%d", goroutines))
	}
	if evidence.GCCollections > 0 {
		parts = append(parts, fmt.Sprintf("gc=%d", evidence.GCCollections))
	}
	if evidence.Action != "" && evidence.Action != "none" {
		parts = append(parts, "action="+statusLineValue(evidence.Action))
	}
	if evidence.TargetPID > 0 {
		parts = append(parts, fmt.Sprintf("target_pid=%d", evidence.TargetPID))
	}
	if evidence.TargetStartTime > 0 {
		parts = append(parts, fmt.Sprintf("target_start_time=%d", evidence.TargetStartTime))
	}
	if len(evidence.Evidence) > 0 {
		cleaned := make([]string, 0, len(evidence.Evidence))
		for _, item := range evidence.Evidence {
			if item = statusLineValue(item); item != "" {
				cleaned = append(cleaned, item)
			}
		}
		if len(cleaned) > 0 {
			parts = append(parts, "evidence="+strings.Join(cleaned, ","))
		}
	}
	if evidence.Message != "" {
		parts = append(parts, "message="+strconv.Quote(statusLineValue(evidence.Message)))
	}
	return strings.Join(parts, " ")
}

func renderChannelLine(channel StatusChannel, runtime RuntimeStatus, pairing PairingPlatformStatus) string {
	lifecycle := "unknown"
	platform, hasPlatform := platformRuntimeStatus(runtime.Platforms, channel.Name)
	if hasPlatform {
		if state := statusLineValue(platform.State); state != "" {
			lifecycle = state
		}
	}

	pairingState := "unpaired"
	pendingCount := 0
	approvedCount := 0
	if pairing.Platform != "" {
		if state := statusLineValue(pairing.State); state != "" {
			pairingState = state
		}
		pendingCount = pairing.PendingCount
		approvedCount = pairing.ApprovedCount
	}

	target := statusLineValue(channel.Detail)
	if target == "" {
		target = "-"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "- %s: lifecycle=%s", statusLineValue(channel.Name), lifecycle)
	if hasPlatform {
		if errorMessage := statusLineValue(platform.ErrorMessage); errorMessage != "" {
			fmt.Fprintf(&b, " error=%q", errorMessage)
		}
	}
	fmt.Fprintf(&b, "; pairing=%s pending=%d approved=%d; target=%s", pairingState, pendingCount, approvedCount, target)
	return b.String()
}

func renderExpiryFinalizedEvidence(evidence RuntimeExpiryFinalizedEvidence) string {
	var b strings.Builder
	b.WriteString("- finalized")
	if sessionID := statusLineValue(evidence.SessionID); sessionID != "" {
		fmt.Fprintf(&b, " session=%s", sessionID)
	}
	if source := statusLineValue(evidence.Source); source != "" {
		fmt.Fprintf(&b, " source=%s", source)
	}
	if chatID := statusLineValue(evidence.ChatID); chatID != "" {
		fmt.Fprintf(&b, " chat=%s", chatID)
	}
	if userID := statusLineValue(evidence.UserID); userID != "" {
		fmt.Fprintf(&b, " user=%s", userID)
	}
	fmt.Fprintf(&b, " expiry_finalized=%t", evidence.ExpiryFinalized)
	if evidence.MigratedMemoryFlushed {
		b.WriteString(" migrated_memory_flushed=true")
	}
	return b.String()
}

func renderExpiryFinalizeEvidence(evidence RuntimeExpiryFinalizeEvidence) string {
	var b strings.Builder
	status := statusLineValue(evidence.Status)
	if status == "" {
		status = "expiry_finalize_pending"
	}
	b.WriteString("- ")
	b.WriteString(status)
	if sessionID := statusLineValue(evidence.SessionID); sessionID != "" {
		fmt.Fprintf(&b, " session=%s", sessionID)
	}
	if source := statusLineValue(evidence.Source); source != "" {
		fmt.Fprintf(&b, " source=%s", source)
	}
	if chatID := statusLineValue(evidence.ChatID); chatID != "" {
		fmt.Fprintf(&b, " chat=%s", chatID)
	}
	if userID := statusLineValue(evidence.UserID); userID != "" {
		fmt.Fprintf(&b, " user=%s", userID)
	}
	fmt.Fprintf(&b, " attempts=%d", nonNegativeInt(evidence.Attempts))
	if errText := statusLineValue(evidence.Error); errText != "" {
		fmt.Fprintf(&b, " error=%q", errText)
	}
	return b.String()
}

func sortedStatusChannels(channels []StatusChannel) []StatusChannel {
	out := make([]StatusChannel, 0, len(channels))
	for _, channel := range channels {
		channel.Name = statusLineValue(channel.Name)
		channel.Detail = statusLineValue(channel.Detail)
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
		platform.Platform = statusLineValue(platform.Platform)
		if platform.Platform == "" {
			continue
		}
		platform.State = statusLineValue(platform.State)
		platform.PendingCount = nonNegativeInt(platform.PendingCount)
		platform.ApprovedCount = nonNegativeInt(platform.ApprovedCount)
		out[platform.Platform] = platform
	}
	return out
}

func platformRuntimeStatus(platforms map[string]PlatformRuntimeStatus, name string) (PlatformRuntimeStatus, bool) {
	if platforms == nil {
		return PlatformRuntimeStatus{}, false
	}
	name = statusLineValue(name)
	if platform, ok := platforms[name]; ok {
		return platform, true
	}
	for key, platform := range platforms {
		if statusLineValue(key) == name {
			return platform, true
		}
	}
	return PlatformRuntimeStatus{}, false
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

func statusLineValue(value string) string {
	value = collapseRedactedStatusAssignments(redaction.RedactSecrets(value))
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return ' '
		}
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func collapseRedactedStatusAssignments(value string) string {
	replacer := strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"authorization=[redacted]", "[redacted]",
		"bearer=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
	)
	fields := strings.Fields(replacer.Replace(value))
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		lower := strings.ToLower(field)
		nextRedacted := i+1 < len(fields) && strings.Contains(strings.ToLower(fields[i+1]), "[redacted]")
		if statusSecretField(lower) && (strings.Contains(lower, "[redacted]") || nextRedacted) {
			out = append(out, "[redacted]")
			if nextRedacted {
				i++
			}
			continue
		}
		out = append(out, field)
	}
	return strings.Join(out, " ")
}

func statusSecretField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "authorization") || strings.Contains(value, "bearer") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
}

func nonNegativeInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func nonNegativeInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
