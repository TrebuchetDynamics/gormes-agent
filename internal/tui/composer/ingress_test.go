package composer

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestDetectComposerDroppedFileAcceptsQuotedRelativePaths(t *testing.T) {
	wantPath := filepath.Clean("./quoted name.txt")
	var statPath string
	got := DetectComposerDroppedFile(`"./quoted name.txt" describe it`, ComposerDropOptions{
		Stat: func(path string) (fs.FileInfo, error) {
			statPath = path
			if path == wantPath {
				return fakeComposerFileInfo{name: filepath.Base(path)}, nil
			}
			return nil, fs.ErrNotExist
		},
	})

	if !got.Matched || got.Path != wantPath || got.Remainder != "describe it" {
		t.Fatalf("DetectComposerDroppedFile quoted relative = %+v (statPath=%q), want matched path %q with remainder", got, statPath, wantPath)
	}
}

func TestCollapseComposerPasteLabelPreservesUTF8(t *testing.T) {
	text := "a" + strings.Repeat("界", 30) + "\n" + strings.Repeat("emoji🙂", 30)

	got := CollapseComposerPaste(text, ComposerPasteOptions{LargePasteChars: 10, LargePasteLines: 99})
	if got.Snippet == nil {
		t.Fatal("CollapseComposerPaste did not create snippet for large paste")
	}
	if !utf8.ValidString(got.InsertText) {
		t.Fatalf("CollapseComposerPaste label is invalid UTF-8: %q", got.InsertText)
	}
	if got.Snippet.Text != text {
		t.Fatal("CollapseComposerPaste snippet text changed while making label")
	}
}

type fakeComposerFileInfo struct {
	name string
}

func (f fakeComposerFileInfo) Name() string       { return f.name }
func (f fakeComposerFileInfo) Size() int64        { return 1 }
func (f fakeComposerFileInfo) Mode() fs.FileMode  { return 0 }
func (f fakeComposerFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeComposerFileInfo) IsDir() bool        { return false }
func (f fakeComposerFileInfo) Sys() any           { return nil }
