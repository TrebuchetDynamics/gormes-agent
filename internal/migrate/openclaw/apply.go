// Writer for `gormes migrate openclaw --yes` and `gormes migrate openclaw
// cleanup`. ApplyManifest applies a dry-run-validated OpenClaw Manifest
// into a destination Gormes config dir, dotenv, memory dir, skills dir,
// and a timestamped report directory under XDG_STATE_HOME. PerformCleanup
// renames leftover OpenClaw directories under HomeDir into
// `.pre-migration` archives without ever deleting data.
//
// The writer never shells out to Python, never imports Hermes optional
// skills, and never invokes pgrep/systemctl directly: process detection
// arrives through an injected ProcessDetector. Tests drive every IO
// boundary through ApplyRequest/CleanupRequest fields.
package openclaw

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Disposition strings used by ApplyOutcome maps. Stable so operator
// tooling can pretty-print without re-mapping.
const (
	DispositionMigrated        = "migrated"
	DispositionSkipped         = "skipped"
	DispositionConflictSkipped = "conflict_skipped"
	DispositionArchived        = "archived"
	DispositionSecretSkipped   = "secret_skipped"
	DispositionError           = "error"
)

// ApplyRequest is the input to ApplyManifest. Each external dependency
// is exposed so tests can drive ApplyManifest without touching real
// filesystem paths outside t.TempDir().
type ApplyRequest struct {
	Manifest          Manifest
	DestConfigDir     string
	DestEnvFile       string
	DestSkillsDir     string
	DestMemoryDir     string
	ReportRootDir     string
	Overwrite         bool
	Yes               bool
	SecretsEnabled    bool
	ExistingGormesEnv map[string]string
	SourceConfigBytes map[string][]byte
	SourceRoot        string
	Now               func() time.Time
}

// ApplyOutcome is the structured report ApplyManifest emits. It is
// JSON-serializable, never contains secret values, and every counter is
// derived from the per-surface Written maps + SecretSkipped tally.
type ApplyOutcome struct {
	ConfigWritten map[string]string `json:"config_written"`
	EnvWritten    map[string]string `json:"env_written"`
	MemoryWritten map[string]string `json:"memory_written"`
	SkillWritten  map[string]string `json:"skill_written"`
	ReportPath    string            `json:"report_path"`
	Backups       []string          `json:"backups,omitempty"`
	Errors        []string          `json:"errors,omitempty"`
	Counts        ApplyCounts       `json:"counts"`
}

// ApplyCounts is split out so cmd/gormes can decode it directly.
type ApplyCounts struct {
	Migrated        int `json:"migrated"`
	Skipped         int `json:"skipped"`
	ConflictSkipped int `json:"conflict_skipped"`
	Archived        int `json:"archived"`
	SecretSkipped   int `json:"secret_skipped"`
	Errors          int `json:"errors"`
}

// configTOMLSectionRoutes mirrors the (limited) OpenClaw->Gormes config
// section mapping. Anything not on this list lands in ConfigWritten as
// "skipped" with a manifest-derived reason already reported by the
// dry-run row.
var configTOMLSectionRoutes = map[string]string{
	"model":            "hermes",
	"providers":        "hermes",
	"custom_providers": "hermes",
	"channels":         "gateway",
	"mcp":              "mcp",
	"tts":              "tts",
	"approvals":        "approvals",
	"tools":            "tools",
	"session_reset":    "session_reset",
	"memory":           "memory",
}

// ApplyManifest persists the manifest's importable items to disk. It
// refuses to act without ApplyRequest.Yes, returns a structured
// ApplyOutcome on success, never includes secret bytes in the outcome,
// and always backs up pre-existing destination files before overwrite.
func ApplyManifest(req ApplyRequest) (ApplyOutcome, error) {
	out := ApplyOutcome{
		ConfigWritten: map[string]string{},
		EnvWritten:    map[string]string{},
		MemoryWritten: map[string]string{},
		SkillWritten:  map[string]string{},
	}
	if !req.Yes {
		return out, errors.New("openclaw migrate writer: use --yes to apply or --dry-run to inspect")
	}
	if req.DestConfigDir == "" {
		return out, errors.New("openclaw migrate writer: DestConfigDir is required")
	}
	if req.DestEnvFile == "" {
		return out, errors.New("openclaw migrate writer: DestEnvFile is required")
	}
	if req.DestSkillsDir == "" {
		return out, errors.New("openclaw migrate writer: DestSkillsDir is required")
	}
	if req.DestMemoryDir == "" {
		return out, errors.New("openclaw migrate writer: DestMemoryDir is required")
	}
	if req.ReportRootDir == "" {
		return out, errors.New("openclaw migrate writer: ReportRootDir is required")
	}
	now := req.Now
	if now == nil {
		now = time.Now
	}

	for _, dir := range []string{req.DestConfigDir, req.DestSkillsDir, req.DestMemoryDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return out, fmt.Errorf("openclaw migrate writer: mkdir %s: %w", dir, err)
		}
	}

	if err := applyConfigEntries(&out, req); err != nil {
		out.Errors = append(out.Errors, sanitizeError(err))
	}
	if err := applyEnvEntries(&out, req); err != nil {
		out.Errors = append(out.Errors, sanitizeError(err))
	}
	if err := applyFileEntries(&out, req); err != nil {
		out.Errors = append(out.Errors, sanitizeError(err))
	}

	finalizeCounts(&out)

	reportPath, err := writeReport(&out, req, now())
	if err != nil {
		out.Errors = append(out.Errors, sanitizeError(err))
		out.Counts.Errors++
		return out, nil
	}
	out.ReportPath = reportPath
	return out, nil
}

func applyConfigEntries(out *ApplyOutcome, req ApplyRequest) error {
	values, err := decodeSourceYAML(req.SourceConfigBytes)
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(req.DestConfigDir, "config.toml")
	doc, docErr := loadDestTOML(cfgPath)
	if docErr != nil {
		return docErr
	}

	hasImportable := false
	for _, e := range req.Manifest.Config {
		if e.Disposition == "importable" {
			hasImportable = true
			break
		}
	}
	if hasImportable {
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
			section, sectionOK := configTOMLSectionRoutes[entry.OpenClawKey]
			value, valueOK := values[entry.OpenClawKey]
			if !sectionOK || !valueOK {
				out.ConfigWritten[entry.OpenClawKey] = DispositionSkipped
				continue
			}
			if err := applyConfigValue(doc, section, entry.OpenClawKey, value); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("config %s: %w", entry.OpenClawKey, err)))
				out.ConfigWritten[entry.OpenClawKey] = DispositionError
				continue
			}
			out.ConfigWritten[entry.OpenClawKey] = DispositionMigrated
			mutated = true
		case "archived":
			out.ConfigWritten[entry.OpenClawKey] = DispositionArchived
		case "skipped":
			out.ConfigWritten[entry.OpenClawKey] = DispositionSkipped
		case "conflict":
			if !req.Overwrite {
				out.ConfigWritten[entry.OpenClawKey] = DispositionConflictSkipped
				continue
			}
			section, sectionOK := configTOMLSectionRoutes[entry.OpenClawKey]
			value, valueOK := values[entry.OpenClawKey]
			if !sectionOK || !valueOK {
				out.ConfigWritten[entry.OpenClawKey] = DispositionSkipped
				continue
			}
			if err := applyConfigValue(doc, section, entry.OpenClawKey, value); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("config %s: %w", entry.OpenClawKey, err)))
				out.ConfigWritten[entry.OpenClawKey] = DispositionError
				continue
			}
			out.ConfigWritten[entry.OpenClawKey] = DispositionMigrated
			mutated = true
		default:
			out.ConfigWritten[entry.OpenClawKey] = DispositionSkipped
		}
	}

	if mutated {
		if err := writeDestTOML(cfgPath, doc); err != nil {
			return fmt.Errorf("write %s: %w", cfgPath, err)
		}
	}
	return nil
}

func applyEnvEntries(out *ApplyOutcome, req ApplyRequest) error {
	envValues, err := decodeSourceEnv(req.SourceConfigBytes)
	if err != nil {
		return err
	}

	hasImportable := false
	for _, e := range req.Manifest.Env {
		if e.Disposition == "importable" || (e.Disposition == "conflict" && req.Overwrite) {
			hasImportable = true
			break
		}
	}
	if hasImportable && req.SecretsEnabled {
		if backup, ok, err := maybeBackup(req.DestEnvFile); err != nil {
			return err
		} else if ok {
			out.Backups = append(out.Backups, backup)
		}
	}

	for _, entry := range req.Manifest.Env {
		target := entry.GormesEnv
		key := target
		if key == "" {
			key = entry.OpenClawKey
		}
		switch entry.Disposition {
		case "importable":
			if !req.SecretsEnabled {
				out.EnvWritten[key] = DispositionSecretSkipped
				continue
			}
			value, present := envValues[entry.OpenClawKey]
			if !present || target == "" {
				out.EnvWritten[key] = DispositionSkipped
				continue
			}
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
			if !req.SecretsEnabled {
				out.EnvWritten[key] = DispositionSecretSkipped
				continue
			}
			value, present := envValues[entry.OpenClawKey]
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
			out.EnvWritten[key] = DispositionArchived
		case "skipped":
			out.EnvWritten[key] = DispositionSkipped
		default:
			out.EnvWritten[key] = DispositionSkipped
		}
	}
	return nil
}

func applyFileEntries(out *ApplyOutcome, req ApplyRequest) error {
	for _, entry := range req.Manifest.Files {
		if entry.Disposition != "importable" {
			switch entry.Kind {
			case "memory", "user_profile", "soul":
				out.MemoryWritten[entry.Kind] = entry.Disposition
			case "skills":
				out.SkillWritten[entry.Kind] = entry.Disposition
			}
			continue
		}
		switch entry.Kind {
		case "memory", "user_profile", "soul":
			destFile := filepath.Join(req.DestMemoryDir, filepath.Base(entry.SourcePath))
			if err := copyFileWithBackup(out, entry.SourcePath, destFile); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("memory %s: %w", entry.Kind, err)))
				out.MemoryWritten[entry.Kind] = DispositionError
				continue
			}
			out.MemoryWritten[entry.Kind] = DispositionMigrated
			// Surface a single aggregate "memory" key so callers can
			// check overall memory disposition without iterating.
			out.MemoryWritten["memory"] = DispositionMigrated
		case "skills":
			destDir := filepath.Join(req.DestSkillsDir, filepath.Base(entry.GormesPath))
			if err := copyTreeWithBackup(out, entry.SourcePath, destDir); err != nil {
				out.Errors = append(out.Errors, sanitizeError(fmt.Errorf("skills: %w", err)))
				out.SkillWritten["skills"] = DispositionError
				continue
			}
			out.SkillWritten["skills"] = DispositionMigrated
		default:
			out.SkillWritten[entry.Kind] = DispositionSkipped
		}
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

func applyConfigValue(doc map[string]any, section, openClawKey string, value any) error {
	if section == "hermes" && openClawKey == "model" {
		return setTOMLLeaf(doc, section, "model", coerceScalar(value))
	}
	if section == "hermes" && (openClawKey == "providers" || openClawKey == "custom_providers") {
		return setTOMLLeaf(doc, section, openClawKey, coerceTOMLValue(value))
	}
	coerced := coerceTOMLValue(value)
	mapped, ok := coerced.(map[string]any)
	if !ok {
		return setTOMLLeaf(doc, section, openClawKey, coerced)
	}
	doc[section] = mergeTables(asTable(doc[section]), mapped)
	return nil
}

func setTOMLLeaf(doc map[string]any, section, field string, value any) error {
	t := asTable(doc[section])
	t[field] = value
	doc[section] = t
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

func copyFileWithBackup(out *ApplyOutcome, srcPath, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(destPath), err)
	}
	if backup, ok, err := maybeBackup(destPath); err != nil {
		return err
	} else if ok {
		out.Backups = append(out.Backups, backup)
	}
	body, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", srcPath, err)
	}
	return os.WriteFile(destPath, body, 0o600)
}

func copyTreeWithBackup(out *ApplyOutcome, srcRoot, destRoot string) error {
	if err := os.MkdirAll(destRoot, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", destRoot, err)
	}
	return filepath.Walk(srcRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dest := filepath.Join(destRoot, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, 0o700)
		}
		if backup, ok, err := maybeBackup(dest); err != nil {
			return err
		} else if ok {
			out.Backups = append(out.Backups, backup)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return err
		}
		return os.WriteFile(dest, body, 0o600)
	})
}

// maybeBackup creates a timestamped .bak copy of path when path exists
// and the original bytes are non-empty. The writer never proceeds with
// overwrites if backup creation fails.
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

func writeReport(out *ApplyOutcome, req ApplyRequest, ts time.Time) (string, error) {
	stamp := ts.UTC().Format("20060102T150405")
	dir := filepath.Join(req.ReportRootDir, stamp)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("mkdir report dir: %w", err)
	}
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal report: %w", err)
	}
	// Defense-in-depth: never let secret bytes from the source manifest
	// reach the report file when secrets are disabled. The outcome maps
	// already exclude raw values, but a malformed Manifest could still
	// carry a value through — strip any byte sequence that matches a
	// known secret prefix.
	body = redactSecrets(body)
	path := filepath.Join(dir, "report.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

// redactSecrets is a defense-in-depth scrubber: looks for sk-... API
// key shapes and replaces them with [REDACTED]. The outcome maps never
// carry raw values themselves.
func redactSecrets(body []byte) []byte {
	s := string(body)
	idx := 0
	for {
		i := strings.Index(s[idx:], "sk-")
		if i < 0 {
			break
		}
		start := idx + i
		end := start + 3
		for end < len(s) && (isAlphanum(s[end]) || s[end] == '-' || s[end] == '_') {
			end++
		}
		s = s[:start] + "[REDACTED]" + s[end:]
		idx = start + len("[REDACTED]")
		if idx >= len(s) {
			break
		}
	}
	return []byte(s)
}

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
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

func finalizeCounts(out *ApplyOutcome) {
	out.Counts = ApplyCounts{}
	for _, m := range []map[string]string{out.ConfigWritten, out.EnvWritten, out.MemoryWritten, out.SkillWritten} {
		// Sort keys for deterministic counting in tests.
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			bumpCount(&out.Counts, m[k])
		}
	}
	// Memory counted aggregate "memory" key duplicates the per-kind tally;
	// avoid double-counting by removing the alias from the count then
	// re-add a single migrated tick.
	if out.MemoryWritten["memory"] == DispositionMigrated {
		// Already counted in the loop above. The aggregate alias means
		// we may have over-counted by one when both "memory" and a
		// per-kind key carry "migrated". Subtract the alias once.
		out.Counts.Migrated--
	}
	out.Counts.Archived += len(out.Backups)
	out.Counts.Errors += len(out.Errors)
}

func bumpCount(c *ApplyCounts, disposition string) {
	switch disposition {
	case DispositionMigrated:
		c.Migrated++
	case DispositionSkipped:
		c.Skipped++
	case DispositionConflictSkipped:
		c.ConflictSkipped++
	case DispositionArchived:
		c.Archived++
	case DispositionSecretSkipped:
		c.SecretSkipped++
	case DispositionError:
		c.Errors++
	}
}

// CleanupRequest is the input to PerformCleanup. The HomeDir field is
// the candidate root that contains `.openclaw`, `.clawdbot`, and
// `.moltbot`. Detector is optional — when nil, no warnings are emitted.
type CleanupRequest struct {
	HomeDir  string
	DryRun   bool
	Detector ProcessDetector
	Now      func() time.Time
}

// CleanupOutcome is the JSON-serializable result of PerformCleanup.
type CleanupOutcome struct {
	Renamed  []CleanupRename `json:"renamed"`
	Warnings []string        `json:"warnings,omitempty"`
	DryRun   bool            `json:"dry_run"`
}

// CleanupRename is one before/after pair. Splitting From/To into a
// named struct avoids inline struct literals that would otherwise leak
// into the JSON shape.
type CleanupRename struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ProcessDetector is the seam used by PerformCleanup to surface
// running-process warnings without invoking pgrep/systemctl directly.
// Tests inject a stub; production callers can wire a real detector
// later without touching the cleanup logic.
type ProcessDetector interface {
	Running(ctx context.Context, hint string) ([]string, error)
}

var openClawCleanupDirs = []string{".openclaw", ".clawdbot", ".moltbot"}

// PerformCleanup either previews (DryRun=true) or applies the rename
// of OpenClaw legacy directories under HomeDir to `.pre-migration`
// archives. PerformCleanup never deletes data; the worst-case is a
// rename that preserves all bytes intact under a new name.
func PerformCleanup(req CleanupRequest) (CleanupOutcome, error) {
	out := CleanupOutcome{DryRun: req.DryRun}
	if req.HomeDir == "" {
		return out, errors.New("openclaw cleanup: HomeDir is required")
	}

	if req.Detector != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		procs, err := req.Detector.Running(ctx, "openclaw")
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("process detector error: %s", sanitizeError(err)))
		}
		for _, p := range procs {
			out.Warnings = append(out.Warnings, fmt.Sprintf("running process detected: %s", p))
		}
	}

	for _, name := range openClawCleanupDirs {
		from := filepath.Join(req.HomeDir, name)
		to := filepath.Join(req.HomeDir, name+".pre-migration")
		info, err := os.Stat(from)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			out.Warnings = append(out.Warnings, fmt.Sprintf("stat %s: %s", from, sanitizeError(err)))
			continue
		}
		if !info.IsDir() {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s is not a directory; skipping", from))
			continue
		}
		out.Renamed = append(out.Renamed, CleanupRename{From: from, To: to})
		if req.DryRun {
			continue
		}
		// If the target archive already exists, append a timestamp so
		// nothing is deleted. PerformCleanup never deletes.
		if _, err := os.Stat(to); err == nil {
			now := time.Now
			if req.Now != nil {
				now = req.Now
			}
			to = to + "." + now().UTC().Format("20060102T150405")
			out.Renamed[len(out.Renamed)-1].To = to
		}
		if err := os.Rename(from, to); err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("rename %s -> %s: %s", from, to, sanitizeError(err)))
		}
	}
	return out, nil
}
