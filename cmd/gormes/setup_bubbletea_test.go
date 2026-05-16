package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestSetupInteractiveMenusUseBubbleTeaPicker(t *testing.T) {
	for _, path := range []string{"setup.go", "setup_first_run.go"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		if strings.Contains(text, "cli.NewInteractiveMenu") {
			t.Fatalf("%s still uses the legacy raw-mode menu instead of the Bubble Tea setup picker", path)
		}
		if !strings.Contains(text, "runBubbleTeaPick") {
			t.Fatalf("%s does not route interactive setup selection through the Bubble Tea setup picker", path)
		}
	}
}

func TestSetupProviderChoiceUsesBubbleTeaPickerForTTY(t *testing.T) {
	raw, err := os.ReadFile("setup.go")
	if err != nil {
		t.Fatalf("read setup.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"setupProviderPickerChoices(entries)",
		"runBubbleTeaPick(",
		"promptSetupProviderChoiceText(cmd, entries, defaultIndex)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("setup provider choice missing Bubble Tea routing marker %q", want)
		}
	}
}

func TestModelChoiceUsesBubbleTeaPickerForTTY(t *testing.T) {
	raw, err := os.ReadFile("model.go")
	if err != nil {
		t.Fatalf("read model.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"runBubbleTeaPick(ctx, stdin, out, \"Select model for \"+provider",
		"modelPickerChoices(models)",
		"defaultModelChoiceID(models, current)",
		"promptModelChoiceText(in, out, provider, current, models)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("model choice missing Bubble Tea routing marker %q", want)
		}
	}
}

func TestSetupProviderPickerChoicesUseStableIndices(t *testing.T) {
	choices := setupProviderPickerChoices([]cli.ProviderMenuEntry{
		{ID: "nous", Label: "Nous Portal"},
		{ID: "openai-codex", Label: "OpenAI Codex"},
	})
	if len(choices) != 2 {
		t.Fatalf("choices len = %d, want 2", len(choices))
	}
	if choices[0] != (tuiPickChoice{ID: "0", Label: "Nous Portal"}) {
		t.Fatalf("choice[0] = %#v", choices[0])
	}
	if choices[1] != (tuiPickChoice{ID: "1", Label: "OpenAI Codex"}) {
		t.Fatalf("choice[1] = %#v", choices[1])
	}
}

func TestNoLegacyRawInteractiveMenuImplementation(t *testing.T) {
	root := filepath.Join("..", "..")
	forbidden := []string{
		"New" + "InteractiveMenu",
		"type " + "InteractiveMenu",
		"cli." + "MenuOption",
		"type " + "MenuOption",
		"Prompt" + "YesNo",
		"term." + "MakeRaw",
	}
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(raw)
			for _, needle := range forbidden {
				if strings.Contains(text, needle) {
					t.Fatalf("%s contains legacy raw interactive menu marker %q; interactive selectors must use Bubble Tea", path, needle)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", dir, err)
		}
	}
}
