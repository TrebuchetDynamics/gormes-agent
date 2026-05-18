package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/pelletier/go-toml/v2"
)

// EnvPath returns the Gormes-native dotenv file path under GormesHome.
func EnvPath() string {
	return filepath.Join(GormesHome(), ".env")
}

// allowedTOMLSections is the closed set of top-level TOML tables this binary
// writes. Hermes parity work that introduces a new section must add it here.
// Keep in sync with the Config struct in config.go.
var allowedTOMLSections = map[string]struct{}{
	"hermes":      {},
	"profiles":    {},
	"credentials": {},
	"runtime":     {},
	"tts":         {},
	"image_gen":   {},
	"terminal":    {},
	"gateway":     {},
	"tui":         {},
	"input":       {},
	"voice":       {},
	"telegram":    {},
	"discord":     {},
	"slack":       {},
	"yuanbao":     {},
	"web":         {},
	"navivox":     {},
	"browser":     {},
	"security":    {},
	"secrets":     {},
	"agents":      {},
	"bindings":    {},
	"cron":        {},
	"skills":      {},
	"delegation":  {},
	"goncho":      {},
	"display":     {},
	"updates":     {},
}

// secretAliases maps user-typed secret aliases (e.g. `api_key`) to the
// canonical environment variable name written into the dotenv file.
var secretAliases = map[string]string{
	"api_key":            "GORMES_API_KEY",
	"hermes.api_key":     "GORMES_API_KEY",
	"telegram.bot_token": "GORMES_TELEGRAM_BOT_TOKEN",
	"discord.token":      "GORMES_DISCORD_TOKEN",
	"slack.bot_token":    "GORMES_SLACK_BOT_TOKEN",
	"slack.app_token":    "GORMES_SLACK_APP_TOKEN",
	"gateway.proxy_key":  "GATEWAY_PROXY_KEY",
	"navivox.token":      "GORMES_NAVIVOX_TOKEN",
}

// IsSecretKey reports whether the user-supplied dotted key targets the
// dotenv file rather than config.toml. Mirrors the Hermes set_config_value
// classifier: known aliases, *_API_KEY, and *_TOKEN suffixes are secrets.
// Section-prefixed keys ending in `_token` (e.g. telegram.bot_token) are
// also routed to .env so raw secrets never land in config.toml.
func IsSecretKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if _, ok := secretAliases[strings.ToLower(key)]; ok {
		return true
	}
	upper := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if strings.HasSuffix(upper, "_API_KEY") || upper == "API_KEY" {
		return true
	}
	if strings.HasSuffix(upper, "_TOKEN") {
		return true
	}
	return false
}

// SecretEnvName returns the canonical environment variable name a secret
// alias persists under. For non-alias secret keys the upper-cased key is
// returned unchanged.
func SecretEnvName(key string) string {
	if mapped, ok := secretAliases[strings.ToLower(strings.TrimSpace(key))]; ok {
		return mapped
	}
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(key), ".", "_"))
}

// WriteTOMLValue persists a single dotted key/value pair into the TOML file
// at path. The dotted key may be a top-level field (interpreted as
// hermes.<name> for the small Hermes-aliased set) or a `<section>.<field>`
// pair. Unknown top-level sections are rejected before any write so a typo
// cannot create an unbounded namespace.
func WriteTOMLValue(path, key, value string) error {
	section, fields, err := splitTOMLDotPath(key)
	if err != nil {
		return err
	}
	if _, ok := allowedTOMLSections[section]; !ok {
		return fmt.Errorf("config: unknown section %q in key %q (allowed: %s)", section, key, allowedSectionsList())
	}

	doc, err := readTOMLDoc(path)
	if err != nil {
		return err
	}
	table, ok := doc[section].(map[string]any)
	if !ok {
		table = map[string]any{}
	}
	coerced, err := coerceTOMLValue(section, fields, value)
	if err != nil {
		return fmt.Errorf("config: %s: %w", key, err)
	}

	if body, readErr := os.ReadFile(path); readErr == nil && len(body) > 0 {
		if updated, ok, preserveErr := updateTOMLValuePreservingLayout(body, section, fields, coerced); preserveErr == nil && ok {
			return writeTOMLRawAtomic(path, updated)
		}
	} else if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("config: read %s: %w", path, readErr)
	}

	setNestedTOMLValue(table, fields, coerced)
	doc[section] = table
	return writeTOMLDoc(path, doc)
}

// WriteEnvValue persists a KEY=VALUE pair into the dotenv file at path.
// An existing line for the same key is replaced in place; otherwise the
// pair is appended. Parent directories are created with 0o700 and the
// dotenv file is written 0o600 so secrets stay operator-readable only.
func WriteEnvValue(path, key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("config: empty env key")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}
	encoded := encodeEnvLine(key, value)

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("config: read %s: %w", path, err)
	}

	var out bytes.Buffer
	replaced := false
	if len(existing) > 0 {
		sc := bufio.NewScanner(bytes.NewReader(existing))
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if envLineMatchesKey(line, key) {
				out.WriteString(encoded)
				out.WriteByte('\n')
				replaced = true
				continue
			}
			out.WriteString(line)
			out.WriteByte('\n')
		}
		if err := sc.Err(); err != nil {
			return fmt.Errorf("config: scan %s: %w", path, err)
		}
	}
	if !replaced {
		if out.Len() > 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n")) {
			out.WriteByte('\n')
		}
		out.WriteString(encoded)
		out.WriteByte('\n')
	}
	if err := os.WriteFile(path, out.Bytes(), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("config: chmod %s: %w", path, err)
	}
	return nil
}

func splitTOMLDotPath(key string) (section string, fields []string, err error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil, fmt.Errorf("config: empty key")
	}
	parts := strings.Split(key, ".")
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			return "", nil, fmt.Errorf("config: malformed key %q", key)
		}
	}
	switch len(parts) {
	case 1:
		// Bare top-level keys map to the [hermes] section to mirror Hermes
		// `hermes config set endpoint ...` ergonomics.
		return "hermes", parts, nil
	default:
		return parts[0], parts[1:], nil
	}
}

func setNestedTOMLValue(root map[string]any, fields []string, value any) {
	table := root
	for _, field := range fields[:len(fields)-1] {
		next, ok := table[field].(map[string]any)
		if !ok {
			next = map[string]any{}
			table[field] = next
		}
		table = next
	}
	table[fields[len(fields)-1]] = value
}

func allowedSectionsList() string {
	names := make([]string, 0, len(allowedTOMLSections))
	for n := range allowedTOMLSections {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func readTOMLDoc(path string) (map[string]any, error) {
	doc := map[string]any{}
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doc, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := toml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return doc, nil
}

func writeTOMLDoc(path string, doc map[string]any) error {
	return writeTOMLAtomic(path, doc)
}

// writeTOMLAtomic marshals doc and writes it to path via a temp-file-then-
// rename so a partially-written config can never replace the previous one.
// The dotenv parent dir is created with 0o700 and the TOML file with 0o600.
func writeTOMLAtomic(path string, doc map[string]any) error {
	body, err := toml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("config: marshal toml: %w", err)
	}
	return writeTOMLRawAtomic(path, body)
}

func writeTOMLRawAtomic(path string, body []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config.toml.*")
	if err != nil {
		return fmt.Errorf("config: tempfile: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("config: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("config: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config: close temp: %w", err)
	}
	if _, err := toolspkg.AtomicReplace(tmpName, path, toolspkg.AtomicReplaceOptions{FirstWriteMode: 0o600}); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config: rename temp -> %s: %w", path, err)
	}
	return nil
}

func updateTOMLValuePreservingLayout(body []byte, section string, fields []string, value any) ([]byte, bool, error) {
	rendered, err := renderTOMLValue(value)
	if err != nil {
		return nil, false, err
	}
	targetPath := append([]string{section}, fields...)
	spans := splitTextLineSpans(body)
	currentTable := []string{}
	for _, span := range spans {
		line := string(body[span.start:span.textEnd])
		if table, ok := parseTOMLTableHeader(line); ok {
			currentTable = table
			continue
		}
		keyPath, valueStart, valueEnd, ok := parseTOMLAssignment(line)
		if !ok {
			continue
		}
		fullPath := append(append([]string{}, currentTable...), keyPath...)
		if !sameTOMLPath(fullPath, targetPath) {
			continue
		}
		updated := make([]byte, 0, len(body)+len(rendered))
		updated = append(updated, body[:span.start+valueStart]...)
		updated = append(updated, rendered...)
		updated = append(updated, body[span.start+valueEnd:]...)
		if err := validateTOMLBytes(updated); err != nil {
			return nil, false, err
		}
		return updated, true, nil
	}

	updated, err := insertTOMLValuePreservingLayout(body, section, fields, rendered, spans)
	if err != nil {
		return nil, false, err
	}
	return updated, true, nil
}

func renderTOMLValue(value any) ([]byte, error) {
	body, err := toml.Marshal(map[string]any{"value": value})
	if err != nil {
		return nil, fmt.Errorf("config: marshal toml value: %w", err)
	}
	line := strings.TrimSpace(string(body))
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return nil, fmt.Errorf("config: marshal toml value: missing assignment")
	}
	return []byte(strings.TrimSpace(line[eq+1:])), nil
}

func insertTOMLValuePreservingLayout(body []byte, section string, fields []string, rendered []byte, spans []textLineSpan) ([]byte, error) {
	tablePath := append([]string{section}, fields[:len(fields)-1]...)
	key := fields[len(fields)-1]
	insertLine := []byte(key + " = " + string(rendered) + "\n")

	insertAt := -1
	for i, span := range spans {
		line := string(body[span.start:span.textEnd])
		table, ok := parseTOMLTableHeader(line)
		if !ok {
			continue
		}
		if sameTOMLPath(table, tablePath) {
			insertAt = len(body)
			for _, next := range spans[i+1:] {
				nextLine := string(body[next.start:next.textEnd])
				if _, tableOK := parseTOMLTableHeader(nextLine); tableOK {
					insertAt = next.start
					break
				}
			}
			break
		}
	}

	var updated []byte
	if insertAt >= 0 {
		updated = make([]byte, 0, len(body)+len(insertLine))
		updated = append(updated, body[:insertAt]...)
		if insertAt > 0 && body[insertAt-1] != '\n' {
			updated = append(updated, '\n')
		}
		updated = append(updated, insertLine...)
		updated = append(updated, body[insertAt:]...)
	} else {
		header := []byte("[" + strings.Join(tablePath, ".") + "]\n")
		updated = append([]byte{}, body...)
		if len(updated) > 0 && updated[len(updated)-1] != '\n' {
			updated = append(updated, '\n')
		}
		if len(updated) > 0 && !bytes.HasSuffix(updated, []byte("\n\n")) {
			updated = append(updated, '\n')
		}
		updated = append(updated, header...)
		updated = append(updated, insertLine...)
	}
	if err := validateTOMLBytes(updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func validateTOMLBytes(body []byte) error {
	var doc map[string]any
	if err := toml.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("config: generated invalid TOML: %w", err)
	}
	return nil
}

type textLineSpan struct {
	start   int
	textEnd int
}

func splitTextLineSpans(body []byte) []textLineSpan {
	spans := []textLineSpan{}
	for start := 0; start < len(body); {
		end := start
		for end < len(body) && body[end] != '\n' {
			end++
		}
		textEnd := end
		if textEnd > start && body[textEnd-1] == '\r' {
			textEnd--
		}
		if end < len(body) {
			end++
		}
		spans = append(spans, textLineSpan{start: start, textEnd: textEnd})
		start = end
	}
	return spans
}

func parseTOMLTableHeader(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "[[") {
		return nil, false
	}
	close := strings.IndexByte(trimmed, ']')
	if close < 0 {
		return nil, false
	}
	raw := strings.TrimSpace(trimmed[1:close])
	if raw == "" {
		return nil, false
	}
	return splitTOMLPath(raw), true
}

func parseTOMLAssignment(line string) (keyPath []string, valueStart int, valueEnd int, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
		return nil, 0, 0, false
	}
	eq := findTOMLTokenOutsideQuotes(line, '=')
	if eq < 0 {
		return nil, 0, 0, false
	}
	key := strings.TrimSpace(line[:eq])
	if key == "" {
		return nil, 0, 0, false
	}
	valueStart = eq + 1
	for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
		valueStart++
	}
	comment := findTOMLCommentStart(line[valueStart:])
	if comment >= 0 {
		valueEnd = valueStart + comment
	} else {
		valueEnd = len(line)
	}
	for valueEnd > valueStart && (line[valueEnd-1] == ' ' || line[valueEnd-1] == '\t') {
		valueEnd--
	}
	return splitTOMLPath(key), valueStart, valueEnd, true
}

func splitTOMLPath(path string) []string {
	parts := strings.Split(path, ".")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func sameTOMLPath(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func findTOMLTokenOutsideQuotes(line string, token byte) int {
	inSingle := false
	inDouble := false
	escaped := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inDouble {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if inSingle {
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		switch c {
		case '#':
			return -1
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		default:
			if c == token {
				return i
			}
		}
	}
	return -1
}

func findTOMLCommentStart(s string) int {
	inSingle := false
	inDouble := false
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inDouble {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if inSingle {
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		switch c {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '#':
			return i
		}
	}
	return -1
}

// EnsureConfigFile creates a root v2 TOML file when it does not exist. It is
// a no-op when path already exists. Used by `gormes config edit` so operators
// never open a missing file.
func EnsureConfigFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("config: stat %s: %w", path, err)
	}
	return writeTOMLAtomic(path, DefaultConfigDocumentV2())
}

func coerceTOMLValue(section string, fields []string, value string) (any, error) {
	key := strings.ToLower(section + "." + strings.Join(fields, "."))
	switch key {
	case "telegram.allowed_user_ids":
		return coerceTOMLInt64List(value)
	case "telegram.home_channel.platform", "telegram.home_channel.chat_id", "telegram.home_channel.name", "telegram.home_channel.thread_id":
		return value, nil
	case "navivox.allow_origins", "navivox.allowed_tailnet_identities":
		return parseEnvCSV(value), nil
	case "agents.defaults.workspaces", "agents.defaults.skills", "agents.defaults.channels":
		// Gormes-owned per-profile multi-workspace list, the existing
		// agents.defaults.skills list, and the per-profile channels list:
		// comma-separated input -> TOML array.
		return parseEnvCSV(value), nil
	default:
		if (section == "discord" || section == "slack") && len(fields) > 0 && fields[len(fields)-1] == "allowed_channel_id" {
			return value, nil
		}
		return coerceTOMLScalar(value)
	}
}

func coerceTOMLInt64List(value string) ([]int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "[]" {
		return []int64{}, nil
	}
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	parts := strings.Split(trimmed, ",")
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected comma-separated integer IDs, got %q", value)
		}
		out = append(out, id)
	}
	return out, nil
}

// coerceTOMLScalar applies the same string→typed coercions as Hermes's
// set_config_value: explicit empty stays as "", on/off/yes/no/true/false
// become bool, integers become int64, floats become float64. Anything else
// is preserved as a string.
func coerceTOMLScalar(value string) (any, error) {
	switch strings.ToLower(value) {
	case "true", "yes", "on":
		return true, nil
	case "false", "no", "off":
		return false, nil
	}
	if value == "" {
		return "", nil
	}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f, nil
	}
	return value, nil
}

func encodeEnvLine(key, value string) string {
	needsQuoting := strings.ContainsAny(value, " \t#\"'\\\n\r")
	if !needsQuoting {
		return key + "=" + value
	}
	var b strings.Builder
	b.Grow(len(value) + 4)
	b.WriteByte('"')
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return key + "=" + b.String()
}

func envLineMatchesKey(line, key string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "export ") || strings.HasPrefix(trimmed, "export\t") {
		trimmed = strings.TrimLeft(trimmed[len("export"):], " \t")
	}
	eq := strings.IndexByte(trimmed, '=')
	if eq <= 0 {
		return false
	}
	return strings.TrimSpace(trimmed[:eq]) == key
}
