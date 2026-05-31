package media

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareMediaContentExtractsHermesTags(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "voice.ogg")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}

	content := PrepareMediaContent("Here is the audio.\n[[audio_as_voice]]\nMEDIA:" + audioPath + "\nDone.")
	if strings.Contains(content.Text, "MEDIA:") || strings.Contains(content.Text, "audio_as_voice") {
		t.Fatalf("cleaned text leaked media tag: %q", content.Text)
	}
	if content.Text != "Here is the audio.\nDone." {
		t.Fatalf("Text = %q, want media line stripped", content.Text)
	}
	if len(content.Media) != 1 || content.Media[0].Path != audioPath || !content.Media[0].AsVoice {
		t.Fatalf("Media = %+v, want one voice attachment", content.Media)
	}
}

func TestPrepareMediaContentExtractsImageDocumentVideoTags(t *testing.T) {
	dir := t.TempDir()
	cases := []string{
		"photo.png",
		"photo.jpg",
		"photo.jpeg",
		"preview.webp",
		"report.pdf",
		"data.csv",
		"notes.txt",
		"bundle.zip",
		"clip.mp4",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, []byte("media"), 0o600); err != nil {
				t.Fatal(err)
			}

			content := PrepareMediaContent("Here is a file.\nMEDIA:" + path + "\nDone.")
			if strings.Contains(content.Text, "MEDIA:") {
				t.Fatalf("cleaned text leaked media tag: %q", content.Text)
			}
			if content.Text != "Here is a file.\nDone." {
				t.Fatalf("Text = %q, want media line stripped", content.Text)
			}
			if len(content.Media) != 1 || content.Media[0].Path != path {
				t.Fatalf("Media = %+v, want one extracted media path %q", content.Media, path)
			}
		})
	}
}

func TestPrepareMediaContentPreservesMixedMediaOrder(t *testing.T) {
	dir := t.TempDir()
	paths := []string{
		fileWithContent(t, dir, "voice.ogg", "audio"),
		fileWithContent(t, dir, "photo.png", "image"),
		fileWithContent(t, dir, "report.pdf", "pdf"),
		fileWithContent(t, dir, "clip.mp4", "video"),
	}

	content := PrepareMediaContent("Files:\n[[audio_as_voice]]\nMEDIA:" + paths[0] + "\nMEDIA:" + paths[1] + "\nMEDIA:" + paths[2] + "\nMEDIA:" + paths[3] + "\nDone.")
	if content.Text != "Files:\nDone." {
		t.Fatalf("Text = %q, want all media lines stripped", content.Text)
	}
	if len(content.Media) != len(paths) {
		t.Fatalf("Media len = %d, want %d: %+v", len(content.Media), len(paths), content.Media)
	}
	for i, path := range paths {
		if content.Media[i].Path != path {
			t.Fatalf("Media[%d].Path = %q, want %q; all=%+v", i, content.Media[i].Path, path, content.Media)
		}
	}
	if !content.Media[0].AsVoice {
		t.Fatalf("Media[0].AsVoice = false, want voice marker preserved")
	}
}

func TestPrepareMediaContentExtractsBracketedPathWithSpaces(t *testing.T) {
	dir := t.TempDir()
	path := fileWithContent(t, dir, "voice memo.ogg", "audio")

	content := PrepareMediaContent("Here is the audio.\n[[audio_as_voice]] [MEDIA:" + path + "]\nDone.")
	if content.Text != "Here is the audio.\nDone." {
		t.Fatalf("Text = %q, want bracketed media tag with spaces stripped", content.Text)
	}
	if len(content.Media) != 1 || content.Media[0].Path != path || !content.Media[0].AsVoice {
		t.Fatalf("Media = %+v, want one voice attachment for %q", content.Media, path)
	}
}

func TestPrepareMediaContentRejectsUnsafeMediaTags(t *testing.T) {
	content := PrepareMediaContent("listen MEDIA:../plain.txt")
	if len(content.Media) != 0 {
		t.Fatalf("Media = %+v, want unsafe path ignored", content.Media)
	}
	if !strings.Contains(content.Text, "[MEDIA:redacted]") {
		t.Fatalf("Text = %q, want redacted marker for unsafe media", content.Text)
	}
	if len(content.Evidence) != 1 || content.Evidence[0].Code != MediaEvidenceIgnored || content.Evidence[0].Target != "[redacted]" {
		t.Fatalf("Evidence = %+v, want redacted ignored evidence", content.Evidence)
	}
}

func fileWithContent(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
