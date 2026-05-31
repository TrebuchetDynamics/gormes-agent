package content

import (
	"fmt"
	"strings"
)

// Structured is a single content block from an MCP tools/call result (or the
// structuredContent envelope). The renderer only inspects the fields relevant
// to its Kind; unknown kinds fall back to Text so unsupported content types
// degrade gracefully instead of leaking the raw protocol envelope.
type Structured struct {
	Kind     string
	Text     string
	MimeType string
	URI      string
}

// Render flattens a sequence of structured content blocks into a single
// model-facing string. Text blocks contribute verbatim; image and resource
// blocks render as bracketed placeholders so the model has a stable signal that
// a non-textual artifact was returned. Unknown kinds emit their Text field if
// any (no panic, no JSON envelope leak).
func Render(parts []Structured) string {
	var b strings.Builder
	first := true
	write := func(s string) {
		if s == "" {
			return
		}
		if !first {
			b.WriteByte('\n')
		}
		b.WriteString(s)
		first = false
	}
	for _, p := range parts {
		switch p.Kind {
		case "text":
			write(p.Text)
		case "image":
			write(fmt.Sprintf("[image: %s]", p.MimeType))
		case "resource":
			write(fmt.Sprintf("[resource: %s]", p.URI))
		default:
			write(p.Text)
		}
	}
	return b.String()
}
