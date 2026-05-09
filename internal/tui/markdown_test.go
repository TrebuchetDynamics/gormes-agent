package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func TestRenderMarkdown_FencedCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "single line code block",
			input:    "```\nfunc main() {}\n```",
			contains: "func main() {}",
		},
		{
			name:     "code block with language",
			input:    "```go\npackage main\n```",
			contains: "go",
		},
		{
			name:     "multiple line code block",
			input:    "```\nline1\nline2\nline3\n```",
			contains: "line1",
		},
		{
			name:     "tilde code blocks",
			input:    "~~~\ncode\n~~~",
			contains: "code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("RenderMarkdown() does not contain expected %q, got:\n%s", tt.contains, result)
			}
		})
	}
}

func TestRenderMarkdown_InlineBold(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold text",
			input:    "This is **bold** text",
			expected: "This is bold text",
		},
		{
			name:     "bold at start",
			input:    "**Hello** world",
			expected: "Hello world",
		},
		{
			name:     "bold at end",
			input:    "Hello **World**",
			expected: "Hello World",
		},
		{
			name:     "multiple bold",
			input:    "**one** and **two**",
			expected: "one and two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.expected)
			}
		})
	}
}

func TestRenderMarkdown_InlineItalic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "italic text",
			input:    "This is *italic* text",
			expected: "This is italic text",
		},
		{
			name:     "italic at start",
			input:    "*Hello* world",
			expected: "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.expected)
			}
		})
	}
}

func TestRenderMarkdown_InlineCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "inline code",
			input:    "Use `fmt.Println()` to print",
			expected: "fmt.Println()",
		},
		{
			name:     "multiple inline codes",
			input:    "`a` and `b`",
			expected: "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.expected)
			}
		})
	}
}

func TestRenderMarkdown_Headers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "h1 header",
			input:    "# Header 1",
			expected: "Header 1",
		},
		{
			name:     "h2 header",
			input:    "## Header 2",
			expected: "Header 2",
		},
		{
			name:     "h3 header",
			input:    "### Header 3",
			expected: "Header 3",
		},
		{
			name:     "h6 header",
			input:    "###### Header 6",
			expected: "Header 6",
		},
		{
			name:     "header with trailing hashes",
			input:    "## Title ##",
			expected: "Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.expected)
			}
		})
	}
}

func TestRenderMarkdown_BulletLists(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "dash bullet list",
			input:    "- item one\n- item two",
			contains: "• item one",
		},
		{
			name:     "asterisk bullet list",
			input:    "* item one\n* item two",
			contains: "•",
		},
		{
			name:     "nested bullet list",
			input:    "- level 1\n  - level 2",
			contains: "•",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestRenderMarkdown_NumberedLists(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "numbered list with dot",
			input:    "1. first\n2. second\n3. third",
			contains: "first",
		},
		{
			name:     "numbered list with parenthesis",
			input:    "1) one\n2) two",
			contains: "one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestRenderMarkdown_Blockquotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "simple blockquote",
			input:    "> This is a quote",
			contains: "│",
		},
		{
			name:     "multi-line blockquote",
			input:    "> Line one\n> Line two",
			contains: "│",
		},
		{
			name:     "nested blockquote",
			input:    "> Quote\n> > Nested",
			contains: "│",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestRenderMarkdown_HorizontalRules(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "dash hr",
			input:    "Text\n\n---\n\nMore text",
			contains: "─",
		},
		{
			name:     "asterisk hr",
			input:    "***",
			contains: "─",
		},
		{
			name:     "underscore hr",
			input:    "___",
			contains: "─",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestRenderMarkdown_Tables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "simple table",
			input:    "| Header |\n| ------- |\n| Cell    |",
			contains: "Header",
		},
		{
			name:     "multi-column table",
			input:    "| A | B |\n| - | - |\n| 1 | 2 |",
			contains: "A",
		},
		{
			name:     "table with alignment",
			input:    "| Left | Center | Right |\n| :--- | :---: | ---: |\n| L | C | R |",
			contains: "Left",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMarkdown(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("RenderMarkdown() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestRenderMarkdown_EmptyInput(t *testing.T) {
	result := RenderMarkdown("")
	if result != "" {
		t.Errorf("RenderMarkdown(\"\") = %q, want empty string", result)
	}
}

func TestRenderMarkdown_CombinedElements(t *testing.T) {
	input := "# Main Title\n\n**Bold** and *italic* text with `inline code`.\n\n## Section\n\n- Bullet item\n- Another item\n\n1. Numbered item\n2. Another numbered\n\n> A blockquote\n\n" + "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```" + "\n\n| Column |\n| ------ |\n| Data   |"

	result := RenderMarkdown(input)

	// Verify all elements are present
	if !strings.Contains(result, "Main Title") {
		t.Error("missing heading 1")
	}
	if !strings.Contains(result, "Bold") {
		t.Error("missing bold")
	}
	if !strings.Contains(result, "Section") {
		t.Error("missing heading 2")
	}
	if !strings.Contains(result, "•") {
		t.Error("missing bullet list")
	}
	if !strings.Contains(result, "func main()") {
		t.Error("missing code block")
	}
	if !strings.Contains(result, "Column") {
		t.Error("missing table")
	}
}

func TestRenderMarkdownSoftWrapTrim_RemovesSingleBoundarySpace(t *testing.T) {
	got := RenderMarkdownSoftWrapTrim("Let me", 5)
	if got != "Let\nme" {
		t.Fatalf("RenderMarkdownSoftWrapTrim() = %q, want %q", got, "Let\nme")
	}
}

func TestRenderMarkdownSoftWrapTrim_PreservesExtraBoundarySpacing(t *testing.T) {
	got := RenderMarkdownSoftWrapTrim("foo  bar", 5)
	if got != "foo \nbar" {
		t.Fatalf("RenderMarkdownSoftWrapTrim() = %q, want %q", got, "foo \nbar")
	}
}

func TestRenderMarkdownSoftWrapTrim_PreservesLeadingIndentation(t *testing.T) {
	got := RenderMarkdownSoftWrapTrim("  indented", 20)
	if got != "  indented" {
		t.Fatalf("RenderMarkdownSoftWrapTrim() = %q, want leading indentation preserved", got)
	}
}

func TestConversationMessageBlock_UsesSoftWrapTrim(t *testing.T) {
	got := conversationMessageBlock(hermes.Message{Role: "assistant", Content: "Let me"}, 5, false)
	if strings.Contains(got, "\n me") {
		t.Fatalf("conversationMessageBlock() kept a boundary space on continuation line: %q", got)
	}
	if !strings.Contains(got, "\nme") {
		t.Fatalf("conversationMessageBlock() = %q, want continuation line without leading boundary space", got)
	}
}
