package tools

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestSearchFilesHiddenRootIncludesFiles(t *testing.T) {
	root := t.TempDir()
	paths := map[string]string{
		".hidden/note.txt":             "needle root\n",
		".hidden/nested/visible.txt":   "nested needle\n",
		".hidden/.secret/secret.txt":   "hidden needle\n",
		"visible/outside-hidden.txt":   "needle outside hidden root\n",
		"visible/.hidden/secret.txt":   "needle hidden descendant\n",
		".hidden/nested/other.bin":     "\x00needle",
		".hidden/nested/ignored.log":   "log without target term\n",
		".hidden/node_modules/mod.txt": "needle dependency\n",
	}
	for rel, content := range paths {
		writeSearchFixture(t, root, rel, content)
	}

	tool := NewSearchFilesTool(FileTaskToolConfig{Root: root})
	content := executeSearchFilesTool(t, tool, `{"pattern":"needle","target":"content","path":".hidden"}`)
	gotContent := searchMatchPaths(content)
	want := []string{".hidden/nested/visible.txt", ".hidden/note.txt"}
	if !reflect.DeepEqual(gotContent, want) {
		t.Fatalf("content search paths = %#v, want %#v; output=%#v", gotContent, want, content)
	}

	files := executeSearchFilesTool(t, tool, `{"pattern":"*.txt","target":"files","path":".hidden"}`)
	gotFiles := searchFileList(files)
	if !reflect.DeepEqual(gotFiles, want) {
		t.Fatalf("file search paths = %#v, want %#v; output=%#v", gotFiles, want, files)
	}
}

func TestSearchFilesHiddenRootStillRootConfined(t *testing.T) {
	root := t.TempDir()
	outsideRoot := t.TempDir()
	outsideHidden := filepath.Join(outsideRoot, ".hidden")
	if err := os.MkdirAll(outsideHidden, 0o755); err != nil {
		t.Fatalf("mkdir outside hidden: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideHidden, "secret.txt"), []byte("do not leak needle\n"), 0o644); err != nil {
		t.Fatalf("write outside hidden: %v", err)
	}

	tool := NewSearchFilesTool(FileTaskToolConfig{Root: root})
	absolute := executeSearchFilesTool(t, tool, `{"pattern":"needle","target":"content","path":`+quoteJSON(t, outsideHidden)+`}`)
	if !strings.Contains(asString(absolute["error"]), "outside workspace root") {
		t.Fatalf("absolute hidden error = %#v, want outside-root denial", absolute)
	}
	if searchPayloadContains(absolute, "do not leak") {
		t.Fatalf("absolute hidden search leaked outside content: %#v", absolute)
	}

	link := filepath.Join(root, "linked-hidden")
	if err := os.Symlink(outsideHidden, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	linked := executeSearchFilesTool(t, tool, `{"pattern":"needle","target":"content","path":"linked-hidden"}`)
	if !strings.Contains(asString(linked["error"]), "outside workspace root") {
		t.Fatalf("linked hidden error = %#v, want outside-root denial", linked)
	}
	if searchPayloadContains(linked, "do not leak") {
		t.Fatalf("linked hidden search leaked outside content: %#v", linked)
	}
}

func TestSearchFilesOffsetWindowReportsTruncatedWhenEarlierCandidatesWereSkipped(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"a.txt", "b.txt", "c.txt"} {
		writeSearchFixture(t, root, rel, "needle\n")
	}

	tool := NewSearchFilesTool(FileTaskToolConfig{Root: root})
	out := executeSearchFilesTool(t, tool, `{"pattern":"needle","target":"content","output_mode":"files_only","offset":2,"limit":1}`)
	got := searchFileList(out)
	want := []string{"c.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %#v, want %#v; output=%#v", got, want, out)
	}
	if out["truncated"] != true {
		t.Fatalf("truncated = %#v, want true when offset skips earlier matches; output=%#v", out["truncated"], out)
	}
}

func TestSearchFilesContentContextIncludesNeighborLines(t *testing.T) {
	root := t.TempDir()
	writeSearchFixture(t, root, "dir/file-12-name.py", "before context\nneedle line\nafter context\n")

	tool := NewSearchFilesTool(FileTaskToolConfig{Root: root})
	out := executeSearchFilesTool(t, tool, `{"pattern":"needle","target":"content","path":"dir","context":1}`)
	matches, _ := out["matches"].([]any)
	if len(matches) != 3 {
		t.Fatalf("matches = %#v, want before/match/after context lines; output=%#v", matches, out)
	}

	got := make([]string, 0, len(matches))
	for _, item := range matches {
		row, _ := item.(map[string]any)
		got = append(got, row["path"].(string)+":"+lineString(row["line"])+":"+row["text"].(string))
	}
	want := []string{
		"dir/file-12-name.py:1:before context",
		"dir/file-12-name.py:2:needle line",
		"dir/file-12-name.py:3:after context",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("context matches = %#v, want %#v", got, want)
	}
}

func TestSearchFilesContentLimitDoesNotReturnOnlyContext(t *testing.T) {
	root := t.TempDir()
	writeSearchFixture(t, root, "dir/file.txt", "before context\nneedle line\nafter context\n")

	tool := NewSearchFilesTool(FileTaskToolConfig{Root: root})
	out := executeSearchFilesTool(t, tool, `{"pattern":"needle","target":"content","path":"dir","context":1,"limit":1}`)
	matches, _ := out["matches"].([]any)
	if len(matches) != 1 {
		t.Fatalf("matches = %#v, want exactly one entry; output=%#v", matches, out)
	}
	row, _ := matches[0].(map[string]any)
	if row["text"] != "needle line" || lineString(row["line"]) != "2" {
		t.Fatalf("first limited content row = %#v, want the matched line rather than context-only output; output=%#v", row, out)
	}
	if out["truncated"] != true {
		t.Fatalf("truncated = %#v, want true for omitted context rows; output=%#v", out["truncated"], out)
	}
}

func TestSearchContentAccumulatorUpgradesContextRowWhenLineAlsoMatches(t *testing.T) {
	var acc searchContentAccumulator
	lines := []string{"first needle", "second needle"}

	acc.appendRange("dir/file.txt", lines, 0, 1)
	acc.appendRange("dir/file.txt", lines, 1, 1)

	if len(acc.entries) != 2 {
		t.Fatalf("entries = %#v, want each physical line once", acc.entries)
	}
	if !acc.entries[1].match {
		t.Fatalf("second entry = %#v, want duplicate context row upgraded to match", acc.entries[1])
	}
	window, _ := windowSearchContentEntries(acc.entries, 1, 1)
	if len(window) != 1 || !window[0].match || window[0].line != 2 {
		t.Fatalf("window = %#v, want offset page to retain match provenance", window)
	}
}

func TestSearchContextLineParserHyphenNumericFilename(t *testing.T) {
	path, line, text, ok := parseSearchContextLine("dir/file-12-name.py-8-context here")
	if !ok {
		t.Fatal("parseSearchContextLine returned ok=false")
	}
	if path != "dir/file-12-name.py" || line != 8 || text != "context here" {
		t.Fatalf("parsed = (%q, %d, %q), want (dir/file-12-name.py, 8, context here)", path, line, text)
	}
	if _, _, _, ok := parseSearchContextLine("--"); ok {
		t.Fatal("separator line parsed as context")
	}
}

func writeSearchFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func searchMatchPaths(out map[string]any) []string {
	matches, _ := out["matches"].([]any)
	paths := make([]string, 0, len(matches))
	for _, item := range matches {
		row, _ := item.(map[string]any)
		path, _ := row["path"].(string)
		if path != "" {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func searchFileList(out map[string]any) []string {
	raw, _ := out["files"].([]any)
	files := make([]string, 0, len(raw))
	for _, item := range raw {
		file, _ := item.(string)
		if file != "" {
			files = append(files, file)
		}
	}
	sort.Strings(files)
	return files
}

func searchPayloadContains(out map[string]any, needle string) bool {
	for _, value := range out {
		if strings.Contains(anyString(value), needle) {
			return true
		}
	}
	return false
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, anyString(item))
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, anyString(item))
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func lineString(value any) string {
	switch typed := value.(type) {
	case float64:
		return strconv.Itoa(int(typed))
	default:
		return ""
	}
}

// TestSearchPatternHasNewline verifies the detection of regex newline escapes
// (Hermes tools/file_operations.py _pattern_has_regex_newline parity).
func TestSearchPatternHasNewline(t *testing.T) {
	cases := []struct {
		pattern string
		want    bool
	}{
		{`\n`, true},          // single backslash-n: regex newline escape
		{`\\\n`, true},        // triple backslash-n: still an odd count before n (\\\ + n → 3 backslashes, odd)
		{`\\n`, false},        // double backslash-n: literal backslash+n, NOT a newline escape
		{"foo\nbar", true},    // literal newline in string
		{"foo", false},        // plain pattern, no newline
		{"bar\\\\n", false},   // four backslashes + n: even count, not a newline escape
		{`\n\n`, true},        // two newline escapes: first one triggers it
		{`a\nb`, true},        // embedded \n
	}
	for _, tc := range cases {
		got := searchPatternHasNewline(tc.pattern)
		if got != tc.want {
			t.Errorf("searchPatternHasNewline(%q) = %v, want %v", tc.pattern, got, tc.want)
		}
	}
}

// TestSearchFilesNewlinePatternWarning verifies that a line-oriented regex
// containing \n that returns zero results emits a diagnostic warning
// (Hermes tools/file_operations.py _maybe_warn_line_oriented_newline_pattern parity).
func TestSearchFilesNewlinePatternWarning(t *testing.T) {
	root := t.TempDir()
	writeSearchFixture(t, root, "a.txt", "hello world\ngoodbye world\n")
	tool := NewSearchFilesTool(FileTaskToolConfig{Root: root})

	for _, mode := range []string{"content", "files_only", "count"} {
		args := `{"pattern":"hello\ngoodbye","target":"content","output_mode":"` + mode + `"}`
		out := executeSearchFilesTool(t, tool, args)
		warning, _ := out["warning"].(string)
		if warning == "" {
			t.Errorf("output_mode=%s: expected warning for \\n pattern with 0 results, got %#v", mode, out)
			continue
		}
		if !strings.Contains(warning, "line-oriented") {
			t.Errorf("output_mode=%s: warning %q does not mention line-oriented", mode, warning)
		}
		if _, hasErr := out["error"]; hasErr {
			t.Errorf("output_mode=%s: unexpected error field in output: %#v", mode, out)
		}
	}

	// A literal backslash+n pattern (\\n) should NOT trigger the warning.
	outNoWarn := executeSearchFilesTool(t, tool, `{"pattern":"\\\\n","target":"content"}`)
	if _, hasWarn := outNoWarn["warning"]; hasWarn {
		t.Errorf("escaped \\\\n pattern triggered spurious newline warning: %#v", outNoWarn)
	}
}
