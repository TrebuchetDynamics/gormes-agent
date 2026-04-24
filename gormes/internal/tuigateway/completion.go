package tuigateway

import (
	"os"
	"runtime"
	"strings"
	"unicode"
)

// NormalizeCompletionPath rewrites the raw path fragment the TUI ships to
// the `complete.path` JSON-RPC method into a filesystem-resolvable form.
// It mirrors `_normalize_completion_path` in `tui_gateway/server.py`
// (lines 221-229) branch-for-branch:
//
//   - `~` and `~/…` are expanded against the current user's home directory
//     (falling back to the input unchanged when `os.UserHomeDir` fails).
//   - On non-Windows runtimes, backslashes are rewritten to forward slashes
//     so operators who type Windows-style separators from the TUI still hit
//     the correct directory.
//   - A drive-letter path (`[A-Za-z]:/…`) is remapped to `/mnt/{lower}/…`
//     so WSL operators get the correct mount point for free.
//
// The function is pure: no filesystem reads beyond the home-directory
// lookup and no mutation of argument state.
func NormalizeCompletionPath(pathPart string) string {
	expanded := expandUserHome(pathPart)
	if runtime.GOOS == "windows" {
		return expanded
	}
	normalized := strings.ReplaceAll(expanded, "\\", "/")
	if len(normalized) >= 3 &&
		normalized[1] == ':' &&
		normalized[2] == '/' &&
		unicode.IsLetter(rune(normalized[0])) {
		drive := strings.ToLower(normalized[:1])
		return "/mnt/" + drive + "/" + normalized[3:]
	}
	return normalized
}

// expandUserHome mirrors the narrow `os.path.expanduser` surface the
// completion helper relies on: a bare `~`, or a `~/…` prefix, is replaced
// by the current user's home directory; anything else (including the
// `~username` form we do not support) is returned unchanged.
func expandUserHome(pathPart string) string {
	if pathPart == "" || pathPart[0] != '~' {
		return pathPart
	}
	if pathPart != "~" && !strings.HasPrefix(pathPart, "~/") {
		return pathPart
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return pathPart
	}
	if pathPart == "~" {
		return home
	}
	return home + pathPart[1:]
}
