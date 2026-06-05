package signal

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestSignalMarkdownFormatsBasicStylesAndUTF16Offsets(t *testing.T) {
	plain, ranges := MarkdownToSignal("hi **bold** and *italic* with ~~gone~~ plus `mono`")

	if plain != "hi bold and italic with gone plus mono" {
		t.Fatalf("plain = %q", plain)
	}
	assertSignalRange(t, plain, ranges, "bold", SignalStyleBold)
	assertSignalRange(t, plain, ranges, "italic", SignalStyleItalic)
	assertSignalRange(t, plain, ranges, "gone", SignalStyleStrikethrough)
	assertSignalRange(t, plain, ranges, "mono", SignalStyleMonospace)
}

func TestSignalMarkdownHeadingsCodeBlocksAndEmojiOffsets(t *testing.T) {
	plain, ranges := MarkdownToSignal("## Plan\n\n```go\nfmt.Println(\"ok\")\n```\n\n👋 **hello**")

	if plain != "Plan\n\nfmt.Println(\"ok\")\n\n👋 hello" {
		t.Fatalf("plain = %q", plain)
	}
	assertSignalRange(t, plain, ranges, "Plan", SignalStyleBold)
	assertSignalRange(t, plain, ranges, "fmt.Println(\"ok\")", SignalStyleMonospace)
	assertSignalRange(t, plain, ranges, "hello", SignalStyleBold)
}

func TestSignalMarkdownFalsePositiveGuardsPreservePlainText(t *testing.T) {
	input := strings.Join([]string{
		"config_file stays literal",
		"/tools/delegate_tool.py stays literal",
		"* tools/file_tools.py is a bullet",
		"* this has *emphasis* inside",
		"*foo",
		"bar*",
	}, "\n")

	plain, ranges := MarkdownToSignal(input)

	wantPlain := strings.Join([]string{
		"config_file stays literal",
		"/tools/delegate_tool.py stays literal",
		"* tools/file_tools.py is a bullet",
		"* this has emphasis inside",
		"*foo",
		"bar*",
	}, "\n")
	if plain != wantPlain {
		t.Fatalf("plain = %q, want %q", plain, wantPlain)
	}
	for _, r := range ranges {
		if r.Style == SignalStyleItalic && signalRangeText(t, plain, r) != "emphasis" {
			t.Fatalf("unexpected italic false positive: range=%+v text=%q", r, signalRangeText(t, plain, r))
		}
	}
	assertSignalRange(t, plain, ranges, "emphasis", SignalStyleItalic)
}

func TestSignalFormatOptionsFlowThroughBotSend(t *testing.T) {
	mc := newMockClient()
	b := New(mc, nil)
	inbox := make(chan gateway.InboundEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mc.push(InboundMessage{
		ChatType:   ChatTypeDirect,
		SenderID:   "+15551234567",
		SenderUUID: "uuid-alice",
		SenderName: "Alice",
		MessageID:  "msg-1",
		Text:       "hello",
	})

	select {
	case <-inbox:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected inbound event before send")
	}

	_, err := b.Send(context.Background(), "+15551234567", "reply with **bold** and `code`")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	direct := mc.directSnapshot()
	if len(direct) != 1 {
		t.Fatalf("direct send count = %d, want 1", len(direct))
	}
	if direct[0].Text != "reply with bold and code" {
		t.Fatalf("direct text = %q", direct[0].Text)
	}
	gotStyles := []SignalTextStyle{direct[0].Options.BodyRanges[0].Style, direct[0].Options.BodyRanges[1].Style}
	if !reflect.DeepEqual(gotStyles, []SignalTextStyle{SignalStyleBold, SignalStyleMonospace}) {
		t.Fatalf("body range styles = %#v", gotStyles)
	}
}

func assertSignalRange(t *testing.T, plain string, ranges []SignalBodyRange, wantText string, wantStyle SignalTextStyle) {
	t.Helper()
	for _, r := range ranges {
		if r.Style == wantStyle && signalRangeText(t, plain, r) == wantText {
			return
		}
	}
	t.Fatalf("missing %s range for %q in plain=%q ranges=%+v", wantStyle, wantText, plain, ranges)
}

func signalRangeText(t *testing.T, plain string, r SignalBodyRange) string {
	t.Helper()
	units := signalUTF16Units(plain)
	if r.Start < 0 || r.Length < 0 || r.Start+r.Length > len(units) {
		t.Fatalf("range %+v outside %d UTF-16 units for %q", r, len(units), plain)
	}
	return string(signalRunesFromUTF16(units[r.Start : r.Start+r.Length]))
}
