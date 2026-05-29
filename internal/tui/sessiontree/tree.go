package sessiontree

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type SessionTreeRequest struct {
	Filter          string
	ActiveSessionID string
}

type SessionTreeResult struct {
	Filter          string
	ActiveSessionID string
	Entries         []SessionTreeEntry
}

type SessionTreeEntry struct {
	ID          string
	ParentID    string
	LineageKind string
	Title       string
	Labels      []string
	UpdatedAt   int64
	Depth       int
	Active      bool
	Status      string
	Messages    []SessionTreeMessage
}

type SessionTreeMessage struct {
	ID       int64
	Role     string
	Content  string
	Evidence string
}

type Page struct {
	Title string
	Body  string
}

func BuildPage(tree SessionTreeResult) (Page, bool) {
	if len(tree.Entries) == 0 {
		return Page{}, false
	}
	filter := NormalizeFilter(tree.Filter)
	blocks := make([]string, 0, len(tree.Entries))
	for _, entry := range tree.Entries {
		blocks = append(blocks, renderSessionTreeEntry(entry, filter))
	}
	title := "Session Tree"
	if filter != "" && filter != "default" {
		title += " — " + filter
	}
	return Page{Title: title, Body: strings.Join(blocks, "\n")}, true
}

func renderSessionTreeEntry(entry SessionTreeEntry, filter string) string {
	active := " "
	if entry.Active {
		active = "*"
	}
	kind := strings.TrimSpace(entry.LineageKind)
	if kind == "" {
		kind = "primary"
	}
	label := firstNonEmptyString(strings.TrimSpace(entry.Title), strings.TrimSpace(entry.ID), "(unknown session)")
	prefix := strings.Repeat("  ", maxInt(entry.Depth, 0))
	if entry.Depth > 0 {
		prefix += "↳ "
	}
	line := fmt.Sprintf("%s %s%s %s", active, prefix, kind, label)
	if id := strings.TrimSpace(entry.ID); id != "" {
		line += " [" + id + "]"
	}
	if len(entry.Labels) > 0 {
		line += " labels: " + strings.Join(entry.Labels, ", ")
	}
	if when := sessionDirectoryTimeLabel(entry.UpdatedAt); when != "" {
		line += " updated: " + when
	}
	if status := strings.TrimSpace(entry.Status); status != "" && status != "ok" {
		line += " status: " + status
	}
	messages := renderSessionTreeMessages(entry.Messages, filter)
	if messages == "" {
		return line
	}
	return line + "\n" + messages
}

func renderSessionTreeMessages(messages []SessionTreeMessage, filter string) string {
	var lines []string
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		switch filter {
		case "user-only":
			if role != "user" {
				continue
			}
		case "", "default", "no-tools", "labeled-only":
			if role == "tool" {
				continue
			}
		}
		content := truncateSessionTreeText(msg.Content, 72)
		if content == "" {
			content = strings.TrimSpace(msg.Evidence)
		}
		if content == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("    #%s %s: %s", strconv.FormatInt(msg.ID, 10), role, content))
	}
	return strings.Join(lines, "\n")
}

func NormalizeFilter(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "default":
		return "default"
	case "no-tools", "notools":
		return "no-tools"
	case "user-only", "users":
		return "user-only"
	case "labeled-only", "labels":
		return "labeled-only"
	case "all-equivalent", "all":
		return "all-equivalent"
	default:
		return "default"
	}
}

func truncateSessionTreeText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sessionDirectoryTimeLabel(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format("2006-01-02 15:04 UTC")
}
