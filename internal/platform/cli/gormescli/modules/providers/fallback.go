package providers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

type FallbackEntry struct {
	Provider string
	Model    string
	BaseURL  string
	APIMode  string
}

type FallbackConfig struct {
	Primary FallbackEntry
	Chain   []FallbackEntry
}

type ModelCommandSeams struct {
	IsTTY            func() bool
	LoadCurrent      func() (cli.ProviderModel, error)
	ListProviders    func() ([]cli.ProviderMenuEntry, error)
	ChooseProvider   func(entries []cli.ProviderMenuEntry, defaultIndex int) (int, error)
	ChooseModel      func(provider string, current string) (string, error)
	PersistSelection func(cli.Selection) error
}

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
	chooseModel := seams.ChooseModel
	if chooseModel != nil {
		chooseModel = func(provider string, current string) (string, error) {
			model, err := seams.ChooseModel(provider, current)
			if err != nil {
				return "", err
			}
			return llm.NormalizeProviderModelID(provider, model), nil
		}
	}
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
	if err != nil && strings.TrimSpace(line) == "" {
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
	if err != nil && strings.TrimSpace(line) == "" {
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
	doc, err := readFallbackTOML(path)
	if err != nil {
		return FallbackConfig{}, err
	}
	return fallbackConfigFromDocument(doc), nil
}

func loadFallbackConfig(path string) (FallbackConfig, error) {
	return LoadFallbackConfig(path)
}

func readFallbackTOML(path string) (map[string]any, error) {
	doc := map[string]any{}
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return doc, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fallback: read config: %w", err)
	}
	if strings.TrimSpace(string(body)) == "" {
		return doc, nil
	}
	if err := toml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("fallback: parse config: %w", err)
	}
	return doc, nil
}

func AppendFallbackSelection(path string, selection cli.Selection) (bool, error) {
	entry := FallbackEntry{
		Provider: strings.TrimSpace(selection.Provider),
		Model:    strings.TrimSpace(selection.Model),
	}
	if entry.Provider == "" || entry.Model == "" {
		return false, cli.ErrSelectorNoMatch
	}
	doc, err := readFallbackTOML(path)
	if err != nil {
		return false, err
	}
	cfg := fallbackConfigFromDocument(doc)
	for _, existing := range cfg.Chain {
		if sameFallbackEntry(existing, entry) {
			return false, nil
		}
	}
	cfg.Chain = append(cfg.Chain, entry)
	return true, writeFallbackChainInDocument(path, doc, cfg.Chain)
}

func sameFallbackEntry(a, b FallbackEntry) bool {
	return a.Provider == b.Provider && a.Model == b.Model
}

func WriteFallbackChain(path string, chain []FallbackEntry) error {
	doc, err := readFallbackTOML(path)
	if err != nil {
		return err
	}
	return writeFallbackChainInDocument(path, doc, chain)
}

func writeFallbackChainInDocument(path string, doc map[string]any, chain []FallbackEntry) error {
	doc["fallback_providers"] = fallbackEntriesToConfigValue(chain)
	delete(doc, "fallback_model")
	return writeFallbackTOML(path, doc)
}

func fallbackEntriesToConfigValue(entries []FallbackEntry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		item := map[string]any{
			"provider": entry.Provider,
			"model":    entry.Model,
		}
		if entry.BaseURL != "" {
			item["base_url"] = entry.BaseURL
		}
		if entry.APIMode != "" {
			item["api_mode"] = entry.APIMode
		}
		out = append(out, item)
	}
	return out
}

func writeFallbackTOML(path string, doc map[string]any) error {
	body, err := toml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("fallback: encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("fallback: create config dir: %w", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("fallback: write config: %w", err)
	}
	return nil
}

func fallbackConfigFromDocument(doc map[string]any) FallbackConfig {
	var cfg FallbackConfig
	if hermesSection, ok := doc["hermes"].(map[string]any); ok {
		cfg.Primary = FallbackEntry{
			Provider: stringFromConfigValue(hermesSection["provider"]),
			Model:    stringFromConfigValue(hermesSection["model"]),
		}
	}
	cfg.Chain = fallbackEntriesFromConfigValue(doc["fallback_providers"])
	if len(cfg.Chain) == 0 {
		cfg.Chain = fallbackEntriesFromConfigValue(doc["fallback_model"])
	}
	return cfg
}

func fallbackEntriesFromConfigValue(value any) []FallbackEntry {
	switch v := value.(type) {
	case nil:
		return nil
	case []any:
		entries := make([]FallbackEntry, 0, len(v))
		for _, item := range v {
			if entry, ok := fallbackEntryFromConfigValue(item); ok {
				entries = append(entries, entry)
			}
		}
		return entries
	case []map[string]any:
		entries := make([]FallbackEntry, 0, len(v))
		for _, item := range v {
			if entry, ok := fallbackEntryFromConfigMap(item); ok {
				entries = append(entries, entry)
			}
		}
		return entries
	case map[string]any:
		if entry, ok := fallbackEntryFromConfigMap(v); ok {
			return []FallbackEntry{entry}
		}
		return nil
	default:
		if entry, ok := fallbackEntryFromConfigValue(v); ok {
			return []FallbackEntry{entry}
		}
		return nil
	}
}

func fallbackEntryFromConfigValue(value any) (FallbackEntry, bool) {
	if item, ok := value.(map[string]any); ok {
		return fallbackEntryFromConfigMap(item)
	}
	return FallbackEntry{}, false
}

func fallbackEntryFromConfigMap(item map[string]any) (FallbackEntry, bool) {
	entry := FallbackEntry{
		Provider: stringFromConfigValue(item["provider"]),
		Model:    stringFromConfigValue(item["model"]),
		BaseURL:  stringFromConfigValue(item["base_url"]),
		APIMode:  stringFromConfigValue(item["api_mode"]),
	}
	if entry.Provider == "" || entry.Model == "" {
		return FallbackEntry{}, false
	}
	return entry, true
}

func stringFromConfigValue(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
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
