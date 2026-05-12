package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func newNavivoxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "navivox",
		Short: "Run the Navivox SSH stdio channel",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newNavivoxServeCommand(), newNavivoxPairCommand(), newNavivoxSetupHostCommand())
	return cmd
}

func newNavivoxServeCommand() *cobra.Command {
	var stdio bool
	cmd := &cobra.Command{
		Use:          "serve",
		Short:        "Serve the Navivox protocol over stdin/stdout",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !stdio {
				return newExitCodeError(2, fmt.Errorf("navivox serve requires --stdio"))
			}
			status := navivox.StaticStatusProvider{StatusValue: navivox.ServerStatus{
				GormesVersion: Version,
				ConfigVersion: fmt.Sprintf("%d", config.CurrentConfigVersion),
				Protocol:      navivox.ProtocolVersion,
				Features:      navivox.DefaultFeatures(),
				ActiveChannels: []string{
					navivox.PlatformName,
				},
			}}
			return navivox.NewServer(navivox.ServerOptions{Status: status}).Serve(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&stdio, "stdio", false, "serve Navivox frames over stdin/stdout")
	return cmd
}

type navivoxPairOptions struct {
	Host       string
	Port       int
	User       string
	DeviceName string
	Command    string
	PrintQR    bool
	JSONOut    bool
}

// navivoxPairReportJSON is the wire shape for `gormes navivox pair
// --json`. Fleet automation provisioning Navivox pairing across machines
// parses this to ingest the descriptor without scraping the multi-line
// "Host: / Port: / Pairing code:" prose. Build provenance leads — same
// convention as the rest of the `--json` arc. The pairing code is the
// data being conveyed (a one-time gateway-issued credential bounded by
// `expires_at`); long-lived secrets like SSH keys remain excluded.
type navivoxPairReportJSON struct {
	Build      buildProvenanceJSON `json:"build"`
	Host       string              `json:"host"`
	HostSource string              `json:"host_source,omitempty"`
	Port       int                 `json:"port"`
	User       string              `json:"user"`
	Command    string              `json:"command"`
	Protocol   uint32              `json:"protocol"`
	Code       string              `json:"code"`
	Device     string              `json:"device"`
	ExpiresAt  string              `json:"expires_at,omitempty"`
	URI        string              `json:"uri"`
}

func newNavivoxPairCommand() *cobra.Command {
	opts := navivoxPairOptions{
		Port:    22,
		Command: "gormes navivox serve --stdio",
		PrintQR: true,
	}
	cmd := &cobra.Command{
		Use:          "pair",
		Short:        "Print a Navivox pairing QR code for this host",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.DeviceName == "" {
				opts.DeviceName = defaultNavivoxDeviceName()
			}
			if opts.User == "" {
				opts.User = defaultNavivoxSSHUser()
			}
			host, source, err := resolveNavivoxPairHost(cmd.Context(), opts.Host)
			if err != nil {
				return err
			}
			opts.Host = host
			store := gateway.NewXDGPairingStore()
			result, err := store.GeneratePairingCode(cmd.Context(), gateway.PairingCodeRequest{
				Platform: navivox.PlatformName,
				UserID:   opts.DeviceName,
				UserName: opts.DeviceName,
			})
			if err != nil {
				return fmt.Errorf("navivox pairing code: %w", err)
			}
			if result.Status != gateway.PairingCodeIssued {
				return fmt.Errorf("navivox pairing code not issued: %s", result.Status)
			}
			uri := buildNavivoxPairingURI(navivoxPairingDescriptor{
				Host:      opts.Host,
				Port:      opts.Port,
				User:      opts.User,
				Command:   opts.Command,
				Protocol:  navivox.ProtocolVersion,
				Code:      result.Code,
				Device:    opts.DeviceName,
				ExpiresAt: result.ExpiresAt,
			})
			if opts.JSONOut {
				return writeNavivoxPairJSON(cmd, opts, source, result, uri)
			}
			renderNavivoxPairing(cmd, opts, source, result, uri)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.Host, "host", "", "host or IP to encode; defaults to Tailscale IPv4 when available, then LAN IPv4")
	cmd.Flags().IntVar(&opts.Port, "port", 22, "SSH port to encode")
	cmd.Flags().StringVar(&opts.User, "user", "", "SSH username to encode; defaults to the current user")
	cmd.Flags().StringVar(&opts.DeviceName, "device-name", "", "Navivox device name to pair; defaults to a generated local label")
	cmd.Flags().StringVar(&opts.Command, "command", opts.Command, "remote command the Navivox app should run over SSH")
	cmd.Flags().BoolVar(&opts.PrintQR, "qr", true, "print a terminal QR code")
	cmd.Flags().BoolVar(&opts.JSONOut, "json", false, "emit the pairing descriptor as machine-readable JSON")
	return cmd
}

func writeNavivoxPairJSON(cmd *cobra.Command, opts navivoxPairOptions, hostSource string, result gateway.PairingCodeResult, uri string) error {
	expires := ""
	if !result.ExpiresAt.IsZero() {
		expires = result.ExpiresAt.UTC().Format(time.RFC3339)
	}
	report := navivoxPairReportJSON{
		Build:      newBuildProvenance(),
		Host:       opts.Host,
		HostSource: hostSource,
		Port:       opts.Port,
		User:       opts.User,
		Command:    opts.Command,
		Protocol:   navivox.ProtocolVersion,
		Code:       result.Code,
		Device:     opts.DeviceName,
		ExpiresAt:  expires,
		URI:        uri,
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

type navivoxPairingDescriptor struct {
	Host      string
	Port      int
	User      string
	Command   string
	Protocol  uint32
	Code      string
	Device    string
	ExpiresAt time.Time
}

func buildNavivoxPairingURI(d navivoxPairingDescriptor) string {
	values := url.Values{}
	values.Set("transport", "ssh")
	values.Set("host", d.Host)
	values.Set("port", strconv.Itoa(d.Port))
	values.Set("user", d.User)
	values.Set("command", d.Command)
	values.Set("protocol", strconv.FormatUint(uint64(d.Protocol), 10))
	values.Set("code", d.Code)
	values.Set("device", d.Device)
	if !d.ExpiresAt.IsZero() {
		values.Set("expires_at", d.ExpiresAt.UTC().Format(time.RFC3339))
	}
	u := url.URL{Scheme: "navivox", Host: "pair", RawQuery: values.Encode()}
	return u.String()
}

func renderNavivoxPairing(cmd *cobra.Command, opts navivoxPairOptions, hostSource string, result gateway.PairingCodeResult, uri string) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Navivox pairing")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Scan this QR from the Navivox app.")
	fmt.Fprintln(out)
	if opts.PrintQR {
		qrterminal.GenerateHalfBlock(uri, qrterminal.M, out)
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "Server")
	fmt.Fprintf(out, "  Host: %s\n", opts.Host)
	if hostSource != "" {
		fmt.Fprintf(out, "  Source: %s\n", hostSource)
	}
	fmt.Fprintf(out, "  Port: %d\n", opts.Port)
	fmt.Fprintf(out, "  User: %s\n", opts.User)
	fmt.Fprintf(out, "  Command: %s\n", opts.Command)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Pairing code: %s\n", result.Code)
	if !result.ExpiresAt.IsZero() {
		fmt.Fprintf(out, "Expires: %s\n", result.ExpiresAt.UTC().Format(time.RFC3339))
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Fallback URI:")
	fmt.Fprintln(out, uri)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Recommended network:")
	fmt.Fprintln(out, "  Tailscale SSH is recommended so this host is reachable without opening public SSH.")
	fmt.Fprintln(out, "  To review host setup, run: gormes navivox setup-host --plan")
}

func newNavivoxSetupHostCommand() *cobra.Command {
	var plan bool
	var apply bool
	var yes bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "setup-host",
		Short:        "Show how to prepare this host for Navivox SSH pairing",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if apply {
				return runNavivoxSetupHostApply(cmd, navivoxHostSetupOptions{Yes: yes})
			}
			if asJSON {
				return writeNavivoxSetupHostPlanJSON(cmd)
			}
			renderNavivoxSetupHostPlan(cmd)
			return nil
		},
	}
	cmd.Flags().BoolVar(&plan, "plan", false, "render the setup plan without changing the host")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply the host setup steps after confirmation")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm setup-host --apply without an interactive confirmation prompt")
	cmd.Flags().BoolVar(&asJSON, "json", false, "with --plan, emit machine-readable JSON: {build, recommended, ssh_service, pair_command}")
	return cmd
}

// navivoxSetupHostPlanReportJSON is the wire shape for
// `gormes navivox setup-host --plan --json`. Fleet automation
// provisioning Navivox SSH hosts across machines parses this to
// inventory the recommended path and per-distro SSH install
// commands without scraping the multi-line preflight prose.
type navivoxSetupHostPlanReportJSON struct {
	Build       buildProvenanceJSON             `json:"build"`
	Recommended navivoxSetupHostRecommendedJSON `json:"recommended"`
	SSHService  map[string][]string             `json:"ssh_service"`
	SudoNote    string                          `json:"sudo_note"`
	PairCommand string                          `json:"pair_command"`
}

type navivoxSetupHostRecommendedJSON struct {
	Path           string `json:"path"`
	InstallCommand string `json:"install_command"`
	JoinCommand    string `json:"join_command"`
}

func writeNavivoxSetupHostPlanJSON(cmd *cobra.Command) error {
	report := navivoxSetupHostPlanReportJSON{
		Build: newBuildProvenance(),
		Recommended: navivoxSetupHostRecommendedJSON{
			Path:           "tailscale",
			InstallCommand: "curl -fsSL https://tailscale.com/install.sh | sh",
			JoinCommand:    "sudo tailscale up --ssh",
		},
		SSHService: map[string][]string{
			"debian": {
				"sudo apt-get update",
				"sudo apt-get install -y openssh-server",
				"sudo systemctl enable --now ssh",
			},
			"fedora": {
				"sudo dnf install -y openssh-server",
				"sudo systemctl enable --now sshd",
			},
		},
		SudoNote:    "sudo password is prompt-only and never stored in Gormes config; setup-host --apply asks once, previews exact steps, then discards it.",
		PairCommand: "gormes navivox pair",
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

func renderNavivoxSetupHostPlan(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Navivox host setup plan")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Goal: make this machine reachable from the Navivox app over SSH.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Recommended path")
	fmt.Fprintln(out, "  Tailscale is the recommended network path.")
	fmt.Fprintln(out, "  Install Tailscale when it is missing:")
	fmt.Fprintln(out, "    curl -fsSL https://tailscale.com/install.sh | sh")
	fmt.Fprintln(out, "  Join the tailnet and enable Tailscale SSH:")
	fmt.Fprintln(out, "    sudo tailscale up --ssh")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "SSH service")
	fmt.Fprintln(out, "  Ensure OpenSSH server is installed and running.")
	fmt.Fprintln(out, "  Debian/Ubuntu:")
	fmt.Fprintln(out, "    sudo apt-get update")
	fmt.Fprintln(out, "    sudo apt-get install -y openssh-server")
	fmt.Fprintln(out, "    sudo systemctl enable --now ssh")
	fmt.Fprintln(out, "  Fedora/RHEL:")
	fmt.Fprintln(out, "    sudo dnf install -y openssh-server")
	fmt.Fprintln(out, "    sudo systemctl enable --now sshd")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Sudo handling")
	fmt.Fprintln(out, "  sudo password is prompt-only and never stored in Gormes config.")
	fmt.Fprintln(out, "  setup-host --apply asks once, previews exact steps, then discards it.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "After setup")
	fmt.Fprintln(out, "  Run: gormes navivox pair")
}

type navivoxHostSetupOptions struct {
	Yes bool
}

type navivoxHostSetupSeams struct {
	GOOS             func() string
	LookPath         func(string) (string, error)
	ReadOSRelease    func() (map[string]string, error)
	Confirm          func(*navivoxHostSetupPlan) (bool, error)
	ReadSudoPassword func() (string, error)
	Run              func(context.Context, navivoxHostSetupCommand) error
}

func (s navivoxHostSetupSeams) withDefaults() navivoxHostSetupSeams {
	if s.GOOS == nil {
		s.GOOS = func() string { return runtime.GOOS }
	}
	if s.LookPath == nil {
		s.LookPath = exec.LookPath
	}
	if s.ReadOSRelease == nil {
		s.ReadOSRelease = readNavivoxOSRelease
	}
	if s.Confirm == nil {
		s.Confirm = confirmNavivoxHostSetup
	}
	if s.ReadSudoPassword == nil {
		s.ReadSudoPassword = readNavivoxSudoPassword
	}
	if s.Run == nil {
		s.Run = runNavivoxHostSetupCommand
	}
	return s
}

var navivoxHostSetup = (navivoxHostSetupSeams{}).withDefaults()

type navivoxHostSetupPlan struct {
	GOOS               string
	Distro             string
	PackageManager     string
	SSHService         string
	TailscaleInstalled bool
	UnsupportedReason  string
	Steps              []navivoxHostSetupStep
}

type navivoxHostSetupStep struct {
	Label   string
	Command navivoxHostSetupCommand
}

type navivoxHostSetupCommand struct {
	Name  string
	Args  []string
	Stdin string
}

func runNavivoxSetupHostApply(cmd *cobra.Command, opts navivoxHostSetupOptions) error {
	if !opts.Yes {
		if file, ok := cmd.InOrStdin().(*os.File); ok && !term.IsTerminal(int(file.Fd())) {
			fmt.Fprintln(cmd.ErrOrStderr(), "navivox_setup_requires_tty: run `gormes navivox setup-host --apply --yes` for non-interactive apply")
			return fmt.Errorf("navivox_setup_requires_tty")
		}
	}
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Navivox host setup")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Tailscale SSH is the recommended path.")
	fmt.Fprintln(out, "Gormes will use sudo only for this command and will not store the password.")
	fmt.Fprintln(out)

	plan := buildNavivoxHostSetupPlan()
	if plan.UnsupportedReason != "" {
		fmt.Fprintf(out, "Unsupported OS: %s\n", plan.GOOS)
		fmt.Fprintf(out, "Reason: %s\n", plan.UnsupportedReason)
		fmt.Fprintln(out, "No changes were made.")
		fmt.Fprintln(out, "Run: gormes navivox setup-host --plan")
		return newExitCodeError(2, fmt.Errorf("navivox setup-host unsupported: %s", plan.UnsupportedReason))
	}

	fmt.Fprintln(out, "Preflight")
	fmt.Fprintf(out, "  Linux distribution: %s\n", plan.Distro)
	fmt.Fprintf(out, "  Package manager: %s\n", plan.PackageManager)
	if plan.TailscaleInstalled {
		fmt.Fprintln(out, "  Tailscale already installed")
	} else {
		fmt.Fprintln(out, "  Tailscale missing; installer step will run")
	}
	fmt.Fprintf(out, "  SSH service: %s\n", plan.SSHService)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Planned changes")
	for i, step := range plan.Steps {
		fmt.Fprintf(out, "  %d. %s\n", i+1, step.Label)
		fmt.Fprintf(out, "     %s\n", navivoxHostSetupCommandString(step.Command))
	}
	fmt.Fprintln(out)

	if !opts.Yes {
		confirmed, err := navivoxHostSetup.Confirm(plan)
		if err != nil {
			return fmt.Errorf("confirm navivox host setup: %w", err)
		}
		if !confirmed {
			fmt.Fprintln(out, "No changes were made.")
			return nil
		}
	} else {
		fmt.Fprintln(out, "Confirmation: --yes")
	}

	password, err := navivoxHostSetup.ReadSudoPassword()
	if err != nil {
		return fmt.Errorf("read sudo password: %w", err)
	}
	defer func() {
		password = ""
	}()

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Applying changes")
	for i, step := range plan.Steps {
		fmt.Fprintf(out, "  %d/%d %s\n", i+1, len(plan.Steps), step.Label)
		c := step.Command
		c.Stdin = password + "\n"
		if err := navivoxHostSetup.Run(cmd.Context(), c); err != nil {
			return fmt.Errorf("navivox setup-host %s: %w", step.Label, err)
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Done")
	fmt.Fprintln(out, "Run: gormes navivox pair")
	return nil
}

func buildNavivoxHostSetupPlan() *navivoxHostSetupPlan {
	seams := navivoxHostSetup.withDefaults()
	plan := &navivoxHostSetupPlan{GOOS: seams.GOOS()}
	if plan.GOOS != "linux" {
		plan.UnsupportedReason = "automatic host setup is currently Linux-only"
		return plan
	}
	osRelease, _ := seams.ReadOSRelease()
	plan.Distro = navivoxOSReleaseValue(osRelease, "ID")
	if plan.Distro == "" {
		plan.Distro = "linux"
	}

	pm, sshService := detectNavivoxHostPackageManager(seams, osRelease)
	if pm == "" {
		plan.UnsupportedReason = "supported package manager not found; install openssh-server and Tailscale manually"
		return plan
	}
	if _, err := seams.LookPath("sudo"); err != nil {
		plan.UnsupportedReason = "sudo not found; install OpenSSH and Tailscale manually or rerun on a sudo-capable host"
		return plan
	}
	if _, err := seams.LookPath("systemctl"); err != nil {
		plan.UnsupportedReason = "systemctl not found; enable SSH and Tailscale manually for this init system"
		return plan
	}
	plan.PackageManager = pm
	plan.SSHService = sshService

	if pm == "apt-get" {
		plan.Steps = append(plan.Steps,
			navivoxHostSetupStep{Label: "Update package indexes", Command: sudoNavivoxHostCommand("apt-get", "update")},
			navivoxHostSetupStep{Label: "Install OpenSSH server", Command: sudoNavivoxHostCommand("apt-get", "install", "-y", "openssh-server")},
		)
	} else {
		plan.Steps = append(plan.Steps,
			navivoxHostSetupStep{Label: "Install OpenSSH server", Command: sudoNavivoxHostCommand(pm, "install", "-y", "openssh-server")},
		)
	}
	plan.Steps = append(plan.Steps,
		navivoxHostSetupStep{Label: "Enable SSH service", Command: sudoNavivoxHostCommand("systemctl", "enable", "--now", sshService)},
	)

	if _, err := seams.LookPath("tailscale"); err == nil {
		plan.TailscaleInstalled = true
	} else {
		plan.Steps = append(plan.Steps,
			navivoxHostSetupStep{Label: "Install Tailscale", Command: sudoNavivoxHostCommand("sh", "-c", "curl -fsSL https://tailscale.com/install.sh | sh")},
		)
	}
	plan.Steps = append(plan.Steps,
		navivoxHostSetupStep{Label: "Enable Tailscale SSH", Command: sudoNavivoxHostCommand("tailscale", "up", "--ssh")},
	)
	return plan
}

func detectNavivoxHostPackageManager(seams navivoxHostSetupSeams, osRelease map[string]string) (manager, sshService string) {
	id := strings.ToLower(navivoxOSReleaseValue(osRelease, "ID"))
	like := strings.ToLower(navivoxOSReleaseValue(osRelease, "ID_LIKE"))
	distroText := id + " " + like
	if strings.Contains(distroText, "debian") || strings.Contains(distroText, "ubuntu") {
		if _, err := seams.LookPath("apt-get"); err == nil {
			return "apt-get", "ssh"
		}
		return "", ""
	}
	if strings.Contains(distroText, "fedora") || strings.Contains(distroText, "rhel") || strings.Contains(distroText, "centos") {
		if _, err := seams.LookPath("dnf"); err == nil {
			return "dnf", "sshd"
		}
		return "", ""
	}
	if _, err := seams.LookPath("apt-get"); err == nil {
		return "apt-get", "ssh"
	}
	if _, err := seams.LookPath("dnf"); err == nil {
		return "dnf", "sshd"
	}
	return "", ""
}

func sudoNavivoxHostCommand(name string, args ...string) navivoxHostSetupCommand {
	out := navivoxHostSetupCommand{Name: "sudo", Args: []string{"-S", "--", name}}
	out.Args = append(out.Args, args...)
	return out
}

func navivoxHostSetupCommandString(c navivoxHostSetupCommand) string {
	parts := make([]string, 0, 1+len(c.Args))
	parts = append(parts, navivoxShellQuote(c.Name))
	for _, arg := range c.Args {
		parts = append(parts, navivoxShellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func navivoxShellQuote(value string) string {
	if value == "" {
		return "''"
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			strings.ContainsRune("_@%+=:,./-", r) {
			continue
		}
		return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
	}
	return value
}

func readNavivoxOSRelease() (map[string]string, error) {
	raw, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		out[strings.TrimSpace(key)] = value
	}
	return out, nil
}

func navivoxOSReleaseValue(values map[string]string, key string) string {
	if values == nil {
		return ""
	}
	return strings.TrimSpace(values[key])
}

func confirmNavivoxHostSetup(*navivoxHostSetupPlan) (bool, error) {
	fmt.Fprint(os.Stderr, "Apply Navivox host setup? Type yes to continue: ")
	text, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(text), "yes"), nil
}

func readNavivoxSudoPassword() (string, error) {
	fmt.Fprint(os.Stderr, "sudo password: ")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
	text, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(text, "\r\n"), nil
}

func runNavivoxHostSetupCommand(ctx context.Context, c navivoxHostSetupCommand) error {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	if c.Stdin != "" {
		cmd.Stdin = strings.NewReader(c.Stdin)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// navivoxPairHostSeams hides the host-detection probes behind a small
// surface so tests can drive the Tailscale → LAN → loopback fallback
// chain deterministically. The default seam delegates to the real
// network/process probes.
type navivoxPairHostSeams struct {
	LookupTailscaleIPv4   func(context.Context) string
	LookupNonLoopbackIPv4 func() string
}

func (s navivoxPairHostSeams) withDefaults() navivoxPairHostSeams {
	if s.LookupTailscaleIPv4 == nil {
		s.LookupTailscaleIPv4 = navivoxTailscaleIPv4
	}
	if s.LookupNonLoopbackIPv4 == nil {
		s.LookupNonLoopbackIPv4 = firstNonLoopbackIPv4
	}
	return s
}

var navivoxPairHostResolver = (navivoxPairHostSeams{}).withDefaults()

func resolveNavivoxPairHost(ctx context.Context, explicit string) (host string, source string, err error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), "manual", nil
	}
	seams := navivoxPairHostResolver.withDefaults()
	if ip := seams.LookupTailscaleIPv4(ctx); ip != "" {
		return ip, "tailscale", nil
	}
	if ip := seams.LookupNonLoopbackIPv4(); ip != "" {
		return ip, "lan", nil
	}
	return "127.0.0.1", "loopback", nil
}

func navivoxTailscaleIPv4(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tailscale", "ip", "-4").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if ip := net.ParseIP(line); ip != nil && ip.To4() != nil {
			return line
		}
	}
	return ""
}

func firstNonLoopbackIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if v4 := ip.To4(); v4 != nil {
				return v4.String()
			}
		}
	}
	return ""
}

func defaultNavivoxSSHUser() string {
	if u, err := user.Current(); err == nil && strings.TrimSpace(u.Username) != "" {
		return shortLocalUsername(u.Username)
	}
	if env := strings.TrimSpace(os.Getenv("USER")); env != "" {
		return env
	}
	return "gormes"
}

func shortLocalUsername(username string) string {
	if idx := strings.LastIndexAny(username, `\`); idx >= 0 && idx+1 < len(username) {
		return username[idx+1:]
	}
	return username
}

func defaultNavivoxDeviceName() string {
	host, _ := os.Hostname()
	host = sanitizeNavivoxLabel(host)
	if host == "" {
		host = "device"
	}
	return host + "-" + randomNavivoxSuffix(6)
}

func sanitizeNavivoxLabel(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return ""
	}
	if len(out) > 32 {
		return out[:32]
	}
	return out
}

func randomNavivoxSuffix(n int) string {
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}
