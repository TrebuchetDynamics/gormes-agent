package input

import (
	"errors"
	"strings"
	"testing"
)

func TestFacadePreservesInputContracts(t *testing.T) {
	if got := StripLeakedBracketedPasteWrappers("\x1b[200~hello\x1b[201~"); got != "hello" {
		t.Fatalf("StripLeakedBracketedPasteWrappers() = %q", got)
	}

	cleaned, meta := StripLeakedTerminalResponsesWithMeta("hello\x1b[53;1Rworld")
	if cleaned != "helloworld" || !meta.Stripped || meta.Evidence != TerminalResponseStrippedEvidence {
		t.Fatalf("terminal sanitizer = %q %+v", cleaned, meta)
	}

	body, err := ResolveSendMessageBody(SendMessageBodyOptions{Stdin: strings.NewReader("piped\n"), StdinIsTTY: false})
	if err != nil {
		t.Fatalf("ResolveSendMessageBody: %v", err)
	}
	if body.Text != "piped\n" || body.Source != "stdin" {
		t.Fatalf("body = %+v", body)
	}

	_, err = ResolveSendMessageBody(SendMessageBodyOptions{StdinIsTTY: true})
	if !errors.Is(err, ErrSendMessageMissingBody) {
		t.Fatalf("missing body error = %v", err)
	}

	if got := FormatUserMessagePreview("one\ntwo\nthree", UserMessagePreviewConfig{FirstLines: 1, LastLines: 1}); !strings.Contains(got, "(+1 more line)") {
		t.Fatalf("preview = %q", got)
	}
}
