package attachments

import (
	"bytes"
	"errors"
	"mime"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
)

const inlineTextDocumentBytes int64 = 100 * 1024

var ErrInvalidPayload = errors.New("discord attachment invalid payload")

// Classification is the gateway-facing classification for a Discord attachment.
type Classification struct {
	Kind      string
	MediaType string
	Ext       string
	FileName  string
}

// Classify returns the supported attachment classification for a Discord message attachment.
func Classify(att *discordgo.MessageAttachment) (Classification, bool) {
	if att == nil {
		return Classification{}, false
	}
	mediaType := CleanMediaType(att.ContentType)
	fileName := SafeFileName(att.Filename)
	ext := InferExt(fileName, mediaType)
	if mediaType == "" {
		mediaType = mime.TypeByExtension(ext)
		mediaType = CleanMediaType(mediaType)
	}
	switch {
	case strings.HasPrefix(mediaType, "image/"):
		if ext == "" {
			ext = ImageExt(mediaType)
		}
		if !ImageExtSupported(ext) {
			ext = ".jpg"
		}
		if fileName == "" {
			fileName = "image" + ext
		}
		return Classification{Kind: "image", MediaType: mediaType, Ext: ext, FileName: fileName}, true
	case strings.HasPrefix(mediaType, "audio/"):
		if ext == "" {
			ext = AudioExt(mediaType)
		}
		if !AudioExtSupported(ext) {
			ext = ".ogg"
		}
		if fileName == "" {
			fileName = "audio" + ext
		}
		return Classification{Kind: "audio", MediaType: mediaType, Ext: ext, FileName: fileName}, true
	default:
		if ext == "" {
			return Classification{}, false
		}
		docType, ok := channelutil.DocumentMediaTypeForExtension(ext)
		if !ok {
			return Classification{}, false
		}
		if mediaType == "" || mediaType == "application/octet-stream" {
			mediaType = docType
		}
		if fileName == "" {
			fileName = "document" + ext
		}
		return Classification{Kind: "document", MediaType: mediaType, Ext: ext, FileName: fileName}, true
	}
}

// InferExt infers a supported extension from a file name or media type.
func InferExt(fileName, mediaType string) string {
	if ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))); ext != "" {
		return ext
	}
	mediaType = CleanMediaType(mediaType)
	if ext := channelutil.MIMEExtensionFallback(mediaType); ext != "" {
		return ext
	}
	extensions, _ := mime.ExtensionsByType(mediaType)
	sort.Strings(extensions)
	for _, ext := range extensions {
		ext = strings.ToLower(ext)
		if _, ok := channelutil.DocumentMediaTypeForExtension(ext); ok || ImageExtSupported(ext) || AudioExtSupported(ext) {
			return ext
		}
	}
	return ""
}

// ValidatePayload rejects empty, HTML/error, or signature-mismatched payloads.
func ValidatePayload(classification Classification, data []byte) error {
	if len(data) == 0 {
		return ErrInvalidPayload
	}
	if LooksLikeHTMLPayload(data) {
		return ErrInvalidPayload
	}
	switch classification.Kind {
	case "image":
		if !LooksLikeImage(data) {
			return ErrInvalidPayload
		}
	case "audio":
		if !LooksLikeAudio(data) {
			return ErrInvalidPayload
		}
	case "document":
		switch classification.Ext {
		case ".pdf":
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				return ErrInvalidPayload
			}
		case ".zip", ".docx", ".xlsx", ".pptx":
			if !bytes.HasPrefix(data, []byte("PK")) {
				return ErrInvalidPayload
			}
		}
	}
	return nil
}

// ShouldInlineDocument reports whether a small text document should be prepended to the message text.
func ShouldInlineDocument(ext string, size int64) bool {
	if size <= 0 || size > inlineTextDocumentBytes {
		return false
	}
	switch strings.ToLower(ext) {
	case ".log", ".md", ".txt":
		return true
	default:
		return false
	}
}

func LooksLikeHTMLPayload(data []byte) bool {
	trimmed := bytes.TrimSpace(bytes.ToLower(data))
	return bytes.HasPrefix(trimmed, []byte("<html")) ||
		bytes.HasPrefix(trimmed, []byte("<!doctype html")) ||
		bytes.Contains(trimmed[:min(len(trimmed), 256)], []byte("<title>forbidden"))
}

func LooksLikeImage(data []byte) bool {
	return bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) ||
		bytes.HasPrefix(data, []byte("\xff\xd8\xff")) ||
		bytes.HasPrefix(data, []byte("GIF87a")) ||
		bytes.HasPrefix(data, []byte("GIF89a")) ||
		bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 12 && string(data[8:12]) == "WEBP"
}

func LooksLikeAudio(data []byte) bool {
	return bytes.HasPrefix(data, []byte("OggS")) ||
		bytes.HasPrefix(data, []byte("ID3")) ||
		bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 12 && string(data[8:12]) == "WAVE" ||
		bytes.HasPrefix(data, []byte("fLaC")) ||
		len(data) >= 2 && data[0] == 0xff && data[1]&0xe0 == 0xe0
}

func ImageExt(mediaType string) string { return channelutil.ImageExtensionForMediaType(mediaType) }

func ImageExtSupported(ext string) bool { return channelutil.ImageExtensionSupported(ext) }

func AudioExt(mediaType string) string {
	switch CleanMediaType(mediaType) {
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/webm":
		return ".webm"
	case "audio/mp4", "audio/aac":
		return ".m4a"
	default:
		return ".ogg"
	}
}

func AudioExtSupported(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".aac", ".flac", ".m4a", ".mp3", ".ogg", ".opus", ".wav", ".webm":
		return true
	default:
		return false
	}
}
