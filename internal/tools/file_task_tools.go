package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/lsp"
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
	v4aContextHintBefore     = 500
	v4aContextHintAfter      = 2000
)

var (
	errPatchNoMatch           = errors.New("old_string not found")
	patchHorizontalWhitespace = regexp.MustCompile(`[ \t]+`)
)

// FileTaskToolConfig constrains local file tools to a workspace root.
type FileTaskToolConfig struct {
	Root           string
	WorkspaceScope *ProfileWorkspaceScope
	ReadGuard      *FileReadGuard
	StateRegistry  *FileStateRegistry
	MutationQueue  *FileMutationQueue
	TaskID         string
	CWDResolver    func() string
	MaxReadChars   int
	LSPDiagnostics lsp.PostEditService
}

type structuredLintResult struct {
	Status  string `json:"status,omitempty"`
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Message string `json:"message,omitempty"`
}

type shellLintSpec struct {
	command string
	args    []string
}

type fileLintCommandResult struct {
	Output   string
	ExitCode int
	Err      error
}

var (
	fileLintLookPath   = exec.LookPath
	fileLintRunCommand = runFileLintCommand
	fileTaskMkdirAll   = os.MkdirAll
	fileTaskReadFile   = os.ReadFile
	fileTaskWriteFile  = atomicFileTaskWrite
	fileTaskRemove     = os.Remove
	fileTaskRename     = os.Rename
)

func atomicFileTaskWrite(name string, data []byte, perm os.FileMode) error {
	return AtomicWrite(name, data)
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
	resolved, rel, root, cwd, err := resolveFileTaskReadPath(t.cfg, in.Path)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": in.Path, "error": err.Error()})
	}
	if decision := redaction.CheckSensitivePath(resolved); decision.Blocked {
		return marshalToolPayload(map[string]any{
			"path":     rel,
			"error":    "sensitive path blocked: " + decision.Reason,
			"evidence": decision.Evidence,
		})
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
		state, _ := t.fileStateRegistry().Record(root, t.cfg.TaskID, cwd, rel, resolved)
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

	state, _ := t.fileStateRegistry().Record(root, t.cfg.TaskID, cwd, rel, resolved)
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
	base, relBase, root, _, err := resolveFileTaskReadPath(t.cfg, searchPath)
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
			if decision := redaction.CheckSensitivePath(path); decision.Blocked {
				return filepath.SkipDir
			}
			if shouldSkipSearchDir(base, path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel := workspaceRel(root, path)
		if decision := redaction.CheckSensitivePath(path); decision.Blocked {
			return nil
		}
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
			if decision := redaction.CheckSensitivePath(path); decision.Blocked {
				return filepath.SkipDir
			}
			if shouldSkipSearchDir(base, path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel := workspaceRel(root, path)
		if decision := redaction.CheckSensitivePath(path); decision.Blocked {
			return nil
		}
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
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(defaultJSONArgs(args), &in); err != nil {
		return marshalToolPayload(map[string]any{"error": "invalid write_file args: " + err.Error()})
	}
	resolved, rel, root, cwd, err := resolveFileTaskWritePath(t.cfg, in.Path)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": in.Path, "error": err.Error()})
	}
	var out json.RawMessage
	queueErr := fileTaskMutationQueue(t.cfg).Run(ctx, resolved, func(ctx context.Context) error {
		var err error
		out, err = t.executeResolvedWrite(ctx, resolved, rel, root, cwd, in.Content)
		return err
	})
	if queueErr != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "file mutation queue: " + queueErr.Error()})
	}
	return out, nil
}

func (t *WriteFileTool) executeResolvedWrite(ctx context.Context, resolved, rel, root, cwd, content string) (json.RawMessage, error) {
	registry := fileTaskStateRegistry(t.cfg)
	if check := registry.Check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
		return marshalToolPayload(fileStateErrorPayload(rel, check))
	}
	if isFileReadGuardStatusText([]byte(content)) {
		return marshalToolPayload(map[string]any{"path": rel, "error": ErrFileReadGuardStatusContent.Error()})
	}
	var preContent *string
	if supportsPostEditLint(rel) {
		if raw, err := os.ReadFile(resolved); err == nil {
			pre := string(raw)
			preContent = &pre
		}
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "create parent directories: " + err.Error()})
	}
	if err := AtomicWrite(resolved, []byte(content)); err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "write file: " + err.Error()})
	}
	state, _ := registry.Record(root, t.cfg.TaskID, cwd, rel, resolved)
	payload := map[string]any{
		"path":          rel,
		"bytes_written": len([]byte(content)),
		"status":        "ok",
	}
	if lint, ok := postEditLintDelta(resolved, rel, content, preContent); ok {
		payload["lint"] = lint
	}
	if lspResult, ok := postEditLSPDiagnostics(ctx, t.cfg.LSPDiagnostics, resolved, rel, content, preContent); ok {
		payload["lsp"] = lspResult
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
		return t.executeV4APatch(ctx, in.Patch)
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
	resolved, rel, root, cwd, err := resolveFileTaskWritePath(t.cfg, in.Path)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": in.Path, "error": err.Error()})
	}
	var out json.RawMessage
	queueErr := fileTaskMutationQueue(t.cfg).Run(ctx, resolved, func(ctx context.Context) error {
		var err error
		out, err = t.executeResolvedReplace(ctx, resolved, rel, root, cwd, in.OldString, in.NewString, in.ReplaceAll)
		return err
	})
	if queueErr != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "file mutation queue: " + queueErr.Error()})
	}
	return out, nil
}

func (t *PatchTool) executeResolvedReplace(ctx context.Context, resolved, rel, root, cwd, oldString, newString string, replaceAll bool) (json.RawMessage, error) {
	registry := fileTaskStateRegistry(t.cfg)
	if check := registry.Check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
		return marshalToolPayload(fileStateErrorPayload(rel, check))
	}
	raw, err := fileTaskReadFile(resolved)
	if err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "read file: " + err.Error()})
	}
	if binaryLike(raw) {
		return marshalToolPayload(map[string]any{"path": rel, "error": "patch cannot edit binary files"})
	}
	content := string(raw)
	updated, replacements, replaceErr := fuzzyPatchReplace(content, oldString, newString, replaceAll)
	if replaceErr != nil {
		payload := map[string]any{"path": rel, "error": replaceErr.Error()}
		if errors.Is(replaceErr, errPatchNoMatch) {
			payload["error"] = "old_string not found"
			if hint := patchNoMatchHint(oldString, content); hint != "" {
				payload["hint"] = hint
			}
		}
		return marshalToolPayload(payload)
	}
	if err := fileTaskWriteFile(resolved, []byte(updated), defaultWriteFileFileMode); err != nil {
		return marshalToolPayload(map[string]any{"path": rel, "error": "write patched file: " + err.Error()})
	}
	verified, err := fileTaskReadFile(resolved)
	if err != nil {
		return marshalToolPayload(map[string]any{
			"path":  rel,
			"error": fmt.Sprintf("post-write verification failed: could not re-read %s: %v", rel, err),
		})
	}
	if normalizePatchVerificationText(string(verified)) != normalizePatchVerificationText(updated) {
		return marshalToolPayload(map[string]any{
			"path":  rel,
			"error": fmt.Sprintf("post-write verification failed for %s: did not persist intended content", rel),
		})
	}
	state, _ := registry.Record(root, t.cfg.TaskID, cwd, rel, resolved)
	payload := map[string]any{
		"path":         rel,
		"replacements": replacements,
		"status":       "ok",
	}
	if lint, ok := postEditLintDelta(resolved, rel, updated, &content); ok {
		payload["lint"] = lint
	}
	if lspResult, ok := postEditLSPDiagnostics(ctx, t.cfg.LSPDiagnostics, resolved, rel, updated, &content); ok {
		payload["lsp"] = lspResult
	}
	if statePayload := fileStatePayload(state); statePayload != nil {
		payload["file_state"] = statePayload
	}
	return marshalToolPayload(payload)
}

func normalizePatchVerificationText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

type patchTextMatch struct {
	start int
	end   int
}

func fuzzyPatchReplace(content, oldString, newString string, replaceAll bool) (string, int, error) {
	if oldString == "" {
		return content, 0, errors.New("old_string is required")
	}
	if oldString == newString {
		return content, 0, errors.New("old_string and new_string are identical")
	}
	strategies := []func(string, string) []patchTextMatch{
		exactPatchMatches,
		lineTrimmedPatchMatches,
		whitespaceNormalizedPatchMatches,
		indentationFlexiblePatchMatches,
		escapeNormalizedPatchMatches,
		trimmedBoundaryPatchMatches,
		unicodeNormalizedPatchMatches,
		blockAnchorPatchMatches,
		contextAwarePatchMatches,
	}
	for _, strategy := range strategies {
		matches := uniquePatchMatches(strategy(content, oldString))
		if len(matches) == 0 {
			continue
		}
		if len(matches) > 1 && !replaceAll {
			return content, 0, fmt.Errorf("old_string matched %d times; set replace_all=true or include more context", len(matches))
		}
		if !replaceAll {
			matches = matches[:1]
		}
		return applyPatchTextMatches(content, matches, newString), len(matches), nil
	}
	return content, 0, errPatchNoMatch
}

func exactPatchMatches(content, pattern string) []patchTextMatch {
	if pattern == "" {
		return nil
	}
	var matches []patchTextMatch
	offset := 0
	for offset <= len(content) {
		idx := strings.Index(content[offset:], pattern)
		if idx < 0 {
			break
		}
		start := offset + idx
		end := start + len(pattern)
		matches = append(matches, patchTextMatch{start: start, end: end})
		offset = end
	}
	return matches
}

func lineTrimmedPatchMatches(content, pattern string) []patchTextMatch {
	return lineBlockPatchMatches(content, pattern, func(line string, _ int, _ int) string {
		return strings.TrimSpace(line)
	})
}

func whitespaceNormalizedPatchMatches(content, pattern string) []patchTextMatch {
	return lineBlockPatchMatches(content, pattern, func(line string, _ int, _ int) string {
		return patchHorizontalWhitespace.ReplaceAllString(line, " ")
	})
}

func indentationFlexiblePatchMatches(content, pattern string) []patchTextMatch {
	return lineBlockPatchMatches(content, pattern, func(line string, _ int, _ int) string {
		return strings.TrimLeft(line, " \t")
	})
}

func escapeNormalizedPatchMatches(content, pattern string) []patchTextMatch {
	normalized := strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\r`, "\r").Replace(pattern)
	if normalized == pattern {
		return nil
	}
	return exactPatchMatches(content, normalized)
}

func trimmedBoundaryPatchMatches(content, pattern string) []patchTextMatch {
	return lineBlockPatchMatches(content, pattern, func(line string, index int, count int) string {
		if index == 0 || index == count-1 {
			return strings.TrimSpace(line)
		}
		return line
	})
}

type patchUnicodeByteMap struct {
	normalized string
	start      []int
	end        []int
	changed    bool
}

func unicodeNormalizedPatchMatches(content, pattern string) []patchTextMatch {
	contentMap := normalizePatchUnicode(content)
	patternMap := normalizePatchUnicode(pattern)
	if !contentMap.changed && !patternMap.changed {
		return nil
	}

	matches := exactPatchMatches(contentMap.normalized, patternMap.normalized)
	if len(matches) == 0 {
		matches = lineTrimmedPatchMatches(contentMap.normalized, patternMap.normalized)
	}
	if len(matches) == 0 {
		return nil
	}

	mapped := make([]patchTextMatch, 0, len(matches))
	for _, match := range matches {
		if match.start < 0 || match.end < match.start || match.start >= len(contentMap.start) || match.end >= len(contentMap.end) {
			continue
		}
		mapped = append(mapped, patchTextMatch{start: contentMap.start[match.start], end: contentMap.end[match.end]})
	}
	return mapped
}

func normalizePatchUnicode(s string) patchUnicodeByteMap {
	var b strings.Builder
	b.Grow(len(s))
	startMap := make([]int, 0, len(s)+1)
	endMap := []int{0}
	changed := false
	for offset, r := range s {
		size := utf8.RuneLen(r)
		if size < 0 {
			size = 1
		}
		origEnd := offset + size
		replacement := patchUnicodeReplacement(r)
		if replacement == "" {
			replacement = string(r)
		} else {
			changed = true
		}
		for i := 0; i < len(replacement); i++ {
			startMap = append(startMap, offset)
			endMap = append(endMap, origEnd)
		}
		b.WriteString(replacement)
	}
	startMap = append(startMap, len(s))
	return patchUnicodeByteMap{normalized: b.String(), start: startMap, end: endMap, changed: changed}
}

func patchUnicodeReplacement(r rune) string {
	switch r {
	case '\u201c', '\u201d':
		return `"`
	case '\u2018', '\u2019':
		return `'`
	case '\u2014':
		return "--"
	case '\u2013':
		return "-"
	case '\u2026':
		return "..."
	case '\u00a0':
		return " "
	default:
		return ""
	}
}

func blockAnchorPatchMatches(content, pattern string) []patchTextMatch {
	contentLines := strings.Split(content, "\n")
	normalizedContent := normalizePatchUnicode(content).normalized
	normalizedPattern := normalizePatchUnicode(pattern).normalized
	normalizedContentLines := strings.Split(normalizedContent, "\n")
	patternLines := strings.Split(strings.ReplaceAll(normalizedPattern, "\r\n", "\n"), "\n")
	if len(patternLines) < 2 || len(patternLines) > len(normalizedContentLines) {
		return nil
	}

	firstLine := strings.TrimSpace(patternLines[0])
	lastLine := strings.TrimSpace(patternLines[len(patternLines)-1])
	var candidates []int
	for i := 0; i <= len(normalizedContentLines)-len(patternLines); i++ {
		if strings.TrimSpace(normalizedContentLines[i]) == firstLine &&
			strings.TrimSpace(normalizedContentLines[i+len(patternLines)-1]) == lastLine {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	threshold := 0.70
	if len(candidates) == 1 {
		threshold = 0.50
	}
	matches := make([]patchTextMatch, 0, len(candidates))
	for _, startLine := range candidates {
		similarity := 1.0
		if len(patternLines) > 2 {
			contentMiddle := strings.Join(normalizedContentLines[startLine+1:startLine+len(patternLines)-1], "\n")
			patternMiddle := strings.Join(patternLines[1:len(patternLines)-1], "\n")
			similarity = patchSequenceSimilarity(contentMiddle, patternMiddle)
		}
		if similarity < threshold {
			continue
		}
		start, end := patchLineRange(contentLines, startLine, len(patternLines))
		matches = append(matches, patchTextMatch{start: start, end: end})
	}
	return matches
}

func contextAwarePatchMatches(content, pattern string) []patchTextMatch {
	contentLines := strings.Split(content, "\n")
	patternLines := strings.Split(strings.ReplaceAll(pattern, "\r\n", "\n"), "\n")
	if len(patternLines) == 0 || len(patternLines) > len(contentLines) {
		return nil
	}

	requiredHighSimilarity := float64(len(patternLines)) * 0.5
	matches := make([]patchTextMatch, 0)
	for startLine := 0; startLine <= len(contentLines)-len(patternLines); startLine++ {
		highSimilarity := 0
		for i, patternLine := range patternLines {
			patternTrimmed := strings.TrimSpace(strings.TrimSuffix(patternLine, "\r"))
			contentTrimmed := strings.TrimSpace(strings.TrimSuffix(contentLines[startLine+i], "\r"))
			if patchSequenceSimilarity(contentTrimmed, patternTrimmed) >= 0.80 {
				highSimilarity++
			}
		}
		if float64(highSimilarity) < requiredHighSimilarity {
			continue
		}
		start, end := patchLineRange(contentLines, startLine, len(patternLines))
		matches = append(matches, patchTextMatch{start: start, end: end})
	}
	return matches
}

func patchSequenceSimilarity(a, b string) float64 {
	if a == b {
		return 1
	}
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 && len(br) == 0 {
		return 1
	}
	if len(ar) == 0 || len(br) == 0 {
		return 0
	}
	prev := make([]int, len(br)+1)
	cur := make([]int, len(br)+1)
	for i := 1; i <= len(ar); i++ {
		for j := 1; j <= len(br); j++ {
			if ar[i-1] == br[j-1] {
				cur[j] = prev[j-1] + 1
				continue
			}
			if prev[j] >= cur[j-1] {
				cur[j] = prev[j]
			} else {
				cur[j] = cur[j-1]
			}
		}
		prev, cur = cur, prev
		for j := range cur {
			cur[j] = 0
		}
	}
	return float64(2*prev[len(br)]) / float64(len(ar)+len(br))
}

func lineBlockPatchMatches(content, pattern string, normalize func(line string, index int, count int) string) []patchTextMatch {
	if strings.TrimSpace(pattern) == "" {
		return nil
	}
	contentLines := strings.Split(content, "\n")
	patternLines := strings.Split(strings.ReplaceAll(pattern, "\r\n", "\n"), "\n")
	if len(patternLines) == 0 || len(patternLines) > len(contentLines) {
		return nil
	}
	normalizedPattern := make([]string, len(patternLines))
	for i, line := range patternLines {
		normalizedPattern[i] = normalize(strings.TrimSuffix(line, "\r"), i, len(patternLines))
	}
	var matches []patchTextMatch
	for i := 0; i <= len(contentLines)-len(patternLines); i++ {
		matched := true
		for j := range patternLines {
			if normalize(strings.TrimSuffix(contentLines[i+j], "\r"), j, len(patternLines)) != normalizedPattern[j] {
				matched = false
				break
			}
		}
		if matched {
			start, end := patchLineRange(contentLines, i, len(patternLines))
			matches = append(matches, patchTextMatch{start: start, end: end})
		}
	}
	return matches
}

func patchLineRange(lines []string, startLine, lineCount int) (int, int) {
	start := 0
	for i := 0; i < startLine; i++ {
		start += len(lines[i]) + 1
	}
	end := start + len(strings.Join(lines[startLine:startLine+lineCount], "\n"))
	return start, end
}

func uniquePatchMatches(matches []patchTextMatch) []patchTextMatch {
	if len(matches) <= 1 {
		return matches
	}
	seen := make(map[patchTextMatch]struct{}, len(matches))
	unique := make([]patchTextMatch, 0, len(matches))
	for _, match := range matches {
		if match.start < 0 || match.end < match.start {
			continue
		}
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		unique = append(unique, match)
	}
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].start == unique[j].start {
			return unique[i].end < unique[j].end
		}
		return unique[i].start < unique[j].start
	})
	return unique
}

func applyPatchTextMatches(content string, matches []patchTextMatch, replacement string) string {
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].start > matches[j].start
	})
	updated := content
	for _, match := range matches {
		updated = updated[:match.start] + replacement + updated[match.end:]
	}
	return updated
}

func patchNoMatchHint(oldString, content string) string {
	if snippet := closestPatchSections(oldString, content, 2, 3); snippet != "" {
		return boundPatchHint("Did you mean one of these sections?\n" + snippet + "\n\nUse read_file to verify the current content, or search_files to locate the text.")
	}
	return "old_string not found. Use read_file to verify the current content, or search_files to locate the text."
}

func closestPatchSections(oldString, content string, contextLines, maxResults int) string {
	if strings.TrimSpace(oldString) == "" || strings.TrimSpace(content) == "" || maxResults <= 0 {
		return ""
	}
	oldLines := strings.Split(strings.ReplaceAll(oldString, "\r\n", "\n"), "\n")
	anchor := ""
	for _, line := range oldLines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			anchor = trimmed
			break
		}
	}
	if anchor == "" {
		return ""
	}

	contentLines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	type scoredLine struct {
		score float64
		index int
	}
	var scored []scoredLine
	for i, line := range contentLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		score := lineSimilarity(anchor, trimmed)
		if score > 0.3 {
			scored = append(scored, scoredLine{score: score, index: i})
		}
	}
	if len(scored) == 0 {
		return ""
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].index < scored[j].index
		}
		return scored[i].score > scored[j].score
	})

	oldLineCount := len(oldLines)
	if oldLineCount < 1 {
		oldLineCount = 1
	}
	seen := map[string]struct{}{}
	var parts []string
	for _, candidate := range scored {
		if len(parts) >= maxResults {
			break
		}
		start := candidate.index - contextLines
		if start < 0 {
			start = 0
		}
		end := candidate.index + oldLineCount + contextLines
		if end > len(contentLines) {
			end = len(contentLines)
		}
		key := strconv.Itoa(start) + ":" + strconv.Itoa(end)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		var b strings.Builder
		for i := start; i < end; i++ {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "%4d| %s", i+1, contentLines[i])
		}
		parts = append(parts, b.String())
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n---\n")
}

func lineSimilarity(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	ar := []rune(a)
	br := []rune(b)
	prev := make([]int, len(br)+1)
	cur := make([]int, len(br)+1)
	for i := 1; i <= len(ar); i++ {
		for j := 1; j <= len(br); j++ {
			if ar[i-1] == br[j-1] {
				cur[j] = prev[j-1] + 1
				continue
			}
			if prev[j] >= cur[j-1] {
				cur[j] = prev[j]
			} else {
				cur[j] = cur[j-1]
			}
		}
		prev, cur = cur, prev
		for j := range cur {
			cur[j] = 0
		}
	}
	lcs := prev[len(br)]
	longest := len(ar)
	if len(br) > longest {
		longest = len(br)
	}
	return float64(lcs) / float64(longest)
}

func boundPatchHint(hint string) string {
	const maxPatchHintChars = 2000
	if len(hint) <= maxPatchHintChars {
		return hint
	}
	return hint[:maxPatchHintChars] + "\n[hint truncated]"
}

func (t *PatchTool) executeV4APatch(ctx context.Context, patchText string) (json.RawMessage, error) {
	ops, err := parseV4APatch(patchText)
	if err != nil {
		return marshalPatchValidationError(err.Error())
	}
	if len(ops) == 0 {
		return marshalPatchValidationError("patch contains no operations")
	}
	lockPaths, err := t.v4aMutationLockPaths(ops)
	if err != nil {
		return marshalPatchValidationError(err.Error())
	}
	var out json.RawMessage
	queueErr := fileTaskMutationQueue(t.cfg).RunMany(ctx, lockPaths, func(context.Context) error {
		var err error
		out, err = t.executeV4APatchLocked(ops)
		return err
	})
	if queueErr != nil {
		return marshalPatchApplyError("", "file mutation queue: "+queueErr.Error(), nil, nil, nil)
	}
	return out, nil
}

func (t *PatchTool) executeV4APatchLocked(ops []v4aPatchOperation) (json.RawMessage, error) {
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
	snapshotRoot, err := v4aPatchSnapshotRoot(actions)
	if err != nil {
		return marshalPatchApplyError("", "prepare rollback snapshot: "+err.Error(), nil, nil, nil)
	}
	rollbackSnapshot, err := TakeWorkspaceSnapshot(snapshotRoot)
	if err != nil {
		return marshalPatchApplyError("", "prepare rollback snapshot: "+err.Error(), nil, nil, nil)
	}

	var modified, created, deleted []string
	lintResults := map[string]structuredLintResult{}
	for _, action := range actions {
		switch action.kind {
		case v4aOperationAdd:
			if err := fileTaskMkdirAll(filepath.Dir(action.abs), 0o755); err != nil {
				return marshalPatchApplyErrorWithRollback(action.rel, "create parent directories: "+err.Error(), modified, created, deleted, rollbackSnapshot)
			}
			if err := fileTaskWriteFile(action.abs, []byte(action.content), defaultWriteFileFileMode); err != nil {
				return marshalPatchApplyErrorWithRollback(action.rel, "write added file: "+err.Error(), modified, created, deleted, rollbackSnapshot)
			}
			created = append(created, action.rel)
			if lint, ok := postEditLintDelta(action.abs, action.rel, action.content, nil); ok {
				lintResults[action.rel] = lint
			}
		case v4aOperationUpdate:
			if err := fileTaskWriteFile(action.abs, []byte(action.content), defaultWriteFileFileMode); err != nil {
				return marshalPatchApplyErrorWithRollback(action.rel, "write patched file: "+err.Error(), modified, created, deleted, rollbackSnapshot)
			}
			modified = append(modified, action.rel)
			if lint, ok := postEditLintDelta(action.abs, action.rel, action.content, action.preContent); ok {
				lintResults[action.rel] = lint
			}
		case v4aOperationDelete:
			if err := fileTaskRemove(action.abs); err != nil {
				return marshalPatchApplyErrorWithRollback(action.rel, "delete file: "+err.Error(), modified, created, deleted, rollbackSnapshot)
			}
			deleted = append(deleted, action.rel)
		case v4aOperationMove:
			if err := fileTaskMkdirAll(filepath.Dir(action.newAbs), 0o755); err != nil {
				return marshalPatchApplyErrorWithRollback(action.rel, "create move parent directories: "+err.Error(), modified, created, deleted, rollbackSnapshot)
			}
			if err := fileTaskRename(action.abs, action.newAbs); err != nil {
				return marshalPatchApplyErrorWithRollback(action.rel, "move file: "+err.Error(), modified, created, deleted, rollbackSnapshot)
			}
			modified = append(modified, action.rel+" -> "+action.newRel)
		}
	}
	_ = rollbackSnapshot.Commit()
	registry := fileTaskStateRegistry(t.cfg)
	for _, action := range actions {
		switch action.kind {
		case v4aOperationAdd, v4aOperationUpdate:
			_, _ = registry.Record(action.root, t.cfg.TaskID, action.cwd, action.rel, action.abs)
		case v4aOperationMove:
			_, _ = registry.Record(action.newRoot, t.cfg.TaskID, action.newCWD, action.newRel, action.newAbs)
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

func (t *PatchTool) v4aMutationLockPaths(ops []v4aPatchOperation) ([]string, error) {
	paths := make([]string, 0, len(ops)*2)
	for _, op := range ops {
		resolved, _, _, _, err := resolveFileTaskWritePath(t.cfg, op.path)
		if err != nil {
			return nil, err
		}
		paths = append(paths, resolved)
		if op.kind == v4aOperationMove {
			if strings.TrimSpace(op.newPath) == "" {
				continue
			}
			newResolved, _, _, _, err := resolveFileTaskWritePath(t.cfg, op.newPath)
			if err != nil {
				return nil, err
			}
			paths = append(paths, newResolved)
		}
	}
	return paths, nil
}

func v4aPatchSnapshotRoot(actions []v4aPatchAction) (string, error) {
	if len(actions) == 0 {
		return "", errors.New("patch contains no operations")
	}
	root := filepath.Clean(actions[0].root)
	if root == "." || root == "" {
		return "", errors.New("workspace root is required")
	}
	for _, action := range actions {
		if filepath.Clean(action.root) != root {
			return "", fmt.Errorf("operation %s uses different workspace root", action.rel)
		}
		if action.kind == v4aOperationMove && filepath.Clean(action.newRoot) != root {
			return "", fmt.Errorf("move destination %s uses different workspace root", action.newRel)
		}
	}
	return root, nil
}

func (t *PatchTool) validateV4AOperation(op v4aPatchOperation) (v4aPatchAction, *fileStateCheck, error) {
	action := v4aPatchAction{kind: op.kind}
	resolved, rel, root, cwd, err := resolveFileTaskWritePath(t.cfg, op.path)
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
		if check := registry.Check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
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
		if check := registry.Check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
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
		if check := registry.Check(root, t.cfg.TaskID, cwd, rel, resolved); check != nil {
			return action, check, nil
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return action, nil, fmt.Errorf("move file %s: stat source: %w", rel, err)
		}
		if info.IsDir() {
			return action, nil, fmt.Errorf("move file %s: expected a file, got a directory", rel)
		}
		newAbs, newRel, newRoot, newCWD, err := resolveFileTaskWritePath(t.cfg, op.newPath)
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
	payload := map[string]any{
		"path":           path,
		"status":         "patch_apply_failed",
		"success":        false,
		"error":          message,
		"files_modified": modified,
		"files_created":  created,
		"files_deleted":  deleted,
		"rolled_back":    false,
	}
	return marshalToolPayload(payload)
}

func marshalPatchApplyErrorWithRollback(path, message string, modified, created, deleted []string, snapshot *WorkspaceSnapshot) (json.RawMessage, error) {
	payload := map[string]any{
		"path":           path,
		"status":         "patch_apply_failed",
		"success":        false,
		"error":          message,
		"files_modified": modified,
		"files_created":  created,
		"files_deleted":  deleted,
	}
	if snapshot == nil {
		payload["rolled_back"] = false
		return marshalToolPayload(payload)
	}
	if err := snapshot.Restore(); err != nil {
		payload["rolled_back"] = false
		payload["rollback_error"] = err.Error()
		return marshalToolPayload(payload)
	}
	payload["rolled_back"] = true
	return marshalToolPayload(payload)
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
		next, err := applyV4AFuzzyHunk(updated, search, replacement, hunk.contextHint)
		if err != nil {
			return "", fmt.Errorf("could not apply hunk: %w", err)
		}
		updated = next
	}
	return updated, nil
}

func applyV4AFuzzyHunk(content, search, replacement, hint string) (string, error) {
	updated, _, err := fuzzyPatchReplace(content, search, replacement, false)
	if err == nil {
		return updated, nil
	}
	if strings.TrimSpace(hint) == "" {
		return "", err
	}
	hintPos := strings.Index(content, hint)
	if hintPos < 0 {
		return "", err
	}
	windowStart := hintPos - v4aContextHintBefore
	if windowStart < 0 {
		windowStart = 0
	}
	windowEnd := hintPos + v4aContextHintAfter
	if windowEnd > len(content) {
		windowEnd = len(content)
	}
	window := content[windowStart:windowEnd]
	windowUpdated, _, windowErr := fuzzyPatchReplace(window, search, replacement, false)
	if windowErr != nil {
		return "", err
	}
	return content[:windowStart] + windowUpdated + content[windowEnd:], nil
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
	case ".json", ".yaml", ".yml", ".toml", ".py":
		return true
	default:
		return false
	}
}

func supportsPostEditLint(path string) bool {
	if supportsStructuredLint(path) {
		return true
	}
	_, ok := shellLintSpecForPath(path)
	return ok
}

func postEditLSPDiagnostics(ctx context.Context, service lsp.PostEditService, path, relPath, postContent string, preContent *string) (lsp.PostEditResult, bool) {
	if service == nil {
		return lsp.PostEditResult{}, false
	}
	return service.PostEditDiagnostics(ctx, lsp.PostEditRequest{
		Path:         path,
		RelativePath: relPath,
		PreContent:   preContent,
		PostContent:  postContent,
	}), true
}

func postEditLintDelta(absPath, relPath, postContent string, preContent *string) (structuredLintResult, bool) {
	if lint, ok := structuredLintDelta(relPath, postContent, preContent); ok {
		return lint, true
	}
	return shellLintDelta(absPath, relPath)
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

func shellLintDelta(absPath, relPath string) (structuredLintResult, bool) {
	spec, ok := shellLintSpecForPath(relPath)
	if !ok {
		return structuredLintResult{}, false
	}
	binary, err := fileLintLookPath(spec.command)
	if err != nil {
		return structuredLintResult{
			Status:  "skipped",
			Success: false,
			Message: spec.command + " not available",
		}, true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result := fileLintRunCommand(ctx, binary, spec.argsFor(absPath)...)
	output := strings.TrimSpace(result.Output)
	if output == "" && result.Err != nil {
		output = result.Err.Error()
	}
	if result.ExitCode != 0 || result.Err != nil {
		return structuredLintResult{
			Status:  "error",
			Success: false,
			Output:  boundStructuredLintOutput(output),
		}, true
	}
	return structuredLintResult{Status: "ok", Success: true}, true
}

func shellLintSpecForPath(path string) (shellLintSpec, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js":
		return shellLintSpec{command: "node", args: []string{"--check"}}, true
	case ".ts":
		return shellLintSpec{command: "npx", args: []string{"tsc", "--noEmit"}}, true
	case ".go":
		return shellLintSpec{command: "go", args: []string{"vet"}}, true
	case ".rs":
		return shellLintSpec{command: "rustfmt", args: []string{"--check"}}, true
	default:
		return shellLintSpec{}, false
	}
}

func (s shellLintSpec) argsFor(path string) []string {
	args := make([]string, 0, len(s.args)+1)
	args = append(args, s.args...)
	args = append(args, path)
	return args
}

func runFileLintCommand(ctx context.Context, name string, args ...string) fileLintCommandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	result := fileLintCommandResult{Output: string(out)}
	if err == nil {
		return result
	}
	result.Err = err
	result.ExitCode = 1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	}
	if ctx.Err() != nil {
		result.Err = ctx.Err()
	}
	return result
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
	case ".py":
		return runPythonSyntaxLint(path, content)
	default:
		return structuredLintResult{}, false
	}
	if err != nil {
		return structuredLintResult{Success: false, Output: boundStructuredLintOutput(err.Error())}, true
	}
	return structuredLintResult{Success: true}, true
}

func runPythonSyntaxLint(path, content string) (structuredLintResult, bool) {
	binary := pythonSyntaxLintBinary()
	if binary == "" {
		return structuredLintResult{}, false
	}
	const script = `import ast
import sys

path = sys.argv[1] if len(sys.argv) > 1 else "<input>"
try:
    ast.parse(sys.stdin.read(), filename=path)
except SyntaxError as exc:
    loc = f" (line {exc.lineno}, column {exc.offset})" if exc.lineno else ""
    print(f"{type(exc).__name__}: {exc.msg}{loc}")
    sys.exit(1)
except Exception as exc:
    print(f"{type(exc).__name__}: {exc}")
    sys.exit(1)
`
	cmd := exec.Command(binary, "-c", script, path)
	cmd.Stdin = strings.NewReader(content)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return structuredLintResult{Success: false, Output: boundStructuredLintOutput(text)}, true
	}
	return structuredLintResult{Success: true}, true
}

func pythonSyntaxLintBinary() string {
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
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

func resolveFileTaskReadPath(cfg FileTaskToolConfig, rawPath string) (string, string, string, string, error) {
	return resolveFileTaskPathForAccess(cfg, rawPath, ProfileWorkspaceAccessRead)
}

func resolveFileTaskWritePath(cfg FileTaskToolConfig, rawPath string) (string, string, string, string, error) {
	return resolveFileTaskPathForAccess(cfg, rawPath, ProfileWorkspaceAccessWrite)
}

func resolveFileTaskPathForAccess(cfg FileTaskToolConfig, rawPath string, access ProfileWorkspaceAccess) (string, string, string, string, error) {
	if cfg.WorkspaceScope != nil {
		base, err := resolveProfileWorkspaceBase(cfg)
		if err != nil {
			return "", "", "", "", err
		}
		decision := cfg.WorkspaceScope.Resolve(rawPath, base, access)
		if !decision.Allowed {
			message := decision.Message
			if message == "" {
				message = "path is outside configured profile workspace roots"
			}
			return "", "", "", "", fmt.Errorf("%s: %s", decision.Evidence, message)
		}
		root := decision.Root
		if root == "" {
			root = cfg.WorkspaceScope.DefaultRoot()
		}
		cwdRel := "."
		if root != "" && pathWithinRoot(root, base) {
			cwdRel = workspaceRel(root, base)
		}
		return decision.Normalized, decision.Relative, root, cwdRel, nil
	}

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

func resolveProfileWorkspaceBase(cfg FileTaskToolConfig) (string, error) {
	raw := ""
	if cfg.CWDResolver != nil {
		raw = strings.TrimSpace(cfg.CWDResolver())
	} else if strings.TrimSpace(cfg.Root) == "" {
		raw = strings.TrimSpace(os.Getenv("TERMINAL_CWD"))
	}
	if raw == "" || terminalCWDPlaceholder(raw) {
		if cfg.WorkspaceScope != nil && !cfg.WorkspaceScope.Configured() {
			if cwd, err := os.Getwd(); err == nil {
				raw = cwd
			}
		}
	}
	if raw == "" || terminalCWDPlaceholder(raw) {
		raw = strings.TrimSpace(cfg.Root)
	}
	if raw == "" || terminalCWDPlaceholder(raw) {
		raw = cfg.WorkspaceScope.DefaultRoot()
	}
	if raw == "" {
		var err error
		raw, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve task cwd: %w", err)
		}
	}
	return normalizeWorkspacePath(raw, cfg.WorkspaceScope.DefaultRoot())
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
	return filesystem.ValidateWorkspaceRealPath(root, abs)
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
	return filesystem.WorkspaceRel(root, path)
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
