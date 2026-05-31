package fragments

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const promptFragmentErrorCode = "prompt_fragment_error"

var (
	promptFragmentIncludePattern  = regexp.MustCompile(`\{\{\s*include\s+((?:"[^"]+")|(?:'[^']+')|original|[^\s{}]+)\s*\}\}`)
	promptFragmentVariablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
)

type PromptFragmentSource struct {
	Name string
	Dir  string
}

type PromptFragmentRequest struct {
	Entry     string
	Sources   []PromptFragmentSource
	Variables map[string]string
	Cache     *PromptFragmentCache
}

type PromptFragmentResult struct {
	Text      string
	Fragments []PromptFragmentEvidence
}

type PromptFragmentEvidence struct {
	Fragment string
	Source   string
	Path     string
}

type PromptFragmentError struct {
	Fragment string
	Reason   string
	Chain    []string
	Err      error
}

func (e *PromptFragmentError) Error() string {
	var b strings.Builder
	b.WriteString(promptFragmentErrorCode)
	if e.Reason != "" {
		b.WriteString(": ")
		b.WriteString(e.Reason)
	}
	if len(e.Chain) > 0 {
		b.WriteString(": ")
		b.WriteString(strings.Join(e.Chain, " -> "))
	} else if e.Fragment != "" {
		b.WriteString(": ")
		b.WriteString(e.Fragment)
	}
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}
	return b.String()
}

func (e *PromptFragmentError) Unwrap() error { return e.Err }

type PromptFragmentCache struct {
	mu    sync.Mutex
	files map[string]promptFragmentCacheEntry
}

type promptFragmentCacheEntry struct {
	modTime time.Time
	size    int64
	text    string
}

func NewPromptFragmentCache() *PromptFragmentCache {
	return &PromptFragmentCache{files: map[string]promptFragmentCacheEntry{}}
}

func RenderPromptFragment(req PromptFragmentRequest) (PromptFragmentResult, error) {
	entry, err := cleanPromptFragmentName(req.Entry)
	if err != nil {
		return PromptFragmentResult{}, &PromptFragmentError{Fragment: req.Entry, Reason: "invalid fragment", Err: err}
	}
	if len(req.Sources) == 0 {
		return PromptFragmentResult{}, &PromptFragmentError{Fragment: entry, Reason: "no fragment sources configured"}
	}

	r := promptFragmentRenderer{
		sources:   normalizePromptFragmentSources(req.Sources),
		variables: req.Variables,
		cache:     req.Cache,
	}
	text, err := r.render(entry, 0, nil)
	if err != nil {
		return PromptFragmentResult{}, err
	}
	return PromptFragmentResult{Text: text, Fragments: r.fragments}, nil
}

type promptFragmentRenderer struct {
	sources   []PromptFragmentSource
	variables map[string]string
	cache     *PromptFragmentCache
	fragments []PromptFragmentEvidence
}

type promptFragmentResolved struct {
	name        string
	sourceName  string
	sourceIndex int
	path        string
}

type promptFragmentFrame struct {
	name string
	path string
}

func (r *promptFragmentRenderer) render(name string, startSource int, stack []promptFragmentFrame) (string, error) {
	cleanName, err := cleanPromptFragmentName(name)
	if err != nil {
		return "", &PromptFragmentError{Fragment: name, Reason: "invalid fragment", Err: err}
	}
	resolved, ok := r.find(cleanName, startSource)
	if !ok {
		return "", &PromptFragmentError{Fragment: cleanName, Reason: "missing fragment", Chain: promptFragmentChain(stack, cleanName)}
	}
	for _, frame := range stack {
		if frame.path == resolved.path {
			return "", &PromptFragmentError{
				Fragment: cleanName,
				Reason:   "circular include",
				Chain:    promptFragmentChain(stack, cleanName),
			}
		}
	}

	text, err := r.read(resolved.path)
	if err != nil {
		return "", &PromptFragmentError{Fragment: cleanName, Reason: "read fragment", Chain: promptFragmentChain(stack, cleanName), Err: err}
	}
	r.fragments = append(r.fragments, PromptFragmentEvidence{
		Fragment: resolved.name,
		Source:   resolved.sourceName,
		Path:     resolved.path,
	})

	stack = append(stack, promptFragmentFrame{name: resolved.name, path: resolved.path})
	text, err = r.expandIncludes(text, resolved, stack)
	if err != nil {
		return "", err
	}
	return r.replaceVariables(text), nil
}

func (r *promptFragmentRenderer) expandIncludes(text string, current promptFragmentResolved, stack []promptFragmentFrame) (string, error) {
	matches := promptFragmentIncludePattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil
	}

	var out strings.Builder
	last := 0
	for _, m := range matches {
		out.WriteString(text[last:m[0]])
		token := strings.TrimSpace(text[m[2]:m[3]])
		includeName := strings.Trim(token, `"'`)

		var rendered string
		var err error
		if includeName == "original" {
			if _, ok := r.find(current.name, current.sourceIndex+1); !ok {
				rendered = ""
			} else {
				rendered, err = r.render(current.name, current.sourceIndex+1, stack)
			}
		} else {
			rendered, err = r.render(includeName, 0, stack)
		}
		if err != nil {
			return "", err
		}
		out.WriteString(rendered)
		last = m[1]
	}
	out.WriteString(text[last:])
	return out.String(), nil
}

func (r *promptFragmentRenderer) find(name string, startSource int) (promptFragmentResolved, bool) {
	for i := startSource; i < len(r.sources); i++ {
		source := r.sources[i]
		path := filepath.Join(source.Dir, filepath.FromSlash(name))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return promptFragmentResolved{
				name:        name,
				sourceName:  source.Name,
				sourceIndex: i,
				path:        path,
			}, true
		}
	}
	return promptFragmentResolved{}, false
}

func (r *promptFragmentRenderer) read(path string) (string, error) {
	if r.cache == nil {
		data, err := os.ReadFile(path)
		return string(data), err
	}
	return r.cache.read(path)
}

func (r *promptFragmentRenderer) replaceVariables(text string) string {
	if len(r.variables) == 0 {
		return text
	}
	return promptFragmentVariablePattern.ReplaceAllStringFunc(text, func(token string) string {
		matches := promptFragmentVariablePattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		if value, ok := r.variables[matches[1]]; ok {
			return value
		}
		return token
	})
}

func (c *PromptFragmentCache) read(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.files[path]; ok && cached.modTime.Equal(info.ModTime()) && cached.size == info.Size() {
		return cached.text, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(data)
	if c.files == nil {
		c.files = map[string]promptFragmentCacheEntry{}
	}
	c.files[path] = promptFragmentCacheEntry{modTime: info.ModTime(), size: info.Size(), text: text}
	return text, nil
}

func cleanPromptFragmentName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty fragment name")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("absolute fragment path %q", name)
	}
	name = filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
	if name == "." || name == ".." || strings.HasPrefix(name, "../") {
		return "", fmt.Errorf("unsafe fragment path %q", name)
	}
	return name, nil
}

func normalizePromptFragmentSources(sources []PromptFragmentSource) []PromptFragmentSource {
	out := make([]PromptFragmentSource, 0, len(sources))
	for i, source := range sources {
		dir := strings.TrimSpace(source.Dir)
		if dir == "" {
			continue
		}
		name := strings.TrimSpace(source.Name)
		if name == "" {
			name = fmt.Sprintf("source-%d", i)
		}
		out = append(out, PromptFragmentSource{Name: name, Dir: dir})
	}
	return out
}

func promptFragmentChain(stack []promptFragmentFrame, tail string) []string {
	chain := make([]string, 0, len(stack)+1)
	for _, frame := range stack {
		chain = append(chain, frame.name)
	}
	if tail != "" {
		chain = append(chain, tail)
	}
	return chain
}
