package contract

import "strings"

const (
	TranscriberModelUnavailable = "model_unavailable"
	TranscriberWAVUnsupported   = "wav_unsupported"
	TranscriberWASIInference    = "wasi_inference_failed"
	TranscriberClosed           = "transcriber_closed"
)

type TranscriberError struct {
	Code string
	Path string
	Err  error
}

func (e *TranscriberError) Error() string {
	var parts []string
	parts = append(parts, e.Code)
	if e.Path != "" {
		parts = append(parts, "path="+e.Path)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *TranscriberError) Unwrap() error {
	return e.Err
}
