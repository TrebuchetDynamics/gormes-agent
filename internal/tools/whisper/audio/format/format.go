package format

import (
	"path/filepath"
	"strings"
)

func Extension(mediaType, fileName string) string {
	if ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName))); ext != "" && len(ext) <= 10 {
		return ext
	}
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/ogg", "audio/opus":
		return ".ogg"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	default:
		return ".ogg"
	}
}

func IsWAVExtension(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".wav", ".wave":
		return true
	default:
		return false
	}
}
