// Writer for `gormes migrate hermes --yes`. ApplyManifest applies a
// dry-run-validated Manifest into a destination Gormes config dir + dotenv
// file, preserving Hermes config.yaml-derived values, importable env
// secrets, and backups of any pre-existing destination files. The writer
// is pure: it never touches ~/.gormes from tests; callers inject
// destination paths and existing-env snapshots.
package hermes

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Disposition strings used in WriteOutcome.ConfigWritten and EnvWritten.
// Kept stable so operator tooling and the cobra command can print them
// without reformatting.
const (
	DispositionMigrated        = "migrated"
	DispositionSkipped         = "skipped"
	DispositionConflictSkipped = "conflict_skipped"
	DispositionArchived        = "archived"
	DispositionRedacted        = "redacted"
	DispositionError           = "error"
)

// WriteRequest is the input to ApplyManifest. The struct intentionally
// owns every external dependency so tests can drive the writer without
// touching the real filesystem outside t.TempDir().
type WriteRequest struct {
	// Manifest is the dry-run plan produced by BuildManifest. Disposition
	// strings on each entry drive what the writer touches.
	Manifest Manifest

	// DestConfigDir is the destination directory that will contain
	// config.toml. Tests inject t.TempDir(); production callers pass
	// filepath.Dir(config.ConfigPath()).
	DestConfigDir string

	// DestEnvFile is the absolute path of the destination dotenv file.
	// Tests inject a path under t.TempDir(); production callers pass
	// config.EnvPath().
	DestEnvFile string

	// ExistingGormesEnv is the snapshot of GORMES_* (and other Gormes-
	// targeted) env vars currently set on the destination. The writer
	// uses this list (and the manifest's "conflict" entries) to decide
	// whether a key needs --overwrite to be applied.
	ExistingGormesEnv map[string]string

	// Overwrite, when true, allows the writer to apply manifest entries
	// classified as "conflict". Without it, those entries are reported
	// as conflict_skipped and never written.
	Overwrite bool

	// Yes is the explicit operator confirmation. The writer refuses to
	// proceed when Yes is false. The cobra command surfaces this as
	// `--yes`.
	Yes bool

	// SourceConfigBytes is the raw file content for source files keyed
	// by relative path. Required keys: "config.yaml" and ".env" when
	// the manifest contains entries from those files.
	SourceConfigBytes map[string][]byte
}

// WriteOutcome is the structured report ApplyManifest returns. It is
// JSON-serializable and never carries secret values — only key names and
// dispositions. Counts are derived from the Written/Backups maps.
type WriteOutcome struct {
	ConfigWritten map[string]string `json:"config_written"`
	EnvWritten    map[string]string `json:"env_written"`
	Backups       []string          `json:"backups"`
	Errors        []string          `json:"errors,omitempty"`
	Counts        struct {
		Migrated        int `json:"migrated"`
		Skipped         int `json:"skipped"`
		ConflictSkipped int `json:"conflict_skipped"`
		Archived        int `json:"archived"`
		Errors          int `json:"errors"`
	} `json:"counts"`
}

// configTOMLSectionRoutes maps Hermes config.yaml top-level keys to a
// destination TOML section name. The writer intentionally stays narrow:
// only the supported set produced by BuildManifest is honored. Bare
// scalar keys ("model") get hoisted into the [hermes] section to mirror
// `hermes config set model ...` ergonomics.
var configTOMLSectionRoutes = map[string]string{
	"model":           "hermes",
	"providers":       "hermes",
	"custom_provider": "hermes",
	"terminal":        "terminal",
	"display":         "display",
	"gateway":         "gateway",
	"memory":          "memory",
}

// ApplyManifest persists the manifest's importable entries to disk.
// It refuses to act without WriteRequest.Yes, returns a structured
// WriteOutcome on success, and never includes secret bytes in the
// outcome. Backups of pre-existing destination files are created
// before any overwrite.
func ApplyManifest(req WriteRequest) (WriteOutcome, error) {
	out := WriteOutcome{
		ConfigWritten: map[string]string{},
		EnvWritten:    map[string]string{},
	}
	if !req.Yes {
		return out, errors.New("hermes migrate writer: use --yes to apply or --dry-run to inspect")
	}
	if req.DestConfigDir == "" {
		return out, errors.New("hermes migrate writer: DestConfigDir is required")
	}
	if req.DestEnvFile == "" {
		return out, errors.New("hermes migrate writer: DestEnvFile is required")
	}

	if err := os.MkdirAll(req.DestConfigDir, 0o700); err != nil {
		return out, fmt.Errorf("hermes migrate writer: mkdir dest config dir: %w", err)
	}

	if err := applyConfigEntries(&out, req); err != nil {
		out.Errors = append(out.Errors, sanitizeError(err))
	}
	if err := applyEnvEntries(&out, req); err != nil {
		out.Errors = append(out.Errors, sanitizeError(err))
	}

	finalizeCounts(&out)
	return out, nil
}

func applyConfigEntries(out *WriteOutcome, req WriteRequest) error {
	values, err := decodeSourceYAML(req.SourceConfigBytes)
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(req.DestConfigDir, "config.toml")
	doc, docErr := loadDestTOML(cfgPath)
	if docErr != nil {
		return docErr
	}

	// If config.toml already exists, back it up before any mutation
	// touches disk. The writer always backs up before overwrite, even
	// when the only entries are env entries — the cobra wiring decides
	// when to call ApplyManifest.
	hasConfigEntries := false
	for _, e := range req.Manifest.Config {
		if e.Disposition == "importable" {
			hasConfigEntries = true
			break
		}
	}
	if hasConfigEntries {
		if backup, ok, err := maybeBackup(cfgPath); err != nil {
			return err
		} else if ok {
			out.Backups = append(out.Backups, backup)
		}
	}

	mutated := false
	for _, entry := range req.Manifest.Config {
		switch entry.Disposition {
		case "importable":
			section, ok := configTOMLSectionRoutes[entry.HermesKey]
			if !ok {
				out.ConfigWritten[entry.HermesKey] = DispositionSkipped
				continue
			}
			value, present := values[entry.HermesKey]
			if !present {
				out.ConfigWritten[entry.HermesKey] = DispositionSkipped
				continue
			}
			if err := applyConfigValue(doc, section, entry.HermesKey, value); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("config %s: %w", entry.HermesKey, err)))
				out.ConfigWritten[entry.HermesKey] = DispositionError
				continue
			}
			out.ConfigWritten[entry.HermesKey] = DispositionMigrated
			mutated = true
		case "archived":
			out.ConfigWritten[entry.HermesKey] = DispositionArchived
		case "skipped":
			out.ConfigWritten[entry.HermesKey] = DispositionSkipped
		case "conflict":
			if req.Overwrite {
				section, ok := configTOMLSectionRoutes[entry.HermesKey]
				if ok {
					if value, present := values[entry.HermesKey]; present {
						if err := applyConfigValue(doc, section, entry.HermesKey, value); err == nil {
							out.ConfigWritten[entry.HermesKey] = DispositionMigrated
							mutated = true
							continue
						}
					}
				}
				out.ConfigWritten[entry.HermesKey] = DispositionSkipped
			} else {
				out.ConfigWritten[entry.HermesKey] = DispositionConflictSkipped
			}
		default:
			out.ConfigWritten[entry.HermesKey] = DispositionSkipped
		}
	}

	if mutated {
		if err := writeDestTOML(cfgPath, doc); err != nil {
			return fmt.Errorf("write %s: %w", cfgPath, err)
		}
	}
	return nil
}

func applyEnvEntries(out *WriteOutcome, req WriteRequest) error {
	envValues, err := decodeSourceEnv(req.SourceConfigBytes)
	if err != nil {
		return err
	}
	if err := applyEnvConfigEntries(out, req, envValues); err != nil {
		return err
	}

	// When env work is needed and the destination dotenv exists, back it
	// up exactly once before the first writeEnvLine mutates the file.
	hasImportable := false
	for _, e := range req.Manifest.Env {
		if e.GormesPath != "" {
			continue
		}
		if e.Disposition == "importable" || (e.Disposition == "conflict" && req.Overwrite) {
			hasImportable = true
			break
		}
	}
	if hasImportable {
		if backup, ok, err := maybeBackup(req.DestEnvFile); err != nil {
			return err
		} else if ok {
			out.Backups = append(out.Backups, backup)
		}
	}

	for _, entry := range req.Manifest.Env {
		if entry.GormesPath != "" {
			continue
		}
		target := entry.GormesEnv
		switch entry.Disposition {
		case "importable":
			value, present := envValues[entry.HermesKey]
			if !present || target == "" {
				if target == "" {
					out.EnvWritten[entry.HermesKey] = DispositionSkipped
				} else {
					out.EnvWritten[target] = DispositionSkipped
				}
				continue
			}
			// Defensive double-check: if the destination env already has
			// the key set in ExistingGormesEnv, route to conflict logic.
			if existing, ok := req.ExistingGormesEnv[target]; ok && existing != "" && !req.Overwrite {
				out.EnvWritten[target] = DispositionConflictSkipped
				continue
			}
			if err := writeEnvLine(req.DestEnvFile, target, value); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("env %s: %w", target, err)))
				out.EnvWritten[target] = DispositionError
				continue
			}
			out.EnvWritten[target] = DispositionMigrated
		case "conflict":
			if !req.Overwrite {
				out.EnvWritten[target] = DispositionConflictSkipped
				continue
			}
			value, present := envValues[entry.HermesKey]
			if !present || target == "" {
				out.EnvWritten[target] = DispositionSkipped
				continue
			}
			if err := writeEnvLine(req.DestEnvFile, target, value); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("env %s: %w", target, err)))
				out.EnvWritten[target] = DispositionError
				continue
			}
			out.EnvWritten[target] = DispositionMigrated
		case "archived":
			key := target
			if key == "" {
				key = entry.HermesKey
			}
			out.EnvWritten[key] = DispositionArchived
		case "skipped":
			key := target
			if key == "" {
				key = entry.HermesKey
			}
			out.EnvWritten[key] = DispositionSkipped
		default:
			key := target
			if key == "" {
				key = entry.HermesKey
			}
			out.EnvWritten[key] = DispositionSkipped
		}
	}
	return nil
}

func applyEnvConfigEntries(out *WriteOutcome, req WriteRequest, envValues map[string]string) error {
	hasConfigTarget := false
	for _, e := range req.Manifest.Env {
		if e.GormesPath != "" && (e.Disposition == "importable" || (e.Disposition == "conflict" && req.Overwrite)) {
			hasConfigTarget = true
			break
		}
	}
	if !hasConfigTarget {
		for _, e := range req.Manifest.Env {
			if e.GormesPath == "" {
				continue
			}
			out.ConfigWritten[e.GormesPath] = DispositionSkipped
		}
		return nil
	}

	cfgPath := filepath.Join(req.DestConfigDir, "config.toml")
	doc, err := loadDestTOML(cfgPath)
	if err != nil {
		return err
	}
	if backup, ok, err := maybeBackup(cfgPath); err != nil {
		return err
	} else if ok {
		out.Backups = append(out.Backups, backup)
	}

	mutated := false
	for _, entry := range req.Manifest.Env {
		if entry.GormesPath == "" {
			continue
		}
		target := entry.GormesPath
		switch entry.Disposition {
		case "importable":
			value, present := envValues[entry.HermesKey]
			if !present {
				out.ConfigWritten[target] = DispositionSkipped
				continue
			}
			coerced, err := coerceEnvConfigValue(target, value)
			if err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("config %s: %w", target, err)))
				out.ConfigWritten[target] = DispositionError
				continue
			}
			if err := setTOMLDottedPath(doc, target, coerced); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("config %s: %w", target, err)))
				out.ConfigWritten[target] = DispositionError
				continue
			}
			out.ConfigWritten[target] = DispositionMigrated
			mutated = true
		case "conflict":
			if !req.Overwrite {
				out.ConfigWritten[target] = DispositionConflictSkipped
				continue
			}
			value, present := envValues[entry.HermesKey]
			if !present {
				out.ConfigWritten[target] = DispositionSkipped
				continue
			}
			coerced, err := coerceEnvConfigValue(target, value)
			if err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("config %s: %w", target, err)))
				out.ConfigWritten[target] = DispositionError
				continue
			}
			if err := setTOMLDottedPath(doc, target, coerced); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("config %s: %w", target, err)))
				out.ConfigWritten[target] = DispositionError
				continue
			}
			out.ConfigWritten[target] = DispositionMigrated
			mutated = true
		case "archived":
			out.ConfigWritten[target] = DispositionArchived
		default:
			out.ConfigWritten[target] = DispositionSkipped
		}
	}
	if !mutated {
		return nil
	}
	if err := writeDestTOML(cfgPath, doc); err != nil {
		return fmt.Errorf("write %s: %w", cfgPath, err)
	}
	return nil
}

func decodeSourceYAML(src map[string][]byte) (map[string]any, error) {
	body, ok := src["config.yaml"]
	if !ok || len(body) == 0 {
		return map[string]any{}, nil
	}
	var raw map[string]any
	if err := yaml.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode source config.yaml: %w", err)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	return raw, nil
}

func decodeSourceEnv(src map[string][]byte) (map[string]string, error) {
	body, ok := src[".env"]
	if !ok || len(body) == 0 {
		return map[string]string{}, nil
	}
	out := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(body))
	for sc.Scan() {
		line := strings.TrimLeft(sc.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "export\t") {
			line = strings.TrimLeft(line[len("export"):], " \t")
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}
		val := strings.TrimRight(line[eq+1:], " \t\r")
		val = strings.Trim(val, `"`)
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func loadDestTOML(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	doc := map[string]any{}
	if err := toml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return doc, nil
}

func writeDestTOML(path string, doc map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	body, err := toml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal toml: %w", err)
	}
	return os.WriteFile(path, body, 0o600)
}

func applyConfigValue(doc map[string]any, section, hermesKey string, value any) error {
	if section == "hermes" && hermesKey == "model" {
		if modelCfg, ok := coerceTOMLValue(value).(map[string]any); ok {
			if model := strings.TrimSpace(fmt.Sprint(modelCfg["default"])); model != "" && model != "<nil>" {
				if err := setTOMLLeaf(doc, section, "model", model); err != nil {
					return err
				}
			}
			if provider := strings.TrimSpace(fmt.Sprint(modelCfg["provider"])); provider != "" && provider != "<nil>" {
				if err := setTOMLLeaf(doc, section, "provider", provider); err != nil {
					return err
				}
			}
			return nil
		}
		return setTOMLLeaf(doc, section, "model", coerceScalar(value))
	}
	if section == "hermes" && (hermesKey == "providers" || hermesKey == "custom_provider") {
		return setTOMLLeaf(doc, section, hermesKey, coerceTOMLValue(value))
	}
	if section == "display" && hermesKey == "display" {
		return applyDisplayConfigValue(doc, value)
	}
	// All other supported keys are top-level sections (terminal, gateway,
	// memory). We replace the section with the source map
	// after coercion. This is the simplest safe behavior; merge semantics
	// can be added in a future row.
	coerced := coerceTOMLValue(value)
	mapped, ok := coerced.(map[string]any)
	if !ok {
		// Scalar at section-level — nest under the manifest key.
		return setTOMLLeaf(doc, section, hermesKey, coerced)
	}
	doc[section] = mergeTables(asTable(doc[section]), mapped)
	return nil
}

func applyDisplayConfigValue(doc map[string]any, value any) error {
	source, ok := coerceTOMLValue(value).(map[string]any)
	if !ok {
		return setTOMLLeaf(doc, "display", "display", coerceScalar(value))
	}
	display := asTable(doc["display"])
	if mode, ok := normalizeMigratedToolProgressMode(source["tool_progress"]); ok {
		display["tool_progress"] = mode
	}
	if command, ok := boolValue(source["tool_progress_command"]); ok {
		display["tool_progress_command"] = command
	}
	platforms := asTable(display["platforms"])
	if sourcePlatforms := asTable(source["platforms"]); len(sourcePlatforms) > 0 {
		for platform, rawPlatform := range sourcePlatforms {
			key := textvalue.LowerTrim(platform)
			if key == "" {
				continue
			}
			platformCfg := asTable(rawPlatform)
			if mode, ok := normalizeMigratedToolProgressMode(platformCfg["tool_progress"]); ok {
				entry := asTable(platforms[key])
				entry["tool_progress"] = mode
				platforms[key] = entry
			}
		}
	}
	if overrides := asTable(source["tool_progress_overrides"]); len(overrides) > 0 {
		for platform, rawMode := range overrides {
			key := textvalue.LowerTrim(platform)
			if key == "" {
				continue
			}
			entry := asTable(platforms[key])
			if strings.TrimSpace(fmt.Sprint(entry["tool_progress"])) != "" && fmt.Sprint(entry["tool_progress"]) != "<nil>" {
				continue
			}
			if mode, ok := normalizeMigratedToolProgressMode(rawMode); ok {
				entry["tool_progress"] = mode
				platforms[key] = entry
			}
		}
	}
	if len(platforms) > 0 {
		display["platforms"] = platforms
	}
	doc["display"] = display
	return nil
}

func setTOMLLeaf(doc map[string]any, section, field string, value any) error {
	t := asTable(doc[section])
	t[field] = value
	doc[section] = t
	return nil
}

func setTOMLDottedPath(doc map[string]any, path string, value any) error {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return fmt.Errorf("expected dotted config path, got %q", path)
	}
	table := asTable(doc[parts[0]])
	doc[parts[0]] = table
	for _, part := range parts[1 : len(parts)-1] {
		next := asTable(table[part])
		table[part] = next
		table = next
	}
	table[parts[len(parts)-1]] = value
	return nil
}

func asTable(v any) map[string]any {
	if t, ok := v.(map[string]any); ok {
		return t
	}
	return map[string]any{}
}

func mergeTables(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = coerceTOMLValue(v)
	}
	return out
}

func coerceEnvConfigValue(path string, value string) (any, error) {
	switch path {
	case "telegram.allowed_user_ids":
		return parseInt64CSV(value)
	case "telegram.home_channel.chat_id", "telegram.home_channel.name", "telegram.home_channel.thread_id":
		return strings.TrimSpace(value), nil
	default:
		return coerceScalar(value), nil
	}
}

func parseInt64CSV(value string) ([]int64, error) {
	parts := strings.Split(value, ",")
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected comma-separated integer IDs, got %q", value)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func boolValue(value any) (bool, bool) {
	v, ok := value.(bool)
	return v, ok
}

func normalizeMigratedToolProgressMode(raw any) (string, bool) {
	switch v := raw.(type) {
	case nil:
		return "", false
	case bool:
		if v {
			return "all", true
		}
		return "off", true
	default:
		mode := textvalue.LowerTrim(fmt.Sprint(v))
		if mode == "" || mode == "<nil>" {
			return "", false
		}
		switch mode {
		case "off", "new", "all", "verbose":
			return mode, true
		default:
			return "all", true
		}
	}
}

// coerceTOMLValue converts yaml-decoded values into TOML-serializable
// shapes (map[string]any rather than map[any]any, []any preserved). It
// is a small adapter, not a full schema migration.
func coerceTOMLValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			out[k] = coerceTOMLValue(vv)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			out[fmt.Sprintf("%v", k)] = coerceTOMLValue(vv)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, vv := range t {
			out[i] = coerceTOMLValue(vv)
		}
		return out
	default:
		return coerceScalar(v)
	}
}

func coerceScalar(v any) any {
	switch t := v.(type) {
	case string:
		// Lower-cased on/off/yes/no/true/false flow through as bools so
		// `gateway.notify_interval=180` stays int but `terminal.backend=local`
		// stays a string.
		switch strings.ToLower(t) {
		case "true", "yes", "on":
			return true
		case "false", "no", "off":
			return false
		}
		if i, err := strconv.ParseInt(t, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f
		}
		return t
	default:
		return v
	}
}

func writeEnvLine(path, key, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	encoded := encodeEnvValue(key, value)

	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var out bytes.Buffer
	replaced := false
	if len(existing) > 0 {
		sc := bufio.NewScanner(bytes.NewReader(existing))
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if envLineKey(line) == key {
				out.WriteString(encoded)
				out.WriteByte('\n')
				replaced = true
				continue
			}
			out.WriteString(line)
			out.WriteByte('\n')
		}
		if err := sc.Err(); err != nil {
			return fmt.Errorf("scan %s: %w", path, err)
		}
	}
	if !replaced {
		if out.Len() > 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n")) {
			out.WriteByte('\n')
		}
		out.WriteString(encoded)
		out.WriteByte('\n')
	}
	return os.WriteFile(path, out.Bytes(), 0o600)
}

func encodeEnvValue(key, value string) string {
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

func envLineKey(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "export ") || strings.HasPrefix(trimmed, "export\t") {
		trimmed = strings.TrimLeft(trimmed[len("export"):], " \t")
	}
	eq := strings.IndexByte(trimmed, '=')
	if eq <= 0 {
		return ""
	}
	return strings.TrimSpace(trimmed[:eq])
}

// maybeBackup creates a timestamped .bak copy of path when path exists
// and the original bytes are non-empty. It returns the backup path
// (when created), a flag indicating whether a backup was created, and
// any error from reading or writing the backup. The writer never
// proceeds with overwrites if backup creation fails.
func maybeBackup(path string) (string, bool, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read %s for backup: %w", path, err)
	}
	if len(body) == 0 {
		return "", false, nil
	}
	stamp := time.Now().UTC().Format("20060102T150405")
	backup := path + ".bak." + stamp
	// Ensure uniqueness when ApplyManifest runs twice in the same second.
	for i := 1; ; i++ {
		if _, err := os.Stat(backup); errors.Is(err, os.ErrNotExist) {
			break
		}
		backup = fmt.Sprintf("%s.bak.%s.%d", path, stamp, i)
		if i > 32 {
			return "", false, fmt.Errorf("backup name collision for %s", path)
		}
	}
	if err := writeAll(backup, body); err != nil {
		return "", false, fmt.Errorf("write backup %s: %w", backup, err)
	}
	return backup, true, nil
}

func writeAll(path string, body []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, bytes.NewReader(body)); err != nil {
		return err
	}
	return f.Sync()
}

// sanitizeError strips known secret substrings before they can land in
// the WriteOutcome.Errors slice. The hermes manifest already redacts
// secret values; this is a defense-in-depth check covering filesystem
// errors that might echo a path containing a secret-looking token.
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Defensive: never echo back content that looks like an API key.
	if idx := strings.Index(msg, "sk-"); idx >= 0 {
		end := idx + 3
		for end < len(msg) && (isAlphanum(msg[end]) || msg[end] == '-' || msg[end] == '_') {
			end++
		}
		msg = msg[:idx] + "[REDACTED]" + msg[end:]
	}
	return msg
}

func isAlphanum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func finalizeCounts(out *WriteOutcome) {
	keys := make([]string, 0, len(out.ConfigWritten))
	for k := range out.ConfigWritten {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	envKeys := make([]string, 0, len(out.EnvWritten))
	for k := range out.EnvWritten {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)

	for _, k := range keys {
		bumpCount(&out.Counts, out.ConfigWritten[k])
	}
	for _, k := range envKeys {
		bumpCount(&out.Counts, out.EnvWritten[k])
	}
	out.Counts.Archived += len(out.Backups)
	out.Counts.Errors += len(out.Errors)
}

func bumpCount(c *struct {
	Migrated        int `json:"migrated"`
	Skipped         int `json:"skipped"`
	ConflictSkipped int `json:"conflict_skipped"`
	Archived        int `json:"archived"`
	Errors          int `json:"errors"`
}, disposition string) {
	switch disposition {
	case DispositionMigrated:
		c.Migrated++
	case DispositionSkipped, DispositionRedacted:
		c.Skipped++
	case DispositionConflictSkipped:
		c.ConflictSkipped++
	case DispositionArchived:
		c.Archived++
	case DispositionError:
		c.Errors++
	}
}
