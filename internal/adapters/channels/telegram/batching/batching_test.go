package batching

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestPhotoBatchHelpers(t *testing.T) {
	ev := gateway.InboundEvent{ChatID: " 42 ", UserID: " 7 ", Attachments: []gateway.Attachment{{Kind: " Photo "}}}
	if !InboundEventHasPhoto(ev) {
		t.Fatal("InboundEventHasPhoto = false, want true")
	}
	if got := PhotoBatchKey(ev, " album "); got != "album:42:album" {
		t.Fatalf("album key = %q", got)
	}
	if got := PhotoBatchKey(ev, " "); got != "burst:42:7" {
		t.Fatalf("burst key = %q", got)
	}
	merged := MergePhotoBatch(gateway.InboundEvent{Text: "first", Attachments: []gateway.Attachment{{Kind: "photo"}}}, gateway.InboundEvent{Text: "second", Attachments: []gateway.Attachment{{Kind: "photo"}}})
	if merged.Text != "first\n\nsecond" || len(merged.Attachments) != 2 {
		t.Fatalf("merged = %#v", merged)
	}
}

func TestTextBatchHelpers(t *testing.T) {
	ev := gateway.InboundEvent{Kind: gateway.EventSubmit, Platform: " telegram ", ChatID: " 42 ", ChatType: "group", UserID: " 7 ", Text: " hello "}
	if !InboundEventIsBatchableText(ev) {
		t.Fatal("InboundEventIsBatchableText = false, want true")
	}
	if got := TextBatchKey(ev); got != "telegram:42:group:-:7" {
		t.Fatalf("TextBatchKey = %q", got)
	}
	merged := MergeTextBatch(gateway.InboundEvent{Text: "first"}, gateway.InboundEvent{Text: " second "})
	if merged.Text != "first\nsecond" {
		t.Fatalf("merged text = %q", merged.Text)
	}
}
