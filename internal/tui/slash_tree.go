package tui

import (
	"context"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/sessiontree"
)

const sessionTreeTimeout = 5 * time.Second

type SessionTreeFunc = sessiontree.QueryFunc

type SessionTreeLabelRequest = sessiontree.LabelRequest

type SessionTreeLabelResult = sessiontree.LabelResult

type SessionTreeLabelFunc = sessiontree.LabelFunc

type SessionTreeRestoreRequest = sessiontree.RestoreRequest

type SessionTreeRestoreResult = sessiontree.RestoreResult

type SessionTreeRestoreFunc = sessiontree.RestoreFunc

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
	parsed, status, ok := sessiontree.ParseLabelRequest(args, action)
	if !ok {
		return SlashResult{Handled: true, StatusMessage: status}
	}
	req := SessionTreeLabelRequest{SessionID: parsed.SessionID, Action: parsed.Action, Label: parsed.Label}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTreeTimeout)
	defer cancel()
	result, err := model.sessionTreeLabel(ctx, req)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "tree: labels: " + err.Error()}
	}
	return SlashResult{Handled: true, StatusMessage: sessiontree.FormatLabelStatus(result.SessionID, req.SessionID, result.Labels)}
}

func treeRestoreSlash(args []string, model *Model) SlashResult {
	if model.inFlight || turnIsActive(model.frame.Phase) {
		return SlashResult{Handled: true, StatusMessage: "tree: restore unavailable while turn is active"}
	}
	if model.sessionTreeRestore == nil {
		return SlashResult{Handled: true, StatusMessage: "tree: restore unavailable"}
	}
	parsed, status, ok := sessiontree.ParseRestoreRequest(args)
	if !ok {
		return SlashResult{Handled: true, StatusMessage: status}
	}
	req := SessionTreeRestoreRequest{SessionID: parsed.SessionID, MessageID: parsed.MessageID}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTreeTimeout)
	defer cancel()
	result, err := model.sessionTreeRestore(ctx, req)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "tree: restore: " + err.Error()}
	}
	status, editable := sessiontree.FormatRestoreStatus(req, result)
	if !editable {
		return SlashResult{Handled: true, StatusMessage: status}
	}
	return SlashResult{Handled: true, StatusMessage: status, EditorText: result.Text}
}

func slashArgs(input string) []string {
	return sessiontree.SlashArgs(input)
}

func treeSlashIsFilter(arg string) bool {
	return sessiontree.SlashIsFilter(arg)
}

func parseTreeSlashFilter(args []string) string {
	return sessiontree.ParseSlashFilter(args)
}
