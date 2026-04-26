package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

// SchemaRejectionReasonInputSchemaNotObject is the SchemaRejection.Reason
// emitted when an MCP tool's InputSchema is non-empty but is not a JSON
// object (e.g. a literal `true`, an array, or a scalar).
const SchemaRejectionReasonInputSchemaNotObject = "input_schema_must_be_object"

// defaultInputSchema is the JSON-Schema fragment substituted when an MCP
// tool advertises no InputSchema or an explicit JSON `null`. Hermes' upstream
// _normalize_mcp_input_schema applies the same fallback so providers that
// require an `object` schema do not reject the descriptor outright.
var defaultInputSchema = json.RawMessage(`{"type":"object","properties":{}}`)

// nameSanitizer collapses any character outside [A-Za-z0-9_] to '_', matching
// hermes' sanitize_mcp_name_component semantics so dynamic-discovery names
// stay compatible with provider validation rules.
var nameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_]`)

// NormalizedTool is the transport-free descriptor produced by NormalizeTools.
// SourceRaw preserves the verbatim MCP envelope so callers can correlate
// sanitized names back to the on-the-wire identifier.
type NormalizedTool struct {
	Name        string
	ServerName  string
	Description string
	InputSchema json.RawMessage
	SourceRaw   MCPRawTool
}

// SchemaRejection records a single MCPRawTool that was dropped during
// normalization and the reason it was dropped.
type SchemaRejection struct {
	ServerName string
	ToolName   string
	Reason     string
}

// NormalizeResult is the aggregate return value of NormalizeTools.
type NormalizeResult struct {
	Tools    []NormalizedTool
	Rejected []SchemaRejection
}

// NormalizeTools converts an MCP server's tools/list response into native
// NormalizedTool descriptors. Tools whose InputSchema is non-empty but not
// a JSON object are dropped into Rejected with reason
// SchemaRejectionReasonInputSchemaNotObject; the remaining tools land in
// Tools with their names sanitized for provider compatibility.
func NormalizeTools(serverName string, raw []MCPRawTool) NormalizeResult {
	out := NormalizeResult{}
	for _, t := range raw {
		schema, ok := normalizeInputSchema(t.InputSchema)
		if !ok {
			out.Rejected = append(out.Rejected, SchemaRejection{
				ServerName: serverName,
				ToolName:   t.Name,
				Reason:     SchemaRejectionReasonInputSchemaNotObject,
			})
			continue
		}
		out.Tools = append(out.Tools, NormalizedTool{
			Name:        sanitizeMCPNameComponent(t.Name),
			ServerName:  serverName,
			Description: t.Description,
			InputSchema: schema,
			SourceRaw:   t,
		})
	}
	return out
}

// sanitizeMCPNameComponent mirrors hermes' upstream helper: characters not in
// [A-Za-z0-9_] are replaced with '_'. The function never panics on empty
// input.
func sanitizeMCPNameComponent(value string) string {
	return nameSanitizer.ReplaceAllString(value, "_")
}

// normalizeInputSchema returns the validated JSON-Schema bytes plus a bool
// indicating whether the input was acceptable. Empty / null is replaced with
// a permissive empty object schema; a JSON object is passed through; anything
// else (true/false/number/string/array) is rejected.
func normalizeInputSchema(raw json.RawMessage) (json.RawMessage, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return defaultInputSchema, true
	}
	if !strings.HasPrefix(trimmed, "{") {
		return nil, false
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, false
	}
	return raw, true
}

// StructuredContent is a single content block from an MCP tools/call result
// (or the structuredContent envelope). The renderer only inspects the fields
// relevant to its Kind; unknown kinds fall back to Text so unsupported
// content types degrade gracefully instead of leaking the raw protocol
// envelope.
type StructuredContent struct {
	Kind     string
	Text     string
	MimeType string
	URI      string
}

// RenderToolCallResult flattens a sequence of structured content blocks into
// a single model-facing string. Text blocks contribute verbatim; image and
// resource blocks render as bracketed placeholders so the model has a stable
// signal that a non-textual artifact was returned. Unknown kinds emit their
// Text field if any (no panic, no JSON envelope leak).
func RenderToolCallResult(parts []StructuredContent) string {
	var b strings.Builder
	first := true
	write := func(s string) {
		if s == "" {
			return
		}
		if !first {
			b.WriteByte('\n')
		}
		b.WriteString(s)
		first = false
	}
	for _, p := range parts {
		switch p.Kind {
		case "text":
			write(p.Text)
		case "image":
			write(fmt.Sprintf("[image: %s]", p.MimeType))
		case "resource":
			write(fmt.Sprintf("[resource: %s]", p.URI))
		default:
			write(p.Text)
		}
	}
	return b.String()
}

// StderrSink is the minimal Writer/Closer surface the MCP transports use to
// drain a server subprocess's stderr without growing unbounded. Implementations
// must be safe for concurrent Write calls.
type StderrSink interface {
	Write(p []byte) (int, error)
	Close() error
}

// boundedStderrSink keeps only the last `tail` bytes of stderr in memory and,
// on Close, flushes them to `path` (prepending a `[truncated <N> bytes]`
// marker when bytes were dropped). When path is empty the sink discards all
// writes and never touches the filesystem.
type boundedStderrSink struct {
	mu     sync.Mutex
	path   string
	tail   int
	buffer []byte
	total  int64
	closed bool
}

// NewBoundedStderrSink returns a StderrSink that buffers at most tailBytes of
// stderr in memory. With path == "" the sink runs in discard mode: Write
// reports success without retaining bytes and Close never creates a file.
func NewBoundedStderrSink(path string, tailBytes int) StderrSink {
	return &boundedStderrSink{path: path, tail: tailBytes}
}

func (s *boundedStderrSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(p)
	s.total += int64(n)
	if s.path == "" || s.tail <= 0 {
		return n, nil
	}
	s.buffer = append(s.buffer, p...)
	if len(s.buffer) > s.tail {
		drop := len(s.buffer) - s.tail
		s.buffer = append(s.buffer[:0], s.buffer[drop:]...)
	}
	return n, nil
}

func (s *boundedStderrSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.path == "" {
		return nil
	}
	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()
	if int64(len(s.buffer)) < s.total {
		dropped := s.total - int64(len(s.buffer))
		if _, err := fmt.Fprintf(f, "[truncated %d bytes]\n", dropped); err != nil {
			return err
		}
	}
	if _, err := f.Write(s.buffer); err != nil {
		return err
	}
	return nil
}
