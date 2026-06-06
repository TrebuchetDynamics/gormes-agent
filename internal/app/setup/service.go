package setup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/pelletier/go-toml/v2"
)

type ToolsOptions struct {
	Out            io.Writer
	ConfigPath     string
	NonInteractive bool

	PromptString       func(prompt, defaultValue string) (string, error)
	RunChecklist       func(title string, choices []ToolsChecklistChoice, selected []string) ([]string, error)
	PickShouldFallback func(error) bool

	ToolOptions      func() ([]ToolOption, error)
	LoadToolsConfig  func(path string) (map[string]any, toolsets.PlatformToolsetConfig, error)
	WriteToolsConfig func(path string, doc map[string]any) error
}

type ToolsChecklistChoice struct {
	ID    string
	Label string
}

func ResetDefaultConfig(path string) (string, error) {
	return ResetDefaultConfigWithClock(path, time.Now)
}

func ResetDefaultConfigWithClock(path string, now func() time.Time) (string, error) {
	if now == nil {
		now = time.Now
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("setup reset: mkdir %s: %w", dir, err)
	}
	var breadcrumb string
	if prior, readErr := os.ReadFile(path); readErr == nil {
		breadcrumb = path + ".before-reset." + now().UTC().Format("20060102T150405Z")
		if err := os.WriteFile(breadcrumb, prior, 0o600); err != nil {
			return "", fmt.Errorf("setup reset: write breadcrumb %s: %w", breadcrumb, err)
		}
	} else if !os.IsNotExist(readErr) {
		return "", fmt.Errorf("setup reset: read prior %s: %w", path, readErr)
	}
	body, err := toml.Marshal(config.DefaultConfigDocumentV2())
	if err != nil {
		return "", fmt.Errorf("setup reset: marshal defaults: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config.toml.reset-*")
	if err != nil {
		return "", fmt.Errorf("setup reset: tempfile: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("setup reset: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("setup reset: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("setup reset: close temp: %w", err)
	}
	if _, err := toolspkg.AtomicReplace(tmpName, path, toolspkg.AtomicReplaceOptions{FirstWriteMode: 0o600}); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("setup reset: replace %s: %w", path, err)
	}
	return breadcrumb, nil
}

func (opts ToolsOptions) withDefaults() ToolsOptions {
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	if opts.ToolOptions == nil {
		opts.ToolOptions = ToolOptions
	}
	if opts.LoadToolsConfig == nil {
		opts.LoadToolsConfig = LoadToolsConfig
	}
	if opts.WriteToolsConfig == nil {
		opts.WriteToolsConfig = WriteToolsConfig
	}
	return opts
}

// RunTools runs the setup tools section for CLI runtime toolsets.
func RunTools(opts ToolsOptions) error {
	opts = opts.withDefaults()
	out := opts.Out
	doc, toolCfg, err := opts.LoadToolsConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	status, err := toolCfg.PlatformStatus("cli")
	if err != nil {
		return err
	}
	toolOptions, err := opts.ToolOptions()
	if err != nil {
		return err
	}
	selected := stringSet(status.RuntimeToolsets)

	if !opts.NonInteractive && opts.RunChecklist != nil {
		chosen, err := opts.RunChecklist("Tools for 🖥️  CLI", toolChecklistChoices(toolOptions), status.RuntimeToolsets)
		if err == nil {
			if chosen == nil {
				fmt.Fprintln(out, "No tool setup changes selected.")
				return nil
			}
			return SaveToolsSelection(out, opts.ConfigPath, opts.WriteToolsConfig, doc, &toolCfg, chosen)
		}
		if opts.PickShouldFallback == nil || !opts.PickShouldFallback(err) {
			return err
		}
	}

	fmt.Fprintln(out, "Tools for CLI")
	fmt.Fprintln(out)
	for i, option := range toolOptions {
		marker := "[ ]"
		if selected[option.Key] {
			marker = "[x]"
		}
		fmt.Fprintf(out, "  %2d. %s %-28s %-16s %s\n", i+1, marker, option.Label, option.Key, option.Description)
	}
	if opts.NonInteractive {
		fmt.Fprintln(out, "\nSkipped (keeping current tool selection).")
		return nil
	}
	if opts.PromptString == nil {
		return fmt.Errorf("setup tools: prompt string is not configured")
	}
	fmt.Fprintln(out)
	selection, err := opts.PromptString("Toolsets (comma-separated numbers or keys, blank to keep current): ", "")
	if err != nil {
		return err
	}
	chosen, err := ParseToolSelection(selection, toolOptions, status.RuntimeToolsets)
	if err != nil {
		return err
	}
	return SaveToolsSelection(out, opts.ConfigPath, opts.WriteToolsConfig, doc, &toolCfg, chosen)
}

func toolChecklistChoices(options []ToolOption) []ToolsChecklistChoice {
	choices := make([]ToolsChecklistChoice, len(options))
	for i, option := range options {
		label := option.Label
		if option.Description != "" {
			label = fmt.Sprintf("%s  (%s)", label, option.Description)
		}
		choices[i] = ToolsChecklistChoice{ID: option.Key, Label: label}
	}
	return choices
}

func SaveToolsSelection(out io.Writer, configPath string, write func(string, map[string]any) error, doc map[string]any, toolCfg *toolsets.PlatformToolsetConfig, chosen []string) error {
	if out == nil {
		out = io.Discard
	}
	if write == nil {
		write = WriteToolsConfig
	}
	report, err := toolCfg.SavePlatformSelection("cli", chosen)
	if err != nil {
		return err
	}
	doc["platform_toolsets"] = toolCfg.PlatformToolsets
	if err := write(configPath, doc); err != nil {
		return err
	}
	fmt.Fprintf(out, "Saved CLI tool configuration: %s\n", strings.Join(report.PersistedToolsets, ", "))
	for _, issue := range report.Issues {
		if issue.Platform == "cli" || issue.Platform == "" {
			fmt.Fprintf(out, "setup_tools_issue: kind=%s toolset=%s detail=%s\n", issue.Kind, issue.Toolset, issue.Detail)
		}
	}
	RenderToolsProviderRows(out, chosen)
	return nil
}

func RenderToolsProviderRows(out io.Writer, selected []string) {
	rows := ToolsProviderRows(selected)
	if len(rows) == 0 {
		return
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Provider/API key setup")
	for _, row := range rows {
		fmt.Fprintf(out, "  setup_tools_provider_row_backed: toolset=%s provider=%s label=%s\n", row.Toolset, row.Kind, row.Label)
	}
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

// ToolOptions returns the setup tools choices sourced from the CLI toolset inventory.
func ToolOptions() ([]ToolOption, error) {
	report, err := toolsets.EffectiveToolsetPickerOptions(plugins.Inventory{})
	if err != nil {
		return nil, err
	}
	out := make([]ToolOption, 0, len(report.Options))
	for _, option := range report.Options {
		out = append(out, ToolOption{
			Key:         option.Key,
			Label:       option.Label,
			Description: option.Description,
		})
	}
	return out, nil
}

// LoadToolsConfig reads the TOML config used by setup tools. A missing config is an empty document.
func LoadToolsConfig(path string) (map[string]any, toolsets.PlatformToolsetConfig, error) {
	doc := map[string]any{}
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg, _ := toolsets.ParsePlatformToolsetConfig(doc)
			return doc, cfg, nil
		}
		return nil, toolsets.PlatformToolsetConfig{}, fmt.Errorf("setup tools: read %s: %w", path, err)
	}
	if err := toml.Unmarshal(body, &doc); err != nil {
		return nil, toolsets.PlatformToolsetConfig{}, fmt.Errorf("setup tools: parse %s: %w", path, err)
	}
	cfg, _ := toolsets.ParsePlatformToolsetConfig(doc)
	return doc, cfg, nil
}

// WriteToolsConfig atomically writes the setup tools TOML config while preserving symlink targets.
func WriteToolsConfig(path string, doc map[string]any) error {
	body, err := toml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("setup tools: marshal config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("setup tools: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config.toml.*")
	if err != nil {
		return fmt.Errorf("setup tools: tempfile: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("setup tools: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("setup tools: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("setup tools: close temp: %w", err)
	}
	if _, err := toolspkg.AtomicReplace(tmpName, path, toolspkg.AtomicReplaceOptions{FirstWriteMode: 0o600}); err != nil {
		return fmt.Errorf("setup tools: rename config: %w", err)
	}
	return nil
}

// ParseToolSelection resolves comma/space-separated numeric or key selections.
func ParseToolSelection(input string, options []ToolOption, current []string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return append([]string(nil), current...), nil
	}
	byKey := make(map[string]ToolOption, len(options))
	for _, option := range options {
		byKey[option.Key] = option
	}
	var selected []string
	for _, token := range strings.FieldsFunc(input, selectionSeparator) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if index, err := strconv.Atoi(token); err == nil {
			if index < 1 || index > len(options) {
				return nil, InvalidToolSelectionError{Token: token}
			}
			selected = append(selected, options[index-1].Key)
			continue
		}
		key := normalizeChoice(token)
		key = strings.ReplaceAll(key, "-", "_")
		if option, ok := byKey[key]; ok {
			selected = append(selected, option.Key)
			continue
		}
		selected = append(selected, token)
	}
	return selected, nil
}

func selectionSeparator(r rune) bool {
	return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == ';'
}

func normalizeChoice(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	if value == "apptainer" {
		return "singularity"
	}
	return value
}

var toolsProviderRows = map[string][]ToolsProviderRow{
	"web": {
		{Toolset: "web", Kind: "web", Label: "Web search and extraction"},
	},
	"browser": {
		{Toolset: "browser", Kind: "browser", Label: "Browser backend"},
	},
	"image_gen": {
		{Toolset: "image_gen", Kind: "image_gen", Label: "Image generation provider"},
	},
	"rl": {
		{Toolset: "rl", Kind: "rl", Label: "RL training provider"},
	},
	"tts": {
		{Toolset: "tts", Kind: "tts", Label: "Voice/TTS provider"},
	},
	"skills": {
		{Toolset: "skills", Kind: "github_skills_hub", Label: "GitHub Skills Hub"},
	},
	"memory": {
		{Toolset: "memory", Kind: "honcho", Label: "Honcho/Goncho memory provider"},
	},
	"homeassistant": {
		{Toolset: "homeassistant", Kind: "homeassistant", Label: "Home Assistant credentials"},
	},
}

// ToolsProviderRows returns provider/API-key follow-up rows in stable display order.
func ToolsProviderRows(selected []string) []ToolsProviderRow {
	selectedSet := map[string]bool{}
	for _, value := range selected {
		selectedSet[value] = true
	}
	var out []ToolsProviderRow
	for _, option := range []string{"web", "browser", "image_gen", "rl", "tts", "skills", "memory", "homeassistant"} {
		if !selectedSet[option] {
			continue
		}
		out = append(out, toolsProviderRows[option]...)
	}
	return out
}
