//go:build !gormes_lite && !slim

// Package local owns pure transcript helpers for local Whisper STT.
package local

import "strings"

// StitchChunkTranscripts joins Whisper chunk transcripts while removing repeated
// words introduced by chunk overlap.
func StitchChunkTranscripts(transcripts []string) string {
	stitched := ""
	for _, transcript := range transcripts {
		transcript = strings.TrimSpace(transcript)
		if transcript == "" {
			continue
		}
		if stitched == "" {
			stitched = transcript
			continue
		}
		stitched = appendTranscriptWithOverlap(stitched, transcript)
	}
	return stitched
}

func appendTranscriptWithOverlap(previous, next string) string {
	prevFields := strings.Fields(previous)
	nextFields := strings.Fields(next)
	if len(prevFields) == 0 {
		return strings.TrimSpace(next)
	}
	if len(nextFields) == 0 {
		return strings.TrimSpace(previous)
	}
	maxOverlap := len(prevFields)
	if len(nextFields) < maxOverlap {
		maxOverlap = len(nextFields)
	}
	for n := maxOverlap; n > 0; n-- {
		if equalFoldWords(prevFields[len(prevFields)-n:], nextFields[:n]) {
			if n == len(nextFields) {
				return strings.TrimSpace(previous)
			}
			return strings.TrimSpace(previous + " " + strings.Join(nextFields[n:], " "))
		}
	}
	return strings.TrimSpace(previous + "\n" + next)
}

func equalFoldWords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.Trim(strings.ToLower(a[i]), `.,!?;:"'()[]{}-`) != strings.Trim(strings.ToLower(b[i]), `.,!?;:"'()[]{}-`) {
			return false
		}
	}
	return true
}
