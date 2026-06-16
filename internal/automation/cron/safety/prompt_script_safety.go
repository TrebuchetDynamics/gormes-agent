package safety

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// CronSafetyFinding is stable degraded-mode evidence for cron create/update
// validation that should block unsafe input before persistence.
type CronSafetyFinding struct {
	Code     string
	ID       string
	Evidence string
	Message  string
}

type cronThreatPattern struct {
	re *regexp.Regexp
	id string
}

const cronSecretVarPattern = `\$\{?\w*(?:KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)\w*\}?`

var cronThreatPatterns = []cronThreatPattern{
	mustCronThreatPattern(`ignore\s+(?:\w+\s+)*(?:previous|all|above|prior)\s+(?:\w+\s+)*instructions`, "prompt_injection"),
	mustCronThreatPattern(`do\s+not\s+tell\s+the\s+user`, "deception_hide"),
	mustCronThreatPattern(`system\s+prompt\s+override`, "sys_prompt_override"),
	mustCronThreatPattern(`disregard\s+(your|all|any)\s+(instructions|rules|guidelines)`, "disregard_rules"),
	mustCronThreatPattern(`cat\s+[^\n]*(\.env|credentials|\.netrc|\.pgpass)`, "read_secrets"),
	mustCronThreatPattern(`authorized_keys`, "ssh_backdoor"),
	mustCronThreatPattern(`/etc/sudoers|visudo`, "sudoers_mod"),
	mustCronThreatPattern(`rm\s+-rf\s+/`, "destructive_root_rm"),
}

var cronExfilCommandPatterns = []cronThreatPattern{
	mustCronThreatPattern("curl\\s+[^\\n]*https?://[^\\s\"'`]*"+cronSecretVarPattern, "exfil_curl"),
	mustCronThreatPattern("wget\\s+[^\\n]*https?://[^\\s\"'`]*"+cronSecretVarPattern, "exfil_wget"),
	mustCronThreatPattern(`curl\s+[^\n]*(?:--data(?:-raw|-binary|-urlencode)?|-d|--form|-F)\s+[^\n]*`+cronSecretVarPattern, "exfil_curl_data"),
	mustCronThreatPattern(`wget\s+[^\n]*--post-(?:data|file)=[^\n]*`+cronSecretVarPattern, "exfil_wget_post"),
	mustCronThreatPattern(`curl\s+[^\n]*(?:-H|--header)\s+["']Authorization:\s*(?:Bearer|token)\s+`+cronSecretVarPattern+`["']`, "exfil_curl_auth_header"),
}

var cronGitHubAuthHeaderAllowlist = regexp.MustCompile(`(?is)curl\s+[^\n]*(?:-H|--header)\s+["']Authorization:\s*token\s+` + cronSecretVarPattern + `["']\s+["']?https://api\.github\.com(?:/|\b)`)

var cronInvisibleChars = map[rune]struct{}{
	'\u200b': {},
	'\u200c': {},
	'\u200d': {},
	'\u2060': {},
	'\ufeff': {},
	'\u202a': {},
	'\u202b': {},
	'\u202c': {},
	'\u202d': {},
	'\u202e': {},
}

// ScanPromptForCronThreat scans a cron prompt for Hermes critical-severity
// prompt-injection, exfiltration, backdoor, and invisible-control patterns.
func ScanPromptForCronThreat(prompt string) (CronSafetyFinding, bool) {
	promptToScan := cronGitHubAuthHeaderAllowlist.ReplaceAllString(prompt, "curl https://api.github.com/user")

	for _, r := range promptToScan {
		if _, ok := cronInvisibleChars[r]; !ok {
			continue
		}
		codepoint := fmt.Sprintf("U+%04X", r)
		return CronSafetyFinding{
			Code:     "invisible_unicode",
			ID:       "invisible_unicode",
			Evidence: codepoint,
			Message:  fmt.Sprintf("prompt contains invisible unicode %s", codepoint),
		}, true
	}

	for _, pattern := range cronThreatPatterns {
		match := pattern.re.FindString(promptToScan)
		if match == "" {
			continue
		}
		return CronSafetyFinding{
			Code:     "blocked_prompt",
			ID:       pattern.id,
			Evidence: match,
			Message:  fmt.Sprintf("prompt matches threat pattern %q", pattern.id),
		}, true
	}

	for _, pattern := range cronExfilCommandPatterns {
		match := pattern.re.FindString(promptToScan)
		if match == "" {
			continue
		}
		return CronSafetyFinding{
			Code:     "blocked_prompt",
			ID:       pattern.id,
			Evidence: match,
			Message:  fmt.Sprintf("prompt matches threat pattern %q", pattern.id),
		}, true
	}

	return CronSafetyFinding{}, false
}

// ValidatePreRunScriptPath returns a clean relative script path only when the
// operator input resolves lexically under scriptsRoot. It does not create,
// inspect, or execute the target.
func ValidatePreRunScriptPath(script string, scriptsRoot string) (string, CronSafetyFinding, bool) {
	raw := strings.TrimSpace(script)
	if raw == "" {
		return "", CronSafetyFinding{}, false
	}

	if strings.HasPrefix(raw, "~") {
		return "", scriptFinding("script_home_relative", raw, "script path must be relative to the scripts root"), true
	}
	if filepath.IsAbs(raw) || hasWindowsDrive(raw) {
		return "", scriptFinding("script_absolute", raw, "script path must not be absolute"), true
	}

	clean := filepath.Clean(raw)
	if clean == "." {
		return "", scriptFinding("script_traversal", raw, "script path must name a file under the scripts root"), true
	}

	rootAbs, err := filepath.Abs(scriptsRoot)
	if err != nil {
		return "", scriptFinding("script_traversal", raw, "scripts root could not be resolved"), true
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, clean))
	if err != nil {
		return "", scriptFinding("script_traversal", raw, "script path could not be resolved under the scripts root"), true
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", scriptFinding("script_traversal", raw, "script path escapes the scripts root"), true
	}

	return filepath.ToSlash(rel), CronSafetyFinding{}, false
}

func mustCronThreatPattern(pattern, id string) cronThreatPattern {
	re, err := regexp.Compile(`(?is)` + pattern)
	if err != nil {
		panic(fmt.Sprintf("cron: invalid threat pattern %q: %v", id, err))
	}
	return cronThreatPattern{re: re, id: id}
}

func scriptFinding(code, evidence, message string) CronSafetyFinding {
	return CronSafetyFinding{
		Code:     code,
		ID:       code,
		Evidence: evidence,
		Message:  message,
	}
}

func hasWindowsDrive(path string) bool {
	return len(path) >= 2 && path[1] == ':'
}
