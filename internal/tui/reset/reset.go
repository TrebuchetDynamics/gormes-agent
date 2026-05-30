package reset

import (
	"fmt"
	"strings"
)

const (
	KindClear = "clear"
	KindNew   = "new"
)

type Func func() error

type SlashResult struct {
	Status string
	Reset  bool
}

// HandleSlash resolves the /clear and /new command status contract while the
// root TUI model remains responsible for clearing its local frame fields after
// Reset=true.
func HandleSlash(input string, busy bool, reset Func) SlashResult {
	kind := Kind(input)
	if busy {
		return SlashResult{Status: "interrupt the current turn before trying to switch sessions"}
	}
	if reset == nil {
		return SlashResult{Status: kind + ": reset unavailable"}
	}
	if err := reset(); err != nil {
		return SlashResult{Status: fmt.Sprintf("%s: reset failed: %v", kind, err)}
	}
	if kind == KindNew {
		return SlashResult{Status: "new session started", Reset: true}
	}
	return SlashResult{Status: "session cleared", Reset: true}
}

// Kind resolves whether a session-reset slash invocation is a clear or new
// session request. Unknown or empty commands preserve the historical clear
// fallback used by the root TUI slash handler.
func Kind(input string) string {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return KindClear
	}
	switch strings.ToLower(strings.TrimPrefix(fields[0], "/")) {
	case KindNew:
		return KindNew
	default:
		return KindClear
	}
}
