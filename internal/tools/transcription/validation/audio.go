//go:build !slim

// Package validation owns audio-file validation constraints for transcription.
package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	EvidenceAudioNotFound          = "audio_not_found"
	EvidenceAudioNotFile           = "audio_not_file"
	EvidenceUnsupportedAudioFormat = "unsupported_audio_format"
	EvidenceAudioTooLarge          = "audio_too_large"
)

// Result describes a validation failure. An empty Evidence means valid.
type Result struct {
	Evidence string
	Message  string
}

// Audio verifies that path names a supported regular audio file within maxBytes.
func Audio(path string, maxBytes int64) Result {
	if path == "" {
		return Result{Evidence: EvidenceAudioNotFound, Message: "audio path is required"}
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{Evidence: EvidenceAudioNotFound, Message: "audio file not found"}
		}
		return Result{Evidence: EvidenceAudioNotFound, Message: err.Error()}
	}
	if !info.Mode().IsRegular() {
		return Result{Evidence: EvidenceAudioNotFile, Message: "audio path is not a regular file"}
	}
	if !supportedExt(filepath.Ext(path)) {
		return Result{Evidence: EvidenceUnsupportedAudioFormat, Message: "unsupported audio format"}
	}
	if info.Size() > maxBytes {
		return Result{Evidence: EvidenceAudioTooLarge, Message: fmt.Sprintf("audio file exceeds max bytes (%d)", maxBytes)}
	}
	return Result{}
}

func supportedExt(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm", ".ogg", ".aac", ".flac":
		return true
	default:
		return false
	}
}
