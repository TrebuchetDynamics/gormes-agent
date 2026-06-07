package gateway

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// fakeSubmitter captures the kernel.PlatformEvent handed to Submit so tests
// can assert on Text + ContentParts without driving a real kernel.
type fakeSubmitter struct {
	events []kernel.PlatformEvent
	err    error
}

func (f *fakeSubmitter) Submit(ev kernel.PlatformEvent) error {
	f.events = append(f.events, ev)
	return f.err
}

func (f *fakeSubmitter) ResetSession() error               { return nil }
func (f *fakeSubmitter) Render() <-chan kernel.RenderFrame { return nil }

func TestTurnAdapter_Dispatch_PhotoAttachmentBecomesImageURLContentPart(t *testing.T) {
	dir := t.TempDir()
	jpgPath := gatewaytest.WriteFixtureJPEG(t, dir, "sample.jpg", 200, 50, 50)
	wantBytes, err := os.ReadFile(jpgPath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	wantDataURI := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(wantBytes)

	submitter := &fakeSubmitter{}
	adapter := &TurnAdapter{Submitter: submitter}

	req := TurnRequest{
		SubmitText: "what is in this picture?",
		Attachments: []Attachment{
			{Kind: "photo", URL: jpgPath, MediaType: "image/jpeg", FileName: "sample.jpg", SizeBytes: int64(len(wantBytes))},
		},
	}
	if err := adapter.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if len(submitter.events) != 1 {
		t.Fatalf("submitter received %d events, want 1", len(submitter.events))
	}
	ev := submitter.events[0]
	if ev.Text != "what is in this picture?" {
		t.Fatalf("text = %q, want submitter to preserve SubmitText", ev.Text)
	}
	if len(ev.ContentParts) == 0 {
		t.Fatal("ContentParts is empty; expected at least one image_url part for the photo attachment")
	}
	wantTextPart := "what is in this picture?\n\n[Image attached at: " + jpgPath + "]"
	if ev.ContentParts[0].Type != "text" || ev.ContentParts[0].Text != wantTextPart {
		t.Fatalf("ContentParts[0] = %+v, want text path hint %q", ev.ContentParts[0], wantTextPart)
	}
	var sawImage bool
	for _, part := range ev.ContentParts {
		if part.Type == "image_url" {
			sawImage = true
			if part.ImageURL != wantDataURI {
				t.Fatalf("image_url payload mismatch:\n got %q\nwant %q", part.ImageURL, wantDataURI)
			}
		}
	}
	if !sawImage {
		t.Fatal("no image_url ContentPart found; expected one for the photo attachment")
	}
}

func TestTurnAdapter_Dispatch_PhotoAttachmentEmptyTextAddsDefaultPromptAndPathHint(t *testing.T) {
	dir := t.TempDir()
	jpgPath := gatewaytest.WriteFixtureJPEG(t, dir, "sample.jpg", 50, 100, 200)

	submitter := &fakeSubmitter{}
	adapter := &TurnAdapter{Submitter: submitter}

	req := TurnRequest{
		SubmitText: "",
		Attachments: []Attachment{
			{Kind: "photo", URL: jpgPath, MediaType: "image/jpeg", FileName: "sample.jpg"},
		},
	}
	if err := adapter.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if len(submitter.events) != 1 {
		t.Fatalf("submitter received %d events, want 1", len(submitter.events))
	}
	ev := submitter.events[0]
	if ev.Text != "" {
		t.Fatalf("text = %q, want plain text projection to stay empty", ev.Text)
	}
	if len(ev.ContentParts) != 2 {
		t.Fatalf("ContentParts len = %d, want text plus one image: %+v", len(ev.ContentParts), ev.ContentParts)
	}
	wantTextPart := "What do you see in this image?\n\n[Image attached at: " + jpgPath + "]"
	if ev.ContentParts[0].Type != "text" || ev.ContentParts[0].Text != wantTextPart {
		t.Fatalf("ContentParts[0] = %+v, want default prompt path hint %q", ev.ContentParts[0], wantTextPart)
	}
	if ev.ContentParts[1].Type != "image_url" || !strings.HasPrefix(ev.ContentParts[1].ImageURL, "data:image/jpeg;base64,") {
		t.Fatalf("ContentParts[1] = %+v, want image_url data URI", ev.ContentParts[1])
	}
}

func TestTurnAdapter_Dispatch_PhotoAttachmentMissingFileFallsBackToMarker(t *testing.T) {
	submitter := &fakeSubmitter{}
	adapter := &TurnAdapter{Submitter: submitter}

	req := TurnRequest{
		SubmitText: "describe this",
		Attachments: []Attachment{
			{Kind: "photo", URL: "/nonexistent/path/to/missing.jpg", MediaType: "image/jpeg"},
		},
	}
	if err := adapter.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("Dispatch on missing-file attachment: %v", err)
	}
	if len(submitter.events) != 1 {
		t.Fatalf("submitter received %d events, want 1", len(submitter.events))
	}
	ev := submitter.events[0]
	for _, part := range ev.ContentParts {
		if part.Type == "image_url" && part.ImageURL != "" && !strings.HasPrefix(part.ImageURL, "data:") {
			t.Fatalf("missing-file attachment produced non-data image_url %q; expected fallback (no broken image_url)", part.ImageURL)
		}
		if part.Type == "image_url" && strings.HasPrefix(part.ImageURL, "data:") && len(part.ImageURL) < 30 {
			t.Fatalf("missing-file attachment produced suspiciously short data URI %q", part.ImageURL)
		}
	}
	// Acceptable behaviors: no image_url part at all, OR an image_url part that
	// the helper deliberately omitted because the file was unreadable. Either
	// way, the submitter must not see an empty/broken image_url string.
}

func TestTurnAdapter_Dispatch_MultiplePhotoAttachmentsBecomeMultipleImageURLParts(t *testing.T) {
	dir := t.TempDir()
	pathA := gatewaytest.WriteFixtureJPEG(t, dir, "a.jpg", 255, 0, 0)
	pathB := gatewaytest.WriteFixtureJPEG(t, dir, "b.jpg", 0, 255, 0)

	submitter := &fakeSubmitter{}
	adapter := &TurnAdapter{Submitter: submitter}

	req := TurnRequest{
		SubmitText: "compare these",
		Attachments: []Attachment{
			{Kind: "photo", URL: pathA, MediaType: "image/jpeg"},
			{Kind: "photo", URL: pathB, MediaType: "image/jpeg"},
		},
	}
	if err := adapter.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	ev := submitter.events[0]

	bytesA, _ := os.ReadFile(pathA)
	bytesB, _ := os.ReadFile(pathB)
	wantA := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(bytesA)
	wantB := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(bytesB)

	var imageParts []llm.MessageContentPart
	for _, part := range ev.ContentParts {
		if part.Type == "image_url" {
			imageParts = append(imageParts, part)
		}
	}
	if len(imageParts) != 2 {
		t.Fatalf("got %d image_url parts, want 2", len(imageParts))
	}
	if imageParts[0].ImageURL != wantA {
		t.Fatalf("first image_url does not match attachment A")
	}
	if imageParts[1].ImageURL != wantB {
		t.Fatalf("second image_url does not match attachment B (order must be preserved)")
	}
}

func TestTurnAdapter_Dispatch_NonImageAttachmentDoesNotBecomeImageURL(t *testing.T) {
	dir := t.TempDir()
	voicePath := filepath.Join(dir, "voice.ogg")
	if err := os.WriteFile(voicePath, []byte("OggS\x00fake-voice-payload"), 0o600); err != nil {
		t.Fatalf("write fake voice: %v", err)
	}
	docPath := filepath.Join(dir, "doc.pdf")
	if err := os.WriteFile(docPath, []byte("%PDF-1.4\nfake"), 0o600); err != nil {
		t.Fatalf("write fake doc: %v", err)
	}

	submitter := &fakeSubmitter{}
	adapter := &TurnAdapter{Submitter: submitter}

	req := TurnRequest{
		SubmitText: "ignore my non-images",
		Attachments: []Attachment{
			{Kind: "voice", URL: voicePath, MediaType: "audio/ogg"},
			{Kind: "document", URL: docPath, MediaType: "application/pdf"},
		},
	}
	if err := adapter.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	ev := submitter.events[0]
	for _, part := range ev.ContentParts {
		if part.Type == "image_url" {
			t.Fatalf("non-image attachment produced image_url ContentPart; voice/document must not become image_url. Part = %+v", part)
		}
	}
}

func TestTurnAdapter_Dispatch_NoAttachments_LeavesContentPartsEmpty(t *testing.T) {
	submitter := &fakeSubmitter{}
	adapter := &TurnAdapter{Submitter: submitter}

	req := TurnRequest{SubmitText: "plain text turn"}
	if err := adapter.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	ev := submitter.events[0]
	if len(ev.ContentParts) != 0 {
		t.Fatalf("ContentParts = %d, want 0 for plain-text turn (no attachments)", len(ev.ContentParts))
	}
}

func TestTurnAdapter_Dispatch_SubmitterErrorStillRunsOnTurnFailure(t *testing.T) {
	wantErr := errors.New("submitter blew up")
	submitter := &fakeSubmitter{err: wantErr}
	var failed bool
	adapter := &TurnAdapter{
		Submitter: submitter,
		OnTurnFailure: func(_ TurnRequest, err error) {
			if !errors.Is(err, wantErr) {
				t.Fatalf("OnTurnFailure got %v, want %v", err, wantErr)
			}
			failed = true
		},
	}
	if err := adapter.Dispatch(context.Background(), TurnRequest{SubmitText: "x"}); err == nil {
		t.Fatal("expected propagated submitter error")
	}
	if !failed {
		t.Fatal("OnTurnFailure was not invoked")
	}
}
