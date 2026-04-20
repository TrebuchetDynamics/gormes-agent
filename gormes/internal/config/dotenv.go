package config

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// parseDotenv reads KEY=VALUE lines from r and returns the key→value
// map. Supports:
//   - Leading `export ` prefix (stripped)
//   - `#` comment lines and blank lines (skipped)
//   - Double-quoted values with `\n`, `\t`, `\"`, `\\` escape sequences
//   - Single-quoted values (literal, no escape expansion)
//   - Unquoted values with trailing whitespace stripped
//
// Malformed lines (no `=` or empty key) are silently skipped — a bad
// dotenv shouldn't prevent a config load. Returns error only on I/O
// failures from the Reader.
func parseDotenv(r io.Reader) (map[string]string, error) {
	out := make(map[string]string)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimLeft(sc.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "export\t") {
			line = strings.TrimLeft(line[len("export"):], " \t")
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			// No `=` or empty key — malformed, skip.
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := line[eq+1:]
		if key == "" {
			continue
		}
		out[key] = unquoteDotenvValue(val)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// unquoteDotenvValue handles the three quoting modes.
func unquoteDotenvValue(raw string) string {
	raw = strings.TrimLeft(raw, " \t")
	if raw == "" {
		return ""
	}
	switch raw[0] {
	case '"':
		// Find the matching closing quote, ignoring escaped `\"`.
		end := findClosingQuote(raw, 1, '"')
		if end < 0 {
			// Unterminated — treat as literal minus the opening quote.
			return raw[1:]
		}
		return expandDoubleQuotedEscapes(raw[1:end])
	case '\'':
		end := strings.IndexByte(raw[1:], '\'')
		if end < 0 {
			return raw[1:]
		}
		// Single quotes are literal — no escape expansion.
		return raw[1 : 1+end]
	default:
		// Unquoted: strip trailing whitespace. Do not treat `#` as a
		// comment-start here (dotenv convention: only leading-`#` is a
		// comment line; trailing `#` is literal).
		return strings.TrimRight(raw, " \t\r")
	}
}

func findClosingQuote(s string, start int, q byte) int {
	i := start
	for i < len(s) {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			// Skip escaped char (including escaped quote).
			i += 2
			continue
		}
		if c == q {
			return i
		}
		i++
	}
	return -1
}

func expandDoubleQuotedEscapes(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		switch s[i+1] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		default:
			// Unknown escape — preserve both chars literally.
			b.WriteByte(s[i])
			b.WriteByte(s[i+1])
		}
		i++ // consume the escaped char
	}
	return b.String()
}

// loadDotenvFiles reads Gormes-native then legacy Hermes dotenv files
// and populates os.Setenv for any key not already present in the
// ORIGINAL shell environment. Precedence: shell env > Gormes `.env` >
// Hermes `.env`. Silently no-ops when a file is missing.
//
// Implementation note: we snapshot the shell env ONCE before applying
// any files, then walk files low→high precedence. That way a later
// (higher-precedence) file can overwrite an earlier file's value for a
// key the shell didn't set, but no file can overwrite a shell value.
func loadDotenvFiles() {
	shellHasKey := snapshotShellEnv()
	for _, p := range dotenvCandidatePaths() {
		applyDotenvFile(p, shellHasKey)
	}
}

func snapshotShellEnv() map[string]struct{} {
	out := make(map[string]struct{}, len(os.Environ()))
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			out[kv[:i]] = struct{}{}
		}
	}
	return out
}

// dotenvCandidatePaths returns the list of dotenv files to load, in
// increasing precedence order (last write wins for unset keys).
func dotenvCandidatePaths() []string {
	var paths []string
	// Legacy Hermes first — lowest dotenv precedence.
	if hermesHome := os.Getenv("HERMES_HOME"); hermesHome != "" {
		paths = append(paths, filepath.Join(hermesHome, ".env"))
	} else {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			paths = append(paths, filepath.Join(home, ".hermes", ".env"))
		}
	}
	// Gormes-native last — wins within the dotenv layer.
	paths = append(paths, filepath.Join(xdgConfigHome(), "gormes", ".env"))
	return paths
}

// applyDotenvFile opens path, parses it, and sets env vars for keys
// not in the shell snapshot. Missing file = silent no-op. Parse errors
// = silent skip (a bad dotenv shouldn't block startup). A key already
// set by an earlier (lower-precedence) dotenv file CAN be overwritten
// because it's not in shellHasKey.
func applyDotenvFile(path string, shellHasKey map[string]struct{}) {
	f, err := os.Open(path)
	if err != nil {
		return // file missing / unreadable — silent
	}
	defer f.Close()

	pairs, err := parseDotenv(f)
	if err != nil {
		return
	}
	for k, v := range pairs {
		if _, fromShell := shellHasKey[k]; fromShell {
			continue // shell env wins
		}
		_ = os.Setenv(k, v)
	}
}
