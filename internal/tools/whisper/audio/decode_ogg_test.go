//go:build !slim

package audio

import (
	"context"
	"strings"
	"testing"
)

func TestDecodeOGGToPCM_EmptyInput_ReturnsError(t *testing.T) {
	_, err := DecodeOGGToPCM(context.Background(), nil, DecodeConfig{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "audio_decode_format_unsupported") {
		t.Fatalf("error = %q, want audio_decode_format_unsupported evidence", err.Error())
	}
}

func TestDecodeOGGToPCM_NonOGGInput_ReturnsConvertError(t *testing.T) {
	samples, err := DecodeOGGToPCM(context.Background(), []byte("not-ogg-data"), DecodeConfig{})
	if err == nil {
		t.Fatal("expected error for non-OGG input")
	}
	if samples != nil {
		t.Fatalf("DecodeOGGToPCM returned %d samples on error, want nil", len(samples))
	}
}

func TestDecodeOGGToPCM_FFmpegMissingEvidence(t *testing.T) {
	_, err := DecodeOGGToPCM(context.Background(), []byte("OggS\x00fake"), DecodeConfig{})
	if err == nil {
		t.Fatal("expected error for fake OGG data")
	}
	if !strings.Contains(err.Error(), "audio_decode_ffmpeg_missing") {
		t.Fatalf("error = %q, want audio_decode_ffmpeg_missing evidence", err.Error())
	}
}
