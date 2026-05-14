package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadNaviboxDefaultsAreSafe(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearNaviboxEnv(t)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Navibox.Enabled {
		t.Fatal("Navibox enabled by default")
	}
	if cfg.Navibox.BindHost != NaviboxDefaultBindHost {
		t.Fatalf("BindHost = %q, want %q", cfg.Navibox.BindHost, NaviboxDefaultBindHost)
	}
	if cfg.Navibox.Port != NaviboxDefaultPort {
		t.Fatalf("Port = %d, want %d", cfg.Navibox.Port, NaviboxDefaultPort)
	}
	if cfg.Navibox.ExposureMode != NaviboxExposureLocal {
		t.Fatalf("ExposureMode = %q, want local", cfg.Navibox.ExposureMode)
	}
	if cfg.Navibox.AuthMode != NaviboxAuthPairingToken {
		t.Fatalf("AuthMode = %q, want pairing_token", cfg.Navibox.AuthMode)
	}
}

func TestLoadNaviboxEnabledRequiresTokenForTokenAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearNaviboxEnv(t)
	writeNaviboxConfig(t, home, `
[navibox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
auth_mode = "pairing_token"
`)

	_, err := Load(nil)
	if err == nil {
		t.Fatal("Load() error = nil, want missing token error")
	}
	if !strings.Contains(err.Error(), "navibox.token is required") {
		t.Fatalf("error = %q, want navibox.token required", err)
	}
}

func TestLoadNaviboxEnvTokenEnablesParseableLocalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearNaviboxEnv(t)
	t.Setenv("GORMES_NAVIBOX_TOKEN", "nvbx_test_token")
	writeNaviboxConfig(t, home, `
[navibox]
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
	if cfg.Navibox.Token != "nvbx_test_token" {
		t.Fatalf("Token loaded from env = %q", cfg.Navibox.Token)
	}
}

func TestLoadNaviboxRejectsAccidentalPublicExposure(t *testing.T) {
	for _, body := range []string{
		`
[navibox]
enabled = true
bind_host = "0.0.0.0"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"
token = "nvbx_test_token"
`,
		`
[navibox]
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
			clearNaviboxEnv(t)
			writeNaviboxConfig(t, home, body)

			_, err := Load(nil)
			if err == nil {
				t.Fatal("Load() error = nil, want public exposure rejection")
			}
			if !strings.Contains(err.Error(), "navibox") {
				t.Fatalf("error = %q, want navibox rejection", err)
			}
		})
	}
}

func TestLoadNaviboxAllowsExplicitPublicMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearNaviboxEnv(t)
	writeNaviboxConfig(t, home, `
[navibox]
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
	if cfg.Navibox.ExposureMode != NaviboxExposurePublic || !cfg.Navibox.PublicConfirmed {
		t.Fatalf("Navibox public config = %+v, want explicit public", cfg.Navibox)
	}
}

func TestConfigWriterRoutesNaviboxTokenToEnv(t *testing.T) {
	if !IsSecretKey("navibox.token") {
		t.Fatal("navibox.token should be classified as a secret")
	}
	if got := SecretEnvName("navibox.token"); got != "GORMES_NAVIBOX_TOKEN" {
		t.Fatalf("SecretEnvName(navibox.token) = %q", got)
	}
}

func clearNaviboxEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GORMES_NAVIBOX_ENABLED",
		"GORMES_NAVIBOX_BIND_HOST",
		"GORMES_NAVIBOX_PORT",
		"GORMES_NAVIBOX_EXPOSURE_MODE",
		"GORMES_NAVIBOX_AUTH_MODE",
		"GORMES_NAVIBOX_TOKEN",
		"GORMES_NAVIBOX_ALLOW_ORIGINS",
		"GORMES_NAVIBOX_ALLOWED_TAILNET_IDENTITIES",
		"GORMES_NAVIBOX_PUBLIC_CONFIRMED",
	} {
		t.Setenv(key, "")
	}
}

func writeNaviboxConfig(t *testing.T, home, body string) {
	t.Helper()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
