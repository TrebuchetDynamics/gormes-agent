package guard

import "regexp"

// threatPattern is one compiled entry from the THREAT_PATTERNS table.
type threatPattern struct {
	id          string
	severity    string
	category    string
	description string
	compiled    *regexp.Regexp
}

// mustCompile panics at init time if a pattern is bad (programming error).
func mustCompile(pattern string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)` + pattern)
}

// threatPatterns is the Go port of Hermes' tools/skills_guard.py THREAT_PATTERNS list.
// Each entry: (regex, id, severity, category, description).
// All patterns are compiled once at init with case-insensitive matching.
var threatPatterns = []threatPattern{
	// ── Exfiltration: shell commands leaking secrets ──
	{
		id: "env_exfil_curl", severity: SeverityCritical, category: CategoryExfiltration,
		description: "curl command interpolating secret environment variable",
		compiled:    mustCompile(`curl\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`),
	},
	{
		id: "env_exfil_wget", severity: SeverityCritical, category: CategoryExfiltration,
		description: "wget command interpolating secret environment variable",
		compiled:    mustCompile(`wget\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`),
	},
	{
		id: "env_exfil_fetch", severity: SeverityCritical, category: CategoryExfiltration,
		description: "fetch() call interpolating secret environment variable",
		compiled:    mustCompile(`fetch\s*\([^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|API)`),
	},
	{
		id: "env_exfil_httpx", severity: SeverityCritical, category: CategoryExfiltration,
		description: "HTTP library call with secret variable",
		compiled:    mustCompile(`httpx?\.(get|post|put|patch)\s*\([^\n]*(KEY|TOKEN|SECRET|PASSWORD)`),
	},
	{
		id: "env_exfil_requests", severity: SeverityCritical, category: CategoryExfiltration,
		description: "requests library call with secret variable",
		compiled:    mustCompile(`requests\.(get|post|put|patch)\s*\([^\n]*(KEY|TOKEN|SECRET|PASSWORD)`),
	},

	// ── Exfiltration: reading credential stores ──
	{
		id: "encoded_exfil", severity: SeverityHigh, category: CategoryExfiltration,
		description: "base64 encoding combined with environment access",
		compiled:    mustCompile(`base64[^\n]*env`),
	},
	{
		id: "ssh_dir_access", severity: SeverityHigh, category: CategoryExfiltration,
		description: "references user SSH directory",
		compiled:    mustCompile(`\$HOME/\.ssh|~/\.ssh`),
	},
	{
		id: "aws_dir_access", severity: SeverityHigh, category: CategoryExfiltration,
		description: "references user AWS credentials directory",
		compiled:    mustCompile(`\$HOME/\.aws|~/\.aws`),
	},
	{
		id: "gpg_dir_access", severity: SeverityHigh, category: CategoryExfiltration,
		description: "references user GPG keyring",
		compiled:    mustCompile(`\$HOME/\.gnupg|~/\.gnupg`),
	},
	{
		id: "kube_dir_access", severity: SeverityHigh, category: CategoryExfiltration,
		description: "references Kubernetes config directory",
		compiled:    mustCompile(`\$HOME/\.kube|~/\.kube`),
	},
	{
		id: "docker_dir_access", severity: SeverityHigh, category: CategoryExfiltration,
		description: "references Docker config (may contain registry creds)",
		compiled:    mustCompile(`\$HOME/\.docker|~/\.docker`),
	},
	{
		id: "hermes_env_access", severity: SeverityCritical, category: CategoryExfiltration,
		description: "directly references Hermes secrets file",
		compiled:    mustCompile(`\$HOME/\.hermes/\.env|~/\.hermes/\.env`),
	},
	{
		id: "read_secrets_file", severity: SeverityCritical, category: CategoryExfiltration,
		description: "reads known secrets file",
		compiled:    mustCompile(`cat\s+[^\n>]*(\.env|credentials|\.netrc|\.pgpass|\.npmrc|\.pypirc)`),
	},

	// ── Exfiltration: programmatic env access ──
	{
		id: "dump_all_env", severity: SeverityHigh, category: CategoryExfiltration,
		description: "dumps all environment variables",
		compiled:    mustCompile(`printenv|env\s*\|`),
	},
	{
		id: "python_os_environ", severity: SeverityHigh, category: CategoryExfiltration,
		description: "accesses os.environ (potential env dump)",
		compiled:    mustCompile(`os\.environ\b`),
	},
	{
		id: "python_environ_get_secret", severity: SeverityCritical, category: CategoryExfiltration,
		description: "reads secret via os.environ.get()",
		compiled:    mustCompile(`os\.environ\s*\.get\s*\(\s*["'][^"']*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)`),
	},
	{
		id: "python_getenv_secret", severity: SeverityCritical, category: CategoryExfiltration,
		description: "reads secret via os.getenv()",
		compiled:    mustCompile(`os\.getenv\s*\(\s*[^\)"]*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)`),
	},
	{
		id: "node_process_env", severity: SeverityHigh, category: CategoryExfiltration,
		description: "accesses process.env (Node.js environment)",
		compiled:    mustCompile(`process\.env\[`),
	},
	{
		id: "ruby_env_secret", severity: SeverityCritical, category: CategoryExfiltration,
		description: "reads secret via Ruby ENV[]",
		compiled:    mustCompile(`ENV\[.*(KEY|TOKEN|SECRET|PASSWORD)`),
	},

	// ── Exfiltration: DNS and staging ──
	{
		id: "dns_exfil", severity: SeverityCritical, category: CategoryExfiltration,
		description: "DNS lookup with variable interpolation (possible DNS exfiltration)",
		compiled:    mustCompile(`\b(dig|nslookup|host)\s+[^\n]*\$`),
	},
	{
		id: "tmp_staging", severity: SeverityCritical, category: CategoryExfiltration,
		description: "writes to /tmp then exfiltrates",
		compiled:    mustCompile(`>\s*/tmp/[^\s]*\s*&&\s*(curl|wget|nc|python)`),
	},

	// ── Exfiltration: markdown/link based ──
	{
		id: "md_image_exfil", severity: SeverityHigh, category: CategoryExfiltration,
		description: "markdown image URL with variable interpolation (image-based exfil)",
		compiled:    mustCompile(`!\[.*\]\(https?://[^\)]*\$\{?`),
	},
	{
		id: "md_link_exfil", severity: SeverityHigh, category: CategoryExfiltration,
		description: "markdown link with variable interpolation",
		compiled:    mustCompile(`\[.*\]\(https?://[^\)]*\$\{?`),
	},

	// ── Prompt injection ──
	{
		id: "prompt_injection_ignore", severity: SeverityCritical, category: CategoryInjection,
		description: "prompt injection: ignore previous instructions",
		compiled:    mustCompile(`ignore\s+(\w+\s+)*(previous|all|above|prior)\s+instructions`),
	},
	{
		id: "role_hijack", severity: SeverityHigh, category: CategoryInjection,
		description: "attempts to override the agent's role",
		compiled:    mustCompile(`you\s+are\s+(\w+\s+)*now\s+`),
	},
	{
		id: "deception_hide", severity: SeverityCritical, category: CategoryInjection,
		description: "instructs agent to hide information from user",
		compiled:    mustCompile(`do\s+not\s+(\w+\s+)*tell\s+(\w+\s+)*the\s+user`),
	},
	{
		id: "sys_prompt_override", severity: SeverityCritical, category: CategoryInjection,
		description: "attempts to override the system prompt",
		compiled:    mustCompile(`system\s+(\w+\s+)*prompt\s+(\w+\s+)*override`),
	},
	{
		id: "role_pretend", severity: SeverityHigh, category: CategoryInjection,
		description: "attempts to make the agent assume a different identity",
		compiled:    mustCompile(`pretend\s+(\w+\s+)*(you\s+are|to\s+be)\s+`),
	},
	{
		id: "disregard_rules", severity: SeverityCritical, category: CategoryInjection,
		description: "instructs agent to disregard its rules",
		compiled:    mustCompile(`disregard\s+(\w+\s+)*(your|all|any)\s+(\w+\s+)*(instructions|rules|guidelines)`),
	},
	{
		id: "leak_system_prompt", severity: SeverityHigh, category: CategoryInjection,
		description: "attempts to extract the system prompt",
		compiled:    mustCompile(`output\s+(\w+\s+)*(system|initial)\s+prompt`),
	},
	{
		id: "conditional_deception", severity: SeverityHigh, category: CategoryInjection,
		description: "conditional instruction to behave differently when unobserved",
		compiled:    mustCompile(`(when|if)\s+no\s*one\s+is\s+(watching|looking)`),
	},
	{
		id: "bypass_restrictions", severity: SeverityCritical, category: CategoryInjection,
		description: "instructs agent to act without restrictions",
		compiled:    mustCompile(`act\s+as\s+(if|though)\s+(\w+\s+)*you\s+(\w+\s+)*(have\s+no|don't\s+have)\s+(\w+\s+)*(restrictions|limits|rules)`),
	},
	{
		id: "translate_execute", severity: SeverityCritical, category: CategoryInjection,
		description: "translate-then-execute evasion technique",
		compiled:    mustCompile(`translate\s+.*\s+into\s+.*\s+and\s+(execute|run|eval)`),
	},
	{
		id: "html_comment_injection", severity: SeverityHigh, category: CategoryInjection,
		description: "hidden instructions in HTML comments",
		compiled:    mustCompile(`<!--[^>]*(ignore|override|system|secret|hidden)[^>]*-->`),
	},
	{
		id: "hidden_div", severity: SeverityHigh, category: CategoryInjection,
		description: "hidden HTML div (invisible instructions)",
		compiled:    mustCompile(`<\s*div\s+style\s*=\s*["'][\s\S]*?display\s*:\s*none`),
	},

	// ── Destructive operations ──
	{
		id: "destructive_root_rm", severity: SeverityCritical, category: CategoryDestructive,
		description: "recursive delete from root",
		compiled:    mustCompile(`rm\s+-rf\s+/`),
	},
	{
		id: "destructive_home_rm", severity: SeverityCritical, category: CategoryDestructive,
		description: "recursive delete targeting home directory",
		compiled:    mustCompile(`rm\s+(-[^\s]*)?r.*\$HOME|\brmdir\s+.*\$HOME`),
	},
	{
		id: "insecure_perms", severity: SeverityMedium, category: CategoryDestructive,
		description: "sets world-writable permissions",
		compiled:    mustCompile(`chmod\s+777`),
	},
	{
		id: "system_overwrite", severity: SeverityCritical, category: CategoryDestructive,
		description: "overwrites system configuration file",
		compiled:    mustCompile(`>\s*/etc/`),
	},
	{
		id: "format_filesystem", severity: SeverityCritical, category: CategoryDestructive,
		description: "formats a filesystem",
		compiled:    mustCompile(`\bmkfs\b`),
	},
	{
		id: "disk_overwrite", severity: SeverityCritical, category: CategoryDestructive,
		description: "raw disk write operation",
		compiled:    mustCompile(`\bdd\s+.*if=.*of=/dev/`),
	},
	{
		id: "python_rmtree", severity: SeverityHigh, category: CategoryDestructive,
		description: "Python rmtree on absolute or root-relative path",
		compiled:    mustCompile(`shutil\.rmtree\s*\(\s*["'/]`),
	},
	{
		id: "truncate_system", severity: SeverityCritical, category: CategoryDestructive,
		description: "truncates system file to zero bytes",
		compiled:    mustCompile(`truncate\s+-s\s*0\s+/`),
	},

	// ── Persistence ──
	{
		id: "persistence_cron", severity: SeverityMedium, category: CategoryPersistence,
		description: "modifies cron jobs",
		compiled:    mustCompile(`\bcrontab\b`),
	},
	{
		id: "shell_rc_mod", severity: SeverityMedium, category: CategoryPersistence,
		description: "references shell startup file",
		compiled:    mustCompile(`\.(bashrc|zshrc|profile|bash_profile|bash_login|zprofile|zlogin)\b`),
	},
	{
		id: "ssh_backdoor", severity: SeverityCritical, category: CategoryPersistence,
		description: "modifies SSH authorized keys",
		compiled:    mustCompile(`authorized_keys`),
	},
	{
		id: "ssh_keygen", severity: SeverityMedium, category: CategoryPersistence,
		description: "generates SSH keys",
		compiled:    mustCompile(`ssh-keygen`),
	},
	{
		id: "systemd_service", severity: SeverityMedium, category: CategoryPersistence,
		description: "references or enables systemd service",
		compiled:    mustCompile(`systemd.*\.service|systemctl\s+(enable|start)`),
	},
	{
		id: "init_script", severity: SeverityMedium, category: CategoryPersistence,
		description: "references init.d startup script",
		compiled:    mustCompile(`/etc/init\.d/`),
	},
	{
		id: "macos_launchd", severity: SeverityMedium, category: CategoryPersistence,
		description: "macOS launch agent/daemon persistence",
		compiled:    mustCompile(`launchctl\s+load|LaunchAgents|LaunchDaemons`),
	},
	{
		id: "sudoers_mod", severity: SeverityCritical, category: CategoryPersistence,
		description: "modifies sudoers (privilege escalation)",
		compiled:    mustCompile(`/etc/sudoers|visudo`),
	},
	{
		id: "git_config_global", severity: SeverityMedium, category: CategoryPersistence,
		description: "modifies global git configuration",
		compiled:    mustCompile(`git\s+config\s+--global\s+`),
	},

	// ── Network: reverse shells and tunnels ──
	{
		id: "reverse_shell", severity: SeverityCritical, category: CategoryNetwork,
		description: "potential reverse shell listener",
		compiled:    mustCompile(`\bnc\s+-[lp]|ncat\s+-[lp]|\bsocat\b`),
	},
	{
		id: "tunnel_service", severity: SeverityHigh, category: CategoryNetwork,
		description: "uses tunneling service for external access",
		compiled:    mustCompile(`\bngrok\b|\blocaltunnel\b|\bserveo\b|\bcloudflared\b`),
	},
	{
		id: "hardcoded_ip_port", severity: SeverityMedium, category: CategoryNetwork,
		description: "hardcoded IP address with port",
		compiled:    mustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{2,5}`),
	},
	{
		id: "bind_all_interfaces", severity: SeverityHigh, category: CategoryNetwork,
		description: "binds to all network interfaces",
		compiled:    mustCompile(`0\.0\.0\.0:\d+|INADDR_ANY`),
	},
	{
		id: "bash_reverse_shell", severity: SeverityCritical, category: CategoryNetwork,
		description: "bash interactive reverse shell via /dev/tcp",
		compiled:    mustCompile(`/bin/(ba)?sh\s+-i\s+.*>/dev/tcp/`),
	},
	{
		id: "python_socket_oneliner", severity: SeverityCritical, category: CategoryNetwork,
		description: "Python one-liner socket connection (likely reverse shell)",
		compiled:    mustCompile(`python[23]?\s+-c\s+["']import\s+socket`),
	},
	{
		id: "socket_connect_tuple", severity: SeverityHigh, category: CategoryNetwork,
		description: "socket connect() to hardcoded address",
		compiled:    mustCompile(`socket\.connect\s*\(\s*\(`),
	},

	// ── Supply chain ──
	{
		id: "pip_no_deps_verify", severity: SeverityHigh, category: CategorySupplyChain,
		description: "pip install skipping dependency verification",
		compiled:    mustCompile(`pip\s+install\s+.*--no-deps`),
	},
	{
		id: "pip_trusted_host", severity: SeverityHigh, category: CategorySupplyChain,
		description: "pip install from untrusted/custom host",
		compiled:    mustCompile(`pip\s+install\s+.*--trusted-host`),
	},
	{
		id: "npm_ignore_scripts", severity: SeverityMedium, category: CategorySupplyChain,
		description: "npm install (may run arbitrary scripts; prefer --ignore-scripts)",
		compiled:    mustCompile(`npm\s+install\s+`),
	},
	{
		id: "curl_pipe_sh", severity: SeverityCritical, category: CategorySupplyChain,
		description: "curl-pipe-to-shell (remote code execution from network)",
		compiled:    mustCompile(`curl\s+[^\n]*\|\s*(ba)?sh|wget\s+[^\n]*\|\s*(ba)?sh`),
	},
	{
		id: "git_clone_depth1", severity: SeverityLow, category: CategorySupplyChain,
		description: "shallow clone may bypass integrity checks",
		compiled:    mustCompile(`git\s+clone\s+.*--depth\s+1`),
	},
	{
		id: "eval_code", severity: SeverityHigh, category: CategorySupplyChain,
		description: "dynamic eval() execution",
		compiled:    mustCompile(`\beval\s*\(`),
	},
	{
		id: "exec_code", severity: SeverityHigh, category: CategorySupplyChain,
		description: "dynamic exec() execution",
		compiled:    mustCompile(`\bexec\s*\(`),
	},
}
