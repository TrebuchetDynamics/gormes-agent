package sessiontree

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	LabelUsage   = "tree: label usage /tree label <session> <label>"
	RestoreUsage = "tree: restore usage /tree restore <session> <turn_id>"
)

// QueryFunc fetches session lineage entries for /tree page rendering.
type QueryFunc func(context.Context, SessionTreeRequest) (SessionTreeResult, error)

// LabelRequest is the pure, UI-package-independent shape parsed from a
// /tree label or /tree unlabel command and sent to the injected label callback.
type LabelRequest struct {
	SessionID string
	Action    string
	Label     string
}

// LabelResult is the label callback response shape needed to format the
// operator-facing status message after label mutation.
type LabelResult struct {
	SessionID string
	Labels    []string
}

// LabelFunc mutates labels for a session tree entry.
type LabelFunc func(context.Context, LabelRequest) (LabelResult, error)

// RestoreRequest is the pure, UI-package-independent shape parsed from a
// /tree restore command and sent to the injected restore callback.
type RestoreRequest struct {
	SessionID string
	MessageID int64
}

// RestoreResult is the restore callback response shape needed to decide
// whether an old prompt can be placed back into the editor.
type RestoreResult struct {
	Text     string
	Editable bool
	Evidence string
}

// RestoreFunc restores an editable prompt from a prior session-tree turn.
type RestoreFunc func(context.Context, RestoreRequest) (RestoreResult, error)

func SlashArgs(input string) []string {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		return nil
	}
	return fields[1:]
}

func SlashIsFilter(arg string) bool {
	arg = strings.ToLower(strings.TrimSpace(arg))
	return arg == "--filter" || strings.HasPrefix(arg, "--filter=") || arg == "filter"
}

func ParseSlashFilter(args []string) string {
	if len(args) == 0 {
		return "default"
	}
	first := strings.ToLower(strings.TrimSpace(args[0]))
	switch {
	case first == "--filter" || first == "filter":
		if len(args) > 1 {
			return NormalizeFilter(args[1])
		}
	case strings.HasPrefix(first, "--filter="):
		return NormalizeFilter(strings.TrimPrefix(first, "--filter="))
	}
	return NormalizeFilter(first)
}

// ParseLabelRequest validates label/unlabel slash arguments while leaving the
// actual label callback and model mutation in the root TUI package.
func ParseLabelRequest(args []string, action string) (LabelRequest, string, bool) {
	if len(args) < 1 || (action == "set" && len(args) < 2) {
		return LabelRequest{}, LabelUsage, false
	}
	req := LabelRequest{SessionID: strings.TrimSpace(args[0]), Action: action}
	if len(args) > 1 {
		req.Label = strings.TrimSpace(strings.Join(args[1:], " "))
	}
	return req, "", true
}

// FormatLabelStatus returns the stable operator-facing status for a successful
// label/unlabel callback.
func FormatLabelStatus(sessionID, fallbackSessionID string, labels []string) string {
	labelList := "none"
	if len(labels) > 0 {
		labelList = strings.Join(labels, ", ")
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = fallbackSessionID
	}
	return fmt.Sprintf("tree: labels for %s: %s", sessionID, labelList)
}

// ParseRestoreRequest validates restore slash arguments and parses the turn id.
func ParseRestoreRequest(args []string) (RestoreRequest, string, bool) {
	if len(args) < 2 {
		return RestoreRequest{}, RestoreUsage, false
	}
	turnID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || turnID <= 0 {
		return RestoreRequest{}, "tree: restore turn_id must be a positive integer", false
	}
	return RestoreRequest{SessionID: strings.TrimSpace(args[0]), MessageID: turnID}, "", true
}

// FormatRestoreStatus returns the status text for restore callback evidence and
// reports whether the restored prompt is editable.
func FormatRestoreStatus(req RestoreRequest, result RestoreResult) (string, bool) {
	if !result.Editable {
		evidence := strings.TrimSpace(result.Evidence)
		if evidence == "" {
			evidence = "replay_unavailable"
		}
		return "tree: replay unavailable: " + evidence, false
	}
	return fmt.Sprintf("tree: restored editable prompt from %s#%d", req.SessionID, req.MessageID), true
}
