package vpnhost

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestList_TailscaleOnly_ReturnsSingleTailscaleEntry(t *testing.T) {
	l := Lister{
		LookPath: func(name string) (string, error) {
			if name == "tailscale" {
				return "/usr/bin/tailscale", nil
			}
			return "", exec.ErrNotFound
		},
		RunTailscale: func(_ context.Context, args ...string) ([]byte, error) {
			switch strings.Join(args, " ") {
			case "ip -4":
				return []byte("100.108.109.96\n"), nil
			case "ip -6":
				return []byte("fd7a:115c:a1e0::4d01:6d94\n"), nil
			}
			return nil, errors.New("unexpected tailscale args")
		},
	}

	got, err := l.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(List) = %d, want 1: %+v", len(got), got)
	}
	if got[0].Kind != KindTailscale {
		t.Errorf("Kind = %q, want %q", got[0].Kind, KindTailscale)
	}
	if got[0].IPv4 != "100.108.109.96" {
		t.Errorf("IPv4 = %q, want 100.108.109.96", got[0].IPv4)
	}
	if got[0].IPv6 != "fd7a:115c:a1e0::4d01:6d94" {
		t.Errorf("IPv6 = %q, want fd7a:115c:a1e0::4d01:6d94", got[0].IPv6)
	}
}

func TestList_WireGuardOnly_DetectsViaIPCommand(t *testing.T) {
	l := Lister{
		LookPath: func(name string) (string, error) {
			if name == "ip" {
				return "/sbin/ip", nil
			}
			return "", exec.ErrNotFound
		},
		RunIP: func(_ context.Context, args ...string) ([]byte, error) {
			joined := strings.Join(args, " ")
			switch joined {
			case "-j -d link show type wireguard":
				return []byte(`[{"ifname":"wg0","link_type":"none"},{"ifname":"wg1","link_type":"none"}]`), nil
			case "-j addr show dev wg0":
				return []byte(`[{"ifname":"wg0","addr_info":[{"family":"inet","local":"10.0.0.1","prefixlen":32}]}]`), nil
			case "-j addr show dev wg1":
				return []byte(`[{"ifname":"wg1","addr_info":[{"family":"inet6","local":"fd00::1","prefixlen":64}]}]`), nil
			case "-j link show type tun":
				return []byte(`[]`), nil
			}
			return nil, errors.New("unexpected ip args: " + joined)
		},
	}

	got, err := l.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(List) = %d, want 2: %+v", len(got), got)
	}
	for i, h := range got {
		if h.Kind != KindWireGuard {
			t.Errorf("entry %d kind = %q, want %q", i, h.Kind, KindWireGuard)
		}
	}
	if got[0].Iface != "wg0" || got[0].IPv4 != "10.0.0.1" {
		t.Errorf("wg0 entry = %+v, want Iface=wg0 IPv4=10.0.0.1", got[0])
	}
	if got[1].Iface != "wg1" || got[1].IPv6 != "fd00::1" {
		t.Errorf("wg1 entry = %+v, want Iface=wg1 IPv6=fd00::1", got[1])
	}
}

func TestList_OrderingTailscaleFirstThenWireGuardThenTun(t *testing.T) {
	l := Lister{
		LookPath: func(string) (string, error) { return "/x", nil },
		RunTailscale: func(_ context.Context, args ...string) ([]byte, error) {
			if strings.Join(args, " ") == "ip -4" {
				return []byte("100.1.2.3\n"), nil
			}
			return []byte(""), nil
		},
		RunIP: func(_ context.Context, args ...string) ([]byte, error) {
			joined := strings.Join(args, " ")
			switch joined {
			case "-j -d link show type wireguard":
				return []byte(`[{"ifname":"wg0","link_type":"none"}]`), nil
			case "-j addr show dev wg0":
				return []byte(`[{"ifname":"wg0","addr_info":[{"family":"inet","local":"10.0.0.1"}]}]`), nil
			case "-j link show type tun":
				return []byte(`[{"ifname":"tailscale0"},{"ifname":"tun0"}]`), nil
			case "-j addr show dev tun0":
				return []byte(`[{"ifname":"tun0","addr_info":[{"family":"inet","local":"10.8.0.5"}]}]`), nil
			}
			return nil, errors.New("unexpected ip args: " + joined)
		},
	}

	got, err := l.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(List) = %d, want 3: %+v", len(got), got)
	}
	if got[0].Kind != KindTailscale {
		t.Errorf("got[0].Kind = %q, want %q", got[0].Kind, KindTailscale)
	}
	if got[1].Kind != KindWireGuard {
		t.Errorf("got[1].Kind = %q, want %q", got[1].Kind, KindWireGuard)
	}
	if got[2].Kind != KindTunOther {
		t.Errorf("got[2].Kind = %q, want %q", got[2].Kind, KindTunOther)
	}
	if got[2].Iface != "tun0" {
		t.Errorf("got[2].Iface = %q, want tun0 (tailscale0 must be excluded from tun-other)", got[2].Iface)
	}
}

func TestList_NoVPNDetected_ReturnsEmpty(t *testing.T) {
	l := Lister{
		LookPath: func(string) (string, error) { return "", exec.ErrNotFound },
	}
	got, err := l.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(List) = %d, want 0: %+v", len(got), got)
	}
}

func TestList_TailscaleErrorIsIgnored_DoesNotBlockOtherDetectors(t *testing.T) {
	l := Lister{
		LookPath: func(name string) (string, error) {
			if name == "tailscale" || name == "ip" {
				return "/x", nil
			}
			return "", exec.ErrNotFound
		},
		RunTailscale: func(context.Context, ...string) ([]byte, error) {
			return nil, errors.New("tailscale not running")
		},
		RunIP: func(_ context.Context, args ...string) ([]byte, error) {
			joined := strings.Join(args, " ")
			switch joined {
			case "-j -d link show type wireguard":
				return []byte(`[{"ifname":"wg0"}]`), nil
			case "-j addr show dev wg0":
				return []byte(`[{"ifname":"wg0","addr_info":[{"family":"inet","local":"10.0.0.1"}]}]`), nil
			case "-j link show type tun":
				return []byte(`[]`), nil
			}
			return nil, errors.New("unexpected ip args: " + joined)
		},
	}
	got, err := l.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0].Kind != KindWireGuard {
		t.Fatalf("List = %+v, want one wireguard entry", got)
	}
}

func TestList_PackageLevelFnUsesDefaultLister(t *testing.T) {
	// Smoke-only: the package-level List(ctx) must call into the default
	// Lister without panicking when no VPN CLIs are present on the test host.
	got, err := List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	_ = got // contents depend on host; we only assert no panic / no error.
}
