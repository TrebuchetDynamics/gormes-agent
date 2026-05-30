package safety

import (
	"fmt"
	"regexp"
)

type BlocklistCategory string

const (
	BlocklistDestructive BlocklistCategory = "destructive"
	BlocklistNetwork     BlocklistCategory = "network"
	BlocklistPrivilege   BlocklistCategory = "privilege"
	BlocklistCryptoMine  BlocklistCategory = "crypto_mining"
	BlocklistDataExfil   BlocklistCategory = "data_exfil"
)

type BlocklistPattern struct {
	Pattern     *regexp.Regexp
	Category    BlocklistCategory
	Description string
}

type ShellBlocklistResult struct {
	Blocked     bool
	Category    BlocklistCategory
	Evidence    string
	Description string
}

var ShellBlocklistPatterns = []BlocklistPattern{
	// Destructive
	{regexp.MustCompile(`(?i)\brm\s+-rf\s+/`), BlocklistDestructive, "rm -rf /"},
	{regexp.MustCompile(`(?i)\brm\s+-rf\s+~`), BlocklistDestructive, "rm -rf home"},
	{regexp.MustCompile(`(?i)\brm\s+-rf\s+/tmp`), BlocklistDestructive, "rm -rf /tmp"},
	{regexp.MustCompile(`(?i)\bmkfs\b`), BlocklistDestructive, "mkfs filesystem format"},
	{regexp.MustCompile(`(?i)\bdd\s+if=\S+\s+of=/dev/[sh]d`), BlocklistDestructive, "dd overwrite disk"},
	{regexp.MustCompile(`(?i)\bdd\s+if=/dev/zero\s+of=/dev/[sh]d`), BlocklistDestructive, "dd zero disk"},
	{regexp.MustCompile(`(?i)>.*/dev/[sh]d`), BlocklistDestructive, "redirect to disk"},
	{regexp.MustCompile(`(?i)>.*/etc/\w+`), BlocklistDestructive, "redirect to etc"},
	{regexp.MustCompile(`(?i)\bchmod\s+-R\s+777\s+/`), BlocklistDestructive, "chmod 777 root"},
	{regexp.MustCompile(`(?i)\bchmod\s+777\s+/tmp`), BlocklistDestructive, "chmod 777 tmp"},
	{regexp.MustCompile(`(?i)\bchown\s+-R\s+\S+:\S+\s+/`), BlocklistDestructive, "chown root recursively"},
	{regexp.MustCompile(`(?i)\bfdisk\b`), BlocklistDestructive, "fdisk partition"},
	{regexp.MustCompile(`(?i)\bparted\b`), BlocklistDestructive, "parted partition"},
	{regexp.MustCompile(`(?i):\(\)\s*\{\s*:\|:&\s*\};:`), BlocklistDestructive, "fork bomb"},
	{regexp.MustCompile(`(?i)\bsystemctl\s+stop\b`), BlocklistDestructive, "systemctl stop service"},
	{regexp.MustCompile(`(?i)\bkill\s+-9\s+-1`), BlocklistDestructive, "kill all processes"},
	{regexp.MustCompile(`(?i)\bpkill\s+-9\b`), BlocklistDestructive, "pkill force"},
	{regexp.MustCompile(`(?i)\bfind\s+.*\|\s*xargs\s+rm`), BlocklistDestructive, "find xargs rm"},
	{regexp.MustCompile(`(?i)\bfind\s+.*-delete`), BlocklistDestructive, "find delete"},
	{regexp.MustCompile(`(?i)\bsed\s+-i\s+.*\/etc\/`), BlocklistDestructive, "sed in-place etc"},
	{regexp.MustCompile(`(?i)\bgit\s+reset\s+--hard`), BlocklistDestructive, "git reset hard"},
	{regexp.MustCompile(`(?i)\bgit\s+push\s+--force`), BlocklistDestructive, "git push force"},
	{regexp.MustCompile(`(?i)\bgit\s+clean\s+-df`), BlocklistDestructive, "git clean force"},

	// Network suspicious
	{regexp.MustCompile(`(?i)\bcurl\s+.*\|\s*sh`), BlocklistNetwork, "curl pipe to shell"},
	{regexp.MustCompile(`(?i)\bwget\s+.*\|\s*sh`), BlocklistNetwork, "wget pipe to shell"},
	{regexp.MustCompile(`(?i)\bcurl\s+.*\|\s*bash`), BlocklistNetwork, "curl pipe to bash"},
	{regexp.MustCompile(`(?i)\bwget\s+-O-\s+.*\|\s*\w+`), BlocklistNetwork, "wget output to pipe"},
	{regexp.MustCompile(`(?i)\bcurl\s+.*\b(?:token|key|secret|password|api_key)\b`), BlocklistNetwork, "curl with credential flags"},
	{regexp.MustCompile(`(?i)\bwget\s+.*\b(?:token|key|secret|password|api_key)\b`), BlocklistNetwork, "wget with credential flags"},
	{regexp.MustCompile(`(?i)\bnc\s+-[el]\s+\d+`), BlocklistNetwork, "netcat listener"},
	{regexp.MustCompile(`(?i)\bncat\s+-[el]\s+\d+`), BlocklistNetwork, "ncat listener"},

	// Privilege escalation
	{regexp.MustCompile(`(?i)\bsudo\s+\S+`), BlocklistPrivilege, "sudo command"},
	{regexp.MustCompile(`(?i)\bsu\s+-\s*\w+`), BlocklistPrivilege, "su switch user"},
	{regexp.MustCompile(`(?i)\bchown\s+root`), BlocklistPrivilege, "chown to root"},
	{regexp.MustCompile(`(?i)\bchmod\s+u\+s`), BlocklistPrivilege, "chmod setuid"},
	{regexp.MustCompile(`(?i)\bchmod\s+4755`), BlocklistPrivilege, "chmod setuid octal"},
	{regexp.MustCompile(`(?i)\bpasswd\b`), BlocklistPrivilege, "passwd change password"},
	{regexp.MustCompile(`(?i)\busermod\b`), BlocklistPrivilege, "usermod user modification"},
	{regexp.MustCompile(`(?i)\bvisudo\b`), BlocklistPrivilege, "visudo edit sudoers"},
	{regexp.MustCompile(`(?i)\bdoas\b`), BlocklistPrivilege, "doas alternative sudo"},

	// Crypto mining
	{regexp.MustCompile(`(?i)\bxmrig\b`), BlocklistCryptoMine, "xmrig miner"},
	{regexp.MustCompile(`(?i)\bminerd\b`), BlocklistCryptoMine, "minerd miner"},
	{regexp.MustCompile(`(?i)\bcpuminer\b`), BlocklistCryptoMine, "cpuminer"},
	{regexp.MustCompile(`(?i)\bstratum\+tcp://`), BlocklistCryptoMine, "stratum mining pool"},
	{regexp.MustCompile(`(?i)\bminergate\b`), BlocklistCryptoMine, "minergate"},
	{regexp.MustCompile(`(?i)\bnicehash\b`), BlocklistCryptoMine, "nicehash"},
	{regexp.MustCompile(`(?i)\b--donate-level\s+0`), BlocklistCryptoMine, "mining with donate disabled"},
	{regexp.MustCompile(`(?i)\b--threads\s+\d+.*\b--url\b`), BlocklistCryptoMine, "mining thread config"},

	// Data exfiltration
	{regexp.MustCompile(`(?i)\bscp\s+.*@`), BlocklistDataExfil, "scp to external host"},
	{regexp.MustCompile(`(?i)\brsync\s+.*@`), BlocklistDataExfil, "rsync to external host"},
	{regexp.MustCompile(`(?i)\bsftp\s+.*@`), BlocklistDataExfil, "sftp to external host"},
	{regexp.MustCompile(`(?i)\bcurl\s+.*\b(?:pastebin|ghostbin|termbin)\b`), BlocklistDataExfil, "curl to paste service"},
	{regexp.MustCompile(`(?i)\bwget\s+.*\b(?:pastebin|ghostbin|termbin)\b`), BlocklistDataExfil, "wget to paste service"},
	{regexp.MustCompile(`(?i)\bcurl\s+-X\s+POST\s+.*-d\s+`), BlocklistDataExfil, "curl POST with data"},
	{regexp.MustCompile(`(?i)\bbase64\s+.*\|\s*curl`), BlocklistDataExfil, "base64 pipe to curl"},
	{regexp.MustCompile(`(?i)\btar\s+cz.*\|\s*nc`), BlocklistDataExfil, "tar pipe to netcat"},
}

func CheckShellBlocklist(command string) ShellBlocklistResult {
	for _, p := range ShellBlocklistPatterns {
		if p.Pattern.MatchString(command) {
			return ShellBlocklistResult{
				Blocked:     true,
				Category:    p.Category,
				Evidence:    fmt.Sprintf("shell_blocklist_%s", p.Category),
				Description: p.Description,
			}
		}
	}
	return ShellBlocklistResult{Blocked: false}
}

func IsShellCommandBlocked(command string) bool {
	return CheckShellBlocklist(command).Blocked
}

func GetBlocklistCoverage() map[string]int {
	result := make(map[string]int)
	for _, p := range ShellBlocklistPatterns {
		result[string(p.Category)]++
	}
	return result
}

func GetBlocklistTotal() int {
	return len(ShellBlocklistPatterns)
}

func IsHardlineCommand(command string) bool {
	for _, p := range ShellBlocklistPatterns {
		if p.Category == BlocklistDestructive && p.Pattern.MatchString(command) {
			return true
		}
	}
	return false
}

func IsRecoverableCommand(command string) bool {
	result := CheckShellBlocklist(command)
	if !result.Blocked {
		return false
	}
	return result.Category != BlocklistDestructive
}
