package tts

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixtureSynthesizerWritesValidWAVWithoutExternalRuntime(t *testing.T) {
	out := filepath.Join(t.TempDir(), "fixture.wav")
	result, err := NewFixtureSynthesizer().Synthesize(context.Background(), Request{
		Text:       "hello from gormes",
		OutputPath: out,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if result.FilePath != out || result.Format != "wav" || result.SampleRate != 16000 || result.Channels != 1 || result.BitsPerSample != 16 {
		t.Fatalf("result = %+v, want 16kHz mono 16-bit wav metadata", result)
	}
	if result.Duration <= 0 || result.Bytes <= 44 {
		t.Fatalf("result = %+v, want non-empty audio payload", result)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) != result.Bytes {
		t.Fatalf("file size = %d, result bytes = %d", len(data), result.Bytes)
	}
	if got := string(data[0:4]); got != "RIFF" {
		t.Fatalf("riff magic = %q", got)
	}
	if got := string(data[8:12]); got != "WAVE" {
		t.Fatalf("wave magic = %q", got)
	}
	if got := binary.LittleEndian.Uint32(data[24:28]); got != 16000 {
		t.Fatalf("sample rate = %d, want 16000", got)
	}
	if got := binary.LittleEndian.Uint16(data[22:24]); got != 1 {
		t.Fatalf("channels = %d, want mono", got)
	}
}

func TestFixtureSynthesizerRejectsUnsupportedInputWithTypedErrors(t *testing.T) {
	out := filepath.Join(t.TempDir(), "fixture.wav")
	_, err := NewFixtureSynthesizer().Synthesize(context.Background(), Request{Text: "   ", OutputPath: out})
	if !IsErrorCode(err, ErrorCodeInvalidInput) {
		t.Fatalf("blank input error = %v, want %s", err, ErrorCodeInvalidInput)
	}

	_, err = NewFixtureSynthesizer().Synthesize(context.Background(), Request{
		Text:          strings.Repeat("a", 8),
		OutputPath:    out,
		MaxTextLength: 4,
	})
	if !IsErrorCode(err, ErrorCodeInvalidInput) {
		t.Fatalf("overlong input error = %v, want %s", err, ErrorCodeInvalidInput)
	}

	disabled := NewFixtureSynthesizer()
	disabled.Disabled = true
	_, err = disabled.Synthesize(context.Background(), Request{Text: "hello", OutputPath: out})
	if !IsErrorCode(err, ErrorCodeProviderUnavailable) {
		t.Fatalf("disabled error = %v, want %s", err, ErrorCodeProviderUnavailable)
	}
}

func BenchmarkFixtureSynthesizer(b *testing.B) {
	synth := NewFixtureSynthesizer()
	for i := 0; i < b.N; i++ {
		out := filepath.Join(b.TempDir(), "fixture.wav")
		result, err := synth.Synthesize(context.Background(), Request{Text: "benchmark local fixture synthesis", OutputPath: out})
		if err != nil {
			b.Fatalf("Synthesize: %v", err)
		}
		b.ReportMetric(float64(result.Bytes), "output_bytes")
		b.ReportMetric(float64(result.Duration.Milliseconds()), "audio_ms")
	}
}
