package debuglog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDebugSessionDisabledIsCheapNoop(t *testing.T) {
	dir := t.TempDir()
	session := NewDebugSession(DebugSessionConfig{
		ToolName: "web_tools",
		EnvVar:   "WEB_TOOLS_DEBUG",
		LogDir:   dir,
		LookupEnv: func(string) string {
			return ""
		},
		Now:   func() time.Time { return time.Unix(10, 0).UTC() },
		NewID: func() string { return "should-not-be-used" },
	})

	if session.Active() {
		t.Fatalf("disabled session reported active")
	}
	if session.SessionID() != "" {
		t.Fatalf("disabled session id = %q, want empty", session.SessionID())
	}
	session.LogCall("web_search", map[string]any{"query": "hello"})
	if err := session.Save(); err != nil {
		t.Fatalf("Save disabled session: %v", err)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("disabled session wrote entries=%v err=%v", entries, err)
	}
	info := session.Info()
	if info.Enabled || info.SessionID != "" || info.LogPath != "" || info.TotalCalls != 0 {
		t.Fatalf("disabled info = %+v", info)
	}
}

func TestDebugSessionSaveFailureDegrades(t *testing.T) {
	logDirFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(logDirFile, []byte("plain file"), 0o600); err != nil {
		t.Fatalf("write logDir fixture: %v", err)
	}
	session := NewDebugSession(DebugSessionConfig{
		ToolName: "web_tools",
		EnvVar:   "WEB_TOOLS_DEBUG",
		LogDir:   logDirFile,
		LookupEnv: func(name string) string {
			if name == "WEB_TOOLS_DEBUG" {
				return "true"
			}
			return ""
		},
		Now:   func() time.Time { return time.Unix(123, 0).UTC() },
		NewID: func() string { return "debug-session-2" },
	})

	result := session.SaveResult()
	if result.Saved {
		t.Fatalf("SaveResult saved with file-as-dir log root: %+v", result)
	}
	if result.Evidence != DebugEvidenceLogUnavailable {
		t.Fatalf("SaveResult evidence = %q, want %q (result=%+v)", result.Evidence, DebugEvidenceLogUnavailable, result)
	}
	if result.Error == "" || strings.Contains(result.Error, logDirFile) {
		t.Fatalf("SaveResult error should be sanitized and path-free, got %+v", result)
	}
}

func TestDebugSessionEnabledSavesRedactedJSON(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(123, 0).UTC()
	session := NewDebugSession(DebugSessionConfig{
		ToolName: "web_tools",
		EnvVar:   "WEB_TOOLS_DEBUG",
		LogDir:   dir,
		LookupEnv: func(name string) string {
			if name == "WEB_TOOLS_DEBUG" {
				return "TRUE"
			}
			return ""
		},
		Now: func() time.Time {
			now = now.Add(time.Second)
			return now
		},
		NewID: func() string { return "debug-session-1" },
	})

	if !session.Active() {
		t.Fatalf("enabled session reported inactive")
	}
	session.LogCall("web_search", map[string]any{
		"query":         "safe query",
		"api_key":       "plain-existing-token",
		"Authorization": "Bearer plain-existing-token",
		"headers": map[string]any{
			"Cookie": "session=plain-existing-token",
			"Accept": "application/json",
		},
		"request_headers": map[string][]string{
			"Authorization": {"Bearer plain-existing-token"},
			"Accept":        {"application/json"},
		},
		"provider_response_body": "raw provider body that should not be stored",
		"file_contents":          "local file contents that should not be stored",
	})
	if err := session.Save(); err != nil {
		t.Fatalf("Save enabled session: %v", err)
	}

	path := filepath.Join(dir, "web_tools_debug_debug-session-1.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read debug log: %v", err)
	}
	text := string(raw)
	for _, leaked := range []string{"plain-existing-token", "Bearer plain-existing-token", "raw provider body", "local file contents"} {
		if strings.Contains(text, leaked) {
			t.Fatalf("debug log leaked %q: %s", leaked, text)
		}
	}
	var payload struct {
		SessionID    string           `json:"session_id"`
		DebugEnabled bool             `json:"debug_enabled"`
		TotalCalls   int              `json:"total_calls"`
		ToolCalls    []map[string]any `json:"tool_calls"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode debug log: %v", err)
	}
	if payload.SessionID != "debug-session-1" || !payload.DebugEnabled || payload.TotalCalls != 1 {
		t.Fatalf("payload summary = %+v", payload)
	}
	call := payload.ToolCalls[0]
	if call["tool_name"] != "web_search" || call["query"] != "safe query" {
		t.Fatalf("call payload = %+v", call)
	}
	if call["api_key"] != "[redacted]" || call["Authorization"] != "[redacted]" {
		t.Fatalf("top-level secret fields not redacted: %+v", call)
	}
	headers, ok := call["headers"].(map[string]any)
	if !ok || headers["Cookie"] != "[redacted]" || headers["Accept"] != "application/json" {
		t.Fatalf("nested headers not sanitized: %#v", call["headers"])
	}
	requestHeaders, ok := call["request_headers"].(map[string]any)
	if !ok || requestHeaders["Authorization"] != "[redacted]" || !reflect.DeepEqual(requestHeaders["Accept"], []any{"application/json"}) {
		t.Fatalf("string-slice headers not sanitized: %#v", call["request_headers"])
	}
	if call["provider_response_body"] != "[redacted]" || call["file_contents"] != "[redacted]" {
		t.Fatalf("sensitive body/content fields not redacted: %+v", call)
	}
	info := session.Info()
	if !info.Enabled || info.SessionID != "debug-session-1" || info.TotalCalls != 1 || info.LogPath != path {
		t.Fatalf("enabled info = %+v, want path %s", info, path)
	}
}
