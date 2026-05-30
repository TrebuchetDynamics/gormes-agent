package llm

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts"
)

type PromptContextFormat = prompts.PromptContextFormat

const (
	PromptContextFormatJSON PromptContextFormat = prompts.PromptContextFormatJSON
	PromptContextFormatTOON PromptContextFormat = prompts.PromptContextFormatTOON
)

type PromptContextEncodingReport = prompts.PromptContextEncodingReport

func EncodePromptContext(raw json.RawMessage, format PromptContextFormat) ([]byte, PromptContextEncodingReport, error) {
	return prompts.EncodePromptContext(raw, format)
}
