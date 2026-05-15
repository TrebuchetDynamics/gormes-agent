package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
			return promptModelChoice(os.Stdin, os.Stdout, provider, current, defaultModelCatalogSuggestions(provider))
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
	return cli.HermesModelProviderMenu(), nil
}

func promptProviderChoice(in *os.File, out io.Writer, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
	if in != nil && term.IsTerminal(int(in.Fd())) {
		choices := make([]tuiPickChoice, len(entries))
		for i, e := range entries {
			choices[i] = tuiPickChoice{ID: fmt.Sprintf("%d", i), Label: e.Label}
		}
		selected, err := runBubbleTeaPick(context.Background(), in, out, "Choose a provider", choices, fmt.Sprintf("%d", defaultIndex))
		if err != nil && !bubbleTeaPickShouldFallback(err) {
			return -1, err
		}
		if err == nil && selected == "" {
			return -1, cli.ErrModelPickerCancelled
		}
		if err != nil {
			return promptProviderChoiceText(in, out, entries, defaultIndex)
		}
		idx, _ := strconv.Atoi(selected)
		if idx >= 0 && idx < len(entries) {
			return idx, nil
		}
		return -1, fmt.Errorf("invalid provider choice")
	}
	return promptProviderChoiceText(in, out, entries, defaultIndex)
}

func promptProviderChoiceText(in *os.File, out io.Writer, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Choose a provider")
	for i, entry := range entries {
		num := fmt.Sprintf("%2d)", i+1)
		var marker, line string
		if i == defaultIndex {
			marker = cli.Yellow(out, "*")
			line = cli.BrightCyan(out, num) + " " + cli.Bold(out, entry.Label)
		} else {
			marker = " "
			line = cli.Dim(out, num) + " " + entry.Label
		}
		fmt.Fprintf(out, "%s %s\n", marker, line)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s ", cli.Bold(out, fmt.Sprintf("Select provider [%d] (or q to cancel):", defaultIndex+1)))
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

func defaultModelCatalogSuggestions(provider string) []string {
	return hermes.ProviderModelCatalogSuggestions(provider, nil)
}

func promptModelChoice(in io.Reader, out io.Writer, provider string, current string, suggestions []string) (string, error) {
	if bounded := boundedModelCatalogSuggestions(suggestions, 5); len(bounded) > 0 {
		fmt.Fprintf(out, "Suggested models for %s: %s\n", provider, strings.Join(bounded, ", "))
	}
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

func boundedModelCatalogSuggestions(suggestions []string, max int) []string {
	if max <= 0 {
		return nil
	}
	out := make([]string, 0, min(len(suggestions), max))
	seen := map[string]struct{}{}
	for _, suggestion := range suggestions {
		suggestion = strings.TrimSpace(suggestion)
		key := strings.ToLower(suggestion)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, suggestion)
		if len(out) == max {
			return out
		}
	}
	return out
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
