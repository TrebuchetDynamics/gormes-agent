package prefill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestConfiguredMessagesLoadsConfiguredPrefillFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prefill.json")
	if err := os.WriteFile(path, []byte(`[{"role":"system","content":"be brief"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	messages := ConfiguredMessages(config.Config{Agent: config.AgentRuntimeCfg{PrefillMessagesFile: path}})
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0].Role != "system" || messages[0].Content != "be brief" {
		t.Fatalf("message = %+v", messages[0])
	}
}

func TestConfiguredMessagesReturnsNilWhenPrefillUnavailable(t *testing.T) {
	messages := ConfiguredMessages(config.Config{Agent: config.AgentRuntimeCfg{PrefillMessagesFile: filepath.Join(t.TempDir(), "missing", "prefill.json")}})
	if messages != nil {
		t.Fatalf("messages = %+v, want nil", messages)
	}
}
