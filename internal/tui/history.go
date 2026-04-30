package tui

// HermesHistory is the in-memory draft history that backs ResolveHermesKey's
// HermesActionHistoryPrev / HermesActionHistoryNext decisions. It mirrors the
// prompt_toolkit InMemoryHistory used by Hermes: every successful submit is
// appended; Up walks backward through entries, Down walks forward and returns
// to a fresh empty draft when it walks past the newest entry. Empty drafts
// are never stored, matching upstream's `if text:` guard.
//
// HermesHistory is not safe for concurrent use; the Bubble Tea Update loop is
// the only writer and reader.
type HermesHistory struct {
	entries []string
	// pos is the navigation cursor. -1 means "fresh draft below the newest
	// entry"; values in [0, len(entries)) point at a stored entry.
	pos int
}

// NewHermesHistory returns an empty history with the navigation cursor parked
// at the fresh-draft slot (pos == -1).
func NewHermesHistory() *HermesHistory {
	return &HermesHistory{pos: -1}
}

// Append records a submitted draft and resets the navigation cursor to the
// fresh-draft slot so the next Prev returns the just-appended entry. Empty
// drafts are ignored.
func (h *HermesHistory) Append(text string) {
	if text == "" {
		return
	}
	h.entries = append(h.entries, text)
	h.pos = -1
}

// Prev walks one step backward and returns the entry now under the cursor.
// On the first call after Append, Prev returns the newest entry. Once the
// cursor has reached the oldest entry, subsequent Prev calls leave the
// cursor in place and return the same entry; ok stays true to communicate
// that the draft is still under history control.
func (h *HermesHistory) Prev() (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}
	if h.pos == -1 {
		h.pos = len(h.entries) - 1
		return h.entries[h.pos], true
	}
	if h.pos > 0 {
		h.pos--
	}
	return h.entries[h.pos], true
}

// Next walks one step forward and returns the entry now under the cursor.
// Walking past the newest entry parks the cursor on the fresh-draft slot
// and returns ("", true) so the editor restores an empty draft.
func (h *HermesHistory) Next() (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}
	if h.pos == -1 {
		return "", false
	}
	if h.pos < len(h.entries)-1 {
		h.pos++
		return h.entries[h.pos], true
	}
	h.pos = -1
	return "", true
}
