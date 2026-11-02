package acp

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// PermissionOption is one choice offered to an ACP client for a dangerous
// command. Option IDs carry Hermes approval semantics because ACP has no
// session-scoped option kind.
type PermissionOption struct {
	OptionID string               `json:"optionId"`
	Kind     PermissionOptionKind `json:"kind"`
	Name     string               `json:"name"`
}

// ToolCallContent is an ACP tool-call content item. Text content is wrapped in
// the union's content variant.
type ToolCallContent struct {
	Type    string           `json:"type"`
	Content *ACPContentBlock `json:"content,omitempty"`
}

// TextToolCallContent builds the text variant of ToolCallContent.
func TextToolCallContent(text string) ToolCallContent {
	return ToolCallContent{Type: "content", Content: &ACPContentBlock{Type: "text", Text: text}}
}

// PermissionToolCall is the tool-call update attached to a permission request.
// It uses a unique perm-check-N id so concurrent requests stay separate.
type PermissionToolCall struct {
	ToolCallID string            `json:"toolCallId"`
	Title      string            `json:"title"`
	Kind       ToolKind          `json:"kind"`
	Status     string            `json:"status"`
	Content    []ToolCallContent `json:"content,omitempty"`
	RawInput   map[string]any    `json:"rawInput,omitempty"`
}

var permissionRequestSeq atomic.Int64

// BuildPermissionOptions mirrors Hermes approval semantics. A gate that
// re-asks every time (allowSession=false, or smartDenied) collapses to
// once/deny so the editor never offers a scope Hermes discards.
func BuildPermissionOptions(allowPermanent, allowSession, smartDenied bool) []PermissionOption {
	onceOnly := smartDenied || !allowSession
	options := []PermissionOption{
		{OptionID: "allow_once", Kind: PermissionAllowOnce, Name: "Allow once"},
	}
	if !onceOnly {
		options = append(options, PermissionOption{OptionID: "allow_session", Kind: PermissionAllowAlways, Name: "Allow for session"})
		if allowPermanent {
			options = append(options, PermissionOption{OptionID: "allow_always", Kind: PermissionAllowAlways, Name: "Allow always"})
		}
	}
	options = append(options, PermissionOption{OptionID: "deny", Kind: PermissionRejectOnce, Name: "Deny"})
	if !onceOnly {
		options = append(options, PermissionOption{OptionID: "deny_always", Kind: PermissionRejectAlways, Name: "Deny always"})
	}
	return options
}

// NewPermissionToolCall builds the pending ToolCallUpdate attached to a
// permission request.
func NewPermissionToolCall(command, description string) PermissionToolCall {
	command = strings.TrimSpace(command)
	description = strings.TrimSpace(description)
	title := command
	if description != "" {
		title = description + ": " + command
	}
	text := "$ " + command
	if description != "" {
		text = description + "\n$ " + command
	}
	id := fmt.Sprintf("perm-check-%d", permissionRequestSeq.Add(1))
	return PermissionToolCall{
		ToolCallID: id,
		Title:      title,
		Kind:       ToolKindExecute,
		Status:     "pending",
		Content:    []ToolCallContent{TextToolCallContent(text)},
		RawInput:   map[string]any{"command": command, "description": description},
	}
}

// PermissionDecisionForOption maps a client-selected option id to a Hermes
// approval result. Unknown or disallowed options deny.
func PermissionDecisionForOption(optionID string, options []PermissionOption) PermissionDecision {
	optionID = strings.TrimSpace(optionID)
	allowed := false
	for _, option := range options {
		if option.OptionID == optionID {
			allowed = true
			break
		}
	}
	if !allowed {
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "unknown_permission_option"}
	}
	switch optionID {
	case "allow_once":
		return PermissionDecision{Outcome: PermissionAllowOnce, Result: "once"}
	case "allow_session":
		return PermissionDecision{Outcome: PermissionAllowAlways, Result: "session"}
	case "allow_always":
		return PermissionDecision{Outcome: PermissionAllowAlways, Result: "always"}
	case "deny", "deny_always":
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny"}
	default:
		return PermissionDecision{Outcome: PermissionRejectOnce, Result: "deny", Reason: "unknown_permission_option"}
	}
}
