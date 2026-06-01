package audio

import (
	"context"
	"slices"
	"testing"
)

func TestFacadeEncodePreprocessRoundTrip(t *testing.T) {
	want := PCM{SampleRate: 16000, Samples: []int16{0, 1, -1, 2048, -2048}}
	raw, err := EncodePCM16MonoWAV(want)
	if err != nil {
		t.Fatalf("EncodePCM16MonoWAV: %v", err)
	}

	got, err := Preprocess(context.Background(), raw, "audio/wav", PreprocessOptions{
		FileName: "chunk.wav",
		Converter: func(context.Context, string, string) error {
			t.Fatal("converter should not run for encoded WAV input")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Preprocess encoded WAV: %v", err)
	}
	if got.SampleRate != want.SampleRate || !slices.Equal(got.Samples, want.Samples) {
		t.Fatalf("round-trip PCM = %+v, want %+v", got, want)
	}
}
