// Package hermes builds the deterministic dry-run manifest for
// `gormes migrate hermes`. It reads an explicit source directory, the
// $HERMES_HOME env var, or ~/.hermes, parses Hermes config.yaml and
// .env, and classifies every supported key as importable, skipped,
// archived, conflict, or error. The package is read-only: it never
// writes any file under the destination XDG directories. The writer
// row introduces apply.go separately.
package hermes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/migrate/shared"
	"gopkg.in/yaml.v3"
)

// RedactedValue replaces every secret value in the manifest. Raw secret
// bytes never enter the manifest output.
const RedactedValue = "[REDACTED]"

// Source candidate origin labels.
const (
	OriginExplicitSource    = "explicit_source"
	OriginHermesHomeEnv     = "hermes_home_env"
	OriginUserHomeDotHermes = "user_home_dot_hermes"
)

// Options drives BuildManifest. It is intentionally narrow: the writer
// slice will widen the surface but this dry-run row stays read-only.
type Options struct {
	// Source, when set, forces use of an explicit Hermes home and is
	// preferred over HERMES_HOME and ~/.hermes. Missing Source dirs
	// produce a hard error from BuildManifest.
	Source string
	// ExistingGormesEnv injects already-set Gormes env values so dotenv
	// keys that would overwrite them are reported as conflict instead
	// of importable. Tests pass synthetic values; production callers
	// pass the live process env subset.
	ExistingGormesEnv map[string]string
}

// Manifest is the deterministic dry-run plan. It is JSON-serializable
// and contains zero secret bytes.
type Manifest struct {
	Source SourceReport  `json:"source"`
	Config []ConfigEntry `json:"config"`
	Env    []EnvEntry    `json:"env"`
	Errors []ErrorEntry  `json:"errors,omitempty"`
	Counts Counts        `json:"counts"`
}

// SourceReport records every candidate path the resolver considered
// plus the one that was actually selected.
type SourceReport struct {
	Selected     string            `json:"selected"`
	SelectedPath string            `json:"selected_path"`
	Candidates   []SourceCandidate `json:"candidates"`
}

// SourceCandidate captures one candidate Hermes home directory.
type SourceCandidate struct {
	Origin string `json:"origin"`
	Path   string `json:"path"`
	Found  bool   `json:"found"`
}

// ConfigEntry describes one top-level key from config.yaml.
type ConfigEntry struct {
	HermesKey   string `json:"hermes_key"`
	Disposition string `json:"disposition"`
	GormesPath  string `json:"gormes_path,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// EnvEntry describes one key/value pair from .env. The raw value is
// never recorded.
type EnvEntry struct {
	HermesKey            string `json:"hermes_key"`
	Disposition          string `json:"disposition"`
	GormesEnv            string `json:"gormes_env,omitempty"`
	GormesPath           string `json:"gormes_path,omitempty"`
	RedactedValue        string `json:"redacted_value"`
	Reason               string `json:"reason,omitempty"`
	ConflictWithExisting bool   `json:"conflict_with_existing,omitempty"`
}

// ErrorEntry records a non-fatal parse or read error so dry-run callers
// can decide whether to proceed without panicking the manifest builder.
type ErrorEntry struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

// Counts summarizes manifest dispositions for the human-readable report
// emitted by `gormes migrate hermes --dry-run`.
type Counts struct {
	Migrated int `json:"migrated"`
	Skipped  int `json:"skipped"`
	Conflict int `json:"conflict"`
	Errors   int `json:"errors"`
}

// supportedConfigKeys maps Hermes top-level config.yaml sections that
// this slice can route into Gormes config.toml. Keys outside this map
// are recorded as skipped with an explicit "not in supported migration
// set" reason.
var supportedConfigKeys = map[string]string{
	"model":           "hermes.model",
	"providers":       "hermes.providers",
	"custom_provider": "hermes.custom_provider",
	"terminal":        "terminal",
	"display":         "display",
	"gateway":         "gateway",
	"memory":          "memory",
}

// archivedConfigKeys names keys that exist in Hermes but are explicitly
// archived rather than routed into Gormes config.toml. The writer slice
// expands this list; this slice locks the vocabulary for known schema-
// migration metadata.
var archivedConfigKeys = map[string]string{
	"_config_version": "Hermes-internal schema version is not migrated; Gormes maintains its own config_version.",
}

// supportedEnvKeys maps Hermes .env keys to Gormes dotenv targets. Keys
// matching common provider-API-key shapes (suffix _API_KEY/_TOKEN) are
// preserved verbatim under the same name when not in this explicit
// map.
type envTarget struct {
	Env  string
	Path string
}

var supportedEnvKeys = map[string]envTarget{
	"TELEGRAM_TOKEN":                  {Env: "GORMES_TELEGRAM_BOT_TOKEN"},
	"TELEGRAM_BOT_TOKEN":              {Env: "GORMES_TELEGRAM_BOT_TOKEN"},
	"TELEGRAM_CHAT_ID":                {Path: "telegram.home_channel.chat_id"},
	"TELEGRAM_HOME_CHANNEL":           {Path: "telegram.home_channel.chat_id"},
	"TELEGRAM_HOME_CHANNEL_NAME":      {Path: "telegram.home_channel.name"},
	"TELEGRAM_HOME_CHANNEL_THREAD_ID": {Path: "telegram.home_channel.thread_id"},
	"TELEGRAM_ALLOWED_USERS":          {Path: "telegram.allowed_user_ids"},
	"DISCORD_TOKEN":                   {Env: "GORMES_DISCORD_TOKEN"},
	"DISCORD_CHANNEL_ID":              {Env: "GORMES_DISCORD_CHANNEL_ID"},
	"SLACK_BOT_TOKEN":                 {Env: "GORMES_SLACK_BOT_TOKEN"},
	"SLACK_APP_TOKEN":                 {Env: "GORMES_SLACK_APP_TOKEN"},
	"SLACK_CHANNEL_ID":                {Env: "GORMES_SLACK_CHANNEL_ID"},
}

// BuildManifest builds the deterministic dry-run manifest from a
// Hermes home directory. It performs zero filesystem writes. An error
// is returned only when an explicit Options.Source is supplied but
// missing or unreadable; absent --source the manifest may be empty.
func BuildManifest(opts Options) (*Manifest, error) {
	candidates := resolveCandidates(opts)
	if opts.Source != "" {
		info, err := os.Stat(opts.Source)
		if err != nil {
			return nil, fmt.Errorf("hermes migration source %q is not accessible: %w", opts.Source, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("hermes migration source %q is not a directory", opts.Source)
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
	m.Counts = deriveCounts(m)
	return m, nil
}

func resolveCandidates(opts Options) []SourceCandidate {
	out := make([]SourceCandidate, 0, 3)
	if opts.Source != "" {
		out = append(out, candidate(OriginExplicitSource, opts.Source))
	}
	if env := strings.TrimSpace(os.Getenv("HERMES_HOME")); env != "" {
		out = append(out, candidate(OriginHermesHomeEnv, env))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		out = append(out, candidate(OriginUserHomeDotHermes, filepath.Join(home, ".hermes")))
	}
	return out
}

func candidate(origin, path string) SourceCandidate {
	return SourceCandidate{Origin: origin, Path: path, Found: shared.DirExists(path)}
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
	path := filepath.Join(root, "config.yaml")
	data, err := os.ReadFile(path)
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
	for _, k := range shared.SortedStringAnyKeys(raw) {
		entry := classifyConfigKey(k, raw[k])
		m.Config = append(m.Config, entry)
	}
}

func classifyConfigKey(key string, value any) ConfigEntry {
	if reason, archived := archivedConfigKeys[key]; archived {
		return ConfigEntry{HermesKey: key, Disposition: "archived", Reason: reason}
	}
	if key == "model" && hermesMapValue(value) != nil {
		return ConfigEntry{HermesKey: key, Disposition: "importable", GormesPath: "hermes.model,llm.provider"}
	}
	if target, ok := supportedConfigKeys[key]; ok {
		return ConfigEntry{HermesKey: key, Disposition: "importable", GormesPath: target}
	}
	return ConfigEntry{
		HermesKey:   key,
		Disposition: "skipped",
		Reason:      "not in supported migration set for this slice; revisit when a writer row covers it",
	}
}

func hermesMapValue(value any) map[string]any {
	switch v := value.(type) {
	case map[string]any:
		return v
	case map[any]any:
		out := map[string]any{}
		for k, vv := range v {
			out[fmt.Sprintf("%v", k)] = vv
		}
		return out
	default:
		return nil
	}
}

func loadDotenv(m *Manifest, root string, existing map[string]string) {
	path := filepath.Join(root, ".env")
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			m.Errors = append(m.Errors, ErrorEntry{Source: ".env", Message: err.Error()})
		}
		return
	}
	defer f.Close()

	keys, err := shared.ReadDotenvKeys(f)
	if err != nil {
		m.Errors = append(m.Errors, ErrorEntry{Source: ".env", Message: err.Error()})
		return
	}
	for _, key := range keys {
		m.Env = append(m.Env, classifyEnvKey(key, existing))
	}
}

func classifyEnvKey(key string, existing map[string]string) EnvEntry {
	target, mapped := supportedEnvKeys[key]
	if !mapped && (strings.HasSuffix(key, "_API_KEY") || strings.HasSuffix(key, "_TOKEN")) {
		target = envTarget{Env: key}
		mapped = true
	}
	if !mapped {
		return EnvEntry{
			HermesKey:     key,
			Disposition:   "skipped",
			RedactedValue: RedactedValue,
			Reason:        "no Gormes dotenv target for this Hermes env key in this slice",
		}
	}
	if target.Env != "" {
		if v, ok := existing[target.Env]; ok && v != "" {
			return EnvEntry{
				HermesKey:            key,
				Disposition:          "conflict",
				GormesEnv:            target.Env,
				RedactedValue:        RedactedValue,
				Reason:               fmt.Sprintf("existing %s is set; --overwrite required by writer slice", target.Env),
				ConflictWithExisting: true,
			}
		}
	}
	return EnvEntry{
		HermesKey:     key,
		Disposition:   "importable",
		GormesEnv:     target.Env,
		GormesPath:    target.Path,
		RedactedValue: RedactedValue,
	}
}

func deriveCounts(m *Manifest) Counts {
	var c Counts
	for _, e := range m.Config {
		switch e.Disposition {
		case "importable":
			c.Migrated++
		case "skipped", "archived":
			c.Skipped++
		case "conflict":
			c.Conflict++
		case "error":
			c.Errors++
		}
	}
	for _, e := range m.Env {
		switch e.Disposition {
		case "importable":
			c.Migrated++
		case "skipped", "archived":
			c.Skipped++
		case "conflict":
			c.Conflict++
		case "error":
			c.Errors++
		}
	}
	c.Errors += len(m.Errors)
	return c
}
