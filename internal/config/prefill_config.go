package config

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/prefill"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// LoadConfiguredPrefillMessages loads Hermes-compatible ephemeral few-shot
// prefill messages from the configured agent.prefill_messages_file path. The
// HERMES_PREFILL_MESSAGES_FILE/GORMES_PREFILL_MESSAGES_FILE environment
// overrides are applied by Load before this helper is called.
func LoadConfiguredPrefillMessages(cfg Config) ([]llm.Message, error) {
	return LoadPrefillMessages(cfg.Agent.PrefillMessagesFile)
}

// LoadPrefillMessages loads a JSON array of Hermes messages. Empty, missing,
// invalid, or non-array files degrade to no prefill messages, matching Hermes'
// nonfatal behavior. Relative paths resolve from GormesHome, the Go-native
// equivalent of Hermes resolving from ~/.hermes.
func LoadPrefillMessages(filePath string) ([]llm.Message, error) {
	return prefill.Load(filePath, GormesHome())
}
