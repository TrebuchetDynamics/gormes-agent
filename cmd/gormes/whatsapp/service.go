package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type Options struct {
	BuildProvenance func() BuildProvenance
}

type commandOptions struct {
	Mode         string
	AllowedUsers string
	AllowAll     bool
	Debug        bool
	PlanOnly     bool
	JSONOut      bool
	BridgeScript string
}

type planReportJSON struct {
	Build         BuildProvenance `json:"build"`
	Mode          string          `json:"mode"`
	ConfigPath    string          `json:"config_path"`
	EnvPath       string          `json:"env_path"`
	HomePath      string          `json:"home_path"`
	BridgeDir     string          `json:"bridge_dir"`
	SessionDir    string          `json:"session_dir"`
	BridgeLog     string          `json:"bridge_log"`
	AllowedUsers  string          `json:"allowed_users"`
	Debug         bool            `json:"debug"`
	AllowAllUsers bool            `json:"allow_all_users"`
	BridgeCommand []string        `json:"bridge_command"`
	PairCommand   []string        `json:"pair_command"`
}

type exitCodeError struct {
	code int
	err  error
}

func newExitCodeError(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("exit code %d", code)
	}
	return exitCodeError{code: code, err: err}
}

func (e exitCodeError) Error() string { return e.err.Error() }
func (e exitCodeError) Unwrap() error { return e.err }
func (e exitCodeError) ExitCode() int { return e.code }

func NewCommand(opts Options) *cobra.Command {
	return NewCommandWithSeams(Seams{}, opts)
}

type Seams struct {
	InstallBridgeDependencies func(context.Context, string, io.Writer) error
	RunBridgePairing          func(context.Context, PairingPlan, io.Writer) error
}

type PairingPlan struct {
	Runtime       channelsmodule.WhatsAppRuntimePlan
	ConfigPath    string
	EnvPath       string
	HomePath      string
	BridgeDir     string
	PairCommand   []string
	Dotenv        map[string]string
	AllowedUsers  string
	AllowAllUsers bool
	Debug         bool
}

func NewCommandWithSeams(seams Seams, opts Options) *cobra.Command {
	seams = seams.withDefaults()
	if opts.BuildProvenance == nil {
		opts.BuildProvenance = func() BuildProvenance { return BuildProvenance{} }
	}
	return channelsmodule.NewWhatsAppCommandWithSeams(channelsmodule.WhatsAppCommandSeams{
		Run: func(cmd *cobra.Command, moduleOpts channelsmodule.WhatsAppOptions) error {
			return runCommand(cmd, commandOptionsFromModule(moduleOpts), seams, opts)
		},
	})
}

func commandOptionsFromModule(opts channelsmodule.WhatsAppOptions) commandOptions {
	return commandOptions{
		Mode:         opts.Mode,
		AllowedUsers: opts.AllowedUsers,
		AllowAll:     opts.AllowAll,
		Debug:        opts.Debug,
		PlanOnly:     opts.PlanOnly,
		JSONOut:      opts.JSONOut,
		BridgeScript: opts.BridgeScript,
	}
}

func (seams Seams) withDefaults() Seams {
	if seams.InstallBridgeDependencies == nil {
		seams.InstallBridgeDependencies = defaultInstallBridgeDependencies
	}
	if seams.RunBridgePairing == nil {
		seams.RunBridgePairing = defaultRunBridgePairing
	}
	return seams
}

func runCommand(cmd *cobra.Command, opts commandOptions, seams Seams, serviceOpts Options) error {
	pairingPlan, err := BuildPairingPlan(opts)
	if err != nil {
		return newExitCodeError(2, err)
	}

	out := cmd.OutOrStdout()
	if opts.PlanOnly {
		if opts.JSONOut {
			return writePlanJSON(out, pairingPlan, serviceOpts.BuildProvenance())
		}
		renderPairingPreflight(out, pairingPlan)
		return nil
	}

	if err := renderPairingWizardIntro(cmd, out, &pairingPlan); err != nil {
		return err
	}
	if err := persistPairingEnv(pairingPlan); err != nil {
		return err
	}
	if err := seams.InstallBridgeDependencies(cmd.Context(), pairingPlan.BridgeDir, out); err != nil {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "✗ Bridge dependencies unavailable: %v\n", err)
		fmt.Fprintln(out, "  Run with --plan to inspect the expected bridge command and config paths.")
		return err
	}

	renderPairingScanInstructions(out, pairingPlan)
	if err := seams.RunBridgePairing(cmd.Context(), pairingPlan, out); err != nil {
		return err
	}

	renderPairingSuccess(out, pairingPlan)
	return nil
}

func BuildPairingPlan(opts commandOptions) (PairingPlan, error) {
	mode, err := normalizeCommandMode(opts.Mode)
	if err != nil {
		return PairingPlan{}, err
	}
	plan, err := channelsmodule.DecideWhatsAppRuntime(channelsmodule.WhatsAppRuntimeConfig{
		StateRoot:   config.GormesHome(),
		AccountMode: mode,
		Bridge: channelsmodule.WhatsAppBridgeRuntimeConfig{
			ScriptPath: opts.BridgeScript,
		},
	})
	if err != nil {
		return PairingPlan{}, err
	}

	dotenv := readDotenvValues(config.EnvPath())
	allowedUsers := resolveAllowedUsers(opts, dotenv)
	bridgeDir := filepath.Dir(plan.Bridge.ScriptPath)
	if abs, err := filepath.Abs(bridgeDir); err == nil {
		bridgeDir = abs
	}
	pairScript := plan.Bridge.ScriptPath
	if abs, err := filepath.Abs(pairScript); err == nil {
		pairScript = abs
	}

	return PairingPlan{
		Runtime:       plan,
		ConfigPath:    config.ConfigPath(),
		EnvPath:       config.EnvPath(),
		HomePath:      config.GormesHome(),
		BridgeDir:     bridgeDir,
		PairCommand:   []string{"node", pairScript, "--pair-only", "--session", plan.Session.Path},
		Dotenv:        dotenv,
		AllowedUsers:  allowedUsers,
		AllowAllUsers: opts.AllowAll,
		Debug:         opts.Debug,
	}, nil
}

func writePlanJSON(out io.Writer, plan PairingPlan, build BuildProvenance) error {
	report := planReportJSON{
		Build:         build,
		Mode:          string(plan.Runtime.Account.Mode),
		ConfigPath:    plan.ConfigPath,
		EnvPath:       plan.EnvPath,
		HomePath:      plan.HomePath,
		BridgeDir:     plan.BridgeDir,
		SessionDir:    plan.Runtime.Session.Path,
		BridgeLog:     plan.Runtime.Bridge.LogPath,
		AllowedUsers:  plan.AllowedUsers,
		Debug:         plan.Debug,
		AllowAllUsers: plan.AllowAllUsers,
		BridgeCommand: plan.Runtime.Bridge.Command,
		PairCommand:   plan.PairCommand,
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(body))
	return err
}

func renderPairingPreflight(out io.Writer, plan PairingPlan) {
	fmt.Fprintln(out, "WhatsApp pairing setup")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Gormes uses a Hermes-compatible Baileys bridge, which acts like WhatsApp Web.")
	fmt.Fprintln(out, "It is not the official Meta Business API.")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Mode: %s\n", plan.Runtime.Account.Mode)
	fmt.Fprintf(out, "Config file:  %s\n", plan.ConfigPath)
	fmt.Fprintf(out, "Secrets file: %s\n", plan.EnvPath)
	fmt.Fprintf(out, "Session dir:  %s\n", plan.Runtime.Session.Path)
	fmt.Fprintf(out, "Bridge log:   %s\n", plan.Runtime.Bridge.LogPath)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Environment to add to the secrets file:")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "WHATSAPP_ENABLED=true")
	fmt.Fprintf(out, "WHATSAPP_MODE=%s\n", plan.Runtime.Account.Mode)
	fmt.Fprintf(out, "WHATSAPP_ALLOWED_USERS=%s\n", plan.AllowedUsers)
	if plan.Debug {
		fmt.Fprintln(out, "WHATSAPP_DEBUG=true")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Bridge command plan:")
	fmt.Fprintf(out, "%s\n", strings.Join(plan.Runtime.Bridge.Command, " "))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "QR pairing steps:")
	fmt.Fprintln(out, "1. Use a dedicated WhatsApp number for bot mode when possible.")
	fmt.Fprintln(out, "2. Open WhatsApp > Settings > Linked Devices > Link a Device.")
	fmt.Fprintln(out, "3. Scan the terminal QR code once live bridge pairing is available.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "After pairing, restart the gateway through your service manager, then run:")
	fmt.Fprintln(out, "gormes gateway status")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "To inspect WhatsApp channel capability metadata without starting the gateway:")
	fmt.Fprintln(out, "gormes channels --channel whatsapp")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Security: the session directory is a login credential; keep it private and mode 0700.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Run without --plan to start the live QR pairing wizard.")
}

func renderPairingWizardIntro(cmd *cobra.Command, out io.Writer, plan *PairingPlan) error {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "⚕ WhatsApp Setup")
	fmt.Fprintln(out, strings.Repeat("=", 50))
	fmt.Fprintln(out)
	fmt.Fprintf(out, "✓ Mode: %s\n", modeLabel(plan.Runtime.Account.Mode))
	fmt.Fprintln(out)

	if strings.EqualFold(plan.Dotenv["WHATSAPP_ENABLED"], "true") {
		fmt.Fprintln(out, "✓ WhatsApp is already enabled")
	} else {
		fmt.Fprintln(out, "✓ WhatsApp enabled")
	}

	if plan.AllowedUsers != "" {
		fmt.Fprintf(out, "✓ Allowed users: %s\n", plan.AllowedUsers)
		if strings.TrimSpace(plan.Dotenv["WHATSAPP_ALLOWED_USERS"]) != "" && !plan.AllowAllUsers {
			updated, err := maybePromptAllowedUsers(cmd, plan.AllowedUsers)
			if err != nil {
				return err
			}
			if updated != "" && updated != plan.AllowedUsers {
				plan.AllowedUsers = updated
				fmt.Fprintf(out, "  ✓ Updated to: %s\n", updated)
			}
		}
		return nil
	}

	if plan.Runtime.Account.Mode == channelsmodule.WhatsAppAccountModeBot {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Who should be allowed to message the bot?")
		fmt.Fprintln(out, "  Phone numbers must include country code and no +, spaces, or dashes.")
	} else {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  Add your own phone number to WHATSAPP_ALLOWED_USERS for self-chat mode.")
	}
	fmt.Fprintln(out, "  ⚠ No allowlist configured — use --allowed-users or edit the secrets file.")
	return nil
}

func maybePromptAllowedUsers(cmd *cobra.Command, current string) (string, error) {
	out := cmd.OutOrStdout()
	fmt.Fprint(out, "\n  Update allowed users? [y/N] ")
	if !commandInputIsTTY(cmd) {
		fmt.Fprintln(out, "n")
		return current, nil
	}
	var response string
	if _, err := fmt.Fscanln(cmd.InOrStdin(), &response); err != nil {
		return current, nil
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return current, nil
	}
	updated, err := promptString(cmd, "  Phone numbers that can message the bot (comma-separated): ", current)
	if err != nil {
		return "", err
	}
	updated = strings.ReplaceAll(strings.TrimSpace(updated), " ", "")
	if updated == "" {
		return current, nil
	}
	return updated, nil
}

func commandInputIsTTY(cmd *cobra.Command) bool {
	file, ok := cmd.InOrStdin().(*os.File)
	if !ok || file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func promptString(cmd *cobra.Command, prompt, defaultVal string) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	var input string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &input)
	if err != nil {
		if err.Error() == "unexpected newline" || strings.Contains(err.Error(), "expected") {
			return defaultVal, nil
		}
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func renderPairingScanInstructions(out io.Writer, plan PairingPlan) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, strings.Repeat("─", 50))
	if plan.Runtime.Account.Mode == channelsmodule.WhatsAppAccountModeBot {
		fmt.Fprintln(out, "📱 Open WhatsApp (or WhatsApp Business) on the")
		fmt.Fprintln(out, "   phone with the BOT's number, then scan:")
	} else {
		fmt.Fprintln(out, "📱 Open WhatsApp on your phone, then scan:")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "   Settings → Linked Devices → Link a Device")
	fmt.Fprintln(out, strings.Repeat("─", 50))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "📱 WhatsApp pairing mode")
	fmt.Fprintf(out, "📁 Session: %s\n", plan.Runtime.Session.Path)
}

func renderPairingSuccess(out io.Writer, plan PairingPlan) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "✓ WhatsApp paired successfully!")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Next steps:")
	fmt.Fprintln(out, "    1. Start the gateway:  gormes gateway")
	if plan.Runtime.Account.Mode == channelsmodule.WhatsAppAccountModeBot {
		fmt.Fprintln(out, "    2. Send a message to the bot's WhatsApp number")
		fmt.Fprintln(out, "    3. The agent will reply automatically")
	} else {
		fmt.Fprintln(out, "    2. Open WhatsApp → Message Yourself")
		fmt.Fprintln(out, "    3. Type a message — the agent will reply")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Tip: Agent responses are prefixed with '⚕ Gormes Agent'")
	if plan.Runtime.Account.Mode == channelsmodule.WhatsAppAccountModeSelfChat {
		fmt.Fprintln(out, "  so you can tell them apart from your own messages.")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Check status: gormes gateway status")
}

func defaultInstallBridgeDependencies(ctx context.Context, bridgeDir string, out io.Writer) error {
	bridgeScript := filepath.Join(bridgeDir, "bridge.js")
	if _, err := os.Stat(bridgeScript); err != nil {
		return newExitCodeError(2, fmt.Errorf("whatsapp: bridge script not found at %s", bridgeScript))
	}
	if _, err := os.Stat(filepath.Join(bridgeDir, "node_modules")); err == nil {
		fmt.Fprintln(out, "✓ Bridge dependencies already installed")
		return nil
	}
	npm, err := exec.LookPath("npm")
	if err != nil {
		return newExitCodeError(2, fmt.Errorf("whatsapp: npm not found on PATH; install Node.js first"))
	}
	fmt.Fprintln(out, "→ Installing WhatsApp bridge dependencies (this can take a few minutes)...")
	var stderr bytes.Buffer
	install := exec.CommandContext(ctx, npm, "install", "--no-fund", "--no-audit", "--progress=false")
	install.Dir = bridgeDir
	install.Stdout = io.Discard
	install.Stderr = &stderr
	if err := install.Run(); err != nil {
		preview := strings.TrimSpace(stderr.String())
		if preview == "" {
			preview = "(no output)"
		}
		return fmt.Errorf("whatsapp: npm install failed:\n%s", tailLines(preview, 30))
	}
	fmt.Fprintln(out, "  ✓ Dependencies installed")
	return nil
}

func defaultRunBridgePairing(ctx context.Context, plan PairingPlan, out io.Writer) error {
	if err := os.MkdirAll(plan.Runtime.Session.Path, 0o700); err != nil {
		return fmt.Errorf("whatsapp: create session directory: %w", err)
	}
	cmd := exec.CommandContext(ctx, plan.PairCommand[0], plan.PairCommand[1:]...)
	cmd.Dir = plan.BridgeDir
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("whatsapp: pairing bridge failed: %w", err)
	}
	credentialsPath := filepath.Join(plan.Runtime.Session.Path, "creds.json")
	if _, err := os.Stat(credentialsPath); err != nil {
		return newExitCodeError(1, fmt.Errorf("whatsapp: pairing may not have completed; credentials not found at %s", credentialsPath))
	}
	return nil
}

func resolveAllowedUsers(opts commandOptions, dotenv map[string]string) string {
	if opts.AllowAll {
		return "*"
	}
	if users := strings.TrimSpace(opts.AllowedUsers); users != "" {
		return strings.ReplaceAll(users, " ", "")
	}
	return strings.TrimSpace(dotenv["WHATSAPP_ALLOWED_USERS"])
}

func persistPairingEnv(plan PairingPlan) error {
	updates := map[string]string{
		"WHATSAPP_ENABLED": "true",
		"WHATSAPP_MODE":    string(plan.Runtime.Account.Mode),
	}
	if plan.AllowedUsers != "" {
		updates["WHATSAPP_ALLOWED_USERS"] = plan.AllowedUsers
	}
	if plan.Debug {
		updates["WHATSAPP_DEBUG"] = "true"
	}
	if err := writeDotenvValues(plan.EnvPath, updates); err != nil {
		return fmt.Errorf("whatsapp: update secrets file: %w", err)
	}
	return nil
}

func readDotenvValues(path string) map[string]string {
	values := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return values
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values
}

func writeDotenvValues(path string, updates map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	var lines []string
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		body := strings.TrimRight(string(data), "\n")
		if body != "" {
			lines = strings.Split(body, "\n")
		}
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(lines)+len(updates))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}
		key, _, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			out = append(out, line)
			continue
		}
		if value, ok := updates[key]; ok {
			out = append(out, key+"="+value)
			seen[key] = true
			continue
		}
		out = append(out, line)
	}

	for _, key := range []string{"WHATSAPP_ENABLED", "WHATSAPP_MODE", "WHATSAPP_ALLOWED_USERS", "WHATSAPP_DEBUG"} {
		value, ok := updates[key]
		if !ok || seen[key] {
			continue
		}
		out = append(out, key+"="+value)
	}

	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o600)
}

func modeLabel(mode channelsmodule.WhatsAppAccountMode) string {
	switch mode {
	case channelsmodule.WhatsAppAccountModeBot:
		return "separate bot number"
	case channelsmodule.WhatsAppAccountModeSelfChat:
		return "personal number (self-chat)"
	default:
		return string(mode)
	}
}

func tailLines(text string, maxLines int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[len(lines)-maxLines:], "\n")
}

func normalizeCommandMode(mode string) (channelsmodule.WhatsAppAccountMode, error) {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	switch channelsmodule.WhatsAppAccountMode(normalized) {
	case channelsmodule.WhatsAppAccountModeBot:
		return channelsmodule.WhatsAppAccountModeBot, nil
	case channelsmodule.WhatsAppAccountModeSelfChat:
		return channelsmodule.WhatsAppAccountModeSelfChat, nil
	default:
		return "", fmt.Errorf("whatsapp: unsupported account mode %q", mode)
	}
}
