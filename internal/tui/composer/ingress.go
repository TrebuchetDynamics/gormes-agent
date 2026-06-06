package composer

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

const (
	defaultLargePasteChars = 8000
	defaultLargePasteLines = 80
)

// ComposerDropOptions holds the injectable filesystem context for
// DetectComposerDroppedFile. Tests use HomeDir to keep "~" expansion away
// from the real operator home.
type ComposerDropOptions struct {
	HomeDir string
	Stat    func(string) (fs.FileInfo, error)
}

// ComposerDropResult is the native TUI equivalent of Hermes'
// _detect_file_drop result.
type ComposerDropResult struct {
	Matched   bool
	Path      string
	IsImage   bool
	Remainder string
	Evidence  string
}

// ComposerPasteOptions holds the injectable long-paste collapse seam.
type ComposerPasteOptions struct {
	Collapse        func(string) (string, error)
	LargePasteChars int
	LargePasteLines int
}

// ComposerPasteSnippet tracks the original paste content behind a readable
// composer token.
type ComposerPasteSnippet struct {
	Label string
	Text  string
	Path  string
}

// ComposerPasteResult is returned after classifying pasted text.
type ComposerPasteResult struct {
	InsertText string
	Snippet    *ComposerPasteSnippet
	Evidence   string
}

// ComposerCopyResult is the pure result behind /copy [number].
type ComposerCopyResult struct {
	OK             bool
	Text           string
	ResponseNumber int
	Evidence       string
}

// DetectComposerDroppedFile detects a pasted or dropped local file path
// without treating short slash commands like /help as files.
func DetectComposerDroppedFile(input string, opts ComposerDropOptions) ComposerDropResult {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || strings.Contains(trimmed, "\n") || !LooksLikeComposerDroppedPath(trimmed) {
		return ComposerDropResult{}
	}
	stat := opts.Stat
	if stat == nil {
		stat = os.Stat
	}

	for _, c := range composerDropCandidates(trimmed) {
		path, ok := normalizeComposerDropPath(c.raw, opts.HomeDir)
		if !ok {
			continue
		}
		info, err := stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		return ComposerDropResult{
			Matched:   true,
			Path:      path,
			IsImage:   IsComposerImagePath(path),
			Remainder: strings.TrimSpace(c.remainder),
		}
	}
	return ComposerDropResult{Evidence: "tui_ingress_file_missing"}
}

// LooksLikeComposerDroppedPath mirrors the current Hermes Ink client-side
// heuristic: enough to decide whether to ask the gateway to validate a path,
// not enough to claim a real file exists.
func LooksLikeComposerDroppedPath(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || strings.Contains(trimmed, "\n") {
		return false
	}
	if strings.HasPrefix(trimmed, "file://") ||
		strings.HasPrefix(trimmed, "~/") ||
		strings.HasPrefix(trimmed, "./") ||
		strings.HasPrefix(trimmed, "../") ||
		hasQuotedComposerPathPrefix(trimmed) ||
		hasWindowsDrivePrefix(trimmed) ||
		hasQuotedWindowsDrivePrefix(trimmed) {
		return true
	}
	if strings.HasPrefix(trimmed, "/") {
		rest := strings.TrimPrefix(trimmed, "/")
		return strings.Contains(rest, "/") || strings.Contains(rest, ".")
	}
	return false
}

// IsComposerImagePath reports whether path has a Hermes-supported image
// suffix. Header validation happens later at the gateway/image metadata seam.
func IsComposerImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".tif", ".svg", ".ico":
		return true
	default:
		return false
	}
}

// CollapseComposerPaste returns either the original small paste text or a
// readable placeholder plus a snippet that can be expanded before external
// editor submission.
func CollapseComposerPaste(text string, opts ComposerPasteOptions) ComposerPasteResult {
	cleaned := stripTrailingComposerPasteNewlines(text)
	if cleaned == "" || !strings.ContainsAny(cleaned, " \t\r\n") {
		return ComposerPasteResult{InsertText: cleaned}
	}
	lineCount := composerLineCount(cleaned)
	maxChars := opts.LargePasteChars
	if maxChars <= 0 {
		maxChars = defaultLargePasteChars
	}
	maxLines := opts.LargePasteLines
	if maxLines <= 0 {
		maxLines = defaultLargePasteLines
	}
	if len(cleaned) < maxChars && lineCount < maxLines {
		return ComposerPasteResult{InsertText: cleaned}
	}

	label := composerPasteTokenLabel(cleaned, lineCount)
	snippet := &ComposerPasteSnippet{Label: label, Text: cleaned}
	result := ComposerPasteResult{InsertText: label, Snippet: snippet}
	if opts.Collapse == nil {
		return result
	}
	path, err := opts.Collapse(cleaned)
	if err != nil {
		result.Evidence = "tui_ingress_paste_collapse_failed"
		return result
	}
	snippet.Path = path
	return result
}

// ExpandComposerPasteSnippets replaces visible paste tokens with their stored
// content before an injected external-editor buffer is opened.
func ExpandComposerPasteSnippets(input string, snippets []ComposerPasteSnippet, readFile func(string) ([]byte, error)) (string, error) {
	out := input
	for _, snip := range snippets {
		if snip.Label == "" || !strings.Contains(out, snip.Label) {
			continue
		}
		text := snip.Text
		if text == "" && snip.Path != "" && readFile != nil {
			body, err := readFile(snip.Path)
			if err != nil {
				return "", err
			}
			text = string(body)
		}
		out = strings.ReplaceAll(out, snip.Label, text)
	}
	return out, nil
}

// RecoverComposerBracketedPaste turns terminal bracketed-paste sequences back
// into normal text, including the unterminated form produced by broken
// terminals after a paste interruption.
func RecoverComposerBracketedPaste(input string) string {
	const begin = "\x1b[200~"
	const end = "\x1b[201~"
	out := strings.ReplaceAll(input, begin, "")
	out = strings.ReplaceAll(out, end, "")
	return out
}

// SelectComposerCopyText selects the latest or indexed assistant response and
// removes reasoning scratchpads before it reaches the clipboard seam.
func SelectComposerCopyText(history []llm.Message, arg string) ComposerCopyResult {
	assistant := make([]llm.Message, 0, len(history))
	for _, msg := range history {
		if msg.Role == "assistant" {
			assistant = append(assistant, msg)
		}
	}
	if len(assistant) == 0 {
		return ComposerCopyResult{Evidence: "tui_ingress_copy_empty"}
	}

	if strings.TrimSpace(arg) != "" {
		n, err := strconv.Atoi(strings.TrimSpace(arg))
		if err != nil || n < 1 || n > len(assistant) {
			return ComposerCopyResult{Evidence: "tui_ingress_copy_invalid_index"}
		}
		text := strings.TrimSpace(StripComposerReasoningBlocks(assistant[n-1].Content))
		if text == "" {
			return ComposerCopyResult{Evidence: "tui_ingress_copy_empty_response", ResponseNumber: n}
		}
		return ComposerCopyResult{OK: true, Text: text, ResponseNumber: n}
	}

	for i := len(assistant) - 1; i >= 0; i-- {
		text := strings.TrimSpace(StripComposerReasoningBlocks(assistant[i].Content))
		if text != "" {
			return ComposerCopyResult{OK: true, Text: text, ResponseNumber: i + 1}
		}
	}
	return ComposerCopyResult{Evidence: "tui_ingress_copy_empty"}
}

// StripComposerReasoningBlocks removes common reasoning/thinking XML blocks
// from assistant text before clipboard copy.
func StripComposerReasoningBlocks(text string) string {
	cleaned := text
	for _, tag := range []string{"reasoning_scratchpad", composerThinkTag(), composerThinkingTag(), "reasoning", "thought"} {
		cleaned = stripComposerTagBlock(cleaned, tag)
	}
	return cleaned
}

func composerThinkTag() string {
	return string([]byte{'t', 'h', 'i', 'n', 'k'})
}

func composerThinkingTag() string {
	return string([]byte{'t', 'h', 'i', 'n', 'k', 'i', 'n', 'g'})
}

func stripComposerTagBlock(text, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	cleaned := text
	for {
		lower := strings.ToLower(cleaned)
		start := strings.Index(lower, open)
		if start < 0 {
			break
		}
		afterOpen := start + len(open)
		relEnd := strings.Index(lower[afterOpen:], close)
		if relEnd < 0 {
			cleaned = cleaned[:start]
			break
		}
		end := afterOpen + relEnd + len(close)
		cleaned = cleaned[:start] + cleaned[end:]
	}
	for {
		lower := strings.ToLower(cleaned)
		idx := strings.Index(lower, close)
		if idx < 0 {
			break
		}
		cleaned = cleaned[:idx] + cleaned[idx+len(close):]
	}
	return cleaned
}

func stripTrailingComposerPasteNewlines(text string) string {
	if !strings.ContainsFunc(text, func(r rune) bool { return r != '\n' }) {
		return text
	}
	return strings.TrimRight(text, "\n")
}

func composerLineCount(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

func composerPasteTokenLabel(text string, lineCount int) string {
	preview := composerPasteTokenPreview(text)
	if preview == "" {
		return "[[ [" + intString(lineCount) + " lines] ]]"
	}
	return "[[ " + preview + " [" + intString(lineCount) + " lines] ]]"
}

func composerPasteTokenPreview(text string) string {
	preview := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	runes := []rune(preview)
	if len(runes) <= 48 {
		return preview
	}
	return strings.TrimSpace(string(runes[:24])) + ".. " + strings.TrimSpace(string(runes[len(runes)-18:]))
}

func intString(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

type composerDropCandidate struct {
	raw       string
	remainder string
}

func composerDropCandidates(input string) []composerDropCandidate {
	if q := input[0]; q == '"' || q == '\'' {
		if end := strings.IndexByte(input[1:], q); end >= 0 {
			pos := end + 1
			return []composerDropCandidate{{
				raw:       input[1:pos],
				remainder: strings.TrimSpace(input[pos+1:]),
			}}
		}
	}

	candidates := make([]composerDropCandidate, 0, composerDropBoundaryCount(input)+1)
	for end := len(input); end > 0; end-- {
		if end != len(input) && !isComposerDropRemainderBoundary(input[end]) {
			continue
		}
		raw := strings.TrimSpace(input[:end])
		if raw == "" {
			continue
		}
		candidates = append(candidates, composerDropCandidate{
			raw:       raw,
			remainder: strings.TrimSpace(input[end:]),
		})
	}
	return candidates
}

func composerDropBoundaryCount(input string) int {
	count := 0
	for i := 0; i < len(input); i++ {
		if isComposerDropRemainderBoundary(input[i]) {
			count++
		}
	}
	return count
}

func isComposerDropRemainderBoundary(b byte) bool {
	return b == ' ' || b == '\t'
}

func normalizeComposerDropPath(raw, home string) (string, bool) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", false
	}
	path = strings.ReplaceAll(path, `\ `, " ")

	if strings.HasPrefix(path, "file://") {
		u, err := url.Parse(path)
		if err != nil || (u.Host != "" && u.Host != "localhost") {
			return "", false
		}
		decoded, err := url.PathUnescape(u.EscapedPath())
		if err != nil {
			return "", false
		}
		path = decoded
	}

	if path == "~" {
		if home == "" {
			return "", false
		}
		path = home
	} else if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		if home == "" {
			return "", false
		}
		path = filepath.Join(home, path[2:])
	}

	return filepath.Clean(path), true
}

func hasQuotedComposerPathPrefix(s string) bool {
	if len(s) < 2 || (s[0] != '"' && s[0] != '\'') {
		return false
	}
	quoted := s[1:]
	return strings.HasPrefix(quoted, "/") ||
		strings.HasPrefix(quoted, "~/") ||
		strings.HasPrefix(quoted, `~\`) ||
		strings.HasPrefix(quoted, "./") ||
		strings.HasPrefix(quoted, "../")
}

func hasWindowsDrivePrefix(s string) bool {
	return len(s) >= 3 && isASCIIDriveLetter(s[0]) && s[1] == ':' && (s[2] == '/' || s[2] == '\\')
}

func hasQuotedWindowsDrivePrefix(s string) bool {
	return len(s) >= 4 && (s[0] == '"' || s[0] == '\'') && hasWindowsDrivePrefix(s[1:])
}

func isASCIIDriveLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
