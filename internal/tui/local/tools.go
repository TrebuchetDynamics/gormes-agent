package local

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/toolsetup"
)

func NewToolsConfigureFunc() tui.ToolsConfigureFunc {
	return toolsetup.NewToolsConfigureFunc()
}

func ConfigureTools(req tui.ToolsConfigureRequest) (tui.ToolsConfigureResult, error) {
	return toolsetup.ConfigureTools(req)
}
