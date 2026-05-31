package guidance

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func upstreamHermesFilePath(t *testing.T, repoRelativePath string) (string, bool) {
	t.Helper()
	candidates := []string{
		filepath.Join("..", "..", "..", "hermes-agent", repoRelativePath),
		filepath.Join("..", "..", "..", "..", "hermes-agent", repoRelativePath),
		filepath.Join("/home/xel/git/sages-openclaw/workspace-mineru/hermes-agent", repoRelativePath),
		filepath.Join("/home/xel/.hermes/hermes-agent", repoRelativePath),
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

// upstreamPromptBuilderPath returns the path to the upstream Hermes
// prompt_builder.py used as the byte-equivalence source-of-truth.
func upstreamPromptBuilderPath(t *testing.T) (string, bool) {
	t.Helper()
	return upstreamHermesFilePath(t, filepath.Join("agent", "prompt_builder.py"))
}

func upstreamDefaultSoulPath(t *testing.T) (string, bool) {
	t.Helper()
	return upstreamHermesFilePath(t, filepath.Join("hermes_cli", "default_soul.py"))
}

// extractPythonStringConstant locates `NAME = ( "..." "..." ... )` (or
// `NAME = "..."`) in src and returns the concatenated Python string literal
// value. Adjacent string literals separated only by whitespace inside the
// parentheses are concatenated, mirroring CPython's literal-concatenation
// behavior. Returns ok=false if the constant cannot be parsed.
func extractPythonStringConstant(src, name string) (string, bool) {
	idx := indexAssignmentStart(src, name)
	if idx < 0 {
		return "", false
	}
	// Move to the start of the value expression.
	rest := src[idx:]
	// Determine whether the value is a parenthesised concatenation or a
	// single string literal.
	rest = skipPythonInlineWhitespace(rest)
	var body string
	if strings.HasPrefix(rest, "(") {
		// Find matching closing paren, ignoring parens inside string literals.
		end := findMatchingParen(rest)
		if end < 0 {
			return "", false
		}
		body = rest[1:end]
	} else {
		// Single-line single literal: read until newline.
		nl := strings.IndexByte(rest, '\n')
		if nl < 0 {
			body = rest
		} else {
			body = rest[:nl]
		}
	}
	parts, ok := parsePythonAdjacentStrings(body)
	if !ok {
		return "", false
	}
	return strings.Join(parts, ""), true
}

// extractPythonTupleOfStrings parses `NAME = ("a", "b", ...)` into a []string
// preserving the upstream tuple order. Returns ok=false if NAME is missing or
// not a tuple of string literals.
func extractPythonTupleOfStrings(src, name string) ([]string, bool) {
	idx := indexAssignmentStart(src, name)
	if idx < 0 {
		return nil, false
	}
	rest := skipPythonInlineWhitespace(src[idx:])
	if !strings.HasPrefix(rest, "(") {
		return nil, false
	}
	end := findMatchingParen(rest)
	if end < 0 {
		return nil, false
	}
	body := rest[1:end]
	// Split by top-level commas. Since the body is only string literals and
	// commas, a simple state-machine string scanner suffices.
	var (
		parts   []string
		current strings.Builder
		inStr   bool
		quote   byte
		esc     bool
	)
	flush := func() {
		piece := strings.TrimSpace(current.String())
		current.Reset()
		if piece == "" {
			return
		}
		// piece is a single string literal like "abc" or 'abc'.
		val, err := strconv.Unquote(normalizeQuotes(piece))
		if err == nil {
			parts = append(parts, val)
		}
	}
	for i := 0; i < len(body); i++ {
		b := body[i]
		if inStr {
			current.WriteByte(b)
			if esc {
				esc = false
				continue
			}
			if b == '\\' {
				esc = true
				continue
			}
			if b == quote {
				inStr = false
			}
			continue
		}
		switch b {
		case '"', '\'':
			inStr = true
			quote = b
			current.WriteByte(b)
		case ',':
			flush()
		default:
			current.WriteByte(b)
		}
	}
	flush()
	return parts, true
}

// indexAssignmentStart returns the byte offset in src immediately after the
// `=` of `NAME = ` at column 0 (module-level assignment). Returns -1 if the
// constant is absent.
func indexAssignmentStart(src, name string) int {
	// Match a line that starts with NAME followed by spaces and `=`.
	needle := "\n" + name + " ="
	idx := strings.Index(src, needle)
	if idx < 0 {
		// First-line case (rare): src may begin with NAME directly.
		if strings.HasPrefix(src, name+" =") {
			return len(name + " =")
		}
		return -1
	}
	return idx + len(needle)
}

// skipPythonInlineWhitespace skips spaces, tabs, and newlines from the start
// of s and returns the remainder.
func skipPythonInlineWhitespace(s string) string {
	for len(s) > 0 {
		switch s[0] {
		case ' ', '\t', '\n', '\r':
			s = s[1:]
		default:
			return s
		}
	}
	return s
}

// findMatchingParen returns the byte offset of the `)` that matches the `(`
// at s[0], ignoring parens that appear inside string literals. Returns -1 if
// no match is found.
func findMatchingParen(s string) int {
	if len(s) == 0 || s[0] != '(' {
		return -1
	}
	depth := 0
	var (
		inStr bool
		quote byte
		esc   bool
	)
	for i := 0; i < len(s); i++ {
		b := s[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if b == '\\' {
				esc = true
				continue
			}
			if b == quote {
				inStr = false
			}
			continue
		}
		switch b {
		case '"', '\'':
			inStr = true
			quote = b
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		case '#':
			// Skip Python line comments outside strings.
			nl := strings.IndexByte(s[i:], '\n')
			if nl < 0 {
				return -1
			}
			i += nl
		}
	}
	return -1
}

// parsePythonAdjacentStrings collects every string literal in body, in order,
// returning their decoded values. Whitespace and comments between literals
// are ignored. Concatenation of these values reproduces CPython's adjacent
// string literal joining.
func parsePythonAdjacentStrings(body string) ([]string, bool) {
	var parts []string
	i := 0
	for i < len(body) {
		b := body[i]
		switch {
		case b == ' ' || b == '\t' || b == '\n' || b == '\r':
			i++
		case b == '#':
			nl := strings.IndexByte(body[i:], '\n')
			if nl < 0 {
				return parts, true
			}
			i += nl
		case b == '"' || b == '\'':
			lit, end, ok := readPythonStringLiteral(body, i)
			if !ok {
				return nil, false
			}
			val, err := strconv.Unquote(normalizeQuotes(lit))
			if err != nil {
				return nil, false
			}
			parts = append(parts, val)
			i = end
		default:
			// Unexpected token (e.g. variable reference); bail.
			return nil, false
		}
	}
	return parts, true
}

// readPythonStringLiteral reads one Python string literal starting at body[i]
// and returns the literal text (including surrounding quotes) and the byte
// offset immediately after it. Triple-quoted literals are NOT used by the
// constants we extract here, so they are intentionally unsupported.
func readPythonStringLiteral(body string, i int) (string, int, bool) {
	if i >= len(body) {
		return "", i, false
	}
	quote := body[i]
	if quote != '"' && quote != '\'' {
		return "", i, false
	}
	start := i
	i++
	for i < len(body) {
		b := body[i]
		if b == '\\' {
			i += 2
			continue
		}
		if b == quote {
			return body[start : i+1], i + 1, true
		}
		i++
	}
	return "", i, false
}

// normalizeQuotes converts a Python single-quoted string literal into a
// Go-compatible double-quoted form so strconv.Unquote can decode it. Python
// string-literal escapes are a strict subset of Go's for the constants we
// port, so byte-level translation suffices.
func normalizeQuotes(lit string) string {
	if len(lit) < 2 {
		return lit
	}
	if lit[0] == '"' {
		return lit
	}
	// Convert outer single quotes to double quotes, escaping any embedded
	// double quotes and unescaping any escaped single quotes.
	inner := lit[1 : len(lit)-1]
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) {
			next := inner[i+1]
			if next == '\'' {
				b.WriteByte('\'')
				i++
				continue
			}
			b.WriteByte(c)
			b.WriteByte(next)
			i++
			continue
		}
		if c == '"' {
			b.WriteString(`\"`)
			continue
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String()
}

func readUpstreamPromptBuilder(t *testing.T) (string, bool) {
	t.Helper()
	p, ok := upstreamPromptBuilderPath(t)
	if !ok {
		return "", false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Logf("upstream prompt_builder.py at %s unreadable: %v", p, err)
		return "", false
	}
	return string(data), true
}
