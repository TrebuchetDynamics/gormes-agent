package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadNavivoxDefaultsAreSafe(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearNavivoxEnv(t)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Navivox.Enabled {
		t.Fatal("Navivox enabled by default")
	}
	if cfg.Navivox.BindHost != NavivoxDefaultBindHost {
		t.Fatalf("BindHost = %q, want %q", cfg.Navivox.BindHost, NavivoxDefaultBindHost)
	}
	if cfg.Navivox.Port != NavivoxDefaultPort {
		t.Fatalf("Port = %d, want %d", cfg.Navivox.Port, NavivoxDefaultPort)
	}
	if cfg.Navivox.ExposureMode != NavivoxExposureLocal {
		t.Fatalf("ExposureMode = %q, want local", cfg.Navivox.ExposureMode)
	}
	if cfg.Navivox.AuthMode != NavivoxAuthPairingToken {
		t.Fatalf("AuthMode = %q, want pairing_token", cfg.Navivox.AuthMode)
	}
}

func TestLoadNavivoxEnabledRequiresTokenForTokenAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearNavivoxEnv(t)
	writeNavivoxConfig(t, home, `
[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
auth_mode = "pairing_token"
`)

	_, err := Load(nil)
	if err == nil {
		t.Fatal("Load() error = nil, want missing token error")
	}
	if !strings.Contains(err.Error(), "navivox.token is required") {
		t.Fatalf("error = %q, want navivox.token required", err)
	}
}

func TestLoadNavivoxEnvTokenEnablesParseableLocalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearNavivoxEnv(t)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "nvbx_test_token")
	writeNavivoxConfig(t, home, `
[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
`)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Navivox.Token != "nvbx_test_token" {
		t.Fatalf("Token loaded from env = %q", cfg.Navivox.Token)
	}
}

func TestLoadNavivoxRejectsAccidentalPublicExposure(t *testing.T) {
	for _, body := range []string{
		`
[navivox]
enabled = true
bind_host = "0.0.0.0"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
token = "nvbx_test_token"
`,
		`
[navivox]
enabled = true
bind_host = "0.0.0.0"
port = 8765
exposure_mode = "public"
auth_mode = "pairing_token"
token = "nvbx_test_token"
`,
	} {
		t.Run("", func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("GORMES_HOME", home)
			clearNavivoxEnv(t)
			writeNavivoxConfig(t, home, body)

			_, err := Load(nil)
			if err == nil {
				t.Fatal("Load() error = nil, want public exposure rejection")
			}
			if !strings.Contains(err.Error(), "navivox") {
				t.Fatalf("error = %q, want navivox rejection", err)
			}
		})
	}
}

func TestLoadNavivoxAllowsExplicitPublicMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearNavivoxEnv(t)
	writeNavivoxConfig(t, home, `
[navivox]
enabled = true
bind_host = "0.0.0.0"
port = 8765
exposure_mode = "public"
auth_mode = "pairing_token"
token = "nvbx_test_token"
public_confirmed = true
`)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Navivox.ExposureMode != NavivoxExposurePublic || !cfg.Navivox.PublicConfirmed {
		t.Fatalf("Navivox public config = %+v, want explicit public", cfg.Navivox)
	}
}

func TestConfigWriterRoutesNavivoxTokenToEnv(t *testing.T) {
	if !IsSecretKey("navivox.token") {
		t.Fatal("navivox.token should be classified as a secret")
	}
	if got := SecretEnvName("navivox.token"); got != "GORMES_NAVIVOX_TOKEN" {
		t.Fatalf("SecretEnvName(navivox.token) = %q", got)
	}
}

func clearNavivoxEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GORMES_NAVIVOX_ENABLED",
		"GORMES_NAVIVOX_BIND_HOST",
		"GORMES_NAVIVOX_PORT",
		"GORMES_NAVIVOX_EXPOSURE_MODE",
		"GORMES_NAVIVOX_AUTH_MODE",
		"GORMES_NAVIVOX_TOKEN",
		"GORMES_NAVIVOX_ALLOW_ORIGINS",
		"GORMES_NAVIVOX_ALLOWED_TAILNET_IDENTITIES",
		"GORMES_NAVIVOX_PUBLIC_CONFIRMED",
	} {
		t.Setenv(key, "")
	}
}

func writeNavivoxConfig(t *testing.T, home, body string) {
	t.Helper()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
