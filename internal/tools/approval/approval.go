package approval

import (
	"regexp"
	"strings"
)

type PatternMatch struct {
	Pattern     string
	Description string
}

type CheckResult struct {
	Approved    bool   `json:"approved"`
	Hardline    bool   `json:"hardline,omitempty"`
	Dangerous   bool   `json:"dangerous,omitempty"`
	Description string `json:"description,omitempty"`
	Message     string `json:"message,omitempty"`
}

var (
	hardlinePatterns = []PatternMatch{
		{_cmdPos + `(?:[\w./-]*/)?(?:python(?:[23](?:\.\d+)?)?|pypy3?)(?:\s|$)`, "Python runtime execution is disabled in Gormes"},
		{_cmdPos + `uv\s+run(?:\s+[^\s]+)*\s+(?:[\w./-]*/)?(?:python(?:[23](?:\.\d+)?)?|pypy3?)(?:\s|$)`, "Python runtime execution is disabled in Gormes"},
		{`\brm\s+(-[^\s]*\s+)*(/|/\*|/ \*)(\s|$)`, "recursive delete of root filesystem"},
		{`\brm\s+(-[^\s]*\s+)*(/home|/home/\*|/root|/root/\*|/etc|/etc/\*|/usr|/usr/\*|/var|/var/\*|/bin|/bin/\*|/sbin|/sbin/\*|/boot|/boot/\*|/lib|/lib/\*)(\s|$)`, "recursive delete of system directory"},
		{`\brm\s+(-[^\s]*\s+)*(~|\$HOME)(/?|/\*)?(\s|$)`, "recursive delete of home directory"},
		{`\bmkfs(\.[a-z0-9]+)?\b`, "format filesystem (mkfs)"},
		{`\bdd\b[^\n]*\bof=/dev/(sd|nvme|hd|mmcblk|vd|xvd)[a-z0-9]*`, "dd to raw block device"},
		{`>\s*/dev/(sd|nvme|hd|mmcblk|vd|xvd)[a-z0-9]*\b`, "redirect to raw block device"},
		{`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`, "fork bomb"},
		{`\bkill\s+(-[^\s]+\s+)*-1\b`, "kill all processes"},
		{_cmdPos + `(shutdown|reboot|halt|poweroff)\b`, "system shutdown/reboot"},
		{_cmdPos + `init\s+[06]\b`, "init 0/6 (shutdown/reboot)"},
		{_cmdPos + `systemctl\s+(poweroff|reboot|halt|kexec)\b`, "systemctl poweroff/reboot"},
		{_cmdPos + `telinit\s+[06]\b`, "telinit 0/6 (shutdown/reboot)"},
	}

	dangerousPatterns = []PatternMatch{
		{`\brm\s+(-[^\s]*\s+)*/`, "delete in root path"},
		{`\brm\s+-[^\s]*r`, "recursive delete"},
		{`\brm\s+--recursive\b`, "recursive delete (long flag)"},
		{`\bchmod\s+(-[^\s]*\s+)*(777|666|o\+[rwx]*w|a\+[rwx]*w)\b`, "world/other-writable permissions"},
		{`\bchmod\s+--recursive\b.*(777|666|o\+[rwx]*w|a\+[rwx]*w)`, "recursive world/other-writable (long flag)"},
		{`\bchown\s+(-[^\s]*)?R\s+root`, "recursive chown to root"},
		{`\bchown\s+--recursive\b.*root`, "recursive chown to root (long flag)"},
		{`\bmkfs\b`, "format filesystem"},
		{`\bdd\s+.*if=`, "disk copy"},
		{`>\s*/dev/sd`, "write to block device"},
		{`\bDROP\s+(TABLE|DATABASE)\b`, "SQL DROP"},
		{`\bDELETE\s+FROM\b`, "SQL DELETE"},
		{`\bTRUNCATE\s+(TABLE)?\s*\w`, "SQL TRUNCATE"},
		{`>\s*/etc/`, "overwrite system config"},
		{`\bsystemctl\s+(-[^\s]+\s+)*(stop|restart|disable|mask)\b`, "stop/restart system service"},
		{`\bkill\s+-9\s+-1\b`, "kill all processes"},
		{`\bpkill\s+-9\b`, "force kill processes"},
		{`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`, "fork bomb"},
		{`\b(bash|sh|zsh|ksh)\s+-[^\s]*c(\s+|$)`, "shell command via -c flag"},
		{`\b(python[23]?|perl|ruby|node)\s+-[ec]\s+`, "script execution via -e/-c flag"},
		{`\b(curl|wget)\b.*\|\s*(ba)?sh\b`, "pipe remote content to shell"},
		{`\b(bash|sh|zsh|ksh)\s+<\s*<?\s*\(\s*(curl|wget)\b`, "execute remote script via process substitution"},
		{`\bxargs\s+.*\brm\b`, "xargs with rm"},
		{`\bfind\b.*-exec\s+(/\S*/)?rm\b`, "find -exec rm"},
		{`\bfind\b.*-delete\b`, "find -delete"},
		{`\bhermes\s+gateway\s+(stop|restart)\b`, "stop/restart hermes gateway"},
		{`\bhermes\s+update\b`, "hermes update"},
		{`gateway\s+run\b.*(&\s*$|&\s*;|\bdisown\b|\bsetsid\b)`, "start gateway outside systemd"},
		{`\bnohup\b.*gateway\s+run\b`, "start gateway outside systemd"},
		{`\b(pkill|killall)\b.*\b(hermes|gateway|cli\.py)\b`, "kill hermes/gateway process"},
		{`\bkill\b.*\$\(\s*pgrep\b`, "kill process via pgrep expansion"},
		{"\\bkill\\b.*`\\s*pgrep\\b", "kill process via backtick pgrep expansion"},
		{`\b(cp|mv|install)\b.*\s/etc/`, "copy/move file into /etc/"},
		{`\bsed\s+-[^\s]*i.*\s/etc/`, "in-place edit of system config"},
		{`\bsed\s+--in-place\b.*\s/etc/`, "in-place edit of system config (long flag)"},
		{`\b(python[23]?|perl|ruby|node)\s+<<`, "script execution via heredoc"},
		{`\bgit\s+reset\s+--hard\b`, "git reset --hard"},
		{`\bgit\s+push\b.*--force\b`, "git force push"},
		{`\bgit\s+push\b.*-f\b`, "git force push short flag"},
		{`\bgit\s+clean\s+-[^\s]*f`, "git clean with force"},
		{`\bgit\s+branch\s+-D\b`, "git branch force delete"},
		{`\bchmod\s+\+x\b.*[;&|]+\s*\./`, "chmod +x followed by immediate execution"},
	}
)

const _cmdPos = "(^|[;&|]\\s*|\\x60\\s*|sudo\\s+|env\\s+[^\\s]+\\s+|nohup\\s+|nice\\s+(-n\\s+\\d+\\s+)?)"

func hardlineBlockResult(desc string) CheckResult {
	return CheckResult{
		Approved:    false,
		Hardline:    true,
		Description: desc,
		Message: "BLOCKED (hardline): " + desc + ". This command is on the unconditional " +
			"blocklist and cannot be executed via the agent — not even with --yolo, " +
			"/yolo, approvals.mode=off, or cron approve mode.",
	}
}

func detectMatch(patterns []PatternMatch, cmd string) (bool, string) {
	normalized := strings.ToLower(normalizeCommand(cmd))
	for _, p := range patterns {
		re := regexp.MustCompile("(?i)" + p.Pattern)
		if re.MatchString(normalized) {
			return true, p.Description
		}
	}
	return false, ""
}

func normalizeCommand(cmd string) string {
	s := strings.ReplaceAll(cmd, "  ", " ")
	s = strings.TrimSpace(s)
	return s
}

func CheckHardline(command string) CheckResult {
	if match, desc := detectMatch(hardlinePatterns, command); match {
		return hardlineBlockResult(desc)
	}
	return CheckResult{Approved: true}
}

func CheckDangerous(command string) (bool, string) {
	return detectMatch(dangerousPatterns, command)
}

func CheckAll(command string) CheckResult {
	if result := CheckHardline(command); !result.Approved {
		return result
	}
	if match, desc := CheckDangerous(command); match {
		return CheckResult{
			Approved:    false,
			Dangerous:   true,
			Description: desc,
			Message:     "This command matched dangerous pattern: " + desc + ". Approval required.",
		}
	}
	return CheckResult{Approved: true}
}
