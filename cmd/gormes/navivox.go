package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func newNavivoxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "navivox",
		Short: "Run the Navivox SSH stdio channel",
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
	return cmd
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
	cmd := &cobra.Command{
		Use:          "setup-host",
		Short:        "Show how to prepare this host for Navivox SSH pairing",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			renderNavivoxSetupHostPlan(cmd)
			return nil
		},
	}
	cmd.Flags().BoolVar(&plan, "plan", false, "render the setup plan without changing the host")
	return cmd
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
	fmt.Fprintln(out, "  Future --apply setup will ask once, run explicit steps, then discard it.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "After setup")
	fmt.Fprintln(out, "  Run: gormes navivox pair")
}

func resolveNavivoxPairHost(ctx context.Context, explicit string) (host string, source string, err error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), "manual", nil
	}
	if ip := navivoxTailscaleIPv4(ctx); ip != "" {
		return ip, "tailscale", nil
	}
	if ip := firstNonLoopbackIPv4(); ip != "" {
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
