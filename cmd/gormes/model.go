package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

type modelCommandSeams struct {
	IsTTY            func() bool
	LoadCurrent      func() (cli.ProviderModel, error)
	ListProviders    func() ([]cli.ProviderMenuEntry, error)
	ChooseProvider   func(entries []cli.ProviderMenuEntry, defaultIndex int) (int, error)
	ChooseModel      func(provider string, current string) (string, error)
	PersistSelection func(cli.Selection) error
}

func newModelCommand() *cobra.Command {
	return newModelCommandWithSeams(defaultModelCommandSeams())
}

func newModelCommandWithSeams(seams modelCommandSeams) *cobra.Command {
	chooseModel := seams.ChooseModel
	if chooseModel != nil {
		chooseModel = func(provider string, current string) (string, error) {
			model, err := seams.ChooseModel(provider, current)
			if err != nil {
				return "", err
			}
			return hermes.NormalizeProviderModelID(provider, model), nil
		}
	}
	cmd := &cobra.Command{
		Use:          "model",
		Short:        "Interactively select the active model/provider",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			picker := cli.NewModelPicker(cli.ModelPickerOptions{
				IsTTY:            seams.IsTTY,
				LoadCurrent:      seams.LoadCurrent,
				ListProviders:    seams.ListProviders,
				ChooseProvider:   seams.ChooseProvider,
				ChooseModel:      chooseModel,
				PersistSelection: seams.PersistSelection,
			})
			selection, err := picker.Pick(cmd.Context())
			if err != nil {
				return fmt.Errorf("gormes model: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "model selection saved: provider=%s model=%s\n", selection.Provider, selection.Model)
			fmt.Fprintf(cmd.OutOrStdout(), "Provider auth was not changed. If credentials are missing, run: gormes auth add %s\n", selection.Provider)
			return nil
		},
	}
	return cmd
}

func defaultModelCommandSeams() modelCommandSeams {
	return modelCommandSeams{
		IsTTY: isStdinTTY,
		LoadCurrent: func() (cli.ProviderModel, error) {
			cfg, err := config.Load(nil)
			if err != nil {
				return cli.ProviderModel{}, err
			}
			return cli.ProviderModel{Provider: cfg.Hermes.Provider, Model: cfg.Hermes.Model}, nil
		},
		ListProviders: defaultModelProviderEntries,
		ChooseProvider: func(entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
			return promptProviderChoice(os.Stdin, os.Stdout, entries, defaultIndex)
		},
		ChooseModel: func(provider string, current string) (string, error) {
			return promptModelChoice(os.Stdin, os.Stdout, provider, current)
		},
		PersistSelection: persistModelSelectionToConfig,
	}
}

func isStdinTTY() bool {
	return stdinIsTerminal(os.Stdin)
}

func stdinIsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func defaultModelProviderEntries() ([]cli.ProviderMenuEntry, error) {
	entries := make([]cli.ProviderMenuEntry, 0)
	for _, entry := range hermes.HermesProviderRegistryManifest() {
		if entry.ImplementationStatus == hermes.ProviderExcluded {
			continue
		}
		label := entry.ID
		if entry.AuthType != "" {
			label += " (" + entry.AuthType + ")"
		}
		entries = append(entries, cli.ProviderMenuEntry{ID: entry.ID, Label: label})
	}
	return entries, nil
}

func promptProviderChoice(in *os.File, out *os.File, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
	for i, entry := range entries {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}
		fmt.Fprintf(out, "%s %d) %s\n", marker, i+1, entry.Label)
	}
	fmt.Fprintf(out, "Select provider [%d] (or q to cancel): ", defaultIndex+1)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return -1, err
	}
	answer := strings.TrimSpace(line)
	if answer == "" {
		return defaultIndex, nil
	}
	if strings.EqualFold(answer, "q") || strings.EqualFold(answer, "cancel") {
		return -1, cli.ErrModelPickerCancelled
	}
	idx, err := strconv.Atoi(answer)
	if err != nil || idx < 1 || idx > len(entries) {
		return -1, fmt.Errorf("invalid provider choice")
	}
	return idx - 1, nil
}

func promptModelChoice(in *os.File, out *os.File, provider string, current string) (string, error) {
	if strings.TrimSpace(current) != "" {
		fmt.Fprintf(out, "Model for %s [%s] (or q to cancel): ", provider, current)
	} else {
		fmt.Fprintf(out, "Model for %s (or q to cancel): ", provider)
	}
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return "", err
	}
	answer := strings.TrimSpace(line)
	if strings.EqualFold(answer, "q") || strings.EqualFold(answer, "cancel") {
		return "", cli.ErrModelPickerCancelled
	}
	if answer == "" {
		answer = strings.TrimSpace(current)
	}
	if answer == "" {
		return "", cli.ErrSelectorNoMatch
	}
	return answer, nil
}

func persistModelSelectionToConfig(selection cli.Selection) error {
	provider := strings.TrimSpace(selection.Provider)
	model := strings.TrimSpace(selection.Model)
	if provider == "" || model == "" {
		return cli.ErrSelectorNoMatch
	}
	path := config.ConfigPath()
	document := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := toml.Unmarshal(data, &document); err != nil {
			return fmt.Errorf("decode config: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read config: %w", err)
	}
	hermesSection, _ := document["hermes"].(map[string]any)
	if hermesSection == nil {
		hermesSection = map[string]any{}
	}
	hermesSection["model"] = model
	hermesSection["provider"] = provider
	document["hermes"] = hermesSection
	out, err := toml.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
