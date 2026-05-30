package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair"

type ToolCallParserStatus = repair.ToolCallParserStatus

const (
	ToolCallParserStatusMapped    ToolCallParserStatus = repair.ToolCallParserStatusMapped
	ToolCallParserStatusRowBacked ToolCallParserStatus = repair.ToolCallParserStatusRowBacked
)

type ToolCallParserEntry = repair.ToolCallParserEntry

func ToolCallParserManifest() []ToolCallParserEntry {
	return repair.ToolCallParserManifest()
}
