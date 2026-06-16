package setuptts

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRunNonInteractivePrintsCurrentDefaultsAndChoiceList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var out bytes.Buffer
	err := Run(&out, &bytes.Buffer{}, true, Runtime{ShouldPrintStaticChoiceMenu: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Text-to-Speech Provider",
		"Default provider: Edge TTS",
		"Default voice/model: provider default",
		"Built-in/default TTS: Edge TTS",
		"OpenAI TTS",
		"Keep current",
		"Skipped (keeping current)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunInteractiveKeepCurrentChoiceNamesCurrentProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var promptChoices []setup.Choice
	var out, errOut bytes.Buffer
	err := Run(&out, &errOut, false, Runtime{
		LoadConfig: func() (config.Config, error) {
			return config.Config{Runtime: config.RuntimeCfg{TTSProvider: "openai"}}, nil
		},
		PromptChoice: func(title, linePrompt, defaultValue string, choices []setup.Choice) (string, error) {
			promptChoices = append([]setup.Choice(nil), choices...)
			return "keep", nil
		},
		PromptString: func(prompt, defaultValue string) (string, error) { return "n", nil },
	})
	if err != nil {
		t.Fatalf("Run() error = %v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Default provider: OpenAI TTS") {
		t.Fatalf("stdout missing current provider:\n%s", out.String())
	}
	if got := keepChoiceLabel(promptChoices); got != "Keep current (OpenAI TTS)" {
		t.Fatalf("keep choice label = %q, want current provider named", got)
	}
}

func TestRunInteractivePersistsListedProviderWithoutRowBackedError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var prompts []string
	var out, errOut bytes.Buffer
	err := Run(&out, &errOut, false, Runtime{
		PromptChoice: func(title, linePrompt, defaultValue string, choices []setup.Choice) (string, error) {
			prompts = append(prompts, title, linePrompt, defaultValue)
			return "minimax", nil
		},
		PromptString: func(prompt, defaultValue string) (string, error) {
			prompts = append(prompts, prompt, defaultValue)
			return "n", nil
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	for _, want := range []string{"Default provider: Edge TTS", "Built-in/default TTS: Edge TTS", "Selected provider: MiniMax TTS", "TTS provider set to: MiniMax TTS"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String()+errOut.String(), "setup_tts_provider_row_backed") || strings.Contains(out.String()+errOut.String(), "Robot") {
		t.Fatalf("TTS setup leaked confusing row-backed/Robot error:\nstdout=%s\nstderr=%s", out.String(), errOut.String())
	}
	cfg, loadErr := config.Load(nil)
	if loadErr != nil {
		t.Fatalf("load config: %v", loadErr)
	}
	if cfg.Runtime.TTSProvider != "minimax" {
		t.Fatalf("Runtime.TTSProvider = %q, want minimax", cfg.Runtime.TTSProvider)
	}
}

func keepChoiceLabel(choices []setup.Choice) string {
	for _, choice := range choices {
		if choice.Value == "keep" {
			return choice.Label
		}
	}
	return ""
}

func TestVoiceModelUsesProviderSpecificVoice(t *testing.T) {
	got := VoiceModel(map[string]any{
		"edge": map[string]any{"voice": "en-US-AriaNeural"},
	}, "edge")
	if got != "en-US-AriaNeural" {
		t.Fatalf("VoiceModel() = %q", got)
	}
}

func TestRunInvalidProviderReturnsExitCodeTwo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	err := Run(&bytes.Buffer{}, &bytes.Buffer{}, false, Runtime{
		PromptChoice: func(string, string, string, []setup.Choice) (string, error) { return "robot", nil },
		PromptString: func(string, string) (string, error) { return "n", nil },
	})
	if err == nil {
		t.Fatal("Run() error = nil, want invalid provider")
	}
	coded, ok := err.(interface{ ExitCode() int })
	if !ok || coded.ExitCode() != 2 {
		t.Fatalf("ExitCode = %v, want 2 (err=%v)", coded, err)
	}
	if !strings.Contains(err.Error(), `TTS provider "robot" is not available`) {
		t.Fatalf("err = %v", err)
	}
}
