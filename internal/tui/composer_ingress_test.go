package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIComposerDetectsDroppedFiles(t *testing.T) {
	tmp := t.TempDir()
	image := writeComposerIngressFile(t, tmp, "Screenshot 2026-04-21 at 1.04.43 PM.png", []byte("fake"))
	text := writeComposerIngressFile(t, tmp, "main.py", []byte("print('hello')\n"))
	noExt := writeComposerIngressFile(t, tmp, "Makefile", []byte("all:\n\techo hi\n"))
	link := filepath.Join(tmp, "link.png")
	if err := os.Symlink(image, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	home := filepath.Join(tmp, "home")
	homeImage := writeComposerIngressFile(t, home, "storage/shared/Pictures/cat.png", []byte("fake"))

	tests := []struct {
		name          string
		input         string
		wantMatched   bool
		wantPath      string
		wantImage     bool
		wantRemainder string
	}{
		{
			name:        "absolute image path",
			input:       image,
			wantMatched: true,
			wantPath:    image,
			wantImage:   true,
		},
		{
			name:          "image path with trailing text",
			input:         image + " analyze this",
			wantMatched:   true,
			wantPath:      image,
			wantImage:     true,
			wantRemainder: "analyze this",
		},
		{
			name:        "escaped spaces",
			input:       filepath.Dir(image) + string(os.PathSeparator) + "Screenshot\\ 2026-04-21\\ at\\ 1.04.43\\ PM.png",
			wantMatched: true,
			wantPath:    image,
			wantImage:   true,
		},
		{
			name:        "file URI",
			input:       pathToFileURI(t, image),
			wantMatched: true,
			wantPath:    image,
			wantImage:   true,
		},
		{
			name:          "tilde path with trailing text",
			input:         "~/storage/shared/Pictures/cat.png what is this?",
			wantMatched:   true,
			wantPath:      homeImage,
			wantImage:     true,
			wantRemainder: "what is this?",
		},
		{
			name:          "non image file",
			input:         text + " review this",
			wantMatched:   true,
			wantPath:      text,
			wantRemainder: "review this",
		},
		{
			name:        "path with no extension",
			input:       noExt,
			wantMatched: true,
			wantPath:    noExt,
		},
		{
			name:        "symlink to image",
			input:       link,
			wantMatched: true,
			wantPath:    link,
			wantImage:   true,
		},
		{name: "slash command", input: "/help"},
		{name: "nonexistent path", input: filepath.Join(tmp, "missing.png")},
		{name: "directory", input: tmp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectComposerDroppedFile(tt.input, ComposerDropOptions{HomeDir: home})
			if got.Matched != tt.wantMatched {
				t.Fatalf("Matched = %v, want %v (result=%+v)", got.Matched, tt.wantMatched, got)
			}
			if !tt.wantMatched {
				return
			}
			if got.Path != tt.wantPath {
				t.Fatalf("Path = %q, want %q", got.Path, tt.wantPath)
			}
			if got.IsImage != tt.wantImage {
				t.Fatalf("IsImage = %v, want %v", got.IsImage, tt.wantImage)
			}
			if got.Remainder != tt.wantRemainder {
				t.Fatalf("Remainder = %q, want %q", got.Remainder, tt.wantRemainder)
			}
		})
	}
}

func TestTUIComposerPasteCollapse(t *testing.T) {
	longPaste := strings.Repeat("alpha beta gamma\n", 85)
	var collapsedText string

	got := CollapseComposerPaste(longPaste, ComposerPasteOptions{
		Collapse: func(text string) (string, error) {
			collapsedText = text
			return "/tmp/paste-1.txt", nil
		},
	})
	if collapsedText == "" {
		t.Fatal("Collapse seam was not called for long paste")
	}
	if !strings.Contains(got.InsertText, "[85 lines]") {
		t.Fatalf("InsertText = %q, want line-count label", got.InsertText)
	}
	if got.Snippet == nil || got.Snippet.Path != "/tmp/paste-1.txt" || got.Snippet.Text != strings.TrimRight(longPaste, "\n") {
		t.Fatalf("Snippet = %+v, want collapsed path and trimmed text", got.Snippet)
	}
	if strings.Contains(got.InsertText, "/tmp/paste-1.txt") {
		t.Fatalf("InsertText leaked paste temp path: %q", got.InsertText)
	}

	failed := CollapseComposerPaste(longPaste, ComposerPasteOptions{
		Collapse: func(string) (string, error) { return "", errors.New("disk full") },
	})
	if failed.Evidence != "tui_ingress_paste_collapse_failed" {
		t.Fatalf("Evidence = %q, want paste-collapse failure evidence", failed.Evidence)
	}
	if failed.Snippet == nil || failed.Snippet.Path != "" {
		t.Fatalf("failed Snippet = %+v, want in-memory snippet without path", failed.Snippet)
	}
}

func TestTUIComposerExternalEditorExpandsPasteSnippets(t *testing.T) {
	expanded, err := ExpandComposerPasteSnippets(
		"before [[ alpha [2 lines] ]] after",
		[]ComposerPasteSnippet{{Label: "[[ alpha [2 lines] ]]", Text: "line one\nline two"}},
		nil,
	)
	if err != nil {
		t.Fatalf("ExpandComposerPasteSnippets() error = %v", err)
	}
	if expanded != "before line one\nline two after" {
		t.Fatalf("expanded = %q", expanded)
	}
}

func TestTUIComposerBracketedPasteRecovery(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "complete", in: "\x1b[200~alpha\nbeta\x1b[201~", want: "alpha\nbeta"},
		{name: "unterminated", in: "\x1b[200~alpha", want: "alpha"},
		{name: "empty", in: "\x1b[200~\x1b[201~", want: ""},
		{name: "plain text", in: "alpha", want: "alpha"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RecoverComposerBracketedPaste(tt.in); got != tt.want {
				t.Fatalf("RecoverComposerBracketedPaste() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTUIComposerCopyAssistantOutput(t *testing.T) {
	history := []llm.Message{
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "one"},
		{Role: "assistant", Content: "<think>hidden</think>Visible answer"},
	}

	latest := SelectComposerCopyText(history, "")
	if !latest.OK || latest.Text != "Visible answer" || latest.ResponseNumber != 2 {
		t.Fatalf("latest copy = %+v, want visible latest assistant response #2", latest)
	}

	first := SelectComposerCopyText(history, "1")
	if !first.OK || first.Text != "one" || first.ResponseNumber != 1 {
		t.Fatalf("indexed copy = %+v, want first assistant response", first)
	}

	invalid := SelectComposerCopyText(history, "99")
	if invalid.OK || invalid.Evidence != "tui_ingress_copy_invalid_index" {
		t.Fatalf("invalid copy = %+v, want invalid-index evidence", invalid)
	}
}

func TestTUIComposerCopySlashUsesInjectedClipboard(t *testing.T) {
	var copied string
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		ClipboardWrite: func(text string) error {
			copied = text
			return nil
		},
	})
	m.frame.History = []llm.Message{
		{Role: "assistant", Content: "one"},
		{Role: "assistant", Content: "<reasoning>hidden</reasoning>Visible answer"},
	}
	m.editor.SetValue("/copy")

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)
	if copied != "Visible answer" {
		t.Fatalf("copied = %q, want visible assistant answer", copied)
	}
	if updated.editor.Value() != "" {
		t.Fatalf("editor value = %q, want reset", updated.editor.Value())
	}
	if !strings.Contains(updated.statusMessage, "Copied assistant response #2") {
		t.Fatalf("statusMessage = %q, want copy confirmation", updated.statusMessage)
	}
}

func writeComposerIngressFile(t *testing.T, root, rel string, body []byte) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func pathToFileURI(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs %s: %v", path, err)
	}
	return "file://" + filepath.ToSlash(abs)
}
