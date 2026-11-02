package acp

import (
	"strings"
	"testing"
)

func optionIDs(options []PermissionOption) []string {
	ids := make([]string, 0, len(options))
	for _, option := range options {
		ids = append(ids, option.OptionID)
	}
	return ids
}

func TestBuildPermissionOptionsMatchesHermesApprovalSemantics(t *testing.T) {
	tests := []struct {
		name        string
		permanent   bool
		session     bool
		smartDenied bool
		want        []string
	}{
		{
			name: "full menu", permanent: true, session: true,
			want: []string{"allow_once", "allow_session", "allow_always", "deny", "deny_always"},
		},
		{
			name: "no permanent", permanent: false, session: true,
			want: []string{"allow_once", "allow_session", "deny", "deny_always"},
		},
		{
			name: "no session scope", permanent: true, session: false,
			want: []string{"allow_once", "deny"},
		},
		{
			name: "smart denied collapses", permanent: true, session: true, smartDenied: true,
			want: []string{"allow_once", "deny"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optionIDs(BuildPermissionOptions(tt.permanent, tt.session, tt.smartDenied))
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("options = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPermissionDecisionForOptionMapsHermesResults(t *testing.T) {
	options := BuildPermissionOptions(true, true, false)
	tests := []struct {
		optionID string
		result   string
		outcome  PermissionOptionKind
	}{
		{"allow_once", "once", PermissionAllowOnce},
		{"allow_session", "session", PermissionAllowAlways},
		{"allow_always", "always", PermissionAllowAlways},
		{"deny", "deny", PermissionRejectOnce},
		{"deny_always", "deny", PermissionRejectOnce},
		{"unknown", "deny", PermissionRejectOnce},
	}
	for _, tt := range tests {
		decision := PermissionDecisionForOption(tt.optionID, options)
		if decision.Result != tt.result || decision.Outcome != tt.outcome {
			t.Fatalf("option %q = %+v, want result %q outcome %q", tt.optionID, decision, tt.result, tt.outcome)
		}
	}
}

func TestNewPermissionToolCallShape(t *testing.T) {
	call := NewPermissionToolCall("rm -rf /tmp/x", "dangerous command")
	if call.ToolCallID == "" || !strings.HasPrefix(call.ToolCallID, "perm-check-") {
		t.Fatalf("tool call id = %q, want perm-check-N", call.ToolCallID)
	}
	if call.Kind != ToolKindExecute || call.Status != "pending" {
		t.Fatalf("tool call = %+v, want execute/pending", call)
	}
	if !strings.Contains(call.Title, "dangerous command") || !strings.Contains(call.Title, "rm -rf /tmp/x") {
		t.Fatalf("title = %q", call.Title)
	}
	if len(call.Content) != 1 || call.Content[0].Content == nil || !strings.Contains(call.Content[0].Content.Text, "$ rm -rf /tmp/x") {
		t.Fatalf("content = %#v", call.Content)
	}
	if call.RawInput["command"] != "rm -rf /tmp/x" || call.RawInput["description"] != "dangerous command" {
		t.Fatalf("raw input = %#v", call.RawInput)
	}
	other := NewPermissionToolCall("echo hi", "")
	if other.ToolCallID == call.ToolCallID {
		t.Fatalf("tool call ids collide: %q", call.ToolCallID)
	}
}
