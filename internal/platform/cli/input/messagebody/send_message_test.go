package messagebody

import (
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/input/sanitization"
)

func TestResolveSendMessageBodyPreservesSources(t *testing.T) {
	t.Run("positional wins", func(t *testing.T) {
		got, err := ResolveSendMessageBody(SendMessageBodyOptions{
			Positional: "hello world",
			FilePath:   "ignored.txt",
			ReadFile: func(string) ([]byte, error) {
				t.Fatal("ReadFile should not be called when positional text exists")
				return nil, nil
			},
			StdinIsTTY: true,
		})
		if err != nil {
			t.Fatalf("ResolveSendMessageBody: %v", err)
		}
		if got.Text != "hello world" || got.Source != "positional" {
			t.Fatalf("body = %+v, want positional text", got)
		}
	})

	t.Run("file preserves newline", func(t *testing.T) {
		got, err := ResolveSendMessageBody(SendMessageBodyOptions{
			FilePath: "report.md",
			ReadFile: func(path string) ([]byte, error) {
				if path != "report.md" {
					t.Fatalf("ReadFile path = %q", path)
				}
				return []byte("from a file\n"), nil
			},
			StdinIsTTY: true,
		})
		if err != nil {
			t.Fatalf("ResolveSendMessageBody: %v", err)
		}
		if got.Text != "from a file\n" || got.Source != "file" {
			t.Fatalf("body = %+v, want file text with trailing newline", got)
		}
	})

	t.Run("piped stdin preserves newline", func(t *testing.T) {
		got, err := ResolveSendMessageBody(SendMessageBodyOptions{
			Stdin:      strings.NewReader("piped body\n"),
			StdinIsTTY: false,
		})
		if err != nil {
			t.Fatalf("ResolveSendMessageBody: %v", err)
		}
		if got.Text != "piped body\n" || got.Source != "stdin" {
			t.Fatalf("body = %+v, want stdin text with trailing newline", got)
		}
	})
}

func TestResolveSendMessageBodyRejectsInvalidText(t *testing.T) {
	_, err := ResolveSendMessageBody(SendMessageBodyOptions{
		FilePath: "bad.bin",
		ReadFile: func(string) ([]byte, error) {
			return []byte{0xff, 0xfe, 0x00}, nil
		},
		StdinIsTTY: true,
	})
	if err == nil {
		t.Fatal("ResolveSendMessageBody error = nil, want invalid text error")
	}
	if !errors.Is(err, ErrSendMessageInvalidText) {
		t.Fatalf("ResolveSendMessageBody error = %v, want ErrSendMessageInvalidText", err)
	}
	for _, forbidden := range []string{"\xff", "\xfe"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("invalid text error leaked raw byte %q: %v", forbidden, err)
		}
	}
}

func TestResolveSendMessageBodyStripsTerminalResponses(t *testing.T) {
	got, err := ResolveSendMessageBody(SendMessageBodyOptions{
		Positional: "hello\x1b[53;1Rworld",
		StdinIsTTY: true,
	})
	if err != nil {
		t.Fatalf("ResolveSendMessageBody: %v", err)
	}
	if got.Text != "helloworld" {
		t.Fatalf("Text = %q, want terminal response stripped", got.Text)
	}
	if !got.SanitizerMeta.Stripped || got.SanitizerMeta.Evidence != sanitization.TerminalResponseStrippedEvidence {
		t.Fatalf("SanitizerMeta = %+v, want stripped evidence", got.SanitizerMeta)
	}
}

func TestResolveSendMessageBodyMissing(t *testing.T) {
	_, err := ResolveSendMessageBody(SendMessageBodyOptions{StdinIsTTY: true})
	if !errors.Is(err, ErrSendMessageMissingBody) {
		t.Fatalf("ResolveSendMessageBody error = %v, want ErrSendMessageMissingBody", err)
	}
}
