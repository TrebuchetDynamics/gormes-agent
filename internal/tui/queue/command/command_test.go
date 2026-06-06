package command

import "testing"

func TestHandleSlashReportsDepthOrQueuedText(t *testing.T) {
	if got := HandleSlash("/queue", 2); got.Enqueue || got.Status != "2 queued message(s)" {
		t.Fatalf("HandleSlash empty = %+v, want depth status", got)
	}
	if got := HandleSlash("/queue follow up after tools", 0); !got.Enqueue || got.Text != "follow up after tools" || got.Status != "" {
		t.Fatalf("HandleSlash text = %+v, want enqueue stripped text", got)
	}
}
