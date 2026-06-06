package gormescli

import (
	"strings"
	"testing"
)

func TestSetupTTSNonInteractiveRendersSectionChromeAndChoices(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "tts", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes Setup — Text-to-Speech",
		"Text-to-Speech Provider",
		"Default provider: Edge TTS",
		"Default voice/model: provider default",
		"Built-in/default TTS: Edge TTS",
		"OpenAI TTS",
		"MiniMax TTS",
		"Keep current",
		"Skipped (keeping current)",
		"Text-to-Speech configuration complete!",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "setup_section_unsupported") {
		t.Fatalf("tts section returned unsupported evidence:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestSetupTTSInteractivePersistsListedProvidersWithoutRowBackedError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "minimax\nn\n", "tts")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Default provider: Edge TTS", "Built-in/default TTS: Edge TTS", "Selected provider: MiniMax TTS", "TTS provider set to: MiniMax TTS"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "setup_tts_provider_row_backed") || strings.Contains(stdout+stderr, "Robot") {
		t.Fatalf("TTS setup leaked confusing row-backed/Robot error:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}
