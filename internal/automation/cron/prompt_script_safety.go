package cron

import cronsafety "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron/safety"

// CronSafetyFinding is stable degraded-mode evidence for cron create/update
// safety checks.
type CronSafetyFinding = cronsafety.CronSafetyFinding

// ScanPromptForCronThreat scans a cron prompt for Hermes critical-severity
// prompt-injection and exfiltration patterns.
func ScanPromptForCronThreat(prompt string) (CronSafetyFinding, bool) {
	return cronsafety.ScanPromptForCronThreat(prompt)
}

// ValidatePreRunScriptPath returns a clean relative script path only when the
// requested pre-run script remains under scriptsRoot.
func ValidatePreRunScriptPath(script string, scriptsRoot string) (string, CronSafetyFinding, bool) {
	return cronsafety.ValidatePreRunScriptPath(script, scriptsRoot)
}
