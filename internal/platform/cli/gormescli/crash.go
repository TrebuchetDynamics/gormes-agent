package gormescli

import (
	"fmt"
	"strings"
)

// CrashStderrMessage formats the user-facing panic message printed to stderr
// after the caller writes the full crash log file. It surfaces the panic excerpt
// inline so operators can diagnose one-line panics without opening the log; the
// path is always appended for cases where the excerpt alone is insufficient.
//
// The excerpt is truncated to the first line and clamped to 200 runes to keep
// the stderr message readable when a panic carries a multi-line stack body or a
// very long detail string.
func CrashStderrMessage(panicValue any, logPath string) string {
	excerpt := fmt.Sprintf("%v", panicValue)
	if i := strings.IndexAny(excerpt, "\r\n"); i >= 0 {
		excerpt = excerpt[:i]
	}
	const maxRunes = 200
	if len([]rune(excerpt)) > maxRunes {
		excerpt = string([]rune(excerpt)[:maxRunes]) + "…"
	}
	if excerpt == "" {
		return "gormes crashed — log at " + logPath
	}
	return "gormes crashed: " + excerpt + "\nfull log: " + logPath
}
