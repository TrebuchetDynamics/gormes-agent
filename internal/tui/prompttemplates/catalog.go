// Package prompttemplates discovers and expands Gormes-owned Markdown prompt templates.
package prompttemplates

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Template is one operator-authored Markdown prompt template.
type Template struct {
	Name         string
	Description  string
	ArgumentHint string
	Body         string
	Source       string
}

// Evidence records a skipped template/root without leaking absolute paths.
type Evidence struct {
	Code   string
	Name   string
	Path   string
	Reason string
}

// Catalog is the deterministic result of prompt-template discovery.
type Catalog struct {
	Templates []Template
	Skipped   []Evidence
}

// DiscoverOptions controls prompt-template discovery.
type DiscoverOptions struct {
	Roots         []string
	ReservedNames []string
}

const (
	EvidenceSkipped = "prompt_template_skipped"
)

var (
	safeNameInvalid = regexp.MustCompile(`[^a-z0-9-]+`)
	safeNameHyphens = regexp.MustCompile(`-+`)
	sliceExprRE     = regexp.MustCompile(`\$\{@:([0-9]+)(?::([0-9]+))?\}`)
	positionalRE    = regexp.MustCompile(`\$([0-9]+)\b`)
)

// Discover scans prompt template roots non-recursively. Earlier roots win.
// Each root may be either a directory of *.md files or one explicit .md file.
func Discover(opts DiscoverOptions) (Catalog, error) {
	reserved := make(map[string]struct{}, len(opts.ReservedNames))
	for _, name := range opts.ReservedNames {
		if normalized := normalizeName(name); normalized != "" {
			reserved[normalized] = struct{}{}
		}
	}

	var catalog Catalog
	seen := map[string]struct{}{}
	for _, root := range opts.Roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				catalog.Skipped = append(catalog.Skipped, Evidence{Code: EvidenceSkipped, Path: filepath.Base(root), Reason: "root_missing"})
				continue
			}
			catalog.Skipped = append(catalog.Skipped, Evidence{Code: EvidenceSkipped, Path: filepath.Base(root), Reason: "root_unreadable"})
			continue
		}
		if !info.IsDir() {
			addTemplateFile(&catalog, root, filepath.Base(root), reserved, seen)
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			catalog.Skipped = append(catalog.Skipped, Evidence{Code: EvidenceSkipped, Path: filepath.Base(root), Reason: "root_unreadable"})
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
				continue
			}
			addTemplateFile(&catalog, filepath.Join(root, entry.Name()), entry.Name(), reserved, seen)
		}
	}
	sort.SliceStable(catalog.Templates, func(i, j int) bool { return catalog.Templates[i].Name < catalog.Templates[j].Name })
	return catalog, nil
}

func addTemplateFile(catalog *Catalog, path, displayPath string, reserved, seen map[string]struct{}) {
	if strings.ToLower(filepath.Ext(displayPath)) != ".md" {
		catalog.Skipped = append(catalog.Skipped, Evidence{Code: EvidenceSkipped, Path: displayPath, Reason: "not_markdown"})
		return
	}
	body, err := os.ReadFile(path)
	if err != nil {
		catalog.Skipped = append(catalog.Skipped, Evidence{Code: EvidenceSkipped, Path: displayPath, Reason: "file_unreadable"})
		return
	}
	name := normalizeName(strings.TrimSuffix(filepath.Base(displayPath), filepath.Ext(displayPath)))
	if name == "" {
		catalog.Skipped = append(catalog.Skipped, Evidence{Code: EvidenceSkipped, Path: displayPath, Reason: "unsafe_name"})
		return
	}
	if _, ok := reserved[name]; ok {
		catalog.Skipped = append(catalog.Skipped, Evidence{Code: EvidenceSkipped, Name: name, Path: displayPath, Reason: "reserved_name"})
		return
	}
	if _, ok := seen[name]; ok {
		catalog.Skipped = append(catalog.Skipped, Evidence{Code: EvidenceSkipped, Name: name, Path: displayPath, Reason: "duplicate_name"})
		return
	}
	front, content := splitFrontmatter(string(body))
	tmpl := Template{
		Name:         name,
		Description:  strings.TrimSpace(front.Description),
		ArgumentHint: strings.TrimSpace(front.ArgumentHint),
		Body:         strings.TrimSpace(content),
		Source:       displayPath,
	}
	if tmpl.Description == "" {
		tmpl.Description = firstDescriptionLine(tmpl.Body)
	}
	catalog.Templates = append(catalog.Templates, tmpl)
	seen[name] = struct{}{}
}

// Lookup finds a template by safe name. A leading slash is accepted.
func (c Catalog) Lookup(name string) (Template, bool) {
	name = normalizeName(strings.TrimPrefix(strings.TrimSpace(name), "/"))
	for _, tmpl := range c.Templates {
		if tmpl.Name == name {
			return tmpl, true
		}
	}
	return Template{}, false
}

func splitFrontmatter(text string) (struct {
	Description  string `yaml:"description"`
	ArgumentHint string `yaml:"argument-hint"`
}, string) {
	var front struct {
		Description  string `yaml:"description"`
		ArgumentHint string `yaml:"argument-hint"`
	}
	trimmed := strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(trimmed, "---\n") && !strings.HasPrefix(trimmed, "---\r\n") {
		return front, text
	}
	lines := strings.Split(trimmed, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(strings.TrimSuffix(lines[i], "\r")) != "---" {
			continue
		}
		raw := strings.Join(lines[1:i], "\n")
		_ = yaml.Unmarshal([]byte(raw), &front)
		return front, strings.Join(lines[i+1:], "\n")
	}
	return front, text
}

func firstDescriptionLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if len([]rune(line)) > 80 {
			return string([]rune(line)[:80])
		}
		return line
	}
	return ""
}

func normalizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = safeNameInvalid.ReplaceAllString(value, "-")
	value = safeNameHyphens.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

// ParseInvocation parses a slash-template invocation into a name and arguments.
func ParseInvocation(input string) (name string, args []string, ok bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", nil, false
	}
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil, false
	}
	name = normalizeName(strings.TrimPrefix(parts[0], "/"))
	if name == "" {
		return "", nil, false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(input, parts[0]))
	return name, ParseArgs(rest), true
}

// ParseArgs splits prompt-template arguments with small, deterministic shell-like quote handling.
func ParseArgs(input string) []string {
	var args []string
	var b strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		args = append(args, b.String())
		b.Reset()
	}
	for _, r := range input {
		if escaped {
			b.WriteRune(r)
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
			b.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t', '\n', '\r':
			flush()
		default:
			b.WriteRune(r)
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	flush()
	return args
}

// Expand applies Pi-style positional substitutions to a template body.
func Expand(t Template, args []string) string {
	out := t.Body
	joined := strings.Join(args, " ")
	out = sliceExprRE.ReplaceAllStringFunc(out, func(expr string) string {
		m := sliceExprRE.FindStringSubmatch(expr)
		if len(m) < 2 {
			return expr
		}
		start, err := strconv.Atoi(m[1])
		if err != nil || start < 1 {
			return expr
		}
		idx := start - 1
		if idx >= len(args) {
			return ""
		}
		end := len(args)
		if len(m) > 2 && m[2] != "" {
			ln, err := strconv.Atoi(m[2])
			if err != nil || ln < 0 {
				return expr
			}
			end = idx + ln
			if end > len(args) {
				end = len(args)
			}
		}
		return strings.Join(args[idx:end], " ")
	})
	out = strings.ReplaceAll(out, "$ARGUMENTS", joined)
	out = strings.ReplaceAll(out, "$@", joined)
	out = positionalRE.ReplaceAllStringFunc(out, func(expr string) string {
		m := positionalRE.FindStringSubmatch(expr)
		if len(m) != 2 {
			return expr
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil || idx < 1 || idx > len(args) {
			return ""
		}
		return args[idx-1]
	})
	return strings.TrimSpace(out)
}
