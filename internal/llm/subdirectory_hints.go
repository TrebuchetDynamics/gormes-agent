package llm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	subdirectoryHintsDefaultMaxChars     = 8000
	subdirectoryHintsDefaultAncestorWalk = 5
	SubdirectoryHintEvidenceLoaded       = "subdirectory_hint_loaded"
	SubdirectoryHintEvidenceStatError    = "subdirectory_hint_stat_error"
	SubdirectoryHintEvidenceReadError    = "subdirectory_hint_read_error"
	SubdirectoryHintEvidenceEmpty        = "subdirectory_hint_empty"
)

var (
	subdirectoryHintFilenames = []string{"AGENTS.md", "agents.md", "CLAUDE.md", "claude.md", ".cursorrules"}
	subdirectoryHintPathKeys  = []string{"path", "file_path", "workdir"}
)

type SubdirectoryHintOptions struct {
	WorkingDir      string
	MaxChars        int
	MaxAncestorWalk int
	Stat            func(string) (os.FileInfo, error)
	ReadFile        func(string) ([]byte, error)
}

type SubdirectoryHintTracker struct {
	workingDir      string
	maxChars        int
	maxAncestorWalk int
	loadedDirs      map[string]struct{}
	stat            func(string) (os.FileInfo, error)
	readFile        func(string) ([]byte, error)
}

type SubdirectoryHintResult struct {
	Text     string
	Hints    []SubdirectoryHint
	Evidence []SubdirectoryHintEvidence
}

type SubdirectoryHint struct {
	Path     string
	RelPath  string
	Content  string
	Evidence ContextFileEvidence
}

type SubdirectoryHintEvidence struct {
	Code    string
	Path    string
	Message string
}

func NewSubdirectoryHintTracker(opts SubdirectoryHintOptions) *SubdirectoryHintTracker {
	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workingDir = wd
		}
	}
	workingDir, _ = filepath.Abs(workingDir)
	maxChars := opts.MaxChars
	if maxChars <= 0 {
		maxChars = subdirectoryHintsDefaultMaxChars
	}
	maxAncestorWalk := opts.MaxAncestorWalk
	if maxAncestorWalk <= 0 {
		maxAncestorWalk = subdirectoryHintsDefaultAncestorWalk
	}
	stat := opts.Stat
	if stat == nil {
		stat = os.Stat
	}
	readFile := opts.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}
	return &SubdirectoryHintTracker{
		workingDir:      workingDir,
		maxChars:        maxChars,
		maxAncestorWalk: maxAncestorWalk,
		loadedDirs:      map[string]struct{}{filepath.Clean(workingDir): {}},
		stat:            stat,
		readFile:        readFile,
	}
}

func (t *SubdirectoryHintTracker) CheckToolCall(toolName string, args map[string]any) SubdirectoryHintResult {
	if t == nil {
		return SubdirectoryHintResult{}
	}
	dirs := t.extractDirectories(toolName, args)
	if len(dirs) == 0 {
		return SubdirectoryHintResult{}
	}
	var result SubdirectoryHintResult
	for _, dir := range dirs {
		hint, evidence, ok := t.loadHintForDirectory(dir)
		result.Evidence = append(result.Evidence, evidence...)
		if ok {
			result.Hints = append(result.Hints, hint)
		}
	}
	if len(result.Hints) == 0 {
		return result
	}
	sections := make([]string, 0, len(result.Hints))
	for _, hint := range result.Hints {
		sections = append(sections, fmt.Sprintf("[Subdirectory context discovered: %s]\n%s", hint.RelPath, hint.Content))
	}
	result.Text = "\n\n" + strings.Join(sections, "\n\n")
	return result
}

func (t *SubdirectoryHintTracker) extractDirectories(toolName string, args map[string]any) []string {
	var dirs []string
	seen := map[string]struct{}{}
	addRaw := func(raw string) {
		for _, dir := range t.directoriesForPath(raw) {
			if _, ok := seen[dir]; ok {
				continue
			}
			seen[dir] = struct{}{}
			dirs = append(dirs, dir)
		}
	}
	for _, key := range subdirectoryHintPathKeys {
		if value, ok := args[key].(string); ok {
			addRaw(value)
		}
	}
	if toolName == "terminal" {
		if command, ok := args["command"].(string); ok {
			for _, token := range shellLikeFields(command) {
				if shouldSkipSubdirectoryHintToken(token) {
					continue
				}
				addRaw(token)
			}
		}
	}
	return dirs
}

func (t *SubdirectoryHintTracker) directoriesForPath(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || shouldSkipSubdirectoryHintToken(raw) {
		return nil
	}
	path := raw
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.workingDir, path)
	}
	path = filepath.Clean(path)
	info, err := t.stat(path)
	if err == nil && !info.IsDir() {
		path = filepath.Dir(path)
	} else if err != nil && filepath.Ext(path) != "" {
		path = filepath.Dir(path)
	}

	var dirs []string
	current := path
	for i := 0; i < t.maxAncestorWalk; i++ {
		current = filepath.Clean(current)
		if _, loaded := t.loadedDirs[current]; loaded {
			break
		}
		if info, err := t.stat(current); err == nil && info.IsDir() {
			dirs = append(dirs, current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return dirs
}

func (t *SubdirectoryHintTracker) loadHintForDirectory(dir string) (SubdirectoryHint, []SubdirectoryHintEvidence, bool) {
	dir = filepath.Clean(dir)
	t.loadedDirs[dir] = struct{}{}
	for _, filename := range subdirectoryHintFilenames {
		path := filepath.Join(dir, filename)
		info, err := t.stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return SubdirectoryHint{}, []SubdirectoryHintEvidence{{
				Code:    SubdirectoryHintEvidenceStatError,
				Path:    t.displayPath(path),
				Message: err.Error(),
			}}, false
		}
		if info.IsDir() {
			continue
		}
		data, err := t.readFile(path)
		if err != nil {
			return SubdirectoryHint{}, []SubdirectoryHintEvidence{{
				Code:    SubdirectoryHintEvidenceReadError,
				Path:    t.displayPath(path),
				Message: err.Error(),
			}}, false
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return SubdirectoryHint{}, []SubdirectoryHintEvidence{{
				Code: SubdirectoryHintEvidenceEmpty,
				Path: t.displayPath(path),
			}}, false
		}
		ev := ContextFileEvidence{
			Source:         filename,
			Path:           path,
			Loaded:         true,
			OriginalLength: len([]rune(content)),
		}
		content, ev = scanContextContent(content, filename, ev)
		if !ev.Blocked {
			content, ev = truncateContextContent(content, filename, t.maxChars, ev)
		}
		relPath := t.displayPath(path)
		hint := SubdirectoryHint{
			Path:     path,
			RelPath:  relPath,
			Content:  content,
			Evidence: ev,
		}
		evidence := []SubdirectoryHintEvidence{{
			Code: SubdirectoryHintEvidenceLoaded,
			Path: relPath,
		}}
		return hint, evidence, true
	}
	return SubdirectoryHint{}, nil, false
}

func (t *SubdirectoryHintTracker) displayPath(path string) string {
	if rel, err := filepath.Rel(t.workingDir, path); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func shouldSkipSubdirectoryHintToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" || strings.HasPrefix(token, "-") {
		return true
	}
	lower := strings.ToLower(token)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(token, "git@") || strings.Contains(lower, "://") {
		return true
	}
	return false
}

func shellLikeFields(command string) []string {
	var fields []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' {
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	sort.Strings(fields)
	return fields
}
