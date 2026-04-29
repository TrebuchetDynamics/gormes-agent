// Package openclaw builds the deterministic dry-run manifest for
// `gormes migrate openclaw`. It is read-only: the writer row introduces
// destination writes separately.
package openclaw

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const RedactedValue = "[REDACTED]"

const (
	OriginExplicitSource      = "explicit_source"
	OriginUserHomeDotOpenClaw = "user_home_dot_openclaw"
	OriginUserHomeDotClawdbot = "user_home_dot_clawdbot"
	OriginUserHomeDotMoltbot  = "user_home_dot_moltbot"
)

type Options struct {
	Source            string
	ExistingGormesEnv map[string]string
}

type Manifest struct {
	Source     SourceReport     `json:"source"`
	Config     []ConfigEntry    `json:"config"`
	Env        []EnvEntry       `json:"env"`
	Files      []FileEntry      `json:"files"`
	SecretRefs []SecretRefEntry `json:"secret_refs"`
	Errors     []ErrorEntry     `json:"errors,omitempty"`
	Counts     Counts           `json:"counts"`
}

type SourceReport struct {
	Selected     string            `json:"selected"`
	SelectedPath string            `json:"selected_path"`
	Candidates   []SourceCandidate `json:"candidates"`
}

type SourceCandidate struct {
	Origin string `json:"origin"`
	Path   string `json:"path"`
	Found  bool   `json:"found"`
}

type ConfigEntry struct {
	OpenClawKey string `json:"openclaw_key"`
	Disposition string `json:"disposition"`
	GormesPath  string `json:"gormes_path,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type EnvEntry struct {
	OpenClawKey          string `json:"openclaw_key"`
	Disposition          string `json:"disposition"`
	GormesEnv            string `json:"gormes_env,omitempty"`
	RedactedValue        string `json:"redacted_value"`
	Reason               string `json:"reason,omitempty"`
	ConflictWithExisting bool   `json:"conflict_with_existing,omitempty"`
}

type FileEntry struct {
	Kind        string `json:"kind"`
	SourcePath  string `json:"source_path"`
	GormesPath  string `json:"gormes_path,omitempty"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason,omitempty"`
}

type SecretRefEntry struct {
	OpenClawKey string `json:"openclaw_key"`
	Source      string `json:"source"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason,omitempty"`
}

type ErrorEntry struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

type Counts struct {
	Migrated int `json:"migrated"`
	Skipped  int `json:"skipped"`
	Conflict int `json:"conflict"`
	Archived int `json:"archived"`
	Errors   int `json:"errors"`
}

var supportedConfigKeys = map[string]string{
	"model":            "hermes.model",
	"providers":        "hermes.providers",
	"custom_providers": "hermes.custom_providers",
	"channels":         "gateway.channels",
	"mcp":              "mcp",
	"tts":              "tts",
	"approvals":        "approvals",
	"tools":            "tools",
	"session_reset":    "session_reset",
	"memory":           "memory",
}

var archivedConfigKeys = map[string]string{
	"ui":      "OpenClaw UI/identity preferences are archived for manual review; Gormes owns its own TUI surface.",
	"logging": "OpenClaw logging/diagnostics configuration is archived for manual review.",
}

var supportedEnvKeys = map[string]string{
	"TELEGRAM_BOT_TOKEN":     "GORMES_TELEGRAM_BOT_TOKEN",
	"TELEGRAM_ALLOWED_USERS": "GORMES_TELEGRAM_ALLOWED_USERS",
	"DISCORD_BOT_TOKEN":      "GORMES_DISCORD_BOT_TOKEN",
	"DISCORD_ALLOWED_USERS":  "GORMES_DISCORD_ALLOWED_USERS",
	"SLACK_BOT_TOKEN":        "GORMES_SLACK_BOT_TOKEN",
	"SLACK_APP_TOKEN":        "GORMES_SLACK_APP_TOKEN",
	"SLACK_ALLOWED_USERS":    "GORMES_SLACK_ALLOWED_USERS",
	"WHATSAPP_ALLOWED_USERS": "GORMES_WHATSAPP_ALLOWED_USERS",
	"SIGNAL_ACCOUNT":         "GORMES_SIGNAL_ACCOUNT",
	"SIGNAL_ALLOWED_USERS":   "GORMES_SIGNAL_ALLOWED_USERS",
}

type workspaceFile struct {
	relPath    string
	kind       string
	gormesPath string
}

var workspaceFileMap = []workspaceFile{
	{relPath: "MEMORY.md", kind: "memory", gormesPath: "memory/MEMORY.md"},
	{relPath: "USER.md", kind: "user_profile", gormesPath: "memory/USER.md"},
	{relPath: "SOUL.md", kind: "soul", gormesPath: "memory/SOUL.md"},
	{relPath: "skills", kind: "skills", gormesPath: "skills/openclaw-imports"},
}

func BuildManifest(opts Options) (*Manifest, error) {
	candidates := resolveCandidates(opts)
	if opts.Source != "" {
		info, err := os.Stat(opts.Source)
		if err != nil {
			return nil, fmt.Errorf("openclaw migration source %q is not accessible: %w", opts.Source, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("openclaw migration source %q is not a directory", opts.Source)
		}
	}
	report := SourceReport{Candidates: candidates}
	if c, ok := firstFoundCandidate(candidates); ok {
		report.Selected = c.Origin
		report.SelectedPath = c.Path
	}
	m := &Manifest{Source: report}
	if report.SelectedPath == "" {
		return m, nil
	}
	loadConfigYAML(m, report.SelectedPath)
	loadDotenv(m, report.SelectedPath, opts.ExistingGormesEnv)
	loadWorkspaceFiles(m, report.SelectedPath)
	m.Counts = deriveCounts(m)
	return m, nil
}

func resolveCandidates(opts Options) []SourceCandidate {
	out := make([]SourceCandidate, 0, 4)
	if opts.Source != "" {
		out = append(out, candidate(OriginExplicitSource, opts.Source))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		out = append(out, candidate(OriginUserHomeDotOpenClaw, filepath.Join(home, ".openclaw")))
		out = append(out, candidate(OriginUserHomeDotClawdbot, filepath.Join(home, ".clawdbot")))
		out = append(out, candidate(OriginUserHomeDotMoltbot, filepath.Join(home, ".moltbot")))
	}
	return out
}

func candidate(origin, path string) SourceCandidate {
	c := SourceCandidate{Origin: origin, Path: path}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		c.Found = true
	}
	return c
}

func firstFoundCandidate(cs []SourceCandidate) (SourceCandidate, bool) {
	for _, c := range cs {
		if c.Found {
			return c, true
		}
	}
	return SourceCandidate{}, false
}

func loadConfigYAML(m *Manifest, root string) {
	data, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	if err != nil {
		if !os.IsNotExist(err) {
			m.Errors = append(m.Errors, ErrorEntry{Source: "config.yaml", Message: err.Error()})
		}
		return
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		m.Errors = append(m.Errors, ErrorEntry{Source: "config.yaml", Message: err.Error()})
		return
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		m.Config = append(m.Config, classifyConfigKey(k))
		collectSecretRefs(m, k, raw[k])
	}
}

func classifyConfigKey(key string) ConfigEntry {
	if reason, archived := archivedConfigKeys[key]; archived {
		return ConfigEntry{OpenClawKey: key, Disposition: "archived", Reason: reason}
	}
	if target, ok := supportedConfigKeys[key]; ok {
		return ConfigEntry{OpenClawKey: key, Disposition: "importable", GormesPath: target}
	}
	return ConfigEntry{OpenClawKey: key, Disposition: "skipped", Reason: "not in supported migration set for this slice; revisit when a writer row covers it"}
}

func collectSecretRefs(m *Manifest, topKey string, value any) {
	if topKey != "providers" && topKey != "channels" {
		return
	}
	walkSecretRefs(m, topKey, value)
}

func walkSecretRefs(m *Manifest, prefix string, value any) {
	switch v := value.(type) {
	case map[string]any:
		if isSecretRef(v) {
			source, _ := v["source"].(string)
			if source != "" && source != "env" {
				m.SecretRefs = append(m.SecretRefs, SecretRefEntry{OpenClawKey: prefix, Source: source, Disposition: "manual", Reason: "non-env SecretRef sources are not auto-resolved by the dry-run manifest; resolve manually"})
			}
			return
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			walkSecretRefs(m, prefix+"."+k, v[k])
		}
	case []any:
		for i, item := range v {
			walkSecretRefs(m, fmt.Sprintf("%s[%d]", prefix, i), item)
		}
	}
}

func isSecretRef(m map[string]any) bool {
	_, hasSource := m["source"]
	_, sourceIsString := m["source"].(string)
	return hasSource && sourceIsString
}

func loadDotenv(m *Manifest, root string, existing map[string]string) {
	f, err := os.Open(filepath.Join(root, ".env"))
	if err != nil {
		if !os.IsNotExist(err) {
			m.Errors = append(m.Errors, ErrorEntry{Source: ".env", Message: err.Error()})
		}
		return
	}
	defer f.Close()
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		m.Env = append(m.Env, classifyEnvKey(key, existing))
	}
	if err := scanner.Err(); err != nil {
		m.Errors = append(m.Errors, ErrorEntry{Source: ".env", Message: err.Error()})
	}
	sort.SliceStable(m.Env, func(i, j int) bool { return m.Env[i].OpenClawKey < m.Env[j].OpenClawKey })
}

func classifyEnvKey(key string, existing map[string]string) EnvEntry {
	target, mapped := supportedEnvKeys[key]
	if !mapped && (strings.HasSuffix(key, "_API_KEY") || strings.HasSuffix(key, "_TOKEN")) {
		target = key
		mapped = true
	}
	if !mapped {
		return EnvEntry{OpenClawKey: key, Disposition: "skipped", RedactedValue: RedactedValue, Reason: "no Gormes dotenv target for this OpenClaw env key in this slice"}
	}
	if v, ok := existing[target]; ok && v != "" {
		return EnvEntry{OpenClawKey: key, Disposition: "conflict", GormesEnv: target, RedactedValue: RedactedValue, Reason: fmt.Sprintf("existing %s is set; --overwrite required by writer slice", target), ConflictWithExisting: true}
	}
	return EnvEntry{OpenClawKey: key, Disposition: "importable", GormesEnv: target, RedactedValue: RedactedValue}
}

func loadWorkspaceFiles(m *Manifest, root string) {
	for _, wf := range workspaceFileMap {
		full := filepath.Join(root, wf.relPath)
		if _, err := os.Stat(full); err == nil {
			m.Files = append(m.Files, FileEntry{Kind: wf.kind, SourcePath: full, GormesPath: wf.gormesPath, Disposition: "importable"})
		}
	}
	sort.SliceStable(m.Files, func(i, j int) bool { return m.Files[i].Kind < m.Files[j].Kind })
}

func deriveCounts(m *Manifest) Counts {
	var c Counts
	tally := func(disp string) {
		switch disp {
		case "importable":
			c.Migrated++
		case "skipped":
			c.Skipped++
		case "archived", "manual":
			c.Archived++
		case "conflict":
			c.Conflict++
		case "error":
			c.Errors++
		}
	}
	for _, e := range m.Config {
		tally(e.Disposition)
	}
	for _, e := range m.Env {
		tally(e.Disposition)
	}
	for _, e := range m.Files {
		tally(e.Disposition)
	}
	for _, e := range m.SecretRefs {
		tally(e.Disposition)
	}
	c.Errors += len(m.Errors)
	return c
}
