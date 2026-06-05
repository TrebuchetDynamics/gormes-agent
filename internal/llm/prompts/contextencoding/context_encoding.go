package contextencoding

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/protocols/toon"
)

type PromptContextFormat string

const (
	PromptContextFormatJSON PromptContextFormat = "json"
	PromptContextFormatTOON PromptContextFormat = "toon"
)

type PromptContextEncodingReport struct {
	Format       string `json:"format"`
	RawBytes     int    `json:"raw_bytes"`
	EncodedBytes int    `json:"encoded_bytes"`
}

func EncodePromptContext(raw json.RawMessage, format PromptContextFormat) ([]byte, PromptContextEncodingReport, error) {
	report := PromptContextEncodingReport{Format: string(format), RawBytes: len(raw)}
	var (
		encoded []byte
		err     error
	)
	switch format {
	case "":
		report.Format = string(PromptContextFormatTOON)
		encoded, err = toon.EncodeJSON(raw)
	case PromptContextFormatJSON:
		report.Format = string(PromptContextFormatJSON)
		var compact bytes.Buffer
		err = json.Compact(&compact, raw)
		encoded = compact.Bytes()
	case PromptContextFormatTOON:
		encoded, err = toon.EncodeJSON(raw)
	default:
		err = fmt.Errorf("hermes: unsupported prompt context format %q", format)
	}
	if err != nil {
		return nil, report, err
	}
	report.EncodedBytes = len(encoded)
	return encoded, report, nil
}
