package config

import configcheck "github.com/TrebuchetDynamics/gormes-agent/internal/config/configcheck"

// CheckIssue describes one structural problem detected by Check. Field
// names operator-targeted dotted paths; Severity is "error" for breaking
// problems and "warning" for recoverable ones.
type CheckIssue = configcheck.Issue

// CheckReport is the structured result returned by Check. It is read-only:
// no file under XDG_CONFIG_HOME is mutated by producing this value.
type CheckReport = configcheck.Report

// Check inspects the on-disk Gormes config without mutating it. It returns
// the resolved config_version, the latest version this binary writes, the
// dotenv presence flag, and a list of structural issues. A non-nil error is
// returned when the loaded config_version is from a newer binary; the
// report is still populated with the raw version for operator evidence.
func Check() (CheckReport, error) {
	return configcheck.Check()
}

// MigrateResult is the structured outcome of MigrateConfigFile.
type MigrateResult = configcheck.MigrateResult

// MigrateConfigFile applies native schema migrations atomically. It is a
// no-op when the file is already at CurrentConfigVersion. It rejects files
// from a newer binary without rewriting them. The write path uses a
// temp-file-then-rename so a partial write cannot corrupt config.toml.
func MigrateConfigFile(path string) (MigrateResult, error) {
	return configcheck.MigrateFile(path)
}
