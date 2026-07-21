package apiserver

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestClientCompatibility_WingAndDesktopSessionContract(t *testing.T) {
	srv := NewServer(Config{
		APIKey:                "wing-token",
		DashboardSessionToken: "desktop-token",
		SessionsList: func() []DashboardSession {
			return []DashboardSession{{ID: "persisted", Title: "Persisted", MessageCount: 2}}
		},
		ChatHistory: func(sessionID string) []DashboardChatMessage {
			if sessionID != "persisted" {
				return nil
			}
			return []DashboardChatMessage{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "hi"}}
		},
	})
	h := srv.Handler()
	wingAuth := map[string]string{"Authorization": "Bearer wing-token"}
	desktopAuth := map[string]string{"X-Hermes-Session-Token": "desktop-token"}

	capabilities := getJSON(t, h, "/v1/capabilities", wingAuth)
	if capabilities.Code != http.StatusOK {
		t.Fatalf("capabilities status = %d; body=%s", capabilities.Code, capabilities.Body.String())
	}
	var caps struct {
		SchemaVersion int `json:"schema_version"`
		Endpoints     map[string]struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(capabilities.Body.Bytes(), &caps); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if caps.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", caps.SchemaVersion)
	}
	for name, want := range map[string]string{
		"sessions":         "/api/sessions",
		"session_create":   "/api/sessions",
		"session_messages": "/api/sessions/{session_id}/messages",
	} {
		if got := caps.Endpoints[name].Path; got != want {
			t.Errorf("endpoint %s = %q, want %q", name, got, want)
		}
	}

	created := postJSON(t, h, "/api/sessions", map[string]any{"id": "wing-empty", "title": "Wing"}, wingAuth)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", created.Code, created.Body.String())
	}

	wingList := getJSON(t, h, "/api/sessions", wingAuth)
	if wingList.Code != http.StatusOK {
		t.Fatalf("Wing list status = %d; body=%s", wingList.Code, wingList.Body.String())
	}
	var wingBody struct {
		Data []DashboardSessionInfo `json:"data"`
	}
	if err := json.Unmarshal(wingList.Body.Bytes(), &wingBody); err != nil {
		t.Fatalf("decode Wing sessions: %v", err)
	}
	if !hasDashboardSession(wingBody.Data, "persisted") || !hasDashboardSession(wingBody.Data, "wing-empty") {
		t.Fatalf("Wing data sessions = %+v, want persisted and wing-empty", wingBody.Data)
	}

	desktopList := getJSON(t, h, "/api/sessions", desktopAuth)
	if desktopList.Code != http.StatusOK {
		t.Fatalf("Desktop list status = %d; body=%s", desktopList.Code, desktopList.Body.String())
	}
	var desktopBody struct {
		Sessions []DashboardSessionInfo `json:"sessions"`
	}
	if err := json.Unmarshal(desktopList.Body.Bytes(), &desktopBody); err != nil {
		t.Fatalf("decode Desktop sessions: %v", err)
	}
	if !hasDashboardSession(desktopBody.Sessions, "persisted") {
		t.Fatalf("Desktop sessions = %+v, want persisted", desktopBody.Sessions)
	}

	messages := getJSON(t, h, "/api/sessions/persisted/messages", desktopAuth)
	if messages.Code != http.StatusOK {
		t.Fatalf("messages status = %d; body=%s", messages.Code, messages.Body.String())
	}
	var history struct {
		Data     []DashboardChatMessage `json:"data"`
		Messages []DashboardChatMessage `json:"messages"`
	}
	if err := json.Unmarshal(messages.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(history.Data) != 2 || len(history.Messages) != 2 {
		t.Fatalf("history = data:%+v messages:%+v, want two rows in both envelopes", history.Data, history.Messages)
	}

	unknown := getJSON(t, h, "/api/sessions/missing/messages", desktopAuth)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown messages status = %d, want 404", unknown.Code)
	}
	invalid := postJSON(t, h, "/api/sessions", map[string]any{"id": "../escape"}, wingAuth)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid session status = %d, want 400", invalid.Code)
	}
	deleted := deleteJSON(t, h, "/api/sessions/wing-empty", wingAuth)
	var deletion struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.Unmarshal(deleted.Body.Bytes(), &deletion); err != nil {
		t.Fatalf("decode deletion: %v", err)
	}
	if deleted.Code != http.StatusOK || deletion.ID != "wing-empty" || !deletion.Deleted {
		t.Fatalf("deletion = status:%d body:%+v", deleted.Code, deletion)
	}
}

func TestClientCompatibility_ClientSessionLimit(t *testing.T) {
	srv := NewServer(Config{APIKey: "wing-token"})
	for i := 0; i < maxClientSessions; i++ {
		id := string(rune('a'+i%26)) + string(rune('0'+i/26))
		srv.clientSessions[id] = DashboardSessionInfo{ID: id}
	}
	rec := postJSON(t, srv.Handler(), "/api/sessions", map[string]any{"id": "overflow"}, map[string]string{"Authorization": "Bearer wing-token"})
	if rec.Code != http.StatusTooManyRequests || !jsonErrorHasCode(rec.Body.Bytes(), "session_limit_exceeded") {
		t.Fatalf("overflow = %d %s, want 429 session_limit_exceeded", rec.Code, rec.Body.String())
	}
}

func TestClientCompatibility_RunDiscoveryStopAndApprovalFailClosed(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{APIKey: "wing-token", Loop: loop})
	h := srv.Handler()
	auth := map[string]string{"Authorization": "Bearer wing-token"}

	capabilities := getJSON(t, h, "/v1/capabilities", auth)
	var caps struct {
		Features  map[string]any `json:"features"`
		Endpoints map[string]struct {
			Path string `json:"path"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(capabilities.Body.Bytes(), &caps); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if caps.Features["run_approval_response"] != true {
		t.Fatalf("run_approval_response = %v, want true", caps.Features["run_approval_response"])
	}
	if got := caps.Endpoints["run_approval"].Path; got != "/v1/runs/{run_id}/approval" {
		t.Fatalf("run_approval path = %q", got)
	}

	started := postJSON(t, h, "/v1/runs", map[string]any{"session_id": "wing-run", "input": "wait"}, auth)
	if started.Code != http.StatusAccepted {
		t.Fatalf("start status = %d; body=%s", started.Code, started.Body.String())
	}
	var run struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(started.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	loop.waitStarted(t)

	invalid := postJSON(t, h, "/v1/runs/"+run.RunID+"/approval", map[string]any{"decision": "unsafe"}, auth)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid approval status = %d, want 400; body=%s", invalid.Code, invalid.Body.String())
	}
	inactive := postJSON(t, h, "/v1/runs/"+run.RunID+"/approval", map[string]any{"approval_id": "approval-1", "decision": "once"}, auth)
	if inactive.Code != http.StatusConflict || !jsonErrorHasCode(inactive.Body.Bytes(), "approval_not_active") {
		t.Fatalf("inactive approval = %d %s, want 409 approval_not_active", inactive.Code, inactive.Body.String())
	}

	stopped := postJSON(t, h, "/v1/runs/"+run.RunID+"/stop", map[string]any{}, auth)
	if stopped.Code != http.StatusOK {
		t.Fatalf("stop status = %d; body=%s", stopped.Code, stopped.Body.String())
	}
}

func jsonErrorHasCode(body []byte, want string) bool {
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal(body, &envelope) == nil && envelope.Error.Code == want
}

func hasDashboardSession(sessions []DashboardSessionInfo, id string) bool {
	for _, session := range sessions {
		if session.ID == id {
			return true
		}
	}
	return false
}
