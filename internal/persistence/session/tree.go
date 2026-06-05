package session

import (
	"sort"
	"strings"
)

type TreeFilter string

const (
	TreeFilterDefault       TreeFilter = "default"
	TreeFilterNoTools       TreeFilter = "no-tools"
	TreeFilterUserOnly      TreeFilter = "user-only"
	TreeFilterLabeledOnly   TreeFilter = "labeled-only"
	TreeFilterAllEquivalent TreeFilter = "all-equivalent"

	ReplayUnavailableEntryMissing  = "entry_missing"
	ReplayUnavailableNonUserEntry  = "non_user_entry"
	ReplayUnavailableEmptyUserTurn = "empty_user_turn"
)

type TreeOptions struct {
	ActiveSessionID string
	Filter          TreeFilter
}

type SessionTree struct {
	Filter          TreeFilter
	ActiveSessionID string
	Entries         []TreeEntry
}

type TreeEntry struct {
	SessionID       string
	ParentSessionID string
	LineageKind     string
	Title           string
	Labels          []string
	CreatedAt       int64
	UpdatedAt       int64
	Depth           int
	Active          bool
	Status          string
	Messages        []TreeMessage
}

type TreeMessage struct {
	ID                int64
	Role              string
	Content           string
	CreatedAtUnix     int64
	Replayable        bool
	ReplayUnavailable string
	TimestampKnown    bool
	TimestampEvidence string
}

func BuildSessionTree(items []Metadata, ledgers map[string]SessionLedger, opts TreeOptions) SessionTree {
	filter := normalizeTreeFilter(opts.Filter)
	byID := make(map[string]Metadata, len(items)+len(ledgers))
	for _, item := range items {
		item = finalizeMetadata(item)
		if item.SessionID == "" {
			continue
		}
		byID[item.SessionID] = item
	}
	for sessionID, ledger := range ledgers {
		sessionID = strings.TrimSpace(firstNonEmpty(sessionID, ledger.SessionID))
		if sessionID == "" {
			continue
		}
		if _, ok := byID[sessionID]; !ok {
			byID[sessionID] = finalizeMetadata(Metadata{SessionID: sessionID, CreatedAt: ledger.CreatedAtUnix, UpdatedAt: ledger.UpdatedAtUnix})
		}
	}

	children := make(map[string][]Metadata)
	for _, item := range byID {
		parent := strings.TrimSpace(item.ParentSessionID)
		children[parent] = append(children[parent], item)
	}
	for parent := range children {
		sortMetadataForTree(children[parent])
	}

	var entries []TreeEntry
	var walk func(Metadata, int)
	walk = func(item Metadata, depth int) {
		entry := treeEntryFromMetadata(item, ledgers[item.SessionID], opts.ActiveSessionID, depth, byID, filter)
		if includeTreeEntry(entry, filter) {
			entries = append(entries, entry)
		}
		for _, child := range children[item.SessionID] {
			walk(child, depth+1)
		}
	}

	roots := treeRoots(byID)
	for _, root := range roots {
		walk(root, 0)
	}
	return SessionTree{Filter: filter, ActiveSessionID: strings.TrimSpace(opts.ActiveSessionID), Entries: entries}
}

func treeRoots(byID map[string]Metadata) []Metadata {
	roots := make([]Metadata, 0, len(byID))
	for _, item := range byID {
		if item.ParentSessionID == "" {
			roots = append(roots, item)
			continue
		}
		if _, ok := byID[item.ParentSessionID]; !ok {
			roots = append(roots, item)
		}
	}
	sortMetadataForTree(roots)
	return roots
}

func treeEntryFromMetadata(item Metadata, ledger SessionLedger, activeID string, depth int, byID map[string]Metadata, filter TreeFilter) TreeEntry {
	item = finalizeMetadata(item)
	entry := TreeEntry{
		SessionID:       item.SessionID,
		ParentSessionID: item.ParentSessionID,
		LineageKind:     item.LineageKind,
		Title:           strings.TrimSpace(item.Title),
		Labels:          append([]string(nil), item.Labels...),
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		Depth:           depth,
		Active:          item.SessionID == strings.TrimSpace(activeID),
		Status:          lineageAuditStatus(item, byID),
	}
	if entry.Title == "" {
		entry.Title = sessionTreeTitleFromLedger(ledger, item.SessionID)
	}
	entry.Messages = treeMessagesFromLedger(ledger, filter)
	return entry
}

func treeMessagesFromLedger(ledger SessionLedger, filter TreeFilter) []TreeMessage {
	messages := make([]TreeMessage, 0, len(ledger.Messages))
	for _, msg := range ledger.Messages {
		role := strings.TrimSpace(msg.Role)
		switch filter {
		case TreeFilterUserOnly:
			if role != "user" {
				continue
			}
		case TreeFilterDefault, TreeFilterNoTools, TreeFilterLabeledOnly:
			if role == "tool" {
				continue
			}
		}
		messages = append(messages, TreeMessage{
			ID:                msg.ID,
			Role:              role,
			Content:           msg.Content,
			CreatedAtUnix:     msg.CreatedAtUnix,
			Replayable:        role == "user" && strings.TrimSpace(msg.Content) != "",
			ReplayUnavailable: replayUnavailableEvidence(role, msg.Content),
			TimestampKnown:    msg.CreatedAtKnown,
			TimestampEvidence: msg.TimestampEvidence,
		})
	}
	return messages
}

func includeTreeEntry(entry TreeEntry, filter TreeFilter) bool {
	if filter == TreeFilterLabeledOnly && len(entry.Labels) == 0 {
		return false
	}
	return true
}

func sessionTreeTitleFromLedger(ledger SessionLedger, fallback string) string {
	for _, msg := range ledger.Messages {
		if strings.TrimSpace(msg.Role) == "user" && strings.TrimSpace(msg.Content) != "" {
			return truncateTreeText(msg.Content, 48)
		}
	}
	return fallback
}

func ReplayPromptFromLedger(ledger SessionLedger, messageID int64) (string, string) {
	for _, msg := range ledger.Messages {
		if msg.ID != messageID {
			continue
		}
		role := strings.TrimSpace(msg.Role)
		if role != "user" {
			return "", ReplayUnavailableNonUserEntry
		}
		text := strings.TrimSpace(msg.Content)
		if text == "" {
			return "", ReplayUnavailableEmptyUserTurn
		}
		return text, ""
	}
	return "", ReplayUnavailableEntryMissing
}

func replayUnavailableEvidence(role, content string) string {
	if strings.TrimSpace(role) != "user" {
		return ReplayUnavailableNonUserEntry
	}
	if strings.TrimSpace(content) == "" {
		return ReplayUnavailableEmptyUserTurn
	}
	return ""
}

func normalizeTreeFilter(filter TreeFilter) TreeFilter {
	switch TreeFilter(strings.ToLower(strings.TrimSpace(string(filter)))) {
	case "", TreeFilterDefault:
		return TreeFilterDefault
	case TreeFilterNoTools:
		return TreeFilterNoTools
	case TreeFilterUserOnly:
		return TreeFilterUserOnly
	case TreeFilterLabeledOnly:
		return TreeFilterLabeledOnly
	case TreeFilterAllEquivalent, "all":
		return TreeFilterAllEquivalent
	default:
		return TreeFilterDefault
	}
}

func sortMetadataForTree(items []Metadata) {
	sort.Slice(items, func(i, j int) bool {
		if leftRank, rightRank := treeLineageRank(items[i]), treeLineageRank(items[j]); leftRank != rightRank {
			return leftRank < rightRank
		}
		leftTime := firstNonZero(items[i].CreatedAt, items[i].UpdatedAt)
		rightTime := firstNonZero(items[j].CreatedAt, items[j].UpdatedAt)
		if leftTime != rightTime {
			return leftTime < rightTime
		}
		return items[i].SessionID < items[j].SessionID
	})
}

func treeLineageRank(item Metadata) int {
	switch effectiveLineageKind(item) {
	case LineageKindPrimary:
		return 0
	case LineageKindCompression:
		return 1
	case LineageKindFork:
		return 2
	default:
		return 3
	}
}

func normalizeLabels(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

func truncateTreeText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
