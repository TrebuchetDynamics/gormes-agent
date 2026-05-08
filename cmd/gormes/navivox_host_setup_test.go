package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestNavivoxSetupHostPlan_JSONEmitsStructuredPlan proves
// `gormes navivox setup-host --plan --json` emits a parseable
// `{build, recommended_path, ssh_service: {debian, fedora}, sudo_note,
// after_setup}` document so fleet automation provisioning Navivox SSH
// hosts across machines can ingest the plan with binary attribution.
// Build provenance leads — same convention as the rest of the
// `--json` arc. The default text output remains unchanged.
func TestNavivoxSetupHostPlan_JSONEmitsStructuredPlan(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "navivox", "setup-host", "--plan", "--json")
	if err != nil {
		t.Fatalf("navivox setup-host --plan --json: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Recommended struct {
			Path           string   `json:"path"`
			InstallCommand string   `json:"install_command"`
			JoinCommand    string   `json:"join_command"`
		} `json:"recommended"`
		SSHService map[string][]string `json:"ssh_service"`
		PairCommand string              `json:"pair_command"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Recommended.Path != "tailscale" {
		t.Errorf("recommended.path = %q, want tailscale", got.Recommended.Path)
	}
	if got.PairCommand != "gormes navivox pair" {
		t.Errorf("pair_command = %q, want %q", got.PairCommand, "gormes navivox pair")
	}
	if _, ok := got.SSHService["debian"]; !ok {
		t.Errorf("ssh_service missing debian key: %+v", got.SSHService)
	}
	if _, ok := got.SSHService["fedora"]; !ok {
		t.Errorf("ssh_service missing fedora key: %+v", got.SSHService)
	}
}

func TestNavivoxSetupHostApplyDebianUsesTransientSudoAndClearUX(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	secret := "sudo-secret-never-print"
	var ran []navivoxHostSetupCommand
	restore := withNavivoxHostSetupTestSeams(t, navivoxHostSetupSeams{
		GOOS: func() string { return "linux" },
		LookPath: func(name string) (string, error) {
			switch name {
			case "apt-get", "systemctl", "sudo", "sh":
				return "/usr/bin/" + name, nil
			case "tailscale":
				return "", errNavivoxHostSetupCommandMissing
			default:
				return "", errNavivoxHostSetupCommandMissing
			}
		},
		ReadOSRelease: func() (map[string]string, error) {
			return map[string]string{"ID": "ubuntu", "ID_LIKE": "debian"}, nil
		},
		Confirm: func(_ *navivoxHostSetupPlan) (bool, error) { return true, nil },
		ReadSudoPassword: func() (string, error) {
			return secret, nil
		},
		Run: func(_ context.Context, c navivoxHostSetupCommand) error {
			ran = append(ran, c)
			return nil
		},
	})
	defer restore()

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "navivox", "setup-host", "--apply", "--yes")
	if err != nil {
		t.Fatalf("navivox setup-host --apply --yes: %v\nstderr=%s", err, stderr)
	}
	for _, want := range []string{
		"Navivox host setup",
		"Tailscale SSH is the recommended path.",
		"Planned changes",
		"sudo -S -- apt-get install -y openssh-server",
		"sudo -S -- sh -c 'curl -fsSL https://tailscale.com/install.sh | sh'",
		"Applying changes",
		"Install OpenSSH server",
		"Enable SSH service",
		"Install Tailscale",
		"Enable Tailscale SSH",
		"Run: gormes navivox pair",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("setup-host apply output missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, secret) || strings.Contains(stderr, secret) {
		t.Fatalf("sudo password leaked\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}

	got := navivoxHostCommandNames(ran)
	want := []string{
		"sudo -S -- apt-get update",
		"sudo -S -- apt-get install -y openssh-server",
		"sudo -S -- systemctl enable --now ssh",
		"sudo -S -- sh -c curl -fsSL https://tailscale.com/install.sh | sh",
		"sudo -S -- tailscale up --ssh",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %#v, want %#v", got, want)
	}
	for _, c := range ran {
		if strings.Join(c.Args, " ") == "" {
			t.Fatalf("empty command recorded: %+v", c)
		}
		if strings.Contains(c.Name+" "+strings.Join(c.Args, " "), secret) {
			t.Fatalf("sudo password leaked into command args: %+v", c)
		}
		if c.Stdin != secret+"\n" {
			t.Fatalf("command stdin = %q, want transient sudo password newline", c.Stdin)
		}
	}
}

func TestNavivoxSetupHostApplyFedoraUsesDnfAndSSHD(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	var ran []navivoxHostSetupCommand
	restore := withNavivoxHostSetupTestSeams(t, navivoxHostSetupSeams{
		GOOS: func() string { return "linux" },
		LookPath: func(name string) (string, error) {
			switch name {
			case "dnf", "systemctl", "sudo", "tailscale":
				return "/usr/bin/" + name, nil
			default:
				return "", errNavivoxHostSetupCommandMissing
			}
		},
		ReadOSRelease: func() (map[string]string, error) {
			return map[string]string{"ID": "fedora", "ID_LIKE": "rhel fedora"}, nil
		},
		Confirm:          func(_ *navivoxHostSetupPlan) (bool, error) { return true, nil },
		ReadSudoPassword: func() (string, error) { return "pw", nil },
		Run: func(_ context.Context, c navivoxHostSetupCommand) error {
			ran = append(ran, c)
			return nil
		},
	})
	defer restore()

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "navivox", "setup-host", "--apply", "--yes")
	if err != nil {
		t.Fatalf("navivox setup-host --apply --yes: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Tailscale already installed") {
		t.Fatalf("setup-host output should report existing Tailscale:\n%s", stdout)
	}

	got := navivoxHostCommandNames(ran)
	want := []string{
		"sudo -S -- dnf install -y openssh-server",
		"sudo -S -- systemctl enable --now sshd",
		"sudo -S -- tailscale up --ssh",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %#v, want %#v", got, want)
	}
}

func TestNavivoxSetupHostApplyUnsupportedOSIsActionableAndNonMutating(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	ran := false
	restore := withNavivoxHostSetupTestSeams(t, navivoxHostSetupSeams{
		GOOS: func() string { return "darwin" },
		Run: func(context.Context, navivoxHostSetupCommand) error {
			ran = true
			return nil
		},
	})
	defer restore()

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "navivox", "setup-host", "--apply", "--yes")
	if err == nil {
		t.Fatalf("navivox setup-host --apply on darwin succeeded; stdout=%s stderr=%s", stdout, stderr)
	}
	for _, want := range []string{
		"Navivox host setup",
		"Unsupported OS: darwin",
		"No changes were made.",
		"Run: gormes navivox setup-host --plan",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("unsupported OS output missing %q:\n%s", want, stdout)
		}
	}
	if ran {
		t.Fatal("unsupported OS should not execute host setup commands")
	}
}

func TestNavivoxSetupHostApplyMissingPackageManagerIsNonMutating(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	ran := false
	restore := withNavivoxHostSetupTestSeams(t, navivoxHostSetupSeams{
		GOOS: func() string { return "linux" },
		LookPath: func(name string) (string, error) {
			switch name {
			case "sudo", "systemctl":
				return "/usr/bin/" + name, nil
			default:
				return "", errNavivoxHostSetupCommandMissing
			}
		},
		ReadOSRelease: func() (map[string]string, error) {
			return map[string]string{"ID": "ubuntu", "ID_LIKE": "debian"}, nil
		},
		Run: func(context.Context, navivoxHostSetupCommand) error {
			ran = true
			return nil
		},
	})
	defer restore()

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "navivox", "setup-host", "--apply", "--yes")
	if err == nil {
		t.Fatalf("navivox setup-host --apply without a package manager succeeded; stdout=%s stderr=%s", stdout, stderr)
	}
	for _, want := range []string{
		"Navivox host setup",
		"supported package manager not found",
		"No changes were made.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing package manager output missing %q:\n%s", want, stdout)
		}
	}
	if ran {
		t.Fatal("missing package manager should not execute host setup commands")
	}
}

func withNavivoxHostSetupTestSeams(t *testing.T, seams navivoxHostSetupSeams) func() {
	t.Helper()
	prev := navivoxHostSetup
	navivoxHostSetup = seams.withDefaults()
	return func() {
		navivoxHostSetup = prev
	}
}

func navivoxHostCommandNames(commands []navivoxHostSetupCommand) []string {
	out := make([]string, 0, len(commands))
	for _, c := range commands {
		out = append(out, c.Name+" "+strings.Join(c.Args, " "))
	}
	return out
}

var errNavivoxHostSetupCommandMissing = errors.New("command missing")
