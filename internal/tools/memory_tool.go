package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	MemoryToolName = "memory"

	MemoryEvidenceInvalidArgs      = "memory_invalid_args"
	MemoryEvidenceStoreUnavailable = "memory_store_unavailable"
	MemoryEvidenceUnsafeContent    = "memory_unsafe_content"
	MemoryEvidenceEntryNotFound    = "memory_entry_not_found"
	MemoryEvidenceAmbiguousMatch   = "memory_ambiguous_match"
	MemoryEvidenceLimitExceeded    = "memory_limit_exceeded"
)

const (
	memoryEntryDelimiter   = "\n\n§\n\n"
	memoryDefaultCharLimit = 20000
	userDefaultCharLimit   = 12000
)

type MemoryToolConfig struct {
	MemoryDir       string
	MemoryCharLimit int
	UserCharLimit   int
}

type MemoryTool struct {
	cfg MemoryToolConfig
	mu  sync.Mutex
}

type MemoryToolResult struct {
	Success    bool     `json:"success"`
	Target     string   `json:"target,omitempty"`
	Entries    []string `json:"entries,omitempty"`
	Usage      string   `json:"usage,omitempty"`
	EntryCount int      `json:"entry_count"`
	Message    string   `json:"message,omitempty"`
	Evidence   string   `json:"evidence,omitempty"`
	Error      string   `json:"error,omitempty"`
	Matches    []string `json:"matches,omitempty"`
}

type memoryToolArgs struct {
	Action     string `json:"action"`
	Target     string `json:"target"`
	Content    string `json:"content"`
	OldText    string `json:"old_text"`
	NewContent string `json:"new_content"`
}

func NewMemoryTool(cfg MemoryToolConfig) *MemoryTool { return &MemoryTool{cfg: cfg} }

func (*MemoryTool) Name() string { return MemoryToolName }

func (*MemoryTool) Description() string {
	return "Save durable memories about the user or environment. Supports add, replace, and remove on user or memory targets."
}

func (*MemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["add","replace","remove"],"description":"Memory mutation to perform."},"target":{"type":"string","enum":["user","memory"],"description":"user stores facts about Juan; memory stores assistant/environment notes."},"content":{"type":"string","description":"Content for add."},"old_text":{"type":"string","description":"Substring identifying the entry to replace or remove."},"new_content":{"type":"string","description":"Replacement content for replace."}},"required":["action","target"]}`)
}

func (*MemoryTool) Timeout() time.Duration { return 0 }

func (t *MemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	if t == nil {
		return json.Marshal(memoryError(MemoryEvidenceStoreUnavailable, "memory tool is not initialized"))
	}
	if len(strings.TrimSpace(string(args))) == 0 {
		args = json.RawMessage(`{}`)
	}
	var in memoryToolArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return json.Marshal(memoryError(MemoryEvidenceInvalidArgs, "invalid memory args: "+err.Error()))
	}
	result := t.execute(in)
	return json.Marshal(result)
}

func (t *MemoryTool) execute(in memoryToolArgs) MemoryToolResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	target, ok := normalizeMemoryTarget(in.Target)
	if !ok {
		return memoryError(MemoryEvidenceInvalidArgs, "target must be user or memory")
	}
	path, err := t.pathFor(target)
	if err != nil {
		return memoryError(MemoryEvidenceStoreUnavailable, err.Error())
	}

	entries, err := readMemoryEntries(path)
	if err != nil {
		return memoryError(MemoryEvidenceStoreUnavailable, "read durable memory store")
	}

	switch strings.ToLower(strings.TrimSpace(in.Action)) {
	case "add":
		return t.add(path, target, entries, in.Content)
	case "replace":
		return t.replace(path, target, entries, in.OldText, in.NewContent)
	case "remove":
		return t.remove(path, target, entries, in.OldText)
	default:
		return memoryError(MemoryEvidenceInvalidArgs, "action must be add, replace, or remove")
	}
}

func (t *MemoryTool) add(path, target string, entries []string, content string) MemoryToolResult {
	content = strings.TrimSpace(content)
	if content == "" {
		return memoryError(MemoryEvidenceInvalidArgs, "content cannot be empty")
	}
	if evidence := scanMemoryContent(content); evidence != "" {
		return memoryError(evidence, "memory content rejected by safety scan")
	}
	for _, entry := range entries {
		if entry == content {
			return memorySuccess(target, entries, t.limitFor(target), "Entry already exists (no duplicate added).")
		}
	}
	updated := append(append([]string(nil), entries...), content)
	if overLimit(updated, t.limitFor(target)) {
		return memoryError(MemoryEvidenceLimitExceeded, "memory target would exceed character limit")
	}
	if err := writeMemoryEntries(path, updated); err != nil {
		return memoryError(MemoryEvidenceStoreUnavailable, "write durable memory store")
	}
	return memorySuccess(target, updated, t.limitFor(target), "Entry added.")
}

func (t *MemoryTool) replace(path, target string, entries []string, oldText, newContent string) MemoryToolResult {
	oldText = strings.TrimSpace(oldText)
	newContent = strings.TrimSpace(newContent)
	if oldText == "" {
		return memoryError(MemoryEvidenceInvalidArgs, "old_text cannot be empty")
	}
	if newContent == "" {
		return memoryError(MemoryEvidenceInvalidArgs, "new_content cannot be empty; use remove to delete entries")
	}
	if evidence := scanMemoryContent(newContent); evidence != "" {
		return memoryError(evidence, "memory content rejected by safety scan")
	}
	idx, matches, evidence := findMemoryEntry(entries, oldText)
	if evidence != "" {
		res := memoryError(evidence, memoryMatchError(evidence, oldText))
		res.Matches = matches
		return res
	}
	updated := append([]string(nil), entries...)
	updated[idx] = newContent
	if overLimit(updated, t.limitFor(target)) {
		return memoryError(MemoryEvidenceLimitExceeded, "replacement would exceed character limit")
	}
	if err := writeMemoryEntries(path, updated); err != nil {
		return memoryError(MemoryEvidenceStoreUnavailable, "write durable memory store")
	}
	return memorySuccess(target, updated, t.limitFor(target), "Entry replaced.")
}

func (t *MemoryTool) remove(path, target string, entries []string, oldText string) MemoryToolResult {
	oldText = strings.TrimSpace(oldText)
	if oldText == "" {
		return memoryError(MemoryEvidenceInvalidArgs, "old_text cannot be empty")
	}
	idx, matches, evidence := findMemoryEntry(entries, oldText)
	if evidence != "" {
		res := memoryError(evidence, memoryMatchError(evidence, oldText))
		res.Matches = matches
		return res
	}
	updated := append([]string(nil), entries[:idx]...)
	updated = append(updated, entries[idx+1:]...)
	if err := writeMemoryEntries(path, updated); err != nil {
		return memoryError(MemoryEvidenceStoreUnavailable, "write durable memory store")
	}
	return memorySuccess(target, updated, t.limitFor(target), "Entry removed.")
}

func (t *MemoryTool) pathFor(target string) (string, error) {
	dir := strings.TrimSpace(t.cfg.MemoryDir)
	if dir == "" {
		return "", errors.New("memory store is not initialized")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve memory store: %w", err)
	}
	name := "MEMORY.md"
	if target == "user" {
		name = "USER.md"
	}
	return filepath.Join(abs, name), nil
}

func (t *MemoryTool) limitFor(target string) int {
	if target == "user" {
		if t.cfg.UserCharLimit > 0 {
			return t.cfg.UserCharLimit
		}
		return userDefaultCharLimit
	}
	if t.cfg.MemoryCharLimit > 0 {
		return t.cfg.MemoryCharLimit
	}
	return memoryDefaultCharLimit
}

func normalizeMemoryTarget(target string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "user":
		return "user", true
	case "memory":
		return "memory", true
	default:
		return "", false
	}
}

func readMemoryEntries(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(string(raw))
	if body == "" {
		return nil, nil
	}
	parts := strings.Split(body, memoryEntryDelimiter)
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		entry := strings.TrimSpace(part)
		if entry != "" {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func writeMemoryEntries(path string, entries []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	body := strings.Join(entries, memoryEntryDelimiter)
	if len(entries) > 0 {
		body += "\n"
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".memory-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.WriteString(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func findMemoryEntry(entries []string, oldText string) (int, []string, string) {
	matches := make([]int, 0, 1)
	previews := make([]string, 0, 1)
	unique := map[string]bool{}
	for i, entry := range entries {
		if strings.Contains(entry, oldText) {
			matches = append(matches, i)
			previews = append(previews, memoryPreview(entry))
			unique[entry] = true
		}
	}
	if len(matches) == 0 {
		return -1, nil, MemoryEvidenceEntryNotFound
	}
	if len(matches) > 1 && len(unique) > 1 {
		return -1, previews, MemoryEvidenceAmbiguousMatch
	}
	return matches[0], nil, ""
}

func memoryMatchError(evidence, oldText string) string {
	if evidence == MemoryEvidenceAmbiguousMatch {
		return "multiple memory entries matched old_text; be more specific"
	}
	return "no memory entry matched old_text"
}

func memoryPreview(entry string) string {
	runes := []rune(strings.TrimSpace(entry))
	if len(runes) <= 80 {
		return string(runes)
	}
	return string(runes[:80]) + "..."
}

func overLimit(entries []string, limit int) bool {
	return utf8.RuneCountInString(strings.Join(entries, memoryEntryDelimiter)) > limit
}

func memorySuccess(target string, entries []string, limit int, message string) MemoryToolResult {
	entries = append([]string(nil), entries...)
	chars := utf8.RuneCountInString(strings.Join(entries, memoryEntryDelimiter))
	pct := 0
	if limit > 0 {
		pct = min(100, int(float64(chars)/float64(limit)*100))
	}
	return MemoryToolResult{
		Success:    true,
		Target:     target,
		Entries:    entries,
		Usage:      fmt.Sprintf("%d%% — %d/%d chars", pct, chars, limit),
		EntryCount: len(entries),
		Message:    message,
	}
}

func memoryError(evidence string, message string) MemoryToolResult {
	return MemoryToolResult{Success: false, Entries: []string{}, Evidence: evidence, Error: message}
}

var memoryThreatPatterns = []struct {
	re *regexp.Regexp
	id string
}{
	{regexp.MustCompile(`(?i)ignore\s+(previous|all|above|prior)\s+instructions`), "prompt_injection"},
	{regexp.MustCompile(`(?i)do\s+not\s+tell\s+the\s+user`), "deception_hide"},
	{regexp.MustCompile(`(?i)system\s+prompt\s+override`), "sys_prompt_override"},
	{regexp.MustCompile(`(?i)disregard\s+(your|all|any)\s+(instructions|rules|guidelines)`), "disregard_rules"},
	{regexp.MustCompile(`(?i)curl\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`), "exfil_curl"},
	{regexp.MustCompile(`(?i)cat\s+[^\n]*(\.env|credentials|\.netrc|\.pgpass)`), "read_secrets"},
}

var memoryInvisibleChars = []rune{'\u200b', '\u200c', '\u200d', '\u2060', '\ufeff', '\u202a', '\u202b', '\u202c', '\u202d', '\u202e'}

func scanMemoryContent(content string) string {
	for _, char := range memoryInvisibleChars {
		if strings.ContainsRune(content, char) {
			return MemoryEvidenceUnsafeContent
		}
	}
	for _, pattern := range memoryThreatPatterns {
		if pattern.re.MatchString(content) {
			return MemoryEvidenceUnsafeContent
		}
	}
	return ""
}
