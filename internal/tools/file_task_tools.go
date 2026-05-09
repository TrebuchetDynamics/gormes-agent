package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const (
	defaultReadFileLimit     = 500
	defaultReadFileMaxLimit  = 2000
	defaultReadFileMaxChars  = 100000
	defaultSearchFilesLimit  = 50
	defaultSearchFilesMax    = 500
	defaultWriteFileFileMode = 0o644
	maxStructuredLintOutput  = 2000
)

// FileTaskToolConfig constrains local file tools to a workspace root.
type FileTaskToolConfig struct {
	Root          string
	ReadGuard     *FileReadGuard
	StateRegistry *FileStateRegistry
	TaskID        string
	CWDResolver   func() string
	MaxReadChars  int
}

type structuredLintResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Message string `json:"message,omitempty"`
}

type ReadFileToolConfig = FileTaskToolConfig

// ReadFileTool implements the Hermes read_file contract for local text files.
type ReadFileTool struct {
	cfg   FileTaskToolConfig
	guard *FileReadGuard
	mu    sync.Mutex
	seen  map[string]struct{}
}

func NewReadFileTool(cfg FileTaskToolConfig) *ReadFileTool {
	guard := cfg.ReadGuard
	if guard == nil {
		guard = NewFileReadGuard(FileReadGuardOptions{WorkspaceRoot: cfg.Root})
	}
	return &ReadFileTool{cfg: cfg, guard: guard, seen: make(map[string]struct{})}
}

func (*ReadFileTool) Name() string { return "read_file" }

func (*ReadFileTool) Description() string {
	return "Read a text file with line numbers and pagination. Use this instead of cat/head/tail in terminal."
}

func (*ReadFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Path to the file to read, relative to the workspace root or absolute inside it."},"offset":{"type":"integer","description":"Line number to start reading from, 1-indexed.","default":1,"minimum":1},"limit":{"type":"integer","description":"Maximum number of lines to read.","default":500,"maximum":2000}},"required":["path"]}`)
}

func (*ReadFileTool) Timeout() time.Duration { return 5 * time.Second }

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	var in struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(defaultJSONArgs(args), &in); err != nil {
		return marshalToolPayload(map[string]any{"error": "invalid read_file args: " + err.Error()})
	}
	resolved, rel, root, cwd, err := resolveFileTaskPath(t.cfg, in.Path)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": in.Path, "error": err.Error()})
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "read file: " + err.Error()})
	}
	if info.IsDir() {
		return marshalToolPayload(map[string]any{"path": rel, "error": "read_file expected a file, got a directory"})
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "read file: " + err.Error()})
	}
	if binaryLike(raw) {
		return marshalToolPayload(map[string]any{"path": rel, "error": "read_file cannot read binary files"})
	}

	lines := splitTextLines(string(raw))
	total := len(lines)
	offset := in.Offset
	if offset < 1 {
		offset = 1
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultReadFileLimit
	}
	if limit > defaultReadFileMaxLimit {
		limit = defaultReadFileMaxLimit
	}

	startIdx := offset - 1
	if startIdx > total {
		startIdx = total
	}
	endIdx := startIdx + limit
	if endIdx > total {
		endIdx = total
	}

	var b strings.Builder
	for i := startIdx; i < endIdx; i++ {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%6d|%s", i+1, lines[i])
	}
	if maxChars := t.maxReadChars(); b.Len() > maxChars {
		return marshalToolPayload(map[string]any{
			"path":        rel,
			"total_lines": total,
			"file_size":   len(raw),
			"error":       fmt.Sprintf("Read produced %d characters which exceeds the safety limit (%d chars). Use offset and limit to read a smaller range.", b.Len(), maxChars),
		})
	}
	if t.duplicateWindow(rel, offset, limit, info) {
		state, _ := t.fileStateRegistry().record(root, t.cfg.TaskID, cwd, rel, resolved)
		status := FileReadResult{
			Path:        rel,
			DedupStatus: FileReadDedupStatusUnchanged,
			Evidence: []FileReadEvidence{{
				Kind:    FileReadDedupStatusUnchanged,
				Path:    rel,
				Message: fileReadDedupStatusMessage,
			}},
		}
		if t.guard != nil {
			status = t.guard.GuardRepeatedReadStatus(status)
		}
		return marshalReadFileStatus(rel, status, state)
	}

	state, _ := t.fileStateRegistry().record(root, t.cfg.TaskID, cwd, rel, resolved)
	truncated := offset > 1 || endIdx < total
	payload := map[string]any{
		"path":             rel,
		"offset":           offset,
		"limit":            limit,
		"start_line":       startIdx + 1,
		"end_line":         endIdx,
		"total_lines":      total,
		"content":          b.String(),
		"content_returned": true,
		"truncated":        truncated,
	}
	if statePayload := fileStatePayload(state); statePayload != nil {
		payload["file_state"] = statePayload
	}
	if endIdx < total {
		payload["hint"] = fmt.Sprintf("Continue with offset=%d to read the next window.", endIdx+1)
	}
	return marshalToolPayload(payload)
}

func (t *ReadFileTool) fileStateRegistry() *FileStateRegistry {
	if t != nil && t.cfg.StateRegistry != nil {
		return t.cfg.StateRegistry
	}
	return defaultFileStateRegistry
}

func (t *ReadFileTool) maxReadChars() int {
	if t != nil && t.cfg.MaxReadChars > 0 {
		return t.cfg.MaxReadChars
	}
	return defaultReadFileMaxChars
}

func (t *ReadFileTool) duplicateWindow(path string, offset, limit int, info os.FileInfo) bool {
	if t == nil {
		return false
	}
	key := fmt.Sprintf("%s:%d:%d:%d:%d", path, offset, limit, info.Size(), info.ModTime().UnixNano())
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.seen == nil {
		t.seen = make(map[string]struct{})
	}
	if _, ok := t.seen[key]; ok {
		return true
	}
	t.seen[key] = struct{}{}
	return false
}

func marshalReadFileStatus(path string, read FileReadResult, state FileStateSnapshot) (json.RawMessage, error) {
	evidence := fileReadEvidenceWithPath(read.Evidence, path)
	payload := map[string]any{
		"path":             path,
		"status":           read.DedupStatus,
		"dedup":            read.DedupStatus == FileReadDedupStatusUnchanged,
		"content_returned": false,
	}
	if len(evidence) > 0 {
		payload["evidence"] = evidence
	}
	for _, ev := range evidence {
		if ev.Kind == read.DedupStatus && strings.TrimSpace(ev.Message) != "" {
			payload["message"] = ev.Message
			break
		}
	}
	if read.DedupStatus == FileReadStatusDedupStubBlocked {
		payload["error"] = payload["message"]
	}
	if statePayload := fileStatePayload(state); statePayload != nil {
		payload["file_state"] = statePayload
	}
	return marshalToolPayload(payload)
}

func fileReadEvidenceWithPath(in []FileReadEvidence, path string) []FileReadEvidence {
	if len(in) == 0 {
		return nil
	}
	out := make([]FileReadEvidence, len(in))
	for i, evidence := range in {
		out[i] = evidence
		out[i].Path = path
	}
	return out
}

// SearchFilesTool implements a local ripgrep-like search_files contract.
type SearchFilesTool struct {
	cfg FileTaskToolConfig
}

func NewSearchFilesTool(cfg FileTaskToolConfig) *SearchFilesTool {
	return &SearchFilesTool{cfg: cfg}
}

func (*SearchFilesTool) Name() string { return "search_files" }

func (*SearchFilesTool) Description() string {
	return "Search file contents or find files by name. Use this instead of grep/rg/find/ls in terminal."
}

func (*SearchFilesTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string","description":"Regex pattern for content search, or glob pattern for file search."},"target":{"type":"string","enum":["content","files"],"default":"content"},"path":{"type":"string","description":"Directory or file to search in.","default":"."},"file_glob":{"type":"string","description":"Optional file glob filter for content search."},"limit":{"type":"integer","default":50},"offset":{"type":"integer","default":0},"output_mode":{"type":"string","enum":["content","files_only","count"],"default":"content"},"context":{"type":"integer","default":0}},"required":["pattern"]}`)
}

func (*SearchFilesTool) Timeout() time.Duration { return 15 * time.Second }

func (t *SearchFilesTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Pattern    string `json:"pattern"`
		Target     string `json:"target"`
		Path       string `json:"path"`
		FileGlob   string `json:"file_glob"`
		Limit      int    `json:"limit"`
		Offset     int    `json:"offset"`
		OutputMode string `json:"output_mode"`
		Context    int    `json:"context"`
	}
	if err := json.Unmarshal(defaultJSONArgs(args), &in); err != nil {
		return marshalToolPayload(map[string]any{"error": "invalid search_files args: " + err.Error()})
	}
	if strings.TrimSpace(in.Pattern) == "" {
		return marshalToolPayload(map[string]any{"error": "search_files pattern is required"})
	}
	target := strings.TrimSpace(in.Target)
	if target == "" || target == "grep" {
		target = "content"
	}
	if target == "find" {
		target = "files"
	}
	if target != "content" && target != "files" {
		return marshalToolPayload(map[string]any{"error": "search_files target must be content or files"})
	}
	searchPath := in.Path
	if strings.TrimSpace(searchPath) == "" {
		searchPath = "."
	}
	base, relBase, root, _, err := resolveFileTaskPath(t.cfg, searchPath)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": searchPath, "error": err.Error()})
	}
	limit := clampInt(in.Limit, defaultSearchFilesLimit, defaultSearchFilesMax)
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	if target == "files" {
		return t.searchFileNames(ctx, root, base, relBase, in.Pattern, offset, limit)
	}
	return t.searchContents(ctx, root, base, relBase, in, offset, limit)
}

func (t *SearchFilesTool) searchFileNames(ctx context.Context, root, base, relBase, pattern string, offset, limit int) (json.RawMessage, error) {
	var matches []string
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if shouldSkipSearchDir(base, path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel := workspaceRel(root, path)
		if globMatches(pattern, rel) || globMatches(pattern, filepath.Base(path)) {
			matches = append(matches, rel)
		}
		return nil
	})
	if err != nil {
		return marshalToolPayload(map[string]any{"path": relBase, "error": "search files: " + err.Error()})
	}
	sort.Strings(matches)
	window, truncated := windowStrings(matches, offset, limit)
	return marshalToolPayload(map[string]any{
		"pattern":   pattern,
		"target":    "files",
		"path":      relBase,
		"count":     len(matches),
		"files":     window,
		"truncated": truncated,
	})
}

func (t *SearchFilesTool) searchContents(ctx context.Context, root, base, relBase string, in struct {
	Pattern    string `json:"pattern"`
	Target     string `json:"target"`
	Path       string `json:"path"`
	FileGlob   string `json:"file_glob"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	OutputMode string `json:"output_mode"`
	Context    int    `json:"context"`
}, offset, limit int) (json.RawMessage, error) {
	re, err := regexp.Compile(in.Pattern)
	if err != nil {
		return marshalToolPayload(map[string]any{"pattern": in.Pattern, "error": "invalid regex: " + err.Error()})
	}
	outputMode := strings.TrimSpace(in.OutputMode)
	if outputMode == "" {
		outputMode = "content"
	}
	counts := map[string]int{}
	filesSeen := map[string]bool{}
	var results []map[string]any
	emittedContext := map[string]bool{}
	contextLines := in.Context
	if contextLines < 0 {
		contextLines = 0
	}
	err = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if shouldSkipSearchDir(base, path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel := workspaceRel(root, path)
		if in.FileGlob != "" && !globMatches(in.FileGlob, rel) && !globMatches(in.FileGlob, filepath.Base(path)) {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil || binaryLike(raw) {
			return nil
		}
		lines := splitTextLines(string(raw))
		for i, line := range lines {
			if !re.MatchString(line) {
				continue
			}
			counts[rel]++
			filesSeen[rel] = true
			if outputMode == "content" {
				start, end := i, i
				if contextLines > 0 {
					start = i - contextLines
					if start < 0 {
						start = 0
					}
					end = i + contextLines
					if end >= len(lines) {
						end = len(lines) - 1
					}
				}
				for lineIndex := start; lineIndex <= end; lineIndex++ {
					key := rel + "\x00" + strconv.Itoa(lineIndex)
					if emittedContext[key] {
						continue
					}
					emittedContext[key] = true
					results = append(results, map[string]any{
						"path": rel,
						"line": lineIndex + 1,
						"text": lines[lineIndex],
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		return marshalToolPayload(map[string]any{"path": relBase, "error": "search contents: " + err.Error()})
	}

	switch outputMode {
	case "files_only":
		files := sortedMapKeys(filesSeen)
		window, truncated := windowStrings(files, offset, limit)
		return marshalToolPayload(map[string]any{
			"pattern":   in.Pattern,
			"target":    "content",
			"path":      relBase,
			"count":     len(files),
			"files":     window,
			"truncated": truncated,
		})
	case "count":
		files := sortedMapKeys(filesSeen)
		window, truncated := windowStrings(files, offset, limit)
		rows := make([]map[string]any, 0, len(window))
		for _, file := range window {
			rows = append(rows, map[string]any{"path": file, "count": counts[file]})
		}
		return marshalToolPayload(map[string]any{
			"pattern":   in.Pattern,
			"target":    "content",
			"path":      relBase,
			"count":     len(files),
			"matches":   rows,
			"truncated": truncated,
		})
	case "content":
		window, truncated := windowMatches(results, offset, limit)
		return marshalToolPayload(map[string]any{
			"pattern":   in.Pattern,
			"target":    "content",
			"path":      relBase,
			"count":     len(results),
			"matches":   window,
			"truncated": truncated,
		})
	default:
		return marshalToolPayload(map[string]any{"error": "search_files output_mode must be content, files_only, or count"})
	}
}

// WriteFileTool implements complete file replacement under the workspace root.
type WriteFileTool struct {
	cfg FileTaskToolConfig
}

func NewWriteFileTool(cfg FileTaskToolConfig) *WriteFileTool {
	return &WriteFileTool{cfg: cfg}
}

func (*WriteFileTool) Name() string { return "write_file" }

func (*WriteFileTool) Description() string {
	return "Write content to a file, completely replacing existing content. Use this instead of shell heredocs in terminal."
}

func (*WriteFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Path to write, relative to the workspace root or absolute inside it."},"content":{"type":"string","description":"Complete replacement content."}},"required":["path","content"]}`)
}

func (*WriteFileTool) Timeout() time.Duration { return 5 * time.Second }

func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(defaultJSONArgs(args), &in); err != nil {
		return marshalToolPayload(map[string]any{"error": "invalid write_file args: " + err.Error()})
	}
	resolved, rel, root, cwd, err := resolveFileTaskPath(t.cfg, in.Path)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": in.Path, "error": err.Error()})
	}
	registry := fileTaskStateRegistry(t.cfg)
	if check := registry.check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
		return marshalToolPayload(fileStateErrorPayload(rel, check))
	}
	if isFileReadGuardStatusText([]byte(in.Content)) {
		return marshalToolPayload(map[string]any{"path": rel, "error": ErrFileReadGuardStatusContent.Error()})
	}
	var preContent *string
	if supportsStructuredLint(rel) {
		if raw, err := os.ReadFile(resolved); err == nil {
			pre := string(raw)
			preContent = &pre
		}
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "create parent directories: " + err.Error()})
	}
	if err := os.WriteFile(resolved, []byte(in.Content), defaultWriteFileFileMode); err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "write file: " + err.Error()})
	}
	state, _ := registry.record(root, t.cfg.TaskID, cwd, rel, resolved)
	payload := map[string]any{
		"path":          rel,
		"bytes_written": len([]byte(in.Content)),
		"status":        "ok",
	}
	if lint, ok := structuredLintDelta(rel, in.Content, preContent); ok {
		payload["lint"] = lint
	}
	if statePayload := fileStatePayload(state); statePayload != nil {
		payload["file_state"] = statePayload
	}
	return marshalToolPayload(payload)
}

// PatchTool implements targeted replace edits under the workspace root.
type PatchTool struct {
	cfg FileTaskToolConfig
}

func NewPatchTool(cfg FileTaskToolConfig) *PatchTool {
	return &PatchTool{cfg: cfg}
}

func (*PatchTool) Name() string { return "patch" }

func (*PatchTool) Description() string {
	return "Targeted find-and-replace edits in files. Use this instead of sed/awk in terminal."
}

func (*PatchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"mode":{"type":"string","enum":["replace","patch"],"default":"replace"},"path":{"type":"string","description":"File path to edit when mode=replace."},"old_string":{"type":"string","description":"Text to find when mode=replace. Must be unique unless replace_all=true."},"new_string":{"type":"string","description":"Replacement text when mode=replace. Can be empty."},"replace_all":{"type":"boolean","default":false},"patch":{"type":"string","description":"V4A patch content when mode=patch."}},"required":["mode"]}`)
}

func (*PatchTool) Timeout() time.Duration { return 5 * time.Second }

func (t *PatchTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	var in struct {
		Mode       string `json:"mode"`
		Path       string `json:"path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
		Patch      string `json:"patch"`
	}
	if err := json.Unmarshal(defaultJSONArgs(args), &in); err != nil {
		return marshalToolPayload(map[string]any{"error": "invalid patch args: " + err.Error()})
	}
	mode := strings.TrimSpace(in.Mode)
	if mode == "" {
		mode = "replace"
	}
	if mode == "patch" {
		return t.executeV4APatch(in.Patch)
	}
	if mode != "replace" {
		return marshalToolPayload(map[string]any{"error": "patch currently supports replace mode only"})
	}
	if in.OldString == "" {
		return marshalToolPayload(map[string]any{"path": in.Path, "error": "old_string is required"})
	}
	if isFileReadGuardStatusText([]byte(in.NewString)) {
		return marshalToolPayload(map[string]any{"path": in.Path, "error": ErrFileReadGuardStatusContent.Error()})
	}
	resolved, rel, root, cwd, err := resolveFileTaskPath(t.cfg, in.Path)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": in.Path, "error": err.Error()})
	}
	registry := fileTaskStateRegistry(t.cfg)
	if check := registry.check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
		return marshalToolPayload(fileStateErrorPayload(rel, check))
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "read file: " + err.Error()})
	}
	if binaryLike(raw) {
		return marshalToolPayload(map[string]any{"path": rel, "error": "patch cannot edit binary files"})
	}
	content := string(raw)
	count := strings.Count(content, in.OldString)
	if count == 0 {
		return marshalToolPayload(map[string]any{"path": rel, "error": "old_string not found"})
	}
	if count > 1 && !in.ReplaceAll {
		return marshalToolPayload(map[string]any{"path": rel, "error": fmt.Sprintf("old_string matched %d times; set replace_all=true or include more context", count)})
	}
	replacements := 1
	if in.ReplaceAll {
		replacements = count
	}
	updated := strings.Replace(content, in.OldString, in.NewString, replacements)
	if err := os.WriteFile(resolved, []byte(updated), defaultWriteFileFileMode); err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "write patched file: " + err.Error()})
	}
	state, _ := registry.record(root, t.cfg.TaskID, cwd, rel, resolved)
	payload := map[string]any{
		"path":         rel,
		"replacements": replacements,
		"status":       "ok",
	}
	if lint, ok := structuredLintDelta(rel, updated, &content); ok {
		payload["lint"] = lint
	}
	if statePayload := fileStatePayload(state); statePayload != nil {
		payload["file_state"] = statePayload
	}
	return marshalToolPayload(payload)
}

func (t *PatchTool) executeV4APatch(patchText string) (json.RawMessage, error) {
	ops, err := parseV4APatch(patchText)
	if err != nil {
		return marshalPatchValidationError(err.Error())
	}
	if len(ops) == 0 {
		return marshalPatchValidationError("patch contains no operations")
	}
	actions := make([]v4aPatchAction, 0, len(ops))
	for _, op := range ops {
		action, check, err := t.validateV4AOperation(op)
		if check != nil {
			return marshalToolPayload(fileStateErrorPayload(action.rel, check))
		}
		if err != nil {
			return marshalPatchValidationError(err.Error())
		}
		actions = append(actions, action)
	}

	var modified, created, deleted []string
	lintResults := map[string]structuredLintResult{}
	registry := fileTaskStateRegistry(t.cfg)
	for _, action := range actions {
		switch action.kind {
		case v4aOperationAdd:
			if err := os.MkdirAll(filepath.Dir(action.abs), 0o755); err != nil {
				return marshalPatchApplyError(action.rel, "create parent directories: "+err.Error(), modified, created, deleted)
			}
			if err := os.WriteFile(action.abs, []byte(action.content), defaultWriteFileFileMode); err != nil {
				return marshalPatchApplyError(action.rel, "write added file: "+err.Error(), modified, created, deleted)
			}
			_, _ = registry.record(action.root, t.cfg.TaskID, action.cwd, action.rel, action.abs)
			created = append(created, action.rel)
			if lint, ok := structuredLintDelta(action.rel, action.content, nil); ok {
				lintResults[action.rel] = lint
			}
		case v4aOperationUpdate:
			if err := os.WriteFile(action.abs, []byte(action.content), defaultWriteFileFileMode); err != nil {
				return marshalPatchApplyError(action.rel, "write patched file: "+err.Error(), modified, created, deleted)
			}
			_, _ = registry.record(action.root, t.cfg.TaskID, action.cwd, action.rel, action.abs)
			modified = append(modified, action.rel)
			if lint, ok := structuredLintDelta(action.rel, action.content, action.preContent); ok {
				lintResults[action.rel] = lint
			}
		case v4aOperationDelete:
			if err := os.Remove(action.abs); err != nil {
				return marshalPatchApplyError(action.rel, "delete file: "+err.Error(), modified, created, deleted)
			}
			deleted = append(deleted, action.rel)
		case v4aOperationMove:
			if err := os.MkdirAll(filepath.Dir(action.newAbs), 0o755); err != nil {
				return marshalPatchApplyError(action.rel, "create move parent directories: "+err.Error(), modified, created, deleted)
			}
			if err := os.Rename(action.abs, action.newAbs); err != nil {
				return marshalPatchApplyError(action.rel, "move file: "+err.Error(), modified, created, deleted)
			}
			_, _ = registry.record(action.newRoot, t.cfg.TaskID, action.newCWD, action.newRel, action.newAbs)
			modified = append(modified, action.rel+" -> "+action.newRel)
		}
	}
	payload := map[string]any{
		"status":         "ok",
		"success":        true,
		"operations":     len(actions),
		"files_modified": modified,
		"files_created":  created,
		"files_deleted":  deleted,
	}
	if len(lintResults) > 0 {
		payload["lint"] = lintResults
	}
	return marshalToolPayload(payload)
}

func (t *PatchTool) validateV4AOperation(op v4aPatchOperation) (v4aPatchAction, *fileStateCheck, error) {
	action := v4aPatchAction{kind: op.kind}
	resolved, rel, root, cwd, err := resolveFileTaskPath(t.cfg, op.path)
	action.abs, action.rel, action.root, action.cwd = resolved, rel, root, cwd
	if err != nil {
		return action, nil, err
	}

	registry := fileTaskStateRegistry(t.cfg)
	switch op.kind {
	case v4aOperationAdd:
		if _, err := os.Stat(resolved); err == nil {
			return action, nil, fmt.Errorf("add file %s: file already exists", rel)
		} else if !os.IsNotExist(err) {
			return action, nil, fmt.Errorf("add file %s: stat target: %w", rel, err)
		}
		action.content = v4aAddContent(op)
	case v4aOperationUpdate:
		if check := registry.check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
			return action, check, nil
		}
		raw, err := os.ReadFile(resolved)
		if err != nil {
			return action, nil, fmt.Errorf("update file %s: read file: %w", rel, err)
		}
		if binaryLike(raw) {
			return action, nil, fmt.Errorf("update file %s: patch cannot edit binary files", rel)
		}
		updated, err := applyV4AHunks(string(raw), op.hunks)
		if err != nil {
			return action, nil, fmt.Errorf("update file %s: %w", rel, err)
		}
		pre := string(raw)
		action.preContent = &pre
		action.content = updated
	case v4aOperationDelete:
		if check := registry.check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
			return action, check, nil
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return action, nil, fmt.Errorf("delete file %s: stat target: %w", rel, err)
		}
		if info.IsDir() {
			return action, nil, fmt.Errorf("delete file %s: expected a file, got a directory", rel)
		}
	case v4aOperationMove:
		if strings.TrimSpace(op.newPath) == "" {
			return action, nil, fmt.Errorf("move file %s: destination path is required", rel)
		}
		if check := registry.check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
			return action, check, nil
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return action, nil, fmt.Errorf("move file %s: stat source: %w", rel, err)
		}
		if info.IsDir() {
			return action, nil, fmt.Errorf("move file %s: expected a file, got a directory", rel)
		}
		newAbs, newRel, newRoot, newCWD, err := resolveFileTaskPath(t.cfg, op.newPath)
		if err != nil {
			return action, nil, err
		}
		action.newAbs, action.newRel, action.newRoot, action.newCWD = newAbs, newRel, newRoot, newCWD
		if _, err := os.Stat(newAbs); err == nil {
			return action, nil, fmt.Errorf("move file %s -> %s: destination already exists", rel, newRel)
		} else if !os.IsNotExist(err) {
			return action, nil, fmt.Errorf("move file %s -> %s: stat destination: %w", rel, newRel, err)
		}
	default:
		return action, nil, fmt.Errorf("unsupported V4A operation for %s", rel)
	}
	return action, nil, nil
}

func marshalPatchValidationError(message string) (json.RawMessage, error) {
	return marshalToolPayload(map[string]any{
		"status":  "patch_validation_failed",
		"success": false,
		"error":   message,
	})
}

func marshalPatchApplyError(path, message string, modified, created, deleted []string) (json.RawMessage, error) {
	return marshalToolPayload(map[string]any{
		"path":           path,
		"status":         "patch_apply_failed",
		"success":        false,
		"error":          message,
		"files_modified": modified,
		"files_created":  created,
		"files_deleted":  deleted,
	})
}

type v4aOperationKind string

const (
	v4aOperationAdd    v4aOperationKind = "add"
	v4aOperationUpdate v4aOperationKind = "update"
	v4aOperationDelete v4aOperationKind = "delete"
	v4aOperationMove   v4aOperationKind = "move"
)

type v4aPatchOperation struct {
	kind    v4aOperationKind
	path    string
	newPath string
	hunks   []v4aPatchHunk
}

type v4aPatchHunk struct {
	contextHint string
	lines       []v4aPatchLine
}

type v4aPatchLine struct {
	prefix byte
	text   string
}

type v4aPatchAction struct {
	kind       v4aOperationKind
	rel        string
	abs        string
	root       string
	cwd        string
	newRel     string
	newAbs     string
	newRoot    string
	newCWD     string
	content    string
	preContent *string
}

func parseV4APatch(patchText string) ([]v4aPatchOperation, error) {
	lines := strings.Split(strings.ReplaceAll(patchText, "\r\n", "\n"), "\n")
	start, end := -1, len(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "*** Begin Patch" || trimmed == "***Begin Patch" {
			start = i
		}
		if trimmed == "*** End Patch" || trimmed == "***End Patch" {
			end = i
			break
		}
	}

	var ops []v4aPatchOperation
	var current *v4aPatchOperation
	var currentHunk *v4aPatchHunk
	flushHunk := func() {
		if current != nil && currentHunk != nil && len(currentHunk.lines) > 0 {
			current.hunks = append(current.hunks, *currentHunk)
		}
		currentHunk = nil
	}
	flushOp := func() {
		if current != nil {
			flushHunk()
			ops = append(ops, *current)
		}
		current = nil
	}

	for i := start + 1; i < end; i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "*** Update File:"):
			flushOp()
			current = &v4aPatchOperation{kind: v4aOperationUpdate, path: strings.TrimSpace(strings.TrimPrefix(line, "*** Update File:"))}
		case strings.HasPrefix(line, "*** Add File:"):
			flushOp()
			current = &v4aPatchOperation{kind: v4aOperationAdd, path: strings.TrimSpace(strings.TrimPrefix(line, "*** Add File:"))}
			currentHunk = &v4aPatchHunk{}
		case strings.HasPrefix(line, "*** Delete File:"):
			flushOp()
			ops = append(ops, v4aPatchOperation{kind: v4aOperationDelete, path: strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File:"))})
		case strings.HasPrefix(line, "*** Move File:"):
			flushOp()
			spec := strings.TrimSpace(strings.TrimPrefix(line, "*** Move File:"))
			parts := strings.SplitN(spec, "->", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("move operation %q missing destination (expected 'src -> dst')", spec)
			}
			ops = append(ops, v4aPatchOperation{kind: v4aOperationMove, path: strings.TrimSpace(parts[0]), newPath: strings.TrimSpace(parts[1])})
		case strings.HasPrefix(line, "@@"):
			if current == nil {
				return nil, fmt.Errorf("hunk marker appears before a file operation")
			}
			flushHunk()
			currentHunk = &v4aPatchHunk{contextHint: parseV4AContextHint(line)}
		case current != nil:
			if currentHunk == nil {
				currentHunk = &v4aPatchHunk{}
			}
			if strings.HasPrefix(line, "\\") {
				continue
			}
			if line == "" {
				currentHunk.lines = append(currentHunk.lines, v4aPatchLine{prefix: ' ', text: ""})
				continue
			}
			switch line[0] {
			case '+', '-', ' ':
				currentHunk.lines = append(currentHunk.lines, v4aPatchLine{prefix: line[0], text: line[1:]})
			default:
				currentHunk.lines = append(currentHunk.lines, v4aPatchLine{prefix: ' ', text: line})
			}
		case strings.TrimSpace(line) == "":
			continue
		default:
			return nil, fmt.Errorf("unexpected patch line before file operation: %q", line)
		}
	}
	flushOp()

	for _, op := range ops {
		if strings.TrimSpace(op.path) == "" {
			return nil, fmt.Errorf("%s operation has empty file path", op.kind)
		}
		if op.kind == v4aOperationUpdate && len(op.hunks) == 0 {
			return nil, fmt.Errorf("update %s: no hunks found", op.path)
		}
	}
	return ops, nil
}

func parseV4AContextHint(line string) string {
	hint := strings.TrimSpace(strings.TrimPrefix(line, "@@"))
	hint = strings.TrimSpace(strings.TrimSuffix(hint, "@@"))
	return hint
}

func v4aAddContent(op v4aPatchOperation) string {
	var lines []string
	for _, hunk := range op.hunks {
		for _, line := range hunk.lines {
			if line.prefix == '+' {
				lines = append(lines, line.text)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func applyV4AHunks(content string, hunks []v4aPatchHunk) (string, error) {
	updated := content
	for _, hunk := range hunks {
		var searchLines, replaceLines []string
		for _, line := range hunk.lines {
			switch line.prefix {
			case ' ':
				searchLines = append(searchLines, line.text)
				replaceLines = append(replaceLines, line.text)
			case '-':
				searchLines = append(searchLines, line.text)
			case '+':
				replaceLines = append(replaceLines, line.text)
			}
		}
		if len(searchLines) == 0 {
			var err error
			updated, err = applyV4AAdditionOnlyHunk(updated, hunk.contextHint, strings.Join(replaceLines, "\n"))
			if err != nil {
				return "", err
			}
			continue
		}
		search := strings.Join(searchLines, "\n")
		replacement := strings.Join(replaceLines, "\n")
		count := strings.Count(updated, search)
		if count == 0 {
			return "", fmt.Errorf("could not apply hunk: context not found")
		}
		if count > 1 {
			return "", fmt.Errorf("could not apply hunk: context matched %d times", count)
		}
		updated = strings.Replace(updated, search, replacement, 1)
	}
	return updated, nil
}

func applyV4AAdditionOnlyHunk(content, hint, insertText string) (string, error) {
	if hint == "" {
		return appendV4AText(content, insertText), nil
	}
	count := strings.Count(content, hint)
	if count == 0 {
		return appendV4AText(content, insertText), nil
	}
	if count > 1 {
		return "", fmt.Errorf("addition-only hunk: context hint %q is ambiguous (%d occurrences)", hint, count)
	}
	pos := strings.Index(content, hint)
	eol := strings.Index(content[pos:], "\n")
	if eol == -1 {
		if strings.HasSuffix(content, "\n") {
			return content + insertText, nil
		}
		return content + "\n" + insertText, nil
	}
	insertAt := pos + eol + 1
	return content[:insertAt] + insertText + "\n" + content[insertAt:], nil
}

func appendV4AText(content, insertText string) string {
	if content == "" {
		return insertText
	}
	return strings.TrimRight(content, "\n") + "\n" + insertText + "\n"
}

func supportsStructuredLint(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml", ".toml":
		return true
	default:
		return false
	}
}

func structuredLintDelta(path, postContent string, preContent *string) (structuredLintResult, bool) {
	post, ok := runStructuredLint(path, postContent)
	if !ok {
		return structuredLintResult{}, false
	}
	if post.Success || preContent == nil {
		return post, true
	}
	pre, ok := runStructuredLint(path, *preContent)
	if !ok || pre.Success || strings.TrimSpace(pre.Output) == "" {
		return post, true
	}
	preLines := lintOutputLineSet(pre.Output)
	var newLines []string
	for _, line := range strings.Split(post.Output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if _, existed := preLines[trimmed]; !existed {
			newLines = append(newLines, line)
		}
	}
	if len(newLines) == 0 {
		post.Message = "Pre-existing lint errors - this edit didn't introduce new ones but the file is still broken."
		return post, true
	}
	post.Output = boundStructuredLintOutput("New lint errors introduced by this edit (pre-existing errors filtered out):\n" + strings.Join(newLines, "\n"))
	return post, true
}

func runStructuredLint(path, content string) (structuredLintResult, bool) {
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		var decoded any
		err = json.Unmarshal([]byte(content), &decoded)
	case ".yaml", ".yml":
		var decoded any
		err = yaml.Unmarshal([]byte(content), &decoded)
	case ".toml":
		var decoded map[string]any
		err = toml.Unmarshal([]byte(content), &decoded)
	default:
		return structuredLintResult{}, false
	}
	if err != nil {
		return structuredLintResult{Success: false, Output: boundStructuredLintOutput(err.Error())}, true
	}
	return structuredLintResult{Success: true}, true
}

func lintOutputLineSet(output string) map[string]struct{} {
	lines := map[string]struct{}{}
	for _, line := range strings.Split(output, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines[trimmed] = struct{}{}
		}
	}
	return lines
}

func boundStructuredLintOutput(output string) string {
	output = strings.TrimSpace(output)
	if len(output) <= maxStructuredLintOutput {
		return output
	}
	if maxStructuredLintOutput <= len("...") {
		return output[:maxStructuredLintOutput]
	}
	return output[:maxStructuredLintOutput-len("...")] + "..."
}

func resolveWorkspaceFile(root, rawPath string) (string, string, error) {
	workspaceRoot, err := resolveWorkspaceRoot(root)
	if err != nil {
		return "", "", err
	}
	return resolveWorkspacePath(workspaceRoot, rawPath)
}

func resolveFileTaskPath(cfg FileTaskToolConfig, rawPath string) (string, string, string, string, error) {
	workspaceRoot, err := resolveWorkspaceRoot(cfg.Root)
	if err != nil {
		return "", "", "", "", err
	}
	cwd, cwdRel, err := resolveFileTaskCWD(cfg, workspaceRoot)
	if err != nil {
		return "", "", "", "", err
	}
	resolved, rel, err := resolveWorkspacePathFromBase(workspaceRoot, cwd, rawPath)
	if err != nil {
		return "", "", "", "", err
	}
	return resolved, rel, workspaceRoot, cwdRel, nil
}

func resolveFileTaskCWD(cfg FileTaskToolConfig, root string) (string, string, error) {
	raw := ""
	if cfg.CWDResolver != nil {
		raw = strings.TrimSpace(cfg.CWDResolver())
	} else if strings.TrimSpace(cfg.Root) == "" {
		raw = strings.TrimSpace(os.Getenv("TERMINAL_CWD"))
	}
	if raw == "" || terminalCWDPlaceholder(raw) {
		return root, ".", nil
	}
	expanded, err := expandUserPath(raw)
	if err != nil {
		return "", "", err
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(root, expanded)
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", "", fmt.Errorf("resolve task cwd: %w", err)
	}
	abs = filepath.Clean(abs)
	if !pathWithinRoot(root, abs) {
		return "", "", fmt.Errorf("task cwd %q is outside workspace root %q", raw, root)
	}
	if err := validateWorkspaceRealPath(root, abs); err != nil {
		return "", "", err
	}
	return abs, workspaceRel(root, abs), nil
}

func resolveWorkspacePathFromBase(root, base, rawPath string) (string, string, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", "", errors.New("path is required")
	}
	expanded, err := expandUserPath(path)
	if err != nil {
		return "", "", err
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(base, expanded)
	}
	return resolveWorkspacePath(root, expanded)
}

func fileTaskStateRegistry(cfg FileTaskToolConfig) *FileStateRegistry {
	if cfg.StateRegistry != nil {
		return cfg.StateRegistry
	}
	return defaultFileStateRegistry
}

func resolveWorkspaceRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve workspace root: %w", err)
		}
		root = cwd
	}
	expanded, err := expandUserPath(root)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	abs = filepath.Clean(abs)
	if realRoot, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(realRoot), nil
	}
	return abs, nil
}

func resolveWorkspacePath(root, rawPath string) (string, string, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", "", errors.New("path is required")
	}
	expanded, err := expandUserPath(path)
	if err != nil {
		return "", "", err
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(root, expanded)
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", "", fmt.Errorf("resolve path: %w", err)
	}
	abs = filepath.Clean(abs)
	if !pathWithinRoot(root, abs) {
		return "", "", fmt.Errorf("path %q is outside workspace root %q", rawPath, root)
	}
	if err := validateWorkspaceRealPath(root, abs); err != nil {
		return "", "", err
	}
	return abs, workspaceRel(root, abs), nil
}

func validateWorkspaceRealPath(root, abs string) error {
	if realPath, err := filepath.EvalSymlinks(abs); err == nil {
		if !pathWithinRoot(root, filepath.Clean(realPath)) {
			return fmt.Errorf("path %q resolves outside workspace root %q", abs, root)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("resolve symlink path: %w", err)
	} else if info, lstatErr := os.Lstat(abs); lstatErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path %q is a broken symlink", abs)
	}

	ancestor := filepath.Dir(abs)
	for {
		if ancestor == "" || ancestor == "." || ancestor == string(filepath.Separator) {
			break
		}
		if realAncestor, err := filepath.EvalSymlinks(ancestor); err == nil {
			if !pathWithinRoot(root, filepath.Clean(realAncestor)) {
				return fmt.Errorf("path parent %q resolves outside workspace root %q", ancestor, root)
			}
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("resolve parent symlink path: %w", err)
		}
		next := filepath.Dir(ancestor)
		if next == ancestor {
			break
		}
		ancestor = next
	}
	return nil
}

func expandUserPath(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func pathWithinRoot(root, path string) bool {
	root = evalExistingPath(root)
	path = evalPathOrParent(path)
	if root == string(filepath.Separator) {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func evalPathOrParent(path string) string {
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(evaluated)
	}
	parent := filepath.Dir(path)
	if evaluatedParent, err := filepath.EvalSymlinks(parent); err == nil {
		return filepath.Clean(filepath.Join(evaluatedParent, filepath.Base(path)))
	}
	return filepath.Clean(path)
}

func evalExistingPath(path string) string {
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(evaluated)
	}
	return filepath.Clean(path)
}

func workspaceRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	if rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}

func splitTextLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}

func binaryLike(raw []byte) bool {
	if bytes.IndexByte(raw, 0) >= 0 {
		return true
	}
	return len(raw) > 0 && !utf8.Valid(raw)
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", ".cache":
		return true
	default:
		return false
	}
}

func shouldSkipSearchDir(base, path, name string) bool {
	if filepath.Clean(path) == filepath.Clean(base) {
		return false
	}
	return shouldSkipDir(name) || isHiddenPathPart(name)
}

func isHiddenPathPart(name string) bool {
	return name != "." && name != ".." && strings.HasPrefix(name, ".")
}

var searchContextLinePattern = regexp.MustCompile(`-(\d+)-`)

func parseSearchContextLine(line string) (string, int, string, bool) {
	if line == "" || line == "--" {
		return "", 0, "", false
	}
	matches := searchContextLinePattern.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return "", 0, "", false
	}
	match := matches[len(matches)-1]
	path := line[:match[0]]
	if path == "" {
		return "", 0, "", false
	}
	lineNumber, err := strconv.Atoi(line[match[2]:match[3]])
	if err != nil {
		return "", 0, "", false
	}
	return path, lineNumber, line[match[1]:], true
}

func globMatches(pattern, value string) bool {
	if pattern == "" {
		return true
	}
	pattern = filepath.ToSlash(pattern)
	value = filepath.ToSlash(value)
	ok, err := filepath.Match(pattern, value)
	if err == nil && ok {
		return true
	}
	ok, err = filepath.Match(pattern, filepath.Base(value))
	return err == nil && ok
}

func sortedMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func clampInt(value, fallback, max int) int {
	if value <= 0 {
		value = fallback
	}
	if value > max {
		return max
	}
	return value
}

func windowStrings(values []string, offset, limit int) ([]string, bool) {
	if offset > len(values) {
		offset = len(values)
	}
	end := offset + limit
	if end > len(values) {
		end = len(values)
	}
	return append([]string(nil), values[offset:end]...), end < len(values)
}

func windowMatches(values []map[string]any, offset, limit int) ([]map[string]any, bool) {
	if offset > len(values) {
		offset = len(values)
	}
	end := offset + limit
	if end > len(values) {
		end = len(values)
	}
	return append([]map[string]any(nil), values[offset:end]...), end < len(values)
}

func defaultJSONArgs(args json.RawMessage) json.RawMessage {
	if len(strings.TrimSpace(string(args))) == 0 {
		return json.RawMessage(`{}`)
	}
	return args
}

func marshalToolPayload(payload any) (json.RawMessage, error) {
	raw, err := json.Marshal(payload)
	return json.RawMessage(raw), err
}
