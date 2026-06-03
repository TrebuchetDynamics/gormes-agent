package setup

import "testing"

func TestSetupChoiceOptionsPreserveDisplayOrder(t *testing.T) {
	tts := TTSProviderOptions()
	if len(tts) != 9 || tts[0].Value != "edge" || tts[len(tts)-1].Value != "keep" {
		t.Fatalf("TTSProviderOptions order = %#v", tts)
	}
	terminal := TerminalBackendOptions()
	if len(terminal) != 7 || terminal[0].Value != "local" || terminal[len(terminal)-1].Value != "keep" {
		t.Fatalf("TerminalBackendOptions order = %#v", terminal)
	}
}

func TestSetupOptionLabels(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  string
		want string
	}{
		{name: "terminal local", got: TerminalBackendLabel("local"), want: "Local"},
		{name: "terminal apptainer", got: TerminalBackendLabel("apptainer"), want: "Singularity/Apptainer"},
		{name: "terminal unknown", got: TerminalBackendLabel("custom"), want: "custom"},
		{name: "tts edge", got: TTSProviderLabel("edge"), want: "Edge TTS"},
		{name: "tts gemini", got: TTSProviderLabel("gemini"), want: "Google Gemini TTS"},
		{name: "tts unknown", got: TTSProviderLabel("custom"), want: "custom"},
	} {
		if tc.got != tc.want {
			t.Fatalf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestSetupPromptParsingAndKnownModes(t *testing.T) {
	if got, ok := ParsePositiveInt(" 42 "); !ok || got != 42 {
		t.Fatalf("ParsePositiveInt = %d, %v, want 42 true", got, ok)
	}
	if _, ok := ParsePositiveInt("0"); ok {
		t.Fatal("ParsePositiveInt(0) ok = true, want false")
	}
	if got, ok := ParseCompressionThreshold("0.75"); !ok || got != 0.75 {
		t.Fatalf("ParseCompressionThreshold = %v, %v, want 0.75 true", got, ok)
	}
	if _, ok := ParseCompressionThreshold("0.49"); ok {
		t.Fatal("ParseCompressionThreshold(0.49) ok = true, want false")
	}
	if !IsKnownToolProgressMode("verbose") || IsKnownToolProgressMode("chatty") {
		t.Fatal("IsKnownToolProgressMode mismatch")
	}
	if got := ToolProgressModeIndex("all"); got != 2 {
		t.Fatalf("ToolProgressModeIndex(all) = %d, want 2", got)
	}
	if got := ToolProgressModeIndex("missing"); got != -1 {
		t.Fatalf("ToolProgressModeIndex(missing) = %d, want -1", got)
	}
	if !IsKnownSessionResetPolicy("none") || IsKnownSessionResetPolicy("weekly") {
		t.Fatal("IsKnownSessionResetPolicy mismatch")
	}
}
