package tuigateway

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTUIComposerIngressImageAttachRPC(t *testing.T) {
	tmp := t.TempDir()
	image := writeIngressRPCFile(t, tmp, "photo.png", []byte("fake"))
	text := writeIngressRPCFile(t, tmp, "notes.txt", []byte("hello\n"))

	resp, err := HandleImageAttach(ImageAttachRequest{
		SessionID: "sess-1",
		Path:      "/image " + image + " describe this",
	}, IngressOptions{})
	if err != nil {
		t.Fatalf("HandleImageAttach(image) error = %v", err)
	}
	if resp.Name != "photo.png" {
		t.Fatalf("Name = %q, want photo.png", resp.Name)
	}
	if resp.Path != image {
		t.Fatalf("Path = %q, want %q", resp.Path, image)
	}
	if resp.Remainder != "describe this" {
		t.Fatalf("Remainder = %q, want describe this", resp.Remainder)
	}

	if _, err := HandleImageAttach(ImageAttachRequest{SessionID: "sess-1", Path: "/image " + text}, IngressOptions{}); err == nil {
		t.Fatal("HandleImageAttach(non-image) error = nil, want rejection")
	}
}

func TestTUIComposerIngressPasteCollapseRPC(t *testing.T) {
	resp, err := HandlePasteCollapse(PasteCollapseRequest{Text: "alpha\nbeta"}, IngressOptions{
		CollapsePaste: func(text string) (string, error) {
			if text != "alpha\nbeta" {
				t.Fatalf("collapse text = %q, want original text", text)
			}
			return "/tmp/paste-1.txt", nil
		},
	})
	if err != nil {
		t.Fatalf("HandlePasteCollapse() error = %v", err)
	}
	if resp.Path != "/tmp/paste-1.txt" {
		t.Fatalf("Path = %q, want /tmp/paste-1.txt", resp.Path)
	}

	if _, err := HandlePasteCollapse(PasteCollapseRequest{Text: "alpha"}, IngressOptions{
		CollapsePaste: func(string) (string, error) { return "", errors.New("disk full") },
	}); err == nil {
		t.Fatal("HandlePasteCollapse(failure) error = nil, want failure")
	}
}

func TestTUIComposerIngressInputDetectDropRPC(t *testing.T) {
	tmp := t.TempDir()
	doc := writeIngressRPCFile(t, tmp, "main.py", []byte("print('hello')\n"))

	resp, err := HandleInputDetectDrop(InputDetectDropRequest{
		SessionID: "sess-1",
		Text:      doc + " review this",
	}, IngressOptions{})
	if err != nil {
		t.Fatalf("HandleInputDetectDrop() error = %v", err)
	}
	if !resp.Matched || resp.Text != "main.py review this" || resp.Path != doc || resp.IsImage {
		t.Fatalf("drop response = %+v, want matched non-image file text", resp)
	}
}

func writeIngressRPCFile(t *testing.T, root, rel string, body []byte) string {
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
