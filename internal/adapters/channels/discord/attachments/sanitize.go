package attachments

import (
	"net/url"
	"path/filepath"
	"strings"
)

// TrustedHost reports whether rawURL is an HTTPS Discord CDN/media URL.
func TrustedHost(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" {
		return false
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "cdn.discordapp.com", "media.discordapp.net":
		return true
	default:
		return false
	}
}

// CleanMediaType strips parameters and normalizes a media type.
func CleanMediaType(mediaType string) string {
	if mediaType = strings.TrimSpace(mediaType); mediaType == "" {
		return ""
	}
	if semi := strings.Index(mediaType, ";"); semi >= 0 {
		mediaType = mediaType[:semi]
	}
	return strings.ToLower(strings.TrimSpace(mediaType))
}

// SafeFileName returns a basename safe for local cache paths and evidence text.
func SafeFileName(fileName string) string {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	var out strings.Builder
	for _, r := range fileName {
		switch {
		case r == 0 || r < 32 || r == 127 || r == '/' || r == '\\':
			out.WriteByte('_')
		default:
			out.WriteRune(r)
		}
	}
	cleaned := strings.Trim(out.String(), " .")
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return ""
	}
	if len(cleaned) <= 160 {
		return cleaned
	}
	ext := filepath.Ext(cleaned)
	stem := strings.TrimSuffix(cleaned, ext)
	if len(ext) > 32 {
		ext = ""
	}
	if len(stem) > 128 {
		stem = stem[:128]
	}
	return stem + ext
}

// SafeToken returns a bounded token safe for a local cache directory name.
func SafeToken(s string) string {
	s = strings.TrimSpace(s)
	var out strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			out.WriteRune(r)
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}
	cleaned := strings.Trim(out.String(), "._-")
	if cleaned == "" {
		return "discord"
	}
	if len(cleaned) > 64 {
		return cleaned[:64]
	}
	return cleaned
}
