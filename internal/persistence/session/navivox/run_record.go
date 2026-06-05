package navivox

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	RunStatusInProgress = "in_progress"
	RunStatusCompleted  = "completed"
	RunStatusFailed     = "failed"
	RunStatusStopped    = "stopped"

	EvidenceAvailable   = "available"
	EvidenceUnavailable = "unavailable"
	EvidenceUnknown     = "unknown"
)

// RunRecord is the redacted backend read model for one Navivox turn.
// It intentionally stores transcript and evidence metadata, never raw audio
// bytes, secrets, full logs, or provider credentials.
type RunRecord struct {
	RunID         string            `json:"run_id"`
	SessionID     string            `json:"session_id"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	Transcript    []TranscriptEntry `json:"transcript"`
	Voice         *VoiceEvidence    `json:"voice,omitempty"`
	ToolEvents    []ToolEvent       `json:"tool_events,omitempty"`
	ProviderUsage ProviderUsage     `json:"provider_usage"`
	ProviderCost  ProviderCost      `json:"provider_cost"`
	ArtifactRefs  []ArtifactRef     `json:"artifact_refs,omitempty"`
}

type TranscriptEntry struct {
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type VoiceEvidence struct {
	DeviceTranscript string      `json:"device_transcript,omitempty"`
	DeviceSTT        STTEvidence `json:"device_stt"`
	ServerSTT        STTEvidence `json:"server_stt"`
	Audio            AudioMeta   `json:"audio"`
	TTS              TTSEvidence `json:"tts"`
}

type STTEvidence struct {
	Provider string `json:"provider,omitempty"`
	Status   string `json:"status"`
}

type AudioMeta struct {
	DurationMS     int    `json:"duration_ms,omitempty"`
	Codec          string `json:"codec,omitempty"`
	RawAudioStored bool   `json:"raw_audio_stored"`
	Retention      string `json:"retention"`
}

type TTSEvidence struct {
	Provider string `json:"provider,omitempty"`
	VoiceID  string `json:"voice_id,omitempty"`
	Status   string `json:"status"`
}

type ToolEvent struct {
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Summary    string         `json:"summary,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

type ProviderUsage struct {
	Status       string `json:"status"`
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
	TotalTokens  int    `json:"total_tokens,omitempty"`
}

type ProviderCost struct {
	Status string `json:"status"`
	USD    string `json:"usd,omitempty"`
}

type ArtifactRef struct {
	ID      string `json:"id,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
	Ref     string `json:"ref,omitempty"`
}

func NewRunRecord(runID, sessionID, userText string, metadata map[string]any, now time.Time) RunRecord {
	now = normalizeTime(now)
	runID = safeEvidenceString(runID, 128)
	sessionID = safeEvidenceString(sessionID, 128)
	if runID == "" {
		runID = sessionID
	}
	rec := RunRecord{
		RunID:         runID,
		SessionID:     sessionID,
		Status:        RunStatusInProgress,
		CreatedAt:     now,
		UpdatedAt:     now,
		ProviderUsage: ProviderUsage{Status: EvidenceUnknown},
		ProviderCost:  ProviderCost{Status: EvidenceUnknown},
	}
	if text := safeEvidenceString(userText, 4096); text != "" {
		rec.Transcript = append(rec.Transcript, TranscriptEntry{Role: "user", Text: text, Timestamp: now})
	}
	if evidence := voiceEvidence(userText, metadata); evidence != nil {
		rec.Voice = evidence
	}
	return rec
}

func (r *RunRecord) AppendAssistant(text string, now time.Time) {
	if r == nil {
		return
	}
	now = normalizeTime(now)
	if text := safeEvidenceString(text, 4096); text != "" {
		r.Transcript = append(r.Transcript, TranscriptEntry{Role: "assistant", Text: text, Timestamp: now})
	}
	r.UpdatedAt = now
}

func (r *RunRecord) AppendToolEvent(toolCallID, name, status, summary string, metadata map[string]any, now time.Time) {
	if r == nil {
		return
	}
	now = normalizeTime(now)
	name = safeEvidenceString(name, 128)
	if name == "" {
		name = "tool_progress"
	}
	status = safeEvidenceString(status, 64)
	if status == "" {
		status = "started"
	}
	r.ToolEvents = append(r.ToolEvents, ToolEvent{
		ToolCallID: safeEvidenceString(toolCallID, 128),
		Name:       name,
		Status:     status,
		Summary:    safeEvidenceString(summary, 512),
		Metadata:   sanitizeEvidenceMap(metadata),
		Timestamp:  now,
	})
	r.UpdatedAt = now
}

func (r *RunRecord) Complete(now time.Time) {
	r.finish(RunStatusCompleted, now)
}

func (r *RunRecord) Stop(now time.Time) {
	r.finish(RunStatusStopped, now)
}

func (r *RunRecord) Fail(now time.Time) {
	r.finish(RunStatusFailed, now)
}

func (r *RunRecord) finish(status string, now time.Time) {
	if r == nil {
		return
	}
	now = normalizeTime(now)
	r.Status = status
	r.UpdatedAt = now
	completed := now
	r.CompletedAt = &completed
}

func (r *RunRecord) SetProviderUsage(inputTokens, outputTokens, totalTokens int) {
	if r == nil {
		return
	}
	if totalTokens <= 0 {
		totalTokens = inputTokens + outputTokens
	}
	r.ProviderUsage = ProviderUsage{
		Status:       EvidenceAvailable,
		InputTokens:  maxInt(inputTokens, 0),
		OutputTokens: maxInt(outputTokens, 0),
		TotalTokens:  maxInt(totalTokens, 0),
	}
}

func (r RunRecord) Clone() RunRecord {
	out := r
	out.Transcript = append([]TranscriptEntry(nil), r.Transcript...)
	out.ToolEvents = make([]ToolEvent, len(r.ToolEvents))
	for i, ev := range r.ToolEvents {
		out.ToolEvents[i] = ev
		out.ToolEvents[i].Metadata = cloneEvidenceMap(ev.Metadata)
	}
	out.ArtifactRefs = append([]ArtifactRef(nil), r.ArtifactRefs...)
	if r.CompletedAt != nil {
		completed := *r.CompletedAt
		out.CompletedAt = &completed
	}
	if r.Voice != nil {
		voice := *r.Voice
		out.Voice = &voice
	}
	return out
}

func voiceEvidence(userText string, metadata map[string]any) *VoiceEvidence {
	if !metadataMarksVoice(metadata) {
		return nil
	}
	rawStored := boolFromAny(metadata["raw_audio_stored"])
	retention := safeEvidenceString(stringFromAny(metadata["raw_audio_retention"]), 128)
	if retention == "" {
		if rawStored {
			retention = "stored"
		} else {
			retention = "not_stored"
		}
	}
	deviceProvider := safeEvidenceString(stringFromAny(metadata["stt_provider"]), 128)
	if deviceProvider == "" {
		deviceProvider = "device"
	}
	deviceStatus := safeEvidenceString(stringFromAny(metadata["stt_status"]), 64)
	if deviceStatus == "" {
		deviceStatus = EvidenceAvailable
	}
	serverProvider := safeEvidenceString(firstString(metadata, "server_stt_provider", "server_stt"), 128)
	serverStatus := safeEvidenceString(firstString(metadata, "server_stt_status"), 64)
	if serverStatus == "" {
		if serverProvider == "" || strings.EqualFold(serverProvider, "none") {
			serverStatus = EvidenceUnavailable
		} else {
			serverStatus = EvidenceAvailable
		}
	}
	ttsProvider := safeEvidenceString(firstString(metadata, "tts_provider"), 128)
	ttsStatus := safeEvidenceString(firstString(metadata, "tts_status"), 64)
	if ttsStatus == "" {
		if ttsProvider == "" {
			ttsStatus = EvidenceUnknown
		} else {
			ttsStatus = EvidenceAvailable
		}
	}
	return &VoiceEvidence{
		DeviceTranscript: safeEvidenceString(userText, 4096),
		DeviceSTT:        STTEvidence{Provider: deviceProvider, Status: deviceStatus},
		ServerSTT:        STTEvidence{Provider: serverProvider, Status: serverStatus},
		Audio: AudioMeta{
			DurationMS:     intFromAny(firstAny(metadata, "audio_duration_ms", "duration_ms")),
			Codec:          safeEvidenceString(firstString(metadata, "audio_codec", "codec"), 128),
			RawAudioStored: rawStored,
			Retention:      retention,
		},
		TTS: TTSEvidence{
			Provider: ttsProvider,
			VoiceID:  safeEvidenceString(firstString(metadata, "tts_voice_id", "voice_id"), 128),
			Status:   ttsStatus,
		},
	}
}

func metadataMarksVoice(metadata map[string]any) bool {
	if len(metadata) == 0 {
		return false
	}
	if strings.EqualFold(stringFromAny(metadata["input_kind"]), "voice") {
		return true
	}
	if boolFromAny(metadata["voice"]) {
		return true
	}
	for key := range metadata {
		key = strings.ToLower(strings.TrimSpace(key))
		if strings.Contains(key, "stt") || strings.Contains(key, "tts") || strings.Contains(key, "audio_") {
			return true
		}
	}
	return false
}

func sanitizeEvidenceMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := map[string]any{}
	for key, value := range raw {
		key = safeMapKey(key)
		if key == "" || sensitiveKey(key) {
			continue
		}
		switch typed := value.(type) {
		case string:
			out[key] = safeEvidenceString(typed, 512)
		case bool:
			out[key] = typed
		case int:
			out[key] = typed
		case int64:
			out[key] = typed
		case float64:
			if math.Trunc(typed) == typed {
				out[key] = int64(typed)
			} else {
				out[key] = typed
			}
		default:
			out[key] = safeEvidenceString(fmt.Sprint(typed), 512)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneEvidenceMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	return out
}

func sensitiveKey(key string) bool {
	key = strings.ToLower(key)
	for _, marker := range []string{"secret", "token", "password", "api_key", "apikey", "credential", "raw_audio", "audio_bytes"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func safeMapKey(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.' || r == ':':
			b.WriteRune(r)
		}
		if b.Len() >= 64 {
			break
		}
	}
	return b.String()
}

func safeEvidenceString(raw string, maxRunes int) string {
	raw = strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if raw == "" {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = 240
	}
	runes := []rune(raw)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes-3]) + "..."
	}
	return raw
}

func firstString(metadata map[string]any, keys ...string) string {
	return stringFromAny(firstAny(metadata, keys...))
}

func firstAny(metadata map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			return value
		}
	}
	return nil
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return maxInt(typed, 0)
	case int64:
		if typed < 0 {
			return 0
		}
		if typed > int64(^uint(0)>>1) {
			return int(^uint(0) >> 1)
		}
		return int(typed)
	case float64:
		if typed < 0 {
			return 0
		}
		if typed > float64(^uint(0)>>1) {
			return int(^uint(0) >> 1)
		}
		return int(typed)
	case jsonNumber:
		parsed, _ := strconv.Atoi(typed.String())
		return maxInt(parsed, 0)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return maxInt(parsed, 0)
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed
	default:
		return false
	}
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type jsonNumber interface{ String() string }
