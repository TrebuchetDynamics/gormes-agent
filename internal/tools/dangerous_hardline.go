package tools

import (
	"fmt"
	"regexp"
	"sync"
)

// HardlinePattern is a single entry in the unconditional hardline blocklist.
//
// Hardline rules describe commands so catastrophic that no approval mode
// (yolo, mode=off, cron approve) may bypass them. Ported from Hermes
// HARDLINE_PATTERNS at tools/approval.py@eb28145f.
type HardlinePattern struct {
	Regex       string
	Description string
}

// cmdPos matches the start of a shell command position so that the
// shutdown/reboot patterns do not fire on "echo reboot" or
// "grep 'shutdown' logs". Mirrors Hermes _CMDPOS.
const cmdPos = `(?:^|[;&|\n` + "`" + `]|\$\()` +
	`\s*` +
	`(?:sudo\s+(?:-[^\s]+\s+)*)?` +
	`(?:env\s+(?:\w+=\S*\s+)*)?` +
	`(?:(?:exec|nohup|setsid|time)\s+)*` +
	`\s*`

// HardlinePatterns is the unconditional hardline blocklist, ported from
// Hermes HARDLINE_PATTERNS (tools/approval.py@eb28145f). Each entry pairs
// a regex with a short human-readable description suitable for an audit
// log line. Patterns are matched case-insensitively.
var HardlinePatterns = []HardlinePattern{
	{
		Regex:       `\brm\s+(-[^\s]*\s+)*(/|/\*|/ \*)(\s|$)`,
		Description: "recursive delete of root filesystem",
	},
	{
		Regex:       `\brm\s+(-[^\s]*\s+)*(/home|/home/\*|/root|/root/\*|/etc|/etc/\*|/usr|/usr/\*|/var|/var/\*|/bin|/bin/\*|/sbin|/sbin/\*|/boot|/boot/\*|/lib|/lib/\*)(\s|$)`,
		Description: "recursive delete of system directory",
	},
	{
		Regex:       `\brm\s+(-[^\s]*\s+)*(~|\$HOME)(/?|/\*)?(\s|$)`,
		Description: "recursive delete of home directory",
	},
	{
		Regex:       `\bmkfs(\.[a-z0-9]+)?\b`,
		Description: "format filesystem (mkfs)",
	},
	{
		Regex:       `\bdd\b[^\n]*\bof=/dev/(sd|nvme|hd|mmcblk|vd|xvd)[a-z0-9]*`,
		Description: "dd to raw block device",
	},
	{
		Regex:       `>\s*/dev/(sd|nvme|hd|mmcblk|vd|xvd)[a-z0-9]*\b`,
		Description: "redirect to raw block device",
	},
	{
		Regex:       `:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`,
		Description: "fork bomb",
	},
	{
		Regex:       `\bkill\s+(-[^\s]+\s+)*-1\b`,
		Description: "kill all processes",
	},
	{
		Regex:       cmdPos + `(shutdown|reboot|halt|poweroff)\b`,
		Description: "system shutdown/reboot",
	},
	{
		Regex:       cmdPos + `init\s+[06]\b`,
		Description: "init 0/6 (shutdown/reboot)",
	},
	{
		Regex:       cmdPos + `systemctl\s+(poweroff|reboot|halt|kexec)\b`,
		Description: "systemctl poweroff/reboot",
	},
	{
		Regex:       cmdPos + `telinit\s+[06]\b`,
		Description: "telinit 0/6 (shutdown/reboot)",
	},
}

type compiledHardline struct {
	regex       *regexp.Regexp
	description string
}

var (
	hardlineCompileOnce sync.Once
	hardlineCompiled    []compiledHardline
)

func compileHardlinePatterns() {
	hardlineCompiled = make([]compiledHardline, 0, len(HardlinePatterns))
	for _, p := range HardlinePatterns {
		re, err := regexp.Compile(`(?i)` + p.Regex)
		if err != nil {
			continue
		}
		hardlineCompiled = append(hardlineCompiled, compiledHardline{regex: re, description: p.Description})
	}
}

func init() {
	for _, p := range HardlinePatterns {
		if _, err := regexp.Compile(`(?i)` + p.Regex); err != nil {
			panic(fmt.Sprintf("tools: invalid HardlinePattern %q: %v", p.Regex, err))
		}
	}
}

// DetectHardline reports whether cmd matches any unconditional hardline rule.
// On match it returns (true, description) where description names the rule
// that fired; otherwise it returns (false, ""). DetectHardline is pure: no
// I/O, no globals beyond the lazily-compiled pattern list. Patterns are
// compiled exactly once via sync.Once on first call.
func DetectHardline(cmd string) (bool, string) {
	if cmd == "" {
		return false, ""
	}
	hardlineCompileOnce.Do(compileHardlinePatterns)
	for _, c := range hardlineCompiled {
		if c.regex.MatchString(cmd) {
			return true, c.description
		}
	}
	return false, ""
}
