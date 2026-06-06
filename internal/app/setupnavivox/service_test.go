package setupnavivox

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setupchoice"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

func TestPairingURIIncludesTokenWhenRequired(t *testing.T) {
	descriptor, err := PairingURI(config.NavivoxCfg{BindHost: "127.0.0.1", Port: 8765, AuthMode: config.NavivoxAuthPairingToken, ExposureMode: config.NavivoxExposureLocal, Token: "nvbx_setup"})
	if err != nil {
		t.Fatalf("PairingURI: %v", err)
	}
	parsed, err := url.Parse(descriptor)
	if err != nil {
		t.Fatalf("parse descriptor: %v", err)
	}
	if parsed.Scheme != "navivox" || parsed.Host != "connect" {
		t.Fatalf("descriptor = %q, want navivox://connect URI", descriptor)
	}
	values := parsed.Query()
	for key, want := range map[string]string{
		"base_url":         "http://127.0.0.1:8765",
		"websocket_url":    "ws://127.0.0.1:8765/v1/navivox/stream",
		"capabilities_url": "http://127.0.0.1:8765/v1/navivox/capabilities",
		"auth_mode":        config.NavivoxAuthPairingToken,
		"exposure_mode":    config.NavivoxExposureLocal,
		"token_required":   "true",
		"rest_token":       "nvbx_setup",
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestBindDefaultPrefersCurrentThenVPNKind(t *testing.T) {
	hosts := []vpnhost.Host{
		{Kind: vpnhost.KindWireGuard, IPv4: "10.0.0.2"},
		{Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
	}
	if got := BindDefault("192.168.5.5", config.NavivoxExposureTailscale, hosts); got != "192.168.5.5" {
		t.Fatalf("current BindDefault = %q", got)
	}
	if got := BindDefault("", config.NavivoxExposureTailscale, hosts); got != "100.64.1.2" {
		t.Fatalf("tailscale BindDefault = %q", got)
	}
	if got := BindDefault("", config.NavivoxExposureWireGuard, hosts); got != "10.0.0.2" {
		t.Fatalf("wireguard BindDefault = %q", got)
	}
}

func TestBindDefaultVPNFallbacks(t *testing.T) {
	if got := BindDefault("", config.NavivoxExposureTailscale, []vpnhost.Host{{Kind: vpnhost.KindTailscale, IPv6: "fd7a:115c:a1e0::1"}}); got != config.NavivoxDefaultBindHost {
		t.Fatalf("tailscale without IPv4 BindDefault = %q", got)
	}
	if got := BindDefault("", config.NavivoxExposureVPN, []vpnhost.Host{{Kind: vpnhost.KindTunOther, IPv4: "10.8.0.5"}, {Kind: vpnhost.KindWireGuard, IPv4: "10.0.0.1"}}); got != "10.8.0.5" {
		t.Fatalf("vpn BindDefault = %q", got)
	}
}

func TestCSVTrimsNonEmptyValues(t *testing.T) {
	got := CSV(" alice@example.com, , bob@example.com ")
	want := []string{"alice@example.com", "bob@example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CSV = %#v, want %#v", got, want)
	}
}

func TestRunGatewayWritesConfigProfileAndRedactsToken(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, "config.toml")
	envPath := filepath.Join(home, ".env")
	var out bytes.Buffer
	var profileBinding ProfileChannelBinding
	var wroteLegacyEnv string
	var wroteToken string
	var qrDescriptor string

	err := RunGateway(GatewayOptions{
		Context:    context.Background(),
		Out:        &out,
		Config:     config.Config{Hermes: config.HermesCfg{Endpoint: "https://api.example/v1", Model: "model-a"}},
		ConfigPath: func() string { return configPath },
		EnvPath:    func() string { return envPath },
		GormesHome: func() string { return home },
		AskYesNo: func(title, _ string, defaultValue bool) (bool, bool, error) {
			switch title {
			case "Enable Navivox Gateway Channel?":
				return true, true, nil
			case "Record manual firewall-open intent?":
				return false, true, nil
			default:
				return defaultValue, true, nil
			}
		},
		PromptChoice: func(title, _, defaultID string, choices []setupchoice.Choice) (string, error) {
			switch title {
			case "Exposure mode":
				return config.NavivoxExposureLocal, nil
			case "Auth mode":
				return config.NavivoxAuthPairingToken, nil
			default:
				return setupchoice.NormalizeAnswer("", choices, defaultID), nil
			}
		},
		PromptString: func(prompt, defaultValue string) (string, error) {
			switch {
			case strings.HasPrefix(prompt, "Bind host"):
				return defaultValue, nil
			case strings.HasPrefix(prompt, "Port"):
				return "8765", nil
			default:
				return defaultValue, nil
			}
		},
		GenerateSetupToken: func() (string, error) { return "setup-secret-token", nil },
		NewGatewayID:       func() (string, error) { return "gw_0123456789abcdef0123456789abcdef", nil },
		WriteProfileChannelBinding: func(opts ProfileChannelOptions) (ProfileChannelBinding, error) {
			profileBinding = ProfileChannelBinding{ProfileID: "main", ChannelID: opts.ChannelID, CredentialID: "main-navivox", SecretEnvName: "GORMES_MAIN_NAVIVOX_TOKEN", RegistryPath: filepath.Join(home, "profiles.toml")}
			return profileBinding, nil
		},
		WriteGatewayTokenEnv: func(binding ProfileChannelBinding, legacyEnvName, token string) error {
			if binding != profileBinding {
				t.Fatalf("token binding = %#v, want %#v", binding, profileBinding)
			}
			wroteLegacyEnv = legacyEnvName
			wroteToken = token
			return nil
		},
		WritePairingQR: func(_ string, descriptor string) error {
			qrDescriptor = descriptor
			return nil
		},
		TerminalQR: func(string) (string, error) { return "QR-LINE\n", nil },
		ProviderAuthConfigured: func(config.Config) bool {
			return true
		},
	})
	if err != nil {
		t.Fatalf("RunGateway: %v\nstdout=%s", err, out.String())
	}
	stdout := out.String()
	for _, want := range []string{
		"Navivox Gateway Channel",
		"Navivox gateway channel configured.",
		"HTTP: http://127.0.0.1:8765",
		"WebSocket: ws://127.0.0.1:8765/v1/navivox/stream",
		"Profile token: GORMES_MAIN_NAVIVOX_TOKEN",
		"Profile channel: profiles.main.channels.navivox",
		"QR-LINE",
		"QR payload includes the token when required; the raw token is not printed.",
		"3. Start gateway: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "setup-secret-token") {
		t.Fatalf("stdout leaked token:\n%s", stdout)
	}
	if wroteLegacyEnv != "GORMES_NAVIVOX_TOKEN" || wroteToken != "setup-secret-token" {
		t.Fatalf("token env write = (%q, %q)", wroteLegacyEnv, wroteToken)
	}
	if !strings.Contains(qrDescriptor, "rest_token=setup-secret-token") {
		t.Fatalf("QR descriptor missing token: %s", qrDescriptor)
	}
	body := readText(t, configPath)
	for _, want := range []string{
		"enabled = true",
		"gateway_id = 'gw_0123456789abcdef0123456789abcdef'",
		"bind_host = '127.0.0.1'",
		"port = 8765",
		"auth_mode = 'pairing_token'",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("config missing %q:\n%s", want, body)
		}
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
