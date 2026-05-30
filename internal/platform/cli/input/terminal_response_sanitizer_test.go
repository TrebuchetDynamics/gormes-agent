package input

import (
	"fmt"
	"strings"
	"testing"
)

func TestStripLeakedTerminalResponses_DSRCanonicalAndVisible(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "plain text", in: "hello world", want: "hello world"},
		{name: "canonical standalone", in: "\x1b[53;1R", want: ""},
		{name: "canonical embedded", in: "hello\x1b[53;1Rworld", want: "helloworld"},
		{name: "visible standalone", in: "^[[53;1R", want: ""},
		{name: "visible embedded", in: "typed^[[53;1Rmore", want: "typedmore"},
		{name: "multiline", in: "line 1\n\x1b[53;1Rline 2", want: "line 1\nline 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripLeakedTerminalResponses(tt.in)
			if got != tt.want {
				t.Fatalf("StripLeakedTerminalResponses() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripLeakedTerminalResponses_MultipleDSRResponses(t *testing.T) {
	got := StripLeakedTerminalResponses("a\x1b[53;1Rb\x1b[51;1Rc\x1b[50;9Rd")
	if got != "abcd" {
		t.Fatalf("StripLeakedTerminalResponses() = %q, want %q", got, "abcd")
	}
}

func TestStripLeakedTerminalResponses_SGRMouseReports(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "escape form", in: "abc\x1b[<65;1;49Mdef", want: "abcdef"},
		{name: "visible form", in: "abc^[[<65;1;49Mdef", want: "abcdef"},
		{name: "bare form", in: "abc<65;1;49Mdef", want: "abcdef"},
		{name: "large coordinates", in: "abc\x1b[<10000;12345;98765Mdef", want: "abcdef"},
		{name: "concatenated", in: "<65;1;49M<35;1;42Mhello<64;1;40m", want: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripLeakedTerminalResponses(tt.in)
			if got != tt.want {
				t.Fatalf("StripLeakedTerminalResponses() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripLeakedTerminalResponses_PreservesSGRColorAndBareDSRText(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "ansi color", in: "\x1b[31mred\x1b[0m"},
		{name: "bare dsr text", in: "see section [53;1R for details"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripLeakedTerminalResponses(tt.in)
			if got != tt.in {
				t.Fatalf("StripLeakedTerminalResponses() = %q, want unchanged %q", got, tt.in)
			}
		})
	}
}

func TestStripLeakedTerminalResponses_PreservesAngleBracketContent(t *testing.T) {
	text := "render <div class='hero'> literal"
	got := StripLeakedTerminalResponses(text)
	if got != text {
		t.Fatalf("StripLeakedTerminalResponses() = %q, want unchanged %q", got, text)
	}
}

func TestStripLeakedTerminalResponsesWithMetaEvidence(t *testing.T) {
	cleaned, meta := StripLeakedTerminalResponsesWithMeta("abc\x1b[<65;1;49Mdef")
	if cleaned != "abcdef" {
		t.Fatalf("cleaned = %q, want %q", cleaned, "abcdef")
	}
	if !meta.Stripped {
		t.Fatal("meta.Stripped = false, want true")
	}
	if !meta.MouseReportsStripped {
		t.Fatal("meta.MouseReportsStripped = false, want true")
	}
	if meta.Evidence != TerminalResponseStrippedEvidence {
		t.Fatalf("meta.Evidence = %q, want %q", meta.Evidence, TerminalResponseStrippedEvidence)
	}

	rendered := fmt.Sprintf("%+v", meta)
	for _, forbidden := range []string{"abc", "def", "65;1;49"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("metadata leaks raw input %q in %#v", forbidden, meta)
		}
	}

	unchanged, emptyMeta := StripLeakedTerminalResponsesWithMeta("ordinary input")
	if unchanged != "ordinary input" {
		t.Fatalf("unchanged = %q, want ordinary input", unchanged)
	}
	if emptyMeta.Stripped || emptyMeta.MouseReportsStripped || emptyMeta.Evidence != "" {
		t.Fatalf("empty metadata = %#v, want no evidence", emptyMeta)
	}
}
