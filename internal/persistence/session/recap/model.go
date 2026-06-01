// Package recap owns the pure session recap read models and formatting.
//
// It exposes value types and human-readable render methods. It must never know
// about Metadata stores, Bolt/SQL persistence, command parsing, or UI widgets.
package recap

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	MaxEntries int
}

type Entry struct {
	SessionID string
	Title     string
	Source    string
	UserID    string
	CreatedAt int64
	UpdatedAt int64
	TokensIn  int
	TokensOut int
}

type Envelope struct {
	Entries       []Entry
	TotalSessions int
	Truncated     bool
}

func (e *Envelope) HumanOutput() string {
	if len(e.Entries) == 0 {
		return "No sessions found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Sessions (%d total", e.TotalSessions))
	if e.Truncated {
		sb.WriteString(fmt.Sprintf(", showing %d", len(e.Entries)))
	}
	sb.WriteString("):\n\n")

	for _, entry := range e.Entries {
		title := entry.Title
		if title == "" {
			title = "(untitled)"
		}
		sb.WriteString(fmt.Sprintf("  %-20s  %s  [%s]\n", entry.SessionID, title, entry.Source))
		if entry.TokensIn > 0 || entry.TokensOut > 0 {
			sb.WriteString(fmt.Sprintf("    tokens: %d in / %d out\n", entry.TokensIn, entry.TokensOut))
		}
		if entry.UpdatedAt > 0 {
			sb.WriteString(fmt.Sprintf("    updated: %s\n", time.Unix(entry.UpdatedAt, 0).Format(time.RFC3339)))
		}
	}

	if e.Truncated {
		sb.WriteString(fmt.Sprintf("\n... %d more sessions not shown. Increase --limit to see more.\n", e.TotalSessions-len(e.Entries)))
	}

	return sb.String()
}

type SessionResult struct {
	SessionID string
	Title     string
	Source    string
	UserID    string
	CreatedAt int64
	UpdatedAt int64
	TokensIn  int
	TokensOut int
	NotFound  bool
}

func (r *SessionResult) HumanOutput() string {
	if r.NotFound {
		return fmt.Sprintf("Session %q not found.", r.SessionID)
	}

	var sb strings.Builder
	title := r.Title
	if title == "" {
		title = "(untitled)"
	}
	sb.WriteString(fmt.Sprintf("Session: %s\n", r.SessionID))
	sb.WriteString(fmt.Sprintf("Title:   %s\n", title))
	sb.WriteString(fmt.Sprintf("Source:  %s\n", r.Source))
	if r.UserID != "" {
		sb.WriteString(fmt.Sprintf("User:    %s\n", r.UserID))
	}
	if r.CreatedAt > 0 {
		sb.WriteString(fmt.Sprintf("Created: %s\n", time.Unix(r.CreatedAt, 0).Format(time.RFC3339)))
	}
	if r.UpdatedAt > 0 {
		sb.WriteString(fmt.Sprintf("Updated: %s\n", time.Unix(r.UpdatedAt, 0).Format(time.RFC3339)))
	}
	if r.TokensIn > 0 || r.TokensOut > 0 {
		sb.WriteString(fmt.Sprintf("Tokens:  %d in / %d out\n", r.TokensIn, r.TokensOut))
	}

	return sb.String()
}
