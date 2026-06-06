package format

import "testing"

func TestExtensionPrefersSafeFileNameExtension(t *testing.T) {
	if got := Extension("audio/ogg", "voice.WAV"); got != ".wav" {
		t.Fatalf("Extension = %q, want .wav", got)
	}
}

func TestExtensionFallsBackToMediaType(t *testing.T) {
	tests := map[string]string{
		"audio/wave": ".wav",
		"audio/opus": ".ogg",
		"audio/mp3":  ".mp3",
		"":           ".ogg",
	}
	for mediaType, want := range tests {
		if got := Extension(mediaType, ""); got != want {
			t.Fatalf("Extension(%q) = %q, want %q", mediaType, got, want)
		}
	}
}

func TestIsWAVExtension(t *testing.T) {
	for _, ext := range []string{".wav", ".WAVE", " .wav "} {
		if !IsWAVExtension(ext) {
			t.Fatalf("IsWAVExtension(%q) = false, want true", ext)
		}
	}
	if IsWAVExtension(".ogg") {
		t.Fatal("IsWAVExtension(.ogg) = true, want false")
	}
}
