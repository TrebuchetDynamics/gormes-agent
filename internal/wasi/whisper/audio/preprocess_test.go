package audio

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreprocess_ReturnsWAVPCMDirectly(t *testing.T) {
	input := testWAVPCM16Mono16k(t, []int16{0, 1024, -1024, 32767, -32768})

	got, err := Preprocess(context.Background(), input, "audio/wav", PreprocessOptions{
		FileName: "voice.wav",
		Converter: func(context.Context, string, string) error {
			t.Fatal("converter should not run for compatible WAV input")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if got.SampleRate != 16000 {
		t.Fatalf("SampleRate = %d, want 16000", got.SampleRate)
	}
	want := []int16{0, 1024, -1024, 32767, -32768}
	if !equalInt16(got.Samples, want) {
		t.Fatalf("Samples = %v, want %v", got.Samples, want)
	}
}

func TestPreprocess_ConvertsOGGToPCMWhenConverterAvailable(t *testing.T) {
	wantInput := []byte("OggS\x00fake-opus")
	wantSamples := []int16{12, 34, -56}
	var convertedFrom string
	var convertedBytes []byte

	got, err := Preprocess(context.Background(), wantInput, "audio/ogg", PreprocessOptions{
		FileName: "voice.oga",
		Converter: func(_ context.Context, inputPath, outputPath string) error {
			convertedFrom = inputPath
			var readErr error
			convertedBytes, readErr = os.ReadFile(inputPath)
			if readErr != nil {
				return readErr
			}
			return os.WriteFile(outputPath, testWAVPCM16Mono16k(t, wantSamples), 0o600)
		},
	})
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if !strings.HasSuffix(convertedFrom, ".oga") {
		t.Fatalf("converted input path = %q, want .oga suffix", convertedFrom)
	}
	if string(convertedBytes) != string(wantInput) {
		t.Fatalf("converter input bytes = %q, want %q", convertedBytes, wantInput)
	}
	if got.SampleRate != 16000 || !equalInt16(got.Samples, wantSamples) {
		t.Fatalf("PCM = %+v, want 16000/%v", got, wantSamples)
	}
	if _, statErr := os.Stat(convertedFrom); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("temp input should be removed after preprocessing, stat err = %v", statErr)
	}
}

func TestPreprocess_ReturnsDegradedWhenFFmpegMissing(t *testing.T) {
	t.Setenv("PATH", filepath.Join(t.TempDir(), "missing-bin"))

	_, err := Preprocess(context.Background(), []byte("OggS"), "audio/ogg", PreprocessOptions{FileName: "voice.ogg"})
	if err == nil {
		t.Fatal("Preprocess returned nil error without ffmpeg")
	}
	var preprocessErr *PreprocessError
	if !errors.As(err, &preprocessErr) {
		t.Fatalf("error = %T %[1]v, want *PreprocessError", err)
	}
	if preprocessErr.Code != AudioPreprocessUnavailable {
		t.Fatalf("Code = %q, want %q", preprocessErr.Code, AudioPreprocessUnavailable)
	}
	if strings.Contains(err.Error(), string(os.PathSeparator)+"voice.ogg") {
		t.Fatalf("error leaked full temp path: %v", err)
	}
}

func TestPreprocess_RedactsConverterTempPaths(t *testing.T) {
	_, err := Preprocess(context.Background(), []byte("OggS"), "audio/ogg", PreprocessOptions{
		FileName: "voice.ogg",
		Converter: func(_ context.Context, inputPath, outputPath string) error {
			return fmt.Errorf("converter failed for %s -> %s", inputPath, outputPath)
		},
	})
	if err == nil {
		t.Fatal("Preprocess returned nil error for converter failure")
	}
	if strings.Contains(err.Error(), string(os.PathSeparator)+"input.ogg") ||
		strings.Contains(err.Error(), string(os.PathSeparator)+"input.wav") {
		t.Fatalf("error leaked full temp paths: %v", err)
	}
	for _, want := range []string{"input.ogg", "input.wav"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want basename %q", err, want)
		}
	}
}

func testWAVPCM16Mono16k(t *testing.T, samples []int16) []byte {
	t.Helper()
	dataSize := len(samples) * 2
	raw := make([]byte, 44+dataSize)
	copy(raw[0:4], "RIFF")
	binary.LittleEndian.PutUint32(raw[4:8], uint32(36+dataSize))
	copy(raw[8:12], "WAVE")
	copy(raw[12:16], "fmt ")
	binary.LittleEndian.PutUint32(raw[16:20], 16)
	binary.LittleEndian.PutUint16(raw[20:22], 1)
	binary.LittleEndian.PutUint16(raw[22:24], 1)
	binary.LittleEndian.PutUint32(raw[24:28], 16000)
	binary.LittleEndian.PutUint32(raw[28:32], 16000*2)
	binary.LittleEndian.PutUint16(raw[32:34], 2)
	binary.LittleEndian.PutUint16(raw[34:36], 16)
	copy(raw[36:40], "data")
	binary.LittleEndian.PutUint32(raw[40:44], uint32(dataSize))
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(raw[44+(i*2):46+(i*2)], uint16(sample))
	}
	return raw
}

func equalInt16(left, right []int16) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
