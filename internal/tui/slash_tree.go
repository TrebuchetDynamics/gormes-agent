package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const sessionTreeTimeout = 5 * time.Second

type SessionTreeFunc func(context.Context, SessionTreeRequest) (SessionTreeResult, error)

type SessionTreeLabelRequest struct {
	SessionID string
	Action    string
	Label     string
}

type SessionTreeLabelResult struct {
	SessionID string
	Labels    []string
}

type SessionTreeLabelFunc func(context.Context, SessionTreeLabelRequest) (SessionTreeLabelResult, error)

type SessionTreeRestoreRequest struct {
	SessionID string
	MessageID int64
}

type SessionTreeRestoreResult struct {
	Text     string
	Editable bool
	Evidence string
}

type SessionTreeRestoreFunc func(context.Context, SessionTreeRestoreRequest) (SessionTreeRestoreResult, error)

func treeSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "tree: TUI unavailable"}
	}
	args := slashArgs(input)
	if len(args) == 0 || treeSlashIsFilter(args[0]) {
		return openTreeSlash(args, model)
	}
	switch strings.ToLower(args[0]) {
	case "label":
		return treeLabelSlash(args[1:], model, "set")
	case "unlabel", "clear-label", "clear":
		return treeLabelSlash(args[1:], model, "clear")
	case "restore", "edit":
		return treeRestoreSlash(args[1:], model)
	default:
		return SlashResult{Handled: true, StatusMessage: "tree: usage /tree [--filter MODE] | /tree label <session> <label> | /tree unlabel <session> [label] | /tree restore <session> <turn_id>"}
	}
}

func openTreeSlash(args []string, model *Model) SlashResult {
	if model.sessionTree == nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "tree: session tree unavailable"}
	}
	filter := parseTreeSlashFilter(args)
	ctx, cancel := context.WithTimeout(context.Background(), sessionTreeTimeout)
	defer cancel()
	result, err := model.sessionTree(ctx, SessionTreeRequest{Filter: filter, ActiveSessionID: model.SessionID()})
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "tree: " + err.Error()}
	}
	if result.Filter == "" {
		result.Filter = filter
	}
	if result.ActiveSessionID == "" {
		result.ActiveSessionID = model.SessionID()
	}
	page, ok := BuildSessionTreePage(result)
	if !ok {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "tree: no sessions found"}
	}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: "tree opened"}
}

func treeLabelSlash(args []string, model *Model, action string) SlashResult {
	if model.sessionTreeLabel == nil {
		return SlashResult{Handled: true, StatusMessage: "tree: labels unavailable"}
	}
	if len(args) < 1 || (action == "set" && len(args) < 2) {
		return SlashResult{Handled: true, StatusMessage: "tree: label usage /tree label <session> <label>"}
	}
	req := SessionTreeLabelRequest{SessionID: strings.TrimSpace(args[0]), Action: action}
	if len(args) > 1 {
		req.Label = strings.TrimSpace(strings.Join(args[1:], " "))
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTreeTimeout)
	defer cancel()
	result, err := model.sessionTreeLabel(ctx, req)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "tree: labels: " + err.Error()}
	}
	labels := "none"
	if len(result.Labels) > 0 {
		labels = strings.Join(result.Labels, ", ")
	}
	return SlashResult{Handled: true, StatusMessage: fmt.Sprintf("tree: labels for %s: %s", firstNonEmptyString(result.SessionID, req.SessionID), labels)}
}

func treeRestoreSlash(args []string, model *Model) SlashResult {
	if model.inFlight || turnIsActive(model.frame.Phase) {
		return SlashResult{Handled: true, StatusMessage: "tree: restore unavailable while turn is active"}
	}
	if model.sessionTreeRestore == nil {
		return SlashResult{Handled: true, StatusMessage: "tree: restore unavailable"}
	}
	if len(args) < 2 {
		return SlashResult{Handled: true, StatusMessage: "tree: restore usage /tree restore <session> <turn_id>"}
	}
	turnID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || turnID <= 0 {
		return SlashResult{Handled: true, StatusMessage: "tree: restore turn_id must be a positive integer"}
	}
	req := SessionTreeRestoreRequest{SessionID: strings.TrimSpace(args[0]), MessageID: turnID}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTreeTimeout)
	defer cancel()
	result, err := model.sessionTreeRestore(ctx, req)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "tree: restore: " + err.Error()}
	}
	if !result.Editable {
		evidence := strings.TrimSpace(result.Evidence)
		if evidence == "" {
			evidence = "replay_unavailable"
		}
		return SlashResult{Handled: true, StatusMessage: "tree: replay unavailable: " + evidence}
	}
	return SlashResult{Handled: true, StatusMessage: fmt.Sprintf("tree: restored editable prompt from %s#%d", req.SessionID, req.MessageID), EditorText: result.Text}
}

func slashArgs(input string) []string {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		return nil
	}
	return fields[1:]
}

func treeSlashIsFilter(arg string) bool {
	arg = strings.ToLower(strings.TrimSpace(arg))
	return arg == "--filter" || strings.HasPrefix(arg, "--filter=") || arg == "filter"
}

func parseTreeSlashFilter(args []string) string {
	if len(args) == 0 {
		return "default"
	}
	first := strings.ToLower(strings.TrimSpace(args[0]))
	switch {
	case first == "--filter" || first == "filter":
		if len(args) > 1 {
			return normalizeTreeSlashFilter(args[1])
		}
	case strings.HasPrefix(first, "--filter="):
		return normalizeTreeSlashFilter(strings.TrimPrefix(first, "--filter="))
	}
	return normalizeTreeSlashFilter(first)
}
