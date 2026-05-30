package modelselection

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestModelPickerRequiresTTY(t *testing.T) {
	picker := NewModelPicker(ModelPickerOptions{
		IsTTY: func() bool { return false },
		LoadCurrent: func() (ProviderModel, error) {
			t.Fatal("LoadCurrent must not be called for non-TTY invocation")
			return ProviderModel{}, nil
		},
	})

	_, err := picker.Pick(context.Background())
	if !errors.Is(err, ErrModelPickerRequiresTTY) {
		t.Fatalf("Pick error = %v, want ErrModelPickerRequiresTTY", err)
	}
}

func TestModelPickerProviderMenuShowsCurrent(t *testing.T) {
	var labels []string
	picker := NewModelPicker(ModelPickerOptions{
		IsTTY:       func() bool { return true },
		LoadCurrent: func() (ProviderModel, error) { return ProviderModel{Provider: "openai-codex", Model: "gpt-5.5"}, nil },
		ListProviders: func() ([]ProviderMenuEntry, error) {
			return []ProviderMenuEntry{{ID: "anthropic", Label: "Anthropic"}, {ID: "openai-codex", Label: "OpenAI Codex"}}, nil
		},
		ChooseProvider: func(entries []ProviderMenuEntry, defaultIndex int) (int, error) {
			for _, entry := range entries {
				labels = append(labels, entry.Label)
			}
			if defaultIndex != 1 {
				t.Fatalf("defaultIndex = %d, want 1", defaultIndex)
			}
			return -1, ErrModelPickerCancelled
		},
	})

	_, err := picker.Pick(context.Background())
	if !errors.Is(err, ErrModelPickerCancelled) {
		t.Fatalf("Pick error = %v, want ErrModelPickerCancelled", err)
	}
	if len(labels) != 2 || !strings.Contains(labels[1], "currently active") {
		t.Fatalf("labels = %#v, want active provider annotated", labels)
	}
}

func TestModelPickerSelectionOnlyDoesNotInvokeAuthFlow(t *testing.T) {
	authCalled := false
	picker := NewModelPicker(ModelPickerOptions{
		IsTTY:       func() bool { return true },
		LoadCurrent: func() (ProviderModel, error) { return ProviderModel{Provider: "anthropic", Model: "old"}, nil },
		ListProviders: func() ([]ProviderMenuEntry, error) {
			return []ProviderMenuEntry{{ID: "openai-codex", Label: "OpenAI Codex"}}, nil
		},
		ChooseProvider: func([]ProviderMenuEntry, int) (int, error) { return 0, nil },
		ChooseModel:    func(provider string, current string) (string, error) { return "gpt-5.5", nil },
		PersistSelection: func(selection Selection) error {
			if selection.Provider != "openai-codex" || selection.Model != "gpt-5.5" {
				t.Fatalf("selection = %#v", selection)
			}
			return nil
		},
		AuthFlow: func(provider string) error {
			authCalled = true
			return nil
		},
	})

	selection, err := picker.Pick(context.Background())
	if err != nil {
		t.Fatalf("Pick error = %v", err)
	}
	if authCalled {
		t.Fatal("Pick invoked auth flow; model picker must be selection-only")
	}
	if selection.Provider != "openai-codex" || selection.Model != "gpt-5.5" {
		t.Fatalf("selection = %#v", selection)
	}
}

func TestModelPickerPersistsModelAndProvider(t *testing.T) {
	var persisted Selection
	picker := NewModelPicker(ModelPickerOptions{
		IsTTY: func() bool { return true },
		LoadCurrent: func() (ProviderModel, error) {
			return ProviderModel{Provider: "anthropic", Model: "claude-sonnet-4"}, nil
		},
		ListProviders: func() ([]ProviderMenuEntry, error) {
			return []ProviderMenuEntry{{ID: "anthropic", Label: "Anthropic"}}, nil
		},
		ChooseProvider: func([]ProviderMenuEntry, int) (int, error) { return 0, nil },
		ChooseModel:    func(provider string, current string) (string, error) { return "claude-opus-4-1", nil },
		PersistSelection: func(selection Selection) error {
			persisted = selection
			return nil
		},
	})

	selection, err := picker.Pick(context.Background())
	if err != nil {
		t.Fatalf("Pick error = %v", err)
	}
	want := Selection{Provider: "anthropic", Model: "claude-opus-4-1"}
	if !reflect.DeepEqual(selection, want) || !reflect.DeepEqual(persisted, want) {
		t.Fatalf("selection=%#v persisted=%#v want=%#v", selection, persisted, want)
	}
}

func TestModelPickerCancellationLeavesConfigUntouched(t *testing.T) {
	persistCalled := false
	picker := NewModelPicker(ModelPickerOptions{
		IsTTY: func() bool { return true },
		LoadCurrent: func() (ProviderModel, error) {
			return ProviderModel{Provider: "anthropic", Model: "claude-sonnet-4"}, nil
		},
		ListProviders: func() ([]ProviderMenuEntry, error) {
			return []ProviderMenuEntry{{ID: "anthropic", Label: "Anthropic"}}, nil
		},
		ChooseProvider: func([]ProviderMenuEntry, int) (int, error) { return 0, nil },
		ChooseModel:    func(provider string, current string) (string, error) { return "", ErrModelPickerCancelled },
		PersistSelection: func(selection Selection) error {
			persistCalled = true
			return nil
		},
	})

	_, err := picker.Pick(context.Background())
	if !errors.Is(err, ErrModelPickerCancelled) {
		t.Fatalf("Pick error = %v, want ErrModelPickerCancelled", err)
	}
	if persistCalled {
		t.Fatal("cancelled picker persisted selection")
	}
}

func TestCuratorAuxiliarySlot_ModelPickerTaskRegistry(t *testing.T) {
	tasks := DefaultAuxiliaryTaskEntries()
	for _, task := range tasks {
		if task.Key == "curator" {
			if task.Label == "" || task.Description == "" {
				t.Fatalf("curator task = %#v, want label and description", task)
			}
			return
		}
	}
	t.Fatalf("DefaultAuxiliaryTaskEntries missing curator: %#v", tasks)
}
