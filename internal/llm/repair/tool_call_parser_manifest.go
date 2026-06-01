package repair

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/parsers"

type ToolCallParserStatus = parsers.ToolCallParserStatus

const (
	ToolCallParserStatusMapped    ToolCallParserStatus = parsers.ToolCallParserStatusMapped
	ToolCallParserStatusRowBacked ToolCallParserStatus = parsers.ToolCallParserStatusRowBacked
)

type ToolCallParserEntry = parsers.ToolCallParserEntry

func ToolCallParserManifest() []ToolCallParserEntry {
	return parsers.ToolCallParserManifest()
}
