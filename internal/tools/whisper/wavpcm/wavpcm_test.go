package wavpcm

import "testing"

func TestEncodePCM16MonoWAVRoundTrip(t *testing.T) {
	want := PCM{SampleRate: 16000, Samples: []int16{0, 1, -1, 2048, -2048}}
	raw, err := EncodePCM16MonoWAV(want)
	if err != nil {
		t.Fatalf("EncodePCM16MonoWAV: %v", err)
	}

	got, err := DecodePCM16Mono16kWAV(raw)
	if err != nil {
		t.Fatalf("DecodePCM16Mono16kWAV: %v", err)
	}
	if got.SampleRate != want.SampleRate || !equalInt16(got.Samples, want.Samples) {
		t.Fatalf("round-trip PCM = %+v, want %+v", got, want)
	}
}

func TestEncodePCM16MonoWAVRequiresSampleRate(t *testing.T) {
	if _, err := EncodePCM16MonoWAV(PCM{}); err == nil {
		t.Fatal("EncodePCM16MonoWAV returned nil error without sample rate")
	}
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
