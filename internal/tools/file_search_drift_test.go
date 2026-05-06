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
