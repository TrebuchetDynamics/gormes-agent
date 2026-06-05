package audio

import "testing"

func TestNormalizeSpeechPCMTrimsSilenceAndKeepsPadding(t *testing.T) {
	pcm := PCM{SampleRate: 10, Samples: []int16{0, 0, 0, 1000, 1200, -1000, 0, 0, 0}}
	got := NormalizeSpeechPCM(pcm)
	if got.SampleRate != pcm.SampleRate {
		t.Fatalf("sample rate = %d, want %d", got.SampleRate, pcm.SampleRate)
	}
	if len(got.Samples) >= len(pcm.Samples) {
		t.Fatalf("normalized sample count = %d, want trimmed below %d: %v", len(got.Samples), len(pcm.Samples), got.Samples)
	}
	if len(got.Samples) < 3 {
		t.Fatalf("normalized samples = %v, want speech retained", got.Samples)
	}
}

func TestNormalizeSpeechPCMPeakNormalizesQuietSpeech(t *testing.T) {
	pcm := PCM{SampleRate: 16000, Samples: []int16{10, -10, 100, -100, 10}}
	got := NormalizeSpeechPCM(pcm)
	peak := 0
	for _, sample := range got.Samples {
		if abs := absInt16(sample); abs > peak {
			peak = abs
		}
	}
	if peak < 20000 {
		t.Fatalf("peak = %d, want quiet speech amplified: %v", peak, got.Samples)
	}
}

func TestNormalizeSpeechPCMRemovesDCOffset(t *testing.T) {
	pcm := PCM{SampleRate: 16000, Samples: []int16{1000, 1100, 900, 1000}}
	got := NormalizeSpeechPCM(pcm)
	var sum int
	for _, sample := range got.Samples {
		sum += int(sample)
	}
	if sum > 4 || sum < -4 {
		t.Fatalf("normalized DC sum = %d samples=%v", sum, got.Samples)
	}
}
