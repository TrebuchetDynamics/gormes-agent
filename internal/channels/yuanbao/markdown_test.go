package yuanbao

import (
	"errors"
	"strings"
	"testing"
)

func TestYuanbaoMarkdown_RendersCodeAndLinks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     string
		wantParts []string
		denyParts []string
	}{
		{
			name:      "inline link keeps url",
			input:     "see [docs](https://gormes.ai/docs) for details",
			wantParts: []string{"docs", "https://gormes.ai/docs"},
		},
		{
			name:      "bare url preserved",
			input:     "open https://example.com/path?x=1 now",
			wantParts: []string{"https://example.com/path?x=1"},
		},
		{
			name:      "fenced code block kept verbatim",
			input:     "before\n```go\nfmt.Println(\"hi\")\n```\nafter",
			wantParts: []string{"fmt.Println(\"hi\")", "before", "after"},
			denyParts: []string{"```"},
		},
		{
			name:      "inline code stripped of backticks",
			input:     "use `os.ReadFile` to load data",
			wantParts: []string{"os.ReadFile", "to load data"},
			denyParts: []string{"`os.ReadFile`"},
		},
		{
			name:      "mention preserved literally",
			input:     "ping @alice and @bob_42 on this",
			wantParts: []string{"@alice", "@bob_42"},
		},
		{
			name:      "bullet list kept as plain lines",
			input:     "- first item\n- second item\n- third with [link](https://x.io)",
			wantParts: []string{"first item", "second item", "third with link", "https://x.io"},
			denyParts: []string{"](https://x.io)"},
		},
		{
			name:      "ordered list kept",
			input:     "1. step one\n2. step two\n3. step three",
			wantParts: []string{"step one", "step two", "step three"},
		},
		{
			name:      "bold and italic markers stripped",
			input:     "this is **bold** and *italic* text",
			wantParts: []string{"this is bold and italic text"},
			denyParts: []string{"**", "*italic*"},
		},
		{
			name:      "image alt + url kept",
			input:     "![logo](https://gormes.ai/logo.png)",
			wantParts: []string{"logo", "https://gormes.ai/logo.png"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := RenderPromptSafe(tc.input)
			for _, want := range tc.wantParts {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\noutput=%q", want, got)
				}
			}
			for _, deny := range tc.denyParts {
				if strings.Contains(got, deny) {
					t.Errorf("output unexpectedly contains %q\noutput=%q", deny, got)
				}
			}
		})
	}
}

func TestYuanbaoMarkdown_EmptyOrPureWhitespaceReturnsDegraded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "whitespace", input: "   \n\t  "},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := RenderPromptSafeStrict(tc.input)
			if err == nil {
				t.Fatalf("expected degraded error for %q", tc.input)
			}
			var deg *DegradedError
			if !errors.As(err, &deg) {
				t.Fatalf("expected *DegradedError, got %T: %v", err, err)
			}
			if deg.Code != DegradedMarkdownParseFailed {
				t.Errorf("Code = %q, want %q", deg.Code, DegradedMarkdownParseFailed)
			}
		})
	}
}
