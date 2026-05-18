package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// CheckIssue describes one structural problem detected by Check. Field
// names operator-targeted dotted paths; Severity is "error" for breaking
// problems and "warning" for recoverable ones.
type CheckIssue struct {
	Severity string
	Field    string
	Message  string
}

// CheckReport is the structured result returned by Check. It is read-only:
// no file under XDG_CONFIG_HOME is mutated by producing this value.
type CheckReport struct {
	ConfigPath    string
	EnvPath       string
	ConfigVersion int
	LatestVersion int
	DotenvPresent bool
	Issues        []CheckIssue
}

// Check inspects the on-disk Gormes config without mutating it. It returns
// the resolved config_version, the latest version this binary writes, the
// dotenv presence flag, and a list of structural issues. A non-nil error is
// returned when the loaded config_version is from a newer binary; the
// report is still populated with the raw version for operator evidence.
func Check() (CheckReport, error) {
	report := CheckReport{
		ConfigPath:    ConfigPath(),
		EnvPath:       EnvPath(),
		LatestVersion: CurrentConfigVersion,
	}

	if _, err := os.Stat(report.EnvPath); err == nil {
		report.DotenvPresent = true
	}

	body, err := os.ReadFile(report.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			report.Issues = append(report.Issues, CheckIssue{
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
		report.Issues = append(report.Issues, CheckIssue{
			Severity: "error",
			Field:    "config.toml",
			Message:  fmt.Sprintf("invalid TOML: %v", err),
		})
		return report, err
	}

	report.ConfigVersion = readConfigVersion(raw)
	if report.ConfigVersion > CurrentConfigVersion {
		return report, fmt.Errorf(
			"config: config_version=%d is from a newer binary (this binary knows up to v%d); upgrade gormes or hand-edit the file",
			report.ConfigVersion, CurrentConfigVersion)
	}
	if report.ConfigVersion == 0 {
		report.ConfigVersion = 1
	}

	report.Issues = append(report.Issues, hermesProviderIssues(raw)...)
	return report, nil
}

func readConfigVersion(raw map[string]any) int {
	v, ok := raw["config_version"]
	if !ok {
		v, ok = raw["_config_version"]
	}
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}

// hermesProviderIssues classifies the [hermes] table for missing or
// configured-but-empty endpoint and model fields. Missing means the key was
// absent from the file; configured-but-empty means the key was set to the
// empty string and is operator-evidence rather than an unspecified default.
func hermesProviderIssues(raw map[string]any) []CheckIssue {
	hermesTable, _ := raw["hermes"].(map[string]any)
	required := []struct {
		key   string
		field string
	}{
		{"endpoint", "hermes.endpoint"},
		{"model", "hermes.model"},
	}
	var issues []CheckIssue
	for _, r := range required {
		val, present := hermesTable[r.key]
		if !present {
			issues = append(issues, CheckIssue{
				Severity: "error",
				Field:    r.field,
				Message:  fmt.Sprintf("%s is missing; set with `gormes config set %s <value>`", r.field, r.key),
			})
			continue
		}
		s, ok := val.(string)
		if ok && strings.TrimSpace(s) == "" {
			issues = append(issues, CheckIssue{
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

// MigrateConfigFile applies native schema migrations atomically. It is a
// no-op when the file is already at CurrentConfigVersion. It rejects files
// from a newer binary without rewriting them. The write path uses a
// temp-file-then-rename so a partial write cannot corrupt config.toml.
func MigrateConfigFile(path string) (MigrateResult, error) {
	result := MigrateResult{Path: path, ToVersion: CurrentConfigVersion}
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

	from := readConfigVersion(raw)
	if from == 0 {
		from = 1
	}
	result.FromVersion = from

	if from > CurrentConfigVersion {
		return result, fmt.Errorf(
			"config: config_version=%d is from a newer binary (this binary knows up to v%d); upgrade gormes or hand-edit the file",
			from, CurrentConfigVersion)
	}

	if from == CurrentConfigVersion {
		// Well-formed v2 files are bit-for-bit unchanged.
		if _, present := raw["config_version"]; present && hasMainProfile(raw) {
			result.NoOp = true
			return result, nil
		}
	}

	delete(raw, "_config_version")
	raw["config_version"] = int64(CurrentConfigVersion)
	ensureMainProfile(raw)
	if err := writeTOMLAtomic(path, raw); err != nil {
		return result, err
	}
	result.Wrote = true
	return result, nil
}

func hasMainProfile(raw map[string]any) bool {
	profiles, ok := raw["profiles"].(map[string]any)
	if !ok {
		return false
	}
	main, ok := profiles[DefaultProfileID].(map[string]any)
	if !ok {
		return false
	}
	_, hasEnabled := main["enabled"]
	_, hasName := main["name"]
	return hasEnabled && hasName
}

func ensureMainProfile(raw map[string]any) {
	profiles, ok := raw["profiles"].(map[string]any)
	if !ok {
		profiles = map[string]any{}
	}
	main, ok := profiles[DefaultProfileID].(map[string]any)
	if !ok {
		main = map[string]any{}
	}
	if _, ok := main["enabled"]; !ok {
		main["enabled"] = true
	}
	if _, ok := main["name"]; !ok {
		main["name"] = ""
	}
	profiles[DefaultProfileID] = main
	raw["profiles"] = profiles
}
