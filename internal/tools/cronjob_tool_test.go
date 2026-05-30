package tools_test

import (
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestCronjobToolFacadeConstructsRootTool(t *testing.T) {
	tool := tools.NewCronjobTool(tools.CronjobToolConfig{RunNowUnsupported: true})
	if tool.Name() != tools.CronjobToolName {
		t.Fatalf("Name() = %q, want %q", tool.Name(), tools.CronjobToolName)
	}
	if len(tool.Schema()) == 0 || !json.Valid(tool.Schema()) {
		t.Fatalf("Schema() is not valid JSON: %s", tool.Schema())
	}
}
