package session

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	NavivoxRunStatusInProgress = "in_progress"
	NavivoxRunStatusCompleted  = "completed"
	NavivoxRunStatusFailed     = "failed"
	NavivoxRunStatusStopped    = "stopped"

	NavivoxEvidenceAvailable   = "available"
	NavivoxEvidenceUnavailable = "unavailable"
	NavivoxEvidenceUnknown     = "unknown"
)

// NavivoxRunRecord is the redacted backend read model for one Navivox turn.
// It intentionally stores transcript and evidence metadata, never raw audio
// bytes, secrets, full logs, or provider credentials.
type NavivoxRunRecord struct {
	RunID         string                   `json:"run_id"`
	SessionID     string                   `json:"session_id"`
	Status        string                   `json:"status"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	CompletedAt   *time.Time               `json:"completed_at,omitempty"`
	Transcript    []NavivoxTranscriptEntry `json:"transcript"`
	Voice         *NavivoxVoiceEvidence    `json:"voice,omitempty"`
	ToolEvents    []NavivoxToolEvent       `json:"tool_events,omitempty"`
	ProviderUsage NavivoxProviderUsage     `json:"provider_usage"`
	ProviderCost  NavivoxProviderCost      `json:"provider_cost"`
	ArtifactRefs  []NavivoxArtifactRef     `json:"artifact_refs,omitempty"`
}

type NavivoxTranscriptEntry struct {
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type NavivoxVoiceEvidence struct {
	DeviceTranscript string             `json:"device_transcript,omitempty"`
	DeviceSTT        NavivoxSTTEvidence `json:"device_stt"`
	ServerSTT        NavivoxSTTEvidence `json:"server_stt"`
	Audio            NavivoxAudioMeta   `json:"audio"`
	TTS              NavivoxTTSEvidence `json:"tts"`
}

type NavivoxSTTEvidence struct {
	Provider string `json:"provider,omitempty"`
	Status   string `json:"status"`
}

type NavivoxAudioMeta struct {
	DurationMS     int    `json:"duration_ms,omitempty"`
	Codec          string `json:"codec,omitempty"`
	RawAudioStored bool   `json:"raw_audio_stored"`
	Retention      string `json:"retention"`
}

type NavivoxTTSEvidence struct {
	Provider string `json:"provider,omitempty"`
	VoiceID  string `json:"voice_id,omitempty"`
	Status   string `json:"status"`
}

type NavivoxToolEvent struct {
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Summary    string         `json:"summary,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

type NavivoxProviderUsage struct {
	Status       string `json:"status"`
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
	TotalTokens  int    `json:"total_tokens,omitempty"`
}

type NavivoxProviderCost struct {
	Status string `json:"status"`
	USD    string `json:"usd,omitempty"`
}

type NavivoxArtifactRef struct {
	ID      string `json:"id,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
	Ref     string `json:"ref,omitempty"`
}

func NewNavivoxRunRecord(runID, sessionID, userText string, metadata map[string]any, now time.Time) NavivoxRunRecord {
	now = normalizeNavivoxTime(now)
	runID = safeNavivoxEvidenceString(runID, 128)
	sessionID = safeNavivoxEvidenceString(sessionID, 128)
	if runID == "" {
		runID = sessionID
	}
	rec := NavivoxRunRecord{
		RunID:         runID,
		SessionID:     sessionID,
		Status:        NavivoxRunStatusInProgress,
		CreatedAt:     now,
		UpdatedAt:     now,
		ProviderUsage: NavivoxProviderUsage{Status: NavivoxEvidenceUnknown},
		ProviderCost:  NavivoxProviderCost{Status: NavivoxEvidenceUnknown},
	}
	if text := safeNavivoxEvidenceString(userText, 4096); text != "" {
		rec.Transcript = append(rec.Transcript, NavivoxTranscriptEntry{Role: "user", Text: text, Timestamp: now})
	}
	if evidence := navivoxVoiceEvidence(userText, metadata); evidence != nil {
		rec.Voice = evidence
	}
	return rec
}

func (r *NavivoxRunRecord) AppendAssistant(text string, now time.Time) {
	if r == nil {
		return
	}
	now = normalizeNavivoxTime(now)
	if text := safeNavivoxEvidenceString(text, 4096); text != "" {
		r.Transcript = append(r.Transcript, NavivoxTranscriptEntry{Role: "assistant", Text: text, Timestamp: now})
	}
	r.UpdatedAt = now
}

func (r *NavivoxRunRecord) AppendToolEvent(toolCallID, name, status, summary string, metadata map[string]any, now time.Time) {
	if r == nil {
		return
	}
	now = normalizeNavivoxTime(now)
	name = safeNavivoxEvidenceString(name, 128)
	if name == "" {
		name = "tool_progress"
	}
	status = safeNavivoxEvidenceString(status, 64)
	if status == "" {
		status = "started"
	}
	r.ToolEvents = append(r.ToolEvents, NavivoxToolEvent{
		ToolCallID: safeNavivoxEvidenceString(toolCallID, 128),
		Name:       name,
		Status:     status,
		Summary:    safeNavivoxEvidenceString(summary, 512),
		Metadata:   sanitizeNavivoxEvidenceMap(metadata),
		Timestamp:  now,
	})
	r.UpdatedAt = now
}

func (r *NavivoxRunRecord) Complete(now time.Time) {
	r.finish(NavivoxRunStatusCompleted, now)
}

func (r *NavivoxRunRecord) Stop(now time.Time) {
	r.finish(NavivoxRunStatusStopped, now)
}

func (r *NavivoxRunRecord) Fail(now time.Time) {
	r.finish(NavivoxRunStatusFailed, now)
}

func (r *NavivoxRunRecord) finish(status string, now time.Time) {
	if r == nil {
		return
	}
	now = normalizeNavivoxTime(now)
	r.Status = status
	r.UpdatedAt = now
	completed := now
	r.CompletedAt = &completed
}

func (r *NavivoxRunRecord) SetProviderUsage(inputTokens, outputTokens, totalTokens int) {
	if r == nil {
		return
	}
	if totalTokens <= 0 {
		totalTokens = inputTokens + outputTokens
	}
	r.ProviderUsage = NavivoxProviderUsage{
		Status:       NavivoxEvidenceAvailable,
		InputTokens:  maxInt(inputTokens, 0),
		OutputTokens: maxInt(outputTokens, 0),
		TotalTokens:  maxInt(totalTokens, 0),
	}
}

func (r NavivoxRunRecord) Clone() NavivoxRunRecord {
	out := r
	out.Transcript = append([]NavivoxTranscriptEntry(nil), r.Transcript...)
	out.ToolEvents = make([]NavivoxToolEvent, len(r.ToolEvents))
	for i, ev := range r.ToolEvents {
		out.ToolEvents[i] = ev
		out.ToolEvents[i].Metadata = cloneNavivoxEvidenceMap(ev.Metadata)
	}
	out.ArtifactRefs = append([]NavivoxArtifactRef(nil), r.ArtifactRefs...)
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

func navivoxVoiceEvidence(userText string, metadata map[string]any) *NavivoxVoiceEvidence {
	if !metadataMarksVoice(metadata) {
		return nil
	}
	rawStored := boolFromNavivoxAny(metadata["raw_audio_stored"])
	retention := safeNavivoxEvidenceString(stringFromNavivoxAny(metadata["raw_audio_retention"]), 128)
	if retention == "" {
		if rawStored {
			retention = "stored"
		} else {
			retention = "not_stored"
		}
	}
	deviceProvider := safeNavivoxEvidenceString(stringFromNavivoxAny(metadata["stt_provider"]), 128)
	if deviceProvider == "" {
		deviceProvider = "device"
	}
	deviceStatus := safeNavivoxEvidenceString(stringFromNavivoxAny(metadata["stt_status"]), 64)
	if deviceStatus == "" {
		deviceStatus = NavivoxEvidenceAvailable
	}
	serverProvider := safeNavivoxEvidenceString(firstNavivoxString(metadata, "server_stt_provider", "server_stt"), 128)
	serverStatus := safeNavivoxEvidenceString(firstNavivoxString(metadata, "server_stt_status"), 64)
	if serverStatus == "" {
		if serverProvider == "" || strings.EqualFold(serverProvider, "none") {
			serverStatus = NavivoxEvidenceUnavailable
		} else {
			serverStatus = NavivoxEvidenceAvailable
		}
	}
	ttsProvider := safeNavivoxEvidenceString(firstNavivoxString(metadata, "tts_provider"), 128)
	ttsStatus := safeNavivoxEvidenceString(firstNavivoxString(metadata, "tts_status"), 64)
	if ttsStatus == "" {
		if ttsProvider == "" {
			ttsStatus = NavivoxEvidenceUnknown
		} else {
			ttsStatus = NavivoxEvidenceAvailable
		}
	}
	return &NavivoxVoiceEvidence{
		DeviceTranscript: safeNavivoxEvidenceString(userText, 4096),
		DeviceSTT:        NavivoxSTTEvidence{Provider: deviceProvider, Status: deviceStatus},
		ServerSTT:        NavivoxSTTEvidence{Provider: serverProvider, Status: serverStatus},
		Audio: NavivoxAudioMeta{
			DurationMS:     intFromNavivoxAny(firstNavivoxAny(metadata, "audio_duration_ms", "duration_ms")),
			Codec:          safeNavivoxEvidenceString(firstNavivoxString(metadata, "audio_codec", "codec"), 128),
			RawAudioStored: rawStored,
			Retention:      retention,
		},
		TTS: NavivoxTTSEvidence{
			Provider: ttsProvider,
			VoiceID:  safeNavivoxEvidenceString(firstNavivoxString(metadata, "tts_voice_id", "voice_id"), 128),
			Status:   ttsStatus,
		},
	}
}

func metadataMarksVoice(metadata map[string]any) bool {
	if len(metadata) == 0 {
		return false
	}
	if strings.EqualFold(stringFromNavivoxAny(metadata["input_kind"]), "voice") {
		return true
	}
	if boolFromNavivoxAny(metadata["voice"]) {
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

func sanitizeNavivoxEvidenceMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := map[string]any{}
	for key, value := range raw {
		key = safeNavivoxMapKey(key)
		if key == "" || navivoxSensitiveKey(key) {
			continue
		}
		switch typed := value.(type) {
		case string:
			out[key] = safeNavivoxEvidenceString(typed, 512)
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
			out[key] = safeNavivoxEvidenceString(fmt.Sprint(typed), 512)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneNavivoxEvidenceMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	return out
}

func navivoxSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	for _, marker := range []string{"secret", "token", "password", "api_key", "apikey", "credential", "raw_audio", "audio_bytes"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func safeNavivoxMapKey(raw string) string {
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

func safeNavivoxEvidenceString(raw string, maxRunes int) string {
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

func firstNavivoxString(metadata map[string]any, keys ...string) string {
	return stringFromNavivoxAny(firstNavivoxAny(metadata, keys...))
}

func firstNavivoxAny(metadata map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			return value
		}
	}
	return nil
}

func stringFromNavivoxAny(value any) string {
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

func intFromNavivoxAny(value any) int {
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

func boolFromNavivoxAny(value any) bool {
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

func normalizeNavivoxTime(t time.Time) time.Time {
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
