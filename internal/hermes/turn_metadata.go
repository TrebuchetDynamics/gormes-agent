package hermes

import (
	"fmt"
	"strings"
	"time"
)

// turnMetadataTimeFormat mirrors Hermes' run_agent.py:3772 strftime layout
// `'%A, %B %d, %Y %I:%M %p'`. Go reference time `Mon Jan 2 15:04:05 MST 2006`
// becomes `Monday, January 2, 2006 03:04 PM` to produce identical output.
const turnMetadataTimeFormat = "Monday, January 2, 2006 03:04 PM"

// TurnMetadataOptions controls deterministic timestamp / session / model /
// provider rendering at the channel-neutral live-turn assembly site.
//
// Hermes parity reference (run_agent.py:3770-3779):
//
//	Conversation started: <Weekday>, <Month> <Day>, <Year> <HH:MM AM/PM>
//	Session ID: <id>
//	Model: <model>
//	Provider: <provider>
type TurnMetadataOptions struct {
	// Now is the clock used for the timestamp line. The supplied
	// time.Time's Location() is honored, so callers can render in UTC,
	// local, or any explicit zone. The zero time elides the timestamp
	// line; if SessionID/Model/Provider are also empty, the entire block
	// collapses to "" so the assembler can drop it.
	Now time.Time
	// SessionID, when non-empty, renders a `Session ID: <id>` line.
	SessionID string
	// Model, when non-empty, renders a `Model: <model>` line.
	Model string
	// Provider, when non-empty, renders a `Provider: <provider>` line.
	Provider string
}

// BuildTurnMetadataBlock renders the Hermes-compatible metadata block. Lines
// for empty fields are omitted, and a zero Now combined with empty
// SessionID/Model/Provider returns "" so callers can elide the block.
func BuildTurnMetadataBlock(opts TurnMetadataOptions) string {
	if opts.Now.IsZero() && opts.SessionID == "" && opts.Model == "" && opts.Provider == "" {
		return ""
	}
	var sb strings.Builder
	if !opts.Now.IsZero() {
		fmt.Fprintf(&sb, "Conversation started: %s", opts.Now.Format(turnMetadataTimeFormat))
	}
	if opts.SessionID != "" {
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "Session ID: %s", opts.SessionID)
	}
	if opts.Model != "" {
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "Model: %s", opts.Model)
	}
	if opts.Provider != "" {
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "Provider: %s", opts.Provider)
	}
	return sb.String()
}
