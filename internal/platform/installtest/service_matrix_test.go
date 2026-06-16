package installtest

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall_DryRunServiceMatrixMacOSInstallsLaunchAgent(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"UNAME":                         "Darwin",
		"GORMES_INSTALL_HOME":           filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":             "1",
		"GORMES_RESTART_GATEWAY":        "never",
		"GORMES_INSTALL_TEST_LAUNCHCTL": "1",
	})
	if !strings.Contains(out, "service    install LaunchAgent") {
		t.Fatalf("macOS dry-run missing LaunchAgent service plan:\n%s", out)
	}
	if strings.Contains(out, "service    install systemd") {
		t.Fatalf("macOS dry-run must not advertise systemd:\n%s", out)
	}
}

func TestInstall_DryRunServiceMatrixWSLWithoutSystemdUsesManualGateway(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":         filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":           "1",
		"GORMES_RESTART_GATEWAY":      "never",
		"GORMES_INSTALL_TEST_WSL":     "1",
		"GORMES_INSTALL_TEST_SYSTEMD": "0",
	})
	for _, want := range []string{"service    manual (WSL without systemd: tmux or gormes gateway)", "gateway    restart=never"} {
		if !strings.Contains(out, want) {
			t.Fatalf("WSL no-systemd dry-run missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "profiles   restart active gormes-gateway-*.service") {
		t.Fatalf("WSL no-systemd plan must not advertise systemd profile restarts:\n%s", out)
	}
}

func TestInstall_DryRunServiceMatrixWSLWithSystemdInstallsUserService(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":           filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":             "1",
		"GORMES_RESTART_GATEWAY":        "never",
		"GORMES_INSTALL_TEST_WSL":       "1",
		"GORMES_INSTALL_TEST_CONTAINER": "0",
		"GORMES_INSTALL_TEST_SYSTEMD":   "1",
	})
	for _, want := range []string{"service    install systemd user service (WSL systemd; not auto-enabled until setup completes)", "profiles   restart active gormes-gateway-*.service units (never)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("WSL systemd dry-run missing %q:\n%s", want, out)
		}
	}
}

func TestInstall_DryRunServiceMatrixContainerRecommendsContainerPolicy(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":           filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":             "1",
		"GORMES_RESTART_GATEWAY":        "never",
		"GORMES_INSTALL_TEST_WSL":       "0",
		"GORMES_INSTALL_TEST_CONTAINER": "1",
		"GORMES_INSTALL_TEST_SYSTEMD":   "1",
	})
	if !strings.Contains(out, "service    container (Docker restart policy; image-owned s6 if available)") {
		t.Fatalf("container dry-run missing container service recommendation:\n%s", out)
	}
	if strings.Contains(out, "service    install systemd") {
		t.Fatalf("container dry-run must not pretend systemd is the right service manager:\n%s", out)
	}
}

func TestInstall_DryRunServiceMatrixFallbackLinuxDoesNotPretendSystemdWorks(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":           filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":             "1",
		"GORMES_RESTART_GATEWAY":        "never",
		"GORMES_INSTALL_TEST_WSL":       "0",
		"GORMES_INSTALL_TEST_CONTAINER": "0",
		"GORMES_INSTALL_TEST_SYSTEMD":   "0",
	})
	if !strings.Contains(out, "service    manual (nohup/tmux/supervisord; systemd user unavailable)") {
		t.Fatalf("fallback Linux dry-run missing manual service recommendation:\n%s", out)
	}
	if strings.Contains(out, "service    install systemd") {
		t.Fatalf("fallback Linux dry-run must not pretend systemd works:\n%s", out)
	}
}

func TestInstall_ServiceMatrixManualInstructionsPrintConcreteExamples(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want []string
	}{
		{
			name: "WSL without systemd",
			env: map[string]string{
				"GORMES_INSTALL_TEST_WSL":       "1",
				"GORMES_INSTALL_TEST_CONTAINER": "0",
				"GORMES_INSTALL_TEST_SYSTEMD":   "0",
			},
			want: []string{"manual    WSL gateway", "tmux new -s gormes-gateway 'gormes gateway'", "Enable WSL systemd"},
		},
		{
			name: "Termux",
			env: map[string]string{
				"TERMUX_VERSION": "1",
			},
			want: []string{"manual    Termux gateway", "tmux new -s gormes-gateway 'termux-wake-lock; gormes gateway'", "termux-wake-unlock"},
		},
		{
			name: "Container",
			env: map[string]string{
				"GORMES_INSTALL_TEST_CONTAINER": "1",
				"GORMES_INSTALL_TEST_SYSTEMD":   "1",
			},
			want: []string{"manual    Container gateway", "docker run --restart unless-stopped", "image-owned s6 services"},
		},
		{
			name: "fallback Linux",
			env: map[string]string{
				"GORMES_INSTALL_TEST_WSL":       "0",
				"GORMES_INSTALL_TEST_CONTAINER": "0",
				"GORMES_INSTALL_TEST_SYSTEMD":   "0",
			},
			want: []string{"manual    Gateway autostart", "nohup gormes gateway", "supervisord example: command=gormes gateway"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := runInstallShellFunction(t, tc.env, "print_service_instructions")
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Fatalf("service instructions missing %q:\n%s", want, out)
				}
			}
		})
	}
}

func runInstallShellFunction(t *testing.T, env map[string]string, function string) string {
	t.Helper()
	root := repoRoot(t)
	home := filepath.Join(t.TempDir(), "home")
	cmd := exec.Command("sh", "-c", ". "+shellQuote(filepath.Join(root, "install.sh"))+"; "+function)
	cmd.Dir = root
	cmd.Env = []string{
		"HOME=" + home,
		"PATH=/usr/bin:/bin",
		"SHELL=/bin/bash",
		"GORMES_INSTALL_TEST_MODE=1",
		"GORMES_INSTALL_HOME=" + home,
		"GORMES_SKIP_SETUP=1",
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\noutput:\n%s", function, err, string(out))
	}
	return string(out)
}
