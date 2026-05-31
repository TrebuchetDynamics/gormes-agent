package providers

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/fallbackconfig"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

type FallbackEntry = fallbackconfig.Entry

type FallbackConfig = fallbackconfig.Config

func NewFallbackCommand(seams ModelCommandSeams) *cobra.Command {
	return NewFallbackCommandWithSeams(seams)
}

func NewFallbackCommandWithSeams(seams ModelCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "fallback",
		Short:        "Manage fallback providers",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFallbackList(cmd)
		},
	}
	addCmd := &cobra.Command{
		Use:          "add",
		Short:        "Pick a provider and model to append to the fallback chain",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFallbackAdd(cmd, seams)
		},
	}
	removeCmd := &cobra.Command{
		Use:          "remove",
		Aliases:      []string{"rm"},
		Short:        "Remove one fallback provider",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFallbackRemove(cmd)
		},
	}
	clearCmd := &cobra.Command{
		Use:          "clear",
		Short:        "Clear the fallback provider chain",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFallbackClear(cmd)
		},
	}
	listCmd := &cobra.Command{
		Use:          "list",
		Aliases:      []string{"ls"},
		Short:        "Show the current fallback chain",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFallbackList(cmd)
		},
	}
	cmd.AddCommand(addCmd)
	cmd.AddCommand(clearCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(removeCmd)
	return cmd
}

func runFallbackList(cmd *cobra.Command) error {
	cfg, err := loadFallbackConfig(config.ConfigPath())
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	if len(cfg.Chain) == 0 {
		fmt.Fprintln(out, "  No fallback providers configured.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Add one with:  gormes fallback add")
		fmt.Fprintln(out)
		return nil
	}
	if cfg.Primary.Provider != "" || cfg.Primary.Model != "" {
		fmt.Fprintf(out, "  Primary:   %s\n", formatFallbackEntry(cfg.Primary))
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "  Fallback chain (%d %s):\n", len(cfg.Chain), pluralizeEntry(len(cfg.Chain)))
	for i, entry := range cfg.Chain {
		fmt.Fprintf(out, "    %d. %s\n", i+1, formatFallbackEntry(entry))
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Tried in order when the primary fails (rate-limit, 5xx, connection errors).")
	fmt.Fprintln(out, "  Docs: https://hermes-agent.nousresearch.com/docs/user-guide/features/fallback-providers")
	fmt.Fprintln(out)
	return nil
}

func runFallbackAdd(cmd *cobra.Command, seams ModelCommandSeams) error {
	chooseModel := normalizedModelChooser(seams)
	var selection cli.Selection
	added := false
	picker := cli.NewModelPicker(cli.ModelPickerOptions{
		IsTTY:          seams.IsTTY,
		LoadCurrent:    seams.LoadCurrent,
		ListProviders:  seams.ListProviders,
		ChooseProvider: seams.ChooseProvider,
		ChooseModel:    chooseModel,
		PersistSelection: func(picked cli.Selection) error {
			selection = picked
			wrote, err := AppendFallbackSelection(config.ConfigPath(), picked)
			added = wrote
			return err
		},
	})
	if _, err := picker.Pick(cmd.Context()); err != nil {
		return fmt.Errorf("gormes fallback add: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	if !added {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s is already in the fallback chain — skipped.\n", formatFallbackEntry(FallbackEntry{
			Provider: selection.Provider,
			Model:    selection.Model,
		}))
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Added fallback: %s\n", formatFallbackEntry(FallbackEntry{
		Provider: selection.Provider,
		Model:    selection.Model,
	}))
	cfg, err := loadFallbackConfig(config.ConfigPath())
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Chain is now %d %s long.\n", len(cfg.Chain), pluralizeEntry(len(cfg.Chain)))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "  Run `gormes fallback list` to view, or `gormes fallback remove` to delete.")
	return nil
}

func runFallbackRemove(cmd *cobra.Command) error {
	cfg, err := loadFallbackConfig(config.ConfigPath())
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if len(cfg.Chain) == 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  No fallback providers configured — nothing to remove.")
		fmt.Fprintln(out)
		return nil
	}
	fmt.Fprintln(out, "Select a fallback to remove:")
	for i, entry := range cfg.Chain {
		fmt.Fprintf(out, "  %d. %s\n", i+1, formatFallbackEntry(entry))
	}
	fmt.Fprintf(out, "  %d. Cancel\n", len(cfg.Chain)+1)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Choice [1-%d]: ", len(cfg.Chain)+1)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && !textvalue.IsNonBlank(line) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Cancelled — no change.")
		return nil
	}
	choice := strings.TrimSpace(line)
	if choice == "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Cancelled — no change.")
		return nil
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(cfg.Chain)+1 {
		return fmt.Errorf("fallback remove: invalid choice %q", choice)
	}
	if idx == len(cfg.Chain)+1 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Cancelled — no change.")
		return nil
	}
	removed := cfg.Chain[idx-1]
	cfg.Chain = append(cfg.Chain[:idx-1], cfg.Chain[idx:]...)
	if err := WriteFallbackChain(config.ConfigPath(), cfg.Chain); err != nil {
		return err
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  Removed fallback: %s\n", formatFallbackEntry(removed))
	if len(cfg.Chain) == 0 {
		fmt.Fprintln(out, "  Fallback chain is now empty.")
	} else {
		fmt.Fprintf(out, "  Chain is now %d %s long.\n", len(cfg.Chain), pluralizeEntry(len(cfg.Chain)))
	}
	fmt.Fprintln(out)
	return nil
}

func runFallbackClear(cmd *cobra.Command) error {
	cfg, err := loadFallbackConfig(config.ConfigPath())
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if len(cfg.Chain) == 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  No fallback providers configured — nothing to clear.")
		fmt.Fprintln(out)
		return nil
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  Current fallback chain (%d %s):\n", len(cfg.Chain), pluralizeEntry(len(cfg.Chain)))
	for i, entry := range cfg.Chain {
		fmt.Fprintf(out, "    %d. %s\n", i+1, formatFallbackEntry(entry))
	}
	fmt.Fprintln(out)
	fmt.Fprint(out, "  Clear all entries? [y/N]: ")
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && !textvalue.IsNonBlank(line) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Cancelled.")
		return nil
	}
	answer := textvalue.LowerTrim(line)
	if answer != "y" && answer != "yes" {
		fmt.Fprintln(out, "  Cancelled — no change.")
		return nil
	}
	if err := WriteFallbackChain(config.ConfigPath(), nil); err != nil {
		return err
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Fallback chain cleared.")
	fmt.Fprintln(out)
	return nil
}

func LoadFallbackConfig(path string) (FallbackConfig, error) {
	return fallbackconfig.Load(path)
}

func loadFallbackConfig(path string) (FallbackConfig, error) {
	return LoadFallbackConfig(path)
}

func AppendFallbackSelection(path string, selection cli.Selection) (bool, error) {
	entry := FallbackEntry{
		Provider: strings.TrimSpace(selection.Provider),
		Model:    strings.TrimSpace(selection.Model),
	}
	if entry.Provider == "" || entry.Model == "" {
		return false, cli.ErrSelectorNoMatch
	}
	return fallbackconfig.Append(path, entry)
}

func WriteFallbackChain(path string, chain []FallbackEntry) error {
	return fallbackconfig.WriteChain(path, chain)
}

func formatFallbackEntry(entry FallbackEntry) string {
	provider := entry.Provider
	if provider == "" {
		provider = "?"
	}
	model := entry.Model
	if model == "" {
		model = "?"
	}
	out := fmt.Sprintf("%s  (via %s)", model, provider)
	if entry.BaseURL != "" {
		out += "  [" + entry.BaseURL + "]"
	}
	return out
}

func pluralizeEntry(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}
