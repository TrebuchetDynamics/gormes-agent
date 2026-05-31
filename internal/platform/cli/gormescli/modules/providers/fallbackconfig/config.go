package fallbackconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// Entry is one primary or fallback provider/model pair from config.
type Entry struct {
	Provider string
	Model    string
	BaseURL  string
	APIMode  string
}

// Config is the fallback-provider portion of the Gormes config.
type Config struct {
	Primary Entry
	Chain   []Entry
}

// Load reads fallback provider config from path.
func Load(path string) (Config, error) {
	doc, err := readTOML(path)
	if err != nil {
		return Config{}, err
	}
	return fromDocument(doc), nil
}

func readTOML(path string) (map[string]any, error) {
	doc := map[string]any{}
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return doc, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fallback: read config: %w", err)
	}
	if !textvalue.IsNonBlank(string(body)) {
		return doc, nil
	}
	if err := toml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("fallback: parse config: %w", err)
	}
	return doc, nil
}

// Append adds entry to the fallback chain unless the same provider/model is already present.
func Append(path string, entry Entry) (bool, error) {
	doc, err := readTOML(path)
	if err != nil {
		return false, err
	}
	cfg := fromDocument(doc)
	for _, existing := range cfg.Chain {
		if sameEntry(existing, entry) {
			return false, nil
		}
	}
	cfg.Chain = append(cfg.Chain, entry)
	return true, writeChainInDocument(path, doc, cfg.Chain)
}

func sameEntry(a, b Entry) bool {
	return a.Provider == b.Provider && a.Model == b.Model
}

// WriteChain replaces the fallback provider chain at path.
func WriteChain(path string, chain []Entry) error {
	doc, err := readTOML(path)
	if err != nil {
		return err
	}
	return writeChainInDocument(path, doc, chain)
}

func writeChainInDocument(path string, doc map[string]any, chain []Entry) error {
	doc["fallback_providers"] = entriesToConfigValue(chain)
	delete(doc, "fallback_model")
	return writeTOML(path, doc)
}

func entriesToConfigValue(entries []Entry) []map[string]any {
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

func writeTOML(path string, doc map[string]any) error {
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

func fromDocument(doc map[string]any) Config {
	var cfg Config
	if hermesSection, ok := doc["hermes"].(map[string]any); ok {
		cfg.Primary = Entry{
			Provider: stringFromConfigValue(hermesSection["provider"]),
			Model:    stringFromConfigValue(hermesSection["model"]),
		}
	}
	cfg.Chain = entriesFromConfigValue(doc["fallback_providers"])
	if len(cfg.Chain) == 0 {
		cfg.Chain = entriesFromConfigValue(doc["fallback_model"])
	}
	return cfg
}

func entriesFromConfigValue(value any) []Entry {
	switch v := value.(type) {
	case nil:
		return nil
	case []any:
		entries := make([]Entry, 0, len(v))
		for _, item := range v {
			if entry, ok := entryFromConfigValue(item); ok {
				entries = append(entries, entry)
			}
		}
		return entries
	case []map[string]any:
		entries := make([]Entry, 0, len(v))
		for _, item := range v {
			if entry, ok := entryFromConfigMap(item); ok {
				entries = append(entries, entry)
			}
		}
		return entries
	case map[string]any:
		if entry, ok := entryFromConfigMap(v); ok {
			return []Entry{entry}
		}
		return nil
	default:
		if entry, ok := entryFromConfigValue(v); ok {
			return []Entry{entry}
		}
		return nil
	}
}

func entryFromConfigValue(value any) (Entry, bool) {
	if item, ok := value.(map[string]any); ok {
		return entryFromConfigMap(item)
	}
	return Entry{}, false
}

func entryFromConfigMap(item map[string]any) (Entry, bool) {
	entry := Entry{
		Provider: stringFromConfigValue(item["provider"]),
		Model:    stringFromConfigValue(item["model"]),
		BaseURL:  stringFromConfigValue(item["base_url"]),
		APIMode:  stringFromConfigValue(item["api_mode"]),
	}
	if entry.Provider == "" || entry.Model == "" {
		return Entry{}, false
	}
	return entry, true
}

func stringFromConfigValue(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
