package httpstream

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SetSSEHeaders applies the adapter-standard Server-Sent Events response
// headers shared by API, bridge, and remote TUI streaming endpoints. Callers
// keep endpoint-specific headers such as Connection or X-Accel-Buffering local.
func SetSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
}

// WriteData writes a JSON-encoded unnamed SSE data event.
func WriteData(w http.ResponseWriter, body any) error {
	raw, _ := json.Marshal(body)
	_, err := fmt.Fprintf(w, "data: %s\n\n", raw)
	return err
}

// WriteEvent writes a JSON-encoded named SSE event.
func WriteEvent(w http.ResponseWriter, event string, body any) error {
	raw, _ := json.Marshal(body)
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw)
	return err
}

// WriteDone writes the OpenAI-compatible terminal SSE marker.
func WriteDone(w http.ResponseWriter) error {
	_, err := w.Write([]byte("data: [DONE]\n\n"))
	return err
}

// WriteComment writes an SSE comment line.
func WriteComment(w http.ResponseWriter, text string) error {
	_, err := w.Write([]byte(": " + text + "\n\n"))
	return err
}
