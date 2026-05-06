package slack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestSlackChannel_SendMediaUploadsFileToRememberedThread(t *testing.T) {
	mc := newMockClient()
	ch := NewChannel(mc, nil)
	ch.rememberThread("C123", "1711111111.000200")
	path := writeSlackTestMedia(t, "photo.png", "image")

	fileID, err := ch.SendMedia(context.Background(), "C123", "", gateway.OutboundMedia{
		Path: path,
		Kind: gateway.OutboundMediaKindImage,
	})
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}
	if fileID == "" {
		t.Fatal("fileID empty")
	}

	uploads := mc.uploads()
	if len(uploads) != 1 {
		t.Fatalf("uploads = %+v, want one upload", uploads)
	}
	if uploads[0].channelID != "C123" || uploads[0].threadTS != "1711111111.000200" || uploads[0].filePath != path {
		t.Fatalf("upload = %+v, want channel/thread/path metadata", uploads[0])
	}
}

func TestBot_FinalMediaDeliveryUploadsSlackFilesWithoutLeakingTags(t *testing.T) {
	mc := newMockClient()
	b := New(Config{ReplyInThread: true}, mc, newIdleSlackKernel(), nil)
	path := writeSlackTestMedia(t, "report.pdf", "pdf")

	err := b.deliverBinding(context.Background(), turnBinding{
		channelID: "C123",
		threadTS:  "1711111111.000300",
	}, "Here is the report.\nMEDIA:"+path)
	if err != nil {
		t.Fatalf("deliverBinding: %v", err)
	}

	output := mc.lastOutputText()
	if strings.Contains(output, "MEDIA:") || strings.Contains(output, path) {
		t.Fatalf("output leaked media tag/path: %q", output)
	}
	if output != "Here is the report." {
		t.Fatalf("output = %q, want stripped report text", output)
	}
	uploads := mc.uploads()
	if len(uploads) != 1 {
		t.Fatalf("uploads = %+v, want one upload", uploads)
	}
	if uploads[0].channelID != "C123" || uploads[0].threadTS != "1711111111.000300" || uploads[0].filePath != path {
		t.Fatalf("upload = %+v, want Slack channel/thread/path metadata", uploads[0])
	}
}

func writeSlackTestMedia(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
