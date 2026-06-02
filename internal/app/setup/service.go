package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/pelletier/go-toml/v2"
)

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
