package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/redaction"
)

const (
	OperatorRunReportSchemaVersion = "operator_run_report.v1"

	OperatorRunReportStatusSuccess  = "success"
	OperatorRunReportStatusDegraded = "degraded"
	OperatorRunReportStatusFailed   = "failed"

	OperatorRunReportReasonProviderAuthUnready = "provider_auth_unready"
	OperatorRunReportReasonTimeout             = "cron_timeout"
	OperatorRunReportReasonSuppressed          = "cron_suppressed"
	OperatorRunReportReasonDeliveryFailed      = "delivery_failed"
	OperatorRunReportReasonCronError           = "cron_error"
)

// OperatorRunReportInput is the hermetic evidence packet used to build an
// unattended-job report. It deliberately accepts already-produced cron/runtime
// evidence and does not start a scheduler, provider, gateway, or kernel turn.
type OperatorRunReportInput struct {
	Job             Job
	Run             Run
	Profile         string
	Workspace       string
	HomeDir         string
	RuntimeEvidence map[string]any
	DeliveryPlan    DeliveryPlan
	DeliveryOutcome DeliveryOutcome
	SessionID       string
	TranscriptRefs  []string
	ReleaseEvidence []ReleaseEvidence
}

// OperatorRunReport is the stable JSON artifact operators can inspect after an
// unattended cron/fleet job. Fields are intentionally strings, numbers, and
// small typed slices so the artifact remains easy to diff and render later.
type OperatorRunReport struct {
	SchemaVersion          string                      `json:"schema_version"`
	JobID                  string                      `json:"job_id"`
	JobName                string                      `json:"job_name,omitempty"`
	RunID                  int64                       `json:"run_id"`
	Profile                string                      `json:"profile,omitempty"`
	Workspace              string                      `json:"workspace,omitempty"`
	Provider               string                      `json:"provider,omitempty"`
	Model                  string                      `json:"model,omitempty"`
	Runtime                OperatorRunRuntimeEvidence  `json:"runtime"`
	DeliveryTargets        []OperatorRunDeliveryTarget `json:"delivery_targets,omitempty"`
	DeliveryEvidence       []OperatorRunEvidence       `json:"delivery_evidence,omitempty"`
	StartedAtUnix          int64                       `json:"started_at_unix"`
	FinishedAtUnix         int64                       `json:"finished_at_unix,omitempty"`
	Status                 string                      `json:"status"`
	DegradedReason         string                      `json:"degraded_reason,omitempty"`
	SessionID              string                      `json:"session_id,omitempty"`
	TranscriptRefs         []string                    `json:"transcript_refs,omitempty"`
	OutputSummary          string                      `json:"output_summary,omitempty"`
	ErrorSummary           string                      `json:"error_summary,omitempty"`
	ReleaseEvidence        []OperatorRunEvidence       `json:"release_evidence,omitempty"`
	RecommendedNextCommand string                      `json:"recommended_next_command"`
	Redacted               bool                        `json:"redacted,omitempty"`
}

type OperatorRunRuntimeEvidence struct {
	Provider        string   `json:"provider,omitempty"`
	Model           string   `json:"model,omitempty"`
	EndpointSource  string   `json:"endpoint_source,omitempty"`
	DegradedReasons []string `json:"degraded_reasons,omitempty"`
}

type OperatorRunDeliveryTarget struct {
	Target   string `json:"target"`
	Platform string `json:"platform,omitempty"`
	ChatID   string `json:"chat_id,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`
	Local    bool   `json:"local,omitempty"`
	Origin   bool   `json:"origin,omitempty"`
	Explicit bool   `json:"explicit,omitempty"`
}

type OperatorRunEvidence struct {
	Code   string            `json:"code"`
	Target string            `json:"target,omitempty"`
	Label  string            `json:"label,omitempty"`
	Detail string            `json:"detail,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
}

// BuildOperatorRunReport maps existing cron/runtime/delivery/session evidence
// into a durable operator artifact. It is pure apart from redaction decisions.
func BuildOperatorRunReport(in OperatorRunReportInput) OperatorRunReport {
	sanitizer := &operatorReportSanitizer{homeDir: strings.TrimSpace(in.HomeDir)}
	runtimeEvidence := normalizeOperatorRuntimeEvidence(in.RuntimeEvidence, in.Job, sanitizer)
	deliveryPlan := in.DeliveryPlan
	if len(deliveryPlan.Targets) == 0 && len(deliveryPlan.Evidence) == 0 {
		deliveryPlan = PlanCronDelivery(DeliveryPlanOptions{
			Deliver: in.Job.Deliver,
			Origin:  in.Job.Origin,
		})
	}

	status, degradedReason := classifyOperatorRunReport(in.Run, runtimeEvidence, deliveryPlan, in.DeliveryOutcome)
	report := OperatorRunReport{
		SchemaVersion:          OperatorRunReportSchemaVersion,
		JobID:                  sanitizer.text(in.Job.ID),
		JobName:                sanitizer.text(in.Job.Name),
		RunID:                  in.Run.ID,
		Profile:                sanitizer.text(in.Profile),
		Workspace:              sanitizer.text(firstNonEmpty(in.Workspace, in.Job.Workdir)),
		Provider:               runtimeEvidence.Provider,
		Model:                  runtimeEvidence.Model,
		Runtime:                runtimeEvidence,
		DeliveryTargets:        normalizeOperatorDeliveryTargets(deliveryPlan.Targets, sanitizer),
		DeliveryEvidence:       normalizeOperatorDeliveryEvidence(appendDeliveryEvidence(deliveryPlan.Evidence, in.DeliveryOutcome.Evidence), sanitizer),
		StartedAtUnix:          in.Run.StartedAt,
		FinishedAtUnix:         in.Run.FinishedAt,
		Status:                 status,
		DegradedReason:         degradedReason,
		SessionID:              sanitizer.text(in.SessionID),
		TranscriptRefs:         sanitizer.textSlice(in.TranscriptRefs),
		OutputSummary:          sanitizer.text(in.Run.OutputPreview),
		ErrorSummary:           buildOperatorRunErrorSummary(in.Run, in.DeliveryOutcome, sanitizer),
		ReleaseEvidence:        normalizeOperatorReleaseEvidence(in.ReleaseEvidence, sanitizer),
		RecommendedNextCommand: operatorRunRecommendedCommand(in.Job.ID, degradedReason),
	}
	report.Redacted = sanitizer.redacted
	return report
}

func OperatorRunReportPath(home string, report OperatorRunReport) string {
	jobID := safeOperatorReportPathComponent(report.JobID)
	if jobID == "" {
		jobID = "unknown-job"
	}
	runID := strconv.FormatInt(report.RunID, 10)
	if report.RunID == 0 {
		runID = "unknown-run"
	}
	return filepath.Join(home, "operator-runs", jobID, runID+".json")
}

func WriteOperatorRunReport(path string, report OperatorRunReport) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("cron: operator run report path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cron: create operator run report directory: %w", err)
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("cron: create operator run report: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	writeErr := enc.Encode(report)
	closeErr := f.Close()
	if writeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("cron: encode operator run report: %w", writeErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("cron: close operator run report: %w", closeErr)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("cron: publish operator run report: %w", err)
	}
	return nil
}

func ReadOperatorRunReport(path string) (OperatorRunReport, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return OperatorRunReport{}, fmt.Errorf("cron: read operator run report: %w", err)
	}
	var report OperatorRunReport
	if err := json.Unmarshal(body, &report); err != nil {
		return OperatorRunReport{}, fmt.Errorf("cron: decode operator run report: %w", err)
	}
	return report, nil
}

func normalizeOperatorRuntimeEvidence(evidence map[string]any, job Job, sanitizer *operatorReportSanitizer) OperatorRunRuntimeEvidence {
	out := OperatorRunRuntimeEvidence{
		Provider:       sanitizer.text(firstNonEmpty(operatorEvidenceString(evidence, "provider"), job.Provider)),
		Model:          sanitizer.text(firstNonEmpty(operatorEvidenceString(evidence, "model"), job.Model)),
		EndpointSource: sanitizer.text(operatorEvidenceString(evidence, "endpoint_source")),
	}
	out.DegradedReasons = sanitizer.textSlice(operatorEvidenceStringSlice(evidence, "degraded_reasons"))
	return out
}

func normalizeOperatorDeliveryTargets(targets []DeliveryTarget, sanitizer *operatorReportSanitizer) []OperatorRunDeliveryTarget {
	if len(targets) == 0 {
		return nil
	}
	out := make([]OperatorRunDeliveryTarget, 0, len(targets))
	for _, target := range targets {
		normalized := sanitizer.text(target.Normalized())
		if normalized == "" {
			continue
		}
		out = append(out, OperatorRunDeliveryTarget{
			Target:   normalized,
			Platform: sanitizer.text(target.Platform),
			ChatID:   sanitizer.text(target.ChatID),
			ThreadID: sanitizer.text(target.ThreadID),
			Local:    target.Local,
			Origin:   target.Origin,
			Explicit: target.Explicit,
		})
	}
	return out
}

func normalizeOperatorDeliveryEvidence(evidence []DeliveryEvidence, sanitizer *operatorReportSanitizer) []OperatorRunEvidence {
	if len(evidence) == 0 {
		return nil
	}
	out := make([]OperatorRunEvidence, 0, len(evidence))
	for _, item := range evidence {
		code := sanitizer.text(item.Code)
		if code == "" {
			continue
		}
		out = append(out, OperatorRunEvidence{
			Code:   code,
			Target: sanitizer.text(item.Target),
			Detail: sanitizer.text(item.Detail),
		})
	}
	return out
}

func normalizeOperatorReleaseEvidence(evidence []ReleaseEvidence, sanitizer *operatorReportSanitizer) []OperatorRunEvidence {
	if len(evidence) == 0 {
		return nil
	}
	out := make([]OperatorRunEvidence, 0, len(evidence))
	for _, item := range evidence {
		code := sanitizer.text(string(item.Code))
		if code == "" {
			continue
		}
		out = append(out, OperatorRunEvidence{
			Code:   code,
			Label:  sanitizer.text(item.Label),
			Fields: sanitizeOperatorEvidenceFields(item.Fields, sanitizer),
		})
	}
	return out
}

func sanitizeOperatorEvidenceFields(fields map[string]any, sanitizer *operatorReportSanitizer) map[string]string {
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(fields))
	for _, key := range keys {
		cleanKey := sanitizer.text(key)
		if cleanKey == "" {
			continue
		}
		out[cleanKey] = sanitizer.text(fmt.Sprint(fields[key]))
	}
	return out
}

func classifyOperatorRunReport(run Run, runtimeEvidence OperatorRunRuntimeEvidence, plan DeliveryPlan, outcome DeliveryOutcome) (string, string) {
	if hasProviderAuthDegradedReason(runtimeEvidence.DegradedReasons) {
		return OperatorRunReportStatusFailed, OperatorRunReportReasonProviderAuthUnready
	}
	switch strings.TrimSpace(run.Status) {
	case "timeout":
		return OperatorRunReportStatusFailed, OperatorRunReportReasonTimeout
	case "suppressed":
		return OperatorRunReportStatusDegraded, OperatorRunReportReasonSuppressed
	}
	if operatorDeliveryFailed(run, plan, outcome) {
		return OperatorRunReportStatusDegraded, OperatorRunReportReasonDeliveryFailed
	}
	switch strings.TrimSpace(run.Status) {
	case "success":
		return OperatorRunReportStatusSuccess, ""
	case "error":
		return OperatorRunReportStatusFailed, OperatorRunReportReasonCronError
	default:
		if strings.TrimSpace(run.Status) == "" {
			return OperatorRunReportStatusDegraded, OperatorRunReportReasonCronError
		}
		return OperatorRunReportStatusFailed, OperatorRunReportReasonCronError
	}
}

func hasProviderAuthDegradedReason(reasons []string) bool {
	for _, reason := range reasons {
		switch strings.TrimSpace(reason) {
		case "provider_config_missing", "native_runtime_unavailable":
			return true
		}
	}
	return false
}

func operatorDeliveryFailed(run Run, plan DeliveryPlan, outcome DeliveryOutcome) bool {
	if strings.TrimSpace(run.Status) != "success" {
		return false
	}
	if !run.Delivered {
		return true
	}
	if outcome.Err != nil {
		return true
	}
	if len(plan.Evidence) > 0 || len(outcome.Evidence) > 0 {
		for _, item := range appendDeliveryEvidence(plan.Evidence, outcome.Evidence) {
			switch strings.TrimSpace(item.Code) {
			case DeliveryEvidenceTargetParseFailed, DeliveryEvidenceChannelDirectoryMissing, DeliveryEvidenceStandaloneSenderFailed:
				return true
			}
		}
	}
	return false
}

func buildOperatorRunErrorSummary(run Run, outcome DeliveryOutcome, sanitizer *operatorReportSanitizer) string {
	var parts []string
	if strings.TrimSpace(run.ErrorMsg) != "" {
		parts = append(parts, run.ErrorMsg)
	}
	if outcome.Err != nil {
		parts = append(parts, outcome.Err.Error())
	}
	if len(parts) == 0 {
		return ""
	}
	return sanitizer.text(strings.Join(parts, "; "))
}

func operatorRunRecommendedCommand(jobID, degradedReason string) string {
	switch degradedReason {
	case OperatorRunReportReasonProviderAuthUnready:
		return "gormes doctor --offline"
	case OperatorRunReportReasonDeliveryFailed:
		return "gormes gateway status --json"
	default:
		jobID = strings.TrimSpace(jobID)
		if jobID == "" {
			return "gormes cron list"
		}
		return "gormes cron status " + jobID
	}
}

func appendDeliveryEvidence(a, b []DeliveryEvidence) []DeliveryEvidence {
	if len(a) == 0 {
		return append([]DeliveryEvidence{}, b...)
	}
	out := append([]DeliveryEvidence{}, a...)
	out = append(out, b...)
	return out
}

func operatorEvidenceString(evidence map[string]any, key string) string {
	if len(evidence) == 0 {
		return ""
	}
	value, ok := evidence[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func operatorEvidenceStringSlice(evidence map[string]any, key string) []string {
	if len(evidence) == 0 {
		return nil
	}
	value, ok := evidence[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if strings.TrimSpace(item) != "" {
				out = append(out, strings.TrimSpace(item))
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		text := strings.TrimSpace(fmt.Sprint(typed))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func safeOperatorReportPathComponent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, string(os.PathSeparator), "_")
	value = strings.Trim(value, ". ")
	return value
}

type operatorReportSanitizer struct {
	homeDir  string
	redacted bool
}

func (s *operatorReportSanitizer) text(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	out, count := redaction.RedactSecretsWithCount(value, "[redacted]")
	if count > 0 {
		s.redacted = true
	}
	if s.homeDir != "" {
		for _, marker := range homePathMarkers(s.homeDir) {
			if strings.Contains(out, marker) {
				out = strings.ReplaceAll(out, marker, "[home]")
				s.redacted = true
			}
		}
	}
	return out
}

func (s *operatorReportSanitizer) textSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text := s.text(value); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func homePathMarkers(home string) []string {
	home = strings.TrimSpace(home)
	if home == "" {
		return nil
	}
	clean := filepath.Clean(home)
	markers := []string{clean, filepath.ToSlash(clean)}
	if abs, err := filepath.Abs(clean); err == nil {
		markers = append(markers, abs, filepath.ToSlash(abs))
	}
	out := make([]string, 0, len(markers))
	seen := map[string]struct{}{}
	for _, marker := range markers {
		if marker == "." || marker == "" {
			continue
		}
		if _, ok := seen[marker]; ok {
			continue
		}
		seen[marker] = struct{}{}
		out = append(out, marker)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return len(out[i]) > len(out[j])
	})
	return out
}
