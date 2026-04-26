package tools

import (
	"fmt"
	"regexp"
	"sync"
)

const (
	sshSensitivePath            = `(?:~|\$home|\$\{home\})/\.ssh(?:/|$)`
	hermesEnvPath               = `(?:~/\.hermes/|(?:\$home|\$\{home\})/\.hermes/|(?:\$hermes_home|\$\{hermes_home\})/)\.env\b`
	projectEnvPath              = "(?:(?:/|\\.{1,2}/)?(?:[^\\s/\"'`]+/)*\\.env(?:\\.[^/\\s\"'`]+)*)"
	projectConfigPath           = "(?:(?:/|\\.{1,2}/)?(?:[^\\s/\"'`]+/)*config\\.yaml)"
	sensitiveWriteTarget        = `(?:/etc/|/dev/sd|` + sshSensitivePath + `|` + hermesEnvPath + `)`
	projectSensitiveWriteTarget = `(?:` + projectEnvPath + `|` + projectConfigPath + `)`
	commandTail                 = `(?:\s*(?:&&|\|\||;).*)?$`
)

// DangerousPatterns is the recoverable dangerous-command pattern table,
// ported from Hermes DANGEROUS_PATTERNS at tools/approval.py@eb28145f.
//
// These rules describe commands that require approval but can be bypassed by
// future approval/yolo policy. GuardCommand checks DetectHardline before this
// table so unconditional hardline rules always win.
var DangerousPatterns = []HardlinePattern{
	{Regex: `\brm\s+(-[^\s]*\s+)*/`, Description: "delete in root path"},
	{Regex: `\brm\s+-[^\s]*r`, Description: "recursive delete"},
	{Regex: `\brm\s+--recursive\b`, Description: "recursive delete (long flag)"},
	{Regex: `\bchmod\s+(-[^\s]*\s+)*(777|666|o\+[rwx]*w|a\+[rwx]*w)\b`, Description: "world/other-writable permissions"},
	{Regex: `\bchmod\s+--recursive\b.*(777|666|o\+[rwx]*w|a\+[rwx]*w)`, Description: "recursive world/other-writable (long flag)"},
	{Regex: `\bchown\s+(-[^\s]*)?R\s+root`, Description: "recursive chown to root"},
	{Regex: `\bchown\s+--recursive\b.*root`, Description: "recursive chown to root (long flag)"},
	{Regex: `\bmkfs\b`, Description: "format filesystem"},
	{Regex: `\bdd\s+.*if=`, Description: "disk copy"},
	{Regex: `>\s*/dev/sd`, Description: "write to block device"},
	{Regex: `\bDROP\s+(TABLE|DATABASE)\b`, Description: "SQL DROP"},
	{Regex: `\bDELETE\s+FROM\b`, Description: "SQL DELETE without WHERE"},
	{Regex: `\bTRUNCATE\s+(TABLE)?\s*\w`, Description: "SQL TRUNCATE"},
	{Regex: `>\s*/etc/`, Description: "overwrite system config"},
	{Regex: `\bsystemctl\s+(-[^\s]+\s+)*(stop|restart|disable|mask)\b`, Description: "stop/restart system service"},
	{Regex: `\bkill\s+-9\s+-1\b`, Description: "kill all processes"},
	{Regex: `\bpkill\s+-9\b`, Description: "force kill processes"},
	{Regex: `:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`, Description: "fork bomb"},
	{Regex: `\b(bash|sh|zsh|ksh)\s+-[^\s]*c(\s+|$)`, Description: "shell command via -c/-lc flag"},
	{Regex: `\b(python[23]?|perl|ruby|node)\s+-[ec]\s+`, Description: "script execution via -e/-c flag"},
	{Regex: `\b(curl|wget)\b.*\|\s*(ba)?sh\b`, Description: "pipe remote content to shell"},
	{Regex: `\b(bash|sh|zsh|ksh)\s+<\s*<?\s*\(\s*(curl|wget)\b`, Description: "execute remote script via process substitution"},
	{Regex: `\btee\b.*["']?` + sensitiveWriteTarget, Description: "overwrite system file via tee"},
	{Regex: `>>?\s*["']?` + sensitiveWriteTarget, Description: "overwrite system file via redirection"},
	{Regex: `\btee\b.*["']?` + projectSensitiveWriteTarget + `["']?` + commandTail, Description: "overwrite project env/config via tee"},
	{Regex: `>>?\s*["']?` + projectSensitiveWriteTarget + `["']?` + commandTail, Description: "overwrite project env/config via redirection"},
	{Regex: `\bxargs\s+.*\brm\b`, Description: "xargs with rm"},
	{Regex: `\bfind\b.*-exec\s+(/\S*/)?rm\b`, Description: "find -exec rm"},
	{Regex: `\bfind\b.*-delete\b`, Description: "find -delete"},
	{Regex: `\bhermes\s+gateway\s+(stop|restart)\b`, Description: "stop/restart hermes gateway (kills running agents)"},
	{Regex: `\bhermes\s+update\b`, Description: "hermes update (restarts gateway, kills running agents)"},
	{Regex: `gateway\s+run\b.*(&\s*$|&\s*;|\bdisown\b|\bsetsid\b)`, Description: "start gateway outside systemd (use 'systemctl --user restart hermes-gateway')"},
	{Regex: `\bnohup\b.*gateway\s+run\b`, Description: "start gateway outside systemd (use 'systemctl --user restart hermes-gateway')"},
	{Regex: `\b(pkill|killall)\b.*\b(hermes|gateway|cli\.py)\b`, Description: "kill hermes/gateway process (self-termination)"},
	{Regex: `\bkill\b.*\$\(\s*pgrep\b`, Description: "kill process via pgrep expansion (self-termination)"},
	{Regex: "\\bkill\\b.*`\\s*pgrep\\b", Description: "kill process via backtick pgrep expansion (self-termination)"},
	{Regex: `\b(cp|mv|install)\b.*\s/etc/`, Description: "copy/move file into /etc/"},
	{Regex: `\b(cp|mv|install)\b.*\s["']?` + projectSensitiveWriteTarget + `["']?` + commandTail, Description: "overwrite project env/config file"},
	{Regex: `\bsed\s+-[^\s]*i.*\s/etc/`, Description: "in-place edit of system config"},
	{Regex: `\bsed\s+--in-place\b.*\s/etc/`, Description: "in-place edit of system config (long flag)"},
	{Regex: `\b(python[23]?|perl|ruby|node)\s+<<`, Description: "script execution via heredoc"},
	{Regex: `\bgit\s+reset\s+--hard\b`, Description: "git reset --hard (destroys uncommitted changes)"},
	{Regex: `\bgit\s+push\b.*--force\b`, Description: "git force push (rewrites remote history)"},
	{Regex: `\bgit\s+push\b.*-f\b`, Description: "git force push short flag (rewrites remote history)"},
	{Regex: `\bgit\s+clean\s+-[^\s]*f`, Description: "git clean with force (deletes untracked files)"},
	{Regex: `\bgit\s+branch\s+-D\b`, Description: "git branch force delete"},
	{Regex: `\bchmod\s+\+x\b.*[;&|]+\s*\./`, Description: "chmod +x followed by immediate execution"},
}

// BlockedResult is the pure guard result returned for commands that match a
// hardline or recoverable dangerous-command rule.
type BlockedResult struct {
	Approved         bool
	Hardline         bool
	ApprovalRequired bool
	Description      string
	Operator         string
	Command          string
	Evidence         map[string]string
}

type compiledDangerous struct {
	regex       *regexp.Regexp
	description string
}

var (
	dangerousCompileOnce sync.Once
	dangerousCompiled    []compiledDangerous
	deleteWherePattern   = regexp.MustCompile(`(?is)\bWHERE\b`)
)

func compileDangerousPatterns() {
	dangerousCompiled = make([]compiledDangerous, 0, len(DangerousPatterns))
	for _, p := range DangerousPatterns {
		re, err := regexp.Compile(`(?is)` + p.Regex)
		if err != nil {
			continue
		}
		dangerousCompiled = append(dangerousCompiled, compiledDangerous{regex: re, description: p.Description})
	}
}

func init() {
	for _, p := range DangerousPatterns {
		if _, err := regexp.Compile(`(?is)` + p.Regex); err != nil {
			panic(fmt.Sprintf("tools: invalid DangerousPattern %q: %v", p.Regex, err))
		}
	}
}

// DetectDangerous reports whether cmd matches any recoverable dangerous rule.
// On match it returns (true, description); otherwise it returns (false, "").
func DetectDangerous(cmd string) (bool, string) {
	if cmd == "" {
		return false, ""
	}
	dangerousCompileOnce.Do(compileDangerousPatterns)
	for _, c := range dangerousCompiled {
		if !c.regex.MatchString(cmd) {
			continue
		}
		if c.description == "SQL DELETE without WHERE" && deleteWherePattern.MatchString(cmd) {
			continue
		}
		return true, c.description
	}
	return false, ""
}

// GuardCommand applies the pure dangerous-command guard. Hardline matches
// always block first; recoverable dangerous matches require approval.
func GuardCommand(cmd, mode string) BlockedResult {
	if matched, description := DetectHardline(cmd); matched {
		return blockedResult(cmd, mode, description, "hardline", true, false)
	}
	if matched, description := DetectDangerous(cmd); matched {
		return blockedResult(cmd, mode, description, "dangerous", false, true)
	}
	return BlockedResult{}
}

func blockedResult(cmd, mode, description, detector string, hardline, approvalRequired bool) BlockedResult {
	return BlockedResult{
		Approved:         false,
		Hardline:         hardline,
		ApprovalRequired: approvalRequired,
		Description:      description,
		Operator:         mode,
		Command:          cmd,
		Evidence: map[string]string{
			"command":             cmd,
			"detector":            detector,
			"pattern_description": description,
		},
	}
}
