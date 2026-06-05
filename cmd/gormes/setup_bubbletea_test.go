package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
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

func TestSetupProviderChoiceUsesSearchableBubbleTeaPickerForTTY(t *testing.T) {
	raw, err := os.ReadFile("setup.go")
	if err != nil {
		t.Fatalf("read setup.go: %v", err)
	}
	text := string(raw)
	start := strings.Index(text, "func promptSetupProviderChoice")
	end := strings.Index(text, "func setupProviderPickerChoices")
	if start < 0 || end <= start {
		t.Fatalf("setup.go missing promptSetupProviderChoice function block")
	}
	providerChoice := text[start:end]
	for _, want := range []string{
		"setupProviderPickerChoices(entries)",
		"runBubbleTeaPickWithOptions(",
		"setupwizard.WithSearchChoices()",
		"promptSetupProviderChoiceText(cmd, entries, defaultIndex)",
	} {
		if !strings.Contains(providerChoice, want) {
			t.Fatalf("setup provider choice missing searchable Bubble Tea routing marker %q", want)
		}
	}
}

func TestModelChoiceUsesBubbleTeaPickerForTTY(t *testing.T) {
	raw, err := os.ReadFile("../../internal/platform/cli/gormescli/model.go")
	if err != nil {
		t.Fatalf("read internal model.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"RunTUIPickWithOptions(ctx, stdin, out, \"Select model for \"+provider",
		"ModelPickerChoices(allModels)",
		"DefaultModelChoiceID(allModels, current)",
		"setupwizard.WithSearchChoices()",
		"PromptModelChoiceText(in, out, provider, current, models)",
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

func TestSetupProviderPickerCodexFilterKeepsProviderLabelsSeparate(t *testing.T) {
	entries, _ := cli.HermesProviderCatalogMenu("")
	byID := make(map[string]string, len(entries))
	choices := setupProviderPickerChoices(entries)
	wizardChoices := make([]setupwizard.Choice, len(choices))
	for i, entry := range entries {
		byID[entry.ID] = entry.Label
		wizardChoices[i] = setupwizard.Choice{ID: entry.ID, Label: choices[i].Label}
		for _, forbidden := range []string{"Codex" + "laude", "OpenAI Codex" + "laude", "OpenAI Codex" + "100+"} {
			if strings.Contains(entry.Label, forbidden) || strings.Contains(choices[i].Label, forbidden) {
				t.Fatalf("provider label for %s contains concatenated stale text %q: entry=%q choice=%q", entry.ID, forbidden, entry.Label, choices[i].Label)
			}
		}
	}

	if got := byID["openai-codex"]; got != "OpenAI Codex" || strings.Contains(got, "Claude") || strings.Contains(got, "100+ models") {
		t.Fatalf("openai-codex label = %q, want clean Codex-only label", got)
	}
	if got := byID["openrouter"]; !strings.HasPrefix(got, "OpenRouter (100+ models") {
		t.Fatalf("openrouter label = %q, want separate OpenRouter row", got)
	}
	if got := byID["anthropic"]; !strings.Contains(got, "Claude Code") {
		t.Fatalf("anthropic label = %q, want Claude Code mentioned only on Anthropic row", got)
	}

	filtered, _ := setupwizard.FilterChoices(wizardChoices, "codex")
	if len(filtered) != 1 || filtered[0].ID != "openai-codex" || filtered[0].Label != "OpenAI Codex" {
		t.Fatalf("codex filter = %#v, want only clean OpenAI Codex row", filtered)
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
