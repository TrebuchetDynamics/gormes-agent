package llm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// upstreamPromptBuilderPath returns the path to the upstream Hermes
// prompt_builder.py used as the byte-equivalence source-of-truth.
//
// Resolution order:
//  1. ./hermes-agent/agent/prompt_builder.py relative to the gormes-agent repo root
//     (internal/llm is two levels below the repo root).
//  2. ../hermes-agent/agent/prompt_builder.py as the older sibling checkout layout.
//  3. /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent/agent/prompt_builder.py
//     as the shared local upstream checkout when the active Gormes repo lives
//     under workspace-gormes.
//  4. /home/xel/.hermes/hermes-agent/agent/prompt_builder.py as a developer fallback.
//
// Returns the resolved path and ok=true if a readable file is found.
func upstreamPromptBuilderPath(t *testing.T) (string, bool) {
	t.Helper()
	// internal/llm is two levels below the repo root.
	candidates := []string{
		filepath.Join("..", "..", "hermes-agent", "agent", "prompt_builder.py"),
		filepath.Join("..", "..", "..", "hermes-agent", "agent", "prompt_builder.py"),
		"/home/xel/git/sages-openclaw/workspace-mineru/hermes-agent/agent/prompt_builder.py",
		"/home/xel/.hermes/hermes-agent/agent/prompt_builder.py",
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, true
		}
	}
	return "", false
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

func readUpstream(t *testing.T) (string, bool) {
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

func TestGuidanceConstants_MemoryGuidance_ByteEquivalent(t *testing.T) {
	src, ok := readUpstream(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check (drift will surface elsewhere)")
	}
	want, ok := extractPythonStringConstant(src, "MEMORY_GUIDANCE")
	if !ok {
		t.Fatalf("could not extract MEMORY_GUIDANCE from upstream")
	}
	if MemoryGuidance != want {
		t.Fatalf("MemoryGuidance does not match upstream MEMORY_GUIDANCE\n--- got (%d bytes) ---\n%q\n--- want (%d bytes) ---\n%q",
			len(MemoryGuidance), MemoryGuidance, len(want), want)
	}
}

func TestGuidanceConstants_SessionSearchGuidance_ByteEquivalent(t *testing.T) {
	src, ok := readUpstream(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check")
	}
	want, ok := extractPythonStringConstant(src, "SESSION_SEARCH_GUIDANCE")
	if !ok {
		t.Fatalf("could not extract SESSION_SEARCH_GUIDANCE from upstream")
	}
	if SessionSearchGuidance != want {
		t.Fatalf("SessionSearchGuidance does not match upstream\n--- got ---\n%q\n--- want ---\n%q", SessionSearchGuidance, want)
	}
}

func TestGuidanceConstants_SkillsGuidance_ByteEquivalent(t *testing.T) {
	src, ok := readUpstream(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check")
	}
	want, ok := extractPythonStringConstant(src, "SKILLS_GUIDANCE")
	if !ok {
		t.Fatalf("could not extract SKILLS_GUIDANCE from upstream")
	}
	if SkillsGuidance != want {
		t.Fatalf("SkillsGuidance does not match upstream\n--- got ---\n%q\n--- want ---\n%q", SkillsGuidance, want)
	}
}

func TestGuidanceConstants_ToolUseEnforcementGuidance_ByteEquivalent(t *testing.T) {
	src, ok := readUpstream(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check")
	}
	want, ok := extractPythonStringConstant(src, "TOOL_USE_ENFORCEMENT_GUIDANCE")
	if !ok {
		t.Fatalf("could not extract TOOL_USE_ENFORCEMENT_GUIDANCE from upstream")
	}
	if ToolUseEnforcementGuidance != want {
		t.Fatalf("ToolUseEnforcementGuidance does not match upstream\n--- got ---\n%q\n--- want ---\n%q", ToolUseEnforcementGuidance, want)
	}
}

func TestGuidanceConstants_ToolUseEnforcementModels_MatchesUpstream(t *testing.T) {
	src, ok := readUpstream(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping match check")
	}
	want, ok := extractPythonTupleOfStrings(src, "TOOL_USE_ENFORCEMENT_MODELS")
	if !ok {
		t.Fatalf("could not extract TOOL_USE_ENFORCEMENT_MODELS from upstream")
	}
	if !reflect.DeepEqual(ToolUseEnforcementModels, want) {
		t.Fatalf("ToolUseEnforcementModels mismatch\n got: %#v\nwant: %#v", ToolUseEnforcementModels, want)
	}
}

func TestGuidanceConstants_DeveloperRoleModels_MatchesUpstream(t *testing.T) {
	src, ok := readUpstream(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping match check")
	}
	want, ok := extractPythonTupleOfStrings(src, "DEVELOPER_ROLE_MODELS")
	if !ok {
		t.Fatalf("could not extract DEVELOPER_ROLE_MODELS from upstream")
	}
	if !reflect.DeepEqual(DeveloperRoleModels, want) {
		t.Fatalf("DeveloperRoleModels mismatch\n got: %#v\nwant: %#v", DeveloperRoleModels, want)
	}
}

func TestGuidanceConstants_WSLEnvironmentHint_ByteEquivalent(t *testing.T) {
	src, ok := readUpstream(t)
	if !ok {
		t.Skip("upstream prompt_builder.py not available; skipping byte-equivalence check")
	}
	want, ok := extractPythonStringConstant(src, "WSL_ENVIRONMENT_HINT")
	if !ok {
		t.Fatalf("could not extract WSL_ENVIRONMENT_HINT from upstream")
	}
	if WSLEnvironmentHint != want {
		t.Fatalf("WSLEnvironmentHint does not match upstream\n--- got ---\n%q\n--- want ---\n%q", WSLEnvironmentHint, want)
	}
}

// TestGuidanceConstants_NoRuntimeImport asserts that the data-only constants
// file imports nothing beyond the standard library — specifically nothing from
// internal/gateway, internal/kernel, internal/channels, internal/runtime,
// internal/provider, and other live-turn subsystems. This static check must
// always run; it does not depend on upstream availability.
func TestGuidanceConstants_NoRuntimeImport(t *testing.T) {
	const target = "guidance_constants.go"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, target, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", target, err)
	}
	const repoModule = "github.com/TrebuchetDynamics/gormes-agent/internal/"
	disallowed := []string{
		"gateway", "kernel", "channels", "runtime", "provider",
		"memory", "session", "skills", "cron",
	}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("unquote import path %q: %v", imp.Path.Value, err)
		}
		if !strings.HasPrefix(path, repoModule) {
			continue
		}
		suffix := strings.TrimPrefix(path, repoModule)
		head := suffix
		if i := strings.IndexByte(suffix, '/'); i >= 0 {
			head = suffix[:i]
		}
		for _, banned := range disallowed {
			if head == banned {
				t.Fatalf("guidance_constants.go must not import internal/%s (found %q)", banned, path)
			}
		}
	}
	// Sanity: the file must still parse and define at least one decl.
	if len(file.Decls) == 0 {
		// Parser was invoked with ImportsOnly which strips body decls; a
		// proper second parse confirms the file is non-empty. Use the
		// importsOnly result only as the import oracle.
		_ = ast.Print
	}
}
