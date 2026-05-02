package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	maxAuditArgsBytes       = 8 * 1024
	maxAuditArgStringRunes  = 1024
	maxAuditURLStringRunes  = 256
	maxAuditErrorRunes      = 512
	maxAuditValueDepth      = 8
	maxAuditCollectionItems = 64
)

var auditSecretPattern = regexp.MustCompile(`(?i)(bearer\s+)[a-z0-9._~+/=-]+|sk-[a-z0-9._-]{8,}|secret[-_a-z0-9]*|token[-_a-z0-9]*`)

// Recorder captures one tool-execution audit record.
type Recorder interface {
	Record(rec Record) error
}

// Record is one append-only tool-execution audit event.
type Record struct {
	Timestamp       time.Time       `json:"timestamp"`
	Source          string          `json:"source"`
	SessionID       string          `json:"session_id"`
	AgentID         string          `json:"agent_id"`
	Tool            string          `json:"tool"`
	Args            json.RawMessage `json:"args"`
	DurationMs      int64           `json:"duration_ms"`
	Status          string          `json:"status"`
	ResultSizeBytes int             `json:"result_size_bytes"`
	Error           string          `json:"error"`
}

// JSONLWriter appends audit records to a JSONL file.
type JSONLWriter struct {
	path string
	mu   sync.Mutex
}

func NewJSONLWriter(path string) *JSONLWriter {
	return &JSONLWriter{path: strings.TrimSpace(path)}
}

func (w *JSONLWriter) Record(rec Record) error {
	if w == nil || w.path == "" {
		return nil
	}

	rec = normalize(rec)
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

func normalize(rec Record) Record {
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now().UTC()
	} else {
		rec.Timestamp = rec.Timestamp.UTC()
	}
	rec.Source = strings.TrimSpace(rec.Source)
	rec.SessionID = strings.TrimSpace(rec.SessionID)
	rec.AgentID = strings.TrimSpace(rec.AgentID)
	rec.Tool = strings.TrimSpace(rec.Tool)
	rec.Status = strings.TrimSpace(rec.Status)
	if rec.Status == "" {
		rec.Status = "unknown"
	}
	if rec.DurationMs < 0 {
		rec.DurationMs = 0
	}
	if len(rec.Args) == 0 {
		rec.Args = json.RawMessage(`null`)
	} else {
		rec.Args = sanitizeAuditArgs(rec.Args)
	}
	rec.Error = sanitizeAuditError(rec.Error)
	return rec
}

func sanitizeAuditArgs(raw json.RawMessage) json.RawMessage {
	var value any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return marshalAuditArgs(map[string]any{
			"_invalid_json": true,
			"preview":       truncateAuditString(redactAuditText(string(raw)), maxAuditArgStringRunes),
		})
	}

	out := marshalAuditArgs(sanitizeAuditValue(value, "", 0))
	if len(out) <= maxAuditArgsBytes {
		return out
	}
	return marshalAuditArgs(map[string]any{
		"_audit_args_truncated": true,
		"preview":               truncateAuditString(string(out), maxAuditArgStringRunes),
	})
}

func sanitizeAuditValue(value any, key string, depth int) any {
	if sensitiveAuditField(key) {
		return "[redacted]"
	}
	if depth >= maxAuditValueDepth {
		return "[truncated: max depth]"
	}
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, min(len(v), maxAuditCollectionItems+1))
		count := 0
		for childKey, childValue := range v {
			if count >= maxAuditCollectionItems {
				out["_truncated"] = fmt.Sprintf("%d entries omitted", len(v)-maxAuditCollectionItems)
				break
			}
			out[childKey] = sanitizeAuditValue(childValue, childKey, depth+1)
			count++
		}
		return out
	case []any:
		limit := min(len(v), maxAuditCollectionItems)
		out := make([]any, 0, limit+1)
		for i := 0; i < limit; i++ {
			out = append(out, sanitizeAuditValue(v[i], key, depth+1))
		}
		if len(v) > limit {
			out = append(out, fmt.Sprintf("[truncated: %d items omitted]", len(v)-limit))
		}
		return out
	case string:
		text := redactAuditText(v)
		if rightEdgeAuditField(key) {
			return truncateAuditStringRight(text, maxAuditURLStringRunes)
		}
		return truncateAuditString(text, maxAuditArgStringRunes)
	default:
		return v
	}
}

func marshalAuditArgs(value any) json.RawMessage {
	out, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"_audit_args_unavailable":true}`)
	}
	return append(json.RawMessage(nil), out...)
}

func sensitiveAuditField(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(normalized)
	if normalized == "" {
		return false
	}
	switch {
	case strings.Contains(normalized, "apikey"),
		strings.Contains(normalized, "authorization"),
		strings.Contains(normalized, "password"),
		strings.Contains(normalized, "passwd"),
		strings.Contains(normalized, "secret"),
		strings.Contains(normalized, "credential"),
		strings.Contains(normalized, "accesstoken"),
		strings.Contains(normalized, "refreshtoken"),
		strings.HasSuffix(normalized, "token"):
		return true
	default:
		return false
	}
}

func rightEdgeAuditField(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(normalized)
	switch {
	case normalized == "url",
		normalized == "uri",
		normalized == "href",
		strings.HasSuffix(normalized, "url"),
		strings.HasSuffix(normalized, "uri"),
		strings.HasSuffix(normalized, "path"):
		return true
	default:
		return false
	}
}

func sanitizeAuditError(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return truncateAuditString(redactAuditText(text), maxAuditErrorRunes)
}

func redactAuditText(text string) string {
	return auditSecretPattern.ReplaceAllStringFunc(text, func(match string) string {
		lower := strings.ToLower(match)
		if strings.HasPrefix(lower, "bearer ") {
			return match[:7] + "[redacted]"
		}
		return "[redacted]"
	})
}

func truncateAuditStringRight(text string, limit int) string {
	text = strings.ReplaceAll(text, "\x00", "")
	runes := []rune(text)
	if limit <= 0 || len(runes) <= limit {
		return text
	}
	return fmt.Sprintf("...[truncated %d chars]...", len(runes)-limit) + string(runes[len(runes)-limit:])
}

func truncateAuditString(text string, limit int) string {
	text = strings.ReplaceAll(text, "\x00", "")
	runes := []rune(text)
	if limit <= 0 || len(runes) <= limit {
		return text
	}
	omitted := len(runes) - limit
	head := limit / 2
	tail := limit - head
	return string(runes[:head]) + fmt.Sprintf("...[truncated %d chars]...", omitted) + string(runes[len(runes)-tail:])
}
