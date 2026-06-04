package configcheck

import (
	"fmt"
	"os"
	"strings"

	configwriter "github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
	configschema "github.com/TrebuchetDynamics/gormes-agent/internal/config/schema"
	"github.com/pelletier/go-toml/v2"
)

// Issue describes one structural problem detected by Check. Field names
// operator-targeted dotted paths; Severity is "error" for breaking problems
// and "warning" for recoverable ones.
type Issue struct {
	Severity string
	Field    string
	Message  string
}

// Report is the structured result returned by Check. It is read-only: no file
// under XDG_CONFIG_HOME is mutated by producing this value.
type Report struct {
	ConfigPath    string
	EnvPath       string
	ConfigVersion int
	LatestVersion int
	DotenvPresent bool
	Issues        []Issue
}

// Check inspects the on-disk Gormes config without mutating it. It returns the
// resolved config_version, the latest version this binary writes, the dotenv
// presence flag, and a list of structural issues. A non-nil error is returned
// when the loaded config_version is from a newer binary; the report is still
// populated with the raw version for operator evidence.
func Check() (Report, error) {
	report := Report{
		ConfigPath:    paths.ConfigPath(),
		EnvPath:       configwriter.EnvPath(),
		LatestVersion: configschema.CurrentConfigVersion,
	}

	if _, err := os.Stat(report.EnvPath); err == nil {
		report.DotenvPresent = true
	}

	body, err := os.ReadFile(report.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			report.Issues = append(report.Issues, Issue{
				Severity: "warning",
				Field:    "config.toml",
				Message:  "config file does not exist; run `gormes config edit` to create it",
			})
			return report, nil
		}
		return report, fmt.Errorf("config: read %s: %w", report.ConfigPath, err)
	}

	raw := map[string]any{}
	if err := toml.Unmarshal(body, &raw); err != nil {
		report.Issues = append(report.Issues, Issue{
			Severity: "error",
			Field:    "config.toml",
			Message:  fmt.Sprintf("invalid TOML: %v", err),
		})
		return report, err
	}

	report.ConfigVersion = configschema.ReadVersion(raw)
	if report.ConfigVersion > configschema.CurrentConfigVersion {
		return report, fmt.Errorf(
			"config: config_version=%d is from a newer binary (this binary knows up to v%d); upgrade gormes or hand-edit the file",
			report.ConfigVersion, configschema.CurrentConfigVersion)
	}
	if report.ConfigVersion == 0 {
		report.ConfigVersion = 1
	}

	report.Issues = append(report.Issues, hermesProviderIssues(raw)...)
	report.Issues = append(report.Issues, configStructureIssues(raw)...)
	return report, nil
}

var customProviderLikeRootFields = map[string]struct{}{
	"api_key":          {},
	"api_mode":         {},
	"base_url":         {},
	"rate_limit_delay": {},
}

// configStructureIssues mirrors Hermes' validate_config_structure startup
// diagnostics for common manual-edit mistakes. The check is intentionally
// read-only and reports operator-targeted field names without echoing values,
// so misplaced API keys or provider URLs do not leak into command output.
func configStructureIssues(raw map[string]any) []Issue {
	issues := make([]Issue, 0)
	for key, value := range raw {
		field := strings.TrimSpace(key)
		if field == "" || field == "config_version" || field == "_config_version" || strings.HasPrefix(field, "_") {
			continue
		}
		if _, ok := customProviderLikeRootFields[field]; ok {
			issues = append(issues, Issue{
				Severity: "warning",
				Field:    field,
				Message:  fmt.Sprintf("root-level key %q looks misplaced; move it under the appropriate provider/model section", field),
			})
			continue
		}
		if configschema.AllowsSection(field) {
			continue
		}
		kind := "field"
		if _, ok := value.(map[string]any); ok {
			kind = "section"
		}
		issues = append(issues, Issue{
			Severity: "warning",
			Field:    field,
			Message:  fmt.Sprintf("unknown top-level config %s %q; check spelling or migrate it to a supported section", kind, field),
		})
	}
	return issues
}

// hermesProviderIssues classifies the [hermes] table for missing or
// configured-but-empty endpoint and model fields. Missing means the key was
// absent from the file; configured-but-empty means the key was set to the empty
// string and is operator-evidence rather than an unspecified default.
func hermesProviderIssues(raw map[string]any) []Issue {
	hermesTable, _ := raw["hermes"].(map[string]any)
	required := []struct {
		key   string
		field string
	}{
		{"endpoint", "hermes.endpoint"},
		{"model", "hermes.model"},
	}
	var issues []Issue
	for _, r := range required {
		val, present := hermesTable[r.key]
		if !present {
			issues = append(issues, Issue{
				Severity: "error",
				Field:    r.field,
				Message:  fmt.Sprintf("%s is missing; set with `gormes config set %s <value>`", r.field, r.key),
			})
			continue
		}
		s, ok := val.(string)
		if ok && strings.TrimSpace(s) == "" {
			issues = append(issues, Issue{
				Severity: "error",
				Field:    r.field,
				Message:  fmt.Sprintf("%s is configured-but-empty; set with `gormes config set %s <value>`", r.field, r.key),
			})
		}
	}
	return issues
}

// MigrateResult is the structured outcome of MigrateConfigFile.
type MigrateResult struct {
	Path        string
	FromVersion int
	ToVersion   int
	NoOp        bool
	Wrote       bool
}

// MigrateFile applies native schema migrations atomically. It is a no-op when
// the file is already at CurrentConfigVersion. It rejects files from a newer
// binary without rewriting them. The write path uses a temp-file-then-rename
// so a partial write cannot corrupt config.toml.
func MigrateFile(path string) (MigrateResult, error) {
	result := MigrateResult{Path: path, ToVersion: configschema.CurrentConfigVersion}
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, fmt.Errorf("config: %s does not exist; run `gormes config edit` first", path)
		}
		return result, fmt.Errorf("config: read %s: %w", path, err)
	}

	raw := map[string]any{}
	if err := toml.Unmarshal(body, &raw); err != nil {
		return result, fmt.Errorf("config: parse %s: %w", path, err)
	}

	from := configschema.ReadVersion(raw)
	if from == 0 {
		from = 1
	}
	result.FromVersion = from

	if from > configschema.CurrentConfigVersion {
		return result, fmt.Errorf(
			"config: config_version=%d is from a newer binary (this binary knows up to v%d); upgrade gormes or hand-edit the file",
			from, configschema.CurrentConfigVersion)
	}

	if from == configschema.CurrentConfigVersion {
		// Well-formed v2 files are bit-for-bit unchanged.
		if _, present := raw["config_version"]; present && configschema.HasMainProfile(raw) {
			result.NoOp = true
			return result, nil
		}
	}

	delete(raw, "_config_version")
	raw["config_version"] = int64(configschema.CurrentConfigVersion)
	configschema.EnsureMainProfile(raw)
	if err := configwriter.WriteTOMLAtomic(path, raw); err != nil {
		return result, err
	}
	result.Wrote = true
	return result, nil
}
