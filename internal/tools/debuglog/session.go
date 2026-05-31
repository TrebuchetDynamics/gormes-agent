package debuglog

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	redactedDebugValue          = "[redacted]"
	DebugEvidenceDisabled       = "debug_disabled"
	DebugEvidenceLogUnavailable = "debug_log_unavailable"
)

// DebugSessionConfig configures a per-tool debug log session. The injectable
// seams keep tests hermetic and keep production callers from reading live state
// unless they explicitly opt in through the tool-specific environment variable.
type DebugSessionConfig struct {
	ToolName  string
	EnvVar    string
	LogDir    string
	LookupEnv func(string) string
	Now       func() time.Time
	NewID     func() string
}

// DebugSessionInfo is the bounded external summary for a debug session.
type DebugSessionInfo struct {
	Enabled    bool   `json:"enabled"`
	SessionID  string `json:"session_id,omitempty"`
	LogPath    string `json:"log_path,omitempty"`
	TotalCalls int    `json:"total_calls"`
}

// DebugSessionSaveResult reports whether a debug log persisted or degraded.
type DebugSessionSaveResult struct {
	Saved    bool   `json:"saved"`
	Evidence string `json:"evidence,omitempty"`
	Error    string `json:"error,omitempty"`
	LogPath  string `json:"log_path,omitempty"`
}

// DebugSession records optional per-tool debug calls to a JSON log file. When
// disabled it is a cheap no-op. When enabled it sanitizes secret/content-shaped
// fields before retaining or writing evidence.
type DebugSession struct {
	toolName string
	enabled  bool
	id       string
	logDir   string
	logPath  string
	now      func() time.Time

	mu        sync.Mutex
	startTime string
	calls     []map[string]any
}

func NewDebugSession(cfg DebugSessionConfig) *DebugSession {
	lookup := cfg.LookupEnv
	if lookup == nil {
		lookup = os.Getenv
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	toolName := strings.TrimSpace(cfg.ToolName)
	if toolName == "" {
		toolName = "tool"
	}
	logDir := strings.TrimSpace(cfg.LogDir)
	if logDir == "" {
		logDir = filepath.Join(os.TempDir(), "gormes-debug")
	}
	enabled := strings.EqualFold(strings.TrimSpace(lookup(cfg.EnvVar)), "true")
	s := &DebugSession{toolName: toolName, enabled: enabled, logDir: logDir, now: now}
	if !enabled {
		return s
	}
	newID := cfg.NewID
	if newID == nil {
		newID = randomDebugSessionID
	}
	s.id = strings.TrimSpace(newID())
	if s.id == "" {
		s.id = randomDebugSessionID()
	}
	s.startTime = now().Format(time.RFC3339Nano)
	s.logPath = filepath.Join(logDir, toolName+"_debug_"+s.id+".json")
	return s
}

func (s *DebugSession) Active() bool {
	return s != nil && s.enabled
}

func (s *DebugSession) SessionID() string {
	if s == nil {
		return ""
	}
	return s.id
}

func (s *DebugSession) LogCall(callName string, callData map[string]any) {
	if s == nil || !s.enabled {
		return
	}
	entry := map[string]any{
		"timestamp": s.now().Format(time.RFC3339Nano),
		"tool_name": strings.TrimSpace(callName),
	}
	for k, v := range sanitizeDebugMap(callData) {
		entry[k] = v
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, entry)
}

func (s *DebugSession) Save() error {
	result := s.SaveResult()
	if result.Saved || result.Evidence == DebugEvidenceDisabled {
		return nil
	}
	if result.Error == "" {
		return errors.New(result.Evidence)
	}
	return errors.New(result.Error)
}

func (s *DebugSession) SaveResult() DebugSessionSaveResult {
	if s == nil || !s.enabled {
		return DebugSessionSaveResult{Evidence: DebugEvidenceDisabled}
	}
	s.mu.Lock()
	calls := cloneDebugCalls(s.calls)
	s.mu.Unlock()
	if err := os.MkdirAll(s.logDir, 0o700); err != nil {
		return DebugSessionSaveResult{Evidence: DebugEvidenceLogUnavailable, Error: "debug log directory unavailable"}
	}
	payload := struct {
		SessionID    string           `json:"session_id"`
		StartTime    string           `json:"start_time"`
		EndTime      string           `json:"end_time"`
		DebugEnabled bool             `json:"debug_enabled"`
		TotalCalls   int              `json:"total_calls"`
		ToolCalls    []map[string]any `json:"tool_calls"`
	}{
		SessionID:    s.id,
		StartTime:    s.startTime,
		EndTime:      s.now().Format(time.RFC3339Nano),
		DebugEnabled: true,
		TotalCalls:   len(calls),
		ToolCalls:    calls,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return DebugSessionSaveResult{Evidence: DebugEvidenceLogUnavailable, Error: "debug log encode failed"}
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(s.logPath, raw, 0o600); err != nil {
		return DebugSessionSaveResult{Evidence: DebugEvidenceLogUnavailable, Error: "debug log write failed"}
	}
	return DebugSessionSaveResult{Saved: true, LogPath: s.logPath}
}

func (s *DebugSession) Info() DebugSessionInfo {
	if s == nil || !s.enabled {
		return DebugSessionInfo{}
	}
	s.mu.Lock()
	calls := len(s.calls)
	s.mu.Unlock()
	return DebugSessionInfo{Enabled: true, SessionID: s.id, LogPath: s.logPath, TotalCalls: calls}
}

func sanitizeDebugMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = sanitizeDebugValue(k, v)
	}
	return out
}

func sanitizeDebugValue(key string, v any) any {
	if isSensitiveDebugKey(key) {
		return redactedDebugValue
	}
	switch value := v.(type) {
	case map[string]any:
		return sanitizeDebugMap(value)
	case map[string]string:
		return sanitizeDebugStringMap(value)
	case map[string][]string:
		return sanitizeDebugStringSliceMap(value)
	case []any:
		out := make([]any, len(value))
		for i, nested := range value {
			out[i] = sanitizeDebugValue(key, nested)
		}
		return out
	default:
		return value
	}
}

func sanitizeDebugStringMap(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, nested := range in {
		out[k] = sanitizeDebugValue(k, nested)
	}
	return out
}

func sanitizeDebugStringSliceMap(in map[string][]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, values := range in {
		sanitized := make([]any, len(values))
		for i, nested := range values {
			sanitized[i] = sanitizeDebugValue(k, nested)
		}
		out[k] = sanitized
	}
	return out
}

func isSensitiveDebugKey(key string) bool {
	k := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
	if k == "" {
		return false
	}
	markers := []string{
		"api_key", "apikey", "access_token", "refresh_token", "token", "secret", "password",
		"authorization", "cookie", "set_cookie", "provider_response_body", "response_body",
		"request_body", "file_contents", "file_content", "raw_body", "body",
	}
	for _, marker := range markers {
		if k == marker || strings.Contains(k, marker) {
			return true
		}
	}
	return false
}

func cloneDebugCalls(in []map[string]any) []map[string]any {
	out := make([]map[string]any, len(in))
	for i, row := range in {
		out[i] = sanitizeDebugMap(row)
	}
	return out
}

func randomDebugSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000"), ".", "")
	}
	return hex.EncodeToString(b[:])
}
