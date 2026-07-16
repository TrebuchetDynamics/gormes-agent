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
	Name     string
	MimeType string
	URI      string
}

// Render flattens structured content without server-specific link guidance.
func Render(parts []Structured) string {
	return RenderForServer(parts, "")
}

// RenderForServer flattens a sequence of structured content blocks into a
// single model-facing string. Text and embedded-resource text contribute
// verbatim; image and URI-only resource blocks render stable placeholders.
// Resource links name the real provider-visible read_resource wire tool when
// the originating MCP server is known. Unknown kinds emit their Text field if
// any (no panic, no JSON envelope leak).
func RenderForServer(parts []Structured, serverName string) string {
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
		case "resource_link":
			write(renderResourceLink(p, serverName))
		case "resource":
			if p.Text != "" {
				write(p.Text)
			} else {
				write(fmt.Sprintf("[resource: %s]", p.URI))
			}
		default:
			write(p.Text)
		}
	}
	return b.String()
}

func renderResourceLink(part Structured, serverName string) string {
	if part.URI == "" {
		return ""
	}
	details := "uri=" + part.URI
	if part.Name != "" {
		details += ", name=" + part.Name
	}
	if part.MimeType != "" {
		details += ", mimeType=" + part.MimeType
	}
	reader := "the MCP server's read_resource tool"
	if server := sanitizeNameComponent(serverName); server != "" {
		reader = "mcp__" + server + "__read_resource"
	}
	return "[MCP resource link: " + details + " — fetch it with " + reader + "]"
}

func sanitizeNameComponent(value string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			return r
		default:
			return '_'
		}
	}, value)
}
