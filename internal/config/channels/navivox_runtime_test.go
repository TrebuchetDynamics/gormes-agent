package channels

import (
	"strings"
	"testing"
)

func TestNavivoxExposureWireGuardAndVPNAreAccepted(t *testing.T) {
	for _, mode := range []string{NavivoxExposureWireGuard, NavivoxExposureVPN} {
		cfg := &NavivoxCfg{
			Enabled:      true,
			BindHost:     "10.0.0.1",
			Port:         NavivoxDefaultPort,
			ExposureMode: mode,
			AuthMode:     NavivoxAuthStaticToken,
			Token:        "x",
		}
		if err := ValidateNavivoxForRuntime(cfg); err != nil {
			t.Errorf("ValidateNavivoxForRuntime(mode=%s) error = %v, want nil", mode, err)
		}
	}
}

func TestNavivoxExposureRequiresVPN(t *testing.T) {
	cases := map[string]bool{
		NavivoxExposureLocal:     false,
		NavivoxExposureTailscale: true,
		NavivoxExposureWireGuard: true,
		NavivoxExposureVPN:       true,
		NavivoxExposurePublic:    false,
		"unknown":                false,
	}
	for mode, want := range cases {
		if got := NavivoxExposureRequiresVPN(mode); got != want {
			t.Errorf("NavivoxExposureRequiresVPN(%q) = %v, want %v", mode, got, want)
		}
	}
}

func TestValidateNavivoxBindAgainstVPN_NilOrDisabled_ReturnsNil(t *testing.T) {
	if err := ValidateNavivoxBindAgainstVPN(nil, nil); err != nil {
		t.Fatalf("nil cfg error = %v, want nil", err)
	}
	cfg := &NavivoxCfg{Enabled: false, ExposureMode: NavivoxExposureTailscale, BindHost: "127.0.0.1"}
	if err := ValidateNavivoxBindAgainstVPN(cfg, nil); err != nil {
		t.Fatalf("disabled cfg error = %v, want nil", err)
	}
}

func TestValidateNavivoxBindAgainstVPN_LocalOrPublic_ReturnsNilRegardlessOfIPs(t *testing.T) {
	for _, mode := range []string{NavivoxExposureLocal, NavivoxExposurePublic} {
		cfg := &NavivoxCfg{Enabled: true, ExposureMode: mode, BindHost: "127.0.0.1"}
		if err := ValidateNavivoxBindAgainstVPN(cfg, nil); err != nil {
			t.Errorf("mode=%s error = %v, want nil (only VPN-class modes are gated)", mode, err)
		}
	}
}

func TestValidateNavivoxBindAgainstVPN_TailscaleMatchesProvidedIP_ReturnsNil(t *testing.T) {
	cfg := &NavivoxCfg{Enabled: true, ExposureMode: NavivoxExposureTailscale, BindHost: "100.64.1.2"}
	if err := ValidateNavivoxBindAgainstVPN(cfg, []string{"100.64.1.2", "10.0.0.1"}); err != nil {
		t.Fatalf("matching IP error = %v, want nil", err)
	}
}

func TestValidateNavivoxBindAgainstVPN_TailscaleNoIPs_FailsClosed(t *testing.T) {
	cfg := &NavivoxCfg{Enabled: true, ExposureMode: NavivoxExposureTailscale, BindHost: "127.0.0.1"}
	err := ValidateNavivoxBindAgainstVPN(cfg, nil)
	if err == nil {
		t.Fatal("error = nil, want VPN-not-detected error")
	}
	if !strings.Contains(err.Error(), "no active VPN interface") {
		t.Errorf("error = %q, want hint about missing VPN", err)
	}
}

func TestValidateNavivoxBindAgainstVPN_TailscaleMismatch_FailsClosedWithBothListed(t *testing.T) {
	cfg := &NavivoxCfg{Enabled: true, ExposureMode: NavivoxExposureWireGuard, BindHost: "127.0.0.1"}
	err := ValidateNavivoxBindAgainstVPN(cfg, []string{"10.0.0.1"})
	if err == nil {
		t.Fatal("error = nil, want bind/vpn-ip mismatch error")
	}
	for _, want := range []string{"127.0.0.1", "10.0.0.1", "exposure_mode=wireguard"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, missing %q", err, want)
		}
	}
}
