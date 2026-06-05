package setup

import "fmt"

// Choice is a setup menu option with an ID value and operator-facing label.
type Choice struct {
	Value string
	Label string
}

// ToolOption is a CLI-configurable runtime toolset shown by setup tools.
type ToolOption struct {
	Key         string
	Label       string
	Description string
}

// ToolsProviderRow describes provider/API-key follow-up setup for a selected toolset.
type ToolsProviderRow struct {
	Toolset string
	Kind    string
	Label   string
}

// InvalidToolSelectionError reports a setup tools selection token that cannot be resolved.
type InvalidToolSelectionError struct {
	Token string
}

func (e InvalidToolSelectionError) Error() string {
	return fmt.Sprintf("setup_tools_invalid_selection: %s", e.Token)
}
