package content

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
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
	Data     string
	Blob     string
}

const (
	maxResourceBytes    = 50 * 1024 * 1024
	maxResourceB64Chars = maxResourceBytes*4/3 + 4
)

// RenderOptions configures the additive IO-capable renderer. ArtifactDir is
// supplied by the caller so this package never guesses a profile or workspace
// path; an empty directory keeps rendering filesystem-free.
type RenderOptions struct {
	ServerName  string
	ArtifactDir string
}

// Render flattens structured content without server-specific link guidance.
func Render(parts []Structured) string {
	return RenderWithOptions(parts, RenderOptions{})
}

// RenderForServer preserves the existing source-compatible server-aware API.
func RenderForServer(parts []Structured, serverName string) string {
	return RenderWithOptions(parts, RenderOptions{ServerName: serverName})
}

// RenderWithOptions flattens a sequence of structured content blocks into a
// single model-facing string. When ArtifactDir is set, binary resources are
// decoded into unique path-safe files beneath that directory.
func RenderWithOptions(parts []Structured, opts RenderOptions) string {
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
		case "audio":
			write(renderAudio(p, opts.ArtifactDir))
		case "resource_link":
			write(renderResourceLink(p, opts.ServerName))
		case "resource":
			if p.Text != "" {
				write(p.Text)
			} else if p.Blob != "" {
				write(renderEmbeddedResource(p, opts.ArtifactDir))
			} else {
				write(fmt.Sprintf("[resource: %s]", p.URI))
			}
		default:
			write(p.Text)
		}
	}
	return b.String()
}

func renderAudio(part Structured, artifactDir string) string {
	mimeType := normalizeMIME(part.MimeType)
	if part.Data == "" || !strings.HasPrefix(mimeType, "audio/") {
		return ""
	}
	if len(part.Data) > maxResourceB64Chars {
		return fmt.Sprintf("[MCP audio resource too large to cache: ~%d bytes]", len(part.Data)*3/4)
	}
	raw, err := base64.StdEncoding.DecodeString(part.Data)
	if err != nil {
		return fmt.Sprintf("[MCP audio resource could not be decoded: %s]", firstNonEmpty(mimeType, "unknown type"))
	}
	if len(raw) > maxResourceBytes {
		return fmt.Sprintf("[MCP audio resource too large to cache: %d bytes]", len(raw))
	}
	if strings.TrimSpace(artifactDir) == "" {
		return fmt.Sprintf("[MCP audio resource received (%d bytes, %s) but audio cache unavailable in this process]", len(raw), firstNonEmpty(mimeType, "unknown type"))
	}
	artifact, err := writeArtifact(artifactDir, audioFilename(mimeType), raw)
	if err != nil {
		return fmt.Sprintf("[MCP audio resource could not be cached: %s]", firstNonEmpty(mimeType, "unknown type"))
	}
	return "MEDIA:" + artifact
}

func audioFilename(mimeType string) string {
	switch mimeType {
	case "audio/wav", "audio/x-wav", "audio/wave":
		return "audio.wav"
	}
	if extensions, err := mime.ExtensionsByType(mimeType); err == nil && len(extensions) > 0 {
		return "audio" + extensions[0]
	}
	return "audio.ogg"
}

func renderEmbeddedResource(part Structured, artifactDir string) string {
	if len(part.Blob) > maxResourceB64Chars {
		return fmt.Sprintf("[MCP embedded resource too large to cache: ~%d bytes, uri=%s]", len(part.Blob)*3/4, safeDetail(part.URI))
	}
	raw, err := base64.StdEncoding.DecodeString(part.Blob)
	if err != nil {
		return fmt.Sprintf("[MCP embedded resource could not be decoded: %s]", firstNonEmpty(part.MimeType, part.URI))
	}
	if len(raw) > maxResourceBytes {
		return fmt.Sprintf("[MCP embedded resource too large to cache: %d bytes, uri=%s]", len(raw), safeDetail(part.URI))
	}
	mimeType := normalizeMIME(part.MimeType)
	if strings.TrimSpace(artifactDir) == "" {
		return fmt.Sprintf("[MCP embedded resource received (%d bytes, %s) but document cache unavailable in this process]", len(raw), firstNonEmpty(mimeType, "unknown type"))
	}
	artifact, err := writeArtifact(artifactDir, resourceFilename(part.URI, mimeType), raw)
	if err != nil {
		return fmt.Sprintf("[MCP embedded resource could not be cached: %s]", firstNonEmpty(mimeType, part.URI))
	}
	return fmt.Sprintf("[MCP resource saved to %s (%s, %d bytes) — read it with read_file or terminal tools]", artifact, firstNonEmpty(mimeType, "unknown type"), len(raw))
}

func writeArtifact(dir, name string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name = safeArtifactName(name)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	if stem == "" {
		stem = "resource"
	}
	file, err := os.CreateTemp(dir, stem+"-*"+ext)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		if !ok {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}
	}()
	if _, err := file.Write(data); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	ok = true
	return file.Name(), nil
}

func resourceFilename(uri, mimeType string) string {
	name := ""
	if parsed, err := url.Parse(uri); err == nil {
		if decoded, err := url.PathUnescape(parsed.Path); err == nil {
			name = path.Base(decoded)
		}
	}
	name = safeArtifactName(name)
	if name == "resource.bin" || filepath.Ext(name) == "" {
		if extensions, err := mime.ExtensionsByType(mimeType); err == nil && len(extensions) > 0 {
			name = strings.TrimSuffix(name, filepath.Ext(name)) + extensions[0]
		}
	}
	return name
}

func safeArtifactName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, strings.TrimSpace(name))
	name = strings.TrimLeft(name, ".")
	if name == "" {
		return "resource.bin"
	}
	if len(name) > 150 {
		ext := filepath.Ext(name)
		name = strings.TrimSuffix(name, ext)
		if len(name) > 130 {
			name = name[:130]
		}
		name += ext
	}
	return name
}

func normalizeMIME(value string) string {
	if parsed, _, err := mime.ParseMediaType(value); err == nil {
		return strings.ToLower(parsed)
	}
	return strings.ToLower(strings.TrimSpace(strings.SplitN(value, ";", 2)[0]))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return safeDetail(value)
		}
	}
	return ""
}

func safeDetail(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
	runes := []rune(value)
	if len(runes) > 200 {
		value = string(runes[:200])
	}
	return value
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
