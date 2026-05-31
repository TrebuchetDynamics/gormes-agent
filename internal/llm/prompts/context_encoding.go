package prompts

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/contextencoding"
)

type PromptContextFormat = contextencoding.PromptContextFormat

const (
	PromptContextFormatJSON PromptContextFormat = contextencoding.PromptContextFormatJSON
	PromptContextFormatTOON PromptContextFormat = contextencoding.PromptContextFormatTOON
)

type PromptContextEncodingReport = contextencoding.PromptContextEncodingReport

func EncodePromptContext(raw json.RawMessage, format PromptContextFormat) ([]byte, PromptContextEncodingReport, error) {
	return contextencoding.EncodePromptContext(raw, format)
}
